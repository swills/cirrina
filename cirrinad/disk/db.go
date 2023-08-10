package disk

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
	diskDb *gorm.DB
}

var instance *singleton

var once sync.Once

func getDiskDb() *gorm.DB {

	noColorLogger := logger.New(
		log.New(os.Stdout, "", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		},
	)

	once.Do(func() {
		instance = &singleton{}
		diskDb, err := gorm.Open(sqlite.Open(config.Config.DB.Path), &gorm.Config{Logger: noColorLogger})
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
