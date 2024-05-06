package db

import (
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/vm"
	"cirrina/cirrinad/vmnic"
)

type meta struct {
	ID            uint   `gorm:"primarykey"`
	SchemaVersion uint32 `gorm:"not null"`
}

type singleton struct {
	metaDB *gorm.DB
}

var instance *singleton

var once sync.Once

func getMetaDB() *gorm.DB {
	noColorLogger := logger.New(
		log.New(os.Stdout, "MetaDb: ", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		},
	)

	once.Do(func() {
		instance = &singleton{}
		metaDB, err := gorm.Open(
			sqlite.Open(config.Config.DB.Path),
			&gorm.Config{
				Logger:      noColorLogger,
				PrepareStmt: true,
			},
		)
		metaDB.Preload("Config")

		if err != nil {
			panic("failed to connect database")
		}

		sqlDB, err := metaDB.DB()
		if err != nil {
			panic("failed to create sqlDB database")
		}

		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)

		instance.metaDB = metaDB
	})

	return instance.metaDB
}

func AutoMigrate() {
	db := getMetaDB()

	err := db.AutoMigrate(&meta{})
	if err != nil {
		panic("failed to auto-migrate meta")
	}
}

func getSchemaVersion() uint32 {
	metaDB := getMetaDB()

	var m meta

	metaDB.Find(&m)

	return m.SchemaVersion
}

func setSchemaVersion(schemaVersion uint32) {
	metaDB := getMetaDB()

	var metaData meta
	metaData.ID = 1 // always!

	var res *gorm.DB

	res = metaDB.Delete(&metaData)
	if res.Error != nil {
		slog.Error("error saving schema_version", "err", res.Error)
		panic(res.Error)
	}

	metaData.SchemaVersion = schemaVersion

	res = metaDB.Create(&metaData)
	if res.Error != nil {
		slog.Error("error saving schema_version", "err", res.Error)
		panic(res.Error)
	}
}

func CustomMigrate() {
	slog.Debug("starting custom migration")

	vmNicDB := vmnic.GetVMNicDB()
	vmDB := vm.GetVMDB()
	reqDB := requests.GetReqDB()

	schemaVersion := getSchemaVersion()
	// 2024022401 - copy nics from config.nics to vm_nics.config_id
	migration2024022401(schemaVersion, vmNicDB, vmDB)

	// 2024022402 - drop config.nics
	migration2024022402(schemaVersion, vmDB)

	// 2024022403 - remove vm_id from requests
	migration2024022403(schemaVersion, reqDB)

	// 2024022403

	slog.Debug("finished custom migration")
}

func migration2024022401(schemaVersion uint32, vmNicDB *gorm.DB, vmDB *gorm.DB) {
	var err error

	if schemaVersion < 2024022401 {
		if vmnic.DBInitialized() {
			if !vmNicDB.Migrator().HasColumn(vmnic.VMNic{}, "config_id") {
				slog.Debug("migrating config.nics to vm_nics.config_id")

				err = vmNicDB.Migrator().AddColumn(vmnic.VMNic{}, "config_id")
				if err != nil {
					slog.Debug("error adding config_id column", "err", err)
					panic(err)
				}

				allVMs := vm.GetAllDB()
				for _, vmInst := range allVMs {
					type Result struct {
						Nics string
					}

					var result Result

					vmDB.Raw("SELECT nics FROM configs WHERE id = ?", vmInst.Config.ID).Scan(&result)

					var thisVmsNics []vmnic.VMNic

					for _, configValue := range strings.Split(result.Nics, ",") {
						if configValue == "" {
							continue
						}

						var aNic *vmnic.VMNic

						aNic, err = vmnic.GetByID(configValue)
						if err == nil {
							thisVmsNics = append(thisVmsNics, *aNic)
						} else {
							slog.Error("bad nic", "nic", configValue, "vm", vmInst.ID)
						}
					}

					if err != nil {
						slog.Debug("error looking up VMs Nics", "err", err)
						panic(err)
					}

					for _, vmNic := range thisVmsNics {
						slog.Debug("migrating vm nic", "nicId", vmNic.ID)
						vmNic.ConfigID = vmInst.Config.ID
						err = vmNic.Save()

						if err != nil {
							slog.Error("failure saving nic", "nicId", vmNic.ID, "err", err)
							panic(err)
						}
					}
				}

				slog.Debug("migration complete", "id", "2024022401", "message", "vm_nics.config_id populated")
				vm.DBReconfig()
			}
		}

		setSchemaVersion(2024022401)
	}
}

func migration2024022402(schemaVersion uint32, vmDB *gorm.DB) {
	if schemaVersion < 2024022402 {
		if vm.DBInitialized() {
			if vmDB.Migrator().HasColumn(&vm.Config{}, "nics") {
				slog.Debug("removing config.nics")

				err := vmDB.Migrator().DropColumn(&vm.Config{}, "nics")
				if err != nil {
					slog.Error("failure removing nics column", "err", err)
					panic(err)
				}

				slog.Debug("migration complete", "id", "2024022402", "message", "config.nics dropped")
				vm.DBReconfig()
			}
		}

		setSchemaVersion(2024022402)
	}
}

func migration2024022403(schemaVersion uint32, reqDB *gorm.DB) {
	if schemaVersion < 2024022403 {
		if requests.DBInitialized() {
			// sqlite doesn't let you remove a column, so just nuke it, the requests table isn't critical
			if reqDB.Migrator().HasColumn(&requests.Request{}, "vm_id") {
				slog.Debug("dropping requests table")

				err := reqDB.Migrator().DropTable("requests")
				if err != nil {
					slog.Error("failure dropping requests table", "err", err)
					panic(err)
				}
			}

			requests.DBReconfig()
		}

		setSchemaVersion(2024022403)
	}
}
