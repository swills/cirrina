package disk

import (
	"cirrina/cirrinad/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"sync"
)

type singleton struct {
	diskDb *gorm.DB
}

var instance *singleton

var once sync.Once

func getDiskDb() *gorm.DB {
	once.Do(func() {
		instance = &singleton{}
		diskDb, err := gorm.Open(sqlite.Open(config.Config.DB.Path), &gorm.Config{})
		if err != nil {
			panic("failed to connect database")
		}
		sqlDB, err := diskDb.DB()
		if err != nil {
			panic("failed to create sqlDB database")
		}
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)
		instance.diskDb = diskDb
	})
	return instance.diskDb
}
