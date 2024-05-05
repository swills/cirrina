package vmswitch

import (
	"log"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewMockDB() (*gorm.DB, sqlmock.Sqlmock) {
	testDB, mock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("An error '%s' was not expected when opening a stub database connection", err)
	}

	mock.ExpectQuery("select sqlite_version()").
		WillReturnRows(sqlmock.NewRows([]string{"sqlite_version()"}).AddRow("3.40.1"))

	gormDB, err := gorm.Open(&sqlite.Dialector{DSN: "testDB", Conn: testDB}, &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		log.Fatalf("An error '%s' was not expected when opening gorm database", err)
	}

	return gormDB, mock
}

func TestGetAll(t *testing.T) {
	testDB, mock := NewMockDB()

	defer func(gormdb *gorm.DB) {
		db, err := gormdb.DB()
		if err != nil {
			t.Fatalf("failed closing db")
		}
		_ = db.Close()
	}(testDB)

	instance = &singleton{
		switchDB: testDB,
	}
	createUpdateTime := time.Now()

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

	tests := []struct {
		name string
		want []*Switch
	}{
		{
			name: "testGetAllSwitches",
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
		t.Run(testCase.name, func(t *testing.T) {
			if got := GetAll(); !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("GetAll() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestGetByName(t *testing.T) {
	testDB, mock := NewMockDB()

	defer func(gormdb *gorm.DB) {
		db, err := gormdb.DB()
		if err != nil {
			t.Fatalf("failed closing db")
		}
		_ = db.Close()
	}(testDB)

	instance = &singleton{
		switchDB: testDB,
	}
	createUpdateTime := time.Now()

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
	mock.ExpectQuery("^SELECT \\* FROM `switches` WHERE name = \\? AND `switches`.`deleted_at` IS NULL LIMIT 1$").
		WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
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

	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    *Switch
		wantErr bool
	}{
		{
			name: "testGetByName_bridge0",
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
			name:    "testGetByName_error",
			args:    args{name: "bridge0"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "testGetByName_notfound",
			args:    args{name: "bridge0"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "testGetByName_emptyName",
			args:    args{name: ""},
			want:    nil,
			wantErr: true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := GetByName(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetByName() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}
			if !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("GetByName() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestGetByID(t *testing.T) {
	testDB, mock := NewMockDB()

	defer func(gormdb *gorm.DB) {
		db, err := gormdb.DB()
		if err != nil {
			t.Fatalf("failed closing db")
		}
		_ = db.Close()
	}(testDB)

	instance = &singleton{
		switchDB: testDB,
	}
	createUpdateTime := time.Now()

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
	mock.ExpectQuery("^SELECT \\* FROM `switches` WHERE id = \\? AND `switches`.`deleted_at` IS NULL LIMIT 1$").
		WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
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

	type args struct {
		switchID string
	}
	tests := []struct {
		name    string
		args    args
		want    *Switch
		wantErr bool
	}{
		{
			name: "testGetByID_success",
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
			name:    "testGetByID_error",
			args:    args{switchID: "0cb98661-6470-432d-8fa4-5eca3668b494"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "testGetByID_notfound",
			args:    args{switchID: "713e2714-eb92-4b53-b129-9d1f914eaa06"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := GetByID(testCase.args.switchID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}
			if !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("GetByID() got = %v, want %v", got, testCase.want)
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := switchNameValid(tt.args.switchInst); got != tt.want {
				t.Errorf("switchNameValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
