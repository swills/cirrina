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
	isoDB *gorm.DB
}

var instance *singleton

var once sync.Once

func getIsoDB() *gorm.DB {
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
		isoDB, err := gorm.Open(
			sqlite.Open(config.Config.DB.Path),
			&gorm.Config{
				Logger:      noColorLogger,
				PrepareStmt: true,
			},
		)
		if err != nil {
			panic("failed to connect database")
		}
		sqlDB, err := isoDB.DB()
		if err != nil {
			panic("failed to create sqlDB database")
		}
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)
		instance.isoDB = isoDB
	})

	return instance.isoDB
}

func (iso *ISO) BeforeCreate(_ *gorm.DB) (err error) {
	iso.ID = uuid.NewString()

	return nil
}

func DBAutoMigrate() {
	db := getIsoDB()
	err := db.AutoMigrate(&ISO{})
	if err != nil {
		panic("failed to auto-migrate ISO")
	}
}
