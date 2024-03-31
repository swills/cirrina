package db

import (
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/vm"
	"cirrina/cirrinad/vm_nics"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type meta struct {
	ID            uint   `gorm:"primarykey"`
	SchemaVersion uint32 `gorm:"not null"`
}

type singleton struct {
	metaDb *gorm.DB
}

var instance *singleton

var once sync.Once

func getMetaDb() *gorm.DB {
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
		metaDb, err := gorm.Open(
			sqlite.Open(config.Config.DB.Path),
			&gorm.Config{
				Logger:      noColorLogger,
				PrepareStmt: true,
			},
		)
		metaDb.Preload("Config")
		if err != nil {
			panic("failed to connect database")
		}
		sqlDB, err := metaDb.DB()
		if err != nil {
			panic("failed to create sqlDB database")
		}
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)
		instance.metaDb = metaDb
	})
	return instance.metaDb
}

func AutoMigrate() {
	db := getMetaDb()
	err := db.AutoMigrate(&meta{})
	if err != nil {
		panic("failed to auto-migrate meta")
	}
}

func getSchemaVersion() (schemaVersion uint32) {
	metaDb := getMetaDb()
	var m meta
	metaDb.Find(&m)
	return m.SchemaVersion
}

func setSchemaVersion(schemaVersion uint32) {
	metaDb := getMetaDb()
	var metaData meta
	metaData.ID = 1 // always!

	var res *gorm.DB
	res = metaDb.Delete(&metaData)
	if res.Error != nil {
		slog.Error("error saving schema_version", "err", res.Error)
		panic(res.Error)
	}
	metaData.SchemaVersion = schemaVersion
	res = metaDb.Create(&metaData)
	if res.Error != nil {
		slog.Error("error saving schema_version", "err", res.Error)
		panic(res.Error)
	}
}

func CustomMigrate() {
	slog.Debug("starting custom migration")
	vmNicDb := vm_nics.GetVmNicDb()
	vmDb := vm.GetVmDb()
	reqDb := requests.GetReqDb()

	schemaVersion := getSchemaVersion()
	// 2024022401 - copy nics from config.nics to vm_nics.config_id
	if schemaVersion < 2024022401 {
		if vm_nics.DbInitialized() {
			if !vmNicDb.Migrator().HasColumn(vm_nics.VmNic{}, "config_id") {
				slog.Debug("migrating config.nics to vm_nics.config_id")
				err := vmNicDb.Migrator().AddColumn(vm_nics.VmNic{}, "config_id")
				if err != nil {
					slog.Debug("error adding config_id column", "err", err)
					panic(err)
				}
				allVMs := vm.GetAllDb()
				for _, vmInst := range allVMs {

					type Result struct {
						Nics string
					}

					var result Result

					vmDb.Raw("SELECT nics FROM configs WHERE id = ?", vmInst.Config.ID).Scan(&result)

					var thisVmsNics []vm_nics.VmNic
					for _, cv := range strings.Split(result.Nics, ",") {
						if cv == "" {
							continue
						}
						aNic, err := vm_nics.GetById(cv)
						if err == nil {
							thisVmsNics = append(thisVmsNics, *aNic)
						} else {
							slog.Error("bad nic", "nic", cv, "vm", vmInst.ID)
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
				vm.DbReconfig()
			}
		}
		setSchemaVersion(2024022401)
	}

	// 2024022402 - drop config.nics
	if schemaVersion < 2024022402 {
		if vm.DbInitialized() {
			if vmDb.Migrator().HasColumn(&vm.Config{}, "nics") {

				slog.Debug("removing config.nics")
				err := vmDb.Migrator().DropColumn(&vm.Config{}, "nics")
				if err != nil {
					slog.Error("failure removing nics column", "err", err)
					panic(err)
				}
				slog.Debug("migration complete", "id", "2024022402", "message", "config.nics dropped")
				vm.DbReconfig()
			}
		}
		setSchemaVersion(2024022402)
	}

	// 2024022403 - remove vm_id from requests
	if schemaVersion < 2024022403 {
		if requests.DbInitialized() {
			// sqlite doesn't let you remove a column, so just nuke it, the requests table isn't critical
			if reqDb.Migrator().HasColumn(&requests.Request{}, "vm_id") {
				slog.Debug("dropping requests table")
				err := reqDb.Migrator().DropTable("requests")
				if err != nil {
					slog.Error("failure dropping requests table", "err", err)
					panic(err)
				}
			}
			requests.DbReconfig()
		}
		setSchemaVersion(2024022403)
	}

	// 2024022403

	slog.Debug("finished custom migration")
}
