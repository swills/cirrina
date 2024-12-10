package vmnic

import (
	"errors"
	"net"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-test/deep"
	"gorm.io/gorm"

	"cirrina/cirrina"
	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
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

	t.Parallel()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := nicTypeValid(testCase.args.nicType)
			if got != testCase.want {
				t.Errorf("nicTypeValid() = %v, want %v", got, testCase.want)
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

	t.Parallel()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := nicDevTypeValid(testCase.args.nicDevType)
			if got != testCase.want {
				t.Errorf("nicDevTypeValid() = %v, want %v", got, testCase.want)
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
		want        *VMNic
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		wantErr     bool
	}{
		{
			name: "getSomeNic",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
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
			},
			args: args{name: "test2024041901_int0"},
			want: &VMNic{
				ID:          "a045696b-1c49-49e7-80a0-12a69fc71ada",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				DeletedAt:   gorm.DeletedAt{},
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
			name: "testGetByName_error",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			args:    args{name: "someNicName"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "testGetByName_notfound",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
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
			},
			args:    args{name: "someRandomNic"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "testGetByName_emptyName",
			args: args{name: ""},
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(t.Name())
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

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mock.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestGetAll(t *testing.T) {
	createUpdateTime := time.Now()

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        []*VMNic
	}{
		{
			name: "testGetAllNics",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE `vm_nics`.`deleted_at` IS NULL"),
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
			},
			want: []*VMNic{
				{
					ID:          "a045696b-1c49-49e7-80a0-12a69fc71ada",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					DeletedAt:   gorm.DeletedAt{},
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
					ID:          "15b67c62-4b9a-491b-bc5f-2d4343ccd02b",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					DeletedAt:   gorm.DeletedAt{},
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
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(t.Name())
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

func TestGetByID(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		id string
	}

	tests := []struct {
		name        string
		args        args
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        *VMNic
		wantErr     bool
	}{
		{
			name: "testGetByID_success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
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
			},
			args: args{id: "824a4217-2bf9-477c-9326-b5aa7326df03"},
			want: &VMNic{
				ID:          "824a4217-2bf9-477c-9326-b5aa7326df03",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				DeletedAt:   gorm.DeletedAt{},
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
			name: "testGetByID_error",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			args:    args{id: "007af66e-9c05-41a6-832a-40273cce3bf8"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "testGetByID_notfound",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
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
			},
			args:    args{id: "bb7061d5-c6a7-44d8-857f-e6f2f813d499"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "testGetByID_empty",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
			},
			args:    args{id: ""},
			want:    nil,
			wantErr: true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mock)

			got, err := GetByID(testCase.args.id)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, testCase.wantErr)

				return
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

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestParseNetDevType(t *testing.T) {
	type args struct {
		netDevType cirrina.NetDevType
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "tap",
			args:    args{netDevType: cirrina.NetDevType_TAP},
			want:    "TAP",
			wantErr: false,
		},
		{
			name:    "vmnet",
			args:    args{netDevType: cirrina.NetDevType_VMNET},
			want:    "VMNET",
			wantErr: false,
		},
		{
			name:    "netgraph",
			args:    args{netDevType: cirrina.NetDevType_NETGRAPH},
			want:    "NETGRAPH",
			wantErr: false,
		},
		{
			name:    "fail1",
			args:    args{netDevType: -1},
			want:    "",
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := ParseNetDevType(testCase.args.netDevType)
			if (err != nil) != testCase.wantErr {
				t.Errorf("ParseNetDevType() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("ParseNetDevType() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestParseNetType(t *testing.T) {
	type args struct {
		netType cirrina.NetType
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "virtio",
			args:    args{netType: cirrina.NetType_VIRTIONET},
			want:    "VIRTIONET",
			wantErr: false,
		},
		{
			name:    "E1000",
			args:    args{netType: cirrina.NetType_E1000},
			want:    "E1000",
			wantErr: false,
		},
		{
			name:    "fail1",
			args:    args{netType: -1},
			want:    "",
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := ParseNetType(testCase.args.netType)
			if (err != nil) != testCase.wantErr {
				t.Errorf("ParseNetType() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("ParseNetType() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestParseMac(t *testing.T) {
	type args struct {
		macAddress string
	}

	tests := []struct {
		name          string
		args          args
		broadcastFunc func(string) (bool, error)
		multicastFunc func(string) (bool, error)
		want          string
		wantErr       bool
	}{
		{
			name: "auto",
			args: args{macAddress: "AUTO"},
			broadcastFunc: func(_ string) (bool, error) {
				return false, nil
			},
			multicastFunc: func(_ string) (bool, error) {
				return false, nil
			},
			want:    "AUTO",
			wantErr: false,
		},
		{
			name: "empty",
			args: args{macAddress: ""},
			broadcastFunc: func(_ string) (bool, error) {
				return false, nil
			},
			multicastFunc: func(_ string) (bool, error) {
				return false, nil
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "broadcastErr",
			broadcastFunc: func(_ string) (bool, error) {
				return false, errors.New("some error") //nolint:err113
			},
			multicastFunc: func(_ string) (bool, error) {
				return false, nil
			},
			args:    args{macAddress: "garbage"},
			want:    "",
			wantErr: true,
		},
		{
			name: "broadcast",
			args: args{macAddress: "FF:FF:FF:FF:FF:FF"},
			broadcastFunc: func(_ string) (bool, error) {
				return true, nil
			},
			multicastFunc: func(_ string) (bool, error) {
				return false, nil
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "multicastErr",
			broadcastFunc: func(_ string) (bool, error) {
				return false, nil
			},
			multicastFunc: func(_ string) (bool, error) {
				return false, errors.New("some error") //nolint:err113
			},
			args:    args{macAddress: "garbage"},
			want:    "",
			wantErr: true,
		},
		{
			name: "broadcast",
			args: args{macAddress: "FF:FF:FF:FF:FF:FF"},
			broadcastFunc: func(_ string) (bool, error) {
				return false, nil
			},
			multicastFunc: func(_ string) (bool, error) {
				return true, nil
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "parseErr",
			args: args{macAddress: "garbage"},
			broadcastFunc: func(_ string) (bool, error) {
				return false, nil
			},
			multicastFunc: func(_ string) (bool, error) {
				return false, nil
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "wrongKindOfMac",
			args: args{macAddress: "02:00:5e:10:00:00:00:01"},
			broadcastFunc: func(_ string) (bool, error) {
				return false, nil
			},
			multicastFunc: func(_ string) (bool, error) {
				return false, nil
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "success1",
			args: args{macAddress: "00:A0:98:11:22:33"},
			broadcastFunc: func(_ string) (bool, error) {
				return false, nil
			},
			multicastFunc: func(_ string) (bool, error) {
				return false, nil
			},
			want:    "00:a0:98:11:22:33",
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			MacIsBroadcastFunc = testCase.broadcastFunc

			t.Cleanup(func() { MacIsBroadcastFunc = util.MacIsBroadcast })

			MacIsMulticastFunc = testCase.multicastFunc

			t.Cleanup(func() { MacIsMulticastFunc = util.MacIsMulticast })

			got, err := ParseMac(testCase.args.macAddress)
			if (err != nil) != testCase.wantErr {
				t.Errorf("ParseMac() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("ParseMac() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestVMNic_Delete(t *testing.T) {
	type fields struct {
		ID          string
		CreatedAt   time.Time
		UpdatedAt   time.Time
		DeletedAt   gorm.DeletedAt
		Name        string
		Description string
		Mac         string
		NetDev      string
		NetType     string
		NetDevType  string
		SwitchID    string
		RateLimit   bool
		RateIn      uint64
		RateOut     uint64
		InstBridge  string
		InstEpair   string
		ConfigID    uint
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields      fields
		wantErr     bool
	}{
		{
			name: "ErrorDeleting",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_nics` WHERE `vm_nics`.`id` = ?"),
				).
					WithArgs("0804a461-4ac3-4f6d-bf40-e7c3c404a77f").
					WillReturnError(gorm.ErrInvalidField)
				mock.ExpectRollback()
			},
			fields: fields{
				ID: "0804a461-4ac3-4f6d-bf40-e7c3c404a77f",
			},
			wantErr: true,
		},
		{
			name: "ErrNicInUse",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
			},
			fields: fields{
				ID:       "00e58e32-b058-4617-a3db-a270e80ff801",
				ConfigID: 1,
			},
			wantErr: true,
		},
		{
			name: "success1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_nics` WHERE `vm_nics`.`id` = ?"),
				).
					WithArgs("00e58e32-b058-4617-a3db-a270e80ff801").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID: "00e58e32-b058-4617-a3db-a270e80ff801",
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testNic := &VMNic{
				ID:          testCase.fields.ID,
				CreatedAt:   testCase.fields.CreatedAt,
				UpdatedAt:   testCase.fields.UpdatedAt,
				DeletedAt:   testCase.fields.DeletedAt,
				Name:        testCase.fields.Name,
				Description: testCase.fields.Description,
				Mac:         testCase.fields.Mac,
				NetDev:      testCase.fields.NetDev,
				NetType:     testCase.fields.NetType,
				NetDevType:  testCase.fields.NetDevType,
				SwitchID:    testCase.fields.SwitchID,
				RateLimit:   testCase.fields.RateLimit,
				RateIn:      testCase.fields.RateIn,
				RateOut:     testCase.fields.RateOut,
				InstBridge:  testCase.fields.InstBridge,
				InstEpair:   testCase.fields.InstEpair,
				ConfigID:    testCase.fields.ConfigID,
			}

			testDB, mock := cirrinadtest.NewMockDB(t.Name())

			testCase.mockClosure(testDB, mock)

			err := testNic.Delete()
			if (err != nil) != testCase.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, testCase.wantErr)
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

func TestVMNic_Save(t *testing.T) {
	createUpdateTime := time.Now()

	type fields struct {
		ID          string
		CreatedAt   time.Time
		UpdatedAt   time.Time
		DeletedAt   gorm.DeletedAt
		Name        string
		Description string
		Mac         string
		NetDev      string
		NetType     string
		NetDevType  string
		SwitchID    string
		RateLimit   bool
		RateIn      uint64
		RateOut     uint64
		InstBridge  string
		InstEpair   string
		ConfigID    uint
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields      fields
		wantErr     bool
	}{
		{
			name: "success1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(321, "a test nic", "", "", "00:a0:98:11:22:33", "someNic", "", "TAP",
						"VIRTIONET", 0, false, 0, "f1a0beec-e49d-4463-98e8-839b6b3f468d", sqlmock.AnyArg(),
						"45b862b3-8459-4253-8116-7f313198c3b6").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:        "45b862b3-8459-4253-8116-7f313198c3b6",
				CreatedAt: createUpdateTime,
				UpdatedAt: createUpdateTime,
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "someNic",
				Description: "a test nic",
				Mac:         "00:a0:98:11:22:33",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "f1a0beec-e49d-4463-98e8-839b6b3f468d",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    321,
			},
			wantErr: false,
		},
		{
			name: "error1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(321, "a test nic", "", "", "00:a0:98:11:22:33", "someNic", "", "TAP",
						"VIRTIONET", 0, false, 0, "f1a0beec-e49d-4463-98e8-839b6b3f468d", sqlmock.AnyArg(),
										"45b862b3-8459-4253-8116-7f313198c3b6").
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
				mock.ExpectRollback()
			},
			fields: fields{
				ID:        "45b862b3-8459-4253-8116-7f313198c3b6",
				CreatedAt: createUpdateTime,
				UpdatedAt: createUpdateTime,
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "someNic",
				Description: "a test nic",
				Mac:         "00:a0:98:11:22:33",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "f1a0beec-e49d-4463-98e8-839b6b3f468d",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    321,
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testNic := &VMNic{
				ID:          testCase.fields.ID,
				CreatedAt:   testCase.fields.CreatedAt,
				UpdatedAt:   testCase.fields.UpdatedAt,
				DeletedAt:   testCase.fields.DeletedAt,
				Name:        testCase.fields.Name,
				Description: testCase.fields.Description,
				Mac:         testCase.fields.Mac,
				NetDev:      testCase.fields.NetDev,
				NetType:     testCase.fields.NetType,
				NetDevType:  testCase.fields.NetDevType,
				SwitchID:    testCase.fields.SwitchID,
				RateLimit:   testCase.fields.RateLimit,
				RateIn:      testCase.fields.RateIn,
				RateOut:     testCase.fields.RateOut,
				InstBridge:  testCase.fields.InstBridge,
				InstEpair:   testCase.fields.InstEpair,
				ConfigID:    testCase.fields.ConfigID,
			}

			testDB, mock := cirrinadtest.NewMockDB(t.Name())

			testCase.mockClosure(testDB, mock)

			err := testNic.Save()

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

func TestGetNics(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		vmConfigID uint
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        []VMNic
		wantErr     bool
	}{
		{
			name: "success1",
			args: args{vmConfigID: 321},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(321).
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
			want: []VMNic{},
		},
		{
			name: "success2",
			args: args{vmConfigID: 321},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
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
						321,
					),
				)
			},
			want: []VMNic{{
				ID:        "f1268949-35b5-40ca-a422-22147d38d700",
				CreatedAt: createUpdateTime,
				UpdatedAt: createUpdateTime,
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "aNic",
				Description: "a description",
				Mac:         "00:11:22:33:44:55",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "6ad8f637-22ee-43aa-b9d8-10df5fd7f50f",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    321,
			}},
		},
		{
			name: "dberr",
			args: args{vmConfigID: 321},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(t.Name())

			testCase.mockClosure(testDB, mock)

			got, err := GetNics(testCase.args.vmConfigID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetNics() error = %v, wantErr %v", err, testCase.wantErr)
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
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

func TestVMNic_SetSwitch(t *testing.T) {
	type fields struct {
		Model       gorm.Model
		ID          string
		Name        string
		Description string
		Mac         string
		NetDev      string
		NetType     string
		NetDevType  string
		SwitchID    string
		RateLimit   bool
		RateIn      uint64
		RateOut     uint64
		InstBridge  string
		InstEpair   string
		ConfigID    uint
	}

	type args struct {
		switchid string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields      fields
		args        args
		wantErr     bool
	}{
		{
			name: "success1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(432, "another test description", "", "", "12:34:56:78:9a:bc", "yetAnotherTestNic", "", "TAP",
						"VIRTIONET", 0, false, 0, "5dca509e-086c-4447-a38d-91a898b29518", sqlmock.AnyArg(),
						"d9eb6d83-379a-4803-9946-f55b99744137").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				Model: gorm.Model{
					ID:        0,
					CreatedAt: time.Time{},
					UpdatedAt: time.Time{},
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
				},
				ID:          "d9eb6d83-379a-4803-9946-f55b99744137",
				Name:        "yetAnotherTestNic",
				Description: "another test description",
				Mac:         "12:34:56:78:9a:bc",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "d1818527-693e-4b4d-be3d-36ec72d6c420",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    432,
			},
			args:    args{switchid: "5dca509e-086c-4447-a38d-91a898b29518"},
			wantErr: false,
		},
		{
			name: "error1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(432, "another test description", "", "", "12:34:56:78:9a:bc", "yetAnotherTestNic", "", "TAP",
						"VIRTIONET", 0, false, 0, "5dca509e-086c-4447-a38d-91a898b29518", sqlmock.AnyArg(),
										"d9eb6d83-379a-4803-9946-f55b99744137").
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
				mock.ExpectRollback()
			},
			fields: fields{
				Model: gorm.Model{
					ID:        0,
					CreatedAt: time.Time{},
					UpdatedAt: time.Time{},
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
				},
				ID:          "d9eb6d83-379a-4803-9946-f55b99744137",
				Name:        "yetAnotherTestNic",
				Description: "another test description",
				Mac:         "12:34:56:78:9a:bc",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "d1818527-693e-4b4d-be3d-36ec72d6c420",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    432,
			},
			args:    args{switchid: "5dca509e-086c-4447-a38d-91a898b29518"},
			wantErr: true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testNic := &VMNic{
				ID:          testCase.fields.ID,
				Name:        testCase.fields.Name,
				Description: testCase.fields.Description,
				Mac:         testCase.fields.Mac,
				NetDev:      testCase.fields.NetDev,
				NetType:     testCase.fields.NetType,
				NetDevType:  testCase.fields.NetDevType,
				SwitchID:    testCase.fields.SwitchID,
				RateLimit:   testCase.fields.RateLimit,
				RateIn:      testCase.fields.RateIn,
				RateOut:     testCase.fields.RateOut,
				InstBridge:  testCase.fields.InstBridge,
				InstEpair:   testCase.fields.InstEpair,
				ConfigID:    testCase.fields.ConfigID,
			}
			testDB, mock := cirrinadtest.NewMockDB(t.Name())

			testCase.mockClosure(testDB, mock)

			err := testNic.SetSwitch(testCase.args.switchid)

			if (err != nil) != testCase.wantErr {
				t.Errorf("SetSwitch() error = %v, wantErr %v", err, testCase.wantErr)
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

func Test_nicExists(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		nicName string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        bool
		wantErr     bool
	}{
		{
			name: "successTrue1",
			args: args{nicName: "test2024030401_int0"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
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
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "successFalse1",
			args: args{nicName: "test2024030401_int0"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
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
							}),
					)
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "error1",
			args: args{nicName: "test2024030401_int0"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(t.Name())

			testCase.mockClosure(testDB, mock)

			got, err := nicExists(testCase.args.nicName)
			if (err != nil) != testCase.wantErr {
				t.Errorf("nicExists() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("nicExists() got = %v, want %v", got, testCase.want)
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

func Test_validateNic(t *testing.T) {
	type args struct {
		vmNicInst *VMNic
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success1",
			args: args{vmNicInst: &VMNic{
				ID:        "283ebc7e-e7b7-4808-ae89-9989afd66ce8",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "genericNicName",
				Description: "yet another test description also",
				Mac:         "00:22:33:44:55:66",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "96590f8a-de19-4701-964c-628147636244",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    567,
			}},
			wantErr: false,
		},
		{
			name: "error1",
			args: args{vmNicInst: &VMNic{
				ID:        "283ebc7e-e7b7-4808-ae89-9989afd66ce8",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "genericNicName!",
				Description: "yet another test description also",
				Mac:         "00:22:33:44:55:66",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "96590f8a-de19-4701-964c-628147636244",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    567,
			}},
			wantErr: true,
		},
		{
			name: "error2",
			args: args{vmNicInst: &VMNic{
				ID:        "283ebc7e-e7b7-4808-ae89-9989afd66ce8",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "genericNicName",
				Description: "yet another test description also",
				Mac:         "00:22:33:44:55:66",
				NetDev:      "",
				NetType:     "garbage",
				NetDevType:  "TAP",
				SwitchID:    "96590f8a-de19-4701-964c-628147636244",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    567,
			}},
			wantErr: true,
		},
		{
			name: "error3",
			args: args{vmNicInst: &VMNic{
				ID:        "283ebc7e-e7b7-4808-ae89-9989afd66ce8",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "genericNicName",
				Description: "yet another test description also",
				Mac:         "00:22:33:44:55:66",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "garbage",
				SwitchID:    "96590f8a-de19-4701-964c-628147636244",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    567,
			}},
			wantErr: true,
		},
		{
			name: "error4",
			args: args{vmNicInst: &VMNic{
				ID:        "283ebc7e-e7b7-4808-ae89-9989afd66ce8",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "genericNicName",
				Description: "yet another test description also",
				Mac:         "garbage",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "96590f8a-de19-4701-964c-628147636244",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    567,
			}},
			wantErr: true,
		},
		{
			name: "error5",
			args: args{vmNicInst: &VMNic{
				ID:        "283ebc7e-e7b7-4808-ae89-9989afd66ce8",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "genericNicName",
				Description: "yet another test description also",
				Mac:         "02:00:5e:10:00:00:00:01",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "96590f8a-de19-4701-964c-628147636244",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    567,
			}},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.args.vmNicInst.Validate()
			if (err != nil) != testCase.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:maintidx
func TestCreate(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		vmNicInst *VMNic
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		wantErr     bool
	}{
		{
			name: "success1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).WithArgs("aTestNic2024061501_int0").
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
							}))
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta("INSERT INTO `vm_nics` (`created_at`,`updated_at`,`deleted_at`,`description`,`mac`,`net_dev`,`net_type`,`net_dev_type`,`switch_id`,`rate_limit`,`rate_in`,`rate_out`,`inst_bridge`,`inst_epair`,`id`,`name`,`config_id`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`,`config_id`"), //nolint:lll
				).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "a test nic description", "00:0f:5f:12:ab:fc", "",
					"VIRTIONET", "TAP", "c7f025f6-f6a0-4ed7-886d-a3cfa923d17a", false, 0, 0, "", "",
					sqlmock.AnyArg(), "aTestNic2024061501_int0", 11).
					WillReturnRows(
						sqlmock.NewRows([]string{"id", "name", "config_id"}).
							AddRow("0bd10557-f1ed-4998-a25d-fc883da80a03", "aTestNic2024061501_int0", 11),
					)
				mock.ExpectCommit()
			},
			args: args{&VMNic{
				ID:        "0bd10557-f1ed-4998-a25d-fc883da80a03",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "aTestNic2024061501_int0",
				Description: "a test nic description",
				Mac:         "00:0F:5F:12:ab:Fc",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "c7f025f6-f6a0-4ed7-886d-a3cfa923d17a",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    11,
			}},
		},
		{
			name: "success2",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).WithArgs("aTestNic2024061501_int0").
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
							}))
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta("INSERT INTO `vm_nics` (`created_at`,`updated_at`,`deleted_at`,`description`,`mac`,`net_dev`,`net_type`,`net_dev_type`,`switch_id`,`rate_limit`,`rate_in`,`rate_out`,`inst_bridge`,`inst_epair`,`id`,`name`,`config_id`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`,`config_id`"), //nolint:lll
				).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "a test nic description", "AUTO", "",
					"VIRTIONET", "TAP", "c7f025f6-f6a0-4ed7-886d-a3cfa923d17a", false, 0, 0, "", "",
					sqlmock.AnyArg(), "aTestNic2024061501_int0", 11).
					WillReturnRows(
						sqlmock.NewRows([]string{"id", "name", "config_id"}).
							AddRow("0bd10557-f1ed-4998-a25d-fc883da80a03", "aTestNic2024061501_int0", 11),
					)
				mock.ExpectCommit()
			},
			args: args{&VMNic{
				ID:        "0bd10557-f1ed-4998-a25d-fc883da80a03",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "aTestNic2024061501_int0",
				Description: "a test nic description",
				Mac:         "",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "c7f025f6-f6a0-4ed7-886d-a3cfa923d17a",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    11,
			}},
		},
		{
			name: "success3",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).WithArgs("aTestNic2024061501_int0").
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
							}))
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta("INSERT INTO `vm_nics` (`created_at`,`updated_at`,`deleted_at`,`description`,`mac`,`net_dev`,`net_type`,`net_dev_type`,`switch_id`,`rate_limit`,`rate_in`,`rate_out`,`inst_bridge`,`inst_epair`,`id`,`name`,`config_id`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`,`config_id`"), //nolint:lll
				).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "a test nic description", "00:0f:5f:12:ab:fc", "",
					"VIRTIONET", "TAP", "c7f025f6-f6a0-4ed7-886d-a3cfa923d17a", false, 0, 0, "", "",
					sqlmock.AnyArg(), "aTestNic2024061501_int0", 11).
					WillReturnRows(
						sqlmock.NewRows([]string{"id", "name", "config_id"}).
							AddRow("0bd10557-f1ed-4998-a25d-fc883da80a03", "aTestNic2024061501_int0", 11),
					)
				mock.ExpectCommit()
			},
			args: args{&VMNic{
				ID:        "0bd10557-f1ed-4998-a25d-fc883da80a03",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "aTestNic2024061501_int0",
				Description: "a test nic description",
				Mac:         "00:0F:5F:12:ab:Fc",
				NetDev:      "",
				NetType:     "",
				NetDevType:  "TAP",
				SwitchID:    "c7f025f6-f6a0-4ed7-886d-a3cfa923d17a",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    11,
			}},
		},
		{
			name: "success4",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).WithArgs("aTestNic2024061501_int0").
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
							}))
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta("INSERT INTO `vm_nics` (`created_at`,`updated_at`,`deleted_at`,`description`,`mac`,`net_dev`,`net_type`,`net_dev_type`,`switch_id`,`rate_limit`,`rate_in`,`rate_out`,`inst_bridge`,`inst_epair`,`id`,`name`,`config_id`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`,`config_id`"), //nolint:lll
				).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "a test nic description", "00:0f:5f:12:ab:fc", "",
					"VIRTIONET", "TAP", "c7f025f6-f6a0-4ed7-886d-a3cfa923d17a", false, 0, 0, "", "",
					sqlmock.AnyArg(), "aTestNic2024061501_int0", 11).
					WillReturnRows(
						sqlmock.NewRows([]string{"id", "name", "config_id"}).
							AddRow("0bd10557-f1ed-4998-a25d-fc883da80a03", "aTestNic2024061501_int0", 11),
					)
				mock.ExpectCommit()
			},
			args: args{&VMNic{
				ID:        "0bd10557-f1ed-4998-a25d-fc883da80a03",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "aTestNic2024061501_int0",
				Description: "a test nic description",
				Mac:         "00:0F:5F:12:ab:Fc",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "",
				SwitchID:    "c7f025f6-f6a0-4ed7-886d-a3cfa923d17a",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    11,
			}},
		},
		{
			name: "error1InvalidNic",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
			},
			args: args{&VMNic{
				ID:        "0bd10557-f1ed-4998-a25d-fc883da80a03",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "aTestNic2024061501_int0!",
				Description: "a test nic description",
				Mac:         "00:0F:5F:12:ab:Fc",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "c7f025f6-f6a0-4ed7-886d-a3cfa923d17a",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    11,
			}},
			wantErr: true,
		},
		{
			name: "error2NicExists",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("aTestNic2024061501_int0").
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
								"aTestNic2024061501_int0",
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
			},
			args: args{&VMNic{
				ID:        "0bd10557-f1ed-4998-a25d-fc883da80a03",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "aTestNic2024061501_int0",
				Description: "a test nic description",
				Mac:         "00:0F:5F:12:ab:Fc",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "c7f025f6-f6a0-4ed7-886d-a3cfa923d17a",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    11,
			}},
			wantErr: true,
		},
		{
			name: "error2NicExistsErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).WithArgs("aTestNic2024061501_int0").
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			args: args{&VMNic{
				ID:        "0bd10557-f1ed-4998-a25d-fc883da80a03",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "aTestNic2024061501_int0",
				Description: "a test nic description",
				Mac:         "00:0F:5F:12:ab:Fc",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "c7f025f6-f6a0-4ed7-886d-a3cfa923d17a",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    11,
			}},
			wantErr: true,
		},
		{
			name: "errDb1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).WithArgs("aTestNic2024061501_int0").
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
							}))
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta("INSERT INTO `vm_nics` (`created_at`,`updated_at`,`deleted_at`,`description`,`mac`,`net_dev`,`net_type`,`net_dev_type`,`switch_id`,`rate_limit`,`rate_in`,`rate_out`,`inst_bridge`,`inst_epair`,`id`,`name`,`config_id`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`,`config_id`"), //nolint:lll
				).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "a test nic description", "00:0f:5f:12:ab:fc", "",
					"VIRTIONET", "TAP", "c7f025f6-f6a0-4ed7-886d-a3cfa923d17a", false, 0, 0, "", "",
					sqlmock.AnyArg(), "aTestNic2024061501_int0", 11).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
				mock.ExpectRollback()
			},
			args: args{&VMNic{
				ID:        "0bd10557-f1ed-4998-a25d-fc883da80a03",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "aTestNic2024061501_int0",
				Description: "a test nic description",
				Mac:         "00:0F:5F:12:ab:Fc",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "c7f025f6-f6a0-4ed7-886d-a3cfa923d17a",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    11,
			}},
			wantErr: true,
		},
		{
			name: "errorNoRows",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).WithArgs("aTestNic2024061501_int0").
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
							}))
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta("INSERT INTO `vm_nics` (`created_at`,`updated_at`,`deleted_at`,`description`,`mac`,`net_dev`,`net_type`,`net_dev_type`,`switch_id`,`rate_limit`,`rate_in`,`rate_out`,`inst_bridge`,`inst_epair`,`id`,`name`,`config_id`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`,`config_id`"), //nolint:lll
				).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "a test nic description", "00:0f:5f:12:ab:fc", "",
					"VIRTIONET", "TAP", "c7f025f6-f6a0-4ed7-886d-a3cfa923d17a", false, 0, 0, "", "",
					sqlmock.AnyArg(), "aTestNic2024061501_int0", 11).
					WillReturnRows(
						sqlmock.NewRows([]string{"id", "name", "config_id"}),
					)
				mock.ExpectCommit()
			},
			args: args{&VMNic{
				ID:        "0bd10557-f1ed-4998-a25d-fc883da80a03",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "aTestNic2024061501_int0",
				Description: "a test nic description",
				Mac:         "00:0F:5F:12:ab:Fc",
				NetDev:      "",
				NetType:     "VIRTIONET",
				NetDevType:  "TAP",
				SwitchID:    "c7f025f6-f6a0-4ed7-886d-a3cfa923d17a",
				RateLimit:   false,
				RateIn:      0,
				RateOut:     0,
				InstBridge:  "",
				InstEpair:   "",
				ConfigID:    11,
			}},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(t.Name())

			testCase.mockClosure(testDB, mock)

			err := Create(testCase.args.vmNicInst)
			if (err != nil) != testCase.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, testCase.wantErr)
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

func Test_getMac(t *testing.T) {
	type args struct {
		thisNic VMNic
		vmName  string
		vmID    string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Auto",
			args: args{
				thisNic: VMNic{
					ID:   "f865c0c5-4a06-40c6-b066-2c10c81691d1",
					Name: "test2024080901_int0",
					Mac:  "AUTO",
				},
				vmName: "test2024080901",
				vmID:   "58b45d43-b1f1-47fd-a94a-d877a89ec34f",
			},
			want: "d9:81:b2:3d:a7:a2",
		},
		{
			name: "Specified",
			args: args{
				thisNic: VMNic{
					ID:   "f865c0c5-4a06-40c6-b066-2c10c81691d1",
					Name: "test2024080901_int0",
					Mac:  "00:22:44:AA:BB:CC",
				},
				vmName: "test2024080901",
				vmID:   "58b45d43-b1f1-47fd-a94a-d877a89ec34f",
			},
			want: "00:22:44:AA:BB:CC",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			config.Config.Network.Mac.Oui = "d9:81:b2"
			got := testCase.args.thisNic.GetMAC(testCase.args.vmID, testCase.args.vmName)

			if got != testCase.want {
				t.Errorf("GetMAC() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestVMNic_Build(t *testing.T) {
	type fields struct {
		ID          string
		CreatedAt   time.Time
		UpdatedAt   time.Time
		DeletedAt   gorm.DeletedAt
		Name        string
		Description string
		Mac         string
		NetDev      string
		NetType     string
		NetDevType  string
		SwitchID    string
		RateLimit   bool
		RateIn      uint64
		RateOut     uint64
		InstBridge  string
		InstEpair   string
		ConfigID    uint
	}

	tests := []struct {
		name            string
		mockCmdFunc     string
		hostIntStubFunc func() ([]net.Interface, error)
		fields          fields
		wantErr         bool
	}{
		{
			name:            "BadType",
			hostIntStubFunc: StubhostinterfacestestBuild,
			mockCmdFunc:     "TestVMNic_BuildSuccess",
			fields: fields{
				NetType: "garbage",
			},
			wantErr: true,
		},
		{
			name:            "SuccessNG",
			hostIntStubFunc: StubhostinterfacestestBuild,
			mockCmdFunc:     "TestVMNic_BuildSuccess",
			fields: fields{
				NetDevType: "NETGRAPH",
			},
			wantErr: false,
		},
		{
			name:            "TapNoNetDev",
			hostIntStubFunc: StubhostinterfacestestBuild,
			mockCmdFunc:     "TestVMNic_BuildSuccess",
			fields: fields{
				NetDev:     "",
				NetDevType: "TAP",
			},
			wantErr: true,
		},
		{
			name:            "TapError",
			hostIntStubFunc: StubhostinterfacestestBuild,
			mockCmdFunc:     "TestVMNic_BuildError",
			fields: fields{
				NetDev:     "tap0",
				NetDevType: "TAP",
			},
			wantErr: true,
		},
		{
			name:            "VmnetError",
			hostIntStubFunc: StubhostinterfacestestBuild,
			mockCmdFunc:     "TestVMNic_BuildError",
			fields: fields{
				NetDev:     "vmnet0",
				NetDevType: "VMNET",
			},
			wantErr: true,
		},
		{
			name:            "TapSuccess",
			hostIntStubFunc: StubhostinterfacestestBuild,
			mockCmdFunc:     "TestVMNic_BuildSuccess",
			fields: fields{
				NetDev:     "tap0",
				NetDevType: "TAP",
			},
			wantErr: false,
		},
		{
			name:            "VmnetSuccess",
			hostIntStubFunc: StubhostinterfacestestBuild,
			mockCmdFunc:     "TestVMNic_BuildSuccess",
			fields: fields{
				NetDev:     "vmnet0",
				NetDevType: "VMNET",
			},
			wantErr: false,
		},
		{
			name:            "TapAlreadyExists",
			hostIntStubFunc: StubhostinterfacestestBuild,
			mockCmdFunc:     "TestVMNic_BuildSuccess",
			fields: fields{
				NetDev:     "tap1",
				NetDevType: "TAP",
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			util.NetInterfacesFunc = testCase.hostIntStubFunc

			t.Cleanup(func() { util.NetInterfacesFunc = net.Interfaces })

			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)

			util.SetupTestCmd(fakeCommand)

			vmNic := &VMNic{
				ID:          testCase.fields.ID,
				CreatedAt:   testCase.fields.CreatedAt,
				UpdatedAt:   testCase.fields.UpdatedAt,
				DeletedAt:   testCase.fields.DeletedAt,
				Name:        testCase.fields.Name,
				Description: testCase.fields.Description,
				Mac:         testCase.fields.Mac,
				NetDev:      testCase.fields.NetDev,
				NetType:     testCase.fields.NetType,
				NetDevType:  testCase.fields.NetDevType,
				SwitchID:    testCase.fields.SwitchID,
				RateLimit:   testCase.fields.RateLimit,
				RateIn:      testCase.fields.RateIn,
				RateOut:     testCase.fields.RateOut,
				InstBridge:  testCase.fields.InstBridge,
				InstEpair:   testCase.fields.InstEpair,
				ConfigID:    testCase.fields.ConfigID,
			}

			err := vmNic.Build()
			if (err != nil) != testCase.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func Test_Demolish(t *testing.T) {
	type args struct {
		vmNic VMNic
	}

	tests := []struct {
		name            string
		hostIntStubFunc func() ([]net.Interface, error)
		mockCmdFunc     string
		args            args
		wantErr         bool
	}{
		{
			name:            "TapSuccessNoRateLimit",
			hostIntStubFunc: StubhostinterfacestestDemolish,
			mockCmdFunc:     "Test_cleanupIfNicSuccess",
			args: args{
				vmNic: VMNic{
					ID:          "8a060fe1-5b6a-4309-9f9b-bcdc0a4bad05",
					Name:        "fladsorp",
					Description: "a silly nic",
					Mac:         "AUTO",
					NetDev:      "tap0",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "21b42894-d0da-410f-b45c-afcdddae15b2",
					RateLimit:   false,
					RateIn:      0,
					RateOut:     0,
					InstBridge:  "",
					InstEpair:   "",
					ConfigID:    25,
				},
			},
		},
		{
			name:            "TapSuccessRateLimit",
			hostIntStubFunc: StubhostinterfacestestDemolish,
			mockCmdFunc:     "Test_cleanupIfNicSuccess",
			args: args{
				vmNic: VMNic{
					ID:          "8a060fe1-5b6a-4309-9f9b-bcdc0a4bad05",
					Name:        "fladsorp",
					Description: "a silly nic",
					Mac:         "AUTO",
					NetDev:      "tap0",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "21b42894-d0da-410f-b45c-afcdddae15b2",
					RateLimit:   true,
					RateIn:      500000,
					RateOut:     250000,
					InstBridge:  "bridge32766",
					InstEpair:   "epair32766",
					ConfigID:    25,
				},
			},
		},
		{
			name:            "Fail1",
			hostIntStubFunc: StubhostinterfacestestDemolish,
			mockCmdFunc:     "Test_cleanupIfNicFail1",
			args: args{
				vmNic: VMNic{
					ID:          "8a060fe1-5b6a-4309-9f9b-bcdc0a4bad05",
					Name:        "fladsorp",
					Description: "a silly nic",
					Mac:         "AUTO",
					NetDev:      "tap0",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "21b42894-d0da-410f-b45c-afcdddae15b2",
					RateLimit:   true,
					RateIn:      500000,
					RateOut:     250000,
					InstBridge:  "bridge32766",
					InstEpair:   "epair32766",
					ConfigID:    25,
				},
			},
			wantErr: true,
		},
		{
			name:            "FailAll",
			hostIntStubFunc: StubhostinterfacestestDemolish,
			mockCmdFunc:     "Test_cleanupIfNicFailAll",
			args: args{
				vmNic: VMNic{
					ID:          "8a060fe1-5b6a-4309-9f9b-bcdc0a4bad05",
					Name:        "fladsorp",
					Description: "a silly nic",
					Mac:         "AUTO",
					NetDev:      "tap0",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "21b42894-d0da-410f-b45c-afcdddae15b2",
					RateLimit:   true,
					RateIn:      500000,
					RateOut:     250000,
					InstBridge:  "bridge32766",
					InstEpair:   "epair32766",
					ConfigID:    25,
				},
			},
			wantErr: true,
		},
		{
			name:        "TapSuccessNG",
			mockCmdFunc: "Test_cleanupIfNicSuccess",
			args: args{
				vmNic: VMNic{
					ID:          "c593b77c-0a97-4eac-be8b-f7a6daffeb70",
					Name:        "bonk",
					Description: "BONK",
					Mac:         "AUTO",
					NetDev:      "bnet0,link2",
					NetType:     "VIRTIONET",
					NetDevType:  "NETGRAPH",
					SwitchID:    "8040b24b-5456-427b-9ff9-819ca756ca25",
					ConfigID:    25,
				},
			},
		},
		{
			name:            "BadType",
			hostIntStubFunc: StubhostinterfacestestDemolish,
			mockCmdFunc:     "Test_cleanupIfNicSuccess",
			args: args{
				vmNic: VMNic{
					ID:          "c593b77c-0a97-4eac-be8b-f7a6daffeb70",
					Name:        "bonk",
					Description: "BONK",
					Mac:         "AUTO",
					NetDev:      "bnet0,link2",
					NetType:     "VIRTIONET",
					NetDevType:  "garbage",
					SwitchID:    "8040b24b-5456-427b-9ff9-819ca756ca25",
					ConfigID:    25,
				},
			},
			wantErr: true,
		},
		{
			name:            "TapSuccessInterfaceDoesNotExist",
			hostIntStubFunc: StubhostinterfacestestDemolish,
			mockCmdFunc:     "Test_cleanupIfNicSuccess",
			args: args{
				vmNic: VMNic{
					ID:          "8a060fe1-5b6a-4309-9f9b-bcdc0a4bad05",
					Name:        "fladsorp",
					Description: "a silly nic",
					Mac:         "AUTO",
					NetDev:      "tap1",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "21b42894-d0da-410f-b45c-afcdddae15b2",
					RateLimit:   false,
					RateIn:      0,
					RateOut:     0,
					InstBridge:  "",
					InstEpair:   "",
					ConfigID:    25,
				},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			util.NetInterfacesFunc = testCase.hostIntStubFunc

			t.Cleanup(func() { util.NetInterfacesFunc = net.Interfaces })

			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)

			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := testCase.args.vmNic.Demolish()
			if (err != nil) != testCase.wantErr {
				t.Errorf("Demolish() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

// test helpers from here down

func Test_cleanupIfNicSuccess(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

func Test_cleanupIfNicFail1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) >= 2 && cmdWithArgs[1] == "/sbin/ifconfig" && cmdWithArgs[2] == "tap0" && cmdWithArgs[3] == "destroy" { //nolint:lll
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:cyclop
func Test_cleanupIfNicFailAll(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) >= 2 && cmdWithArgs[1] == "/sbin/ifconfig" && cmdWithArgs[2] == "tap0" && cmdWithArgs[3] == "destroy" { //nolint:lll
		os.Exit(1)
	}

	if len(cmdWithArgs) >= 2 && cmdWithArgs[1] == "/sbin/ifconfig" && cmdWithArgs[2] == "epair32766a" && cmdWithArgs[3] == "destroy" { //nolint:lll
		os.Exit(1)
	}

	if len(cmdWithArgs) >= 2 && cmdWithArgs[1] == "/sbin/ifconfig" && cmdWithArgs[2] == "bridge32766" && cmdWithArgs[3] == "destroy" { //nolint:lll
		os.Exit(1)
	}

	if len(cmdWithArgs) >= 2 && cmdWithArgs[1] == "/usr/sbin/ngctl" && strings.HasSuffix(cmdWithArgs[3], "a_pipe:") {
		os.Exit(1)
	}

	if len(cmdWithArgs) >= 2 && cmdWithArgs[1] == "/usr/sbin/ngctl" && strings.HasSuffix(cmdWithArgs[3], "b_pipe:") {
		os.Exit(1)
	}

	os.Exit(0)
}

func TestVMNic_BuildSuccess(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

func TestVMNic_BuildError(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

func StubhostinterfacestestDemolish() ([]net.Interface, error) {
	return []net.Interface{
		{
			Index:        1,
			MTU:          1500,
			Name:         "abc0",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x28, 0x73, 0x3e},
			Flags:        0x33,
		},
		{
			Index:        2,
			MTU:          1500,
			Name:         "def0",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x32, 0x6e, 0x6},
			Flags:        0x33,
		},
		{
			Index:        3,
			MTU:          16384,
			Name:         "lo0",
			HardwareAddr: net.HardwareAddr(nil),
			Flags:        0x35,
		},
		{
			Index:        4,
			MTU:          1500,
			Name:         "tap0",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x32, 0x6e, 0x6},
			Flags:        0x33,
		},
	}, nil
}

func StubhostinterfacestestBuild() ([]net.Interface, error) {
	return []net.Interface{
		{
			Index:        1,
			MTU:          1500,
			Name:         "abc0",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x28, 0x73, 0x3e},
			Flags:        0x33,
		},
		{
			Index:        2,
			MTU:          1500,
			Name:         "def0",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x32, 0x6e, 0x6},
			Flags:        0x33,
		},
		{
			Index:        3,
			MTU:          16384,
			Name:         "lo0",
			HardwareAddr: net.HardwareAddr(nil),
			Flags:        0x35,
		},
		{
			Index:        4,
			MTU:          1500,
			Name:         "tap1",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x32, 0x6e, 0x6},
			Flags:        0x33,
		},
	}, nil
}
