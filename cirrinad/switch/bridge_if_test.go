package vmswitch

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-test/deep"
	"gorm.io/gorm"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vmnic"
)

//nolint:paralleltest
func TestGetAllIfBridges(t *testing.T) {
	tests := []struct {
		name        string
		mockCmdFunc string
		want        []string
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestGetAllIfBridgesSuccess1",
			want:        []string{"bridge0", "bridge1"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "TestGetAllIfBridgesError1",
			want:        nil,
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			got, err := getAllIfSwitches()
			if (err != nil) != testCase.wantErr {
				t.Errorf("getAllIfSwitches() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

//nolint:paralleltest
func Test_getIfBridgeMembers(t *testing.T) {
	type args struct {
		name string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		want        []string
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_getIfBridgeMembersSuccess1",
			args:        args{name: "bridge0"},
			want:        []string{"em0"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_getIfBridgeMembersError1",
			args:        args{name: "bridge0"},
			want:        nil,
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			got, err := getIfBridgeMembers(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("getIfBridgeMembers() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

//nolint:paralleltest
func Test_createIfBridge(t *testing.T) {
	type args struct {
		name string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_createIfBridgeSuccess1",
			args:        args{name: "bridge0"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_createIfBridgeSuccess1",
			args:        args{name: ""},
			wantErr:     true,
		},
		{
			name:        "error2",
			mockCmdFunc: "Test_createIfBridgeSuccess1",
			args:        args{name: "garbage"},
			wantErr:     true,
		},
		{
			name:        "error3",
			mockCmdFunc: "Test_createIfBridgeError1",
			args:        args{name: "bridge0"},
			wantErr:     true,
		},
		{
			name:        "error4",
			mockCmdFunc: "Test_createIfBridgeError4",
			args:        args{name: "bridge0"},
			wantErr:     true,
		},
		{
			name:        "error5",
			mockCmdFunc: "Test_createIfBridgeError5",
			args:        args{name: "bridge0"},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := createIfBridge(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("createIfBridge() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func Test_actualIfBridgeCreate(t *testing.T) {
	type args struct {
		name string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_actualIfBridgeCreateSuccess1",
			args:        args{name: "bridge0"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_actualIfBridgeCreateError1",
			args:        args{name: "bridge0"},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := actualIfBridgeCreate(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("actualIfBridgeCreate() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func Test_bridgeIfDeleteAllMembers(t *testing.T) {
	type args struct {
		name string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_bridgeIfDeleteAllMembersSuccess1",
			args:        args{name: "bridge0"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_bridgeIfDeleteAllMembersError1",
			args:        args{name: "bridge0"},
			wantErr:     true,
		},
		{
			name:        "error2",
			mockCmdFunc: "Test_bridgeIfDeleteAllMembersError2",
			args:        args{name: "bridge0"},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := bridgeIfDeleteAllMembers(testCase.args.name)

			if (err != nil) != testCase.wantErr {
				t.Errorf("bridgeIfDeleteAllMembers() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func Test_bridgeIfDeleteMember(t *testing.T) {
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
			mockCmdFunc: "Test_bridgeIfDeleteMemberSuccess1",
			args:        args{bridgeName: "bridge0", memberName: "em0"},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "Test_bridgeIfDeleteMemberError1",
			args:        args{bridgeName: "bridge0", memberName: "em0"},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := switchIfDeleteMember(testCase.args.bridgeName, testCase.args.memberName)
			if (err != nil) != testCase.wantErr {
				t.Errorf("switchIfDeleteMember() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func TestCreateIfBridgeWithMembers(t *testing.T) {
	type args struct {
		bridgeName    string
		bridgeMembers []string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestCreateIfBridgeWithMembersSuccess1",
			args:        args{bridgeName: "bridge0", bridgeMembers: []string{"tap0"}},
			wantErr:     false,
		},
		{
			name:        "error1",
			mockCmdFunc: "TestCreateIfBridgeWithMembersError1",
			args:        args{bridgeName: "bridge0", bridgeMembers: []string{"tap0"}},
			wantErr:     true,
		},
		{
			name:        "error2",
			mockCmdFunc: "TestCreateIfBridgeWithMembersError2",
			args:        args{bridgeName: "bridge0", bridgeMembers: []string{"tap0"}},
			wantErr:     true,
		},
		{
			name:        "error3",
			mockCmdFunc: "TestCreateIfBridgeWithMembersError3",
			args:        args{bridgeName: "bridge0", bridgeMembers: []string{"tap0"}},
			wantErr:     true,
		},
		{
			name:        "emptyBridgeName",
			mockCmdFunc: "TestCreateIfBridgeWithMembersError3",
			args:        args{bridgeName: "", bridgeMembers: []string{"tap0"}},
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := createIfSwitchWithMembers(testCase.args.bridgeName, testCase.args.bridgeMembers)
			if (err != nil) != testCase.wantErr {
				t.Errorf("createIfSwitchWithMembers() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func TestGetDummyBridgeName(t *testing.T) {
	tests := []struct {
		name        string
		mockCmdFunc string
		want        string
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestGetDummyBridgeNameSuccess1",
			want:        "bridge32767",
		},
		{
			name:        "success2",
			mockCmdFunc: "TestGetDummyBridgeNameSuccess2",
			want:        "bridge32765",
		},
		{
			name:        "error1",
			mockCmdFunc: "TestGetDummyBridgeNameError1",
			want:        "",
		},
		{
			name:        "error2",
			mockCmdFunc: "TestGetDummyBridgeNameError2",
			want:        "",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			got := getDummyBridgeName()
			if got != testCase.want {
				t.Errorf("getDummyBridgeName() = %v, want %v", got, testCase.want)
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
		{
			name: "createEpairErr",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
			},
			mockCmdFunc: "Test_setupVMNicRateLimitCreateEpairErr",
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
			name: "saveErr",
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
			name: "setRateLimitError",
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
			mockCmdFunc: "Test_setupVMNicRateLimit_setRateLimitErr",
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
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			testDB, mock := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mock)

			got, err := setupVMNicRateLimit(&testCase.args.vmNic)
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

//nolint:paralleltest,maintidx
func Test_unsetVMNicRateLimit(t *testing.T) {
	type args struct {
		vmNic *vmnic.VMNic
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name: "SuccessRateLimited",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(661, "a NIC also", "", "", "AUTO", "someOtherNic", "tap0", "TAP",
						"VIRTIONET", 400000000, true, 100000000, "04afeae3-09fc-4d45-9c00-81c3f785f1c1", sqlmock.AnyArg(),
						"7c1887c6-bdf1-4e2d-a56f-4ac2e36d1303").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			mockCmdFunc: "Test_unsetVMNicRateLimitSuccess",
			args: args{
				vmNic: &vmnic.VMNic{
					ID:          "7c1887c6-bdf1-4e2d-a56f-4ac2e36d1303",
					Name:        "someOtherNic",
					Description: "a NIC also",
					Mac:         "AUTO",
					NetDev:      "tap0",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "04afeae3-09fc-4d45-9c00-81c3f785f1c1",
					RateLimit:   true,
					RateIn:      400000000,
					RateOut:     100000000,
					InstBridge:  "bridge32767",
					InstEpair:   "epair32767",
					ConfigID:    661,
				},
			},
			wantErr: false,
		},
		{
			name: "SuccessNotRateLimited",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
			},
			mockCmdFunc: "Test_unsetVMNicRateLimitSuccess",
			args: args{
				vmNic: &vmnic.VMNic{
					ID:          "7c1887c6-bdf1-4e2d-a56f-4ac2e36d1303",
					Name:        "someOtherNic",
					Description: "a NIC also",
					Mac:         "AUTO",
					NetDev:      "tap0",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "04afeae3-09fc-4d45-9c00-81c3f785f1c1",
					RateLimit:   true,
					RateIn:      400000000,
					RateOut:     100000000,
					InstBridge:  "",
					InstEpair:   "",
					ConfigID:    661,
				},
			},
			wantErr: false,
		},
		{
			name: "SuccessRateLimited_destroyIfSwitchErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(661, "a NIC also", "", "", "AUTO", "someOtherNic", "tap0", "TAP",
						"VIRTIONET", 400000000, true, 100000000, "04afeae3-09fc-4d45-9c00-81c3f785f1c1", sqlmock.AnyArg(),
						"7c1887c6-bdf1-4e2d-a56f-4ac2e36d1303").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			mockCmdFunc: "Test_unsetVMNicRateLimitSuccessRateLimited_destroyIfSwitchErr",
			args: args{
				vmNic: &vmnic.VMNic{
					ID:          "7c1887c6-bdf1-4e2d-a56f-4ac2e36d1303",
					Name:        "someOtherNic",
					Description: "a NIC also",
					Mac:         "AUTO",
					NetDev:      "tap0",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "04afeae3-09fc-4d45-9c00-81c3f785f1c1",
					RateLimit:   true,
					RateIn:      400000000,
					RateOut:     100000000,
					InstBridge:  "bridge32767",
					InstEpair:   "epair32767",
					ConfigID:    661,
				},
			},
			wantErr: false,
		},
		{
			name: "SuccessRateLimited_shutdownEpairAErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(661, "a NIC also", "", "", "AUTO", "someOtherNic", "tap0", "TAP",
						"VIRTIONET", 400000000, true, 100000000, "04afeae3-09fc-4d45-9c00-81c3f785f1c1", sqlmock.AnyArg(),
						"7c1887c6-bdf1-4e2d-a56f-4ac2e36d1303").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			mockCmdFunc: "Test_unsetVMNicRateLimitSuccessRateLimited_shutdownEpairAErr",
			args: args{
				vmNic: &vmnic.VMNic{
					ID:          "7c1887c6-bdf1-4e2d-a56f-4ac2e36d1303",
					Name:        "someOtherNic",
					Description: "a NIC also",
					Mac:         "AUTO",
					NetDev:      "tap0",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "04afeae3-09fc-4d45-9c00-81c3f785f1c1",
					RateLimit:   true,
					RateIn:      400000000,
					RateOut:     100000000,
					InstBridge:  "bridge32767",
					InstEpair:   "epair32767",
					ConfigID:    661,
				},
			},
			wantErr: false,
		},
		{
			name: "SuccessRateLimited_shutdownEpairBErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(661, "a NIC also", "", "", "AUTO", "someOtherNic", "tap0", "TAP",
						"VIRTIONET", 400000000, true, 100000000, "04afeae3-09fc-4d45-9c00-81c3f785f1c1", sqlmock.AnyArg(),
						"7c1887c6-bdf1-4e2d-a56f-4ac2e36d1303").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			mockCmdFunc: "Test_unsetVMNicRateLimitSuccessRateLimited_shutdownEpairBErr",
			args: args{
				vmNic: &vmnic.VMNic{
					ID:          "7c1887c6-bdf1-4e2d-a56f-4ac2e36d1303",
					Name:        "someOtherNic",
					Description: "a NIC also",
					Mac:         "AUTO",
					NetDev:      "tap0",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "04afeae3-09fc-4d45-9c00-81c3f785f1c1",
					RateLimit:   true,
					RateIn:      400000000,
					RateOut:     100000000,
					InstBridge:  "bridge32767",
					InstEpair:   "epair32767",
					ConfigID:    661,
				},
			},
			wantErr: false,
		},
		{
			name: "SuccessRateLimited_DestroyEpairErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(661, "a NIC also", "", "", "AUTO", "someOtherNic", "tap0", "TAP",
						"VIRTIONET", 400000000, true, 100000000, "04afeae3-09fc-4d45-9c00-81c3f785f1c1", sqlmock.AnyArg(),
						"7c1887c6-bdf1-4e2d-a56f-4ac2e36d1303").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			mockCmdFunc: "Test_unsetVMNicRateLimitSuccessRateLimited_DestroyEpairErr",
			args: args{
				vmNic: &vmnic.VMNic{
					ID:          "7c1887c6-bdf1-4e2d-a56f-4ac2e36d1303",
					Name:        "someOtherNic",
					Description: "a NIC also",
					Mac:         "AUTO",
					NetDev:      "tap0",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "04afeae3-09fc-4d45-9c00-81c3f785f1c1",
					RateLimit:   true,
					RateIn:      400000000,
					RateOut:     100000000,
					InstBridge:  "bridge32767",
					InstEpair:   "epair32767",
					ConfigID:    661,
				},
			},
			wantErr: false,
		},
		{
			name: "RateLimitedSaveErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(661, "a NIC also", "", "", "AUTO", "someOtherNic", "tap0", "TAP",
						"VIRTIONET", 400000000, true, 100000000, "04afeae3-09fc-4d45-9c00-81c3f785f1c1", sqlmock.AnyArg(),
										"7c1887c6-bdf1-4e2d-a56f-4ac2e36d1303").
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			mockCmdFunc: "Test_unsetVMNicRateLimitSuccess",
			args: args{
				vmNic: &vmnic.VMNic{
					ID:          "7c1887c6-bdf1-4e2d-a56f-4ac2e36d1303",
					Name:        "someOtherNic",
					Description: "a NIC also",
					Mac:         "AUTO",
					NetDev:      "tap0",
					NetType:     "VIRTIONET",
					NetDevType:  "TAP",
					SwitchID:    "04afeae3-09fc-4d45-9c00-81c3f785f1c1",
					RateLimit:   true,
					RateIn:      400000000,
					RateOut:     100000000,
					InstBridge:  "bridge32767",
					InstEpair:   "epair32767",
					ConfigID:    661,
				},
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			testDB, mock := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mock)

			err := unsetVMNicRateLimit(testCase.args.vmNic)
			if (err != nil) != testCase.wantErr {
				t.Errorf("unsetVMNicRateLimit() error = %v, wantErr %v", err, testCase.wantErr)
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
func TestGetAllIfBridgesSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Printf("bridge0\nbridge1\nthis is test garbage\n") //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func TestGetAllIfBridgesError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_getIfBridgeMembersSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

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
	os.Exit(0)
}

//nolint:paralleltest
func Test_getIfBridgeMembersError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_createIfBridgeSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_createIfBridgeError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_createIfBridgeError4(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Printf("bridge0\nbridge1\n") //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func Test_createIfBridgeError5(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[2] == "create" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_actualIfBridgeCreateSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_actualIfBridgeCreateError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[2] == "create" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_bridgeIfDeleteAllMembersSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_bridgeIfDeleteAllMembersError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_bridgeIfDeleteAllMembersError2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "bridge0" {
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
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/sbin/ifconfig" && cmdWithArgs[3] == "deletem" {
		os.Exit(1)
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_bridgeIfDeleteMemberSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_bridgeIfDeleteMemberError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestCreateIfBridgeWithMembersSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestCreateIfBridgeWithMembersError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestCreateIfBridgeWithMembersError2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if cmdWithArgs[0] == "/sbin/ifconfig" && cmdWithArgs[1] == "bridge0" {
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
		os.Exit(0)
	}

	if cmdWithArgs[1] == "/sbin/ifconfig" && cmdWithArgs[3] == "deletem" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestCreateIfBridgeWithMembersError3(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	for _, v := range os.Args {
		if v == "addm" {
			os.Exit(1)
		}
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestGetDummyBridgeNameSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestGetDummyBridgeNameSuccess2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Printf("bridge32767\nbridge32766\nbridge1\nbridge0\n") //nolint:forbidigo

	os.Exit(0)
}

//nolint:paralleltest
func TestGetDummyBridgeNameError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestGetDummyBridgeNameError2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	for i := 32767; i >= 0; i-- {
		fmt.Printf("bridge%d\n", i) //nolint:forbidigo
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_unsetVMNicRateLimitSuccess(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_unsetVMNicRateLimitSuccessRateLimited_destroyIfSwitchErr(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	for _, v := range os.Args {
		if strings.Contains(v, "ifconfig") {
			os.Exit(1)
		}
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_unsetVMNicRateLimitSuccessRateLimited_shutdownEpairAErr(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	for _, v := range os.Args {
		if strings.Contains(v, "epair") && strings.HasSuffix(v, "a_pipe:") {
			os.Exit(1)
		}
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_unsetVMNicRateLimitSuccessRateLimited_shutdownEpairBErr(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	for _, v := range os.Args {
		if strings.Contains(v, "epair") && strings.HasSuffix(v, "b_pipe:") {
			os.Exit(1)
		}
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_unsetVMNicRateLimitSuccessRateLimited_DestroyEpairErr(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	for i, v := range os.Args {
		if strings.Contains(v, "destroy") && i > 1 && strings.HasPrefix(os.Args[i-1], "epair") {
			os.Exit(1)
		}
	}

	os.Exit(0)
}
