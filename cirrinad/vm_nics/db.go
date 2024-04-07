package vm_nics

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
	vmNicDb *gorm.DB
}

var instance *singleton

var once sync.Once

func GetVmNicDb() *gorm.DB {
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
		instance = &singleton{}
		vmNicDb, err := gorm.Open(
			sqlite.Open(config.Config.DB.Path),
			&gorm.Config{
				Logger:      noColorLogger,
				PrepareStmt: true,
			},
		)
		if err != nil {
			panic("failed to connect database")
		}
		sqlDB, err := vmNicDb.DB()
		if err != nil {
			panic("failed to create sqlDB database")
		}
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)
		instance.vmNicDb = vmNicDb
	})

	return instance.vmNicDb
}

func (d *VmNic) BeforeCreate(_ *gorm.DB) (err error) {
	d.ID = uuid.NewString()

	return nil
}

func DbAutoMigrate() {
	vmNicDb := GetVmNicDb()
	err := vmNicDb.AutoMigrate(&VmNic{})
	if err != nil {
		panic("failed to auto-migrate VmNics")
	}
}

func DbInitialized() bool {
	db := GetVmNicDb()

	return db.Migrator().HasColumn(VmNic{}, "id")
}
