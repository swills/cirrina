package vmnic

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
	vmNicDB *gorm.DB
}

var instance *singleton

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
		if instance != nil {
			return
		}
		instance = &singleton{}
		vmNicDB, err := gorm.Open(
			sqlite.Open(config.Config.DB.Path),
			&gorm.Config{
				Logger:      noColorLogger,
				PrepareStmt: true,
			},
		)
		if err != nil {
			panic("failed to connect database")
		}
		sqlDB, err := vmNicDB.DB()
		if err != nil {
			panic("failed to create sqlDB database")
		}
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)
		instance.vmNicDB = vmNicDB
	})

	return instance.vmNicDB
}

func (d *VMNic) BeforeCreate(_ *gorm.DB) error {
	d.ID = uuid.NewString()

	return nil
}

func DBAutoMigrate() {
	vmNicDB := GetVMNicDB()
	err := vmNicDB.AutoMigrate(&VMNic{})
	if err != nil {
		panic("failed to auto-migrate VmNics")
	}
}

func DBInitialized() bool {
	db := GetVMNicDB()

	return db.Migrator().HasColumn(VMNic{}, "id")
}
