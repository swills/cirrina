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

type Singleton struct {
	SwitchDB *gorm.DB
}

var Instance *Singleton

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
		// allow override for testing
		if Instance != nil {
			return
		}

		Instance = &Singleton{}

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

		Instance.SwitchDB = switchDB
	})

	return Instance.SwitchDB
}

func (s *Switch) BeforeCreate(_ *gorm.DB) error {
	if s == nil || s.Name == "" {
		return ErrSwitchInvalidName
	}

	err := uuid.Validate(s.ID)
	if err != nil || len(s.ID) != 36 {
		s.ID = uuid.NewString()
	}

	return nil
}

func (s *Switch) AfterCreate(_ *gorm.DB) error {
	return s.bringUpNewSwitch()
}

func DBAutoMigrate() {
	db := getSwitchDB()

	err := db.AutoMigrate(&Switch{})
	if err != nil {
		panic("failed to auto-migrate switches")
	}
}
