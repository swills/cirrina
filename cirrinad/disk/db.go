package disk

import (
	"cirrina/cirrinad/config"
	"errors"
	"log"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type singleton struct {
	diskDb *gorm.DB
}

var instance *singleton

var once sync.Once

func getDiskDb() *gorm.DB {

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
		instance = &singleton{}
		diskDb, err := gorm.Open(
			sqlite.Open(config.Config.DB.Path),
			&gorm.Config{
				Logger:      noColorLogger,
				PrepareStmt: true,
			},
		)
		if err != nil {
			panic("failed to connect database")
		}
		sqlDB, err := diskDb.DB()
		if err != nil {
			panic("failed to create sqlDB database")
		}
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)
		instance.diskDb = diskDb
	})
	return instance.diskDb
}

func (d *Disk) BeforeCreate(_ *gorm.DB) (err error) {
	d.ID = uuid.NewString()
	if d.Name == "" {
		return errors.New("invalid disk name")
	}
	return nil
}

func DbAutoMigrate() {
	db := getDiskDb()
	err := db.AutoMigrate(&Disk{})
	if err != nil {
		panic("failed to auto-migrate disk")
	}
	err = db.Migrator().DropColumn(&Disk{}, "path")
	if err != nil {
		slog.Error("DiskDb DbAutoMigrate failed to drop path column, continuing anyway")
	}
	for _, diskInst := range GetAllDb() {
		InitOneDisk(diskInst)
	}
}
