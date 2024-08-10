package vm

import (
	"database/sql"
	"log/slog"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-test/deep"
	"gorm.io/gorm"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/iso"
)

func TestGetAllDB(t *testing.T) { //nolint:maintidx
	createUpdateTime := time.Now()

	tests := []struct {
		name            string
		mockVMClosure   func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		mockISOClosure  func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		mockDiskClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want            []*VM
		wantErr         bool
	}{
		{
			name: "testVMGetAllDB",
			mockDiskClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				disk1 := &disk.Disk{
					ID:          "1f78cf92-6dc3-4a29-bdd2-0eff351bb2d8",
					Name:        "aSecondTestDisk",
					Description: "some second test disk description",
					Type:        "NVME",
					DevType:     "ILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk2 := &disk.Disk{
					ID:          "44e8ad0d-53a3-4ef5-9611-9289d1b2b331",
					Name:        "aTestDisk",
					Description: "some test disk description",
					Type:        "NVME",
					DevType:     "ILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}

				disk.List.DiskList[disk1.ID] = disk1
				disk.List.DiskList[disk2.ID] = disk2

				disk.Instance = &disk.Singleton{ // prevents parallel testing
					DiskDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `disks` WHERE id = ? AND `disks`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("44e8ad0d-53a3-4ef5-9611-9289d1b2b331").
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"name",
								"description",
								"type",
								"dev_type",
								"disk_cache",
								"disk_direct",
							}).
							AddRow(
								"44e8ad0d-53a3-4ef5-9611-9289d1b2b331",
								createUpdateTime,
								createUpdateTime,
								nil,
								"aTestDisk",
								"some test disk description",
								"NVME",
								"FILE",
								1,
								0,
							),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `disks` WHERE id = ? AND `disks`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("1f78cf92-6dc3-4a29-bdd2-0eff351bb2d8").
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"name",
								"description",
								"type",
								"dev_type",
								"disk_cache",
								"disk_direct",
							}).
							AddRow(
								"1f78cf92-6dc3-4a29-bdd2-0eff351bb2d8",
								createUpdateTime,
								createUpdateTime,
								nil,
								"aSecondTestDisk",
								"some second test disk description",
								"NVME",
								"FILE",
								1,
								0,
							),
					)
			},
			mockISOClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{ // prevents parallel testing
					ISODB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("c2c82cc7-7549-497b-8e21-1ac563aad239").
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"name",
								"description",
								"path",
								"size",
								"checksum",
							}).
							AddRow(
								"c2c82cc7-7549-497b-8e21-1ac563aad239",
								createUpdateTime,
								createUpdateTime,
								nil,
								"someTest.iso",
								"some description",
								"/bhyve/isos/someTest.iso",
								2094096384,
								"259e034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3e3b630ba3", //nolint:lll
							),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("c6e1c826-42a6-4e12-a10f-80ee4845063c").
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"name",
								"description",
								"path",
								"size",
								"checksum",
							}).
							AddRow(
								"c6e1c826-42a6-4e12-a10f-80ee4845063c",
								createUpdateTime,
								createUpdateTime,
								nil,
								"someTest2.iso",
								"some description",
								"/bhyve/isos/someTest2.iso",
								4188192768,
								"259f034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3e3b630ba3", //nolint:lll
							),
					)
			},
			mockVMClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vms` WHERE `vms`.`deleted_at` IS NULL"),
				).
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"name",
								"description",
								"status",
								"bhyve_pid",
								"vnc_port",
								"com1_dev",
								"com2_dev",
								"com3_dev",
								"com4_dev",
								"debug_port",
							}).
							AddRow(
								"38d38177-2309-48a1-8076-0687caa803fb",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2023061001",
								"a test VM",
								"STOPPED",
								0,
								0,
								"",
								"",
								"",
								"",
								0,
							).
							AddRow(
								"263ca626-7e08-4534-8670-06339bcd2381",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2023061002",
								"another test VM",
								"STOPPED",
								0,
								0,
								"",
								"",
								"",
								"",
								0,
							),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `configs` WHERE `configs`.`vm_id` IN (?,?) AND `configs`.`deleted_at` IS NULL"), //nolint:lll
				).
					WithArgs("38d38177-2309-48a1-8076-0687caa803fb", "263ca626-7e08-4534-8670-06339bcd2381").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"id",
							"created_at",
							"updated_at",
							"deleted_at",
							"vm_id",
							"cpu",
							"mem",
							"max_wait",
							"restart",
							"restart_delay",
							"screen",
							"screen_width",
							"screen_height",
							"vnc_wait",
							"vnc_port",
							"tablet",
							"store_uefi_vars",
							"utc_time",
							"host_bridge",
							"acpi",
							"use_hlt",
							"exit_on_pause",
							"wire_guest_mem",
							"destroy_power_off",
							"ignore_unknown_msr",
							"kbd_layout",
							"auto_start",
							"sound",
							"sound_in",
							"sound_out",
							"com1",
							"com1_dev",
							"com1_log",
							"com2",
							"com2_dev",
							"com2_log",
							"com3",
							"com3_dev",
							"com3_log",
							"com4",
							"com4_dev",
							"com4_log",
							"extra_args",
							"com1_speed",
							"com2_speed",
							"com3_speed",
							"com4_speed",
							"auto_start_delay",
							"debug",
							"debug_wait",
							"debug_port",
							"priority",
							"protect",
							"pcpu",
							"rbps",
							"wbps",
							"riops",
							"wiops",
						},
						).
							AddRow(
								1,
								createUpdateTime,
								createUpdateTime,
								nil,
								"38d38177-2309-48a1-8076-0687caa803fb",
								2,
								4096,
								120,
								1,
								1,
								1,
								1920,
								1080,
								0,
								"AUTO",
								1,
								1,
								1,
								1,
								1,
								1,
								1,
								0,
								1,
								1,
								"us_unix",
								0,
								0,
								"/dev/dsp0",
								"/dev/dsp0",
								1,
								"AUTO",
								0,
								0,
								"AUTO",
								0,
								0,
								"AUTO",
								0,
								0,
								"AUTO",
								0,
								"",
								115200,
								115200,
								115200,
								115200,
								0,
								0,
								0,
								"AUTO",
								10,
								sql.NullBool{
									Bool:  false,
									Valid: true,
								},
								0,
								0,
								0,
								300,
								300,
							).AddRow(
							2,
							createUpdateTime,
							createUpdateTime,
							nil,
							"263ca626-7e08-4534-8670-06339bcd2381",
							2,
							4096,
							120,
							1,
							1,
							1,
							1920,
							1080,
							0,
							"AUTO",
							1,
							1,
							1,
							1,
							1,
							1,
							1,
							0,
							1,
							1,
							"us_unix",
							0,
							0,
							"/dev/dsp0",
							"/dev/dsp0",
							1,
							"AUTO",
							0,
							0,
							"AUTO",
							0,
							0,
							"AUTO",
							0,
							0,
							"AUTO",
							0,
							"",
							115200,
							115200,
							115200,
							115200,
							0,
							0,
							0,
							"AUTO",
							10,
							sql.NullBool{
								Bool:  false,
								Valid: true,
							},
							0,
							0,
							0,
							300,
							300,
						),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT vm_id,iso_id,position FROM `vm_isos` WHERE vm_id LIKE ? ORDER BY position"),
				).
					WithArgs("38d38177-2309-48a1-8076-0687caa803fb").
					WillReturnRows(sqlmock.NewRows([]string{"vm_id", "iso_id", "position"}))
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT vm_id,disk_id,position FROM `vm_disks` WHERE vm_id LIKE ? ORDER BY position"),
				).
					WithArgs("38d38177-2309-48a1-8076-0687caa803fb").
					WillReturnRows(sqlmock.NewRows([]string{"vm_id", "disk_id", "position"}))
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT vm_id,iso_id,position FROM `vm_isos` WHERE vm_id LIKE ? ORDER BY position"),
				).
					WithArgs("263ca626-7e08-4534-8670-06339bcd2381").
					WillReturnRows(sqlmock.NewRows([]string{"vm_id", "iso_id", "position"}).
						AddRow("263ca626-7e08-4534-8670-06339bcd2381", "c2c82cc7-7549-497b-8e21-1ac563aad239", 0).
						AddRow("263ca626-7e08-4534-8670-06339bcd2381", "c6e1c826-42a6-4e12-a10f-80ee4845063c", 1),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT vm_id,disk_id,position FROM `vm_disks` WHERE vm_id LIKE ? ORDER BY position"),
				).
					WithArgs("263ca626-7e08-4534-8670-06339bcd2381").
					WillReturnRows(sqlmock.NewRows([]string{"vm_id", "disk_id", "position"}).
						AddRow("263ca626-7e08-4534-8670-06339bcd2381", "44e8ad0d-53a3-4ef5-9611-9289d1b2b331", 0).
						AddRow("263ca626-7e08-4534-8670-06339bcd2381", "1f78cf92-6dc3-4a29-bdd2-0eff351bb2d8", 1),
					)
			},
			want: []*VM{
				{
					ID:          "38d38177-2309-48a1-8076-0687caa803fb",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					DeletedAt:   gorm.DeletedAt{},
					Name:        "test2023061001",
					Description: "a test VM",
					Status:      "STOPPED",
					BhyvePid:    0,
					VNCPort:     0,
					DebugPort:   0,
					proc:        nil,
					mu:          sync.RWMutex{},
					log:         slog.Logger{},
					Config: Config{
						Model: gorm.Model{
							ID:        1,
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
							DeletedAt: gorm.DeletedAt{},
						},
						VMID:             "38d38177-2309-48a1-8076-0687caa803fb",
						CPU:              2,
						Mem:              4096,
						MaxWait:          120,
						Restart:          true,
						RestartDelay:     1,
						Screen:           true,
						ScreenWidth:      1920,
						ScreenHeight:     1080,
						VNCWait:          false,
						VNCPort:          "AUTO",
						Tablet:           true,
						StoreUEFIVars:    true,
						UTCTime:          true,
						HostBridge:       true,
						ACPI:             true,
						UseHLT:           true,
						ExitOnPause:      true,
						WireGuestMem:     false,
						DestroyPowerOff:  true,
						IgnoreUnknownMSR: true,
						KbdLayout:        "us_unix",
						AutoStart:        false,
						Sound:            false,
						SoundIn:          "/dev/dsp0",
						SoundOut:         "/dev/dsp0",
						Com1:             true,
						Com1Dev:          "AUTO",
						Com1Log:          false,
						Com2:             false,
						Com2Dev:          "AUTO",
						Com2Log:          false,
						Com3:             false,
						Com3Dev:          "AUTO",
						Com3Log:          false,
						Com4:             false,
						Com4Dev:          "AUTO",
						Com4Log:          false,
						ExtraArgs:        "",
						Com1Speed:        115200,
						Com2Speed:        115200,
						Com3Speed:        115200,
						Com4Speed:        115200,
						AutoStartDelay:   0,
						Debug:            false,
						DebugWait:        false,
						DebugPort:        "AUTO",
						Priority:         10,
						Protect: sql.NullBool{
							Bool:  false,
							Valid: true,
						},
						Pcpu:  0,
						Rbps:  0,
						Wbps:  0,
						Riops: 300,
						Wiops: 300,
					},
					ISOs:      nil,
					Disks:     nil,
					Com1Dev:   "",
					Com2Dev:   "",
					Com3Dev:   "",
					Com4Dev:   "",
					Com1:      nil,
					Com2:      nil,
					Com3:      nil,
					Com4:      nil,
					Com1lock:  sync.Mutex{},
					Com2lock:  sync.Mutex{},
					Com3lock:  sync.Mutex{},
					Com4lock:  sync.Mutex{},
					Com1rchan: nil,
					Com1write: false,
					Com2rchan: nil,
					Com2write: false,
					Com3rchan: nil,
					Com3write: false,
					Com4rchan: nil,
					Com4write: false,
				},
				{
					ID:          "263ca626-7e08-4534-8670-06339bcd2381",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					DeletedAt:   gorm.DeletedAt{},
					Name:        "test2023061002",
					Description: "another test VM",
					Status:      "STOPPED",
					BhyvePid:    0,
					VNCPort:     0,
					DebugPort:   0,
					proc:        nil,
					mu:          sync.RWMutex{},
					log:         slog.Logger{},
					Config: Config{
						Model: gorm.Model{
							ID:        2,
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
							DeletedAt: gorm.DeletedAt{
								Time:  time.Time{},
								Valid: false,
							},
						},
						VMID:             "263ca626-7e08-4534-8670-06339bcd2381",
						CPU:              2,
						Mem:              4096,
						MaxWait:          120,
						Restart:          true,
						RestartDelay:     1,
						Screen:           true,
						ScreenWidth:      1920,
						ScreenHeight:     1080,
						VNCWait:          false,
						VNCPort:          "AUTO",
						Tablet:           true,
						StoreUEFIVars:    true,
						UTCTime:          true,
						HostBridge:       true,
						ACPI:             true,
						UseHLT:           true,
						ExitOnPause:      true,
						WireGuestMem:     false,
						DestroyPowerOff:  true,
						IgnoreUnknownMSR: true,
						KbdLayout:        "us_unix",
						AutoStart:        false,
						Sound:            false,
						SoundIn:          "/dev/dsp0",
						SoundOut:         "/dev/dsp0",
						Com1:             true,
						Com1Dev:          "AUTO",
						Com1Log:          false,
						Com2:             false,
						Com2Dev:          "AUTO",
						Com2Log:          false,
						Com3:             false,
						Com3Dev:          "AUTO",
						Com3Log:          false,
						Com4:             false,
						Com4Dev:          "AUTO",
						Com4Log:          false,
						ExtraArgs:        "",
						Com1Speed:        115200,
						Com2Speed:        115200,
						Com3Speed:        115200,
						Com4Speed:        115200,
						AutoStartDelay:   0,
						Debug:            false,
						DebugWait:        false,
						DebugPort:        "AUTO",
						Priority:         10,
						Protect: sql.NullBool{
							Bool:  false,
							Valid: true,
						},
						Pcpu:  0,
						Rbps:  0,
						Wbps:  0,
						Riops: 300,
						Wiops: 300,
					},
					ISOs: []*iso.ISO{
						{
							ID:        "c2c82cc7-7549-497b-8e21-1ac563aad239",
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
							DeletedAt: gorm.DeletedAt{
								Time:  time.Time{},
								Valid: false,
							},
							Name:        "someTest.iso",
							Description: "some description",
							Path:        "/bhyve/isos/someTest.iso",
							Size:        2094096384,
							Checksum:    "259e034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3e3b630ba3", //nolint:lll
						},
						{
							ID:        "c6e1c826-42a6-4e12-a10f-80ee4845063c",
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
							DeletedAt: gorm.DeletedAt{
								Time:  time.Time{},
								Valid: false,
							},
							Name:        "someTest2.iso",
							Description: "some description",
							Path:        "/bhyve/isos/someTest2.iso",
							Size:        4188192768,
							Checksum:    "259f034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3e3b630ba3", //nolint:lll
						},
					},
					Disks: []*disk.Disk{
						{
							ID:          "44e8ad0d-53a3-4ef5-9611-9289d1b2b331",
							Name:        "aTestDisk",
							Description: "some test disk description",
							Type:        "NVME",
							DevType:     "ILE",
							DiskCache: sql.NullBool{
								Bool:  true,
								Valid: true,
							},
							DiskDirect: sql.NullBool{
								Bool:  false,
								Valid: true,
							},
						},
						{
							ID:          "1f78cf92-6dc3-4a29-bdd2-0eff351bb2d8",
							Name:        "aSecondTestDisk",
							Description: "some second test disk description",
							Type:        "NVME",
							DevType:     "ILE",
							DiskCache: sql.NullBool{
								Bool:  true,
								Valid: true,
							},
							DiskDirect: sql.NullBool{
								Bool:  false,
								Valid: true,
							},
						},
					},
					Com1Dev:   "",
					Com2Dev:   "",
					Com3Dev:   "",
					Com4Dev:   "",
					Com1:      nil,
					Com2:      nil,
					Com3:      nil,
					Com4:      nil,
					Com1lock:  sync.Mutex{},
					Com2lock:  sync.Mutex{},
					Com3lock:  sync.Mutex{},
					Com4lock:  sync.Mutex{},
					Com1rchan: nil,
					Com1write: false,
					Com2rchan: nil,
					Com2write: false,
					Com3rchan: nil,
					Com3write: false,
					Com4rchan: nil,
					Com4write: false,
				},
			},
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			isoTestDB, isoMock := cirrinadtest.NewMockDB("isoTest")
			testCase.mockISOClosure(isoTestDB, isoMock)

			diskTestDB, diskMock := cirrinadtest.NewMockDB("diskTest")
			testCase.mockDiskClosure(diskTestDB, diskMock)

			vmTestDB, VMmock := cirrinadtest.NewMockDB("vmTest")
			testCase.mockVMClosure(vmTestDB, VMmock)

			got, err := GetAllDB()

			if (err != nil) != testCase.wantErr {
				t.Errorf("CreateEpair() error = %v, wantErr %v", err, testCase.wantErr)
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			VMmock.ExpectClose()

			db, err := vmTestDB.DB()
			if err != nil {
				t.Error(err)
			}

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = VMmock.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestVM_SetStopped(t *testing.T) {
	createUpdateTime := time.Now()

	type fields struct {
		ID          string
		CreatedAt   time.Time
		UpdatedAt   time.Time
		DeletedAt   gorm.DeletedAt
		Name        string
		Description string
		Status      StatusType
		BhyvePid    uint32
		VNCPort     int32
		DebugPort   int32
		Config      Config
		ISOs        []*iso.ISO
		Disks       []*disk.Disk
		Com1Dev     string
		Com2Dev     string
		Com3Dev     string
		Com4Dev     string
		Com1write   bool
		Com2write   bool
		Com3write   bool
		Com4write   bool
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields      fields
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `bhyve_pid`=?,`com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`status`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs(0, "", "", "", "", 0, "STOPPED", 0, sqlmock.AnyArg(), "7c4bc431-5730-11ef-8fec-6c4b9035bdee").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:          "7c4bc431-5730-11ef-8fec-6c4b9035bdee",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "mccoy",
				Description: "a real vm",
				Status:      "RUNNING",
				BhyvePid:    87812,
				VNCPort:     6900,
				Config: Config{
					Model: gorm.Model{
						ID:        0,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID:             "7c4bc431-5730-11ef-8fec-6c4b9035bdee",
					CPU:              2,
					Mem:              2048,
					MaxWait:          120,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					VNCWait:          false,
					VNCPort:          "AUTO",
					Tablet:           true,
					StoreUEFIVars:    true,
					UTCTime:          true,
					HostBridge:       true,
					ACPI:             true,
					UseHLT:           true,
					ExitOnPause:      true,
					DestroyPowerOff:  true,
					IgnoreUnknownMSR: true,
					KbdLayout:        "DEFAULT",
					AutoStart:        true,
					Com1:             true,
					Com1Dev:          "AUTO",
					Com1Speed:        19200,
					AutoStartDelay:   60,
					Protect: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
				},
				Com1Dev:   "/dev/nmdm-mccoy-com1-A",
				Com1write: true,
			},
			wantErr: false,
		},
		{
			name: "Error",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `bhyve_pid`=?,`com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`status`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs(0, "", "", "", "", 0, "STOPPED", 0, sqlmock.AnyArg(), "7c4bc431-5730-11ef-8fec-6c4b9035bdee").
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
				mock.ExpectRollback()
			},
			fields: fields{
				ID:          "7c4bc431-5730-11ef-8fec-6c4b9035bdee",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "mccoy",
				Description: "a real vm",
				Status:      "RUNNING",
				BhyvePid:    87812,
				VNCPort:     6900,
				Config: Config{
					Model: gorm.Model{
						ID:        0,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID:             "7c4bc431-5730-11ef-8fec-6c4b9035bdee",
					CPU:              2,
					Mem:              2048,
					MaxWait:          120,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					VNCWait:          false,
					VNCPort:          "AUTO",
					Tablet:           true,
					StoreUEFIVars:    true,
					UTCTime:          true,
					HostBridge:       true,
					ACPI:             true,
					UseHLT:           true,
					ExitOnPause:      true,
					DestroyPowerOff:  true,
					IgnoreUnknownMSR: true,
					KbdLayout:        "DEFAULT",
					AutoStart:        true,
					Com1:             true,
					Com1Dev:          "AUTO",
					Com1Speed:        19200,
					AutoStartDelay:   60,
					Protect: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
				},
				Com1Dev:   "/dev/nmdm-mccoy-com1-A",
				Com1write: true,
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("vmTest")
			testCase.mockClosure(testDB, mock)

			testVM := &VM{
				ID:          testCase.fields.ID,
				CreatedAt:   testCase.fields.CreatedAt,
				UpdatedAt:   testCase.fields.UpdatedAt,
				DeletedAt:   testCase.fields.DeletedAt,
				Name:        testCase.fields.Name,
				Description: testCase.fields.Description,
				Status:      testCase.fields.Status,
				BhyvePid:    testCase.fields.BhyvePid,
				VNCPort:     testCase.fields.VNCPort,
				DebugPort:   testCase.fields.DebugPort,
				Config:      testCase.fields.Config,
				ISOs:        testCase.fields.ISOs,
				Disks:       testCase.fields.Disks,
				Com1Dev:     testCase.fields.Com1Dev,
				Com2Dev:     testCase.fields.Com2Dev,
				Com3Dev:     testCase.fields.Com3Dev,
				Com4Dev:     testCase.fields.Com4Dev,
				Com1write:   testCase.fields.Com1write,
				Com2write:   testCase.fields.Com2write,
				Com3write:   testCase.fields.Com3write,
				Com4write:   testCase.fields.Com4write,
			}

			err := testVM.SetStopped()

			if (err != nil) != testCase.wantErr {
				t.Errorf("SetStopped() error = %v, wantErr %v", err, testCase.wantErr)
			}

			mock.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mock.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
