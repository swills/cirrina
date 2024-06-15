package vmnic

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-test/deep"
	"gorm.io/gorm"

	"cirrina/cirrina"
	"cirrina/cirrinad/cirrinadtest"
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
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := nicTypeValid(testCase.args.nicType); got != testCase.want {
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
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := nicDevTypeValid(testCase.args.nicDevType); got != testCase.want {
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
				instance = &singleton{ // prevents parallel testing
					vmNicDB: testDB,
				}
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
			},
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
			name: "testGetByName_error",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					vmNicDB: testDB,
				}
				mock.ExpectQuery("^SELECT \\* FROM `vm_nics` WHERE name = \\? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1$").
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			args:    args{name: "someNicName"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "testGetByName_notfound",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					vmNicDB: testDB,
				}
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
			},
			args:    args{name: "someRandomNic"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "testGetByName_emptyName",
			args: args{name: ""},
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					vmNicDB: testDB,
				}
			},
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
				instance = &singleton{ // prevents parallel testing
					vmNicDB: testDB,
				}
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
			},
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
	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("nicTest")
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
				instance = &singleton{ // prevents parallel testing
					vmNicDB: testDB,
				}
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
			},
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
			name: "testGetByID_error",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					vmNicDB: testDB,
				}
				mock.ExpectQuery("^SELECT \\* FROM `vm_nics` WHERE id = \\? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1$").
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			args:    args{id: "007af66e-9c05-41a6-832a-40273cce3bf8"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "testGetByID_notfound",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					vmNicDB: testDB,
				}
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
			},
			args:    args{id: "bb7061d5-c6a7-44d8-857f-e6f2f813d499"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "testGetByID_empty",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					vmNicDB: testDB,
				}
			},
			args:    args{id: ""},
			want:    nil,
			wantErr: true,
		},
	}
	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("nicTest")
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
		testCase := testCase // shadow to avoid loop variable capture
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
		testCase := testCase // shadow to avoid loop variable capture
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
		testCase := testCase // shadow to avoid loop variable capture

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

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields      fields
		wantErr     bool
	}{
		{
			name: "err1",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					vmNicDB: testDB,
				}
			},
			fields: fields{
				ID: "",
			},
			wantErr: true,
		},
		{
			name: "err2",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					vmNicDB: testDB,
				}
			},
			fields: fields{
				ID: "garbage",
			},
			wantErr: true,
		},
		{
			name: "err3",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					vmNicDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_nics` WHERE `vm_nics`.`id` = ?"),
				).
					WithArgs("00e58e32-b058-4617-a3db-a270e80ff801").
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
				mock.ExpectRollback()
			},
			fields: fields{
				ID: "00e58e32-b058-4617-a3db-a270e80ff801",
			},
			wantErr: true,
		},
		{
			name: "success1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					vmNicDB: testDB,
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
				Model:       testCase.fields.Model,
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

			testDB, mock := cirrinadtest.NewMockDB("isoTest")

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

			if err = db.Close(); err != nil {
				t.Error(err)
			}

			if err = mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
