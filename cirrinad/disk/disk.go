package disk

import (
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
	"database/sql"
	"errors"
	"fmt"
	"golang.org/x/exp/slog"
	"os/exec"
	"strconv"
	"strings"
)

func parseDiskSize(size string) (sizeBytes uint64, err error) {
	var t string
	var n uint
	var m uint64
	if strings.HasSuffix(size, "k") {
		t = strings.TrimSuffix(size, "k")
		m = 1024
	} else if strings.HasSuffix(size, "K") {
		t = strings.TrimSuffix(size, "K")
		m = 1024
	} else if strings.HasSuffix(size, "m") {
		t = strings.TrimSuffix(size, "m")
		m = 1024 * 1024
	} else if strings.HasSuffix(size, "M") {
		t = strings.TrimSuffix(size, "M")
		m = 1024 * 1024
	} else if strings.HasSuffix(size, "g") {
		t = strings.TrimSuffix(size, "g")
		m = 1024 * 1024 * 1024
	} else if strings.HasSuffix(size, "G") {
		t = strings.TrimSuffix(size, "G")
		m = 1024 * 1024 * 1024
	} else if strings.HasSuffix(size, "t") {
		t = strings.TrimSuffix(size, "t")
		m = 1024 * 1024 * 1024 * 1024
	} else if strings.HasSuffix(size, "T") {
		t = strings.TrimSuffix(size, "T")
		m = 1024 * 1024 * 1024 * 1024
	} else if strings.HasSuffix(size, "b") {
		t = strings.TrimSuffix(size, "b")
		m = 1024 * 1024 * 1024 * 1024
	} else if strings.HasSuffix(size, "B") {
		t = size
		m = 1
	} else {
		t = size
		m = 1
	}
	nu, err := strconv.Atoi(t)
	if err != nil {
		return 0, err
	}
	n = uint(nu)
	r := uint64(n) * m
	return r, nil
}

func Create(name string, description string, size string, diskType string, diskDevType string, diskCache bool, diskDirect bool) (disk *Disk, err error) {
	var diskInst *Disk
	var diskSize uint64

	filePath := config.Config.Disk.VM.Path.Image + "/" + name
	volName := config.Config.Disk.VM.Path.Zpool + "/" + name
	volPath := "/dev/zvol/" + volName

	// check disk name
	if !util.ValidDiskName(name) {
		return &Disk{}, errors.New("invalid name")
	}

	// check db for existing disk
	existingDisk, err := GetByName(name)
	if err != nil {
		slog.Error("error checking db for disk", "name", name, "err", err)
		return &Disk{}, err
	}
	if existingDisk.Name != "" {
		slog.Error("disk exists", "disk", name)
		return &Disk{}, errors.New("disk exists")
	}

	// check disk size
	if size == "" {
		diskSize, err = parseDiskSize(config.Config.Disk.Default.Size)
		if err != nil {
			return &Disk{}, err
		}
	} else {
		diskSize, err = parseDiskSize(size)
		if diskSize == 0 || err != nil {
			return &Disk{}, errors.New("invalid disk size")
		}
		// limit disks to min 512 bytes
		if diskSize < 512 {
			diskSize = 512
		}
		// limit disks to max 128TB
		if diskSize > 1024*1024*1024*1024*128 {
			diskSize = 1024 * 1024 * 1024 * 1024 * 128
		}
	}

	// check disk type
	if diskType != "NVME" && diskType != "AHCI-HD" && diskType != "VIRTIO-BLK" {
		slog.Error("disk create", "msg", "invalid disk type", "diskType", diskType)
		return &Disk{}, err
	}

	// check disk dev type
	if diskDevType != "FILE" && diskDevType != "ZVOL" {
		slog.Error("disk create", "msg", "invalid disk dev type", "diskDevType", diskDevType)
		return &Disk{}, err
	}

	// check system for existing disk
	if diskDevType == "FILE" {
		// for files, just check the filePath
		filePath = config.Config.Disk.VM.Path.Image + "/" + name + ".img"
		diskExists, err := util.PathExists(filePath)
		if err != nil {
			slog.Error("error checking if disk exists", "filePath", filePath, "err", err)
			return &Disk{}, err
		}
		if diskExists {
			slog.Error("disk exists", "disk", name)
			return &Disk{}, errors.New("disk exists")
		}
	} else if diskDevType == "ZVOL" {
		// for zvols, check both the volPath and the volume name in zfs list
		diskExists, err := util.PathExists(volPath)
		if err != nil {
			slog.Error("error checking if disk exists", "volPath", volPath, "err", err)
			return diskInst, err
		}
		if diskExists {
			slog.Error("disk exists", "disk", name, "volPath", volPath)
			return &Disk{}, errors.New("disk exists")
		}

		allVolumes, err := GetAllZfsVolumes()
		if err != nil {
			slog.Error("error checking if disk exists", "volName", volName, "err", err)
			return &Disk{}, err
		}
		if util.ContainsStr(allVolumes, volName) {
			slog.Error("disk exists", "disk", name, "volName", volName)
			return &Disk{}, errors.New("disk exists")
		}
	}

	diskInst = &Disk{
		Description: description,
		Type:        diskType,
		DevType:     diskDevType,
		DiskCache:   sql.NullBool{Bool: diskCache, Valid: true},
		DiskDirect:  sql.NullBool{Bool: diskDirect, Valid: true},
	}

	// actually create disk!
	if diskDevType == "FILE" {
		args := []string{"/usr/bin/truncate", "-s", strconv.FormatUint(diskSize, 10), filePath}
		slog.Debug("creating disk", "filePath", filePath, "size", diskSize, "args", args)
		cmd := exec.Command(config.Config.Sys.Sudo, args...)
		err = cmd.Run()
		if err != nil {
			slog.Error("failed to create disk", "err", err)
			return &Disk{}, err
		}
		diskInst.Name = name + ".img"
		diskInst.Path = filePath
	} else if diskDevType == "ZVOL" {
		args := []string{"zfs", "create", "-o", "volmode=dev", "-V", size, "-s", volName}
		slog.Debug("creating disk", "volName", volName, "size", diskSize, "args", args)
		cmd := exec.Command(config.Config.Sys.Sudo, args...)
		err = cmd.Run()
		if err != nil {
			slog.Error("failed to create disk", "err", err)
			return &Disk{}, err
		}
		diskInst.Name = name
		diskInst.Path = volPath
	}

	db := getDiskDb()
	res := db.Create(&diskInst)
	return diskInst, res.Error
}

func GetAll() []*Disk {
	var result []*Disk
	db := getDiskDb()
	db.Find(&result)
	return result
}

func GetById(id string) (d *Disk, err error) {
	db := getDiskDb()
	db.Limit(1).Find(&d, "id = ?", id)
	return d, nil
}

func GetByName(name string) (d *Disk, err error) {
	db := getDiskDb()
	db.Limit(1).Find(&d, "name = ?", name+".img")
	return d, nil
}

func Delete(id string) (err error) {
	if id == "" {
		return errors.New("unable to delete, disk id empty")
	}
	db := getDiskDb()
	dDisk, err := GetById(id)
	if err != nil {
		errorText := fmt.Sprintf("disk %v not found", id)
		return errors.New(errorText)
	}
	res := db.Limit(1).Delete(&dDisk)
	if res.RowsAffected == 1 {
		return nil
	} else {
		errText := fmt.Sprintf("disk delete error, rows affected %v", res.RowsAffected)
		return errors.New(errText)
	}
}
