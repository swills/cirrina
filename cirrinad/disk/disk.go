package disk

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os/user"
	"strconv"
	"sync"

	exec "golang.org/x/sys/execabs"
	"gorm.io/gorm"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

func Create(name string, description string, size string, diskType string, diskDevType string, diskCache bool, diskDirect bool) (disk *Disk, err error) {
	var diskInst *Disk
	var diskSize uint64

	// keep this in sync with GetPath()
	filePath := config.Config.Disk.VM.Path.Image + "/" + name + ".img"
	volName := config.Config.Disk.VM.Path.Zpool + "/" + name

	err = validateDisk(name, diskDevType, diskType, filePath, volName)
	if err != nil {
		return &Disk{}, err
	}

	diskSize, err = util.ParseDiskSize(size)
	if err != nil {
		return &Disk{}, err
	}

	// limit disks to min 512 bytes, max 128TB
	if diskSize < 512 || diskSize > 1024*1024*1024*1024*128 {
		return &Disk{}, errors.New("invalid disk size")
	}

	// actually create disk!
	switch diskDevType {
	case "FILE":
		err := createDiskFile(diskSize, filePath)
		if err != nil {
			return &Disk{}, err
		}
	case "ZVOL":
		err := createDiskZvol(volName, size)
		if err != nil {
			return &Disk{}, err
		}
	default:
		return &Disk{}, errors.New("invalid disk type")
	}

	diskInst = &Disk{
		Name:        name,
		Description: description,
		Type:        diskType,
		DevType:     diskDevType,
		DiskCache:   sql.NullBool{Bool: diskCache, Valid: true},
		DiskDirect:  sql.NullBool{Bool: diskDirect, Valid: true},
	}

	// save disk to DB
	db := getDiskDB()
	res := db.Create(&diskInst)
	List.DiskList[diskInst.ID] = diskInst

	return diskInst, res.Error
}

func validateDisk(name string, diskDevType string, diskType string, filePath string, volName string) error {
	if diskDevType == "ZVOL" && config.Config.Disk.VM.Path.Zpool == "" {
		return errors.New("zfs pool not configured, cannot create zvol disks")
	}

	volPath := "/dev/zvol/" + volName

	// check disk name
	if !util.ValidDiskName(name) {
		return errors.New("invalid disk name")
	}

	// check db for existing disk
	existingDisk, err := GetByName(name)
	if err != nil {
		slog.Error("error checking db for disk", "name", name, "err", err)

		return err
	}
	if existingDisk.Name != "" {
		slog.Error("disk exists in DB", "disk", name, "id", existingDisk.ID, "type", existingDisk.Type)

		return fmt.Errorf("disk %s exists in db", name)
	}

	// check disk type
	if diskType != "NVME" && diskType != "AHCI-HD" && diskType != "VIRTIO-BLK" {
		slog.Error("disk create", "msg", "invalid disk type", "diskType", diskType)

		return errors.New("invalid disk type")
	}

	// check disk dev type
	if diskDevType != "FILE" && diskDevType != "ZVOL" {
		slog.Error("disk create", "msg", "invalid disk dev type", "diskDevType", diskDevType)

		return errors.New("invalid disk dev type")
	}

	// check system for existing disk
	if diskDevType == "FILE" {
		err := checkDiskExistsFileType(name, filePath, existingDisk)
		if err != nil {
			return err
		}
	} else if diskDevType == "ZVOL" {
		err := checkDiskExistsZvolType(name, volName, existingDisk, volPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func checkDiskExistsZvolType(name string, volName string, existingDisk *Disk, volPath string) error {
	// for zvols, check both the volPath and the volume name in zfs list
	allVolumes, err := GetAllZfsVolumes()
	if err != nil {
		slog.Error("error checking if disk exists", "volName", volName, "err", err)

		return err
	}
	if util.ContainsStr(allVolumes, volName) {
		slog.Error("disk volume exists", "disk", name, "id", existingDisk.ID, "type", existingDisk.Type, "volName", volName)

		return errors.New("disk exists")
	}

	diskExists, err := util.PathExists(volPath)
	if err != nil {
		slog.Error("error checking if disk exists", "volPath", volPath, "err", err)

		return err
	}
	if diskExists {
		slog.Error("disk vol path exists", "disk", name, "id", existingDisk.ID, "type", existingDisk.Type, "volPath", volPath)

		return errors.New("disk exists")
	}

	return nil
}

func checkDiskExistsFileType(name string, filePath string, existingDisk *Disk) error {
	// for files, just check the filePath
	diskExists, err := util.PathExists(filePath)
	if err != nil {
		slog.Error("error checking if disk exists", "filePath", filePath, "err", err)

		return err
	}
	if diskExists {
		slog.Error("disk file exists", "disk", name, "id", existingDisk.ID, "type", existingDisk.Type, "filePath", filePath)

		return errors.New("disk exists")
	}

	return nil
}

func createDiskFile(diskSize uint64, filePath string) error {
	args := []string{"/usr/bin/truncate", "-s", strconv.FormatUint(diskSize, 10), filePath}
	slog.Debug("creating disk", "filePath", filePath, "size", diskSize, "args", args)
	myUser, err := user.Current()
	if err != nil {
		return err
	}
	cmd := exec.Command(config.Config.Sys.Sudo, args...)
	err = cmd.Run()
	if err != nil {
		slog.Error("failed to create disk", "err", err)

		return err
	}
	args = []string{"/usr/sbin/chown", myUser.Username, filePath}
	cmd = exec.Command(config.Config.Sys.Sudo, args...)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to fix ownership of disk file %s: %w", filePath, err)
	}
	slog.Debug("disk.Create user mismatch fixed")

	return nil
}

func createDiskZvol(volName string, size string) error {
	args := []string{"/sbin/zfs", "create", "-o", "volmode=dev", "-V", size, "-s", volName}
	slog.Debug("creating disk", "args", args)
	cmd := exec.Command(config.Config.Sys.Sudo, args...)
	err := cmd.Run()
	if err != nil {
		slog.Error("failed to create disk", "err", err)

		return err
	}

	return err
}

func GetAllDB() []*Disk {
	var result []*Disk
	db := getDiskDB()
	db.Find(&result)

	return result
}

func GetByID(id string) (*Disk, error) {
	defer List.Mu.RUnlock()
	List.Mu.RLock()
	diskInst, valid := List.DiskList[id]
	if valid {
		return diskInst, nil
	}

	return nil, errors.New("not found")
}

func GetByName(name string) (*Disk, error) {
	for _, diskInst := range List.DiskList {
		if diskInst.Name == name {
			return diskInst, nil
		}
	}

	return &Disk{}, nil
}

func Delete(id string) (err error) {
	if id == "" {
		return errors.New("unable to delete, disk id empty")
	}

	_, valid := List.DiskList[id]
	if !valid {
		return errors.New("invalid disk id")
	}
	delete(List.DiskList, id)

	db := getDiskDB()
	res := db.Limit(1).Delete(&Disk{ID: id})
	if res.RowsAffected == 1 {
		return nil
	} else {
		errText := fmt.Sprintf("disk delete error, rows affected %v", res.RowsAffected)

		return errors.New(errText)
	}
}

func (d *Disk) Save() error {
	db := getDiskDB()

	res := db.Model(&d).
		Updates(map[string]interface{}{
			"description": &d.Description,
			"type":        &d.Type,
			"name":        &d.Name,
		},
		)

	if res.Error != nil {
		return errors.New("error updating disk")
	}

	return nil
}

func (d *Disk) GetPath() (diskPath string, err error) {
	switch d.DevType {
	case "ZVOL":
		diskPath = "/dev/zvol/" + config.Config.Disk.VM.Path.Zpool + "/" + d.Name
	case "FILE":
		diskPath = config.Config.Disk.VM.Path.Image + "/" + d.Name + ".img"
	default:
		return "", errors.New("unknown disk dev type")
	}

	return diskPath, nil
}

func (d *Disk) VerifyExists() (exists bool, err error) {
	var diskPath string
	diskPath, err = d.GetPath()
	if err != nil {
		return false, err
	}

	// perhaps it's not necessary to check the volume -- as long as there's a /dev/zvol entry, we're fine, right?
	exists, err = util.PathExists(diskPath)

	return exists, err
}

func (d *Disk) Lock() {
	d.mu.Lock()
}

func (d *Disk) Unlock() {
	d.mu.Unlock()
}

func InitOneDisk(d *Disk) {
	defer List.Mu.Unlock()
	List.Mu.Lock()
	List.DiskList[d.ID] = d
}

type Disk struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null;default:null"`
	Name        string `gorm:"uniqueIndex;not null;default:null"`
	Description string
	Type        string       `gorm:"default:NVME;check:type IN ('NVME','AHCI-HD','VIRTIO-BLK')"`
	DevType     string       `gorm:"default:FILE;check:dev_type IN ('FILE','ZVOL')"`
	DiskCache   sql.NullBool `gorm:"default:True;check:disk_cache IN(0,1)"`
	DiskDirect  sql.NullBool `gorm:"default:False;check:disk_direct IN(0,1)"`
	mu          sync.Mutex
}

type ListType struct {
	Mu       sync.RWMutex
	DiskList map[string]*Disk
}

var List = &ListType{
	DiskList: make(map[string]*Disk),
}
