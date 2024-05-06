package disk

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

type singleton struct {
	diskDB *gorm.DB
}

var instance *singleton

var once sync.Once

func getDiskDB() *gorm.DB {
	noColorLogger := logger.New(
		log.New(os.Stdout, "DiskDb: ", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		},
	)

	once.Do(func() {
		// allow override for testing
		if instance != nil {
			return
		}

		instance = &singleton{}

		diskDB, err := gorm.Open(
			sqlite.Open(config.Config.DB.Path),
			&gorm.Config{
				Logger:      noColorLogger,
				PrepareStmt: true,
			},
		)
		if err != nil {
			panic("failed to connect database")
		}

		sqlDB, err := diskDB.DB()
		if err != nil {
			panic("failed to create sqlDB database")
		}

		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)

		instance.diskDB = diskDB
	})

	return instance.diskDB
}

func (d *Disk) BeforeCreate(_ *gorm.DB) error {
	d.ID = uuid.NewString()
	if d.Name == "" {
		return errDiskInvalidName
	}

	return nil
}

func DBAutoMigrate() {
	diskDB := getDiskDB()

	err := diskDB.AutoMigrate(&Disk{})
	if err != nil {
		panic("failed to auto-migrate disk")
	}

	err = diskDB.Migrator().DropColumn(&Disk{}, "path")
	if err != nil {
		slog.Error("DiskDb DBAutoMigrate failed to drop path column, continuing anyway")
	}

	for _, diskInst := range GetAllDB() {
		initOneDisk(diskInst)
	}
}
