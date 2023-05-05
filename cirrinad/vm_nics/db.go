package vm_nics

import (
	"cirrina/cirrinad/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"sync"
)

type singleton struct {
	vmNicDb *gorm.DB
}

var instance *singleton

var once sync.Once

func getSwitchDb() *gorm.DB {
	once.Do(func() {
		instance = &singleton{}
		vmNicDb, err := gorm.Open(sqlite.Open(config.Config.DB.Path), &gorm.Config{})
		if err != nil {
			panic("failed to connect database")
		}
		sqlDB, err := vmNicDb.DB()
		if err != nil {
			panic("failed to create sqlDB database")
		}
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)
		instance.vmNicDb = vmNicDb
	})
	return instance.vmNicDb
}
