package vm

import (
	"cirrina/cirrinad/config"
	"golang.org/x/exp/slog"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"sync"
	"time"
)

type singleton struct {
	vmDb *gorm.DB
}

var instance *singleton

var once sync.Once

func getVmDb() *gorm.DB {

	noColorLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		},
	)

	once.Do(func() {
		instance = &singleton{}
		vmDb, err := gorm.Open(sqlite.Open(config.Config.DB.Path), &gorm.Config{Logger: noColorLogger})
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
	})
	return instance.vmDb
}

func (vm *VM) setRunning(pid int) {
	db := getVmDb()
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

func (vm *VM) setStarting() {
	db := getVmDb()
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

// this can in some cases get called on already stopped/deleted VMs and that's OK
func setStopped(id string) {
	vm := VM{ID: id}
	db := getVmDb()
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

func setStopping(id string) {
	vm := VM{ID: id}
	db := getVmDb()
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

func (vm *VM) setVNCPort(port int) {
	slog.Debug("setVNCPort", "port", port)
	vm.VNCPort = int32(port)
	_ = vm.Save()
}

func (vm *VM) setDebugPort(port int) {
	slog.Debug("setDebugPort", "port", port)
	vm.DebugPort = int32(port)
	_ = vm.Save()
}
