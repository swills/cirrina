package vm

import (
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
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
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
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
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
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
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
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
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
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
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
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
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
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
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
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
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
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
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
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
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
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
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
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
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
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
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
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("vmTest")
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
			pathExistsFunc = func(testPath string) (bool, error) {
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

			t.Cleanup(func() { pathExistsFunc = util.PathExists })

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
