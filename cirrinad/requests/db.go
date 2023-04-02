package requests

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"sync"
)

type singleton struct {
	reqDb *gorm.DB
}

var instance *singleton

var once sync.Once

func getReqDb() *gorm.DB {
	once.Do(func() {
		instance = &singleton{}
		reqDb, err := gorm.Open(sqlite.Open("db/cirrina.sqlite"), &gorm.Config{})
		if err != nil {
			panic("failed to connect database")
		}
		sqlDB, err := reqDb.DB()
		if err != nil {
			panic("failed to create sqlDB database")
		}
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)
		instance.reqDb = reqDb
	})
	return instance.reqDb
}
