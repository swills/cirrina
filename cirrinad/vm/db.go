package vm

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
)

func getVmDb() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("cirrina.sqlite"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	return db
}

func setRunning(id string, pid int) {
	log.Printf("VM %v started, pid: %v", id, pid)
	vm := VM{ID: id}
	db := getVmDb()
	vm.Status = RUNNING
	vm.BhyvePid = uint32(pid)
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		panic("Error saving VM start")
	}
}

func setStarting(id string) {
	vm := VM{ID: id}
	db := getVmDb()
	vm.Status = STARTING
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		log.Printf("Error saving VM stop")
	}
}

// this can in some cases get called on already stopped/deleted VMs and that's OK
func setStopped(id string) {
	vm := VM{ID: id}
	db := getVmDb()
	vm.Status = STOPPED
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		log.Printf("Error saving VM stop")
	}
}

func setStopping(id string) {
	vm := VM{ID: id}
	db := getVmDb()
	vm.Status = STOPPING
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		log.Printf("Error saving VM stop")
	}
}

func (vm *VM) setVNCPort(port int) {
	vm.VNCPort = int32(port)
	_ = vm.Save()
}
