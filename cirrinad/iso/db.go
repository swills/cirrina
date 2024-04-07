package iso

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"cirrina/cirrinad/config"
)

type singleton struct {
	isoDb *gorm.DB
}

var instance *singleton

var once sync.Once

func getIsoDb() *gorm.DB {
	noColorLogger := logger.New(
		log.New(os.Stdout, "IsoDb: ", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		},
	)

	once.Do(func() {
		instance = &singleton{}
		isoDb, err := gorm.Open(
			sqlite.Open(config.Config.DB.Path),
			&gorm.Config{
				Logger:      noColorLogger,
				PrepareStmt: true,
			},
		)
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

func (iso *ISO) BeforeCreate(_ *gorm.DB) (err error) {
	iso.ID = uuid.NewString()
	return nil
}

func DbAutoMigrate() {
	db := getIsoDb()
	err := db.AutoMigrate(&ISO{})
	if err != nil {
		panic("failed to auto-migrate ISO")
	}
}
