package vm

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/iso"
)

type singleton struct {
	vmDB *gorm.DB
}

var Instance *singleton

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
		return Instance.vmDB
	}

	if !dbInitialized {
		Instance = &singleton{}
		vmDB, err := gorm.Open(
			sqlite.Open(config.Config.DB.Path),
			&gorm.Config{
				Logger:      noColorLogger,
				PrepareStmt: true,
			},
		)

		if err != nil {
			panic("failed to connect database")
		}

		sqlDB, err := vmDB.DB()
		if err != nil {
			panic("failed to create sqlDB database")
		}

		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)

		Instance.vmDB = vmDB
		dbInitialized = true
	}

	return Instance.vmDB
}

func (vm *VM) SetRunning(pid int) {
	vmdb := GetVMDB()
	defer vm.mu.Unlock()
	vm.mu.Lock()
	vm.Status = RUNNING
	vm.BhyvePid = uint32(pid)

	res := vmdb.Select([]string{
		"status",
		"bhyve_pid",
		"com_devs",
	}).Model(&vm).
		Updates(map[string]interface{}{
			"status":    &vm.Status,
			"bhyve_pid": &vm.BhyvePid,
			"com1_dev":  &vm.Com1Dev,
			"com2_dev":  &vm.Com2Dev,
			"com3_dev":  &vm.Com3Dev,
			"com4_dev":  &vm.Com4Dev,
		})
	if res.Error != nil {
		slog.Error("error saving VM running", "err", res.Error)
	}
}

func (vm *VM) SetStarting() {
	db := GetVMDB()
	defer vm.mu.Unlock()
	vm.mu.Lock()
	vm.Status = STARTING

	res := db.Select([]string{
		"status",
	}).Model(&vm).
		Updates(map[string]interface{}{
			"status": &vm.Status,
		})
	if res.Error != nil {
		slog.Error("error saving VM start", "err", res.Error)
	}
}

// SetStopped this can in some cases get called on already stopped/deleted VMs and that's OK
func (vm *VM) SetStopped() error {
	vmDB := GetVMDB()
	defer vm.mu.Unlock()
	vm.mu.Lock()
	vm.Status = STOPPED
	vm.VNCPort = 0
	vm.DebugPort = 0
	vm.BhyvePid = 0
	vm.Com1Dev = ""
	vm.Com2Dev = ""
	vm.Com3Dev = ""
	vm.Com4Dev = ""

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
	}).Model(&vm).
		Updates(map[string]interface{}{
			"status":     &vm.Status,
			"vnc_port":   &vm.VNCPort,
			"debug_port": &vm.DebugPort,
			"bhyve_pid":  &vm.BhyvePid,
			"com1_dev":   &vm.Com1Dev,
			"com2_dev":   &vm.Com2Dev,
			"com3_dev":   &vm.Com3Dev,
			"com4_dev":   &vm.Com4Dev,
		})
	if res.Error != nil {
		slog.Error("error saving VM stopped", "err", res.Error)

		return fmt.Errorf("error saving VM: %w", res.Error)
	}

	return nil
}

func (vm *VM) SetStopping() {
	db := GetVMDB()
	defer vm.mu.Unlock()
	vm.mu.Lock()
	vm.Status = STOPPING

	res := db.Select([]string{
		"status",
	}).Model(&vm).
		Updates(map[string]interface{}{
			"status": &vm.Status,
		})
	if res.Error != nil {
		slog.Error("error saving VM stopping", "err", res.Error)
	}
}

func (vm *VM) SetVNCPort(port int) {
	slog.Debug("SetVNCPort", "port", port)
	defer vm.mu.Unlock()
	vm.mu.Lock()
	vm.VNCPort = int32(port)
	_ = vm.Save()
}

func (vm *VM) SetDebugPort(port int) {
	slog.Debug("SetDebugPort", "port", port)
	defer vm.mu.Unlock()
	vm.mu.Lock()
	vm.DebugPort = int32(port)
	_ = vm.Save()
}

func (vm *VM) BeforeCreate(_ *gorm.DB) error {
	if vm.ID == "" {
		vm.ID = uuid.NewString()
	}

	return nil
}

func DBAutoMigrate() {
	vmdb := GetVMDB()

	err := vmdb.AutoMigrate(&VM{})
	if err != nil {
		panic("failed to auto-migrate VMs")
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
		panic(err)
	}

	for _, vmInst := range allVMs {
		initOneVM(vmInst)
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
		_ = rows.Close()
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

		return returnDisks, fmt.Errorf("error getting VM Diss: %w", rowErr)
	}

	defer func() {
		_ = rows.Close()
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

		return result, res.Error
	}

	// manually load VM Disks/ISOs because GORM can't do what is needed in terms of allowing duplicates or
	// preserving position
	for _, vmResult := range result {
		vmResult.ISOs, err = getIsosForVM(vmResult.ID, vmDB)
		if err != nil {
			slog.Error("failed getting isos for VM", "err", err)

			return result, err
		}

		vmResult.Disks, err = getDisksForVM(vmResult.ID, vmDB)
		if err != nil {
			slog.Error("failed getting disks for VM", "err", err)

			return result, err
		}
	}

	return result, nil
}
