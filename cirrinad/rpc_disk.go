package main

import (
	"bufio"
	"context"
	"crypto/sha512"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"

	"cirrina/cirrina"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/vm"

	"github.com/google/uuid"
	"log/slog"
)

func (s *server) GetDisks(_ *cirrina.DisksQuery, stream cirrina.VMInfo_GetDisksServer) error {
	var DiskId cirrina.DiskId
	for _, diskInst := range disk.List.DiskList {
		DiskId.Value = diskInst.ID
		err := stream.Send(&DiskId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *server) AddDisk(_ context.Context, i *cirrina.DiskInfo) (*cirrina.DiskId, error) {
	var diskType string
	var diskDevType string

	defaultDiskDescription := ""
	defaultDiskType := cirrina.DiskType_NVME
	defaultDiskSize := config.Config.Disk.Default.Size
	defaultDiskDevType := cirrina.DiskDevType_FILE
	defaultDiskCache := true
	defaultDiskDirect := false

	if i.Name == nil || *i.Name == "" {
		return &cirrina.DiskId{}, errors.New("name not specified")
	}

	if i.Size == nil || *i.Size == "" {
		i.Size = &defaultDiskSize
	}

	if i.Description == nil {
		i.Description = &defaultDiskDescription
	}

	if i.DiskType == nil {
		i.DiskType = &defaultDiskType
	}

	if i.DiskDevType == nil {
		i.DiskDevType = &defaultDiskDevType
	}

	if i.Cache == nil {
		i.Cache = &defaultDiskCache
	}

	if i.Direct == nil {
		i.Direct = &defaultDiskDirect
	}

	if *i.DiskType == cirrina.DiskType_NVME {
		diskType = "NVME"
	} else if *i.DiskType == cirrina.DiskType_AHCIHD {
		diskType = "AHCI-HD"
	} else if *i.DiskType == cirrina.DiskType_VIRTIOBLK {
		diskType = "VIRTIO-BLK"
	} else {
		return &cirrina.DiskId{}, errors.New("invalid disk type")
	}

	if *i.DiskDevType == cirrina.DiskDevType_FILE {
		diskDevType = "FILE"
	} else if *i.DiskDevType == cirrina.DiskDevType_ZVOL {
		diskDevType = "ZVOL"
	} else {
		return &cirrina.DiskId{}, errors.New("invalid disk dev type")
	}

	diskInst, err := disk.Create(*i.Name, *i.Description, *i.Size, diskType, diskDevType, *i.Cache, *i.Direct)
	if err != nil {
		return &cirrina.DiskId{}, err
	}
	if diskInst != nil && diskInst.ID != "" {
		return &cirrina.DiskId{Value: diskInst.ID}, nil
	} else {
		return &cirrina.DiskId{}, errors.New("unknown error creating disk")
	}
}

func (s *server) GetDiskInfo(_ context.Context, i *cirrina.DiskId) (*cirrina.DiskInfo, error) {
	var ic cirrina.DiskInfo
	var stat syscall.Stat_t
	var blockSize int64 = 512
	defaultDiskCache := true
	defaultDiskDirect := false

	diskUuid, err := uuid.Parse(i.Value)
	if err != nil {
		return &ic, errors.New("invalid disk id")
	}
	diskInst, err := disk.GetById(diskUuid.String())
	if err != nil {
		slog.Error("error getting disk", "disk", i.Value, "err", err)
		return &ic, errors.New("not found")
	}
	if diskInst.Name == "" {
		slog.Debug("disk not found")
		return &ic, errors.New("not found")
	}
	ic.Name = &diskInst.Name
	ic.Description = &diskInst.Description
	if diskInst.DiskCache.Valid {
		ic.Cache = &diskInst.DiskCache.Bool
	} else {
		ic.Cache = &defaultDiskCache
	}
	if diskInst.DiskDirect.Valid {
		ic.Direct = &diskInst.DiskDirect.Bool
	} else {
		ic.Direct = &defaultDiskDirect
	}
	DiskTypeNVME := cirrina.DiskType_NVME
	DiskTypeAHCI := cirrina.DiskType_AHCIHD
	DiskTypeVIRT := cirrina.DiskType_VIRTIOBLK
	DiskDevTypeFile := cirrina.DiskDevType_FILE
	DiskDevTypeZvol := cirrina.DiskDevType_ZVOL

	if diskInst.Type == "NVME" {
		ic.DiskType = &DiskTypeNVME
	} else if diskInst.Type == "AHCI-HD" {
		ic.DiskType = &DiskTypeAHCI
	} else if diskInst.Type == "VIRTIO-BLK" {
		ic.DiskType = &DiskTypeVIRT
	} else {
		slog.Error("GetDiskInfo invalid disk type", "diskid", i.Value, "disktype", diskInst.Type)
		return nil, errors.New("invalid disk type")
	}

	if diskInst.DevType == "FILE" {
		diskPath, err := diskInst.GetPath()
		if err != nil {
			slog.Error("GetDiskInfo error getting disk size", "err", err)
			return nil, errors.New("unable to get file size")
		}

		diskFileStat, err := os.Stat(diskPath)
		if err != nil {
			slog.Error("GetDiskInfo error getting disk size", "err", err)
			return nil, errors.New("unable to get file size")
		}

		err = syscall.Stat(diskPath, &stat)
		if err != nil {
			slog.Error("unable to stat diskPath", "diskPath", diskPath)
			return nil, errors.New("unable to stat")
		}

		diskSize := strconv.FormatInt(diskFileStat.Size(), 10)
		diskBlocks := strconv.FormatInt(stat.Blocks*blockSize, 10)
		diskSizeNum := uint64(diskFileStat.Size())
		diskUsageNum := uint64(stat.Blocks * blockSize)

		ic.Size = &diskSize
		ic.SizeNum = &diskSizeNum
		ic.Usage = &diskBlocks
		ic.UsageNum = &diskUsageNum

		if strings.HasSuffix(*ic.Name, ".img") {
			*ic.Name = strings.TrimSuffix(*ic.Name, ".img")
		}
		ic.DiskDevType = &DiskDevTypeFile
	} else if diskInst.DevType == "ZVOL" {
		if config.Config.Disk.VM.Path.Zpool == "" {
			return &cirrina.DiskInfo{}, errors.New("zfs pool not configured, cannot manage zvol disks")
		}

		diskSizeNum, err := disk.GetZfsVolumeSize(config.Config.Disk.VM.Path.Zpool + "/" + diskInst.Name)
		if err != nil {
			diskSizeNum = 0
		}

		diskUsageNum, err := disk.GetZfsVolumeUsage(config.Config.Disk.VM.Path.Zpool + "/" + diskInst.Name)
		if err != nil {
			diskUsageNum = 0
		}

		diskBlocks := strconv.FormatInt(int64(diskUsageNum), 10)
		diskSize := strconv.FormatInt(int64(diskSizeNum), 10)

		ic.Size = &diskSize
		ic.SizeNum = &diskSizeNum
		ic.Usage = &diskBlocks
		ic.UsageNum = &diskUsageNum

		ic.DiskDevType = &DiskDevTypeZvol
	}

	return &ic, nil
}

func (s *server) RemoveDisk(_ context.Context, i *cirrina.DiskId) (*cirrina.ReqBool, error) {
	slog.Debug("deleting disk", "diskid", i.Value)
	re := cirrina.ReqBool{}
	re.Success = false

	diskUuid, err := uuid.Parse(i.Value)
	if err != nil {
		return &re, errors.New("invalid disk id")
	}

	diskInst, err := disk.GetById(diskUuid.String())
	if err != nil {
		slog.Error("error getting disk", "disk", i.Value, "err", err)
		return &re, errors.New("not found")
	}
	if diskInst.Name == "" {
		slog.Debug("disk not found")
		return &re, errors.New("not found")
	}

	// check that disk is not in use by a VM
	var diskVm *vm.VM
	diskVm, err = getDiskVm(diskUuid)
	if err != nil {
		return &re, err
	}

	if diskVm != nil {
		errorMessage := fmt.Sprintf("disk in use by VM %s", diskVm.ID)
		return &re, errors.New(errorMessage)
	}

	// prevent deleting disk while it's being uploaded etc
	defer diskInst.Unlock()
	diskInst.Lock()

	res := disk.Delete(diskUuid.String())
	if res != nil {
		slog.Error("error deleting disk", "res", res)
		return &re, errors.New("error deleting disk")
	}

	re.Success = true
	return &re, nil
}

func (s *server) GetDiskVm(_ context.Context, i *cirrina.DiskId) (v *cirrina.VMID, err error) {
	var pvmId cirrina.VMID

	diskUuid, err := uuid.Parse(i.Value)
	if err != nil {
		return &pvmId, errors.New("invalid disk id")
	}

	var rv *vm.VM
	rv, err = getDiskVm(diskUuid)
	if err != nil {
		return &cirrina.VMID{}, err
	}

	if rv != nil {
		pvmId.Value = rv.ID
	}
	return &pvmId, nil
}

func getDiskVm(diskUuid uuid.UUID) (*vm.VM, error) {
	allVMs := vm.GetAll()
	found := false
	var rv *vm.VM

	for _, thisVm := range allVMs {
		thisVmDisks, err := thisVm.GetDisks()
		if err != nil {
			return &vm.VM{}, err
		}
		for _, vmDisk := range thisVmDisks {
			if vmDisk.ID == diskUuid.String() {
				if found == true {
					slog.Error("GetDiskVm disk in use by more than one VM",
						"diskUuid", diskUuid,
						"vmid", thisVm.ID,
					)
					return &vm.VM{}, errors.New("disk in use by more than one VM")
				}
				found = true
				rv = thisVm
			}
		}
	}
	return rv, nil
}

func (s *server) SetDiskInfo(_ context.Context, diu *cirrina.DiskInfoUpdate) (*cirrina.ReqBool, error) {
	var re cirrina.ReqBool
	re.Success = false

	if diu.Id == "" {
		return &re, errors.New("id not specified or invalid")
	}

	DiskUuid, err := uuid.Parse(diu.Id)
	if err != nil {
		return &re, errors.New("id not specified or invalid")
	}

	diskInst, err := disk.GetById(DiskUuid.String())
	if err != nil {
		return &re, err
	}

	if diu.Description != nil {
		diskInst.Description = *diu.Description
		slog.Debug("SetDiskInfo", "description", *diu.Description)
	}

	if diu.DiskType != nil {
		if *diu.DiskType == cirrina.DiskType_NVME {
			diskInst.Type = "NVME"
		} else if *diu.DiskType == cirrina.DiskType_AHCIHD {
			diskInst.Type = "AHCI-HD"
		} else if *diu.DiskType == cirrina.DiskType_VIRTIOBLK {
			diskInst.Type = "VIRTIO-BLK"
		} else {
			return &re, errors.New("invalid disk type")
		}
		slog.Debug("SetDiskInfo", "type", diskInst.Type)
	}

	if diu.Cache != nil {
		diskInst.DiskCache = sql.NullBool{Bool: *diu.Cache, Valid: true}
	}

	if diu.Direct != nil {
		diskInst.DiskDirect = sql.NullBool{Bool: *diu.Direct, Valid: true}
	}

	slog.Debug("SetDiskInfo saving disk")
	err = diskInst.Save()
	if err != nil {
		return &re, errors.New("failed to update Disk")
	}
	re.Success = true

	return &re, nil
}

func (s *server) UploadDisk(stream cirrina.VMInfo_UploadDiskServer) error {
	var re cirrina.ReqBool
	re.Success = false
	var imageSize uint64
	imageSize = 0

	req, err := stream.Recv()
	if err != nil {
		slog.Error("UploadDisk", "msg", "cannot receive image info")
	}
	diskUploadReq := req.GetDiskuploadinfo()
	if diskUploadReq == nil || diskUploadReq.Diskid == nil {
		slog.Error("nil diskUploadReq or disk id")
		return errors.New("nil diskUploadReq or disk id")
	}
	diskId := diskUploadReq.Diskid

	diskUuid, err := uuid.Parse(diskId.Value)
	if err != nil {
		slog.Error("disk id not specified or invalid on upload")
		return errors.New("id not specified or invalid")
	}
	diskInst, err := disk.GetById(diskUuid.String())
	if err != nil {
		slog.Error("error getting disk", "id", diskUuid.String(), "err", err)
		return errors.New("not found")
	}

	if diskInst.Name == "" {
		slog.Debug("disk not found")
		return errors.New("not found")
	}

	slog.Debug("UploadDisk",
		"diskId", diskId.Value,
		"diskName", diskInst.Name,
		"size", diskUploadReq.Size, "checksum", diskUploadReq.Sha512Sum,
	)

	diskPath, err := diskInst.GetPath()
	if err != nil {
		return err
	}

	if diskPath == "" {
		return errors.New("disk path empty")
	}

	var diskVm *vm.VM
	diskVm, err = getDiskVm(diskUuid)
	if err != nil {
		return err
	}

	if diskVm != nil {
		if diskVm.Status != "STOPPED" {
			slog.Error("UploadDisk can not upload disk to VM that is not stopped")
			return errors.New("can not upload disk to VM that is not stopped")
		}
	}

	slog.Debug("UploadDisk debug",
		"devtype", diskInst.DevType,
		"path", diskPath,
		"newsize", diskUploadReq.Size,
	)

	defer diskInst.Unlock()
	diskInst.Lock()

	if diskInst.DevType == "ZVOL" {
		err = disk.SetZfsVolumeSize(config.Config.Disk.VM.Path.Zpool+"/"+diskInst.Name, diskUploadReq.Size)
		if err != nil {
			slog.Error("UploadDisk", "msg", "failed setting new volume size", "err", err)
			return err
		}
	}

	err = diskInst.Save()
	if err != nil {
		slog.Error("UploadDisk", "msg", "Failed saving to db")
		err = stream.SendAndClose(&re)
		if err != nil {
			slog.Error("UploadDisk cannot send response", "err", err)
			return errors.New("failed sending response")
		}
		return err
	}

	diskFile, err := os.Create(diskPath)
	if err != nil {
		slog.Error("Failed to open disk file", "err", err.Error())
		return err
	}
	diskFileBuffer := bufio.NewWriter(diskFile)

	hasher := sha512.New()

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			slog.Debug("UploadDisk", "msg", "no more data")
			break
		}
		if err != nil {
			slog.Error("UploadDisk failed receiving", "err", err)
			return errors.New("failed reading image date")
		}

		chunk := req.GetImage()
		size := len(chunk)

		imageSize += uint64(size)
		_, err = diskFileBuffer.Write(chunk)
		if err != nil {
			slog.Error("UploadDisk failed writing", "err", err)
			return errors.New("failed writing disk image data")
		}
		hasher.Write(chunk)
	}

	// flush buffer
	err = diskFileBuffer.Flush()
	if err != nil {
		slog.Error("UploadDisk cannot send response", "err", err)
		return errors.New("failed flushing disk image data")
	}

	// verify size
	if imageSize != diskUploadReq.Size {
		slog.Error("UploadDisk image upload size incorrect")
		err = stream.SendAndClose(&re)
		if err != nil {
			slog.Error("UploadDisk cannot send response", "err", err)
		}
		return nil
	}

	// verify checksum
	diskChecksum := hex.EncodeToString(hasher.Sum(nil))
	if diskChecksum != diskUploadReq.Sha512Sum {
		slog.Debug("UploadDisk image upload checksum incorrect")
		err = stream.SendAndClose(&re)
		if err != nil {
			slog.Error("UploadDisk cannot send response", "err", err)
		}
		return nil
	}

	// finish saving file
	err = diskFile.Close()
	if err != nil {
		slog.Debug("UploadDisk", "msg", "Failed writing disk", "err", err)
		err = stream.SendAndClose(&re)
		if err != nil {
			slog.Error("UploadDisk cannot send response", "err", err)
		}
		return nil
	}

	// save to db
	err = diskInst.Save()
	if err != nil {
		slog.Error("UploadDisk", "msg", "Failed saving to db")
		err = stream.SendAndClose(&re)
		if err != nil {
			slog.Error("UploadDisk cannot send response", "err", err)
		}
		return nil
	}

	// we're done, return success to client
	re.Success = true
	err = stream.SendAndClose(&re)
	if err != nil {
		slog.Error("UploadDisk cannot send response", "err", err)
		return err
	}
	slog.Debug("UploadDisk complete")
	return nil
}
