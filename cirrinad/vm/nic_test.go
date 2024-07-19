package vm

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/iso"
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
