package iso

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"sync"
)

type singleton struct {
	isoDb *gorm.DB
}

var instance *singleton

var once sync.Once

func getIsoDb() *gorm.DB {
	once.Do(func() {
		instance = &singleton{}
		isoDb, err := gorm.Open(sqlite.Open("db/cirrina.sqlite"), &gorm.Config{})
		if err != nil {
			panic("failed to connect database")
		}
		sqlDB, err := isoDb.DB()
		if err != nil {
			panic("failed to create sqlDB database")
		}
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)
		instance.isoDb = isoDb
	})
	return instance.isoDb
}
