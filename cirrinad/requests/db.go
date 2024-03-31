package requests

import (
	"cirrina/cirrinad/config"
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type singleton struct {
	reqDb *gorm.DB
}

var instance *singleton

var dbInitialized bool

func DbReconfig() {
	dbInitialized = false
}

func GetReqDb() *gorm.DB {

	noColorLogger := logger.New(
		log.New(os.Stdout, "ReqDb: ", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		},
	)

	if !dbInitialized {
		instance = &singleton{}
		reqDb, err := gorm.Open(
			sqlite.Open(config.Config.DB.Path),
			&gorm.Config{
				Logger:      noColorLogger,
				PrepareStmt: true,
			},
		)
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
		dbInitialized = true
	}
	return instance.reqDb
}

func DbAutoMigrate() {
	db := GetReqDb()
	err := db.AutoMigrate(&Request{})
	if err != nil {
		panic("failed to auto-migrate Requests")
	}
}
