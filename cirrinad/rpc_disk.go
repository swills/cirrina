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
	"log/slog"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/google/uuid"

	"cirrina/cirrina"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/vm"
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
	var err error

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
	} else {
		diskType, err = mapDiskTypeTypeToDBString(*i.DiskType)
		if err != nil {
			return &cirrina.DiskId{}, err
		}
	}

	if i.DiskDevType == nil {
		i.DiskDevType = &defaultDiskDevType
	} else {
		diskDevType, err = mapDiskDevTypeTypeToDBString(*i.DiskDevType)
		if err != nil {
			return &cirrina.DiskId{}, err
		}
	}

	if i.Cache == nil {
		i.Cache = &defaultDiskCache
	}

	if i.Direct == nil {
		i.Direct = &defaultDiskDirect
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

	ic.DiskType, err = mapDiskTypeDbStringToType(diskInst.Type)
	if err != nil {
		return &ic, err
	}

	ic.DiskDevType, err = mapDiskDevTypeDbStringToType(diskInst.DevType)
	if err != nil {
		return &ic, err
	}

	switch *ic.DiskDevType {
	case cirrina.DiskDevType_FILE:
		diskSize, diskBlocks, diskSizeNum, diskUsageNum, err := getDiskInfoFile(diskInst)
		if err != nil {
			return &ic, err
		}
		ic.Size = &diskSize
		ic.SizeNum = &diskSizeNum
		ic.Usage = &diskBlocks
		ic.UsageNum = &diskUsageNum

		if strings.HasSuffix(*ic.Name, ".img") {
			*ic.Name = strings.TrimSuffix(*ic.Name, ".img")
		}
	case cirrina.DiskDevType_ZVOL:
		if config.Config.Disk.VM.Path.Zpool == "" {
			return &cirrina.DiskInfo{}, errors.New("zfs pool not configured, cannot manage zvol disks")
		}
		diskSize, diskBlocks, diskSizeNum, diskUsageNum, err := getDiskInfoZVOL(diskInst)
		if err != nil {
			return &ic, err
		}

		ic.Size = &diskSize
		ic.SizeNum = &diskSizeNum
		ic.Usage = &diskBlocks
		ic.UsageNum = &diskUsageNum
	default:
		return &ic, errors.New("unknown dev type")
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
		errorMessage := "disk in use by VM " + diskVm.ID

		return &re, errors.New(errorMessage)
	}

	// prevent deleting disk while it's being uploaded etc
	defer diskInst.Unlock()
	diskInst.Lock()

	res := disk.Delete(diskInst.ID)
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
				if found {
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
		switch *diu.DiskType {
		case cirrina.DiskType_NVME:
			diskInst.Type = "NVME"
		case cirrina.DiskType_AHCIHD:
			diskInst.Type = "AHCI-HD"
		case cirrina.DiskType_VIRTIOBLK:
			diskInst.Type = "VIRTIO-BLK"
		default:
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

	req, err := stream.Recv()
	if err != nil {
		slog.Error("cannot receive image info")
	}
	diskUploadReq := req.GetDiskuploadinfo()
	diskInst, err := validateDiskReq(diskUploadReq)
	if err != nil {
		return err
	}
	diskPath, err := diskInst.GetPath()
	if err != nil {
		slog.Error("error getting path", "err", err)

		return err
	}

	defer diskInst.Unlock()
	diskInst.Lock()
	err = receiveDiskFile(stream, diskPath, imageSize, diskUploadReq)
	if err != nil {
		slog.Error("error during disk upload", "err", err)
		err2 := stream.SendAndClose(&re)
		if err2 != nil {
			slog.Error("UploadIso cannot send error response", "err", err, "err2", err2)

			return err
		}

		return err
	}

	// save to db
	err = diskInst.Save()
	if err != nil {
		slog.Error("UploadDisk", "msg", "Failed saving to db")
		err2 := stream.SendAndClose(&re)
		if err != nil {
			slog.Error("UploadDisk failed sending error response, ignoring", "err", err, "err2", err2)
		}

		return nil
	}
	// we're done, return success to client
	re.Success = true
	err = stream.SendAndClose(&re)
	if err != nil {
		slog.Error("UploadDisk cannot send response", "err", err)
	}

	return nil
}

func validateDiskReq(diskUploadReq *cirrina.DiskUploadInfo) (*disk.Disk, error) {
	if diskUploadReq == nil || diskUploadReq.Diskid == nil {
		slog.Error("nil diskUploadReq or disk id")

		return &disk.Disk{}, errors.New("nil diskUploadReq or disk id")
	}
	diskUuid, err := uuid.Parse(diskUploadReq.Diskid.Value)
	if err != nil {
		slog.Error("disk id not specified or invalid on upload")

		return &disk.Disk{}, errors.New("id not specified or invalid")
	}
	diskInst, err := disk.GetById(diskUuid.String())
	if err != nil {
		slog.Error("error getting disk", "id", diskUuid.String(), "err", err)

		return &disk.Disk{}, errors.New("not found")
	}
	if diskInst.Name == "" {
		slog.Debug("disk not found")

		return &disk.Disk{}, errors.New("not found")
	}
	var diskVm *vm.VM
	diskVm, err = getDiskVm(diskUuid)
	if err != nil {
		slog.Error("error getting disk VM", "err", err)

		return &disk.Disk{}, err
	}
	if diskVm != nil {
		if diskVm.Status != "STOPPED" {
			slog.Error("UploadDisk can not upload disk to VM that is not stopped")

			return &disk.Disk{}, errors.New("can not upload disk to VM that is not stopped")
		}
	}
	// not technically "validation" per se, but it needs to be done
	if diskInst.DevType == "ZVOL" {
		err = disk.SetZfsVolumeSize(config.Config.Disk.VM.Path.Zpool+"/"+diskInst.Name, diskUploadReq.Size)
		if err != nil {
			slog.Error("UploadDisk", "msg", "failed setting new volume size", "err", err)

			return &disk.Disk{}, err
		}
	}

	return diskInst, nil
}

func receiveDiskFile(stream cirrina.VMInfo_UploadDiskServer, diskPath string, imageSize uint64, diskUploadReq *cirrina.DiskUploadInfo) error {
	diskFile, err := os.Create(diskPath)
	if err != nil {
		slog.Error("Failed to open disk file", "err", err.Error())

		return err
	}
	diskFileBuffer := bufio.NewWriter(diskFile)

	hasher := sha512.New()

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			slog.Error("UploadDisk failed receiving", "err", err)

			return err
		}

		chunk := req.GetImage()
		size := len(chunk)

		imageSize += uint64(size)
		_, err = diskFileBuffer.Write(chunk)
		if err != nil {
			slog.Error("UploadDisk failed writing", "err", err)

			return err
		}
		hasher.Write(chunk)
	}
	diskChecksum := hex.EncodeToString(hasher.Sum(nil))

	// flush buffer
	err = diskFileBuffer.Flush()
	if err != nil {
		slog.Error("UploadDisk cannot send response", "err", err)

		return err
	}

	// verify size
	if imageSize != diskUploadReq.Size {
		slog.Error("UploadDisk  upload size incorrect",
			"imageSize", imageSize,
			"diskUploadReq.Size", diskUploadReq.Size,
		)

		return errors.New("disk upload size incorrect")
	}

	// verify checksum
	if diskChecksum != diskUploadReq.Sha512Sum {
		slog.Debug("UploadDisk image upload checksum incorrect",
			"diskChecksum", diskChecksum,
			"diskUploadReq.Sha512Sum", diskUploadReq.Sha512Sum,
		)

		return errors.New("disk upload checksum incorrect")
	}

	// finish saving file
	err = diskFile.Close()
	if err != nil {
		slog.Error("error closing disk file", "err", err)

		return err
	}

	return err
}

func mapDiskDevTypeTypeToDBString(diskDevType cirrina.DiskDevType) (string, error) {
	switch diskDevType {
	case cirrina.DiskDevType_FILE:
		return "FILE", nil
	case cirrina.DiskDevType_ZVOL:
		return "ZVOL", nil
	default:
		return "", fmt.Errorf("invalid disk dev type %s specified", diskDevType)
	}
}

func mapDiskDevTypeDbStringToType(diskDevType string) (*cirrina.DiskDevType, error) {
	DiskDevTypeFile := cirrina.DiskDevType_FILE
	DiskDevTypeZvol := cirrina.DiskDevType_ZVOL

	switch diskDevType {
	case "FILE":
		return &DiskDevTypeFile, nil
	case "ZVOL":
		return &DiskDevTypeZvol, nil
	default:
		return nil, errors.New("invalid disk dev type")
	}
}

func mapDiskTypeTypeToDBString(diskType cirrina.DiskType) (string, error) {
	switch diskType {
	case cirrina.DiskType_NVME:
		return "NVME", nil
	case cirrina.DiskType_AHCIHD:
		return "AHCI-HD", nil
	case cirrina.DiskType_VIRTIOBLK:
		return "VIRTIO-BLK", nil
	default:
		return "", fmt.Errorf("invalid disk type %s specified", diskType)
	}
}

func mapDiskTypeDbStringToType(diskType string) (*cirrina.DiskType, error) {
	DiskTypeNVME := cirrina.DiskType_NVME
	DiskTypeAHCI := cirrina.DiskType_AHCIHD
	DiskTypeVIRT := cirrina.DiskType_VIRTIOBLK

	switch diskType {
	case "NVME":
		return &DiskTypeNVME, nil
	case "AHCI-HD":
		return &DiskTypeAHCI, nil
	case "VIRTIO-BLK":
		return &DiskTypeVIRT, nil
	default:
		return nil, errors.New("invalid disk type")
	}
}

func getDiskInfoZVOL(diskInst *disk.Disk) (string, string, uint64, uint64, error) {
	diskSizeNum, err := disk.GetZfsVolumeSize(config.Config.Disk.VM.Path.Zpool + "/" + diskInst.Name)
	if err != nil {
		slog.Error("getDiskInfoZVOL GetZfsVolumeSize error", "err", err)

		return "", "", 0, 0, err
	}

	diskUsageNum, err := disk.GetZfsVolumeUsage(config.Config.Disk.VM.Path.Zpool + "/" + diskInst.Name)
	if err != nil {
		slog.Error("getDiskInfoZVOL GetZfsVolumeUsage error", "err", err)
		diskUsageNum = 0
	}

	diskBlocks := strconv.FormatInt(int64(diskUsageNum), 10)
	diskSize := strconv.FormatInt(int64(diskSizeNum), 10)

	return diskSize, diskBlocks, diskSizeNum, diskUsageNum, nil
}

func getDiskInfoFile(diskInst *disk.Disk) (string, string, uint64, uint64, error) {
	var stat syscall.Stat_t
	var blockSize int64 = 512
	diskPath, err := diskInst.GetPath()
	if err != nil {
		slog.Error("getDiskInfoFile error getting disk size", "err", err)

		return "", "", 0, 0, err
	}

	diskFileStat, err := os.Stat(diskPath)
	if err != nil {
		slog.Error("getDiskInfoFile error getting disk size", "err", err)

		return "", "", 0, 0, err
	}

	err = syscall.Stat(diskPath, &stat)
	if err != nil {
		slog.Error("getDiskInfoFile unable to stat diskPath", "diskPath", diskPath)

		return "", "", 0, 0, err
	}

	diskSize := strconv.FormatInt(diskFileStat.Size(), 10)
	diskBlocks := strconv.FormatInt(stat.Blocks*blockSize, 10)
	diskSizeNum := uint64(diskFileStat.Size())
	diskUsageNum := uint64(stat.Blocks * blockSize)

	return diskSize, diskBlocks, diskSizeNum, diskUsageNum, nil
}
