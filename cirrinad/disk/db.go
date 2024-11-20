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

type Singleton struct {
	DiskDB *gorm.DB
}

var Instance *Singleton

var once sync.Once

func GetDiskDB() *gorm.DB {
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
		if Instance != nil {
			return
		}

		Instance = &Singleton{}

		diskDB, err := gorm.Open(
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

		sqlDB, err := diskDB.DB()
		if err != nil {
			slog.Error("failed to create sqlDB database", "err", err)
			panic("failed to create sqlDB database, err: " + err.Error())
		}

		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)

		Instance.DiskDB = diskDB
	})

	return Instance.DiskDB
}

func (d *Disk) BeforeCreate(_ *gorm.DB) error {
	if d == nil || d.Name == "" {
		return ErrDiskInvalidName
	}

	err := uuid.Validate(d.ID)
	if err != nil || len(d.ID) != 36 {
		d.ID = uuid.NewString()
	}

	return nil
}

func DBAutoMigrate() {
	diskDB := GetDiskDB()

	err := diskDB.AutoMigrate(&Disk{})
	if err != nil {
		slog.Error("failed to auto-migrate disk table", "err", err)
		panic("failed to auto-migrate disk, err: " + err.Error())
	}

	err = diskDB.Migrator().DropColumn(&Disk{}, "path")
	if err != nil {
		slog.Error("DiskDb DBAutoMigrate failed to drop path column, continuing anyway")
	}
}

func CacheInit() {
	defer List.Mu.Unlock()
	List.Mu.Lock()
	for _, diskInst := range GetAllDB() {
		diskInst.initOneDisk()
	}
}
