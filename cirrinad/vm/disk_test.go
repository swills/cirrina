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

//nolint:paralleltest
func Test_diskAttached(t *testing.T) {
	type args struct {
		aDisk  string
		thisVM *VM
	}

	tests := []struct {
		name        string
		mockClosure func()
		args        args
		want        bool
	}{
		{
			name: "Fail1",
			mockClosure: func() {
				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
			},
			args: args{
				aDisk:  "someDisk",
				thisVM: nil,
			},
			want: false,
		},
		{
			name: "Fail2",
			mockClosure: func() {
				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
			},
			args: args{
				aDisk: "",
				thisVM: &VM{
					ID:          "f143252e-9eb2-43c6-b1c6-8f2d274474a2",
					Name:        "someTestVM",
					Description: "test Vm of the day",
					Status:      "STOPPED",
				},
			},
			want: false,
		},
		{
			name: "Success1",
			mockClosure: func() {
				testVM := VM{
					ID:          "f143252e-9eb2-43c6-b1c6-8f2d274474a2",
					Name:        "someTestVM",
					Description: "test Vm of the day",
					Status:      "STOPPED",
					Disks: []*disk.Disk{
						{
							ID: "b9c51f58-ef3a-425b-80ab-7f67486c0931",
						},
					},
				}

				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM
			},
			args: args{
				aDisk: "b9c51f58-ef3a-425b-80ab-7f67486c0931",
				thisVM: &VM{
					ID:          "a7c48313-de26-472d-a7aa-38f19a7aa794",
					Name:        "someTestVM",
					Description: "test Vm of the day",
					Status:      "STOPPED",
					Disks: []*disk.Disk{
						{
							ID: "b9c51f58-ef3a-425b-80ab-7f67486c0931",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Success2",
			mockClosure: func() {
				testVM := VM{
					ID:          "f143252e-9eb2-43c6-b1c6-8f2d274474a2",
					Name:        "someTestVM",
					Description: "test Vm of the day",
					Status:      "STOPPED",
					Disks:       []*disk.Disk{nil},
				}

				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM
			},
			args: args{
				aDisk: "b9c51f58-ef3a-425b-80ab-7f67486c0931",
				thisVM: &VM{
					ID:          "a7c48313-de26-472d-a7aa-38f19a7aa794",
					Name:        "someTestVM",
					Description: "test Vm of the day",
					Status:      "STOPPED",
					Disks: []*disk.Disk{
						{
							ID: "b9c51f58-ef3a-425b-80ab-7f67486c0931",
						},
					},
				},
			},
			want: false,
		},
	}

	//nolint:paralleltest
	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()

			got := diskAttached(testCase.args.aDisk, testCase.args.thisVM)
			if got != testCase.want {
				t.Errorf("diskAttached() = %v, want %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func Test_validateDisks(t *testing.T) {
	type args struct {
		diskids []string
		thisVM  *VM
	}

	tests := []struct {
		name        string
		mockClosure func()
		args        args
		wantErr     bool
	}{
		{
			name:        "Empty",
			mockClosure: func() {},
			args:        args{diskids: []string{}, thisVM: &VM{}},
			wantErr:     false,
		},
		{
			name:        "BadUUID",
			mockClosure: func() {},
			args:        args{diskids: []string{"80acc7c8-b55d-415"}, thisVM: &VM{}},
			wantErr:     true,
		},
		{
			name: "EmptyVM",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args:    args{diskids: []string{"80acc7c8-b55d-415c-8a9d-2b02608a4894"}, thisVM: &VM{}},
			wantErr: true,
		},
		{
			name: "EmptyDiskName",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args:    args{diskids: []string{"0d4a0338-0b68-4645-b99d-9cbb30df272d"}, thisVM: &VM{}},
			wantErr: true,
		},
		{
			name: "DiskNotInUse",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args:    args{diskids: []string{"0d4a0338-0b68-4645-b99d-9cbb30df272d"}, thisVM: &VM{}},
			wantErr: false,
		},
		{
			name: "DiskDupe",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diskids: []string{
					"0d4a0338-0b68-4645-b99d-9cbb30df272d",
					"0d4a0338-0b68-4645-b99d-9cbb30df272d",
				},
				thisVM: &VM{},
			},
			wantErr: true,
		},
		{
			name: "DiskAlreadyInUse",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst

				testVM := VM{
					ID:          "f143252e-9eb2-43c6-b1c6-8f2d274474a2",
					Name:        "someTestVM",
					Description: "test Vm of the day",
					Status:      "STOPPED",
					Disks: []*disk.Disk{
						{
							ID: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
						},
					},
				}
				List.VMList[testVM.ID] = &testVM
			},
			args: args{
				diskids: []string{"0d4a0338-0b68-4645-b99d-9cbb30df272d"},
				thisVM: &VM{
					ID:          "22a719c6-a4e6-4824-88c2-de5b946e228c",
					Name:        "notTheSame",
					Description: "a completely different VM",
					Status:      "STOPPED",
					Disks: []*disk.Disk{
						{
							ID: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Normal",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "7091c957-3720-4d41-804b-25b443e60cb8",
					Name:        "aNewDisk",
					Description: "a new description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst

				testVM := VM{
					ID:          "f143252e-9eb2-43c6-b1c6-8f2d274474a2",
					Name:        "someTestVM",
					Description: "test Vm of the day",
					Status:      "STOPPED",
					Disks: []*disk.Disk{
						{
							ID: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
						},
					},
				}
				List.VMList[testVM.ID] = &testVM
			},
			args: args{
				diskids: []string{"7091c957-3720-4d41-804b-25b443e60cb8"},
				thisVM: &VM{
					ID:          "22a719c6-a4e6-4824-88c2-de5b946e228c",
					Name:        "notTheSame",
					Description: "a completely different VM",
					Status:      "STOPPED",
					Disks: []*disk.Disk{
						{
							ID: "cf5b91af-0f24-4991-a5d7-8a21c5c483d8",
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("diskTest")

			testCase.mockClosure()

			err := validateDisks(testCase.args.diskids, testCase.args.thisVM)
			if (err != nil) != testCase.wantErr {
				t.Errorf("validateDisks() error = %v, wantErr %v", err, testCase.wantErr)
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

//nolint:paralleltest,maintidx
func TestVM_AttachDisks(t *testing.T) {
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
		diskids []string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields      fields
		args        args
		wantErr     bool
	}{
		{
			name: "NotStopped",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}
			},
			fields: fields{
				ID:        "",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "aCoolVM",
				Description: "a cool VM",
				Status:      "RUNNING",
				BhyvePid:    12345,
				VNCPort:     5900,
				DebugPort:   3434,
				ISOs:        nil,
				Disks: []*disk.Disk{
					{
						ID:        "d5778ab7-7a61-436f-9304-f72bdb5fe068",
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
						Name:        "aDisk",
						Description: "another test disk",
						Type:        "NVME",
						DevType:     "FILE",
						DiskCache: sql.NullBool{
							Bool:  false,
							Valid: false,
						},
						DiskDirect: sql.NullBool{
							Bool:  false,
							Valid: false,
						},
					},
				},
			},
			args: args{
				diskids: []string{"93ec6c94-32c7-408d-8c6f-88b68a514385"},
			},
			wantErr: true,
		},
		{
			name: "BadDisk",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}

				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			fields: fields{
				ID:        "",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "aCoolVM",
				Description: "a cool VM",
				Status:      "STOPPED",
				BhyvePid:    12345,
				VNCPort:     5900,
				DebugPort:   3434,
				ISOs:        nil,
				Disks: []*disk.Disk{
					{
						ID:        "d5778ab7-7a61-436f-9304-f72bdb5fe068",
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
						Name:        "aDisk",
						Description: "another test disk",
						Type:        "NVME",
						DevType:     "FILE",
						DiskCache: sql.NullBool{
							Bool:  false,
							Valid: false,
						},
						DiskDirect: sql.NullBool{
							Bool:  false,
							Valid: false,
						},
					},
				},
			},
			args: args{
				diskids: []string{"93ec6c94-32c7-408d-8c6f-88b68a514385"},
			},
			wantErr: true,
		},
		{
			name: "AddOneDisk",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}

				diskInst := &disk.Disk{
					ID:          "18ad3d8a-d82d-4b0b-b13b-f58d66591f22",
					Name:        "someTestDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 0, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "us_unix", 120, 1024, 0, 10, false, 0, true, 1, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "AUTO", false, 0, 0, true, sqlmock.AnyArg(), 7271). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("AUTO", "AUTO", "AUTO", "AUTO", 3434, "a cool VM", "aCoolVM", 5900, sqlmock.AnyArg(), "3622cde8-3bb5-4aa6-8954-e8b3a098f9f2"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("3622cde8-3bb5-4aa6-8954-e8b3a098f9f2").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 27))
				mock.ExpectBegin()
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("3622cde8-3bb5-4aa6-8954-e8b3a098f9f2").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_disks` (`vm_id`,`disk_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("3622cde8-3bb5-4aa6-8954-e8b3a098f9f2", "18ad3d8a-d82d-4b0b-b13b-f58d66591f22", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:        "3622cde8-3bb5-4aa6-8954-e8b3a098f9f2",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "aCoolVM",
				Description: "a cool VM",
				Status:      "STOPPED",
				BhyvePid:    12345,
				VNCPort:     5900,
				DebugPort:   3434,
				Config: Config{
					Model: gorm.Model{
						ID:        7271,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID:             "3622cde8-3bb5-4aa6-8954-e8b3a098f9f2",
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
				Disks:     []*disk.Disk{},
				Com1Dev:   "AUTO",
				Com2Dev:   "AUTO",
				Com3Dev:   "AUTO",
				Com4Dev:   "AUTO",
				Com1write: false,
				Com2write: false,
				Com3write: false,
				Com4write: false,
			},
			args: args{
				diskids: []string{"18ad3d8a-d82d-4b0b-b13b-f58d66591f22"},
			},
			wantErr: false,
		},
		{
			name: "AddTwoDisks",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}

				diskInst1 := &disk.Disk{
					ID:          "18ad3d8a-d82d-4b0b-b13b-f58d66591f22",
					Name:        "someTestDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				diskInst2 := &disk.Disk{
					ID:          "0f16c751-998b-4ee4-a359-84e0a0cbd21d",
					Name:        "someTestDisk2",
					Description: "a second description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst1.ID] = diskInst1
				disk.List.DiskList[diskInst2.ID] = diskInst2

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 0, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "us_unix", 120, 1024, 0, 10, false, 0, true, 1, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "AUTO", false, 0, 0, true, sqlmock.AnyArg(), 7271). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("AUTO", "AUTO", "AUTO", "AUTO", 3434, "a cool VM", "aCoolVM", 5900, sqlmock.AnyArg(), "3622cde8-3bb5-4aa6-8954-e8b3a098f9f2"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("3622cde8-3bb5-4aa6-8954-e8b3a098f9f2").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 27))
				mock.ExpectBegin()
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("3622cde8-3bb5-4aa6-8954-e8b3a098f9f2").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_disks` (`vm_id`,`disk_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("3622cde8-3bb5-4aa6-8954-e8b3a098f9f2", "18ad3d8a-d82d-4b0b-b13b-f58d66591f22", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_disks` (`vm_id`,`disk_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("3622cde8-3bb5-4aa6-8954-e8b3a098f9f2", "0f16c751-998b-4ee4-a359-84e0a0cbd21d", 1).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:        "3622cde8-3bb5-4aa6-8954-e8b3a098f9f2",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "aCoolVM",
				Description: "a cool VM",
				Status:      "STOPPED",
				BhyvePid:    12345,
				VNCPort:     5900,
				DebugPort:   3434,
				Config: Config{
					Model: gorm.Model{
						ID:        7271,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID:             "3622cde8-3bb5-4aa6-8954-e8b3a098f9f2",
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
				Disks:     []*disk.Disk{},
				Com1Dev:   "AUTO",
				Com2Dev:   "AUTO",
				Com3Dev:   "AUTO",
				Com4Dev:   "AUTO",
				Com1write: false,
				Com2write: false,
				Com3write: false,
				Com4write: false,
			},
			args: args{
				diskids: []string{"18ad3d8a-d82d-4b0b-b13b-f58d66591f22", "0f16c751-998b-4ee4-a359-84e0a0cbd21d"},
			},
			wantErr: false,
		},
		{
			name: "SaveError",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}

				diskInst1 := &disk.Disk{
					ID:          "18ad3d8a-d82d-4b0b-b13b-f58d66591f22",
					Name:        "someTestDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				diskInst2 := &disk.Disk{
					ID:          "0f16c751-998b-4ee4-a359-84e0a0cbd21d",
					Name:        "someTestDisk2",
					Description: "a second description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst1.ID] = diskInst1
				disk.List.DiskList[diskInst2.ID] = diskInst2

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 0, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "us_unix", 120, 1024, 0, 10, false, 0, true, 1, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "AUTO", false, 0, 0, true, sqlmock.AnyArg(), 7271). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("AUTO", "AUTO", "AUTO", "AUTO", 3434, "a cool VM", "aCoolVM", 5900, sqlmock.AnyArg(), "3622cde8-3bb5-4aa6-8954-e8b3a098f9f2"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("3622cde8-3bb5-4aa6-8954-e8b3a098f9f2").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 27))
				mock.ExpectBegin()
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("3622cde8-3bb5-4aa6-8954-e8b3a098f9f2").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_disks` (`vm_id`,`disk_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("3622cde8-3bb5-4aa6-8954-e8b3a098f9f2", "18ad3d8a-d82d-4b0b-b13b-f58d66591f22", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_disks` (`vm_id`,`disk_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("3622cde8-3bb5-4aa6-8954-e8b3a098f9f2", "0f16c751-998b-4ee4-a359-84e0a0cbd21d", 1).
					// does not matter what error is returned
					WillReturnError(gorm.ErrInvalidField)
				mock.ExpectRollback()
			},
			fields: fields{
				ID:        "3622cde8-3bb5-4aa6-8954-e8b3a098f9f2",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "aCoolVM",
				Description: "a cool VM",
				Status:      "STOPPED",
				BhyvePid:    12345,
				VNCPort:     5900,
				DebugPort:   3434,
				Config: Config{
					Model: gorm.Model{
						ID:        7271,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID:             "3622cde8-3bb5-4aa6-8954-e8b3a098f9f2",
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
				Disks:     []*disk.Disk{},
				Com1Dev:   "AUTO",
				Com2Dev:   "AUTO",
				Com3Dev:   "AUTO",
				Com4Dev:   "AUTO",
				Com1write: false,
				Com2write: false,
				Com3write: false,
				Com4write: false,
			},
			args: args{
				diskids: []string{"18ad3d8a-d82d-4b0b-b13b-f58d66591f22", "0f16c751-998b-4ee4-a359-84e0a0cbd21d"},
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("diskTest")
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

			err := testVM.AttachDisks(testCase.args.diskids)
			if (err != nil) != testCase.wantErr {
				t.Errorf("AttachDisks() error = %v, wantErr %v", err, testCase.wantErr)
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
