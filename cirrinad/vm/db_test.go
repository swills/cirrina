package vm

import (
	"database/sql"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-test/deep"
	"gorm.io/gorm"

	"cirrina/cirrinad/cirrinadtest"
)

func TestGetAllDB(t *testing.T) { //nolint:maintidx
	createUpdateTime := time.Now()

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        []*VM
	}{
		{
			name: "testVMGetAllDB",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}
				mock.ExpectQuery("^SELECT \\* FROM `vms` WHERE `vms`.`deleted_at` IS NULL$").
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
					"^SELECT \\* FROM `configs` WHERE `configs`.`vm_id` IN \\(\\?,\\?\\) AND `configs`.`deleted_at` IS NULL$"). //nolint:lll
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
							"is_os",
							"disks",
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
								"",
								"7d588080-585d-489e-975c-0290fe1be2e0",
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
							"",
							"8967e7a4-c0c6-4aee-8cfe-43e5d953ca71",
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
			},
			want: []*VM{
				{
					Model: gorm.Model{
						ID:        0,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
						DeletedAt: gorm.DeletedAt{},
					},
					ID:          "38d38177-2309-48a1-8076-0687caa803fb",
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
						ISOs:             "",
						Disks:            "7d588080-585d-489e-975c-0290fe1be2e0",
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
					Model: gorm.Model{
						ID:        0,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
						DeletedAt: gorm.DeletedAt{},
					},
					ID:          "263ca626-7e08-4534-8670-06339bcd2381",
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
						ISOs:             "",
						Disks:            "8967e7a4-c0c6-4aee-8cfe-43e5d953ca71",
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
			testDB, mock := cirrinadtest.NewMockDB("diskTest")
			testCase.mockClosure(testDB, mock)

			got := GetAllDB()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			mock.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			if err = db.Close(); err != nil {
				t.Error(err)
			}

			if err = mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
