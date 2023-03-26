package main

import (
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"time"
)

func (vm *VM) BeforeCreate(_ *gorm.DB) (err error) {
	vm.ID = uuid.NewString()
	return nil
}

func (req *Request) BeforeCreate(_ *gorm.DB) (err error) {
	req.ID = uuid.NewString()
	return nil
}

func getVMDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("cirrina.sqlite"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	err = db.AutoMigrate(&VM{})
	if err != nil {
		panic("failed to auto-migrate VMs")
	}
	err = db.AutoMigrate(&VMConfig{})
	if err != nil {
		panic("failed to auto-migrate Configs")
	}
	err = db.AutoMigrate(&Request{})
	if err != nil {
		panic("failed to auto-migrate Requests")
	}
	return db
}

func dbSetVMStopped(id string) {
	vm := VM{ID: id}
	db := getVMDB()
	vm.Status = STOPPED
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		log.Printf("Error saving VM stop")
	}
}

func dbSetVMRunning(id string, pid int) {
	log.Printf("VM %v started, pid: %v", id, pid)
	vm := VM{ID: id}
	db := getVMDB()
	vm.Status = RUNNING
	vm.BhyvePid = uint32(pid)
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		panic("Error saving VM start")
	}
}

func dbSetVMStopping(id string) {
	vm := VM{ID: id}
	db := getVMDB()
	vm.Status = STOPPING
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		log.Printf("Error saving VM stop")
	}
}

func dbSetVMStarting(id string) {
	vm := VM{ID: id}
	db := getVMDB()
	vm.Status = STARTING
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		log.Printf("Error saving VM stop")
	}
}

func dbVMExists(name string) bool {
	db := getVMDB()
	var evm VM
	db.Limit(1).Find(&evm, &VM{Name: name})
	if evm.ID != "" {
		return true
	}
	return false
}

func dbCreateVM(vm VM) error {
	db := getVMDB()
	res := db.Create(&vm)
	return res.Error
}

func getUnStartedReq() Request {
	db := getVMDB()
	rs := Request{}
	db.Limit(1).Where("started_at IS NULL").Find(&rs)
	return rs
}

func startReq(rs Request) {
	db := getVMDB()
	rs.StartedAt.Time = time.Now()
	rs.StartedAt.Valid = true
	db.Model(&rs).Limit(1).Updates(rs)
}

func MarkReqSuccessful(rs *Request) *gorm.DB {
	db := getVMDB()
	return db.Model(&rs).Limit(1).Updates(
		Request{
			Successful: true,
			Complete:   true,
		},
	)
}

func MarkReqFailed(rs *Request) *gorm.DB {
	db := getVMDB()
	return db.Model(&rs).Limit(1).Updates(
		Request{
			Successful: false,
			Complete:   true,
		},
	)
}
