package requests

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"cirrina/cirrinad/config"
)

type Singleton struct {
	ReqDB *gorm.DB
}

var Instance *Singleton

var dbInitialized bool

func DBReconfig() {
	dbInitialized = false
}

func GetReqDB() *gorm.DB {
	noColorLogger := logger.New(
		log.New(os.Stdout, "ReqDb: ", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		},
	)

	// allow override for testing
	if Instance != nil {
		return Instance.ReqDB
	}

	if !dbInitialized {
		Instance = &Singleton{}

		reqDB, err := gorm.Open(
			sqlite.Open(config.Config.DB.Path),
			&gorm.Config{
				Logger:      noColorLogger,
				PrepareStmt: true,
			},
		)
		if err != nil {
			panic("failed to connect database")
		}

		sqlDB, err := reqDB.DB()
		if err != nil {
			panic("failed to create sqlDB database")
		}

		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)

		Instance.ReqDB = reqDB
		dbInitialized = true
	}

	return Instance.ReqDB
}

func DBAutoMigrate() {
	db := GetReqDB()

	err := db.AutoMigrate(&Request{})
	if err != nil {
		panic("failed to auto-migrate Requests")
	}
}

func (req *Request) BeforeCreate(_ *gorm.DB) error {
	if req.ID == "" {
		newUUID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}

		req.ID = newUUID.String()
	}

	return nil
}
