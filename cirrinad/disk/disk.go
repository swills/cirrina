package disk

import (
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
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

func Create(name string, description string, size string, diskType string) (disk *Disk, err error) {
	var diskInst *Disk
	if !util.ValidDiskName(name) {
		return diskInst, errors.New("invalid name")
	}
	path := config.Config.Disk.VM.Path.Image + "/" + name + ".img"
	diskExists, err := util.PathExists(path)
	if err != nil {
		slog.Error("error checking if disk exists", "path", path, "err", err)
		return diskInst, err
	}
	if diskExists {
		slog.Error("disk exists", "disk", name)
		return diskInst, errors.New("disk exists")
	}
	existingDisk, err := GetByName(name)
	if err != nil {
		slog.Error("error checking db for disk", "name", name, "err", err)
		return diskInst, err
	}
	if existingDisk.Name != "" {
		slog.Error("disk exists", "disk", name)
		return diskInst, errors.New("disk exists")
	}

	var diskSize uint64
	if size == "" {
		diskSize, err = parseDiskSize(config.Config.Disk.Default.Size)
		if err != nil {
			return diskInst, err
		}
	} else {
		diskSize, err = parseDiskSize(size)
		if diskSize == 0 || err != nil {
			return diskInst, errors.New("invalid disk size")
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

	if diskType != "NVME" && diskType != "AHCI-HD" && diskType != "VIRTIO-BLK" {
		slog.Error("disk create", "msg", "invalid disk type", "diskType", diskType)
		return diskInst, err
	}

	args := []string{"/usr/bin/truncate", "-s", strconv.FormatUint(diskSize, 10), path}
	slog.Debug("creating disk", "path", path, "size", diskSize, "args", args)
	cmd := exec.Command(config.Config.Sys.Sudo, args...)
	err = cmd.Run()
	if err != nil {
		slog.Error("failed to create disk", "err", err)
		return diskInst, err
	}
	diskInst = &Disk{
		Name:        name + ".img",
		Description: description,
		Path:        path,
		Type:        diskType,
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
