package vmswitch

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-test/deep"
	"gorm.io/gorm"

	"cirrina/cirrinad/cirrinadtest"
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
