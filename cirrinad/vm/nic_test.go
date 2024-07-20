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
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			// clear out list from other parallel test runs
			List.VMList = map[string]*VM{}

			testDB, mock := cirrinadtest.NewMockDB("nicTest")

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
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			// clear out list from other parallel test runs
			List.VMList = map[string]*VM{}

			testDB, mock := cirrinadtest.NewMockDB("nicTest")

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
			testDB, mock := cirrinadtest.NewMockDB("nicTest")

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

//nolint:paralleltest
func TestVM_netStartup(t *testing.T) {
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
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields      fields
		wantErr     bool
	}{
		{
			name: "getNicsErr",
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
			name: "noNics",
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
			name: "badType",
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
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			// clear out list from other parallel test runs
			List.VMList = map[string]*VM{}

			testDB, mock := cirrinadtest.NewMockDB("nicTest")

			testCase.mockClosure(testDB, mock)

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

			err := testVM.netStartup()
			if (err != nil) != testCase.wantErr {
				t.Errorf("netStartup() error = %v, wantErr %v", err, testCase.wantErr)
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
func Test_netStartupIf(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		vmNic vmnic.VMNic
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "ifconfigFail",
			mockCmdFunc: "Test_netStartupIfFail",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
			},
			args: args{
				vmNic: vmnic.VMNic{
					ID:          "7fada8be-fd0d-4525-a6ae-ba8a0c36b12c",
					Name:        "testNic01",
					Description: "yet another test nic",
					Mac:         "82:21:32:af:bc:aa",
					NetDev:      "AUTO",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "71204573-4d12-49e0-b4cf-42fb46da70e4",
				},
			},
			wantErr: true,
		},
		{
			name:        "noUplink",
			mockCmdFunc: "Test_netStartupIfSuccess",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				vmNic: vmnic.VMNic{
					ID:          "7fada8be-fd0d-4525-a6ae-ba8a0c36b12c",
					Name:        "testNic01",
					Description: "yet another test nic",
					Mac:         "82:21:32:af:bc:aa",
					NetDev:      "AUTO",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "",
				},
			},
			wantErr: false,
		},
		{
			name:        "getByIdErr",
			mockCmdFunc: "Test_netStartupIfSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmswitch.Instance = &vmswitch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("71204573-4d12-49e0-b4cf-42fb46da70e4").
					WillReturnError(gorm.ErrInvalidData)
			},
			args: args{
				vmNic: vmnic.VMNic{
					ID:          "7fada8be-fd0d-4525-a6ae-ba8a0c36b12c",
					Name:        "testNic01",
					Description: "yet another test nic",
					Mac:         "82:21:32:af:bc:aa",
					NetDev:      "AUTO",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "71204573-4d12-49e0-b4cf-42fb46da70e4",
				},
			},
			wantErr: true,
		},
		{
			name:        "badSwitchType",
			mockCmdFunc: "Test_netStartupIfSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmswitch.Instance = &vmswitch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("71204573-4d12-49e0-b4cf-42fb46da70e4").
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
								"71204573-4d12-49e0-b4cf-42fb46da70e4",
								createUpdateTime,
								createUpdateTime,
								nil,
								"bridge0",
								"some if switch description",
								"NG",
								"em1",
							),
					)
			},
			args: args{
				vmNic: vmnic.VMNic{
					ID:          "7fada8be-fd0d-4525-a6ae-ba8a0c36b12c",
					Name:        "testNic01",
					Description: "yet another test nic",
					Mac:         "82:21:32:af:bc:aa",
					NetDev:      "AUTO",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "71204573-4d12-49e0-b4cf-42fb46da70e4",
				},
			},
			wantErr: true,
		},
		{
			name:        "bridgeIfAddMemberErr",
			mockCmdFunc: "Test_netStartupIfBridgeIfAddMemberErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmswitch.Instance = &vmswitch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("71204573-4d12-49e0-b4cf-42fb46da70e4").
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
								"71204573-4d12-49e0-b4cf-42fb46da70e4",
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
			args: args{
				vmNic: vmnic.VMNic{
					ID:          "7fada8be-fd0d-4525-a6ae-ba8a0c36b12c",
					Name:        "testNic01",
					Description: "yet another test nic",
					Mac:         "82:21:32:af:bc:aa",
					NetDev:      "AUTO",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "71204573-4d12-49e0-b4cf-42fb46da70e4",
				},
			},
			wantErr: true,
		},
		{
			name:        "bridgeIfAddMemberOk",
			mockCmdFunc: "Test_netStartupIfBridgeIfAddMemberOk",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmswitch.Instance = &vmswitch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("71204573-4d12-49e0-b4cf-42fb46da70e4").
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
								"71204573-4d12-49e0-b4cf-42fb46da70e4",
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
			args: args{
				vmNic: vmnic.VMNic{
					ID:          "7fada8be-fd0d-4525-a6ae-ba8a0c36b12c",
					Name:        "testNic01",
					Description: "yet another test nic",
					Mac:         "82:21:32:af:bc:aa",
					NetDev:      "AUTO",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "71204573-4d12-49e0-b4cf-42fb46da70e4",
				},
			},
			wantErr: false,
		},
		{
			name:        "setupRateLimitErr",
			mockCmdFunc: "Test_netStartupIfBridgeIfAddMemberErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmswitch.Instance = &vmswitch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("71204573-4d12-49e0-b4cf-42fb46da70e4").
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
								"71204573-4d12-49e0-b4cf-42fb46da70e4",
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
			args: args{
				vmNic: vmnic.VMNic{
					ID:          "7fada8be-fd0d-4525-a6ae-ba8a0c36b12c",
					Name:        "testNic01",
					Description: "yet another test nic",
					Mac:         "82:21:32:af:bc:aa",
					NetDev:      "AUTO",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "71204573-4d12-49e0-b4cf-42fb46da70e4",
					RateLimit:   true,
				},
			},
			wantErr: true,
		},
		{
			name:        "setupRateLimitOK",
			mockCmdFunc: "Test_netStartupIfSetupRateLimitOK",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmswitch.Instance = &vmswitch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("71204573-4d12-49e0-b4cf-42fb46da70e4").
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
								"71204573-4d12-49e0-b4cf-42fb46da70e4",
								createUpdateTime,
								createUpdateTime,
								nil,
								"bridge0",
								"some if switch description",
								"IF",
								"em1",
							),
					)

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(9912, "yet another test nic", "", "epair32767", "82:21:32:af:bc:aa", "testNic01", "tap0", "TAP",
						"VIRTIONET", 400000000, true, 100000000, "71204573-4d12-49e0-b4cf-42fb46da70e4", sqlmock.AnyArg(),
						"7fada8be-fd0d-4525-a6ae-ba8a0c36b12c").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(9912, "yet another test nic", "bridge32767", "epair32767", "82:21:32:af:bc:aa", "testNic01", "tap0", "TAP", //nolint:lll
						"VIRTIONET", 400000000, true, 100000000, "71204573-4d12-49e0-b4cf-42fb46da70e4", sqlmock.AnyArg(),
						"7fada8be-fd0d-4525-a6ae-ba8a0c36b12c").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args: args{
				vmNic: vmnic.VMNic{
					ID:          "7fada8be-fd0d-4525-a6ae-ba8a0c36b12c",
					Name:        "testNic01",
					Description: "yet another test nic",
					Mac:         "82:21:32:af:bc:aa",
					NetDev:      "tap0",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "71204573-4d12-49e0-b4cf-42fb46da70e4",
					RateLimit:   true,
					RateIn:      400000000,
					RateOut:     100000000,
					ConfigID:    9912,
				},
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("switchTest")
			testCase.mockClosure(testDB, mock)

			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)

			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := netStartupIf(testCase.args.vmNic)
			if (err != nil) != testCase.wantErr {
				t.Errorf("netStartupIf() error = %v, wantErr %v", err, testCase.wantErr)
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
func Test_setupVMNicRateLimit(t *testing.T) {
	type args struct {
		vmNic vmnic.VMNic
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		mockCmdFunc string
		args        args
		want        string
		wantErr     bool
	}{
		{
			name: "createEpairErr",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
			},
			mockCmdFunc: "Test_setupVMNicRateLimitFail",
			args: args{
				vmNic: vmnic.VMNic{
					ID:          "",
					Name:        "someNic",
					Description: "a NIC",
					Mac:         "",
					NetDev:      "",
					NetType:     "",
					NetDevType:  "",
					SwitchID:    "",
					RateLimit:   false,
					RateIn:      0,
					RateOut:     0,
					InstBridge:  "",
					InstEpair:   "",
					ConfigID:    0,
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "saveErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(9912, "a NIC", "bridge0", "epair32766", "00:22:44:aa:bb:cc", "someNic", "", "TAP",
						"VIRTIONET", 400000000, true, 100000000, "81184199-c672-4641-b6a7-75ad01c48059", sqlmock.AnyArg(),
										"8a99b08f-1105-4f81-ac87-48edd69bc058").
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
				mock.ExpectRollback()
			},
			mockCmdFunc: "Test_setupVMNicRateLimitOk",
			args: args{
				vmNic: vmnic.VMNic{
					ID:          "8a99b08f-1105-4f81-ac87-48edd69bc058",
					Name:        "someNic",
					Description: "a NIC",
					Mac:         "00:22:44:aa:bb:cc",
					NetDev:      "",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "81184199-c672-4641-b6a7-75ad01c48059",
					RateLimit:   true,
					RateIn:      400000000,
					RateOut:     100000000,
					InstBridge:  "bridge0",
					InstEpair:   "",
					ConfigID:    9912,
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "setRateLimitFail",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(9912, "a NIC", "bridge0", "epair32767", "00:22:44:aa:bb:cc", "someNic", "", "TAP",
						"VIRTIONET", 400000000, true, 100000000, "81184199-c672-4641-b6a7-75ad01c48059", sqlmock.AnyArg(),
						"8a99b08f-1105-4f81-ac87-48edd69bc058").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},

			mockCmdFunc: "Test_setupVMNicRateLimitSetRateLimitFail",
			args: args{
				vmNic: vmnic.VMNic{
					ID:          "8a99b08f-1105-4f81-ac87-48edd69bc058",
					Name:        "someNic",
					Description: "a NIC",
					Mac:         "00:22:44:aa:bb:cc",
					NetDev:      "",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "81184199-c672-4641-b6a7-75ad01c48059",
					RateLimit:   true,
					RateIn:      400000000,
					RateOut:     100000000,
					InstBridge:  "bridge0",
					InstEpair:   "",
					ConfigID:    9912,
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "createIfBridgeErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(9912, "a NIC", "", "epair32767", "00:22:44:aa:bb:cc", "someNic", "tap0", "TAP",
						"VIRTIONET", 400000000, true, 100000000, "81184199-c672-4641-b6a7-75ad01c48059", sqlmock.AnyArg(),
						"8a99b08f-1105-4f81-ac87-48edd69bc058").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			mockCmdFunc: "Test_setupVMNicRateLimitCreateIfBridgeErr",
			args: args{
				vmNic: vmnic.VMNic{
					ID:          "8a99b08f-1105-4f81-ac87-48edd69bc058",
					Name:        "someNic",
					Description: "a NIC",
					Mac:         "00:22:44:aa:bb:cc",
					NetDev:      "tap0",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "81184199-c672-4641-b6a7-75ad01c48059",
					RateLimit:   true,
					RateIn:      400000000,
					RateOut:     100000000,
					InstBridge:  "",
					InstEpair:   "",
					ConfigID:    9912,
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "saveErr2",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(9912, "a NIC", "", "epair32767", "00:22:44:aa:bb:cc", "someNic", "tap0", "TAP",
						"VIRTIONET", 400000000, true, 100000000, "81184199-c672-4641-b6a7-75ad01c48059", sqlmock.AnyArg(),
						"8a99b08f-1105-4f81-ac87-48edd69bc058").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(9912, "a NIC", "bridge32767", "epair32767", "00:22:44:aa:bb:cc", "someNic", "tap0", "TAP",
						"VIRTIONET", 400000000, true, 100000000, "81184199-c672-4641-b6a7-75ad01c48059", sqlmock.AnyArg(),
										"8a99b08f-1105-4f81-ac87-48edd69bc058").
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
				mock.ExpectRollback()
			},
			mockCmdFunc: "Test_setupVMNicRateLimitSuccess",
			args: args{
				vmNic: vmnic.VMNic{
					ID:          "8a99b08f-1105-4f81-ac87-48edd69bc058",
					Name:        "someNic",
					Description: "a NIC",
					Mac:         "00:22:44:aa:bb:cc",
					NetDev:      "tap0",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "81184199-c672-4641-b6a7-75ad01c48059",
					RateLimit:   true,
					RateIn:      400000000,
					RateOut:     100000000,
					InstBridge:  "",
					InstEpair:   "",
					ConfigID:    9912,
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(9912, "a NIC", "", "epair32767", "00:22:44:aa:bb:cc", "someNic", "tap0", "TAP",
						"VIRTIONET", 400000000, true, 100000000, "81184199-c672-4641-b6a7-75ad01c48059", sqlmock.AnyArg(),
						"8a99b08f-1105-4f81-ac87-48edd69bc058").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(9912, "a NIC", "bridge32767", "epair32767", "00:22:44:aa:bb:cc", "someNic", "tap0", "TAP",
						"VIRTIONET", 400000000, true, 100000000, "81184199-c672-4641-b6a7-75ad01c48059", sqlmock.AnyArg(),
						"8a99b08f-1105-4f81-ac87-48edd69bc058").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			mockCmdFunc: "Test_setupVMNicRateLimitSuccess",
			args: args{
				vmNic: vmnic.VMNic{
					ID:          "8a99b08f-1105-4f81-ac87-48edd69bc058",
					Name:        "someNic",
					Description: "a NIC",
					Mac:         "00:22:44:aa:bb:cc",
					NetDev:      "tap0",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "81184199-c672-4641-b6a7-75ad01c48059",
					RateLimit:   true,
					RateIn:      400000000,
					RateOut:     100000000,
					InstBridge:  "",
					InstEpair:   "",
					ConfigID:    9912,
				},
			},
			want:    "epair32767",
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			testDB, mock := cirrinadtest.NewMockDB("switchTest")
			testCase.mockClosure(testDB, mock)

			got, err := setupVMNicRateLimit(testCase.args.vmNic)
			if (err != nil) != testCase.wantErr {
				t.Errorf("setupVMNicRateLimit() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("setupVMNicRateLimit() got = %v, want %v", got, testCase.want)
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
func Test_netStartupNg(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		vmNic vmnic.VMNic
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		wantErr     bool
	}{
		{
			name: "getByIdErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmswitch.Instance = &vmswitch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("503c83af-221c-4e4a-8cf0-21d560dbfd73").
					WillReturnError(gorm.ErrInvalidData)
			},
			args: args{
				vmNic: vmnic.VMNic{
					ID:          "d75953b6-e891-42de-b85e-f053cf960dbe",
					Name:        "testNic01",
					Description: "yet another test nic",
					Mac:         "82:21:32:af:bc:aa",
					NetDev:      "AUTO",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "503c83af-221c-4e4a-8cf0-21d560dbfd73",
				},
			},
			wantErr: true,
		},
		{
			name: "badSwitchType",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmswitch.Instance = &vmswitch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("d21f5367-8f5c-4a4c-95ce-75cb37c6c449").
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
								"d21f5367-8f5c-4a4c-95ce-75cb37c6c449",
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
			args: args{
				vmNic: vmnic.VMNic{
					ID:          "2d247fe1-bfb6-43df-994c-4f879a2e3ff5",
					Name:        "testNic01",
					Description: "yet another test nic",
					Mac:         "82:21:32:af:bc:aa",
					NetDev:      "AUTO",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "d21f5367-8f5c-4a4c-95ce-75cb37c6c449",
				},
			},
			wantErr: true,
		},
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmswitch.Instance = &vmswitch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("d21f5367-8f5c-4a4c-95ce-75cb37c6c449").
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
								"d21f5367-8f5c-4a4c-95ce-75cb37c6c449",
								createUpdateTime,
								createUpdateTime,
								nil,
								"bridge0",
								"some if switch description",
								"NG",
								"em1",
							),
					)
			},
			args: args{
				vmNic: vmnic.VMNic{
					ID:          "2d247fe1-bfb6-43df-994c-4f879a2e3ff5",
					Name:        "testNic01",
					Description: "yet another test nic",
					Mac:         "82:21:32:af:bc:aa",
					NetDev:      "AUTO",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "d21f5367-8f5c-4a4c-95ce-75cb37c6c449",
				},
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("switchTest")
			testCase.mockClosure(testDB, mock)

			err := netStartupNg(testCase.args.vmNic)
			if (err != nil) != testCase.wantErr {
				t.Errorf("netStartupNg() error = %v, wantErr %v", err, testCase.wantErr)
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

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		fmt.Printf("bridge0\n") //nolint:forbidigo
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/usr/sbin/ngctl" {
		os.Exit(0)
	}

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[2] == "create" && cmdWithArgs[3] == "group" && cmdWithArgs[4] == "cirrinad" && cmdWithArgs[5] == "up" { //nolint:lll
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
