package vm

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
)

func setRunning(id string, pid int) {
	log.Printf("VM %v started, pid: %v", id, pid)
	vm := VM{ID: id}
	db := GetVMDB()
	vm.Status = RUNNING
	vm.BhyvePid = uint32(pid)
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		panic("Error saving VM start")
	}
}

func DbSetVMStopped(id string) {
	vm := VM{ID: id}
	db := GetVMDB()
	vm.Status = STOPPED
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		log.Printf("Error saving VM stop")
	}
}

func DbSetVMStopping(id string) {
	vm := VM{ID: id}
	db := GetVMDB()
	vm.Status = STOPPING
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		log.Printf("Error saving VM stop")
	}
}

func DbSetVMStarting(id string) {
	vm := VM{ID: id}
	db := GetVMDB()
	vm.Status = STARTING
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		log.Printf("Error saving VM stop")
	}
}

func DbVMExists(name string) bool {
	db := GetVMDB()
	var evm VM
	db.Limit(1).Find(&evm, &VM{Name: name})
	if evm.ID != "" {
		return true
	}
	return false
}

func DbCreateVM(vm *VM) error {
	db := GetVMDB()
	res := db.Create(&vm)
	return res.Error
}

func GetVMDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("cirrina.sqlite"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	err = db.AutoMigrate(&VM{})
	if err != nil {
		panic("failed to auto-migrate VMs")
	}
	err = db.AutoMigrate(&Config{})
	if err != nil {
		panic("failed to auto-migrate Configs")
	}
	return db
}
