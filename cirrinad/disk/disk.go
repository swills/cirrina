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
	diskAlreadyExists, err := diskExists(diskInst)
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

func checkDiskExistsZvolType(name string, volName string) (bool, error) {
	// for zvols, check both the volName and the volume name in zfs list
	allVolumes, err := GetAllZfsVolumes()
	if err != nil {
		slog.Error("error checking if disk exists", "err", err)

		// assume disks exists if there's an error checking to be on safe side
		return true, fmt.Errorf("error checking if disk exists: %w", err)
	}

	if util.ContainsStr(allVolumes, volName) {
		slog.Error("disk volume exists", "disk", name, "volName", volName)

		return true, nil
	}

	diskExists, err := util.PathExists("/dev/zvol/" + volName)
	if err != nil {
		slog.Error("error checking if disk exists", "volName", volName, "err", err)

		// assume disks exists if there's an error checking to be on safe side
		return true, fmt.Errorf("error checking if disk exists: %w", err)
	}

	if diskExists {
		slog.Error("disk vol path exists", "disk", name, "volName", volName)

		return true, nil
	}

	return false, nil
}

func checkDiskExistsFileType(name string, filePath string) (bool, error) {
	// for files, just check the filePath
	diskPathExists, err := util.PathExists(filePath)
	if err != nil {
		slog.Error("error checking if disk exists", "filePath", filePath, "err", err)

		// assume disks exists if there's an error checking to be on safe side
		return true, fmt.Errorf("error checking if disk exists: %w", err)
	}

	if diskPathExists {
		slog.Error("disk file exists", "disk", name, "filePath", filePath)

		return true, nil
	}

	return false, nil
}

func createDiskFile(filePath string, diskSize uint64) error {
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/bin/truncate", "-s", strconv.FormatUint(diskSize, 10), filePath},
	)
	if err != nil {
		slog.Error("failed to create disk",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("error creating disk: %w", err)
	}

	myUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("error creating disk: %w", err)
	}

	stdOutBytes, stdErrBytes, returnCode, err = util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/chown", myUser.Username, filePath},
	)
	if err != nil {
		slog.Error("failed to fix ownership of disk file",
			"filePath", filePath,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("failed to fix ownership of disk file %s: %w", filePath, err)
	}

	return nil
}

func createDiskZvol(volName string, size uint64) error {
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/zfs", "create", "-o", "volmode=dev", "-V", strconv.FormatUint(size, 10), "-s", volName},
	)
	if err != nil {
		slog.Error("failed to create disk",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

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

// GetByID lookup disk by ID from in memory disk list
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

// GetByName lookup disk by name from in memory disk list
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
			"name":        &d.Name,
			"description": &d.Description,
			"type":        &d.Type,
			"dev_type":    &d.DevType,
			"disk_cache":  &d.DiskCache,
			"disk_direct": &d.DiskDirect,
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

// GetPath return path to disk to use with bhyve -- either full disk path for file
// or zvol name
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

	if !diskTypeValid(diskInst.Type) {
		return errDiskInvalidType
	}

	if !diskDevTypeValid(diskInst.DevType) {
		return errDiskInvalidDevType
	}

	if diskInst.DevType == "ZVOL" && config.Config.Disk.VM.Path.Zpool == "" {
		return errDiskZPoolNotConfigured
	}

	return nil
}

func diskExists(diskInst *Disk) (bool, error) {
	var err error

	// check in memory for disk
	memDiskInst, err := GetByName(diskInst.Name)

	if err != nil {
		// if errDiskNotFound, check other places just to be sure
		// if not errDiskNotFound, there must be some internal issue, play it safe
		if !errors.Is(err, errDiskNotFound) {
			slog.Error("error checking db for disk", "name", diskInst.Name, "err", err)

			// assume disks exists if there's an error checking to be on safe side
			return true, err
		}
	}

	// check db for disk
	if memDiskInst != nil && memDiskInst.Name != "" {
		return true, nil
	}

	allDisks := GetAllDB()
	for _, dbDiskInst := range allDisks {
		if dbDiskInst.Name == diskInst.Name {
			return true, nil
		}
	}

	// check file system/zpool for disk
	diskFileExists, err := checkDiskExistsFileType(diskInst.Name, diskInst.GetPath())
	if err != nil {
		return true, err
	}

	if diskFileExists {
		return true, nil
	}

	diskZvolExists, err := checkDiskExistsZvolType(diskInst.Name, diskInst.GetPath())
	if err != nil {
		return true, err
	}

	if diskZvolExists {
		return true, nil
	}

	return false, nil
}
