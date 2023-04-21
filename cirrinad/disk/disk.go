package disk

import (
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
	"errors"
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
	// limit disks to 128TB
	if r > 1024*1024*1024*1024*128 {
		r = 1024 * 1024 * 1024 * 1024 * 128
	}
	return r, nil
}

func Create(name string, description string, size string) (disk *Disk, err error) {
	var diskInst *Disk
	if strings.Contains(name, "/") {
		return diskInst, errors.New("illegal character in disk name")
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
