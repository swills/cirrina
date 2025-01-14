package vm

import (
	"database/sql"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/iso"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vmnic"
)

//nolint:paralleltest,maintidx
func TestVM_Save(t *testing.T) {
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
		name          string
		mockVMClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields        fields
		wantErr       bool
	}{
		{
			name: "Success",
			mockVMClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 60, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "default", 60, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 81). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "test vm", "test2024081101", 0, sqlmock.AnyArg(), "7915ac31-f554-47ff-9ad8-4e22aacfdf5d"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 27))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_isos` (`vm_id`,`iso_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d", "c3930747-de5d-4b90-bc7d-64cb855f7466", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_disks` (`vm_id`,`disk_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d", "be5c03e7-3e58-41c0-8384-c878e66dd2a9", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:          "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081101",
				Description: "test vm",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        81,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
					},
					VMID:             "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
					CPU:              2,
					Mem:              2048,
					MaxWait:          60,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					Sound:            false,
					SoundIn:          "/dev/dsp0",
					SoundOut:         "/dev/dsp0",
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
					KbdLayout:        "default",
					Com1:             true,
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        115200,
					Com2Speed:        115200,
					Com3Speed:        115200,
					Com4Speed:        115200,
					AutoStartDelay:   60,
					DebugPort:        "AUTO",
				},
				ISOs: []*iso.ISO{
					{
						ID: "c3930747-de5d-4b90-bc7d-64cb855f7466",
					},
				},
				Disks: []*disk.Disk{
					{
						ID: "be5c03e7-3e58-41c0-8384-c878e66dd2a9",
					},
				},
			},
		},
		{
			name: "Fail1",
			mockVMClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 60, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "default", 60, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 81). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "test vm", "test2024081101", 0, sqlmock.AnyArg(), "7915ac31-f554-47ff-9ad8-4e22aacfdf5d"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 27))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_isos` (`vm_id`,`iso_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d", "c3930747-de5d-4b90-bc7d-64cb855f7466", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_disks` (`vm_id`,`disk_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d", "be5c03e7-3e58-41c0-8384-c878e66dd2a9", 0).
					WillReturnError(gorm.ErrInvalidField)
				mock.ExpectRollback()
			},
			fields: fields{
				ID:          "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081101",
				Description: "test vm",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        81,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
					},
					VMID:             "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
					CPU:              2,
					Mem:              2048,
					MaxWait:          60,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					Sound:            false,
					SoundIn:          "/dev/dsp0",
					SoundOut:         "/dev/dsp0",
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
					KbdLayout:        "default",
					Com1:             true,
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        115200,
					Com2Speed:        115200,
					Com3Speed:        115200,
					Com4Speed:        115200,
					AutoStartDelay:   60,
					DebugPort:        "AUTO",
				},
				ISOs: []*iso.ISO{
					{
						ID: "c3930747-de5d-4b90-bc7d-64cb855f7466",
					},
				},
				Disks: []*disk.Disk{
					{
						ID: "be5c03e7-3e58-41c0-8384-c878e66dd2a9",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Fail2",
			mockVMClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 60, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "default", 60, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 81). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "test vm", "test2024081101", 0, sqlmock.AnyArg(), "7915ac31-f554-47ff-9ad8-4e22aacfdf5d"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 27))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_isos` (`vm_id`,`iso_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d", "c3930747-de5d-4b90-bc7d-64cb855f7466", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d").
					WillReturnError(gorm.ErrInvalidField)
			},
			fields: fields{
				ID:          "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081101",
				Description: "test vm",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        81,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
					},
					VMID:             "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
					CPU:              2,
					Mem:              2048,
					MaxWait:          60,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					Sound:            false,
					SoundIn:          "/dev/dsp0",
					SoundOut:         "/dev/dsp0",
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
					KbdLayout:        "default",
					Com1:             true,
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        115200,
					Com2Speed:        115200,
					Com3Speed:        115200,
					Com4Speed:        115200,
					AutoStartDelay:   60,
					DebugPort:        "AUTO",
				},
				ISOs: []*iso.ISO{
					{
						ID: "c3930747-de5d-4b90-bc7d-64cb855f7466",
					},
				},
				Disks: []*disk.Disk{
					{
						ID: "be5c03e7-3e58-41c0-8384-c878e66dd2a9",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Fail3",
			mockVMClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 60, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "default", 60, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 81). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "test vm", "test2024081101", 0, sqlmock.AnyArg(), "7915ac31-f554-47ff-9ad8-4e22aacfdf5d"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 27))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_isos` (`vm_id`,`iso_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d", "c3930747-de5d-4b90-bc7d-64cb855f7466", 0).
					WillReturnError(gorm.ErrInvalidField)
				mock.ExpectRollback()
			},
			fields: fields{
				ID:          "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081101",
				Description: "test vm",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        81,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
					},
					VMID:             "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
					CPU:              2,
					Mem:              2048,
					MaxWait:          60,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					Sound:            false,
					SoundIn:          "/dev/dsp0",
					SoundOut:         "/dev/dsp0",
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
					KbdLayout:        "default",
					Com1:             true,
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        115200,
					Com2Speed:        115200,
					Com3Speed:        115200,
					Com4Speed:        115200,
					AutoStartDelay:   60,
					DebugPort:        "AUTO",
				},
				ISOs: []*iso.ISO{
					{
						ID: "c3930747-de5d-4b90-bc7d-64cb855f7466",
					},
				},
				Disks: []*disk.Disk{
					{
						ID: "be5c03e7-3e58-41c0-8384-c878e66dd2a9",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Fail4",
			mockVMClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 60, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "default", 60, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 81). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "test vm", "test2024081101", 0, sqlmock.AnyArg(), "7915ac31-f554-47ff-9ad8-4e22aacfdf5d"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d").
					WillReturnError(gorm.ErrInvalidField)
			},
			fields: fields{
				ID:          "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081101",
				Description: "test vm",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        81,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
					},
					VMID:             "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
					CPU:              2,
					Mem:              2048,
					MaxWait:          60,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					Sound:            false,
					SoundIn:          "/dev/dsp0",
					SoundOut:         "/dev/dsp0",
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
					KbdLayout:        "default",
					Com1:             true,
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        115200,
					Com2Speed:        115200,
					Com3Speed:        115200,
					Com4Speed:        115200,
					AutoStartDelay:   60,
					DebugPort:        "AUTO",
				},
				ISOs: []*iso.ISO{
					{
						ID: "c3930747-de5d-4b90-bc7d-64cb855f7466",
					},
				},
				Disks: []*disk.Disk{
					{
						ID: "be5c03e7-3e58-41c0-8384-c878e66dd2a9",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Fail5",
			mockVMClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 60, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "default", 60, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 81). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "test vm", "test2024081101", 0, sqlmock.AnyArg(), "7915ac31-f554-47ff-9ad8-4e22aacfdf5d"). //nolint:lll
					WillReturnError(gorm.ErrInvalidField)
				mock.ExpectRollback()
			},
			fields: fields{
				ID:          "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081101",
				Description: "test vm",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        81,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
					},
					VMID:             "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
					CPU:              2,
					Mem:              2048,
					MaxWait:          60,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					Sound:            false,
					SoundIn:          "/dev/dsp0",
					SoundOut:         "/dev/dsp0",
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
					KbdLayout:        "default",
					Com1:             true,
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        115200,
					Com2Speed:        115200,
					Com3Speed:        115200,
					Com4Speed:        115200,
					AutoStartDelay:   60,
					DebugPort:        "AUTO",
				},
				ISOs: []*iso.ISO{
					{
						ID: "c3930747-de5d-4b90-bc7d-64cb855f7466",
					},
				},
				Disks: []*disk.Disk{
					{
						ID: "be5c03e7-3e58-41c0-8384-c878e66dd2a9",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Fail6",
			mockVMClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 60, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "default", 60, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 81). //nolint:lll
					WillReturnError(gorm.ErrInvalidField)
				mock.ExpectRollback()
			},
			fields: fields{
				ID:          "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081101",
				Description: "test vm",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        81,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
					},
					VMID:             "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
					CPU:              2,
					Mem:              2048,
					MaxWait:          60,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					Sound:            false,
					SoundIn:          "/dev/dsp0",
					SoundOut:         "/dev/dsp0",
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
					KbdLayout:        "default",
					Com1:             true,
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        115200,
					Com2Speed:        115200,
					Com3Speed:        115200,
					Com4Speed:        115200,
					AutoStartDelay:   60,
					DebugPort:        "AUTO",
				},
				ISOs: []*iso.ISO{
					{
						ID: "c3930747-de5d-4b90-bc7d-64cb855f7466",
					},
				},
				Disks: []*disk.Disk{
					{
						ID: "be5c03e7-3e58-41c0-8384-c878e66dd2a9",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "FailNilSliceIso",
			mockVMClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
			},
			fields: fields{
				ID: "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
				Config: Config{
					Model: gorm.Model{
						ID: 81,
					},
				},
				ISOs: []*iso.ISO{
					nil,
					{
						ID: "c3930747-de5d-4b90-bc7d-64cb855f7466",
					},
				},
				Disks: []*disk.Disk{
					{
						ID: "be5c03e7-3e58-41c0-8384-c878e66dd2a9",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "FailNilSliceDisk",
			mockVMClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
			},
			fields: fields{
				ID: "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
				Config: Config{
					Model: gorm.Model{
						ID: 81,
					},
				},
				ISOs: []*iso.ISO{
					{
						ID: "c3930747-de5d-4b90-bc7d-64cb855f7466",
					},
				},
				Disks: []*disk.Disk{
					nil,
					{
						ID: "be5c03e7-3e58-41c0-8384-c878e66dd2a9",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "SuccessNoDisks",
			mockVMClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 60, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "default", 60, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 81). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "test vm", "test2024081101", 0, sqlmock.AnyArg(), "7915ac31-f554-47ff-9ad8-4e22aacfdf5d"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 27))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_isos` (`vm_id`,`iso_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d", "c3930747-de5d-4b90-bc7d-64cb855f7466", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			fields: fields{
				ID:          "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081101",
				Description: "test vm",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        81,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
					},
					VMID:             "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
					CPU:              2,
					Mem:              2048,
					MaxWait:          60,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					Sound:            false,
					SoundIn:          "/dev/dsp0",
					SoundOut:         "/dev/dsp0",
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
					KbdLayout:        "default",
					Com1:             true,
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        115200,
					Com2Speed:        115200,
					Com3Speed:        115200,
					Com4Speed:        115200,
					AutoStartDelay:   60,
					DebugPort:        "AUTO",
				},
				ISOs: []*iso.ISO{
					{
						ID: "c3930747-de5d-4b90-bc7d-64cb855f7466",
					},
				},
				Disks: nil,
			},
		},
		{
			name: "SuccessNoISOs",
			mockVMClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 60, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "default", 60, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 81). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "test vm", "test2024081101", 0, sqlmock.AnyArg(), "7915ac31-f554-47ff-9ad8-4e22aacfdf5d"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 27))

				mock.ExpectBegin()
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_disks` (`vm_id`,`disk_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("7915ac31-f554-47ff-9ad8-4e22aacfdf5d", "be5c03e7-3e58-41c0-8384-c878e66dd2a9", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:          "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081101",
				Description: "test vm",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        81,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
					},
					VMID:             "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
					CPU:              2,
					Mem:              2048,
					MaxWait:          60,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					Sound:            false,
					SoundIn:          "/dev/dsp0",
					SoundOut:         "/dev/dsp0",
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
					KbdLayout:        "default",
					Com1:             true,
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        115200,
					Com2Speed:        115200,
					Com3Speed:        115200,
					Com4Speed:        115200,
					AutoStartDelay:   60,
					DebugPort:        "AUTO",
				},
				ISOs: nil,
				Disks: []*disk.Disk{
					{
						ID: "be5c03e7-3e58-41c0-8384-c878e66dd2a9",
					},
				},
			},
		},
		{
			name: "FailEmptyID",
			mockVMClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
			},
			fields: fields{
				ID:          "",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081101",
				Description: "test vm",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        81,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
					},
					VMID:             "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
					CPU:              2,
					Mem:              2048,
					MaxWait:          60,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					Sound:            false,
					SoundIn:          "/dev/dsp0",
					SoundOut:         "/dev/dsp0",
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
					KbdLayout:        "default",
					Com1:             true,
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        115200,
					Com2Speed:        115200,
					Com3Speed:        115200,
					Com4Speed:        115200,
					AutoStartDelay:   60,
					DebugPort:        "AUTO",
				},
				ISOs: []*iso.ISO{
					{
						ID: "c3930747-de5d-4b90-bc7d-64cb855f7466",
					},
				},
				Disks: []*disk.Disk{
					{
						ID: "be5c03e7-3e58-41c0-8384-c878e66dd2a9",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "FailZeroConfigID",
			mockVMClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
			},
			fields: fields{
				ID:          "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081101",
				Description: "test vm",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        0,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
					},
					VMID:             "7915ac31-f554-47ff-9ad8-4e22aacfdf5d",
					CPU:              2,
					Mem:              2048,
					MaxWait:          60,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					Sound:            false,
					SoundIn:          "/dev/dsp0",
					SoundOut:         "/dev/dsp0",
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
					KbdLayout:        "default",
					Com1:             true,
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        115200,
					Com2Speed:        115200,
					Com3Speed:        115200,
					Com4Speed:        115200,
					AutoStartDelay:   60,
					DebugPort:        "AUTO",
				},
				ISOs: []*iso.ISO{
					{
						ID: "c3930747-de5d-4b90-bc7d-64cb855f7466",
					},
				},
				Disks: []*disk.Disk{
					{
						ID: "be5c03e7-3e58-41c0-8384-c878e66dd2a9",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(t.Name())
			testCase.mockVMClosure(testDB, mock)

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

			err := testVM.Save()

			if (err != nil) != testCase.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, testCase.wantErr)
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

//nolint:paralleltest
func TestVM_BhyvectlDestroy(t *testing.T) {
	type fields struct {
		Name string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		fields      fields
		wantPath    bool
		wantPathErr bool
	}{
		{
			name:        "Success",
			mockCmdFunc: "TestVM_BhyvectlDestroySuccess",
			fields: fields{
				Name: "untangledVM",
			},
			wantPath: true,
		},
		{
			name:        "NoPath",
			mockCmdFunc: "TestVM_BhyvectlDestroySuccess",
			fields: fields{
				Name: "untangledVM",
			},
			wantPath: false,
		},
		{
			name:        "PathErr",
			mockCmdFunc: "TestVM_BhyvectlDestroySuccess",
			fields: fields{
				Name: "untangledVM",
			},
			wantPathErr: true,
		},
		{
			name:        "ExecErr",
			mockCmdFunc: "TestVM_BhyvectlDestroyError",
			fields: fields{
				Name: "untangledVM",
			},
			wantPath: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			PathExistsFunc = func(testPath string) (bool, error) {
				if testCase.wantPathErr {
					return true, errors.New("another error") //nolint:goerr113
				}

				if strings.Contains(testPath, "dsp48") {
					return false, errors.New("sound error") //nolint:goerr113
				}

				if strings.Contains(testPath, "dsp45") {
					return false, nil
				}

				if testCase.wantPath {
					return true, nil
				}

				return false, nil
			}

			t.Cleanup(func() { PathExistsFunc = util.PathExists })

			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			testVM := &VM{
				Name: testCase.fields.Name,
			}
			testVM.BhyvectlDestroy()
		})
	}
}

func Test_validateVM(t *testing.T) {
	type args struct {
		vmInst *VM
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Success",
			args: args{
				vmInst: &VM{
					ID:          "6066fece-b9e5-4353-9789-79f1fb18fdd0",
					Name:        "test2024081401",
					Description: "test vm",
					Status:      "STOPPED",
					Config: Config{
						Model: gorm.Model{
							ID: 512,
						},
						VMID: "6066fece-b9e5-4353-9789-79f1fb18fdd0",
						CPU:  2,
						Mem:  2048,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Error",
			args: args{
				vmInst: &VM{
					ID:          "6066fece-b9e5-4353-9789-79f1fb18fdd0",
					Name:        "b0gus!name",
					Description: "test vm",
					Status:      "STOPPED",
					Config: Config{
						Model: gorm.Model{
							ID: 512,
						},
						VMID: "6066fece-b9e5-4353-9789-79f1fb18fdd0",
						CPU:  2,
						Mem:  2048,
					},
				},
			},
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := testCase.args.vmInst.validate()
			if (err != nil) != testCase.wantErr {
				t.Errorf("validateVM() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestVM_Running(t *testing.T) {
	type fields struct {
		Status StatusType
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "Stopped",
			fields: fields{Status: STOPPED},
			want:   false,
		},
		{
			name:   "Starting",
			fields: fields{Status: STARTING},
			want:   false,
		},
		{
			name:   "Running",
			fields: fields{Status: RUNNING},
			want:   true,
		},
		{
			name:   "Stopping",
			fields: fields{Status: STOPPING},
			want:   true,
		},
		{
			name:   "Other",
			fields: fields{Status: "someJunk"},
			want:   false,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testVM := &VM{
				Status: testCase.fields.Status,
			}

			t.Parallel()

			got := testVM.Running()
			if got != testCase.want {
				t.Errorf("Running() = %v, want %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func Test_Exists(t *testing.T) {
	type args struct {
		vmName string
	}

	tests := []struct {
		name        string
		mockClosure func()
		args        args
		want        bool
	}{
		{
			name: "Success",
			mockClosure: func() {
				List.VMList = map[string]*VM{}
			},
			args: args{
				vmName: "46153591-b8b1-419f-8bdb-d82981abb118",
			},
			want: false,
		},
		{
			name: "ErrorExists",
			mockClosure: func() {
				testVM1 := VM{
					ID:     "46153591-b8b1-419f-8bdb-d82981abb119",
					Name:   "test2024082504",
					Status: STOPPED,
					Config: Config{
						Model: gorm.Model{
							ID: 696,
						},
					},
				}

				List.VMList = map[string]*VM{}
				List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmName: "test2024082504",
			},
			want: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()

			got := Exists(testCase.args.vmName)
			if got != testCase.want {
				t.Errorf("vmExists() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest,maintidx
func TestVM_Delete(t *testing.T) {
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
				Instance = &Singleton{VMDB: testDB}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}

				testVM1 := VM{
					ID:     "506fa4f9-307e-40cf-ac3e-9196423042fe",
					Name:   "test2024082504",
					Status: STOPPED,
					Config: Config{
						Model: gorm.Model{
							ID: 378,
						},
					},
				}

				List.VMList = map[string]*VM{}
				List.VMList[testVM1.ID] = &testVM1

				// save
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(false, false, false, false, false, 0, "", false, 0, "", false, 0, "", false, 0, false, "", false, 0, 2, false, "", false, false, false, "", false, false, "", 120, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "", "", false, false, false, false, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 378). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "a test VM", "test2024082510", 0, sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 27))
				mock.ExpectBegin()
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()

				// save
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(false, false, false, false, false, 0, "", false, 0, "", false, 0, "", false, 0, false, "", false, 0, 2, false, "", false, false, false, "", false, false, "", 120, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "", "", false, false, false, false, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 378). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "a test VM", "test2024082510", 0, sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 27))
				mock.ExpectBegin()
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(378).
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"mac",
						"net_type",
						"net_dev_type",
						"switch_id",
						"net_dev",
						"rate_limit",
						"rate_in",
						"rate_out",
						"inst_bridge",
						"inst_epair",
						"config_id",
					}))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `configs` SET `deleted_at`=? WHERE `configs`.`id` = ? AND `configs`.`deleted_at` IS NULL",
					),
				).
					WithArgs(sqlmock.AnyArg(), 378).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `deleted_at`=? WHERE `vms`.`id` = ? AND `vms`.`deleted_at` IS NULL",
					),
				).
					WithArgs(sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:        "506fa4f9-307e-40cf-ac3e-9196423042fe",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "test2024082510",
				Description: "a test VM",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        378,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID:         "506fa4f9-307e-40cf-ac3e-9196423042fe",
					CPU:          2,
					Mem:          2048,
					MaxWait:      120,
					Restart:      true,
					RestartDelay: 0,
					Screen:       true,
					ScreenWidth:  1920,
					ScreenHeight: 1080,
					VNCPort:      "AUTO",
				},
			},
		},
		{
			name: "ErrorDeletingVM",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{VMDB: testDB}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}

				testVM1 := VM{
					ID:     "506fa4f9-307e-40cf-ac3e-9196423042fe",
					Name:   "test2024082504",
					Status: STOPPED,
					Config: Config{
						Model: gorm.Model{
							ID: 378,
						},
					},
				}

				List.VMList = map[string]*VM{}
				List.VMList[testVM1.ID] = &testVM1

				// save
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(false, false, false, false, false, 0, "", false, 0, "", false, 0, "", false, 0, false, "", false, 0, 2, false, "", false, false, false, "", false, false, "", 120, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "", "", false, false, false, false, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 378). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "a test VM", "test2024082510", 0, sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 27))
				mock.ExpectBegin()
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()

				// save
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(false, false, false, false, false, 0, "", false, 0, "", false, 0, "", false, 0, false, "", false, 0, 2, false, "", false, false, false, "", false, false, "", 120, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "", "", false, false, false, false, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 378). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "a test VM", "test2024082510", 0, sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 27))
				mock.ExpectBegin()
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(378).
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"mac",
						"net_type",
						"net_dev_type",
						"switch_id",
						"net_dev",
						"rate_limit",
						"rate_in",
						"rate_out",
						"inst_bridge",
						"inst_epair",
						"config_id",
					}))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `configs` SET `deleted_at`=? WHERE `configs`.`id` = ? AND `configs`.`deleted_at` IS NULL",
					),
				).
					WithArgs(sqlmock.AnyArg(), 378).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `deleted_at`=? WHERE `vms`.`id` = ? AND `vms`.`deleted_at` IS NULL",
					),
				).
					WithArgs(sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnError(gorm.ErrInvalidData)
				mock.ExpectRollback()
			},
			fields: fields{
				ID:        "506fa4f9-307e-40cf-ac3e-9196423042fe",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "test2024082510",
				Description: "a test VM",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        378,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID:         "506fa4f9-307e-40cf-ac3e-9196423042fe",
					CPU:          2,
					Mem:          2048,
					MaxWait:      120,
					Restart:      true,
					RestartDelay: 0,
					Screen:       true,
					ScreenWidth:  1920,
					ScreenHeight: 1080,
					VNCPort:      "AUTO",
				},
			},
			wantErr: true,
		},
		{
			name: "ErrorDeletingConfig",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{VMDB: testDB}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}

				testVM1 := VM{
					ID:     "506fa4f9-307e-40cf-ac3e-9196423042fe",
					Name:   "test2024082504",
					Status: STOPPED,
					Config: Config{
						Model: gorm.Model{
							ID: 378,
						},
					},
				}

				List.VMList = map[string]*VM{}
				List.VMList[testVM1.ID] = &testVM1

				// save
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(false, false, false, false, false, 0, "", false, 0, "", false, 0, "", false, 0, false, "", false, 0, 2, false, "", false, false, false, "", false, false, "", 120, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "", "", false, false, false, false, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 378). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "a test VM", "test2024082510", 0, sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 27))
				mock.ExpectBegin()
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()

				// save
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(false, false, false, false, false, 0, "", false, 0, "", false, 0, "", false, 0, false, "", false, 0, 2, false, "", false, false, false, "", false, false, "", 120, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "", "", false, false, false, false, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 378). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "a test VM", "test2024082510", 0, sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 27))
				mock.ExpectBegin()
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(378).
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"mac",
						"net_type",
						"net_dev_type",
						"switch_id",
						"net_dev",
						"rate_limit",
						"rate_in",
						"rate_out",
						"inst_bridge",
						"inst_epair",
						"config_id",
					}))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `configs` SET `deleted_at`=? WHERE `configs`.`id` = ? AND `configs`.`deleted_at` IS NULL",
					),
				).
					WithArgs(sqlmock.AnyArg(), 378).
					WillReturnError(gorm.ErrInvalidData)
				mock.ExpectRollback()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `deleted_at`=? WHERE `vms`.`id` = ? AND `vms`.`deleted_at` IS NULL",
					),
				).
					WithArgs(sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:        "506fa4f9-307e-40cf-ac3e-9196423042fe",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "test2024082510",
				Description: "a test VM",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        378,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID:         "506fa4f9-307e-40cf-ac3e-9196423042fe",
					CPU:          2,
					Mem:          2048,
					MaxWait:      120,
					Restart:      true,
					RestartDelay: 0,
					Screen:       true,
					ScreenWidth:  1920,
					ScreenHeight: 1080,
					VNCPort:      "AUTO",
				},
			},
			wantErr: false,
		},
		{
			name: "DetachNicsError",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{VMDB: testDB}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}

				testVM1 := VM{
					ID:     "506fa4f9-307e-40cf-ac3e-9196423042fe",
					Name:   "test2024082504",
					Status: STOPPED,
					Config: Config{
						Model: gorm.Model{
							ID: 378,
						},
					},
				}

				List.VMList = map[string]*VM{}
				List.VMList[testVM1.ID] = &testVM1

				// save
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(false, false, false, false, false, 0, "", false, 0, "", false, 0, "", false, 0, false, "", false, 0, 2, false, "", false, false, false, "", false, false, "", 120, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "", "", false, false, false, false, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 378). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "a test VM", "test2024082510", 0, sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 27))
				mock.ExpectBegin()
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()

				// save
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(false, false, false, false, false, 0, "", false, 0, "", false, 0, "", false, 0, false, "", false, 0, 2, false, "", false, false, false, "", false, false, "", 120, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "", "", false, false, false, false, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 378). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "a test VM", "test2024082510", 0, sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 27))
				mock.ExpectBegin()
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(378).
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"mac",
						"net_type",
						"net_dev_type",
						"switch_id",
						"net_dev",
						"rate_limit",
						"rate_in",
						"rate_out",
						"inst_bridge",
						"inst_epair",
						"config_id",
					}).
						AddRow(
							"94defb6b-ddb5-45ca-823d-be7c897e63ee",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:34:56:78",
							"VIRTIONET",
							"TAP",
							"",
							"",
							false,
							0,
							0,
							nil,
							nil,
							555,
						))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs(0, "a description", "", "", "00:11:22:34:56:78", "aNic", "", "TAP",
						"VIRTIONET", 0, false, 0, "", sqlmock.AnyArg(),
						"94defb6b-ddb5-45ca-823d-be7c897e63ee").
					WillReturnError(gorm.ErrInvalidField)
				mock.ExpectRollback()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `configs` SET `deleted_at`=? WHERE `configs`.`id` = ? AND `configs`.`deleted_at` IS NULL",
					),
				).
					WithArgs(sqlmock.AnyArg(), 378).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `deleted_at`=? WHERE `vms`.`id` = ? AND `vms`.`deleted_at` IS NULL",
					),
				).
					WithArgs(sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:        "506fa4f9-307e-40cf-ac3e-9196423042fe",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "test2024082510",
				Description: "a test VM",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        378,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID:         "506fa4f9-307e-40cf-ac3e-9196423042fe",
					CPU:          2,
					Mem:          2048,
					MaxWait:      120,
					Restart:      true,
					RestartDelay: 0,
					Screen:       true,
					ScreenWidth:  1920,
					ScreenHeight: 1080,
					VNCPort:      "AUTO",
				},
			},
		},
		{
			name: "DetachIsosError",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{VMDB: testDB}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}

				testISO := iso.ISO{
					ID:          "47b974ce-733c-4a29-8c4a-8df7272c7dea",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					DeletedAt:   gorm.DeletedAt{},
					Name:        "some.iso",
					Description: "",
					Path:        "/some/path",
					Size:        87123820,
					Checksum:    "some_checksum_should_go_here",
				}

				testVM1 := VM{
					ID:     "506fa4f9-307e-40cf-ac3e-9196423042fe",
					Name:   "test2024082504",
					Status: STOPPED,
					Config: Config{
						Model: gorm.Model{
							ID: 378,
						},
					},
					ISOs: []*iso.ISO{&testISO},
				}

				List.VMList = map[string]*VM{}
				List.VMList[testVM1.ID] = &testVM1

				// save
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(false, false, false, false, false, 0, "", false, 0, "", false, 0, "", false, 0, false, "", false, 0, 2, false, "", false, false, false, "", false, false, "", 120, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "", "", false, false, false, false, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 378). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "a test VM", "test2024082510", 0, sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 27))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"INSERT INTO `vm_isos` (`vm_id`,`iso_id`, `position`) VALUES (?,?,?)",
					),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe", "47b974ce-733c-4a29-8c4a-8df7272c7dea", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()

				// save
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(false, false, false, false, false, 0, "", false, 0, "", false, 0, "", false, 0, false, "", false, 0, 2, false, "", false, false, false, "", false, false, "", 120, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "", "", false, false, false, false, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 378). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "a test VM", "test2024082510", 0, sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe"). //nolint:lll
					WillReturnError(gorm.ErrInvalidField)
				mock.ExpectRollback()

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(378).
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"mac",
						"net_type",
						"net_dev_type",
						"switch_id",
						"net_dev",
						"rate_limit",
						"rate_in",
						"rate_out",
						"inst_bridge",
						"inst_epair",
						"config_id",
					}).
						AddRow(
							"94defb6b-ddb5-45ca-823d-be7c897e63ee",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:34:56:78",
							"VIRTIONET",
							"TAP",
							"",
							"",
							false,
							0,
							0,
							nil,
							nil,
							555,
						))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs(0, "a description", "", "", "00:11:22:34:56:78", "aNic", "", "TAP",
						"VIRTIONET", 0, false, 0, "", sqlmock.AnyArg(),
						"94defb6b-ddb5-45ca-823d-be7c897e63ee").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `configs` SET `deleted_at`=? WHERE `configs`.`id` = ? AND `configs`.`deleted_at` IS NULL",
					),
				).
					WithArgs(sqlmock.AnyArg(), 378).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `deleted_at`=? WHERE `vms`.`id` = ? AND `vms`.`deleted_at` IS NULL",
					),
				).
					WithArgs(sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:        "506fa4f9-307e-40cf-ac3e-9196423042fe",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "test2024082510",
				Description: "a test VM",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        378,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID:         "506fa4f9-307e-40cf-ac3e-9196423042fe",
					CPU:          2,
					Mem:          2048,
					MaxWait:      120,
					Restart:      true,
					RestartDelay: 0,
					Screen:       true,
					ScreenWidth:  1920,
					ScreenHeight: 1080,
					VNCPort:      "AUTO",
				},
				ISOs: []*iso.ISO{
					{
						ID:          "47b974ce-733c-4a29-8c4a-8df7272c7dea",
						CreatedAt:   createUpdateTime,
						UpdatedAt:   createUpdateTime,
						DeletedAt:   gorm.DeletedAt{},
						Name:        "some.iso",
						Description: "",
						Path:        "/some/path",
						Size:        87123820,
						Checksum:    "some_checksum_should_go_here",
					},
				},
			},
		},
		{
			name: "DetachDisksError",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{VMDB: testDB}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}

				testDisk := disk.Disk{
					ID:          "dc28cbf0-56b8-4769-b1c6-32874555a0e0",
					CreatedAt:   time.Time{},
					UpdatedAt:   time.Time{},
					DeletedAt:   gorm.DeletedAt{},
					Name:        "anotherDisk",
					Description: "another test disk",
					Type:        "FILE",
					DevType:     "NVME",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}

				testVM1 := VM{
					ID:     "506fa4f9-307e-40cf-ac3e-9196423042fe",
					Name:   "test2024082504",
					Status: STOPPED,
					Config: Config{
						Model: gorm.Model{
							ID: 378,
						},
					},
					Disks: []*disk.Disk{&testDisk},
				}

				List.VMList = map[string]*VM{}
				List.VMList[testVM1.ID] = &testVM1

				// save
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(false, false, false, false, false, 0, "", false, 0, "", false, 0, "", false, 0, false, "", false, 0, 2, false, "", false, false, false, "", false, false, "", 120, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "", "", false, false, false, false, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 378). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "a test VM", "test2024082510", 0, sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 27))

				mock.ExpectBegin()
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnError(gorm.ErrInvalidField)

				// save
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(false, false, false, false, false, 0, "", false, 0, "", false, 0, "", false, 0, false, "", false, 0, 2, false, "", false, false, false, "", false, false, "", 120, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "", "", false, false, false, false, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 378). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "a test VM", "test2024082510", 0, sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 27))
				mock.ExpectBegin()
				mock.ExpectCommit()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(378).
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"mac",
						"net_type",
						"net_dev_type",
						"switch_id",
						"net_dev",
						"rate_limit",
						"rate_in",
						"rate_out",
						"inst_bridge",
						"inst_epair",
						"config_id",
					}))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `configs` SET `deleted_at`=? WHERE `configs`.`id` = ? AND `configs`.`deleted_at` IS NULL",
					),
				).
					WithArgs(sqlmock.AnyArg(), 378).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `deleted_at`=? WHERE `vms`.`id` = ? AND `vms`.`deleted_at` IS NULL",
					),
				).
					WithArgs(sqlmock.AnyArg(), "506fa4f9-307e-40cf-ac3e-9196423042fe").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:        "506fa4f9-307e-40cf-ac3e-9196423042fe",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "test2024082510",
				Description: "a test VM",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        378,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID:         "506fa4f9-307e-40cf-ac3e-9196423042fe",
					CPU:          2,
					Mem:          2048,
					MaxWait:      120,
					Restart:      true,
					RestartDelay: 0,
					Screen:       true,
					ScreenWidth:  1920,
					ScreenHeight: 1080,
					VNCPort:      "AUTO",
				},
				Disks: []*disk.Disk{{
					ID:          "dc28cbf0-56b8-4769-b1c6-32874555a0e0",
					CreatedAt:   time.Time{},
					UpdatedAt:   time.Time{},
					DeletedAt:   gorm.DeletedAt{},
					Name:        "anotherDisk",
					Description: "another test disk",
					Type:        "FILE",
					DevType:     "NVME",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mockDB := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mockDB)

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

			err := testVM.Delete()
			if (err != nil) != testCase.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, testCase.wantErr)
			}

			mockDB.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mockDB.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

//nolint:paralleltest,maintidx
func TestCreate(t *testing.T) {
	type args struct {
		vmInst *VM
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		wantErr     bool
		wantPath    bool
		wantPathErr bool
	}{
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{VMDB: testDB}

				List.VMList = map[string]*VM{}

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `vms` (`created_at`,`updated_at`,`deleted_at`,`name`,`description`,`status`,`bhyve_pid`,`vnc_port`,`debug_port`,`com1_dev`,`com2_dev`,`com3_dev`,`com4_dev`,`id`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING `id`", //nolint:lll
					),
				).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						nil,
						"test2024082511",
						"a cool vm or something",
						"",
						0,
						0,
						0,
						"",
						"",
						"",
						"",
						sqlmock.AnyArg(),
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("b56b7d60-8075-4fbe-b3bc-8a575ed301a5"))
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `configs` (`created_at`,`updated_at`,`deleted_at`,`vm_id`,`cpu`,`mem`,`max_wait`,`restart`,`restart_delay`,`screen`,`screen_width`,`screen_height`,`vnc_wait`,`vnc_port`,`tablet`,`store_uefi_vars`,`utc_time`,`host_bridge`,`acpi`,`use_hlt`,`exit_on_pause`,`wire_guest_mem`,`destroy_power_off`,`ignore_unknown_msr`,`kbd_layout`,`auto_start`,`sound`,`sound_in`,`sound_out`,`com1`,`com1_dev`,`com1_log`,`com2`,`com2_dev`,`com2_log`,`com3`,`com3_dev`,`com3_log`,`com4`,`com4_dev`,`com4_log`,`extra_args`,`com1_speed`,`com2_speed`,`com3_speed`,`com4_speed`,`auto_start_delay`,`debug`,`debug_wait`,`debug_port`,`priority`,`protect`,`pcpu`,`rbps`,`wbps`,`riops`,`wiops`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT (`id`) DO UPDATE SET `vm_id`=`excluded`.`vm_id` RETURNING `id`", //nolint:lll
					),
				).WithArgs(
					sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "b56b7d60-8075-4fbe-b3bc-8a575ed301a5", 2, 2048, 120, true, 1, true, 1920, 1080, false, "AUTO", true, true, true, true, true, true, true, false, true, true, "default", false, false, "/dev/dsp0", "/dev/dsp0", true, "AUTO", false, false, "AUTO", false, false, "AUTO", false, false, "AUTO", false, "", 115200, 115200, 115200, 115200, 0, false, false, "AUTO", 0, true, 0, 0, 0, 0, 0). //nolint:lll
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(345))
				mock.ExpectCommit()
			},
			args: args{
				vmInst: &VM{
					Name:        "test2024082511",
					Description: "a cool vm or something",
					Config: Config{
						CPU: 2,
						Mem: 2048,
					},
				},
			},
			wantPath: true,
			wantErr:  false,
		},
		{
			name: "ErrorSavingWrongNumberOfRows",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{VMDB: testDB}

				List.VMList = map[string]*VM{}

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `vms` (`created_at`,`updated_at`,`deleted_at`,`name`,`description`,`status`,`bhyve_pid`,`vnc_port`,`debug_port`,`com1_dev`,`com2_dev`,`com3_dev`,`com4_dev`,`id`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING `id`", //nolint:lll
					),
				).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						nil,
						"test2024082511",
						"a cool vm or something",
						"",
						0,
						0,
						0,
						"",
						"",
						"",
						"",
						sqlmock.AnyArg(),
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `configs` (`created_at`,`updated_at`,`deleted_at`,`vm_id`,`cpu`,`mem`,`max_wait`,`restart`,`restart_delay`,`screen`,`screen_width`,`screen_height`,`vnc_wait`,`vnc_port`,`tablet`,`store_uefi_vars`,`utc_time`,`host_bridge`,`acpi`,`use_hlt`,`exit_on_pause`,`wire_guest_mem`,`destroy_power_off`,`ignore_unknown_msr`,`kbd_layout`,`auto_start`,`sound`,`sound_in`,`sound_out`,`com1`,`com1_dev`,`com1_log`,`com2`,`com2_dev`,`com2_log`,`com3`,`com3_dev`,`com3_log`,`com4`,`com4_dev`,`com4_log`,`extra_args`,`com1_speed`,`com2_speed`,`com3_speed`,`com4_speed`,`auto_start_delay`,`debug`,`debug_wait`,`debug_port`,`priority`,`protect`,`pcpu`,`rbps`,`wbps`,`riops`,`wiops`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT (`id`) DO UPDATE SET `vm_id`=`excluded`.`vm_id` RETURNING `id`", //nolint:lll
					),
				).WithArgs(
					sqlmock.AnyArg(), sqlmock.AnyArg(), nil, sqlmock.AnyArg(), 2, 2048, 120, true, 1, true, 1920, 1080, false, "AUTO", true, true, true, true, true, true, true, false, true, true, "default", false, false, "/dev/dsp0", "/dev/dsp0", true, "AUTO", false, false, "AUTO", false, false, "AUTO", false, false, "AUTO", false, "", 115200, 115200, 115200, 115200, 0, false, false, "AUTO", 0, true, 0, 0, 0, 0, 0). //nolint:lll
					WillReturnRows(sqlmock.NewRows([]string{"id"}))
				mock.ExpectCommit()
			},
			args: args{
				vmInst: &VM{
					Name:        "test2024082511",
					Description: "a cool vm or something",
					Config: Config{
						CPU: 2,
						Mem: 2048,
					},
				},
			},
			wantPath: true,
			wantErr:  true,
		},
		{
			name: "ErrorSaving",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{VMDB: testDB}

				List.VMList = map[string]*VM{}

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `vms` (`created_at`,`updated_at`,`deleted_at`,`name`,`description`,`status`,`bhyve_pid`,`vnc_port`,`debug_port`,`com1_dev`,`com2_dev`,`com3_dev`,`com4_dev`,`id`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING `id`", //nolint:lll
					),
				).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						nil,
						"test2024082511",
						"a cool vm or something",
						"",
						0,
						0,
						0,
						"",
						"",
						"",
						"",
						sqlmock.AnyArg(),
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("b56b7d60-8075-4fbe-b3bc-8a575ed301a5"))
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `configs` (`created_at`,`updated_at`,`deleted_at`,`vm_id`,`cpu`,`mem`,`max_wait`,`restart`,`restart_delay`,`screen`,`screen_width`,`screen_height`,`vnc_wait`,`vnc_port`,`tablet`,`store_uefi_vars`,`utc_time`,`host_bridge`,`acpi`,`use_hlt`,`exit_on_pause`,`wire_guest_mem`,`destroy_power_off`,`ignore_unknown_msr`,`kbd_layout`,`auto_start`,`sound`,`sound_in`,`sound_out`,`com1`,`com1_dev`,`com1_log`,`com2`,`com2_dev`,`com2_log`,`com3`,`com3_dev`,`com3_log`,`com4`,`com4_dev`,`com4_log`,`extra_args`,`com1_speed`,`com2_speed`,`com3_speed`,`com4_speed`,`auto_start_delay`,`debug`,`debug_wait`,`debug_port`,`priority`,`protect`,`pcpu`,`rbps`,`wbps`,`riops`,`wiops`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT (`id`) DO UPDATE SET `vm_id`=`excluded`.`vm_id` RETURNING `id`", //nolint:lll
					),
				).WithArgs(
					sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "b56b7d60-8075-4fbe-b3bc-8a575ed301a5", 2, 2048, 120, true, 1, true, 1920, 1080, false, "AUTO", true, true, true, true, true, true, true, false, true, true, "default", false, false, "/dev/dsp0", "/dev/dsp0", true, "AUTO", false, false, "AUTO", false, false, "AUTO", false, false, "AUTO", false, "", 115200, 115200, 115200, 115200, 0, false, false, "AUTO", 0, true, 0, 0, 0, 0, 0). //nolint:lll
					WillReturnError(gorm.ErrInvalidData)
				mock.ExpectRollback()
			},
			args: args{
				vmInst: &VM{
					Name:        "test2024082511",
					Description: "a cool vm or something",
					Config: Config{
						CPU: 2,
						Mem: 2048,
					},
				},
			},
			wantPath: true,
			wantErr:  true,
		},
		{
			name: "ErrorInvalidName",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &Singleton{VMDB: testDB}

				List.VMList = map[string]*VM{}
			},
			args: args{
				vmInst: &VM{
					Name:        "test2024!082511",
					Description: "a cool vm or something",
					Config: Config{
						CPU: 2,
						Mem: 2048,
					},
				},
			},
			wantPath: true,
			wantErr:  true,
		},
		{
			name: "ErrorVMAlreadyExists",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &Singleton{VMDB: testDB}

				testVM1 := VM{
					ID:   "07b4520d-c8c2-4c60-a55e-9c9ed6be688b",
					Name: "test2024082511",
				}

				List.VMList = map[string]*VM{}
				List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmInst: &VM{
					Name:        "test2024082511",
					Description: "a cool vm or something",
					Config: Config{
						CPU: 2,
						Mem: 2048,
					},
				},
			},
			wantPath: true,
			wantErr:  true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			PathExistsFunc = func(_ string) (bool, error) {
				if testCase.wantPathErr {
					return true, errors.New("another error") //nolint:goerr113
				}

				if testCase.wantPath {
					return true, nil
				}

				return false, nil
			}

			t.Cleanup(func() { PathExistsFunc = util.PathExists })

			OsOpenFileFunc = func(_ string, _ int, _ os.FileMode) (*os.File, error) {
				o := os.File{}

				return &o, nil
			}

			t.Cleanup(func() { OsOpenFileFunc = os.OpenFile })

			testDB, mockDB := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mockDB)

			err := Create(testCase.args.vmInst)
			if (err != nil) != testCase.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, testCase.wantErr)
			}

			mockDB.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mockDB.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

//nolint:paralleltest
func Test_checkNicAttachments(t *testing.T) {
	createUpdateTime := time.Now()

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
	}{
		{
			name: "NoNics",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE `vm_nics`.`deleted_at` IS NULL"),
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
								"mac",
								"net_type",
								"net_dev_type",
								"switch_id",
								"net_dev",
								"rate_limit",
								"rate_in",
								"rate_out",
								"inst_bridge",
								"inst_epair",
								"config_id",
							}),
					)
			},
		},
		{
			name: "OneNicNotAttachedToSwitchNorVM",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE `vm_nics`.`deleted_at` IS NULL"),
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
								"mac",
								"net_type",
								"net_dev_type",
								"switch_id",
								"net_dev",
								"rate_limit",
								"rate_in",
								"rate_out",
								"inst_bridge",
								"inst_epair",
								"config_id",
							}).
							AddRow(
								"b72fa5ce-08ac-4f7c-99fa-473e3339d4de",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024121201_int0",
								"first VM nic for test2024121201",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"",
								"",
								false,
								0,
								0,
								"",
								"",
								0,
							),
					)
			},
		},
		{
			name: "OneNicOneVMNoSwitch",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				testVM := VM{
					ID:     "42e845b4-fe69-4240-90d8-6038e81f60f4",
					Name:   "test2024121202",
					Status: "STOPPED",
					Config: Config{
						Model: gorm.Model{
							ID: 1919,
						},
						VMID: "42e845b4-fe69-4240-90d8-6038e81f60f4",
						CPU:  2,
						Mem:  1024,
					},
				}
				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE `vm_nics`.`deleted_at` IS NULL"),
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
								"mac",
								"net_type",
								"net_dev_type",
								"switch_id",
								"net_dev",
								"rate_limit",
								"rate_in",
								"rate_out",
								"inst_bridge",
								"inst_epair",
								"config_id",
							}).
							AddRow(
								"b72fa5ce-08ac-4f7c-99fa-473e3339d4de",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024121201_int0",
								"first VM nic for test2024121201",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"",
								"",
								false,
								0,
								0,
								"",
								"",
								1919,
							),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT vm_id FROM `configs` WHERE id LIKE ?",
					),
				).
					WithArgs(1919).
					WillReturnRows(sqlmock.NewRows([]string{"vm_id"}).
						AddRow("42e845b4-fe69-4240-90d8-6038e81f60f4"))
			},
		},
		{
			name: "OneNicOneVMDoesNotExist",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE `vm_nics`.`deleted_at` IS NULL"),
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
								"mac",
								"net_type",
								"net_dev_type",
								"switch_id",
								"net_dev",
								"rate_limit",
								"rate_in",
								"rate_out",
								"inst_bridge",
								"inst_epair",
								"config_id",
							}).
							AddRow(
								"b72fa5ce-08ac-4f7c-99fa-473e3339d4de",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024121201_int0",
								"first VM nic for test2024121201",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"",
								"",
								false,
								0,
								0,
								"",
								"",
								1919,
							),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT vm_id FROM `configs` WHERE id LIKE ?",
					),
				).
					WithArgs(1919).
					WillReturnRows(sqlmock.NewRows([]string{"vm_id"}).
						AddRow("42e845b4-fe69-4240-90d8-6038e81f60f4"))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(0, "first VM nic for test2024121201", "", "", "AUTO", "test2024121201_int0", "", "TAP",
						"VIRTIONET", 0, false, 0, "", sqlmock.AnyArg(),
						"b72fa5ce-08ac-4f7c-99fa-473e3339d4de").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mock)

			checkNicAttachments()

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

//nolint:paralleltest
func Test_checkDiskAttachments(t *testing.T) {
	createUpdateTime := time.Now()

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
	}{
		{
			name: "NoDisks",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				disk.Instance = &disk.Singleton{DiskDB: testDB}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `disks` WHERE `disks`.`deleted_at` IS NULL"),
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
								"type",
								"dev_type",
								"disk_cache",
								"disk_direct",
							}),
					)
			},
		},
		{
			name: "OneDiskNoVM",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				disk.Instance = &disk.Singleton{DiskDB: testDB}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `disks` WHERE `disks`.`deleted_at` IS NULL"),
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
								"type",
								"dev_type",
								"disk_cache",
								"disk_direct",
							}).
							AddRow(
								"bbdb621e-eddd-4b98-b2fb-26c15cc6e190",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024121201_hd0",
								"a virtual hard disk image",
								"NVME",
								"FILE",
								1,
								0,
							),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT vm_id FROM `vm_disks` WHERE disk_id LIKE ?",
					),
				).WillReturnRows(sqlmock.NewRows([]string{"vm_id"}))
			},
		},
		{
			name: "OneDiskOneVMExists",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				disk.Instance = &disk.Singleton{DiskDB: testDB}

				testVM := VM{
					ID:     "c13a8cab-a725-44bf-8a4e-489e6bd147e3",
					Name:   "test2024121201",
					Status: "STOPPED",
					Config: Config{
						Model: gorm.Model{
							ID: 1920,
						},
						VMID: "c13a8cab-a725-44bf-8a4e-489e6bd147e3",
						CPU:  2,
						Mem:  1024,
					},
				}
				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `disks` WHERE `disks`.`deleted_at` IS NULL"),
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
								"type",
								"dev_type",
								"disk_cache",
								"disk_direct",
							}).
							AddRow(
								"bbdb621e-eddd-4b98-b2fb-26c15cc6e190",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024121201_hd0",
								"a virtual hard disk image",
								"NVME",
								"FILE",
								1,
								0,
							),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT vm_id FROM `vm_disks` WHERE disk_id LIKE ?",
					),
				).WillReturnRows(sqlmock.NewRows([]string{"vm_id"}).AddRow("c13a8cab-a725-44bf-8a4e-489e6bd147e3"))
			},
		},
		{
			name: "OneDiskOneVMDoesNotExist",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{VMDB: testDB}
				disk.Instance = &disk.Singleton{DiskDB: testDB}

				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `disks` WHERE `disks`.`deleted_at` IS NULL"),
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
								"type",
								"dev_type",
								"disk_cache",
								"disk_direct",
							}).
							AddRow(
								"bbdb621e-eddd-4b98-b2fb-26c15cc6e190",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024121201_hd0",
								"a virtual hard disk image",
								"NVME",
								"FILE",
								1,
								0,
							),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT vm_id FROM `vm_disks` WHERE disk_id LIKE ?",
					),
				).WillReturnRows(sqlmock.NewRows([]string{"vm_id"}).AddRow("c13a8cab-a725-44bf-8a4e-489e6bd147e3"))

				mock.ExpectExec(
					regexp.QuoteMeta(
						"DELETE FROM `vm_disks` WHERE `vm_id` = ?",
					),
				).
					WithArgs("c13a8cab-a725-44bf-8a4e-489e6bd147e3").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mock)

			checkDiskAttachments()

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

//nolint:paralleltest
func Test_checkIsoAttachments(t *testing.T) {
	createUpdateTime := time.Now()

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
	}{
		{
			name: "NoISOs",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{ISODB: testDB}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE `isos`.`deleted_at` IS NULL"),
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
								"path",
								"size",
								"checksum",
							}),
					)
			},
		},
		{
			name: "OneISONotAttached",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{ISODB: testDB}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE `isos`.`deleted_at` IS NULL"),
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
								"path",
								"size",
								"checksum",
							}).
							AddRow(
								"5ad10b52-94f9-4666-95a7-7aa4f136aae0",
								createUpdateTime,
								createUpdateTime,
								nil,
								"FreeBSD-13.1-RELEASE-amd64-dvd1.iso",
								"FreeBSD 13.1",
								"/bhyve/isos/FreeBSD-13.1-RELEASE-amd64-dvd1.iso",
								4621281280,
								"326c7a07a393972d3fcd47deaa08e2b932d9298d96e9b4f63a17a2730f93384abc5feb1f511436dc91fcc8b6f56ed25b43dc91d9cdfc700d2655f7e35420d494", //nolint:lll
							),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT vm_id FROM `vm_isos` WHERE iso_id LIKE ?",
					),
				).
					WithArgs("5ad10b52-94f9-4666-95a7-7aa4f136aae0").
					WillReturnRows(sqlmock.NewRows([]string{"vm_id"}))
			},
		},
		{
			name: "OneISOAttachedVMExists",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{ISODB: testDB}

				testVM := VM{
					ID:     "1385122d-555d-48e1-9843-97edd4c193c4",
					Name:   "test2024121201",
					Status: "STOPPED",
					Config: Config{
						Model: gorm.Model{
							ID: 1922,
						},
						VMID: "1385122d-555d-48e1-9843-97edd4c193c4",
						CPU:  2,
						Mem:  1024,
					},
				}
				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE `isos`.`deleted_at` IS NULL"),
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
								"path",
								"size",
								"checksum",
							}).
							AddRow(
								"5ad10b52-94f9-4666-95a7-7aa4f136aae0",
								createUpdateTime,
								createUpdateTime,
								nil,
								"FreeBSD-13.1-RELEASE-amd64-dvd1.iso",
								"FreeBSD 13.1",
								"/bhyve/isos/FreeBSD-13.1-RELEASE-amd64-dvd1.iso",
								4621281280,
								"326c7a07a393972d3fcd47deaa08e2b932d9298d96e9b4f63a17a2730f93384abc5feb1f511436dc91fcc8b6f56ed25b43dc91d9cdfc700d2655f7e35420d494", //nolint:lll
							),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT vm_id FROM `vm_isos` WHERE iso_id LIKE ?",
					),
				).
					WithArgs("5ad10b52-94f9-4666-95a7-7aa4f136aae0").
					WillReturnRows(
						sqlmock.NewRows([]string{"vm_id"}).
							AddRow("1385122d-555d-48e1-9843-97edd4c193c4"),
					)
			},
		},
		{
			name: "OneISOAttachedVMDoesNotExist",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{VMDB: testDB}
				iso.Instance = &iso.Singleton{ISODB: testDB}

				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE `isos`.`deleted_at` IS NULL"),
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
								"path",
								"size",
								"checksum",
							}).
							AddRow(
								"5ad10b52-94f9-4666-95a7-7aa4f136aae0",
								createUpdateTime,
								createUpdateTime,
								nil,
								"FreeBSD-13.1-RELEASE-amd64-dvd1.iso",
								"FreeBSD 13.1",
								"/bhyve/isos/FreeBSD-13.1-RELEASE-amd64-dvd1.iso",
								4621281280,
								"326c7a07a393972d3fcd47deaa08e2b932d9298d96e9b4f63a17a2730f93384abc5feb1f511436dc91fcc8b6f56ed25b43dc91d9cdfc700d2655f7e35420d494", //nolint:lll
							),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT vm_id FROM `vm_isos` WHERE iso_id LIKE ?",
					),
				).
					WithArgs("5ad10b52-94f9-4666-95a7-7aa4f136aae0").
					WillReturnRows(
						sqlmock.NewRows([]string{"vm_id"}).
							AddRow("1385122d-555d-48e1-9843-97edd4c193c4"),
					)

				mock.ExpectExec(
					regexp.QuoteMeta(
						"DELETE FROM `vm_isos` WHERE `vm_id` = ?",
					),
				).
					WithArgs("1385122d-555d-48e1-9843-97edd4c193c4").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mock)

			checkIsoAttachments()

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

// test helpers from here down

//nolint:paralleltest
func TestVM_BhyvectlDestroySuccess(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) >= 2 && cmdWithArgs[1] == "/usr/sbin/bhyvectl" && cmdWithArgs[2] == "--destroy" {
		os.Exit(0)
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestVM_BhyvectlDestroyError(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) >= 2 && cmdWithArgs[1] == "/usr/sbin/bhyvectl" && cmdWithArgs[2] == "--destroy" {
		os.Exit(1)
	}

	os.Exit(0)
}
