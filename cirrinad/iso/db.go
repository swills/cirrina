package iso

import (
	"cirrina/cirrinad/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"sync"
	"time"
)

type singleton struct {
	isoDb *gorm.DB
}

var instance *singleton

var once sync.Once

func getIsoDb() *gorm.DB {

	noColorLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		},
	)

	once.Do(func() {
		instance = &singleton{}
		isoDb, err := gorm.Open(sqlite.Open(config.Config.DB.Path), &gorm.Config{Logger: noColorLogger})
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
