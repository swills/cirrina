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
				mock.ExpectQuery("^SELECT \\* FROM `switches` WHERE `switches`.`deleted_at` IS NULL$").
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
				mock.ExpectQuery("^SELECT \\* FROM `switches` WHERE name = \\? AND `switches`.`deleted_at` IS NULL LIMIT 1$").
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
				mock.ExpectQuery("^SELECT \\* FROM `switches` WHERE name = \\? AND `switches`.`deleted_at` IS NULL LIMIT 1$").
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
				mock.ExpectQuery("^SELECT \\* FROM `switches` WHERE name = \\? AND `switches`.`deleted_at` IS NULL LIMIT 1$").
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
			testDB, mock := cirrinadtest.NewMockDB("nicTest")
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
				mock.ExpectQuery("^SELECT \\* FROM `switches` WHERE id = \\? AND `switches`.`deleted_at` IS NULL LIMIT 1$").
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
				mock.ExpectQuery("^SELECT \\* FROM `switches` WHERE id = \\? AND `switches`.`deleted_at` IS NULL LIMIT 1$").
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
				mock.ExpectQuery("^SELECT \\* FROM `switches` WHERE id = \\? AND `switches`.`deleted_at` IS NULL LIMIT 1$").
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
			testDB, mock := cirrinadtest.NewMockDB("nicTest")
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
			testDB, mock := cirrinadtest.NewMockDB("nicTest")
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
