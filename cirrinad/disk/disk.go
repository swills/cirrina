package disk

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os/user"
	"path/filepath"
	"strconv"
	"sync"

	exec "golang.org/x/sys/execabs"
	"gorm.io/gorm"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

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

func Create(diskInst *Disk, size string) error {
	var err error
	var diskSize uint64

	// check db for existing disk
	diskAlreadyExists, err := diskExists(diskInst.Name)
	if err != nil {
		slog.Error("error checking db for disk", "name", diskInst.Name, "err", err)

		return err
	}
	if diskAlreadyExists {
		slog.Error("disk exists", "disk", diskInst.Name)

		return errDiskExists
	}

	err = validateDisk(diskInst)
	if err != nil {
		return fmt.Errorf("error creating disk: %w", err)
	}

	diskSize, err = util.ParseDiskSize(size)
	if err != nil {
		return fmt.Errorf("error creating disk: %w", err)
	}

	// actually create disk!
	switch diskInst.DevType {
	case "FILE":
		err := createDiskFile(diskInst.GetPath(), diskSize)
		if err != nil {
			return err
		}
	case "ZVOL":
		err := createDiskZvol(diskInst.GetPath(), diskSize)
		if err != nil {
			return err
		}
	default:
		return errDiskInvalidDevType
	}

	db := getDiskDB()
	res := db.Create(&diskInst)
	if res.RowsAffected != 1 {
		return fmt.Errorf("incorrect number of rows affected, err: %w", res.Error)
	}
	if res.Error != nil {
		return res.Error
	}
	List.DiskList[diskInst.ID] = diskInst

	return nil
}

func diskDevTypeValid(diskDevType string) bool {
	switch diskDevType {
	case "FILE":
		return true
	case "ZVOL":
		return true
	default:
		return false
	}
}

func diskTypeValid(diskType string) bool {
	// check disk type
	switch diskType {
	case "NVME":
		return true
	case "AHCI-HD":
		return true
	case "VIRTIO-BLK":
		return true
	default:
		return false
	}
}

func checkDiskExistsZvolType(name string, volName string) error {
	// for zvols, check both the volName and the volume name in zfs list
	allVolumes, err := GetAllZfsVolumes()
	if err != nil {
		slog.Error("error checking if disk exists", "err", err)

		return fmt.Errorf("error checking if disk exists: %w", err)
	}
	if util.ContainsStr(allVolumes, volName) {
		slog.Error("disk volume exists", "disk", name, "volName", volName)

		return errDiskExists
	}

	diskExists, err := util.PathExists("/dev/zvol/" + volName)
	if err != nil {
		slog.Error("error checking if disk exists", "volName", volName, "err", err)

		return fmt.Errorf("error checking if disk exists: %w", err)
	}
	if diskExists {
		slog.Error("disk vol path exists", "disk", name, "volName", volName)

		return errDiskExists
	}

	return nil
}

func checkDiskExistsFileType(name string, filePath string) error {
	// for files, just check the filePath
	diskPathExists, err := util.PathExists(filePath)
	if err != nil {
		slog.Error("error checking if disk exists", "filePath", filePath, "err", err)

		return fmt.Errorf("error checking if disk exists: %w", err)
	}
	if diskPathExists {
		slog.Error("disk file exists", "disk", name, "filePath", filePath)

		return errDiskExists
	}

	return nil
}

func createDiskFile(filePath string, diskSize uint64) error {
	args := []string{"/usr/bin/truncate", "-s", strconv.FormatUint(diskSize, 10), filePath}
	slog.Debug("creating disk", "filePath", filePath, "size", diskSize, "args", args)
	myUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("error creating disk: %w", err)
	}
	cmd := exec.Command(config.Config.Sys.Sudo, args...)
	err = cmd.Run()
	if err != nil {
		slog.Error("failed to create disk", "err", err)

		return fmt.Errorf("error creating disk: %w", err)
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

func createDiskZvol(volName string, size uint64) error {
	args := []string{"/sbin/zfs", "create", "-o", "volmode=dev", "-V", strconv.FormatUint(size, 10), "-s", volName}
	slog.Debug("creating disk", "args", args)
	cmd := exec.Command(config.Config.Sys.Sudo, args...)
	err := cmd.Run()
	if err != nil {
		slog.Error("failed to create disk", "err", err)

		return fmt.Errorf("error creating disk: %w", err)
	}

	return nil
}

func GetAllDB() []*Disk {
	var result []*Disk
	db := getDiskDB()
	db.Find(&result)

	return result
}

func GetByID(diskID string) (*Disk, error) {
	if diskID == "" {
		return nil, errDiskIDEmptyOrInvalid
	}
	defer List.Mu.RUnlock()
	List.Mu.RLock()
	diskInst, valid := List.DiskList[diskID]
	if valid {
		return diskInst, nil
	}

	return nil, errDiskNotFound
}

func GetByName(name string) (*Disk, error) {
	for _, diskInst := range List.DiskList {
		if diskInst.Name == name {
			return diskInst, nil
		}
	}

	return nil, errDiskNotFound
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
		slog.Error("error saving disk", "res", res)

		return errDiskInternalDB
	}

	return nil
}

func Delete(diskID string) error {
	if diskID == "" {
		return errDiskIDEmptyOrInvalid
	}

	_, valid := List.DiskList[diskID]
	if !valid {
		return errDiskIDEmptyOrInvalid
	}
	delete(List.DiskList, diskID)

	db := getDiskDB()
	res := db.Limit(1).Delete(&Disk{ID: diskID})
	if res.RowsAffected != 1 {
		slog.Error("error saving disk", "res", res)

		return errDiskInternalDB
	}

	return nil
}

func (d *Disk) GetPath() string {
	var diskPath string
	switch d.DevType {
	case "FILE":
		diskPath = filepath.Join(config.Config.Disk.VM.Path.Image, d.Name+".img")
	case "ZVOL":
		diskPath = filepath.Join(config.Config.Disk.VM.Path.Zpool, d.Name)
	default:
		return ""
	}

	return diskPath
}

func (d *Disk) VerifyExists() (bool, error) {
	var err error
	var exists bool
	var diskPath string
	diskPath = d.GetPath()
	if d.DevType == "ZVOL" {
		diskPath = filepath.Join("/dev/zvol/", diskPath)
	}

	// perhaps it's not necessary to check the volume -- as long as there's a /dev/zvol entry, we're fine, right?
	exists, err = util.PathExists(diskPath)
	if err != nil {
		return exists, fmt.Errorf("failed checking disk exists: %w", err)
	}

	return exists, nil
}

func (d *Disk) Lock() {
	d.mu.Lock()
}

func (d *Disk) Unlock() {
	d.mu.Unlock()
}

func initOneDisk(d *Disk) {
	defer List.Mu.Unlock()
	List.Mu.Lock()
	List.DiskList[d.ID] = d
}

func validateDisk(diskInst *Disk) error {
	if !util.ValidDiskName(diskInst.Name) {
		return errDiskInvalidName
	}
	if diskInst.DevType == "ZVOL" && config.Config.Disk.VM.Path.Zpool == "" {
		return errDiskZPoolNotConfigured
	}
	if !diskTypeValid(diskInst.Type) {
		return errDiskInvalidType
	}
	if !diskDevTypeValid(diskInst.DevType) {
		return errDiskInvalidDevType
	}

	return nil
}

func diskExists(diskName string) (bool, error) {
	var err error
	memDiskInst, err := GetByName(diskName)

	if err != nil {
		if !errors.Is(err, errDiskNotFound) {
			slog.Error("error checking db for disk", "name", diskName, "err", err)

			return false, err
		}

		return false, nil
	}
	if memDiskInst != nil && memDiskInst.Name != "" {
		return true, nil
	}

	allDisks := GetAllDB()
	for _, dbDiskInst := range allDisks {
		if dbDiskInst.Name == diskName {
			return true, nil
		}
	}

	// check system for existing disk
	switch memDiskInst.DevType {
	case "FILE":
		err := checkDiskExistsFileType(memDiskInst.Name, memDiskInst.GetPath())
		if err != nil {
			return false, err
		}
	case "ZVOL":
		err := checkDiskExistsZvolType(memDiskInst.Name, memDiskInst.GetPath())
		if err != nil {
			return false, err
		}
	default:
		return false, errDiskInvalidDevType
	}

	return false, nil
}
