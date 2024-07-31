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
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/iso"
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
	isoDB := iso.GetIsoDB()
	diskDB := disk.GetDiskDB()

	schemaVersion := getSchemaVersion()
	// 2024022401 - copy nics from config.nics to vm_nics.config_id
	migration2024022401(schemaVersion, vmNicDB, vmDB)

	// 2024022402 - drop config.nics
	migration2024022402(schemaVersion, vmDB)

	// 2024022403 - remove vm_id from requests
	migration2024022403(schemaVersion, reqDB)

	// 2024062701 - add vm_isos
	migration2024062701(schemaVersion, vmDB)

	// 2024062702 - migrate old iso data to new table
	migration2024062702(schemaVersion, isoDB)

	// 2024062703 - delete is_os from config
	migration2024062703(schemaVersion, vmDB)

	// 2024063001 - add vm_disks
	migration2024063001(schemaVersion, diskDB)

	// 2024063002 - migrate old disk data to new table
	migration2024063002(schemaVersion, diskDB)

	// 2024063003 - delete disks from config
	migration2024063003(schemaVersion, diskDB)

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

				allVMs, err := vm.GetAllDB()
				if err != nil {
					panic(err)
				}

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
		db := vm.GetVMDB()

		if db.Migrator().HasColumn(vm.VM{}, "id") {
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

func migration2024062701(schemaVersion uint32, vmDB *gorm.DB) {
	if schemaVersion < 2024062701 {
		if !vmDB.Migrator().HasTable("vm_isos") {
			//nolint:lll
			vmIsoCreateTableRawSQL := `CREATE TABLE "vm_isos" (
    vm_id text default null
      constraint fk_vm_isos_vm
      references vms,
    iso_id text default null
      constraint fk_vm_isos_iso
      references isos,
    position integer not null,
    primary key (vm_id, iso_id, position),
    CONSTRAINT ` + "`" + `fk_vm_isos_vm` + "`" + ` FOREIGN KEY (` + "`" + `vm_id` + "`" + `) REFERENCES ` + "`" + `vms` + "`" + `(` + "`" + `id` + "`" + `),
    CONSTRAINT ` + "`" + `fk_vm_isos_iso` + "`" + ` FOREIGN KEY (` + "`" + `iso_id` + "`" + `) REFERENCES ` + "`" + `isos` + "`" + `(` + "`" + `id` + "`" + `)
);
`

			res := vmDB.Exec(vmIsoCreateTableRawSQL)
			if res.Error != nil {
				panic(res.Error)
			}
		}

		setSchemaVersion(2024062701)
	}
}

func migration2024062702(schemaVersion uint32, vmDB *gorm.DB) {
	if schemaVersion < 2024062702 {
		type Result struct {
			VMID string
			ISOs string
		}

		haveOldISOsColumn := vmDB.Migrator().HasColumn(&vm.Config{}, "is_os")

		if haveOldISOsColumn {
			var result []Result

			res := vmDB.Raw("SELECT vm_id, is_os from configs where deleted_at is null and is_os != \"\"").Scan(&result)
			if res.Error != nil {
				panic(res.Error)
			}

			for _, val := range result {
				isoList := strings.Split(val.ISOs, ",")
				position := 0

				type newVMISOs struct {
					VMID     string
					IsoID    string
					Position int
				}

				var newVMISOsData []newVMISOs

				for _, ISOv := range isoList {
					if ISOv != "" {
						newVMISOsData = append(newVMISOsData, newVMISOs{VMID: val.VMID, IsoID: ISOv, Position: position})
					}

					position++
				}

				for _, v := range newVMISOsData {
					vmDB.Exec("INSERT INTO vm_isos (vm_id, iso_id, position) VALUES (?,?,?)", v.VMID, v.IsoID, v.Position)
				}
			}
		}

		setSchemaVersion(2024062702)
	}
}

func migration2024062703(schemaVersion uint32, vmDB *gorm.DB) {
	if schemaVersion < 2024062703 {
		haveOldISOsColumn := vmDB.Migrator().HasColumn(&vm.Config{}, "is_os")

		if haveOldISOsColumn {
			err := vmDB.Migrator().DropColumn(&vm.Config{}, "is_os")
			if err != nil {
				panic(err)
			}
		}

		setSchemaVersion(2024062703)
	}
}

func migration2024063001(schemaVersion uint32, vmDB *gorm.DB) {
	if schemaVersion < 2024063001 {
		if !vmDB.Migrator().HasTable("vm_disks") {
			//nolint:lll
			vmDiskCreateTableRawSQL := `CREATE TABLE "vm_disks" (
    vm_id text default null
      constraint fk_vm_disks_vm
      references vms,
    disk_id text default null
      constraint fk_vm_disks_disk
      references disks,
    position integer not null,
    primary key (vm_id, disk_id, position),
    CONSTRAINT ` + "`" + `fk_vm_disks_vm` + "`" + ` FOREIGN KEY (` + "`" + `vm_id` + "`" + `) REFERENCES ` + "`" + `vms` + "`" + `(` + "`" + `id` + "`" + `),
    CONSTRAINT ` + "`" + `fk_vm_disks_disk` + "`" + ` FOREIGN KEY (` + "`" + `disk_id` + "`" + `) REFERENCES ` + "`" + `disks` + "`" + `(` + "`" + `id` + "`" + `)
);
`

			res := vmDB.Exec(vmDiskCreateTableRawSQL)
			if res.Error != nil {
				panic(res.Error)
			}
		}

		setSchemaVersion(2024063001)
	}
}

func migration2024063002(schemaVersion uint32, vmDB *gorm.DB) {
	if schemaVersion < 2024063002 {
		type Result struct {
			VMID  string
			Disks string
		}

		haveOldISOsColumn := vmDB.Migrator().HasColumn(&vm.Config{}, "disks")

		if haveOldISOsColumn {
			var result []Result

			res := vmDB.Raw("SELECT vm_id, disks from configs where deleted_at is null and disks != \"\"").Scan(&result)
			if res.Error != nil {
				panic(res.Error)
			}

			for _, val := range result {
				diskList := strings.Split(val.Disks, ",")
				position := 0

				type newVMDisks struct {
					VMID     string
					DiskID   string
					Position int
				}

				var newVMDisksData []newVMDisks

				for _, DiskV := range diskList {
					if DiskV != "" {
						newVMDisksData = append(newVMDisksData, newVMDisks{VMID: val.VMID, DiskID: DiskV, Position: position})
					}

					position++
				}

				for _, v := range newVMDisksData {
					vmDB.Exec("INSERT INTO vm_disks (vm_id, disk_id, position) VALUES (?,?,?)", v.VMID, v.DiskID, v.Position)
				}
			}
		}

		setSchemaVersion(2024063002)
	}
}

func migration2024063003(schemaVersion uint32, vmDB *gorm.DB) {
	if schemaVersion < 2024063003 {
		haveOldDisksColumn := vmDB.Migrator().HasColumn(&vm.Config{}, "disks")

		if haveOldDisksColumn {
			err := vmDB.Migrator().DropColumn(&vm.Config{}, "disks")
			if err != nil {
				panic(err)
			}
		}

		setSchemaVersion(2024062703)
	}
}
