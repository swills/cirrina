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
			slog.Error("failed to connect database", "err", err)
			panic("failed to connect database, err: " + err.Error())
		}

		sqlDB, err := metaDB.DB()
		if err != nil {
			slog.Error("failed to create sqlDB database", "err", err)
			panic("failed to create sqlDB database, err: " + err.Error())
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
		slog.Error("failed to auto-migrate meta", "err", err)
		panic("failed to auto-migrate meta, err: " + err.Error())
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

	// 2024110601 - update configs table constraints on screen_width and screen_height
	migration2024110601(schemaVersion, vmDB)

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
					slog.Error("error adding config_id column", "err", err)
					panic(err)
				}

				allVMs, err := vm.GetAllDB()
				if err != nil {
					slog.Error("migration failed", "error", err)
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
						slog.Error("error looking up VMs Nics", "err", err)
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
				slog.Error("migration failed", "error", res.Error)
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
				slog.Error("migration failed", "error", res.Error)
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
				slog.Error("migration failed", "err", err)
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
				slog.Error("migration failed", "error", res.Error)
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
				slog.Error("migration failed", "error", res.Error)
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
				slog.Error("migration failed", "err", err)
				panic(err)
			}
		}

		setSchemaVersion(2024062703)
	}
}

func migration2024110601(schemaVersion uint32, vmDB *gorm.DB) { //nolint:funlen
	if schemaVersion < 2024110601 {
		var res *gorm.DB

		dropIndexConfigsDeletedAt := `drop index idx_configs_deleted_at`

		res = vmDB.Exec(dropIndexConfigsDeletedAt)
		if res.Error != nil {
			slog.Error("migration failed", "error", res.Error)
			panic(res.Error)
		}

		createConfigsNew := `
create table configs_new
(
    id                 integer
        primary key autoincrement,
    created_at         datetime,
    updated_at         datetime,
    deleted_at         datetime,
    vm_id              text
        constraint fk_vms_config
            references vms,
    cpu                integer default 1,
    mem                integer default 128,
    max_wait           integer default 120,
    restart            numeric default true,
    restart_delay      integer default 1,
    screen             numeric default true,
    screen_width       integer default 1920,
    screen_height      integer default 1080,
    vnc_wait           numeric default false,
    vnc_port           text    default "AUTO",
    tablet             numeric default true,
    store_uefi_vars    numeric default true,
    utc_time           numeric default true,
    host_bridge        numeric default true,
    acpi               numeric default true,
    use_hlt            numeric default true,
    exit_on_pause      numeric default true,
    wire_guest_mem     numeric default false,
    destroy_power_off  numeric default true,
    ignore_unknown_msr numeric default true,
    kbd_layout         text    default "default",
    auto_start         numeric default false,
    sound              numeric default false,
    sound_in           text    default "/dev/dsp0",
    sound_out          text    default "/dev/dsp0",
    com1               numeric default true,
    com1_dev           text    default "AUTO",
    com1_log           numeric default false,
    com2               numeric default false,
    com2_dev           text    default "AUTO",
    com2_log           numeric default false,
    com3               numeric default false,
    com3_dev           text    default "AUTO",
    com3_log           numeric default false,
    com4               numeric default false,
    com4_dev           text    default "AUTO",
    com4_log           numeric default false,
    extra_args         text,
    com1_speed         integer default 115200,
    com2_speed         integer default 115200,
    com3_speed         integer default 115200,
    com4_speed         integer default 115200,
    auto_start_delay   integer default 0,
    debug              numeric default false,
    debug_wait         numeric default false,
    debug_port         text    default "AUTO",
    priority           integer default 0,
    protect            numeric default true,
    pcpu               integer,
    rbps               integer,
    wbps               integer,
    riops              integer,
    wiops              integer,
    constraint chk_configs_acpi
        check (acpi IN (0, 1)),
    constraint chk_configs_auto_start
        check (auto_start IN (0, 1)),
    constraint chk_configs_auto_start_delay
        check (auto_start_delay >= 0),
    constraint chk_configs_com1
        check (com1 IN (0, 1)),
    constraint chk_configs_com1_log
        check (com1_log IN (0, 1)),
    constraint chk_configs_com1_speed
        check (com1_speed IN
               (115200, 57600, 38400, 19200, 9600, 4800, 2400, 1200, 600, 300, 200, 150, 134, 110, 75, 50)),
    constraint chk_configs_com2
        check (com2 IN (0, 1)),
    constraint chk_configs_com2_log
        check (com2_log IN (0, 1)),
    constraint chk_configs_com2_speed
        check (com2_speed IN
               (115200, 57600, 38400, 19200, 9600, 4800, 2400, 1200, 600, 300, 200, 150, 134, 110, 75, 50)),
    constraint chk_configs_com3
        check (com3 IN (0, 1)),
    constraint chk_configs_com3_log
        check (com3_log IN (0, 1)),
    constraint chk_configs_com3_speed
        check (com3_speed IN
               (115200, 57600, 38400, 19200, 9600, 4800, 2400, 1200, 600, 300, 200, 150, 134, 110, 75, 50)),
    constraint chk_configs_com4
        check (com4 IN (0, 1)),
    constraint chk_configs_com4_log
        check (com4_log IN (0, 1)),
    constraint chk_configs_com4_speed
        check (com4_speed IN
               (115200, 57600, 38400, 19200, 9600, 4800, 2400, 1200, 600, 300, 200, 150, 134, 110, 75, 50)),
    constraint chk_configs_cpu
        check (cpu >= 1),
    constraint chk_configs_debug
        check (debug IN (0, 1)),
    constraint chk_configs_debug_wait
        check (debug_wait IN (0, 1)),
    constraint chk_configs_destroy_power_off
        check (destroy_power_off IN (0, 1)),
    constraint chk_configs_exit_on_pause
        check (exit_on_pause IN (0, 1)),
    constraint chk_configs_host_bridge
        check (host_bridge IN (0, 1)),
    constraint chk_configs_ignore_unknown_msr
        check (ignore_unknown_msr IN (0, 1)),
    constraint chk_configs_max_wait
        check (max_wait >= 0),
    constraint chk_configs_mem
        check (mem >= 128),
    constraint chk_configs_priority
        check (priority BETWEEN -20 and 20),
    constraint chk_configs_protect
        check (protect IN (0, 1)),
    constraint chk_configs_restart
        check (restart IN (0, 1)),
    constraint chk_configs_restart_delay
        check (restart_delay >= 0),
    constraint chk_configs_screen
        check (screen IN (0, 1)),
    constraint chk_configs_screen_width
        check (screen_width BETWEEN 640 and 3840),
    constraint chk_configs_screen_height
        check (screen_height BETWEEN 480 and 2160),
    constraint chk_configs_sound
        check (sound IN (0, 1)),
    constraint chk_configs_store_uefi_vars
        check (store_uefi_vars IN (0, 1)),
    constraint chk_configs_tablet
        check (tablet IN (0, 1)),
    constraint chk_configs_use_hlt
        check (use_hlt IN (0, 1)),
    constraint chk_configs_utc_time
        check (utc_time IN (0, 1)),
    constraint chk_configs_vnc_wait
        check (vnc_wait IN (0, 1)),
    constraint chk_configs_wire_guest_mem
        check (wire_guest_mem IN (0, 1))
)`

		res = vmDB.Exec(createConfigsNew)
		if res.Error != nil {
			slog.Error("migration failed", "error", res.Error)
			panic(res.Error)
		}

		createConfigsNewDeletedAtIndex := `create index idx_configs_deleted_at on configs_new (deleted_at)`

		res = vmDB.Exec(createConfigsNewDeletedAtIndex)
		if res.Error != nil {
			slog.Error("migration failed", "error", res.Error)
			panic(res.Error)
		}

		insertIntoConfigsNew := `INSERT INTO configs_new (id, created_at, updated_at, deleted_at, vm_id, cpu, mem, max_wait, restart, restart_delay, screen, screen_width, screen_height, vnc_wait, vnc_port, tablet, store_uefi_vars, utc_time, host_bridge, acpi, use_hlt, exit_on_pause, wire_guest_mem, destroy_power_off, ignore_unknown_msr, kbd_layout, auto_start, sound, sound_in, sound_out, com1, com1_dev, com1_log, com2, com2_dev, com2_log, com3, com3_dev, com3_log, com4, com4_dev, com4_log, extra_args, com1_speed, com2_speed, com3_speed, com4_speed, auto_start_delay, debug, debug_wait, debug_port, priority, protect, pcpu, rbps, wbps, riops, wiops) SELECT id, created_at, updated_at, deleted_at, vm_id, cpu, mem, max_wait, restart, restart_delay, screen, screen_width, screen_height, vnc_wait, vnc_port, tablet, store_uefi_vars, utc_time, host_bridge, acpi, use_hlt, exit_on_pause, wire_guest_mem, destroy_power_off, ignore_unknown_msr, kbd_layout, auto_start, sound, sound_in, sound_out, com1, com1_dev, com1_log, com2, com2_dev, com2_log, com3, com3_dev, com3_log, com4, com4_dev, com4_log, extra_args, com1_speed, com2_speed, com3_speed, com4_speed, auto_start_delay, debug, debug_wait, debug_port, priority, protect, pcpu, rbps, wbps, riops, wiops FROM configs` //nolint:lll

		res = vmDB.Exec(insertIntoConfigsNew)
		if res.Error != nil {
			slog.Error("migration failed", "error", res.Error)
			panic(res.Error)
		}

		renameConfigsToConfigsOld := "ALTER TABLE `configs` RENAME TO `configs_2024110601`"

		res = vmDB.Exec(renameConfigsToConfigsOld)
		if res.Error != nil {
			slog.Error("migration failed", "error", res.Error)
			panic(res.Error)
		}

		renameConfigsNewToConfigs := "ALTER TABLE `configs_new` RENAME TO `configs`;"

		res = vmDB.Exec(renameConfigsNewToConfigs)
		if res.Error != nil {
			slog.Error("migration failed", "error", res.Error)
			panic(res.Error)
		}

		setSchemaVersion(2024110601)
	}
}
