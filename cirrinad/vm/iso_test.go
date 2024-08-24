package vm

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/iso"
)

//nolint:maintidx,paralleltest
func TestVM_AttachIsos(t *testing.T) {
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

	type args struct {
		ISOs []*iso.ISO
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields      fields
		args        args
		wantErr     bool
	}{
		{
			name: "AddOneISO",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 0, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "us_unix", 120, 1024, 0, 10, false, 0, true, 1, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "AUTO", false, 0, 0, true, sqlmock.AnyArg(), 1). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("AUTO", "AUTO", "AUTO", "AUTO", 0, "another test VM", "testVM1", 0, sqlmock.AnyArg(), "dcc6cfde-25f0-4e8c-80d2-fa7f4f4054bb"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("dcc6cfde-25f0-4e8c-80d2-fa7f4f4054bb").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 27))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_isos` (`vm_id`,`iso_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("dcc6cfde-25f0-4e8c-80d2-fa7f4f4054bb", "6e37ef3f-7199-42de-8d2c-9d7001500acd", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("dcc6cfde-25f0-4e8c-80d2-fa7f4f4054bb").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			fields: fields{
				ID:        "dcc6cfde-25f0-4e8c-80d2-fa7f4f4054bb",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "testVM1",
				Description: "another test VM",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID:        1,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID:             "dcc6cfde-25f0-4e8c-80d2-fa7f4f4054bb",
					CPU:              2,
					Mem:              1024,
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
					WireGuestMem:     true,
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
					Riops: 0,
					Wiops: 0,
				},
				ISOs:      nil,
				Disks:     nil,
				Com1Dev:   "AUTO",
				Com2Dev:   "AUTO",
				Com3Dev:   "AUTO",
				Com4Dev:   "AUTO",
				Com1write: false,
				Com2write: false,
				Com3write: false,
				Com4write: false,
			},
			args: args{ISOs: []*iso.ISO{
				{
					ID:        "6e37ef3f-7199-42de-8d2c-9d7001500acd",
					CreatedAt: time.Time{},
					UpdatedAt: time.Time{},
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
					Name:        "test.iso",
					Description: "a test ISO",
					Path:        "/some/path/test.iso",
					Size:        123123123,
					Checksum:    "stuffgoeshere",
				},
			},
			},
		},
		{
			name: "AddTwoISOs",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 0, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "us_unix", 120, 1024, 0, 10, false, 0, true, 1, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "AUTO", false, 0, 0, true, sqlmock.AnyArg(), 1). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("AUTO", "AUTO", "AUTO", "AUTO", 0, "another test VM", "testVM1", 0, sqlmock.AnyArg(), "dcc6cfde-25f0-4e8c-80d2-fa7f4f4054bb"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("dcc6cfde-25f0-4e8c-80d2-fa7f4f4054bb").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 27))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_isos` (`vm_id`,`iso_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("dcc6cfde-25f0-4e8c-80d2-fa7f4f4054bb", "6e37ef3f-7199-42de-8d2c-9d7001500acd", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_isos` (`vm_id`,`iso_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("dcc6cfde-25f0-4e8c-80d2-fa7f4f4054bb", "63c64708-6fc5-4a8b-858a-e1341b462013", 1).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("dcc6cfde-25f0-4e8c-80d2-fa7f4f4054bb").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			fields: fields{
				ID:        "dcc6cfde-25f0-4e8c-80d2-fa7f4f4054bb",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "testVM1",
				Description: "another test VM",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID:        1,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID:             "dcc6cfde-25f0-4e8c-80d2-fa7f4f4054bb",
					CPU:              2,
					Mem:              1024,
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
					WireGuestMem:     true,
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
					Riops: 0,
					Wiops: 0,
				},
				ISOs:      nil,
				Disks:     nil,
				Com1Dev:   "AUTO",
				Com2Dev:   "AUTO",
				Com3Dev:   "AUTO",
				Com4Dev:   "AUTO",
				Com1write: false,
				Com2write: false,
				Com3write: false,
				Com4write: false,
			},
			args: args{ISOs: []*iso.ISO{
				{
					ID:        "6e37ef3f-7199-42de-8d2c-9d7001500acd",
					CreatedAt: time.Time{},
					UpdatedAt: time.Time{},
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
					Name:        "test.iso",
					Description: "a test ISO",
					Path:        "/some/path/test.iso",
					Size:        123123123,
					Checksum:    "stuffgoeshere",
				},
				{
					ID:        "63c64708-6fc5-4a8b-858a-e1341b462013",
					CreatedAt: time.Time{},
					UpdatedAt: time.Time{},
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
					Name:        "test2.iso",
					Description: "a second test ISO",
					Path:        "/some/path/test2.iso",
					Size:        9123123123,
					Checksum:    "otherstuffgoeshere",
				},
			},
			},
		},
		{
			name: "ErrorVMNotStopped",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
			},
			fields: fields{
				ID:        "dcc6cfde-25f0-4e8c-80d2-fa7f4f4054bb",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "testVM1",
				Description: "another test VM",
				Status:      "RUNNING",
				Config: Config{
					Model: gorm.Model{
						ID:        1,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID:             "dcc6cfde-25f0-4e8c-80d2-fa7f4f4054bb",
					CPU:              2,
					Mem:              1024,
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
					WireGuestMem:     true,
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
					Riops: 0,
					Wiops: 0,
				},
				ISOs:      nil,
				Disks:     nil,
				Com1Dev:   "AUTO",
				Com2Dev:   "AUTO",
				Com3Dev:   "AUTO",
				Com4Dev:   "AUTO",
				Com1write: false,
				Com2write: false,
				Com3write: false,
				Com4write: false,
			},
			args: args{ISOs: []*iso.ISO{
				{
					ID:        "6e37ef3f-7199-42de-8d2c-9d7001500acd",
					CreatedAt: time.Time{},
					UpdatedAt: time.Time{},
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
					Name:        "test.iso",
					Description: "a test ISO",
					Path:        "/some/path/test.iso",
					Size:        123123123,
					Checksum:    "stuffgoeshere",
				},
			},
			},
			wantErr: true,
		},
		{
			name: "ErrorSaving",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 0, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "us_unix", 120, 1024, 0, 10, false, 0, true, 1, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "AUTO", false, 0, 0, true, sqlmock.AnyArg(), 1). //nolint:lll
					// does not matter what error is returned
					WillReturnError(gorm.ErrInvalidField)
				mock.ExpectRollback()
			},
			fields: fields{
				ID:        "dcc6cfde-25f0-4e8c-80d2-fa7f4f4054bb",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "testVM1",
				Description: "another test VM",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID:        1,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID:             "dcc6cfde-25f0-4e8c-80d2-fa7f4f4054bb",
					CPU:              2,
					Mem:              1024,
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
					WireGuestMem:     true,
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
					Riops: 0,
					Wiops: 0,
				},
				ISOs:      nil,
				Disks:     nil,
				Com1Dev:   "AUTO",
				Com2Dev:   "AUTO",
				Com3Dev:   "AUTO",
				Com4Dev:   "AUTO",
				Com1write: false,
				Com2write: false,
				Com3write: false,
				Com4write: false,
			},
			args: args{ISOs: []*iso.ISO{
				{
					ID:        "6e37ef3f-7199-42de-8d2c-9d7001500acd",
					CreatedAt: time.Time{},
					UpdatedAt: time.Time{},
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
					Name:        "test.iso",
					Description: "a test ISO",
					Path:        "/some/path/test.iso",
					Size:        123123123,
					Checksum:    "stuffgoeshere",
				},
			},
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("isoTest")
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
				Com1Dev:     testCase.fields.Com1Dev,
				Com2Dev:     testCase.fields.Com2Dev,
				Com3Dev:     testCase.fields.Com3Dev,
				Com4Dev:     testCase.fields.Com4Dev,
				Com1write:   testCase.fields.Com1write,
				Com2write:   testCase.fields.Com2write,
				Com3write:   testCase.fields.Com3write,
				Com4write:   testCase.fields.Com4write,
			}

			err := testVM.AttachIsos(testCase.args.ISOs)
			if (err != nil) != testCase.wantErr {
				t.Errorf("AttachIsos() error = %v, wantErr %v", err, testCase.wantErr)
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
