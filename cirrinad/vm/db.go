package vm

import (
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
		vmDb, err := gorm.Open(sqlite.Open("cirrina.sqlite"), &gorm.Config{})
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
	log.Printf("VM %v started, pid: %v", id, pid)
	vm := VM{ID: id}
	db := getVmDb()
	vm.Status = RUNNING
	vm.BhyvePid = uint32(pid)
	res := db.Updates(&vm)
	if res.Error != nil {
		panic("Error saving VM start")
	}
}

func setStarting(id string) {
	vm := VM{ID: id}
	db := getVmDb()
	vm.Status = STARTING
	res := db.Updates(&vm)
	if res.Error != nil {
		log.Printf("Error saving VM stop")
	}
}

// this can in some cases get called on already stopped/deleted VMs and that's OK
func setStopped(id string) {
	vm := VM{ID: id}
	db := getVmDb()
	vm.Status = STOPPED
	res := db.Updates(&vm)
	if res.Error != nil {
		log.Printf("Error saving VM stop")
	}
}

func setStopping(id string) {
	vm := VM{ID: id}
	db := getVmDb()
	vm.Status = STOPPING
	res := db.Updates(&vm)
	if res.Error != nil {
		log.Printf("Error saving VM stop")
	}
}

func (vm *VM) setVNCPort(port int) {
	vm.VNCPort = int32(port)
	_ = vm.Save()
}
