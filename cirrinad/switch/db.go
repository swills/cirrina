package vmswitch

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
	switchDB *gorm.DB
}

var instance *singleton

var once sync.Once

func getSwitchDB() *gorm.DB {
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
		switchDB, err := gorm.Open(
			sqlite.Open(config.Config.DB.Path),
			&gorm.Config{
				Logger:      noColorLogger,
				PrepareStmt: true,
			},
		)
		if err != nil {
			panic("failed to connect database")
		}
		sqlDB, err := switchDB.DB()
		if err != nil {
			panic("failed to create sqlDB database")
		}
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)
		instance.switchDB = switchDB
	})

	return instance.switchDB
}

func (d *Switch) BeforeCreate(_ *gorm.DB) error {
	d.ID = uuid.NewString()

	return nil
}

func DBAutoMigrate() {
	db := getSwitchDB()
	err := db.AutoMigrate(&Switch{})
	if err != nil {
		panic("failed to auto-migrate switches")
	}
}
