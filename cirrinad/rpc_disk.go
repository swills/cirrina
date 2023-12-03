package main

import (
	"context"
	"errors"
	"fmt"
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
	var disks []*disk.Disk
	var DiskId cirrina.DiskId
	disks = disk.GetAll()
	for e := range disks {
		DiskId.Value = disks[e].ID
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

	deafultDiskDescription := ""
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
		i.Description = &deafultDiskDescription
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
	allVMs := vm.GetAll()
	for _, thisVm := range allVMs {
		slog.Debug("vm checks", "vm", thisVm)
		thisVmDisks, err := thisVm.GetDisks()
		if err != nil {
			return &re, err
		}
		for _, vmDisk := range thisVmDisks {
			if vmDisk.ID == diskUuid.String() {
				errorMessage := fmt.Sprintf("disk in use by VM %s", thisVm.ID)
				return &re, errors.New(errorMessage)
			}
		}
	}

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

	allVMs := vm.GetAll()
	found := false
	for _, thisVm := range allVMs {
		thisVmDisks, err := thisVm.GetDisks()
		if err != nil {
			return nil, err
		}
		for _, vmDisk := range thisVmDisks {
			if vmDisk.ID == diskUuid.String() {
				if found == true {
					slog.Error("GetDiskVm disk in use by more than one VM",
						"diskid", i.Value,
						"vmid", thisVm.ID,
					)
					return nil, errors.New("disk in use by more than one VM")
				}
				found = true
				pvmId.Value = thisVm.ID
			}
		}
	}

	if pvmId.Value == "" {
		// no VMs use this disk
		return &pvmId, errors.New("not found")
	}
	return &pvmId, nil
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

	DiskInst, err := disk.GetById(DiskUuid.String())
	if err != nil {
		return &re, err
	}

	if diu.Description != nil {
		DiskInst.Description = *diu.Description
		slog.Debug("SetDiskInfo", "description", *diu.Description)
	}

	if diu.DiskType != nil {
		if *diu.DiskType == cirrina.DiskType_NVME {
			DiskInst.Type = "NVME"
		} else if *diu.DiskType == cirrina.DiskType_AHCIHD {
			DiskInst.Type = "AHCI-HD"
		} else if *diu.DiskType == cirrina.DiskType_VIRTIOBLK {
			DiskInst.Type = "VIRTIO-BLK"
		} else {
			return &re, errors.New("invalid disk type")
		}
		slog.Debug("SetDiskInfo", "type", DiskInst.Type)
	}

	slog.Debug("SetDiskInfo saving disk")
	err = DiskInst.Save()
	if err != nil {
		return &re, errors.New("failed to update Disk")
	}
	re.Success = true

	return &re, nil
}
