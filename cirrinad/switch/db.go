package _switch

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
	switchDb *gorm.DB
}

var instance *singleton

var once sync.Once

func getSwitchDb() *gorm.DB {
	noColorLogger := logger.New(
		log.New(os.Stdout, "SwitchDb: ", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		},
	)

	once.Do(func() {
		instance = &singleton{}
		switchDb, err := gorm.Open(
			sqlite.Open(config.Config.DB.Path),
			&gorm.Config{
				Logger:      noColorLogger,
				PrepareStmt: true,
			},
		)
		if err != nil {
			panic("failed to connect database")
		}
		sqlDB, err := switchDb.DB()
		if err != nil {
			panic("failed to create sqlDB database")
		}
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)
		instance.switchDb = switchDb
	})

	return instance.switchDb
}

func (d *Switch) BeforeCreate(_ *gorm.DB) (err error) {
	d.ID = uuid.NewString()

	return nil
}

func DbAutoMigrate() {
	db := getSwitchDb()
	err := db.AutoMigrate(&Switch{})
	if err != nil {
		panic("failed to auto-migrate switches")
	}
}
