package vm

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/iso"
	vmswitch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vmnic"
)

//nolint:paralleltest
func TestVM_nicAttached(t *testing.T) {
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

	type args struct {
		aNic string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields      fields
		args        args
		wantErr     bool
	}{
		{
			name: "notAttached",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				testVM := VM{
					ID:          "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
					Name:        "smartTestVM",
					Description: "working VM",
					Status:      "STOPPED",
					Config: Config{
						Model: gorm.Model{
							ID: 723,
						},
						VMID: "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}
				List.VMList[testVM.ID] = &testVM

				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(723).
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
							"01a216af-5b1c-4566-843c-7b74189a9233",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:33:44:55",
							"VIRTIONET",
							"TAP",
							"6ad8f637-22ee-43aa-b9d8-10df5fd7f50f",
							"",
							false,
							0,
							0,
							nil,
							nil,
							723,
						),
					)
			},
			args:    args{aNic: "1e3e509d-e659-43b7-a36b-59e304b94567"},
			wantErr: false,
		},
		{
			name: "alreadyAttached",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				testVM := VM{
					ID:          "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
					Name:        "smartTestVM",
					Description: "working VM",
					Status:      "STOPPED",
					Config: Config{
						Model: gorm.Model{
							ID: 723,
						},
						VMID: "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}
				List.VMList[testVM.ID] = &testVM

				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(723).
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
							"01a216af-5b1c-4566-843c-7b74189a9233",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:33:44:55",
							"VIRTIONET",
							"TAP",
							"6ad8f637-22ee-43aa-b9d8-10df5fd7f50f",
							"",
							false,
							0,
							0,
							nil,
							nil,
							723,
						),
					)
			},
			args:    args{aNic: "01a216af-5b1c-4566-843c-7b74189a9233"},
			wantErr: true,
		},
		{
			name: "getNicErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				testVM := VM{
					ID:          "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
					Name:        "smartTestVM",
					Description: "working VM",
					Status:      "STOPPED",
					Config: Config{
						Model: gorm.Model{
							ID: 723,
						},
						VMID: "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}
				List.VMList[testVM.ID] = &testVM

				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(723).
					WillReturnError(gorm.ErrInvalidData)
			},
			args:    args{aNic: "01a216af-5b1c-4566-843c-7b74189a9233"},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// clear out list from other parallel test runs
			List.VMList = map[string]*VM{}

			testDB, mock := cirrinadtest.NewMockDB(testCase.name)

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

			err := testVM.nicAttached(testCase.args.aNic)
			if (err != nil) != testCase.wantErr {
				t.Errorf("nicAttached() error = %v, wantErr %v", err, testCase.wantErr)
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
func TestVM_validateNics(t *testing.T) {
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

	type args struct {
		nicIDs []string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields      fields
		args        args
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad").
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
							},
						).
							AddRow(
								"a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024050501_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"b7d4cafe-4665-467c-9642-d9c739a9c3b4",
								"",
								false,
								0,
								0,
								"",
								"",
								123,
							),
					)
			},
			fields: fields{
				ID:          "bb8cc2c6-65cc-4a8e-99a7-7cd2b936d69c",
				Name:        "crapVM",
				Description: "unneeded VM",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 892,
					},
					VMID:             "bb8cc2c6-65cc-4a8e-99a7-7cd2b936d69c",
					CPU:              4,
					Mem:              4096,
					MaxWait:          120,
					Restart:          true,
					RestartDelay:     0,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
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
					KbdLayout:        "AUTO",
					SoundIn:          "AUTO",
					SoundOut:         "AUTO",
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        19200,
					Com2Speed:        19200,
					Com3Speed:        19200,
					Com4Speed:        19200,
					DebugPort:        "AUTO",
				},
				ISOs:  nil,
				Disks: nil,
			},
			args: args{
				nicIDs: []string{"a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad"},
			},
			wantErr: false,
		},
		{
			name: "badUUID",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
			},
			fields: fields{
				ID:          "bb8cc2c6-65cc-4a8e-99a7-7cd2b936d69c",
				Name:        "crapVM",
				Description: "unneeded VM",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 892,
					},
					VMID:             "bb8cc2c6-65cc-4a8e-99a7-7cd2b936d69c",
					CPU:              4,
					Mem:              4096,
					MaxWait:          120,
					Restart:          true,
					RestartDelay:     0,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
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
					KbdLayout:        "AUTO",
					SoundIn:          "AUTO",
					SoundOut:         "AUTO",
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        19200,
					Com2Speed:        19200,
					Com3Speed:        19200,
					Com4Speed:        19200,
					DebugPort:        "AUTO",
				},
				ISOs:  nil,
				Disks: nil,
			},
			args: args{
				nicIDs: []string{"a830b4dc-ad25-4"},
			},
			wantErr: true,
		},
		{
			name: "getByIDErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad").
					WillReturnError(errVMInternalDB)
			},
			fields: fields{
				ID:          "bb8cc2c6-65cc-4a8e-99a7-7cd2b936d69c",
				Name:        "crapVM",
				Description: "unneeded VM",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 892,
					},
					VMID:             "bb8cc2c6-65cc-4a8e-99a7-7cd2b936d69c",
					CPU:              4,
					Mem:              4096,
					MaxWait:          120,
					Restart:          true,
					RestartDelay:     0,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
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
					KbdLayout:        "AUTO",
					SoundIn:          "AUTO",
					SoundOut:         "AUTO",
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        19200,
					Com2Speed:        19200,
					Com3Speed:        19200,
					Com4Speed:        19200,
					DebugPort:        "AUTO",
				},
				ISOs:  nil,
				Disks: nil,
			},
			args: args{
				nicIDs: []string{"a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad"},
			},
			wantErr: true,
		},
		{
			name: "emptyNicName",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad").
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
							},
						).
							AddRow(
								"a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad",
								createUpdateTime,
								createUpdateTime,
								nil,
								"",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"b7d4cafe-4665-467c-9642-d9c739a9c3b4",
								"",
								false,
								0,
								0,
								"",
								"",
								123,
							),
					)
			},
			fields: fields{
				ID:          "bb8cc2c6-65cc-4a8e-99a7-7cd2b936d69c",
				Name:        "crapVM",
				Description: "unneeded VM",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 892,
					},
					VMID:             "bb8cc2c6-65cc-4a8e-99a7-7cd2b936d69c",
					CPU:              4,
					Mem:              4096,
					MaxWait:          120,
					Restart:          true,
					RestartDelay:     0,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
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
					KbdLayout:        "AUTO",
					SoundIn:          "AUTO",
					SoundOut:         "AUTO",
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        19200,
					Com2Speed:        19200,
					Com3Speed:        19200,
					Com4Speed:        19200,
					DebugPort:        "AUTO",
				},
				ISOs:  nil,
				Disks: nil,
			},
			args: args{
				nicIDs: []string{"a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad"},
			},
			wantErr: true,
		},
		{
			name: "dupeNic",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad").
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
							},
						).
							AddRow(
								"a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024050501_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"b7d4cafe-4665-467c-9642-d9c739a9c3b4",
								"",
								false,
								0,
								0,
								"",
								"",
								123,
							),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad").
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
							},
						).
							AddRow(
								"a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024050501_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"b7d4cafe-4665-467c-9642-d9c739a9c3b4",
								"",
								false,
								0,
								0,
								"",
								"",
								123,
							),
					)
			},
			fields: fields{
				ID:          "bb8cc2c6-65cc-4a8e-99a7-7cd2b936d69c",
				Name:        "crapVM",
				Description: "unneeded VM",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 892,
					},
					VMID:             "bb8cc2c6-65cc-4a8e-99a7-7cd2b936d69c",
					CPU:              4,
					Mem:              4096,
					MaxWait:          120,
					Restart:          true,
					RestartDelay:     0,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
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
					KbdLayout:        "AUTO",
					SoundIn:          "AUTO",
					SoundOut:         "AUTO",
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        19200,
					Com2Speed:        19200,
					Com3Speed:        19200,
					Com4Speed:        19200,
					DebugPort:        "AUTO",
				},
				ISOs:  nil,
				Disks: nil,
			},
			args: args{
				nicIDs: []string{"a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad", "a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad"},
			},
			wantErr: true,
		},
		{
			name: "nicAttachedErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				testVM := VM{
					ID:          "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
					Name:        "smartTestVM",
					Description: "working VM",
					Status:      "STOPPED",
					Config: Config{
						Model: gorm.Model{
							ID: 723,
						},
						VMID: "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}
				List.VMList[testVM.ID] = &testVM

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad").
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
							},
						).
							AddRow(
								"a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024050501_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"b7d4cafe-4665-467c-9642-d9c739a9c3b4",
								"",
								false,
								0,
								0,
								"",
								"",
								123,
							),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(723).
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
							"a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:33:44:55",
							"VIRTIONET",
							"TAP",
							"6ad8f637-22ee-43aa-b9d8-10df5fd7f50f",
							"",
							false,
							0,
							0,
							nil,
							nil,
							723,
						),
					)
			},
			fields: fields{
				ID:          "bb8cc2c6-65cc-4a8e-99a7-7cd2b936d69c",
				Name:        "crapVM",
				Description: "unneeded VM",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 892,
					},
					VMID:             "bb8cc2c6-65cc-4a8e-99a7-7cd2b936d69c",
					CPU:              4,
					Mem:              4096,
					MaxWait:          120,
					Restart:          true,
					RestartDelay:     0,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
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
					KbdLayout:        "AUTO",
					SoundIn:          "AUTO",
					SoundOut:         "AUTO",
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        19200,
					Com2Speed:        19200,
					Com3Speed:        19200,
					Com4Speed:        19200,
					DebugPort:        "AUTO",
				},
				ISOs:  nil,
				Disks: nil,
			},
			args: args{
				nicIDs: []string{"a830b4dc-ad25-44b8-b7b4-2f2c7594f9ad"},
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// clear out list from other parallel test runs
			List.VMList = map[string]*VM{}

			testDB, mock := cirrinadtest.NewMockDB(testCase.name)

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

			err := testVM.validateNics(testCase.args.nicIDs)
			if (err != nil) != testCase.wantErr {
				t.Errorf("validateNics() error = %v, wantErr %v", err, testCase.wantErr)
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
func TestVM_removeAllNicsFromVM(t *testing.T) {
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
			name: "noneAttached",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).WillReturnRows(sqlmock.NewRows([]string{
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
			},
			fields: fields{
				ID:          "265203e2-a250-4454-934c-463690cf869c",
				Name:        "sleepyVM",
				Description: "a very tired VM",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 555,
					},
					VMID: "265203e2-a250-4454-934c-463690cf869c",
				},
			},
			wantErr: false,
		},
		{
			name: "oneAttached",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(555).
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
							"f1268949-35b5-40ca-a422-22147d38d700",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:33:44:55",
							"VIRTIONET",
							"TAP",
							"6ad8f637-22ee-43aa-b9d8-10df5fd7f50f",
							"",
							false,
							0,
							0,
							nil,
							nil,
							555,
						),
					)

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(0, "a description", "", "", "00:11:22:33:44:55", "aNic", "", "TAP",
						"VIRTIONET", 0, false, 0, "6ad8f637-22ee-43aa-b9d8-10df5fd7f50f", sqlmock.AnyArg(),
						"f1268949-35b5-40ca-a422-22147d38d700").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:          "265203e2-a250-4454-934c-463690cf869c",
				Name:        "sleepyVM",
				Description: "a very tired VM",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 555,
					},
					VMID: "265203e2-a250-4454-934c-463690cf869c",
				},
			},
			wantErr: false,
		},
		{
			name: "saveErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(555).
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
							"f1268949-35b5-40ca-a422-22147d38d700",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:33:44:55",
							"VIRTIONET",
							"TAP",
							"6ad8f637-22ee-43aa-b9d8-10df5fd7f50f",
							"",
							false,
							0,
							0,
							nil,
							nil,
							555,
						),
					)

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(0, "a description", "", "", "00:11:22:33:44:55", "aNic", "", "TAP",
						"VIRTIONET", 0, false, 0, "6ad8f637-22ee-43aa-b9d8-10df5fd7f50f", sqlmock.AnyArg(),
						"f1268949-35b5-40ca-a422-22147d38d700").
					WillReturnError(errVMInternalDB)
				mock.ExpectCommit()
			},
			fields: fields{
				ID:          "265203e2-a250-4454-934c-463690cf869c",
				Name:        "sleepyVM",
				Description: "a very tired VM",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 555,
					},
					VMID: "265203e2-a250-4454-934c-463690cf869c",
				},
			},
			wantErr: true,
		},
		{
			name: "getNicErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(555).
					WillReturnError(errVMInternalDB)
			},
			fields: fields{
				ID:          "265203e2-a250-4454-934c-463690cf869c",
				Name:        "sleepyVM",
				Description: "a very tired VM",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 555,
					},
					VMID: "265203e2-a250-4454-934c-463690cf869c",
				},
			},
			wantErr: true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(testCase.name)

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

			err := testVM.removeAllNicsFromVM()
			if (err != nil) != testCase.wantErr {
				t.Errorf("removeAllNicsFromVM() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest,maintidx
func TestVM_NetStartup(t *testing.T) {
	createUpdateTime := time.Now()

	type fields struct {
		ID          string
		Name        string
		Description string
		Status      StatusType
		BhyvePid    uint32
		VNCPort     int32
		DebugPort   int32
		Config      Config
		ISOs        []*iso.ISO
		Disks       []*disk.Disk
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields      fields
		wantErr     bool
	}{
		{
			name:        "getNicsErr",
			mockCmdFunc: "TestVM_netStartupSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				testVM := VM{
					ID:          "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
					Name:        "smartTestVM",
					Description: "working VM",
					Status:      "STOPPED",
					Config: Config{
						Model: gorm.Model{
							ID: 723,
						},
						VMID: "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}
				List.VMList[testVM.ID] = &testVM

				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(821).
					WillReturnError(gorm.ErrInvalidData)
			},
			fields: fields{
				ID:          "a7fecf30-b54d-44e8-b549-90cbc08471c4",
				Name:        "fridayVM",
				Description: "yay friday",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 821,
					},
					VMID: "a7fecf30-b54d-44e8-b549-90cbc08471c4",
				},
			},
			wantErr: true,
		},
		{
			name:        "noNics",
			mockCmdFunc: "TestVM_netStartupSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(821).
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
			},
			fields: fields{
				ID:          "a7fecf30-b54d-44e8-b549-90cbc08471c4",
				Name:        "fridayVM",
				Description: "yay friday",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 821,
					},
					VMID: "a7fecf30-b54d-44e8-b549-90cbc08471c4",
				},
			},
			wantErr: false,
		},
		{
			name:        "badType",
			mockCmdFunc: "TestVM_netStartupSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(821).
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
							"f1268949-35b5-40ca-a422-22147d38d700",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:33:44:55",
							"VIRTIONET",
							"garbage",
							"6ad8f637-22ee-43aa-b9d8-10df5fd7f50f",
							"",
							false,
							0,
							0,
							nil,
							nil,
							821,
						),
					)
			},
			fields: fields{
				ID:          "a7fecf30-b54d-44e8-b549-90cbc08471c4",
				Name:        "fridayVM",
				Description: "yay friday",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 821,
					},
					VMID: "a7fecf30-b54d-44e8-b549-90cbc08471c4",
				},
			},
			wantErr: false,
		},
		{
			name:        "noUplink",
			mockCmdFunc: "TestVM_netStartupSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(821).
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
							"b424ef1b-34df-41eb-9756-aef024d17896",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:33:44:56",
							"VIRTIONET",
							"TAP",
							"",
							"tap0",
							false,
							0,
							0,
							nil,
							nil,
							821,
						),
					)
			},
			fields: fields{
				ID:          "a7fecf30-b54d-44e8-b549-90cbc08471c4",
				Name:        "fridayVM",
				Description: "yay friday",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 821,
					},
					VMID: "a7fecf30-b54d-44e8-b549-90cbc08471c4",
				},
			},
			wantErr: false,
		},
		{
			name:        "SwitchNotFound",
			mockCmdFunc: "TestVM_netStartupSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(821).
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
							"b424ef1b-34df-41eb-9756-aef024d17896",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:33:44:56",
							"VIRTIONET",
							"TAP",
							"43ad7b62-7866-4f5b-8dfe-4f3f0b348d96",
							"tap0",
							false,
							0,
							0,
							nil,
							nil,
							821,
						),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("43ad7b62-7866-4f5b-8dfe-4f3f0b348d96").
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
								"uplink",
							},
						),
					)
			},
			fields: fields{
				ID:          "a7fecf30-b54d-44e8-b549-90cbc08471c4",
				Name:        "fridayVM",
				Description: "yay friday",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 821,
					},
					VMID: "a7fecf30-b54d-44e8-b549-90cbc08471c4",
				},
			},
			wantErr: false,
		},
		{
			name:        "SuccessIfNotRateLimited",
			mockCmdFunc: "TestVM_netStartupSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(821).
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
							"b424ef1b-34df-41eb-9756-aef024d17896",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:33:44:56",
							"VIRTIONET",
							"TAP",
							"43ad7b62-7866-4f5b-8dfe-4f3f0b348d96",
							"tap0",
							false,
							0,
							0,
							nil,
							nil,
							821,
						),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("43ad7b62-7866-4f5b-8dfe-4f3f0b348d96").
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
								"uplink",
							},
						).
							AddRow(
								"43ad7b62-7866-4f5b-8dfe-4f3f0b348d96",
								createUpdateTime,
								createUpdateTime,
								nil,
								"bridge0",
								"some if switch description",
								"IF",
								"em9",
							),
					)
			},
			fields: fields{
				ID:          "a7fecf30-b54d-44e8-b549-90cbc08471c4",
				Name:        "fridayVM",
				Description: "yay friday",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 821,
					},
					VMID: "a7fecf30-b54d-44e8-b549-90cbc08471c4",
				},
			},
			wantErr: false,
		},
		{
			name:        "SuccessIfConnectError",
			mockCmdFunc: "TestVM_netStartupSuccessIfConnectError",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(821).
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
							"b424ef1b-34df-41eb-9756-aef024d17896",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:33:44:56",
							"VIRTIONET",
							"TAP",
							"43ad7b62-7866-4f5b-8dfe-4f3f0b348d96",
							"tap0",
							false,
							0,
							0,
							nil,
							nil,
							821,
						),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("43ad7b62-7866-4f5b-8dfe-4f3f0b348d96").
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
								"uplink",
							},
						).
							AddRow(
								"43ad7b62-7866-4f5b-8dfe-4f3f0b348d96",
								createUpdateTime,
								createUpdateTime,
								nil,
								"bridge0",
								"some if switch description",
								"IF",
								"em9",
							),
					)
			},
			fields: fields{
				ID:          "a7fecf30-b54d-44e8-b549-90cbc08471c4",
				Name:        "fridayVM",
				Description: "yay friday",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 821,
					},
					VMID: "a7fecf30-b54d-44e8-b549-90cbc08471c4",
				},
			},
			wantErr: false,
		},
		{
			name:        "SuccessIfRateLimited",
			mockCmdFunc: "TestVM_netStartupSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(821).
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
							"b424ef1b-34df-41eb-9756-aef024d17896",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:33:44:56",
							"VIRTIONET",
							"TAP",
							"43ad7b62-7866-4f5b-8dfe-4f3f0b348d96",
							"tap0",
							true,
							400000000,
							100000000,
							nil,
							nil,
							821,
						),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("43ad7b62-7866-4f5b-8dfe-4f3f0b348d96").
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
								"uplink",
							},
						).
							AddRow(
								"43ad7b62-7866-4f5b-8dfe-4f3f0b348d96",
								createUpdateTime,
								createUpdateTime,
								nil,
								"bridge0",
								"some if switch description",
								"IF",
								"em9",
							),
					)

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(
						821,
						"a description",
						"",
						"epair32767",
						"00:11:22:33:44:56",
						"aNic",
						"tap0",
						"TAP",
						"VIRTIONET",
						400000000,
						true,
						100000000,
						"43ad7b62-7866-4f5b-8dfe-4f3f0b348d96",
						sqlmock.AnyArg(),
						"b424ef1b-34df-41eb-9756-aef024d17896",
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(
						821,
						"a description",
						"bridge32767",
						"epair32767",
						"00:11:22:33:44:56",
						"aNic",
						"tap0",
						"TAP",
						"VIRTIONET",
						400000000,
						true,
						100000000,
						"43ad7b62-7866-4f5b-8dfe-4f3f0b348d96",
						sqlmock.AnyArg(),
						"b424ef1b-34df-41eb-9756-aef024d17896",
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:          "a7fecf30-b54d-44e8-b549-90cbc08471c4",
				Name:        "fridayVM",
				Description: "yay friday",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 821,
					},
					VMID: "a7fecf30-b54d-44e8-b549-90cbc08471c4",
				},
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)

			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			// clear out list from other parallel test runs
			List.VMList = map[string]*VM{}

			testDB, mock := cirrinadtest.NewMockDB(testCase.name)
			mock.MatchExpectationsInOrder(true)

			testCase.mockClosure(testDB, mock)

			vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
				VMNicDB: testDB,
			}
			vmswitch.Instance = &vmswitch.Singleton{SwitchDB: testDB}

			testVM := &VM{
				ID:          testCase.fields.ID,
				Name:        testCase.fields.Name,
				Description: testCase.fields.Description,
				Status:      testCase.fields.Status,
				BhyvePid:    testCase.fields.BhyvePid,
				VNCPort:     testCase.fields.VNCPort,
				DebugPort:   testCase.fields.DebugPort,
				Config:      testCase.fields.Config,
				ISOs:        testCase.fields.ISOs,
				Disks:       testCase.fields.Disks,
			}

			err := testVM.netStart()
			if (err != nil) != testCase.wantErr {
				t.Errorf("netStart() error = %v, wantErr %v", err, testCase.wantErr)
			}

			mock.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Fatal(err)
			}

			err = db.Close()
			if err != nil {
				t.Fatal(err)
			}

			err = mock.ExpectationsWereMet()
			if err != nil {
				t.Fatalf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

//nolint:paralleltest,maintidx
func TestVM_NetStop(t *testing.T) {
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
		mockCmdFunc string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields      fields
	}{
		{
			name:        "SuccessNG",
			mockCmdFunc: "TestVM_NetCleanupSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				vmswitch.Instance = &vmswitch.Singleton{SwitchDB: testDB}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(78).
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
							"1584b845-27dc-4386-b5e1-412439e2c87c",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:33:44:56",
							"VIRTIONET",
							"NETGRAPH",
							"82cc8195-1acd-4bad-9d8f-53073e872270",
							"",
							false,
							0,
							0,
							nil,
							nil,
							78,
						),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("82cc8195-1acd-4bad-9d8f-53073e872270").
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
								"uplink",
							},
						).
							AddRow(
								"82cc8195-1acd-4bad-9d8f-53073e872270",
								createUpdateTime,
								createUpdateTime,
								nil,
								"bnet0",
								"some ng switch description",
								"NG",
								"em1",
							),
					)
			},
			fields: fields{
				ID:          "c88437b4-dff3-486d-a2e5-b899318fa14f",
				Name:        "testVmNetgraph",
				Description: "a test VM with a netgraph nic",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID: 78,
					},
					VMID: "c88437b4-dff3-486d-a2e5-b899318fa14f",
					CPU:  2,
					Mem:  1024,
				},
			},
		},
		{
			name:        "SaveErrNG",
			mockCmdFunc: "TestVM_NetCleanupSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				vmswitch.Instance = &vmswitch.Singleton{SwitchDB: testDB}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(78).
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
							"1584b845-27dc-4386-b5e1-412439e2c87c",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:33:44:56",
							"VIRTIONET",
							"NETGRAPH",
							"82cc8195-1acd-4bad-9d8f-53073e872270",
							"",
							false,
							0,
							0,
							nil,
							nil,
							78,
						),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("82cc8195-1acd-4bad-9d8f-53073e872270").
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
								"uplink",
							},
						).
							AddRow(
								"82cc8195-1acd-4bad-9d8f-53073e872270",
								createUpdateTime,
								createUpdateTime,
								nil,
								"bnet0",
								"some ng switch description",
								"NG",
								"em1",
							),
					)
			},
			fields: fields{
				ID:          "c88437b4-dff3-486d-a2e5-b899318fa14f",
				Name:        "testVmNetgraph",
				Description: "a test VM with a netgraph nic",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID: 78,
					},
					VMID: "c88437b4-dff3-486d-a2e5-b899318fa14f",
					CPU:  2,
					Mem:  1024,
				},
			},
		},
		{
			name:        "UnknownNetType",
			mockCmdFunc: "TestVM_NetCleanupSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				vmswitch.Instance = &vmswitch.Singleton{SwitchDB: testDB}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(78).
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
							"1584b845-27dc-4386-b5e1-412439e2c87c",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:33:44:56",
							"VIRTIONET",
							"garbage",
							"82cc8195-1acd-4bad-9d8f-53073e872270",
							"",
							false,
							0,
							0,
							nil,
							nil,
							78,
						),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("82cc8195-1acd-4bad-9d8f-53073e872270").
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
								"uplink",
							},
						).
							AddRow(
								"82cc8195-1acd-4bad-9d8f-53073e872270",
								createUpdateTime,
								createUpdateTime,
								nil,
								"bridge0",
								"some if switch description",
								"IF",
								"em1",
							),
					)
			},
			fields: fields{
				ID:          "c88437b4-dff3-486d-a2e5-b899318fa14f",
				Name:        "testVmNetgraph",
				Description: "a test VM with a netgraph nic",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID: 78,
					},
					VMID: "c88437b4-dff3-486d-a2e5-b899318fa14f",
					CPU:  2,
					Mem:  1024,
				},
			},
		},
		{
			name:        "GetNicErr",
			mockCmdFunc: "TestVM_NetCleanupSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(78).
					WillReturnError(gorm.ErrInvalidData)
			},
			fields: fields{
				ID:          "c88437b4-dff3-486d-a2e5-b899318fa14f",
				Name:        "testVmNetgraph",
				Description: "a test VM with a netgraph nic",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID: 78,
					},
					VMID: "c88437b4-dff3-486d-a2e5-b899318fa14f",
					CPU:  2,
					Mem:  1024,
				},
			},
		},
		{
			name:        "SuccessIF",
			mockCmdFunc: "TestVM_NetCleanupSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				vmswitch.Instance = &vmswitch.Singleton{SwitchDB: testDB}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(78).
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
							"1584b845-27dc-4386-b5e1-412439e2c87c",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:33:44:56",
							"VIRTIONET",
							"TAP",
							"82cc8195-1acd-4bad-9d8f-53073e872270",
							"",
							false,
							0,
							0,
							nil,
							nil,
							78,
						),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("82cc8195-1acd-4bad-9d8f-53073e872270").
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
								"uplink",
							},
						).
							AddRow(
								"82cc8195-1acd-4bad-9d8f-53073e872270",
								createUpdateTime,
								createUpdateTime,
								nil,
								"bridge0",
								"some if switch description",
								"IF",
								"em1",
							),
					)
			},
			fields: fields{
				ID:          "c88437b4-dff3-486d-a2e5-b899318fa14f",
				Name:        "testVmNetgraph",
				Description: "a test VM with a netgraph nic",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID: 78,
					},
					VMID: "c88437b4-dff3-486d-a2e5-b899318fa14f",
					CPU:  2,
					Mem:  1024,
				},
			},
		},
		{
			name:        "FailIF",
			mockCmdFunc: "TestVM_NetCleanupFail",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				vmswitch.Instance = &vmswitch.Singleton{SwitchDB: testDB}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(78).
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
							"1584b845-27dc-4386-b5e1-412439e2c87c",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:33:44:56",
							"VIRTIONET",
							"TAP",
							"82cc8195-1acd-4bad-9d8f-53073e872270",
							"tap0",
							false,
							0,
							0,
							nil,
							nil,
							78,
						),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("82cc8195-1acd-4bad-9d8f-53073e872270").
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
								"uplink",
							},
						).
							AddRow(
								"82cc8195-1acd-4bad-9d8f-53073e872270",
								createUpdateTime,
								createUpdateTime,
								nil,
								"bridge0",
								"some if switch description",
								"IF",
								"em1",
							),
					)
			},
			fields: fields{
				ID:          "c88437b4-dff3-486d-a2e5-b899318fa14f",
				Name:        "testVmNetgraph",
				Description: "a test VM with a netgraph nic",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID: 78,
					},
					VMID: "c88437b4-dff3-486d-a2e5-b899318fa14f",
					CPU:  2,
					Mem:  1024,
				},
			},
		},
		{
			name:        "FailUnknownSwitch",
			mockCmdFunc: "TestVM_NetCleanupSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				vmswitch.Instance = &vmswitch.Singleton{SwitchDB: testDB}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(78).
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
							"1584b845-27dc-4386-b5e1-412439e2c87c",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"00:11:22:33:44:56",
							"VIRTIONET",
							"TAP",
							"82cc8195-1acd-4bad-9d8f-53073e872270",
							"",
							false,
							0,
							0,
							nil,
							nil,
							78,
						),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("82cc8195-1acd-4bad-9d8f-53073e872270").
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
								"uplink",
							},
						),
					)
			},
			fields: fields{
				ID:          "c88437b4-dff3-486d-a2e5-b899318fa14f",
				Name:        "testVmNetgraph",
				Description: "a test VM with a netgraph nic",
				Status:      "STOPPED",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID: 78,
					},
					VMID: "c88437b4-dff3-486d-a2e5-b899318fa14f",
					CPU:  2,
					Mem:  1024,
				},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name,
			func(t *testing.T) {
				// prevents parallel testing
				fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)

				util.SetupTestCmd(fakeCommand)

				t.Cleanup(func() { util.TearDownTestCmd() })

				testDB, mock := cirrinadtest.NewMockDB(testCase.name)
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

				testVM.NetStop()

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
			},
		)
	}
}

//nolint:paralleltest,maintidx
func TestVM_SetNics(t *testing.T) {
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

	type args struct {
		nicIDs []string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		mockCmdFunc string
		fields      fields
		args        args
		wantErr     bool
	}{
		{
			name:        "notStopped",
			mockCmdFunc: "TestVM_SetNicsSuccess",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			fields: fields{
				ID:        "a9394322-ac61-4bab-9fae-e33be4af709e",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "runningTestVM",
				Description: "a VM that is running",
				Status:      "RUNNING",
				BhyvePid:    71892,
				VNCPort:     6900,
				Config: Config{
					VMID: "a9394322-ac61-4bab-9fae-e33be4af709e",
					CPU:  2,
					Mem:  1024,
				},
			},
			wantErr: true,
		},
		{
			name:        "noNics",
			mockCmdFunc: "TestVM_SetNicsSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).
					WithArgs(9191).
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
			},
			fields: fields{
				ID:        "a9394322-ac61-4bab-9fae-e33be4af709e",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "stoppedTestVM",
				Description: "a VM that is stopped",
				Status:      "STOPPED",
				BhyvePid:    71892,
				VNCPort:     6900,
				Config: Config{
					Model: gorm.Model{
						ID: 9191,
					},
					VMID: "a9394322-ac61-4bab-9fae-e33be4af709e",
					CPU:  2,
					Mem:  1024,
				},
			},
		},
		{
			name:        "oneNic",
			mockCmdFunc: "TestVM_SetNicsSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).
					WithArgs(9191).
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

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("c4b4dd48-f186-4cad-83c2-698df1900778").
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
							},
						).
							AddRow(
								"c4b4dd48-f186-4cad-83c2-698df1900778",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024072301_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"4eda640c-eeb8-4762-8441-d2079326df24",
								"",
								false,
								0,
								0,
								"",
								"",
								0,
							),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).
					WithArgs(9191).
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

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("c4b4dd48-f186-4cad-83c2-698df1900778").
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
							},
						).
							AddRow(
								"c4b4dd48-f186-4cad-83c2-698df1900778",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024072301_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"4eda640c-eeb8-4762-8441-d2079326df24",
								"",
								false,
								0,
								0,
								"",
								"",
								0,
							),
					)
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(9191, "another test nic", "", "", "AUTO", "test2024072301_int0", "", "TAP",
						"VIRTIONET", 0, false, 0, "4eda640c-eeb8-4762-8441-d2079326df24", sqlmock.AnyArg(),
						"c4b4dd48-f186-4cad-83c2-698df1900778").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:        "a9394322-ac61-4bab-9fae-e33be4af709e",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "stoppedTestVM",
				Description: "a VM that is stopped",
				Status:      "STOPPED",
				BhyvePid:    71892,
				VNCPort:     6900,
				Config: Config{
					Model: gorm.Model{
						ID: 9191,
					},
					VMID: "a9394322-ac61-4bab-9fae-e33be4af709e",
					CPU:  2,
					Mem:  1024,
				},
			},
			args: args{
				nicIDs: []string{"c4b4dd48-f186-4cad-83c2-698df1900778"},
			},
		},
		{
			name:        "saveErr",
			mockCmdFunc: "TestVM_SetNicsSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).
					WithArgs(9191).
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

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("c4b4dd48-f186-4cad-83c2-698df1900778").
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
							},
						).
							AddRow(
								"c4b4dd48-f186-4cad-83c2-698df1900778",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024072301_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"4eda640c-eeb8-4762-8441-d2079326df24",
								"",
								false,
								0,
								0,
								"",
								"",
								0,
							),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).
					WithArgs(9191).
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

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("c4b4dd48-f186-4cad-83c2-698df1900778").
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
							},
						).
							AddRow(
								"c4b4dd48-f186-4cad-83c2-698df1900778",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024072301_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"4eda640c-eeb8-4762-8441-d2079326df24",
								"",
								false,
								0,
								0,
								"",
								"",
								0,
							),
					)
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(9191, "another test nic", "", "", "AUTO", "test2024072301_int0", "", "TAP",
						"VIRTIONET", 0, false, 0, "4eda640c-eeb8-4762-8441-d2079326df24", sqlmock.AnyArg(),
						"c4b4dd48-f186-4cad-83c2-698df1900778").
					WillReturnError(gorm.ErrInvalidData)
				mock.ExpectRollback()
			},
			fields: fields{
				ID:        "a9394322-ac61-4bab-9fae-e33be4af709e",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "stoppedTestVM",
				Description: "a VM that is stopped",
				Status:      "STOPPED",
				BhyvePid:    71892,
				VNCPort:     6900,
				Config: Config{
					Model: gorm.Model{
						ID: 9191,
					},
					VMID: "a9394322-ac61-4bab-9fae-e33be4af709e",
					CPU:  2,
					Mem:  1024,
				},
			},
			args: args{
				nicIDs: []string{"c4b4dd48-f186-4cad-83c2-698df1900778"},
			},
			wantErr: true,
		},
		{
			name:        "getByIdErr",
			mockCmdFunc: "TestVM_SetNicsSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).
					WithArgs(9191).
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

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("c4b4dd48-f186-4cad-83c2-698df1900778").
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
							},
						).
							AddRow(
								"c4b4dd48-f186-4cad-83c2-698df1900778",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024072301_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"4eda640c-eeb8-4762-8441-d2079326df24",
								"",
								false,
								0,
								0,
								"",
								"",
								0,
							),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).
					WithArgs(9191).
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

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("c4b4dd48-f186-4cad-83c2-698df1900778").
					WillReturnError(gorm.ErrInvalidData)
			},
			fields: fields{
				ID:        "a9394322-ac61-4bab-9fae-e33be4af709e",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "stoppedTestVM",
				Description: "a VM that is stopped",
				Status:      "STOPPED",
				BhyvePid:    71892,
				VNCPort:     6900,
				Config: Config{
					Model: gorm.Model{
						ID: 9191,
					},
					VMID: "a9394322-ac61-4bab-9fae-e33be4af709e",
					CPU:  2,
					Mem:  1024,
				},
			},
			args: args{
				nicIDs: []string{"c4b4dd48-f186-4cad-83c2-698df1900778"},
			},
			wantErr: true,
		},
		{
			name:        "validateErr",
			mockCmdFunc: "TestVM_SetNicsSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).
					WithArgs(9191).
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

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("c4b4dd48-f186-4cad-83c2-698df1900778").
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
							},
						).
							AddRow(
								"c4b4dd48-f186-4cad-83c2-698df1900778",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024072301_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"4eda640c-eeb8-4762-8441-d2079326df24",
								"",
								false,
								0,
								0,
								"",
								"",
								0,
							),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).
					WithArgs(9191).
					WillReturnError(gorm.ErrInvalidData)
			},
			fields: fields{
				ID:        "a9394322-ac61-4bab-9fae-e33be4af709e",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "stoppedTestVM",
				Description: "a VM that is stopped",
				Status:      "STOPPED",
				BhyvePid:    71892,
				VNCPort:     6900,
				Config: Config{
					Model: gorm.Model{
						ID: 9191,
					},
					VMID: "a9394322-ac61-4bab-9fae-e33be4af709e",
					CPU:  2,
					Mem:  1024,
				},
			},
			args: args{
				nicIDs: []string{"c4b4dd48-f186-4cad-83c2-698df1900778"},
			},
			wantErr: true,
		},
		{
			name:        "removeAllNicsErr",
			mockCmdFunc: "TestVM_SetNicsSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).
					WithArgs(9191).
					WillReturnError(gorm.ErrInvalidData)
			},
			fields: fields{
				ID:        "a9394322-ac61-4bab-9fae-e33be4af709e",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "stoppedTestVM",
				Description: "a VM that is stopped",
				Status:      "STOPPED",
				BhyvePid:    71892,
				VNCPort:     6900,
				Config: Config{
					Model: gorm.Model{
						ID: 9191,
					},
					VMID: "a9394322-ac61-4bab-9fae-e33be4af709e",
					CPU:  2,
					Mem:  1024,
				},
			},
			args: args{
				nicIDs: []string{"c4b4dd48-f186-4cad-83c2-698df1900778"},
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// clear out list(s) from other parallel test runs
			disk.List.DiskList = map[string]*disk.Disk{}
			List.VMList = map[string]*VM{}

			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

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

			List.VMList[testVM.ID] = testVM

			testDB, mock := cirrinadtest.NewMockDB(testCase.name)
			testCase.mockClosure(testDB, mock)

			err := testVM.SetNics(testCase.args.nicIDs)
			if (err != nil) != testCase.wantErr {
				t.Errorf("SetNics() error = %v, wantErr %v", err, testCase.wantErr)
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

// test helpers from here down

//nolint:paralleltest
func Test_netStartupIfSuccess(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_netStartupIfFail(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_netStartupIfBridgeIfAddMemberErr(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	for _, v := range cmdWithArgs {
		if v == "addm" {
			os.Exit(1)
		}
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_netStartupIfBridgeIfAddMemberOk(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[1] == "/sbin/ifconfig" {
		os.Exit(0)
	}

	for _, v := range cmdWithArgs {
		if v == "addm" {
			os.Exit(0)
		}
	}

	fmt.Printf("args: %+v\n", os.Args) //nolint:forbidigo

	os.Exit(1)
}

//nolint:paralleltest
func Test_setupVMNicRateLimitFail(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_setupVMNicRateLimitOk(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "epair" {
		fmt.Printf("epair32767b\nepair32767a\n") //nolint:forbidigo
		os.Exit(0)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_setupVMNicRateLimitSetRateLimitFail(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		fmt.Printf("epair32767b\nepair32767a\n") //nolint:forbidigo
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_setupVMNicRateLimitCreateIfBridgeErr(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) >= 3 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" { //nolint:lll
		fmt.Printf("bridge0\n") //nolint:forbidigo
		os.Exit(0)
	}

	if len(cmdWithArgs) >= 2 && cmdWithArgs[1] == "/usr/sbin/ngctl" {
		os.Exit(0)
	}

	if len(cmdWithArgs) >= 6 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[2] == "create" && cmdWithArgs[3] == "group" && cmdWithArgs[4] == "cirrinad" && cmdWithArgs[5] == "up" { //nolint:lll
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_setupVMNicRateLimitSuccess(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		fmt.Printf("bridge0\n") //nolint:forbidigo
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" {
		os.Exit(0)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_netStartupIfSetupRateLimitOK(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestVM_NetCleanupSuccess(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestVM_NetCleanupFail(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestVM_SetNicsSuccess(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestVM_netStartupSuccess(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestVM_netStartupError(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestVM_netStartupSuccessIfConnectError(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	for _, v := range cmdWithArgs {
		if v == "addm" {
			os.Exit(1)
		}
	}

	os.Exit(0)
}
