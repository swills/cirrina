package vmswitch

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-test/deep"
	"gorm.io/gorm"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vmnic"
)

func TestGetAll(t *testing.T) {
	createUpdateTime := time.Now()

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        []*Switch
	}{
		{
			name: "testGetAllSwitches",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE `switches`.`deleted_at` IS NULL"),
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
								"uplink",
							}).
							AddRow(
								"0cb98661-6470-432d-8fa4-5eca3668b494",
								createUpdateTime,
								createUpdateTime,
								nil,
								"bridge0",
								"some if switch description",
								"IF",
								"em1",
							).
							AddRow(
								"76290cc3-7143-4c0b-980f-25f74b12673f",
								createUpdateTime,
								createUpdateTime,
								nil,
								"bnet0",
								"some ng switch description",
								"NG",
								"em0",
							),
					)
			},
			want: []*Switch{
				{
					Model: gorm.Model{
						ID:        0,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
						DeletedAt: gorm.DeletedAt{},
					},
					ID:          "0cb98661-6470-432d-8fa4-5eca3668b494",
					Name:        "bridge0",
					Description: "some if switch description",
					Type:        "IF",
					Uplink:      "em1",
				},
				{
					Model: gorm.Model{
						ID:        0,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
						DeletedAt: gorm.DeletedAt{},
					},
					ID:          "76290cc3-7143-4c0b-980f-25f74b12673f",
					Name:        "bnet0",
					Description: "some ng switch description",
					Type:        "NG",
					Uplink:      "em0",
				},
			},
		},
	}
	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("switchTest")
			testCase.mockClosure(testDB, mock)

			got := GetAll()

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

func TestGetByName(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		name string
	}

	tests := []struct {
		name        string
		args        args
		want        *Switch
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		wantErr     bool
	}{
		{
			name: "testGetByName_bridge0",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE name = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
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
								"uplink",
							}).
							AddRow(
								"0cb98661-6470-432d-8fa4-5eca3668b494",
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
			args: args{name: "bridge0"},
			want: &Switch{
				Model: gorm.Model{
					ID:        0,
					CreatedAt: createUpdateTime,
					UpdatedAt: createUpdateTime,
					DeletedAt: gorm.DeletedAt{},
				},
				ID:          "0cb98661-6470-432d-8fa4-5eca3668b494",
				Name:        "bridge0",
				Description: "some if switch description",
				Type:        "IF",
				Uplink:      "em1",
			},
		},
		{
			name: "testGetByName_error",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE name = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			args:    args{name: "bridge0"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "testGetByName_notfound",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE name = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
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
								"uplink",
							}),
					)
			},
			args:    args{name: "bridge0"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "testGetByName_emptyName",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
			},
			args:    args{name: ""},
			want:    nil,
			wantErr: true,
		},
	}
	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("switchTest")
			testCase.mockClosure(testDB, mock)

			got, err := GetByName(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetByName() error = %v, wantErr %v", err, testCase.wantErr)

				return
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

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestGetByID(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		switchID string
	}

	tests := []struct {
		name        string
		args        args
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        *Switch
		wantErr     bool
	}{
		{
			name: "testGetByID_success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
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
								"uplink",
							},
						).
							AddRow(
								"0cb98661-6470-432d-8fa4-5eca3668b494",
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
			args: args{switchID: "0cb98661-6470-432d-8fa4-5eca3668b494"},
			want: &Switch{
				Model: gorm.Model{
					ID:        0,
					CreatedAt: createUpdateTime,
					UpdatedAt: createUpdateTime,
					DeletedAt: gorm.DeletedAt{},
				},
				ID:          "0cb98661-6470-432d-8fa4-5eca3668b494",
				Name:        "bridge0",
				Description: "some if switch description",
				Type:        "IF",
				Uplink:      "em1",
			},
		},
		{
			name: "testGetByID_error",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			args:    args{switchID: "0cb98661-6470-432d-8fa4-5eca3668b494"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "testGetByID_notfound",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
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
								"uplink",
							}),
					)
			},
			args:    args{switchID: "713e2714-eb92-4b53-b129-9d1f914eaa06"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("switchTest")
			testCase.mockClosure(testDB, mock)

			got, err := GetByID(testCase.args.switchID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, testCase.wantErr)

				return
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

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func Test_switchNameValid(t *testing.T) {
	type args struct {
		switchInst *Switch
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "empty",
			args: args{switchInst: &Switch{Name: ""}},
			want: false,
		},
		{
			name: "goodIFBridge",
			args: args{switchInst: &Switch{Name: "bridge0", Type: "IF"}},
			want: true,
		},
		{
			name: "goodNGBridge",
			args: args{switchInst: &Switch{Name: "bnet0", Type: "NG"}},
			want: true,
		},
		{
			name: "badIFBridge",
			args: args{switchInst: &Switch{Name: "bnet0", Type: "IF"}},
			want: false,
		},
		{
			name: "badNGBridge",
			args: args{switchInst: &Switch{Name: "bridge0", Type: "NG"}},
			want: false,
		},
		{
			name: "sillyIFBridge",
			args: args{switchInst: &Switch{Name: "bridge01", Type: "IF"}},
			want: false,
		},
		{
			name: "sillyNGBridge",
			args: args{switchInst: &Switch{Name: "bnet01", Type: "NG"}},
			want: false,
		},
		{
			name: "unicodeBridgeNameIF",
			args: args{switchInst: &Switch{Name: "☃︎︎", Type: "IF"}},
			want: false,
		},
		{
			name: "unicodeBridgeNameNG",
			args: args{switchInst: &Switch{Name: "☃︎︎", Type: "NG"}},
			want: false,
		},
		{
			name: "badNumIF",
			args: args{switchInst: &Switch{Name: "bridge0abc", Type: "IF"}},
			want: false,
		},
		{
			name: "badNumNG",
			args: args{switchInst: &Switch{Name: "bnet0abc", Type: "NG"}},
			want: false,
		},
		{
			name: "badTypeTest",
			args: args{switchInst: &Switch{Name: "bridge0", Type: "blah"}},
			want: false,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := switchNameValid(testCase.args.switchInst); got != testCase.want {
				t.Errorf("switchNameValid() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestParseSwitchID(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		switchID   string
		netDevType string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        string
		wantErr     bool
	}{
		{
			name: "success1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
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
								"uplink",
							},
						).
							AddRow(
								"90b2b502-13c9-4132-a0c5-3bbb54a4b443",
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
			args: args{switchID: "90b2b502-13c9-4132-a0c5-3bbb54a4b443", netDevType: "TAP"},
			want: "90b2b502-13c9-4132-a0c5-3bbb54a4b443",
		},
		{
			name: "errorEmptySwitchID",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
			},
			args:    args{switchID: "", netDevType: "TAP"},
			want:    "",
			wantErr: true,
		},
		{
			name: "errBadSwitchID",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
			},
			args:    args{switchID: "bogusSwitchId", netDevType: "TAP"},
			want:    "",
			wantErr: true,
		},
		{
			name: "errorGettingSwitchID",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			args:    args{switchID: "90b2b502-13c9-4132-a0c5-3bbb54a4b443", netDevType: "TAP"},
			want:    "",
			wantErr: true,
		},
		{
			name: "errReturnedEmptySwitchName",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
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
								"uplink",
							},
						).
							AddRow(
								"90b2b502-13c9-4132-a0c5-3bbb54a4b443",
								createUpdateTime,
								createUpdateTime,
								nil,
								"",
								"some if switch description",
								"IF",
								"em1",
							),
					)
			},
			args:    args{switchID: "90b2b502-13c9-4132-a0c5-3bbb54a4b443", netDevType: "TAP"},
			want:    "",
			wantErr: true,
		},
		{
			name: "errorSwitchTypeMismatchIF",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
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
								"uplink",
							},
						).
							AddRow(
								"90b2b502-13c9-4132-a0c5-3bbb54a4b443",
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
			args:    args{switchID: "90b2b502-13c9-4132-a0c5-3bbb54a4b443", netDevType: "TAP"},
			want:    "",
			wantErr: true,
		},
		{
			name: "errorSwitchTypeMismatchNG",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
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
								"uplink",
							},
						).
							AddRow(
								"90b2b502-13c9-4132-a0c5-3bbb54a4b443",
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
			args:    args{switchID: "90b2b502-13c9-4132-a0c5-3bbb54a4b443", netDevType: "NETGRAPH"},
			want:    "",
			wantErr: true,
		},
		{
			name: "errorSwitchTypeUnknown",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
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
								"uplink",
							},
						).
							AddRow(
								"90b2b502-13c9-4132-a0c5-3bbb54a4b443",
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
			args:    args{switchID: "90b2b502-13c9-4132-a0c5-3bbb54a4b443", netDevType: "garbage"},
			want:    "",
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("switchTest")
			testCase.mockClosure(testDB, mock)

			got, err := ParseSwitchID(testCase.args.switchID, testCase.args.netDevType)
			if (err != nil) != testCase.wantErr {
				t.Errorf("ParseSwitchID() error = %v, wantErr %v", err, testCase.wantErr)

				return
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

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func Test_bringUpNewSwitch(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		switchInst *Switch
	}

	tests := []struct {
		name            string
		mockCmdFunc     string
		hostIntStubFunc func() ([]net.Interface, error)
		args            args
		wantErr         bool
	}{
		{
			name:            "successIfNoUplink",
			hostIntStubFunc: StubBringUpNewSwitchHostInterfacesSuccess1,
			mockCmdFunc:     "Test_bringUpNewSwitchSuccess1",
			args: args{switchInst: &Switch{
				Model: gorm.Model{
					ID:        0,
					CreatedAt: createUpdateTime,
					UpdatedAt: createUpdateTime,
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
				},
				ID:          "4f5f7bad-0718-492f-af75-d6f4c179b6c1",
				Name:        "bridge0",
				Description: "some description",
				Type:        "IF",
				Uplink:      "",
			}},
		},
		{
			name:            "successNGNoUplink",
			hostIntStubFunc: StubBringUpNewSwitchHostInterfacesSuccess1,
			mockCmdFunc:     "Test_bringUpNewSwitchSuccess1",
			args: args{switchInst: &Switch{
				Model: gorm.Model{
					ID:        0,
					CreatedAt: createUpdateTime,
					UpdatedAt: createUpdateTime,
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
				},
				ID:          "4f5f7bad-0718-492f-af75-d6f4c179b6c1",
				Name:        "bnet0",
				Description: "some description",
				Type:        "NG",
				Uplink:      "",
			}},
		},
		{
			name:            "errInvalidSwitchType",
			hostIntStubFunc: StubBringUpNewSwitchHostInterfacesSuccess1,
			mockCmdFunc:     "Test_bringUpNewSwitchSuccess1",
			args: args{switchInst: &Switch{
				Model: gorm.Model{
					ID:        0,
					CreatedAt: createUpdateTime,
					UpdatedAt: createUpdateTime,
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
				},
				ID:          "4f5f7bad-0718-492f-af75-d6f4c179b6c1",
				Name:        "bridge0",
				Description: "some description",
				Type:        "garbage",
				Uplink:      "",
			}},
			wantErr: true,
		},
		{
			name:            "successIFWithUplink",
			hostIntStubFunc: StubBringUpNewSwitchHostInterfacesSuccess1,
			mockCmdFunc:     "Test_bringUpNewSwitchSuccess1",
			args: args{switchInst: &Switch{
				Model: gorm.Model{
					ID:        0,
					CreatedAt: createUpdateTime,
					UpdatedAt: createUpdateTime,
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
				},
				ID:          "4f5f7bad-0718-492f-af75-d6f4c179b6c1",
				Name:        "bridge0",
				Description: "some description",
				Type:        "IF",
				Uplink:      "em0",
			}},
		},
		{
			name:            "successNGWithUplink",
			hostIntStubFunc: StubBringUpNewSwitchHostInterfacesSuccess1,
			mockCmdFunc:     "Test_bringUpNewSwitchSuccess1",
			args: args{switchInst: &Switch{
				Model: gorm.Model{
					ID:        0,
					CreatedAt: createUpdateTime,
					UpdatedAt: createUpdateTime,
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
				},
				ID:          "4f5f7bad-0718-492f-af75-d6f4c179b6c1",
				Name:        "bnet0",
				Description: "some description",
				Type:        "NG",
				Uplink:      "em0",
			}},
		},
		{
			name:            "errSwitchNil",
			hostIntStubFunc: StubBringUpNewSwitchHostInterfacesSuccess1,
			mockCmdFunc:     "Test_bringUpNewSwitchSuccess1",
			args:            args{switchInst: nil},
			wantErr:         true,
		},
		{
			name:            "errSwitchIDEmpty",
			hostIntStubFunc: StubBringUpNewSwitchHostInterfacesSuccess1,
			mockCmdFunc:     "Test_bringUpNewSwitchSuccess1",
			args:            args{switchInst: &Switch{ID: ""}},
			wantErr:         true,
		},
		{
			name:            "errBuildIF",
			hostIntStubFunc: StubBringUpNewSwitchHostInterfacesSuccess1,
			mockCmdFunc:     "Test_bringUpNewSwitchSuccess1",
			args: args{switchInst: &Switch{
				Model: gorm.Model{
					ID:        0,
					CreatedAt: createUpdateTime,
					UpdatedAt: createUpdateTime,
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
				},
				ID:          "4f5f7bad-0718-492f-af75-d6f4c179b6c1",
				Name:        "bnet0",
				Description: "some description",
				Type:        "IF",
				Uplink:      "em0",
			}},
		},
		{
			name:            "errBuildNG",
			hostIntStubFunc: StubBringUpNewSwitchHostInterfacesSuccess1,
			mockCmdFunc:     "Test_bringUpNewSwitchSuccess1",
			args: args{switchInst: &Switch{
				Model: gorm.Model{
					ID:        0,
					CreatedAt: createUpdateTime,
					UpdatedAt: createUpdateTime,
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
				},
				ID:          "4f5f7bad-0718-492f-af75-d6f4c179b6c1",
				Name:        "bridge0",
				Description: "some description",
				Type:        "NG",
				Uplink:      "em0",
			}},
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			util.NetInterfacesFunc = testCase.hostIntStubFunc

			t.Cleanup(func() { util.NetInterfacesFunc = net.Interfaces })

			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)

			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := bringUpNewSwitch(testCase.args.switchInst)
			if (err != nil) != testCase.wantErr {
				t.Errorf("bringUpNewSwitch() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func Test_switchTypeValid(t *testing.T) {
	type args struct {
		switchInst *Switch
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "okIFBridge",
			args: args{
				switchInst: &Switch{
					Type: "IF",
				},
			},
			want: true,
		},
		{
			name: "okNGBridge",
			args: args{
				switchInst: &Switch{
					Type: "NG",
				},
			},
			want: true,
		},
		{
			name: "badGarbage",
			args: args{
				switchInst: &Switch{
					Type: "garbage",
				},
			},
			want: false,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			got := switchTypeValid(testCase.args.switchInst)
			if got != testCase.want {
				t.Errorf("switchTypeValid() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func Test_memberUsedByIfBridge(t *testing.T) {
	type args struct {
		member string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		want        bool
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_memberUsedByIfBridgeSuccess1",
			args:        args{member: "em0"},
			want:        false,
			wantErr:     false,
		},
		{
			name:        "success2",
			mockCmdFunc: "Test_memberUsedByIfBridgeSuccess2",
			args:        args{member: "em0"},
			want:        false,
			wantErr:     false,
		},
		{
			name:        "success3",
			mockCmdFunc: "Test_memberUsedByIfBridgeSuccess3",
			args:        args{member: "em0"},
			want:        true,
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_memberUsedByIfBridgeError1",
			args:        args{member: "em0"},
			want:        true,
			wantErr:     true,
		},
		{
			name:        "error2",
			mockCmdFunc: "Test_memberUsedByIfBridgeError2",
			args:        args{member: "em0"},
			want:        true,
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			got, err := memberUsedByIfBridge(testCase.args.member)
			if (err != nil) != testCase.wantErr {
				t.Errorf("memberUsedByIfBridge() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("memberUsedByIfBridge() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func Test_memberUsedByNgBridge(t *testing.T) {
	type args struct {
		member string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		want        bool
		wantErr     bool
	}{
		{
			name:        "success1",
			args:        args{member: "em0"},
			mockCmdFunc: "Test_memberUsedByNgBridgeSuccess1",
			want:        true,
			wantErr:     false,
		},
		{
			name:        "error1",
			args:        args{member: "em0"},
			mockCmdFunc: "Test_memberUsedByNgBridgeError1",
			want:        false,
			wantErr:     true,
		},
		{
			name:        "error2",
			args:        args{member: "em0"},
			mockCmdFunc: "Test_memberUsedByNgBridgeError2",
			want:        false,
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			got, err := memberUsedByNgBridge(testCase.args.member)
			if (err != nil) != testCase.wantErr {
				t.Errorf("memberUsedByNgBridge() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("memberUsedByNgBridge() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func Test_ngGetBridgeNextLink(t *testing.T) {
	type args struct {
		bridge string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		want        string
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_ngGetBridgeNextLinkSuccess1",
			args:        args{bridge: "bnet0"},
			want:        "link2",
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_ngGetBridgeNextLinkError1",
			args:        args{bridge: "bnet0"},
			want:        "",
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			got, err := ngGetBridgeNextLink(testCase.args.bridge)
			if (err != nil) != testCase.wantErr {
				t.Errorf("ngGetBridgeNextLink() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("ngGetBridgeNextLink() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func Test_validateIfSwitch(t *testing.T) {
	type args struct {
		switchInst *Switch
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_validateIfSwitchSuccess1",
			args: args{switchInst: &Switch{
				Model:  gorm.Model{},
				Name:   "bridge1",
				Uplink: "em1",
			}},
			wantErr: false,
		},
		{
			name:        "success2",
			mockCmdFunc: "Test_validateIfSwitchSuccess1",
			args: args{switchInst: &Switch{
				Model:  gorm.Model{},
				Name:   "bridge1",
				Uplink: "em0",
			}},
			wantErr: true,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_validateIfSwitchError1",
			args: args{switchInst: &Switch{
				Model:  gorm.Model{},
				Name:   "bridge1",
				Uplink: "em0",
			}},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := validateIfSwitch(testCase.args.switchInst)
			if (err != nil) != testCase.wantErr {
				t.Errorf("validateIfSwitch() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func Test_validateNgSwitch(t *testing.T) {
	type args struct {
		switchInst *Switch
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_validateNgSwitchSuccess1",
			args: args{switchInst: &Switch{
				Model:  gorm.Model{},
				Name:   "bnet1",
				Uplink: "em1",
			}},
		},
		{
			name:        "success2",
			mockCmdFunc: "Test_validateNgSwitchSuccess1",
			args: args{switchInst: &Switch{
				Model:  gorm.Model{},
				Name:   "bnet1",
				Uplink: "em0",
			}},
			wantErr: true,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_validateNgSwitchError1",
			args: args{switchInst: &Switch{
				Model:  gorm.Model{},
				Name:   "bnet1",
				Uplink: "em0",
			}},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := validateNgSwitch(testCase.args.switchInst)
			if (err != nil) != testCase.wantErr {
				t.Errorf("validateNgSwitch() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestDestroyNgBridge(t *testing.T) {
	type args struct {
		netDev string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestDestroyNgBridgeSuccess1",
			args:        args{netDev: "bnet0"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "TestDestroyNgBridgeError1",
			args:        args{netDev: ""},
			wantErr:     true,
		},
		{
			name:        "error2",
			mockCmdFunc: "TestDestroyNgBridgeError1",
			args:        args{netDev: "bnet0"},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			if err := DestroyNgBridge(testCase.args.netDev); (err != nil) != testCase.wantErr {
				t.Errorf("DestroyNgBridge() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestDestroyIfBridge(t *testing.T) {
	type args struct {
		name    string
		cleanup bool
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestDestroyIfBridgeSuccess1",
			args:        args{name: "bridge0", cleanup: false},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "TestDestroyIfBridgeSuccess1",
			args:        args{name: "garbage", cleanup: false},
			wantErr:     true,
		},
		{
			name:        "error2",
			mockCmdFunc: "TestDestroyIfBridgeError2",
			args:        args{name: "bridge0", cleanup: false},
			wantErr:     true,
		},
		{
			name:        "error3",
			mockCmdFunc: "TestDestroyIfBridgeError2",
			args:        args{name: "bridge0", cleanup: true},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			if err := DestroyIfBridge(testCase.args.name, testCase.args.cleanup); (err != nil) != testCase.wantErr {
				t.Errorf("DestroyIfBridge() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestBridgeIfAddMember(t *testing.T) {
	type args struct {
		bridgeName string
		memberName string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestBridgeIfAddMemberSuccess1",
			args:        args{bridgeName: "bridge0", memberName: "tap0"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "TestBridgeIfAddMemberError1",
			args:        args{bridgeName: "bridge0", memberName: "tap0"},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := BridgeIfAddMember(testCase.args.bridgeName, testCase.args.memberName)
			if (err != nil) != testCase.wantErr {
				t.Errorf("BridgeIfAddMember() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestBridgeNgAddMember(t *testing.T) {
	type args struct {
		bridgeName string
		memberName string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestBridgeNgAddMemberSuccess1",
			args:        args{bridgeName: "bnet0", memberName: "tap0"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "TestBridgeNgAddMemberError1",
			args:        args{bridgeName: "bnet0", memberName: "tap0"},
			wantErr:     true,
		},
		{
			name:        "error2",
			mockCmdFunc: "TestBridgeNgAddMemberError2",
			args:        args{bridgeName: "bnet0", memberName: "tap0"},
			wantErr:     true,
		},
		{
			name:        "error3",
			mockCmdFunc: "TestBridgeNgAddMemberError3",
			args:        args{bridgeName: "bnet0", memberName: "tap0"},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := BridgeNgAddMember(testCase.args.bridgeName, testCase.args.memberName)
			if (err != nil) != testCase.wantErr {
				t.Errorf("BridgeNgAddMember() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestCheckSwitchInUse(t *testing.T) {
	type args struct {
		id string
	}

	tests := []struct {
		name        string
		mockClosure func() []*vmnic.VMNic
		args        args
		wantErr     bool
	}{
		{
			name: "success1",
			mockClosure: func() []*vmnic.VMNic {
				return []*vmnic.VMNic{{
					SwitchID: "14152233-f90c-49e2-b53e-89d1f8b5ac2b",
				}}
			},
			args:    args{id: "56df0e88-9edd-4536-af80-6b53537f1708"},
			wantErr: false,
		},
		{
			name: "error1",
			mockClosure: func() []*vmnic.VMNic {
				return []*vmnic.VMNic{{
					SwitchID: "56df0e88-9edd-4536-af80-6b53537f1708",
				}}
			},
			args:    args{id: "56df0e88-9edd-4536-af80-6b53537f1708"},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			vmnicGetAllFunc = testCase.mockClosure

			t.Cleanup(func() { vmnicGetAllFunc = vmnic.GetAll })

			err := CheckSwitchInUse(testCase.args.id)
			if (err != nil) != testCase.wantErr {
				t.Errorf("CheckSwitchInUse() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func Test_switchExists(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		switchName string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        bool
		wantErr     bool
	}{
		{
			name: "success1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE name = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
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
								"uplink",
							}).
							AddRow(
								"0cb98661-6470-432d-8fa4-5eca3668b494",
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
			args:    args{switchName: "bridge0"},
			want:    true,
			wantErr: false,
		},
		{
			name: "error1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE name = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
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
								"uplink",
							},
						),
					)
			},
			args:    args{switchName: "bridge0"},
			want:    false,
			wantErr: false,
		},
		{
			name: "error2",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE name = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			args:    args{switchName: "bridge0"},
			want:    false,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("switchTest")
			testCase.mockClosure(testDB, mock)

			got, err := switchExists(testCase.args.switchName)
			if (err != nil) != testCase.wantErr {
				t.Errorf("switchExists() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("switchExists() got = %v, want %v", got, testCase.want)
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

func TestSwitch_Save(t *testing.T) {
	type switchFields struct {
		Model       gorm.Model
		ID          string
		Name        string
		Description string
		Type        string
		Uplink      string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		testSwitch  switchFields
		wantErr     bool
	}{
		{
			name: "success1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `switches` SET `description`=?,`name`=?,`type`=?,`uplink`=?,`updated_at`=? WHERE `switches`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("a simple test bridge", "bridge0", "IF", "em0", sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			testSwitch: switchFields{
				ID:          "f219ec59-cda7-4c7c-a57b-84ca3f063c39",
				Name:        "bridge0",
				Description: "a simple test bridge",
				Type:        "IF",
				Uplink:      "em0",
			},
			wantErr: false,
		},
		{
			name: "error1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `switches` SET `description`=?,`name`=?,`type`=?,`uplink`=?,`updated_at`=? WHERE `switches`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("a simple test bridge", "bridge0", "IF", "em0", sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
				mock.ExpectRollback()
			},
			testSwitch: switchFields{
				ID:          "f219ec59-cda7-4c7c-a57b-84ca3f063c39",
				Name:        "bridge0",
				Description: "a simple test bridge",
				Type:        "IF",
				Uplink:      "em0",
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("switchTest")
			testCase.mockClosure(testDB, mock)

			testSwitch := &Switch{
				ID:          testCase.testSwitch.ID,
				Name:        testCase.testSwitch.Name,
				Description: testCase.testSwitch.Description,
				Type:        testCase.testSwitch.Type,
				Uplink:      testCase.testSwitch.Uplink,
			}

			err := testSwitch.Save()
			if (err != nil) != testCase.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, testCase.wantErr)
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

func TestDelete(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		switchID string
	}

	tests := []struct {
		name                string
		mockClosure         func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		mockVmnicGetAllFunc func() []*vmnic.VMNic
		args                args
		wantErr             bool
	}{
		{
			name: "success1",
			mockVmnicGetAllFunc: func() []*vmnic.VMNic {
				return []*vmnic.VMNic{{
					SwitchID: "56df0e88-9edd-4536-af80-6b53537f1708",
				}}
			},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("9a463b0e-094a-401b-b508-2390367b376a").
					WillReturnRows(sqlmock.NewRows(
						[]string{
							"id",
							"created_at",
							"updated_at",
							"deleted_at",
							"name",
							"description",
							"type",
							"uplink",
						}).
						AddRow(
							"9a463b0e-094a-401b-b508-2390367b376a",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge0",
							"a simple test bridge",
							"IF",
							"em0",
						),
					)
				mock.ExpectBegin()

				mock.ExpectExec(
					regexp.QuoteMeta(
						"DELETE FROM `switches` WHERE `switches`.`id` = ?",
					),
				).
					WithArgs("9a463b0e-094a-401b-b508-2390367b376a").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args:    args{switchID: "9a463b0e-094a-401b-b508-2390367b376a"},
			wantErr: false,
		},
		{
			name: "error1",
			mockVmnicGetAllFunc: func() []*vmnic.VMNic {
				return []*vmnic.VMNic{{
					SwitchID: "9a463b0e-094a-401b-b508-2390367b376a",
				}}
			},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("9a463b0e-094a-401b-b508-2390367b376a").
					WillReturnRows(sqlmock.NewRows(
						[]string{
							"id",
							"created_at",
							"updated_at",
							"deleted_at",
							"name",
							"description",
							"type",
							"uplink",
						}).
						AddRow(
							"9a463b0e-094a-401b-b508-2390367b376a",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge0",
							"a simple test bridge",
							"IF",
							"em0",
						),
					)
			},
			args:    args{switchID: "9a463b0e-094a-401b-b508-2390367b376a"},
			wantErr: true,
		},
		{
			name: "error2",
			mockVmnicGetAllFunc: func() []*vmnic.VMNic {
				return []*vmnic.VMNic{{
					SwitchID: "56df0e88-9edd-4536-af80-6b53537f1708",
				}}
			},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("9a463b0e-094a-401b-b508-2390367b376a").
					WillReturnRows(sqlmock.NewRows(
						[]string{
							"id",
							"created_at",
							"updated_at",
							"deleted_at",
							"name",
							"description",
							"type",
							"uplink",
						}).
						AddRow(
							"9a463b0e-094a-401b-b508-2390367b376a",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge0",
							"a simple test bridge",
							"IF",
							"em0",
						),
					)
				mock.ExpectBegin()

				mock.ExpectExec(
					regexp.QuoteMeta(
						"DELETE FROM `switches` WHERE `switches`.`id` = ?",
					),
				).
					WithArgs("9a463b0e-094a-401b-b508-2390367b376a").
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
				mock.ExpectRollback()
			},
			args:    args{switchID: "9a463b0e-094a-401b-b508-2390367b376a"},
			wantErr: true,
		},
		{
			name: "error3",
			mockVmnicGetAllFunc: func() []*vmnic.VMNic {
				return []*vmnic.VMNic{{
					SwitchID: "56df0e88-9edd-4536-af80-6b53537f1708",
				}}
			},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("9a463b0e-094a-401b-b508-2390367b376a").
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			args:    args{switchID: "9a463b0e-094a-401b-b508-2390367b376a"},
			wantErr: true,
		},
		{
			name: "error4",
			mockVmnicGetAllFunc: func() []*vmnic.VMNic {
				return []*vmnic.VMNic{}
			},
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
			},
			args:    args{switchID: ""},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("switchTest")
			testCase.mockClosure(testDB, mock)

			vmnicGetAllFunc = testCase.mockVmnicGetAllFunc

			t.Cleanup(func() { vmnicGetAllFunc = vmnic.GetAll })

			err := Delete(testCase.args.switchID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, testCase.wantErr)
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

func Test_switchCheckUplink(t *testing.T) {
	type args struct {
		switchInst *Switch
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "SuccessIF1",
			mockCmdFunc: "Test_switchCheckUplinkSuccessIF1",
			args: args{switchInst: &Switch{
				Name:        "bridge0",
				Description: "test if switch",
				Type:        "IF",
				Uplink:      "em0",
			}},
			wantErr: false,
		},
		{
			name:        "SuccessNG1",
			mockCmdFunc: "Test_switchCheckUplinkSuccessNG1",
			args: args{switchInst: &Switch{
				Name:        "bnet0",
				Description: "test ng switch",
				Type:        "NG",
				Uplink:      "em0",
			}},
			wantErr: false,
		},
		{
			name:        "ErrorIF1",
			mockCmdFunc: "Test_switchCheckUplinkErrorIF1",
			args: args{switchInst: &Switch{
				Name:        "bridge0",
				Description: "test if switch",
				Type:        "IF",
				Uplink:      "em0",
			}},
			wantErr: true,
		},
		{
			name:        "InUseIF1",
			mockCmdFunc: "Test_switchCheckUplinkInUseIF1",
			args: args{switchInst: &Switch{
				Name:        "bridge0",
				Description: "test if switch",
				Type:        "IF",
				Uplink:      "em0",
			}},
			wantErr: true,
		},
		{
			name:        "ErrorNG1",
			mockCmdFunc: "Test_switchCheckUplinkErrorNG1",
			args: args{switchInst: &Switch{
				Name:        "bnet0",
				Description: "test ng switch",
				Type:        "NG",
				Uplink:      "em0",
			}},
			wantErr: true,
		},
		{
			name:        "InUseNG1",
			mockCmdFunc: "Test_switchCheckUplinkInUseNG1",
			args: args{switchInst: &Switch{
				Name:        "bnet0",
				Description: "test if switch",
				Type:        "NG",
				Uplink:      "em0",
			}},
			wantErr: true,
		},
		{
			name: "BadType",
			args: args{switchInst: &Switch{
				Type: "garbage",
			}},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := switchCheckUplink(testCase.args.switchInst)
			if (err != nil) != testCase.wantErr {
				t.Errorf("switchCheckUplink() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func Test_setUplinkIf(t *testing.T) {
	type args struct {
		uplink     string
		switchInst *Switch
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_setUplinkIfSuccess1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `switches` SET `description`=?,`name`=?,`type`=?,`uplink`=?,`updated_at`=? WHERE `switches`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("some bridge", "bridge0", "IF", "em0", sqlmock.AnyArg(), "83bd9693-ea10-43f4-b888-49d3b8bb7f35").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args: args{
				switchInst: &Switch{
					ID:          "83bd9693-ea10-43f4-b888-49d3b8bb7f35",
					Name:        "bridge0",
					Description: "some bridge",
					Type:        "IF",
					Uplink:      "",
				},
				uplink: "em0",
			},
		},
		{
			name:        "MemberCheckError",
			mockCmdFunc: "Test_setUplinkIfMemberCheckError",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
			},
			args: args{
				switchInst: &Switch{
					ID:          "83bd9693-ea10-43f4-b888-49d3b8bb7f35",
					Name:        "bridge0",
					Description: "some bridge",
					Type:        "IF",
					Uplink:      "",
				},
				uplink: "em0",
			},
			wantErr: true,
		},
		{
			name:        "MemberInUse1",
			mockCmdFunc: "Test_setUplinkIfMemberInUse1",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
			},
			args: args{
				switchInst: &Switch{
					ID:          "83bd9693-ea10-43f4-b888-49d3b8bb7f35",
					Name:        "bridge1",
					Description: "some bridge",
					Type:        "IF",
					Uplink:      "",
				},
				uplink: "em0",
			},
			wantErr: true,
		},
		{
			name:        "AddMemberError1",
			mockCmdFunc: "Test_setUplinkIfAddMemberError1",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
			},
			args: args{
				switchInst: &Switch{
					ID:          "83bd9693-ea10-43f4-b888-49d3b8bb7f35",
					Name:        "bridge0",
					Description: "some bridge",
					Type:        "IF",
					Uplink:      "",
				},
				uplink: "em0",
			},
			wantErr: true,
		},
		{
			name:        "SaveError",
			mockCmdFunc: "Test_setUplinkIfSuccess1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `switches` SET `description`=?,`name`=?,`type`=?,`uplink`=?,`updated_at`=? WHERE `switches`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("some bridge", "bridge0", "IF", "em0", sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
				mock.ExpectRollback()
			},
			args: args{
				switchInst: &Switch{
					ID:          "83bd9693-ea10-43f4-b888-49d3b8bb7f35",
					Name:        "bridge0",
					Description: "some bridge",
					Type:        "IF",
					Uplink:      "",
				},
				uplink: "em0",
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			testDB, mock := cirrinadtest.NewMockDB("switchTest")
			testCase.mockClosure(testDB, mock)

			err := setUplinkIf(testCase.args.uplink, testCase.args.switchInst)
			if (err != nil) != testCase.wantErr {
				t.Errorf("setUplinkIf() error = %v, wantErr %v", err, testCase.wantErr)
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

func Test_setUplinkNG(t *testing.T) {
	type args struct {
		uplink     string
		switchInst *Switch
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_setUplinkNGSuccess1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `switches` SET `description`=?,`name`=?,`type`=?,`uplink`=?,`updated_at`=? WHERE `switches`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("some bridge", "bnet0", "NG", "em0", sqlmock.AnyArg(), "20405b69-6d32-4690-8145-4d55a60f16a7").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args: args{
				switchInst: &Switch{
					ID:          "20405b69-6d32-4690-8145-4d55a60f16a7",
					Name:        "bnet0",
					Description: "some bridge",
					Type:        "NG",
					Uplink:      "",
				},
				uplink: "em0",
			},
		},
		{
			name:        "MemberUsedError",
			mockCmdFunc: "Test_setUplinkNGMemberUsedError",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
			},
			args: args{
				switchInst: &Switch{
					ID:          "20405b69-6d32-4690-8145-4d55a60f16a7",
					Name:        "bnet0",
					Description: "some bridge",
					Type:        "NG",
					Uplink:      "",
				},
				uplink: "em0",
			},
			wantErr: true,
		},
		{
			name:        "MemberAlreadyUsed",
			mockCmdFunc: "Test_setUplinkNGMemberAlreadyUsed",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
			},
			args: args{
				switchInst: &Switch{
					ID:          "20405b69-6d32-4690-8145-4d55a60f16a7",
					Name:        "bnet0",
					Description: "some bridge",
					Type:        "NG",
					Uplink:      "",
				},
				uplink: "em0",
			},
			wantErr: true,
		},
		{
			name:        "MemberAddError",
			mockCmdFunc: "Test_setUplinkNGMemberAddError",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
			},
			args: args{
				switchInst: &Switch{
					ID:          "20405b69-6d32-4690-8145-4d55a60f16a7",
					Name:        "bnet0",
					Description: "some bridge",
					Type:        "NG",
					Uplink:      "",
				},
				uplink: "em0",
			},
			wantErr: true,
		},
		{
			name:        "SaveError",
			mockCmdFunc: "Test_setUplinkNGSuccess1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `switches` SET `description`=?,`name`=?,`type`=?,`uplink`=?,`updated_at`=? WHERE `switches`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("some bridge", "bnet0", "NG", "em0", sqlmock.AnyArg(), "20405b69-6d32-4690-8145-4d55a60f16a7").
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
				mock.ExpectRollback()
			},
			args: args{
				switchInst: &Switch{
					ID:          "20405b69-6d32-4690-8145-4d55a60f16a7",
					Name:        "bnet0",
					Description: "some bridge",
					Type:        "NG",
					Uplink:      "",
				},
				uplink: "em0",
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			testDB, mock := cirrinadtest.NewMockDB("switchTest")
			testCase.mockClosure(testDB, mock)

			err := setUplinkNG(testCase.args.uplink, testCase.args.switchInst)
			if (err != nil) != testCase.wantErr {
				t.Errorf("setUplinkNG() error = %v, wantErr %v", err, testCase.wantErr)
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

func TestSwitch_SetUplink(t *testing.T) {
	type fields struct {
		Model       gorm.Model
		ID          string
		Name        string
		Description string
		Type        string
		Uplink      string
	}

	type args struct {
		uplink string
	}

	tests := []struct {
		name                string
		hostIntStubFunc     func() ([]net.Interface, error)
		getIntGroupStubFunc func(string) ([]string, error)
		mockClosure         func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		mockCmdFunc         string
		fields              fields
		args                args
		wantErr             bool
	}{
		{
			name:                "successIF",
			hostIntStubFunc:     StubHostInterfacesSuccess1,
			getIntGroupStubFunc: StubGetHostIntGroupSuccess1,
			mockCmdFunc:         "Test_setUplinkIfSuccess1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `switches` SET `description`=?,`name`=?,`type`=?,`uplink`=?,`updated_at`=? WHERE `switches`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("another test if bridge", "bridge0", "IF", "em0", sqlmock.AnyArg(), "1c336538-84ed-4303-8be0-e80f6367fb24"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:          "1c336538-84ed-4303-8be0-e80f6367fb24",
				Name:        "bridge0",
				Description: "another test if bridge",
				Type:        "IF",
				Uplink:      "",
			},
			args: args{
				uplink: "em0",
			},
		},
		{
			name:                "successNG",
			hostIntStubFunc:     StubHostInterfacesSuccess1,
			getIntGroupStubFunc: StubGetHostIntGroupSuccess1,
			mockCmdFunc:         "Test_setUplinkNgSuccess1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `switches` SET `description`=?,`name`=?,`type`=?,`uplink`=?,`updated_at`=? WHERE `switches`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("another test if bridge", "bnet0", "NG", "em0", sqlmock.AnyArg(), "1c336538-84ed-4303-8be0-e80f6367fb24").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:          "1c336538-84ed-4303-8be0-e80f6367fb24",
				Name:        "bnet0",
				Description: "another test if bridge",
				Type:        "NG",
				Uplink:      "",
			},
			args: args{
				uplink: "em0",
			},
		},
		{
			name:                "UplinkNotFound",
			hostIntStubFunc:     StubHostInterfacesSuccess1,
			getIntGroupStubFunc: StubGetHostIntGroupSuccess1,
			mockCmdFunc:         "Test_setUplinkIfSuccess1",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
			},
			fields: fields{
				ID:          "1c336538-84ed-4303-8be0-e80f6367fb24",
				Name:        "bridge0",
				Description: "another test if bridge",
				Type:        "IF",
				Uplink:      "",
			},
			args: args{
				uplink: "em2",
			},
			wantErr: true,
		},
		{
			name:                "InvalidSwitchType",
			hostIntStubFunc:     StubHostInterfacesSuccess1,
			getIntGroupStubFunc: StubGetHostIntGroupSuccess1,
			mockCmdFunc:         "Test_setUplinkIfSuccess1",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					switchDB: testDB,
				}
			},
			fields: fields{
				ID:          "1c336538-84ed-4303-8be0-e80f6367fb24",
				Name:        "bridge0",
				Description: "another test if bridge",
				Type:        "garbage",
				Uplink:      "",
			},
			args: args{
				uplink: "em0",
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			util.NetInterfacesFunc = testCase.hostIntStubFunc

			t.Cleanup(func() { util.NetInterfacesFunc = net.Interfaces })

			util.GetIntGroupsFunc = testCase.getIntGroupStubFunc

			t.Cleanup(func() { util.GetIntGroupsFunc = util.GetIntGroups })

			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			testDB, mock := cirrinadtest.NewMockDB("switchTest")
			testCase.mockClosure(testDB, mock)

			testSwitch := &Switch{
				Model:       testCase.fields.Model,
				ID:          testCase.fields.ID,
				Name:        testCase.fields.Name,
				Description: testCase.fields.Description,
				Type:        testCase.fields.Type,
				Uplink:      testCase.fields.Uplink,
			}

			err := testSwitch.SetUplink(testCase.args.uplink)
			if (err != nil) != testCase.wantErr {
				t.Errorf("SetUplink() error = %v, wantErr %v", err, testCase.wantErr)
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

// test helpers from here down

func Test_bringUpNewSwitchSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

func Test_bringUpNewSwitchError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]
	if len(cmdWithArgs) == 2 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "bridge0" {
		os.Exit(1)
	}

	os.Exit(0)
}

func Test_bringUpNewSwitchSuccess2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]
	//nolint:lll
	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		ifconfigOutput := `bridge1: flags=1008843<UP,BROADCAST,RUNNING,SIMPLEX,MULTICAST,LOWER_UP> metric 0 mtu 1500
        options=0
        ether 58:9c:fc:10:d6:22
        id 00:00:00:00:00:00 priority 32768 hellotime 2 fwddelay 15
        maxage 20 holdcnt 6 proto rstp maxaddr 2000 timeout 1200
        root id 00:00:00:00:00:00 priority 32768 ifcost 0 port 0
        member: em0 flags=143<LEARNING,DISCOVER,AUTOEDGE,AUTOPTP>
                ifmaxaddr 0 port 2 priority 128 path cost 20000
        groups: bridge cirrinad
        nd6 options=9<PERFORMNUD,IFDISABLED>
`
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

func StubBringUpNewSwitchHostInterfacesSuccess1() ([]net.Interface, error) {
	return []net.Interface{
		{
			Index:        0,
			MTU:          16384,
			Name:         "lo0",
			HardwareAddr: net.HardwareAddr(nil),
			Flags:        0x35,
		},
		{
			Index:        1,
			MTU:          1500,
			Name:         "em0",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x28, 0x73, 0x3e},
			Flags:        0x33,
		},
	}, nil
}

func Test_memberUsedByIfBridgeSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	//nolint:lll
	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		ifconfigOutput := "bridge0\n"
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 2 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "bridge0" {
		ifconfigOutput := `bridge0: flags=1008843<UP,BROADCAST,RUNNING,SIMPLEX,MULTICAST,LOWER_UP> metric 0 mtu 1500
        options=0
        ether 58:9c:fc:10:d6:22
        id 00:00:00:00:00:00 priority 32768 hellotime 2 fwddelay 15
        maxage 20 holdcnt 6 proto rstp maxaddr 2000 timeout 1200
        root id 00:00:00:00:00:00 priority 32768 ifcost 0 port 0
        groups: bridge cirrinad
        nd6 options=9<PERFORMNUD,IFDISABLED>
`
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

func Test_memberUsedByIfBridgeSuccess2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	//nolint:lll
	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		ifconfigOutput := "bridge0\n"
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 2 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "bridge0" {
		ifconfigOutput := `bridge0: flags=1008843<UP,BROADCAST,RUNNING,SIMPLEX,MULTICAST,LOWER_UP> metric 0 mtu 1500
        options=0
        ether 58:9c:fc:10:d6:22
        id 00:00:00:00:00:00 priority 32768 hellotime 2 fwddelay 15
        maxage 20 holdcnt 6 proto rstp maxaddr 2000 timeout 1200
        root id 00:00:00:00:00:00 priority 32768 ifcost 0 port 0
        member: ix0 flags=143<LEARNING,DISCOVER,AUTOEDGE,AUTOPTP>
                ifmaxaddr 0 port 2 priority 128 path cost 20000
        groups: bridge cirrinad
        nd6 options=9<PERFORMNUD,IFDISABLED>
`
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

func Test_memberUsedByIfBridgeSuccess3(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	//nolint:lll
	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		ifconfigOutput := "bridge0\n"
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 2 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "bridge0" {
		ifconfigOutput := `bridge0: flags=1008843<UP,BROADCAST,RUNNING,SIMPLEX,MULTICAST,LOWER_UP> metric 0 mtu 1500
        options=0
        ether 58:9c:fc:10:d6:22
        id 00:00:00:00:00:00 priority 32768 hellotime 2 fwddelay 15
        maxage 20 holdcnt 6 proto rstp maxaddr 2000 timeout 1200
        root id 00:00:00:00:00:00 priority 32768 ifcost 0 port 0
        member: em0 flags=143<LEARNING,DISCOVER,AUTOEDGE,AUTOPTP>
                ifmaxaddr 0 port 2 priority 128 path cost 20000
        groups: bridge cirrinad
        nd6 options=9<PERFORMNUD,IFDISABLED>
`
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

func Test_memberUsedByIfBridgeError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	//nolint:lll
	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		os.Exit(1)
	}

	os.Exit(0)
}

func Test_memberUsedByIfBridgeError2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	//nolint:lll
	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		ifconfigOutput := "bridge0\n"
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 2 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "bridge0" {
		os.Exit(1)
	}

	os.Exit(0)
}

func Test_memberUsedByNgBridgeSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "list" {
		ngctlOutput := `There are 8 total nodes:
  Name: igb0            Type: ether           ID: 00000001   Num hooks: 0
  Name: ix0             Type: ether           ID: 00000002   Num hooks: 2
  Name: ue0             Type: ether           ID: 00000003   Num hooks: 0
  Name: bridge0         Type: ether           ID: 00000006   Num hooks: 0
  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Name: bridge1         Type: ether           ID: 00000014   Num hooks: 0
  Name: bnet1           Type: bridge          ID: 00000018   Num hooks: 0
  Name: ngctl23503      Type: socket          ID: 0000001e   Num hooks: 0
`

		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 4 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" && cmdWithArgs[3] == "bnet0:" { //nolint:lll
		ngctlOutput := `  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Local hook      Peer name       Peer type    Peer ID         Peer hook      
  ----------      ---------       ---------    -------         ---------      
  link1           em0             ether        00000002        upper          
  link0           em0             ether        00000002        lower          
`
		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

func Test_memberUsedByNgBridgeError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "list" {
		os.Exit(1)
	}

	os.Exit(0)
}

func Test_memberUsedByNgBridgeError2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "list" {
		ngctlOutput := `There are 8 total nodes:
  Name: igb0            Type: ether           ID: 00000001   Num hooks: 0
  Name: ix0             Type: ether           ID: 00000002   Num hooks: 2
  Name: ue0             Type: ether           ID: 00000003   Num hooks: 0
  Name: bridge0         Type: ether           ID: 00000006   Num hooks: 0
  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Name: bridge1         Type: ether           ID: 00000014   Num hooks: 0
  Name: bnet1           Type: bridge          ID: 00000018   Num hooks: 0
  Name: ngctl23503      Type: socket          ID: 0000001e   Num hooks: 0
`

		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 4 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" && cmdWithArgs[3] == "bnet0:" { //nolint:lll
		os.Exit(1)
	}

	os.Exit(0)
}

func Test_ngGetBridgeNextLinkSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	ngctlOutput := `  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Local hook      Peer name       Peer type    Peer ID         Peer hook      
  ----------      ---------       ---------    -------         ---------      
  link1           em0             ether        00000002        upper          
  link0           em0             ether        00000002        lower          
`

	fmt.Print(ngctlOutput) //nolint:forbidigo
	os.Exit(0)
}

func Test_ngGetBridgeNextLinkError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

func Test_validateIfSwitchSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	//nolint:lll
	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		ifconfigOutput := "bridge0\n"
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 2 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "bridge0" {
		ifconfigOutput := `bridge0: flags=1008843<UP,BROADCAST,RUNNING,SIMPLEX,MULTICAST,LOWER_UP> metric 0 mtu 1500
        options=0
        ether 58:9c:fc:10:d6:22
        id 00:00:00:00:00:00 priority 32768 hellotime 2 fwddelay 15
        maxage 20 holdcnt 6 proto rstp maxaddr 2000 timeout 1200
        root id 00:00:00:00:00:00 priority 32768 ifcost 0 port 0
        member: em0 flags=143<LEARNING,DISCOVER,AUTOEDGE,AUTOPTP>
                ifmaxaddr 0 port 2 priority 128 path cost 20000
        groups: bridge cirrinad
        nd6 options=9<PERFORMNUD,IFDISABLED>
`
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

func Test_validateIfSwitchError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	//nolint:lll
	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		os.Exit(1)
	}

	os.Exit(0)
}

func Test_validateNgSwitchSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "list" {
		ngctlOutput := `There are 8 total nodes:
  Name: igb0            Type: ether           ID: 00000001   Num hooks: 0
  Name: ix0             Type: ether           ID: 00000002   Num hooks: 2
  Name: ue0             Type: ether           ID: 00000003   Num hooks: 0
  Name: bridge0         Type: ether           ID: 00000006   Num hooks: 0
  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Name: bridge1         Type: ether           ID: 00000014   Num hooks: 0
  Name: ngctl23503      Type: socket          ID: 0000001e   Num hooks: 0
`

		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 4 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" && cmdWithArgs[3] == "bnet0:" { //nolint:lll
		ngctlOutput := `  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Local hook      Peer name       Peer type    Peer ID         Peer hook      
  ----------      ---------       ---------    -------         ---------      
  link1           em0             ether        00000002        upper          
  link0           em0             ether        00000002        lower          
`
		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

func Test_validateNgSwitchError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "list" {
		ngctlOutput := `There are 8 total nodes:
  Name: igb0            Type: ether           ID: 00000001   Num hooks: 0
  Name: ix0             Type: ether           ID: 00000002   Num hooks: 2
  Name: ue0             Type: ether           ID: 00000003   Num hooks: 0
  Name: bridge0         Type: ether           ID: 00000006   Num hooks: 0
  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Name: bridge1         Type: ether           ID: 00000014   Num hooks: 0
  Name: bnet1           Type: bridge          ID: 00000018   Num hooks: 0
  Name: ngctl23503      Type: socket          ID: 0000001e   Num hooks: 0
`

		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 4 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" && cmdWithArgs[3] == "bnet0:" { //nolint:lll
		os.Exit(1)
	}

	os.Exit(0)
}

func TestDestroyNgBridgeSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

func TestDestroyNgBridgeError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

func TestDestroyIfBridgeSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

func TestDestroyIfBridgeError2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

func TestBridgeIfAddMemberSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

func TestBridgeIfAddMemberError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

func TestBridgeNgAddMemberSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

func TestBridgeNgAddMemberError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

func TestBridgeNgAddMemberError2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 4 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" {
		ngctlOutput := `  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Local hook      Peer name       Peer type    Peer ID         Peer hook      
  ----------      ---------       ---------    -------         ---------      
  link1           em0             ether        00000002        upper          
  link0           em0             ether        00000002        lower          
`

		fmt.Print(ngctlOutput) //nolint:forbidigo

		os.Exit(0)
	}

	os.Exit(1)
}

func TestBridgeNgAddMemberError3(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 4 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" {
		ngctlOutput := `  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Local hook      Peer name       Peer type    Peer ID         Peer hook      
  ----------      ---------       ---------    -------         ---------      
  link1           em0             ether        00000002        upper          
  link0           em0             ether        00000002        lower          
`

		fmt.Print(ngctlOutput) //nolint:forbidigo

		os.Exit(0)
	}

	//nolint:lll
	if len(cmdWithArgs) == 7 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "connect" && cmdWithArgs[5] == "lower" {
		os.Exit(0)
	}

	os.Exit(1)
}

func Test_switchCheckUplinkSuccessIF1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	//nolint:lll
	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		ifconfigOutput := "bridge0\n"
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 2 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "bridge0" {
		ifconfigOutput := `bridge0: flags=1008843<UP,BROADCAST,RUNNING,SIMPLEX,MULTICAST,LOWER_UP> metric 0 mtu 1500
        options=0
        ether 58:9c:fc:10:d6:22
        id 00:00:00:00:00:00 priority 32768 hellotime 2 fwddelay 15
        maxage 20 holdcnt 6 proto rstp maxaddr 2000 timeout 1200
        root id 00:00:00:00:00:00 priority 32768 ifcost 0 port 0
        groups: bridge cirrinad
        nd6 options=9<PERFORMNUD,IFDISABLED>
`
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

func Test_switchCheckUplinkSuccessNG1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "list" {
		ngctlOutput := `There are 8 total nodes:
  Name: igb0            Type: ether           ID: 00000001   Num hooks: 0
  Name: ix0             Type: ether           ID: 00000002   Num hooks: 2
  Name: ue0             Type: ether           ID: 00000003   Num hooks: 0
  Name: bridge0         Type: ether           ID: 00000006   Num hooks: 0
  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Name: bridge1         Type: ether           ID: 00000014   Num hooks: 0
  Name: bnet1           Type: bridge          ID: 00000018   Num hooks: 0
  Name: ngctl23503      Type: socket          ID: 0000001e   Num hooks: 0
`

		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 4 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" && cmdWithArgs[3] == "bnet0:" { //nolint:lll
		ngctlOutput := `  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 0
`
		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

func Test_switchCheckUplinkErrorIF1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	//nolint:lll
	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		os.Exit(1)
	}

	os.Exit(0)
}

func Test_switchCheckUplinkInUseIF1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	//nolint:lll
	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		ifconfigOutput := "bridge0\n"
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 2 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "bridge0" {
		ifconfigOutput := `bridge0: flags=1008843<UP,BROADCAST,RUNNING,SIMPLEX,MULTICAST,LOWER_UP> metric 0 mtu 1500
        options=0
        ether 58:9c:fc:10:d6:22
        id 00:00:00:00:00:00 priority 32768 hellotime 2 fwddelay 15
        maxage 20 holdcnt 6 proto rstp maxaddr 2000 timeout 1200
        root id 00:00:00:00:00:00 priority 32768 ifcost 0 port 0
        member: em0 flags=143<LEARNING,DISCOVER,AUTOEDGE,AUTOPTP>
                ifmaxaddr 0 port 2 priority 128 path cost 20000
        groups: bridge cirrinad
        nd6 options=9<PERFORMNUD,IFDISABLED>
`
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

func Test_switchCheckUplinkErrorNG1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "list" {
		os.Exit(1)
	}

	os.Exit(0)
}

func Test_switchCheckUplinkInUseNG1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "list" {
		ngctlOutput := `There are 8 total nodes:
  Name: igb0            Type: ether           ID: 00000001   Num hooks: 0
  Name: ix0             Type: ether           ID: 00000002   Num hooks: 2
  Name: ue0             Type: ether           ID: 00000003   Num hooks: 0
  Name: bridge0         Type: ether           ID: 00000006   Num hooks: 0
  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Name: bridge1         Type: ether           ID: 00000014   Num hooks: 0
  Name: bnet1           Type: bridge          ID: 00000018   Num hooks: 0
  Name: ngctl23503      Type: socket          ID: 0000001e   Num hooks: 0
`

		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 4 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" && cmdWithArgs[3] == "bnet0:" { //nolint:lll
		ngctlOutput := `  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Local hook      Peer name       Peer type    Peer ID         Peer hook      
  ----------      ---------       ---------    -------         ---------      
  link1           em0             ether        00000002        upper          
  link0           em0             ether        00000002        lower          
`
		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

func Test_setUplinkIfSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	//nolint:lll
	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		ifconfigOutput := "bridge0\n"
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 2 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "bridge0" {
		ifconfigOutput := `bridge0: flags=1008843<UP,BROADCAST,RUNNING,SIMPLEX,MULTICAST,LOWER_UP> metric 0 mtu 1500
        options=0
        ether 58:9c:fc:10:d6:22
        id 00:00:00:00:00:00 priority 32768 hellotime 2 fwddelay 15
        maxage 20 holdcnt 6 proto rstp maxaddr 2000 timeout 1200
        root id 00:00:00:00:00:00 priority 32768 ifcost 0 port 0
        groups: bridge cirrinad
        nd6 options=9<PERFORMNUD,IFDISABLED>
`
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

func Test_setUplinkIfMemberCheckError(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	//nolint:lll
	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		os.Exit(1)
	}

	os.Exit(0)
}

func Test_setUplinkIfMemberInUse1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	//nolint:lll
	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		ifconfigOutput := "bridge0\n"
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 2 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "bridge0" {
		ifconfigOutput := `bridge0: flags=1008843<UP,BROADCAST,RUNNING,SIMPLEX,MULTICAST,LOWER_UP> metric 0 mtu 1500
        options=0
        ether 58:9c:fc:10:d6:22
        id 00:00:00:00:00:00 priority 32768 hellotime 2 fwddelay 15
        maxage 20 holdcnt 6 proto rstp maxaddr 2000 timeout 1200
        root id 00:00:00:00:00:00 priority 32768 ifcost 0 port 0
        member: em0 flags=143<LEARNING,DISCOVER,AUTOEDGE,AUTOPTP>
                ifmaxaddr 0 port 2 priority 128 path cost 20000
        groups: bridge cirrinad
        nd6 options=9<PERFORMNUD,IFDISABLED>
`
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

func Test_setUplinkIfAddMemberError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	//nolint:lll
	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "-g" && cmdWithArgs[2] == "bridge" {
		ifconfigOutput := "bridge0\n"
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 2 && cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "bridge0" {
		ifconfigOutput := `bridge0: flags=1008843<UP,BROADCAST,RUNNING,SIMPLEX,MULTICAST,LOWER_UP> metric 0 mtu 1500
        options=0
        ether 58:9c:fc:10:d6:22
        id 00:00:00:00:00:00 priority 32768 hellotime 2 fwddelay 15
        maxage 20 holdcnt 6 proto rstp maxaddr 2000 timeout 1200
        root id 00:00:00:00:00:00 priority 32768 ifcost 0 port 0
        groups: bridge cirrinad
        nd6 options=9<PERFORMNUD,IFDISABLED>
`
		fmt.Print(ifconfigOutput) //nolint:forbidigo
	}

	for _, v := range cmdWithArgs {
		if v == "addm" {
			os.Exit(1)
		}
	}

	os.Exit(0)
}

func Test_setUplinkNGSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "list" {
		ngctlOutput := `There are 8 total nodes:
  Name: igb0            Type: ether           ID: 00000001   Num hooks: 0
  Name: ix0             Type: ether           ID: 00000002   Num hooks: 2
  Name: ue0             Type: ether           ID: 00000003   Num hooks: 0
  Name: bridge0         Type: ether           ID: 00000006   Num hooks: 0
  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 0
  Name: bridge1         Type: ether           ID: 00000014   Num hooks: 0
  Name: bnet1           Type: bridge          ID: 00000018   Num hooks: 0
  Name: ngctl23503      Type: socket          ID: 0000001e   Num hooks: 0
`

		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 4 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" && cmdWithArgs[3] == "bnet0:" { //nolint:lll
		ngctlOutput := `  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 0
`
		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

func Test_setUplinkNGMemberUsedError(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "list" {
		os.Exit(1)
	}

	os.Exit(0)
}

func Test_setUplinkNGMemberAlreadyUsed(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "list" {
		ngctlOutput := `There are 8 total nodes:
  Name: igb0            Type: ether           ID: 00000001   Num hooks: 0
  Name: ix0             Type: ether           ID: 00000002   Num hooks: 2
  Name: ue0             Type: ether           ID: 00000003   Num hooks: 0
  Name: bridge0         Type: ether           ID: 00000006   Num hooks: 0
  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 0
  Name: bridge1         Type: ether           ID: 00000014   Num hooks: 0
  Name: bnet1           Type: bridge          ID: 00000018   Num hooks: 0
  Name: ngctl23503      Type: socket          ID: 0000001e   Num hooks: 0
`

		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 4 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" && cmdWithArgs[3] == "bnet0:" { //nolint:lll
		ngctlOutput := `  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Local hook      Peer name       Peer type    Peer ID         Peer hook      
  ----------      ---------       ---------    -------         ---------      
  link1           em0             ether        00000002        upper          
  link0           em0             ether        00000002        lower          
`
		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	os.Exit(0)
}

func Test_setUplinkNGMemberAddError(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "list" {
		ngctlOutput := `There are 8 total nodes:
  Name: igb0            Type: ether           ID: 00000001   Num hooks: 0
  Name: ix0             Type: ether           ID: 00000002   Num hooks: 2
  Name: ue0             Type: ether           ID: 00000003   Num hooks: 0
  Name: bridge0         Type: ether           ID: 00000006   Num hooks: 0
  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 0
  Name: bridge1         Type: ether           ID: 00000014   Num hooks: 0
  Name: bnet1           Type: bridge          ID: 00000018   Num hooks: 0
  Name: ngctl23503      Type: socket          ID: 0000001e   Num hooks: 0
`

		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	if len(cmdWithArgs) == 4 && cmdWithArgs[1] == "/usr/sbin/ngctl" && cmdWithArgs[2] == "show" && cmdWithArgs[3] == "bnet0:" { //nolint:lll
		ngctlOutput := `  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 0
`
		fmt.Print(ngctlOutput) //nolint:forbidigo
	}

	for _, v := range cmdWithArgs {
		if v == "connect" {
			os.Exit(1)
		}
	}

	os.Exit(0)
}

func StubHostInterfacesSuccess1() ([]net.Interface, error) {
	return []net.Interface{
		{
			Index:        1,
			MTU:          1500,
			Name:         "em0",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x28, 0x73, 0x3e},
			Flags:        0x33,
		},
		{
			Index:        2,
			MTU:          16384,
			Name:         "lo0",
			HardwareAddr: net.HardwareAddr(nil),
			Flags:        0x35,
		},
	}, nil
}

func StubGetHostIntGroupSuccess1(intName string) ([]string, error) {
	switch intName {
	case "em0":
		return []string{}, nil
	case "lo0":
		return []string{"lo"}, nil
	default:
		return nil, nil
	}
}
