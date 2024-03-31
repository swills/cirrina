package vm

import (
	"cirrina/cirrinad/config"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"log/slog"
	"os"
	"time"
)

type singleton struct {
	vmDb *gorm.DB
}

var instance *singleton

var dbInitialized bool

func DbReconfig() {
	dbInitialized = false
}

func GetVmDb() *gorm.DB {

	noColorLogger := logger.New(
		log.New(os.Stdout, "VmDb: ", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		},
	)

	if !dbInitialized {
		instance = &singleton{}
		vmDb, err := gorm.Open(
			sqlite.Open(config.Config.DB.Path),
			&gorm.Config{
				Logger:      noColorLogger,
				PrepareStmt: true,
			},
		)
		vmDb.Preload("Config")
		if err != nil {
			panic("failed to connect database")
		}
		sqlDB, err := vmDb.DB()
		if err != nil {
			panic("failed to create sqlDB database")
		}
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)
		instance.vmDb = vmDb
		dbInitialized = true
	}
	return instance.vmDb
}

func (vm *VM) SetRunning(pid int) {
	db := GetVmDb()
	defer vm.mu.Unlock()
	vm.mu.Lock()
	vm.Status = RUNNING
	vm.BhyvePid = uint32(pid)
	res := db.Select([]string{
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
	db := GetVmDb()
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
func (vm *VM) SetStopped() {
	db := GetVmDb()
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
	res := db.Select([]string{
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
	}
}

func (vm *VM) SetStopping() {
	db := GetVmDb()
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

func (vm *VM) BeforeCreate(_ *gorm.DB) (err error) {
	vm.ID = uuid.NewString()
	return nil
}

func DbAutoMigrate() {
	db := GetVmDb()
	err := db.AutoMigrate(&VM{})
	if err != nil {
		panic("failed to auto-migrate VMs")
	}
	err = db.AutoMigrate(&Config{})
	if err != nil {
		slog.Error("failed db migration", "err", err)
		panic("failed to auto-migrate Configs")
	}

	defer List.Mu.Unlock()
	List.Mu.Lock()
	for _, vmInst := range GetAllDb() {
		InitOneVm(vmInst)
	}
}

func GetAllDb() []*VM {
	var result []*VM

	db := GetVmDb()
	db.Preload("Config").Find(&result)

	return result
}

func DbInitialized() bool {
	db := GetVmDb()
	return db.Migrator().HasColumn(VM{}, "id")
}
