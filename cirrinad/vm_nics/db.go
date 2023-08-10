package vm_nics

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
	vmNicDb *gorm.DB
}

var instance *singleton

var once sync.Once

func getVmNicDb() *gorm.DB {

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
		vmNicDb, err := gorm.Open(sqlite.Open(config.Config.DB.Path), &gorm.Config{Logger: noColorLogger})
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
