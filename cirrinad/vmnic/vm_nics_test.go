package vmnic

import (
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"

	"cirrina/cirrinad/cirrinadtest"
)

func Test_nicTypeValid(t *testing.T) {
	type args struct {
		nicType string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "goodVirtIO",
			args: args{nicType: "VIRTIONET"},
			want: true,
		},
		{
			name: "goodE1000",
			args: args{nicType: "E1000"},
			want: true,
		},
		{
			name: "badJunk",
			args: args{nicType: "asdf"},
			want: false,
		},
		{
			name: "badEmpty",
			args: args{nicType: ""},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nicTypeValid(tt.args.nicType); got != tt.want {
				t.Errorf("nicTypeValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_nicDevTypeValid(t *testing.T) {
	type args struct {
		nicDevType string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "goodTap",
			args: args{nicDevType: "TAP"},
			want: true,
		},
		{
			name: "goodVMNet",
			args: args{nicDevType: "VMNET"},
			want: true,
		},
		{
			name: "goodNetGraph",
			args: args{nicDevType: "NETGRAPH"},
			want: true,
		},
		{
			name: "badEmpty",
			args: args{nicDevType: ""},
			want: false,
		},
		{
			name: "badJunk",
			args: args{nicDevType: "asdf"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nicDevTypeValid(tt.args.nicDevType); got != tt.want {
				t.Errorf("nicDevTypeValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetByName(t *testing.T) {
	testDB, mock := cirrinadtest.NewMockDB("nicTest")

	defer func(gormdb *gorm.DB) {
		db, err := gormdb.DB()
		if err != nil {
			t.Fatalf("failed closing db")
		}
		_ = db.Close()
	}(testDB)

	instance = &singleton{
		vmNicDB: testDB,
	}
	createUpdateTime := time.Now()

	mock.ExpectQuery("^SELECT \\* FROM `vm_nics` WHERE name = \\? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1$").
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
					"a045696b-1c49-49e7-80a0-12a69fc71ada",
					createUpdateTime,
					createUpdateTime,
					nil,
					"test2024030401_int0",
					"some description",
					"AUTO",
					"VIRTIONET",
					"TAP",
					"b7d4cafe-4665-467c-9642-d9c739a9c3b4",
					"tap0",
					false,
					0,
					0,
					"",
					"",
					123,
				),
		)
	mock.ExpectQuery("^SELECT \\* FROM `vm_nics` WHERE name = \\? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1$").
		WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
	mock.ExpectQuery("^SELECT \\* FROM `vm_nics` WHERE name = \\? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1$").
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
			),
		)

	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    *VMNic
		wantErr bool
	}{
		{
			name: "getSomeNic",
			args: args{name: "test2024041901_int0"},
			want: &VMNic{
				Model: gorm.Model{
					ID:        0,
					CreatedAt: createUpdateTime,
					UpdatedAt: createUpdateTime,
					DeletedAt: gorm.DeletedAt{},
				},
				ID:          "a045696b-1c49-49e7-80a0-12a69fc71ada",
				Name:        "test2024030401_int0",
				Description: "some description",
				Mac:         "AUTO",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "b7d4cafe-4665-467c-9642-d9c739a9c3b4",
				NetDev:      "tap0",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    123,
			},
			wantErr: false,
		},
		{
			name:    "testGetByName_error",
			args:    args{name: "someNicName"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "testGetByName_notfound",
			args:    args{name: "someRandomNic"},
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

func TestGetAll(t *testing.T) {
	testDB, mock := cirrinadtest.NewMockDB("nicTest")

	defer func(gormdb *gorm.DB) {
		db, err := gormdb.DB()
		if err != nil {
			t.Fatalf("failed closing db")
		}
		_ = db.Close()
	}(testDB)

	instance = &singleton{
		vmNicDB: testDB,
	}
	createUpdateTime := time.Now()

	mock.ExpectQuery("^SELECT \\* FROM `vm_nics` WHERE `vm_nics`.`deleted_at` IS NULL$").
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
					"a045696b-1c49-49e7-80a0-12a69fc71ada",
					createUpdateTime,
					createUpdateTime,
					nil,
					"test2024030401_int0",
					"first VM nic for test2024030401",
					"AUTO",
					"VIRTIONET",
					"TAP",
					"b7d4cafe-4665-467c-9642-d9c739a9c3b4",
					"tap0",
					false,
					0,
					0,
					"",
					"",
					123,
				).
				AddRow(
					"15b67c62-4b9a-491b-bc5f-2d4343ccd02b",
					createUpdateTime,
					createUpdateTime,
					nil,
					"test2024030401_int1",
					"second VM nic for test2024030401",
					"AUTO",
					"VIRTIONET",
					"TAP",
					"b7d4cafe-4665-467c-9642-d9c739a9c3b4",
					"tap1",
					false,
					0,
					0,
					"",
					"",
					123,
				),
		)

	tests := []struct {
		name string
		want []*VMNic
	}{
		{
			name: "testGetAllNics",
			want: []*VMNic{
				{
					Model: gorm.Model{
						ID:        0,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
						DeletedAt: gorm.DeletedAt{},
					},
					ID:          "a045696b-1c49-49e7-80a0-12a69fc71ada",
					Name:        "test2024030401_int0",
					Description: "first VM nic for test2024030401",
					Mac:         "AUTO",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "b7d4cafe-4665-467c-9642-d9c739a9c3b4",
					NetDev:      "tap0",
					RateLimit:   false,
					RateIn:      0,
					RateOut:     0,
					InstBridge:  "",
					InstEpair:   "",
					ConfigID:    123,
				},
				{
					Model: gorm.Model{
						ID:        0,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
						DeletedAt: gorm.DeletedAt{},
					},
					ID:          "15b67c62-4b9a-491b-bc5f-2d4343ccd02b",
					Name:        "test2024030401_int1",
					Description: "second VM nic for test2024030401",
					Mac:         "AUTO",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "b7d4cafe-4665-467c-9642-d9c739a9c3b4",
					NetDev:      "tap1",
					RateLimit:   false,
					RateIn:      0,
					RateOut:     0,
					InstBridge:  "",
					InstEpair:   "",
					ConfigID:    123,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAll(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetByID(t *testing.T) {
	testDB, mock := cirrinadtest.NewMockDB("nicTest")

	defer func(gormdb *gorm.DB) {
		db, err := gormdb.DB()
		if err != nil {
			t.Fatalf("failed closing db")
		}
		_ = db.Close()
	}(testDB)

	instance = &singleton{
		vmNicDB: testDB,
	}
	createUpdateTime := time.Now()

	mock.ExpectQuery("^SELECT \\* FROM `vm_nics` WHERE id = \\? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1$").
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
					"824a4217-2bf9-477c-9326-b5aa7326df03",
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
	mock.ExpectQuery("^SELECT \\* FROM `vm_nics` WHERE id = \\? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1$").
		WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
	mock.ExpectQuery("^SELECT \\* FROM `vm_nics` WHERE id = \\? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1$").
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
			),
		)

	type args struct {
		id string
	}
	tests := []struct {
		name    string
		args    args
		want    *VMNic
		wantErr bool
	}{
		{
			name: "testGetByID_success",
			args: args{id: "824a4217-2bf9-477c-9326-b5aa7326df03"},
			want: &VMNic{
				Model: gorm.Model{
					ID:        0,
					CreatedAt: createUpdateTime,
					UpdatedAt: createUpdateTime,
					DeletedAt: gorm.DeletedAt{},
				},
				ID:          "824a4217-2bf9-477c-9326-b5aa7326df03",
				Name:        "test2024050501_int0",
				Description: "another test nic",
				Mac:         "AUTO",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "b7d4cafe-4665-467c-9642-d9c739a9c3b4",
				NetDev:      "",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    123,
			},
			wantErr: false,
		},
		{
			name:    "testGetByID_error",
			args:    args{id: "007af66e-9c05-41a6-832a-40273cce3bf8"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "testGetByID_notfound",
			args:    args{id: "bb7061d5-c6a7-44d8-857f-e6f2f813d499"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "testGetByID_empty",
			args:    args{id: ""},
			want:    nil,
			wantErr: true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := GetByID(testCase.args.id)
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
