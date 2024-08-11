package disk

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"gorm.io/gorm"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

type Disk struct {
	ID          string `gorm:"uniqueIndex;not null;default:null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Name        string         `gorm:"uniqueIndex;not null;default:null"`
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

type InfoServicer interface {
	GetSize(name string) (uint64, error)
	GetUsage(name string) (uint64, error)
	SetSize(name string, newSize uint64) error
	Exists(name string) (bool, error)
	Create(name string, size uint64) error
	GetAll() ([]string, error)
}

var List = &ListType{
	DiskList: make(map[string]*Disk),
}

var PathExistsFunc = util.PathExists
var diskExistsCacheDBFunc = diskExistsCacheDB
var validateDiskFunc = validateDisk
var getDiskDBFunc = GetDiskDB
var FileInfoFetcherImpl FileInfoFetcher = FileInfoCmds{}
var ZfsInfoFetcherImpl ZfsVolInfoFetcher = ZfsVolInfoCmds{}
var GetByNameFunc = GetByName

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

// diskExistsCacheDB checks if a disk exists in the in-memory cache or in the database
func diskExistsCacheDB(diskInst *Disk) (bool, error) {
	var err error

	// check in memory cache for disk
	memDiskInst, err := GetByNameFunc(diskInst.Name)

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

	return false, nil
}

func Create(diskInst *Disk, size string) error {
	var err error

	var exists bool

	var diskSize uint64

	var diskService InfoServicer

	switch diskInst.DevType {
	case "FILE":
		diskService = NewFileInfoService(FileInfoFetcherImpl)

	case "ZVOL":
		diskService = NewZfsVolInfoService(ZfsInfoFetcherImpl)

	default:
		return errDiskInvalidDevType
	}

	// check db for existing disk
	exists, err = diskExistsCacheDBFunc(diskInst)
	if err != nil {
		slog.Error("error checking db for disk", "name", diskInst.Name, "err", err)

		return fmt.Errorf("error checking disk exists: %w", err)
	}

	if exists {
		slog.Error("disk exists", "disk", diskInst.Name)

		return errDiskExists
	}

	// check file system/zpool for disk
	exists, err = diskService.Exists(diskInst.GetPath())
	if err != nil {
		slog.Error("error checking for disk", "name", diskInst.Name, "err", err)

		return fmt.Errorf("error checking disk exists: %w", err)
	}

	if exists {
		slog.Error("disk exists", "disk", diskInst.Name)

		return errDiskExists
	}

	err = validateDiskFunc(diskInst)
	if err != nil {
		return fmt.Errorf("error creating disk: %w", err)
	}

	diskSize, err = util.ParseDiskSize(size)
	if err != nil {
		return fmt.Errorf("error creating disk: %w", err)
	}

	// actually create disk!
	err = diskService.Create(diskInst.GetPath(), diskSize)
	if err != nil {
		return fmt.Errorf("erro creating disk: %w", err)
	}

	db := getDiskDBFunc()

	res := db.Create(&diskInst)

	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected != 1 {
		return fmt.Errorf("db err: %w, incorrect number of rows affected: %d", errDiskInternalDB, res.RowsAffected)
	}

	defer List.Mu.Unlock()
	List.Mu.Lock()
	diskInst.initOneDisk()

	return nil
}

func GetAllDB() []*Disk {
	var result []*Disk

	db := GetDiskDB()
	db.Find(&result)

	return result
}

// GetByID lookup disk by ID from in-memory disk list
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

// GetByName lookups disk by name from in-memory disk list
func GetByName(name string) (*Disk, error) {
	for _, diskInst := range List.DiskList {
		if diskInst.Name == name {
			return diskInst, nil
		}
	}

	return nil, errDiskNotFound
}

func Delete(diskID string) error {
	if diskID == "" {
		return errDiskIDEmptyOrInvalid
	}

	_, valid := List.DiskList[diskID]
	if !valid {
		return errDiskIDEmptyOrInvalid
	}

	db := GetDiskDB()

	res := db.Limit(1).Delete(&Disk{ID: diskID})

	if res.Error != nil || res.RowsAffected != 1 {
		slog.Error("error saving disk", "res", res)

		return errDiskInternalDB
	}

	delete(List.DiskList, diskID)

	return nil
}

func (d *Disk) Save() error {
	db := GetDiskDB()

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
	exists, err = PathExistsFunc(diskPath)
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

// initOneDisk initializes and adds a Disk to the in memory cache of Disks
// note, callers must lock the in memory cache via List.Mu.Lock()
func (d *Disk) initOneDisk() {
	if d == nil {
		return
	}

	List.DiskList[d.ID] = d
}
