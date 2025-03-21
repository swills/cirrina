package vm

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cast"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/iso"
)

type Singleton struct {
	VMDB *gorm.DB
}

var Instance *Singleton

var dbInitialized bool

func DBReconfig() {
	dbInitialized = false
}

func GetVMDB() *gorm.DB {
	noColorLogger := logger.New(
		log.New(os.Stdout, "VmDb: ", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		},
	)

	// allow override for testing
	if Instance != nil {
		return Instance.VMDB
	}

	if !dbInitialized {
		Instance = &Singleton{}
		vmDB, err := gorm.Open(
			sqlite.Open(config.Config.DB.Path),
			&gorm.Config{
				Logger:      noColorLogger,
				PrepareStmt: true,
			},
		)

		if err != nil {
			slog.Error("failed to connect to database", "err", err)
			panic("failed to connect database, err: " + err.Error())
		}

		sqlDB, err := vmDB.DB()
		if err != nil {
			slog.Error("failed to create sqlDB", "err", err)
			panic("failed to create sqlDB database, err: " + err.Error())
		}

		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)

		Instance.VMDB = vmDB
		dbInitialized = true
	}

	return Instance.VMDB
}

func (v *VM) SetRunning(pid uint32) {
	vmdb := GetVMDB()
	defer v.mu.Unlock()
	v.mu.Lock()
	v.Status = RUNNING
	v.BhyvePid = pid

	res := vmdb.Select([]string{
		"status",
		"bhyve_pid",
		"com_devs",
	}).Model(&v).
		Updates(map[string]interface{}{
			"status":    &v.Status,
			"bhyve_pid": &v.BhyvePid,
			"com1_dev":  &v.Com1Dev,
			"com2_dev":  &v.Com2Dev,
			"com3_dev":  &v.Com3Dev,
			"com4_dev":  &v.Com4Dev,
		})
	if res.Error != nil {
		slog.Error("error saving VM running", "err", res.Error)
	}
}

func (v *VM) SetStarting() {
	db := GetVMDB()
	defer v.mu.Unlock()
	v.mu.Lock()
	v.Status = STARTING

	res := db.Select([]string{
		"status",
	}).Model(&v).
		Updates(map[string]interface{}{
			"status": &v.Status,
		})
	if res.Error != nil {
		slog.Error("error saving VM start", "err", res.Error)
	}
}

// SetStopped this can in some cases get called on already stopped/deleted VMs and that's OK
func (v *VM) SetStopped() error {
	vmDB := GetVMDB()
	defer v.mu.Unlock()
	v.mu.Lock()
	v.Status = STOPPED
	v.VNCPort = 0
	v.DebugPort = 0
	v.BhyvePid = 0
	v.Com1Dev = ""
	v.Com2Dev = ""
	v.Com3Dev = ""
	v.Com4Dev = ""

	res := vmDB.Select([]string{
		"status",
		"net_dev",
		"vnc_port",
		"debug_port",
		"bhyve_pid",
		"com1_dev",
		"com2_dev",
		"com3_dev",
		"com4_dev",
	}).Model(&v).
		Updates(map[string]interface{}{
			"status":     &v.Status,
			"vnc_port":   &v.VNCPort,
			"debug_port": &v.DebugPort,
			"bhyve_pid":  &v.BhyvePid,
			"com1_dev":   &v.Com1Dev,
			"com2_dev":   &v.Com2Dev,
			"com3_dev":   &v.Com3Dev,
			"com4_dev":   &v.Com4Dev,
		})
	if res.Error != nil {
		slog.Error("error saving VM stopped", "err", res.Error)

		return fmt.Errorf("error saving VM: %w", res.Error)
	}

	return nil
}

func (v *VM) SetStopping() {
	db := GetVMDB()
	defer v.mu.Unlock()
	v.mu.Lock()
	v.Status = STOPPING

	res := db.Select([]string{
		"status",
	}).Model(&v).
		Updates(map[string]interface{}{
			"status": &v.Status,
		})
	if res.Error != nil {
		slog.Error("error saving VM stopping", "err", res.Error)
	}
}

func (v *VM) SetVNCPort(port uint16) {
	slog.Debug("SetVNCPort", "port", port)
	defer v.mu.Unlock()
	v.mu.Lock()
	v.VNCPort = cast.ToInt32(port)
	_ = v.Save()
}

func (v *VM) SetDebugPort(port uint16) {
	slog.Debug("SetDebugPort", "port", port)
	defer v.mu.Unlock()
	v.mu.Lock()
	v.DebugPort = cast.ToInt32(port)
	_ = v.Save()
}

func (v *VM) BeforeCreate(_ *gorm.DB) error {
	if v == nil || v.Name == "" {
		return errVMInvalidName
	}

	err := uuid.Validate(v.ID)
	if err != nil || len(v.ID) != 36 {
		v.ID = uuid.NewString()
	}

	return nil
}

func DBAutoMigrate() {
	vmdb := GetVMDB()

	err := vmdb.AutoMigrate(&VM{})
	if err != nil {
		slog.Error("failed to auto-migrate VMs", "err", err)
		panic("failed to auto-migrate VMs, err: " + err.Error())
	}

	err = vmdb.AutoMigrate(&Config{})
	if err != nil {
		slog.Error("failed db migration", "err", err)
		panic("failed to auto-migrate Configs")
	}
}

func CacheInit() {
	defer List.Mu.Unlock()
	List.Mu.Lock()

	allVMs, err := GetAllDB()
	if err != nil {
		slog.Error("failed to init cache", "err", err)
		panic(err)
	}

	for _, vmInst := range allVMs {
		vmInst.initVM()
	}
}

func getIsosForVM(vmID string, vmDB *gorm.DB) ([]*iso.ISO, error) {
	var returnISOs []*iso.ISO

	if vmID == "" {
		return returnISOs, errVMIDEmptyOrInvalid
	}

	res := vmDB.Table("vm_isos").Select([]string{"vm_id", "iso_id", "position"}).
		Where("vm_id LIKE ?", vmID).Order("position")

	rows, rowErr := res.Rows()
	if rowErr != nil {
		slog.Error("error getting vm_isos rows", "rowErr", rowErr)

		return returnISOs, fmt.Errorf("error getting VM ISOs: %w", rowErr)
	}

	err := rows.Err()
	if err != nil {
		slog.Error("error getting vm_isos rows", "err", err)

		return returnISOs, fmt.Errorf("error getting VM ISOs: %w", rowErr)
	}

	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	for rows.Next() {
		var vmISOsVMID string

		var vmISOsIsoID string

		var vmISOsPosition int

		err = rows.Scan(&vmISOsVMID, &vmISOsIsoID, &vmISOsPosition)
		if err != nil {
			slog.Error("error scanning vm_isos", "err", err)

			continue
		}

		slog.Debug("found a vm_iso",
			"vmISOsVMID", vmISOsVMID,
			"vmISOsIsoID", vmISOsIsoID,
			"vmISOsPosition", vmISOsPosition,
		)

		var thisVMIso *iso.ISO

		thisVMIso, err = iso.GetByID(vmISOsIsoID)
		if err != nil {
			slog.Error("error looking up VM ISO", "err", err)

			continue
		}

		returnISOs = append(returnISOs, thisVMIso)
	}

	return returnISOs, nil
}

func getDisksForVM(vmID string, vmDB *gorm.DB) ([]*disk.Disk, error) {
	var returnDisks []*disk.Disk

	if vmID == "" {
		return returnDisks, errVMIDEmptyOrInvalid
	}

	res := vmDB.Table("vm_disks").Select([]string{"vm_id", "disk_id", "position"}).
		Where("vm_id LIKE ?", vmID).Order("position")

	rows, rowErr := res.Rows()
	if rowErr != nil {
		slog.Error("error getting vm_disks rows", "rowErr", rowErr)

		return returnDisks, fmt.Errorf("error getting VM Disks: %w", rowErr)
	}

	err := rows.Err()
	if err != nil {
		slog.Error("error getting vm_disks rows", "err", err)

		return returnDisks, fmt.Errorf("error getting VM Disks: %w", rowErr)
	}

	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	for rows.Next() {
		var vmDisksVMID string

		var vmDisksDiskID string

		var vmDisksPosition int

		err = rows.Scan(&vmDisksVMID, &vmDisksDiskID, &vmDisksPosition)
		if err != nil {
			slog.Error("error scanning vm_disks", "err", err)

			continue
		}

		slog.Debug("found a vm_disk",
			"vmDisksVMID", vmDisksVMID,
			"vmDisksDiskID", vmDisksDiskID,
			"vmDisksPosition", vmDisksPosition,
		)

		var thisVMDisk *disk.Disk

		thisVMDisk, err = disk.GetByID(vmDisksDiskID)
		if err != nil {
			slog.Error("error looking up VM Disk", "err", err)

			continue
		}

		returnDisks = append(returnDisks, thisVMDisk)
	}

	return returnDisks, nil
}

func GetAllDB() ([]*VM, error) {
	var result []*VM

	var err error

	vmDB := GetVMDB()

	res := vmDB.Preload("Config").Find(&result)
	if res.Error != nil {
		slog.Error("error looking up VMs", "resErr", res.Error)

		return nil, res.Error
	}

	// manually load VM Disks/ISOs because GORM can't do what is needed in terms of allowing duplicates or
	// preserving position
	for _, vmResult := range result {
		vmResult.ISOs, err = getIsosForVM(vmResult.ID, vmDB)
		if err != nil {
			slog.Error("failed getting isos for VM", "err", err)

			return nil, err
		}

		vmResult.Disks, err = getDisksForVM(vmResult.ID, vmDB)
		if err != nil {
			slog.Error("failed getting disks for VM", "err", err)

			return nil, err
		}
	}

	return result, nil
}
