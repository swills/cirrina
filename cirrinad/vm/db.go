package vm

import (
	"cirrina/cirrinad/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
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
		log.Printf("Error saving VM running")
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
		log.Printf("Error saving VM start")
	}
}

// this can in some cases get called on already stopped/deleted VMs and that's OK
func setStopped(id string) {
	vm := VM{ID: id}
	db := getVmDb()
	vm.Status = STOPPED
	vm.NetDev = ""
	vm.VNCPort = 0
	vm.BhyvePid = 0
	res := db.Select([]string{
		"status",
		"net_dev",
		"vnc_port",
		"bhyve_pid",
	}).Model(&vm).
		Updates(map[string]interface{}{
			"status":    &vm.Status,
			"net_dev":   &vm.NetDev,
			"vnc_port":  &vm.VNCPort,
			"bhyve_pid": &vm.BhyvePid,
		})
	if res.Error != nil {
		log.Printf("Error saving VM stop")
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
		log.Printf("Error saving VM stop")
	}
}

func (vm *VM) setVNCPort(port int) {
	vm.VNCPort = int32(port)
	_ = vm.Save()
}
