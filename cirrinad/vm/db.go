package vm

import (
	"cirrina/cirrinad/config"
	"golang.org/x/exp/slog"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"sync"
)

type singleton struct {
	vmDb *gorm.DB
}

var instance *singleton

var once sync.Once

func getVmDb() *gorm.DB {
	once.Do(func() {
		instance = &singleton{}
		vmDb, err := gorm.Open(sqlite.Open(config.Config.DB.Path), &gorm.Config{})
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

func setRunning(id string, pid int) {
	vm := VM{ID: id}
	db := getVmDb()
	vm.Status = RUNNING
	vm.BhyvePid = uint32(pid)
	res := db.Select([]string{
		"status",
		"bhyve_pid",
	}).Model(&vm).
		Updates(map[string]interface{}{
			"status":    &vm.Status,
			"bhyve_pid": &vm.BhyvePid,
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
	vm.BhyvePid = 0
	vm.ComDevs = ""
	res := db.Select([]string{
		"status",
		"net_dev",
		"vnc_port",
		"bhyve_pid",
		"com_devs",
	}).Model(&vm).
		Updates(map[string]interface{}{
			"status":    &vm.Status,
			"vnc_port":  &vm.VNCPort,
			"bhyve_pid": &vm.BhyvePid,
			"com_devs":  &vm.ComDevs,
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
	vm.VNCPort = int32(port)
	_ = vm.Save()
}

func (vm *VM) setComPorts(ports string) {
	vm.ComDevs = ports
	_ = vm.Save()
}
