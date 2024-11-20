package iso

import (
	"log"
	"log/slog"
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
	ISODB *gorm.DB
}

var Instance *Singleton

var once sync.Once

func GetIsoDB() *gorm.DB {
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
		// allow override for testing
		if Instance != nil {
			return
		}

		Instance = &Singleton{}

		isoDB, err := gorm.Open(
			sqlite.Open(config.Config.DB.Path),
			&gorm.Config{
				Logger:      noColorLogger,
				PrepareStmt: true,
			},
		)
		if err != nil {
			slog.Error("failed to connect to database", "err", err)
			panic("failed to connect database, err: " + err.Error())
		}

		sqlDB, err := isoDB.DB()
		if err != nil {
			slog.Error("failed to create sqlDB database", "err", err)
			panic("failed to create sqlDB database, err: " + err.Error())
		}

		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)

		Instance.ISODB = isoDB
	})

	return Instance.ISODB
}

func (i *ISO) BeforeCreate(_ *gorm.DB) error {
	if i == nil || i.Name == "" {
		return ErrIsoInvalidName
	}

	err := uuid.Validate(i.ID)
	if err != nil || len(i.ID) != 36 {
		i.ID = uuid.NewString()
	}

	return nil
}

func DBAutoMigrate() {
	db := GetIsoDB()

	err := db.AutoMigrate(&ISO{})
	if err != nil {
		slog.Error("failed to auto-migrate ISOs", "err", err)
		panic("failed to auto-migrate ISOs, err: " + err.Error())
	}
}
