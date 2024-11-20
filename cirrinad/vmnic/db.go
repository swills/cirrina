package vmnic

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
	VMNicDB *gorm.DB
}

var Instance *Singleton

var once sync.Once

func GetVMNicDB() *gorm.DB {
	noColorLogger := logger.New(
		log.New(os.Stdout, "VmNicDb: ", log.LstdFlags),
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

		vmNicDB, err := gorm.Open(
			sqlite.Open(config.Config.DB.Path),
			&gorm.Config{
				Logger:      noColorLogger,
				PrepareStmt: true,
			},
		)
		if err != nil {
			slog.Error("failed to connect database", "err", err)
			panic("failed to connect database, err: " + err.Error())
		}

		sqlDB, err := vmNicDB.DB()
		if err != nil {
			slog.Error("failed to create sqlDB database", "err", err)
			panic("failed to create sqlDB database, err: " + err.Error())
		}

		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)

		Instance.VMNicDB = vmNicDB
	})

	return Instance.VMNicDB
}

func (vmNic *VMNic) BeforeCreate(_ *gorm.DB) error {
	if vmNic == nil || vmNic.Name == "" {
		return ErrInvalidNicName
	}

	err := uuid.Validate(vmNic.ID)
	if err != nil || len(vmNic.ID) != 36 {
		vmNic.ID = uuid.NewString()
	}

	return nil
}

func DBAutoMigrate() {
	vmNicDB := GetVMNicDB()

	err := vmNicDB.AutoMigrate(&VMNic{})
	if err != nil {
		slog.Error("failed to auto-migrate VmNics", "err", err)
		panic("failed to auto-migrate VmNics, err:" + err.Error())
	}
}

func DBInitialized() bool {
	db := GetVMNicDB()

	return db.Migrator().HasColumn(VMNic{}, "id")
}
