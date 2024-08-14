package vm

import (
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
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

func Test_parsePsJSONOutput(t *testing.T) {
	type args struct {
		psJSONOutput []byte
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "valid1",
			args:    args{psJSONOutput: []byte(`{"process-information": {"process": [{"pid":"83821","terminal-name":"27 ","state":"I","cpu-time":"0:00.02","command":"/usr/local/bin/sudo /usr/bin/protect /usr/sbin/bhyve -U 50f994e3-5c30-4d4d-a330-f5c46106cffe -A -H -P -D -w -u -l bootrom,/usr/local/share/uefi-firmware/BHYVE_UEFI.fd,/var/tmp/cirrinad/state/something"}]}}`)}, //nolint:lll
			want:    "/usr/local/bin/sudo",
			wantErr: false,
		},
		{
			name:    "valid2",
			args:    args{psJSONOutput: []byte(`{"process-information": {"process": [{"pid":"71004","terminal-name":"28 ","state":"S","cpu-time":"0:00.03","command":"/usr/sbin/bhyve -U f5b761a1-8193-4db3-a914-b37edc848d29 -A -H -P -D -w -u -l bootrom,/usr/local/share/uefi-firmware/BHYVE_UEFI.fd,/var/tmp/cirrinad/state/something"}]}}`)}, //nolint:lll
			want:    "/usr/sbin/bhyve",
			wantErr: false,
		},
		{
			name:    "valid3",
			args:    args{psJSONOutput: []byte(`{"process-information": {"process": [{"pid":"85540","terminal-name":"28 ","state":"SC","cpu-time":"1:41.54","command":"bhyve: test2024010401 (bhyve)"}]}}`)}, //nolint:lll
			want:    "bhyve:",
			wantErr: false,
		},
		{
			name:    "invalid1",
			args:    args{psJSONOutput: []byte(``)},
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid2",
			args:    args{psJSONOutput: []byte(`{"process-information": 1}`)},
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid3",
			args:    args{psJSONOutput: []byte(`{"process-information": {"blah": 1}}`)},
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid4",
			args:    args{psJSONOutput: []byte(`{"process-information": {"process": [1,2]}}`)},
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid5",
			args:    args{psJSONOutput: []byte(`{"process-information": {"process": [1]}}`)},
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid6",
			args:    args{psJSONOutput: []byte(`{"process-information": {"process": [{"number": 1}]}}`)},
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid7",
			args:    args{psJSONOutput: []byte(`{"process-information": {"process": [{"pid":"83821","terminal-name":"27 ","state":"I","cpu-time":"0:00.02","command":123}]}}`)}, //nolint:lll
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid8",
			args:    args{psJSONOutput: []byte(`{"process-information": {"process": [{"pid":"83821","terminal-name":"27 ","state":"I","cpu-time":"0:00.02","command":""}]}}`)}, //nolint:lll
			want:    "",
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := parsePsJSONOutput(testCase.args.psJSONOutput)
			if (err != nil) != testCase.wantErr {
				t.Errorf("parsePsJSONOutput() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("parsePsJSONOutput() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func Test_findProcName(t *testing.T) {
	type args struct {
		pid uint32
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		want        string
	}{
		{
			name:        "Sleep",
			mockCmdFunc: "Test_findProcNameSleep",
			args:        args{pid: 12345},
			want:        "sleep",
		},
		{
			name:        "Error",
			mockCmdFunc: "Test_findProcNameError",
			args:        args{pid: 12345},
			want:        "",
		},
		{
			name:        "BadJson",
			mockCmdFunc: "Test_findProcNameBadJson",
			args:        args{pid: 12345},
			want:        "",
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)

			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			got := findProcName(testCase.args.pid)
			if got != testCase.want {
				t.Errorf("findProcName() = %v, want %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func TestGetAll(t *testing.T) {
	tests := []struct {
		name        string
		mockClosure func()
		want        []*VM
	}{
		{
			name: "Success1",
			mockClosure: func() {
				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
			},
			want: nil,
		},
		{
			name: "Success2",
			mockClosure: func() {
				testVM := VM{
					ID:          "7563edac-3a68-4950-9dec-ca53dd8c7fca",
					Name:        "",
					Description: "",
					Status:      "",
					Config: Config{
						Model: gorm.Model{
							ID: 2,
						},
						VMID: "7563edac-3a68-4950-9dec-ca53dd8c7fca",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}
				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM
			},
			want: []*VM{
				{
					ID:          "7563edac-3a68-4950-9dec-ca53dd8c7fca",
					Name:        "",
					Description: "",
					Status:      "",
					Config: Config{
						Model: gorm.Model{
							ID: 2,
						},
						VMID: "7563edac-3a68-4950-9dec-ca53dd8c7fca",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				},
			},
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()

			got := GetAll()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

//nolint:paralleltest
func TestGetByID(t *testing.T) {
	type args struct {
		id string
	}

	tests := []struct {
		name        string
		mockClosure func()
		args        args
		want        *VM
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func() {
				testVM := VM{
					ID:          "7563edac-3a68-4950-9dec-ca53dd8c7fca",
					Name:        "noName",
					Description: "no description",
					Status:      "STOPPED",
					Config: Config{
						Model: gorm.Model{
							ID: 2,
						},
						VMID: "7563edac-3a68-4950-9dec-ca53dd8c7fca",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}
				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM
			},
			args: args{id: "7563edac-3a68-4950-9dec-ca53dd8c7fca"},
			want: &VM{
				ID:          "7563edac-3a68-4950-9dec-ca53dd8c7fca",
				Name:        "noName",
				Description: "no description",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 2,
					},
					VMID: "7563edac-3a68-4950-9dec-ca53dd8c7fca",
					CPU:  2,
					Mem:  1024,
				},
				ISOs:  nil,
				Disks: nil,
			},
			wantErr: false,
		},
		{
			name: "Failure",
			mockClosure: func() {
				testVM := VM{
					ID:          "7563edac-3a68-4950-9dec-ca53dd8c7fca",
					Name:        "noName",
					Description: "no description",
					Status:      "STOPPED",
					Config: Config{
						Model: gorm.Model{
							ID: 2,
						},
						VMID: "7563edac-3a68-4950-9dec-ca53dd8c7fca",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}
				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM
			},
			args:    args{id: "3da3352e-e541-4327-87c3-85b15ce8ac2f"},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()

			got, err := GetByID(testCase.args.id)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestGetByName(t *testing.T) {
	type args struct {
		name string
	}

	tests := []struct {
		name        string
		mockClosure func()
		args        args
		want        *VM
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func() {
				testVM := VM{
					ID:          "bdcabf90-6852-4f81-8084-66bbfd72c3eb",
					Name:        "someVM",
					Description: "test description",
					Status:      "STOPPED",
					Config: Config{
						Model: gorm.Model{
							ID: 3,
						},
						VMID: "bdcabf90-6852-4f81-8084-66bbfd72c3eb",
						CPU:  2,
						Mem:  2048,
					},
					ISOs:  nil,
					Disks: nil,
				}
				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM
			},
			args: args{
				name: "someVM",
			},
			want: &VM{
				ID:          "bdcabf90-6852-4f81-8084-66bbfd72c3eb",
				Name:        "someVM",
				Description: "test description",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID: 3,
					},
					VMID: "bdcabf90-6852-4f81-8084-66bbfd72c3eb",
					CPU:  2,
					Mem:  2048,
				},
			},
			wantErr: false,
		},
		{
			name: "NotFound",
			mockClosure: func() {
				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
			},
			args: args{
				name: "someVM",
			},
			want:    &VM{},
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()

			got, err := GetByName(testCase.args.name)

			t.Parallel()

			if (err != nil) != testCase.wantErr {
				t.Errorf("GetByName() error = %v, wantErr %v", err, testCase.wantErr)

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
func Test_getUsedVncPorts(t *testing.T) {
	tests := []struct {
		name        string
		mockClosure func()
		want        []int
	}{
		{
			name: "NoneUsed",
			mockClosure: func() {
				testVM := VM{
					Status: "STOPPED",
				}
				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM
			},
			want: nil,
		},
		{
			name: "OneUsed",
			mockClosure: func() {
				testVM := VM{
					Status:  "RUNNING",
					VNCPort: 5900,
				}
				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM
			},
			want: []int{5900},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()

			got := getUsedVncPorts()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

//nolint:paralleltest
func Test_getUsedDebugPorts(t *testing.T) {
	tests := []struct {
		name        string
		mockClosure func()
		want        []int
	}{
		{
			name: "NoneUsed",
			mockClosure: func() {
				testVM := VM{
					Status: "STOPPED",
				}
				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM
			},
			want: nil,
		},
		{
			name: "OneUsed",
			mockClosure: func() {
				testVM := VM{
					Status:    "RUNNING",
					DebugPort: 3434,
				}
				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM
			},
			want: []int{3434},
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()

			got := getUsedDebugPorts()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

//nolint:paralleltest
func Test_getUsedNetPorts(t *testing.T) {
	createUpdateTime := time.Now()

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        []string
	}{
		{
			name: "NotRunning",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				testVM := VM{
					ID:          "c6a9839e-3a7d-488d-b3f5-379aeea8718d",
					Name:        "testVm",
					Description: "a test VM without a nic",
					Status:      "STOPPED",
					Config: Config{
						Model: gorm.Model{
							ID: 123,
						},
						VMID: "c6a9839e-3a7d-488d-b3f5-379aeea8718d",
					},
				}

				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM
			},
			want: nil,
		},
		{
			name: "GetNicsErr",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				testVM := VM{
					ID:          "c6a9839e-3a7d-488d-b3f5-379aeea8718d",
					Name:        "testVm",
					Description: "a test VM without a nic",
					Status:      "RUNNING",
					Config: Config{
						Model: gorm.Model{
							ID: 123,
						},
						VMID: "c6a9839e-3a7d-488d-b3f5-379aeea8718d",
					},
				}

				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM
			},
			want: nil,
		},
		{
			name: "NoneUsed",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				testVM := VM{
					ID:          "c6a9839e-3a7d-488d-b3f5-379aeea8718d",
					Name:        "testVm",
					Description: "a test VM without a nic",
					Status:      "RUNNING",
					Config: Config{
						Model: gorm.Model{
							ID: 123,
						},
						VMID: "c6a9839e-3a7d-488d-b3f5-379aeea8718d",
					},
				}

				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(123).
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
					}),
					)
			},
			want: nil,
		},
		{
			name: "OneUsed",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				testVM := VM{
					ID:          "c6a9839e-3a7d-488d-b3f5-379aeea8718d",
					Name:        "testVm",
					Description: "a test VM without a nic",
					Status:      "RUNNING",
					Config: Config{
						Model: gorm.Model{
							ID: 123,
						},
						VMID: "c6a9839e-3a7d-488d-b3f5-379aeea8718d",
					},
				}

				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(123).
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
					}).AddRow(
						"623a6641-fb63-4e58-bcca-cb7e569e8083",
						createUpdateTime,
						createUpdateTime,
						nil,
						"aNic",
						"a description",
						"00:11:22:33:44:55",
						"VIRTIONET",
						"TAP",
						"6093f6a1-82dc-4281-891f-549606f299dd",
						"tap0",
						false,
						0,
						0,
						nil,
						nil,
						123,
					),
					)
			},
			want: []string{"tap0"},
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("nicTest")

			testCase.mockClosure(testDB, mock)

			got := getUsedNetPorts()

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

//nolint:paralleltest
func Test_isNetPortUsed(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		netPort string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        bool
	}{
		{
			name: "PortIsUsed",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				testVM := VM{
					ID:          "c6a9839e-3a7d-488d-b3f5-379aeea8718d",
					Name:        "testVm",
					Description: "a test VM without a nic",
					Status:      "RUNNING",
					Config: Config{
						Model: gorm.Model{
							ID: 123,
						},
						VMID: "c6a9839e-3a7d-488d-b3f5-379aeea8718d",
					},
				}

				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(123).
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
					}).AddRow(
						"623a6641-fb63-4e58-bcca-cb7e569e8083",
						createUpdateTime,
						createUpdateTime,
						nil,
						"aNic",
						"a description",
						"00:11:22:33:44:55",
						"VIRTIONET",
						"TAP",
						"6093f6a1-82dc-4281-891f-549606f299dd",
						"tap0",
						false,
						0,
						0,
						nil,
						nil,
						123,
					),
					)
			},
			args: args{netPort: "tap0"},
			want: true,
		},
		{
			name: "PortIsNotUsed",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				testVM := VM{
					ID:          "c6a9839e-3a7d-488d-b3f5-379aeea8718d",
					Name:        "testVm",
					Description: "a test VM without a nic",
					Status:      "RUNNING",
					Config: Config{
						Model: gorm.Model{
							ID: 123,
						},
						VMID: "c6a9839e-3a7d-488d-b3f5-379aeea8718d",
					},
				}

				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(123).
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
					}).AddRow(
						"623a6641-fb63-4e58-bcca-cb7e569e8083",
						createUpdateTime,
						createUpdateTime,
						nil,
						"aNic",
						"a description",
						"00:11:22:33:44:55",
						"VIRTIONET",
						"TAP",
						"6093f6a1-82dc-4281-891f-549606f299dd",
						"tap0",
						false,
						0,
						0,
						nil,
						nil,
						123,
					),
					)
			},
			args: args{netPort: "tap1"},
			want: false,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("nicTest")

			testCase.mockClosure(testDB, mock)

			got := isNetPortUsed(testCase.args.netPort)

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

//nolint:paralleltest
func Test_findChildPid(t *testing.T) {
	type args struct {
		findPid uint32
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		want        uint32
	}{
		{
			name:        "Simple",
			mockCmdFunc: "Test_findChildPidSimple",
			args:        args{findPid: 54321},
			want:        12345,
		},
		{
			name:        "PgrepError",
			mockCmdFunc: "Test_findChildPidPgrepError",
			args:        args{findPid: 54321},
			want:        0,
		},
		{
			name:        "PgrepExtraFields",
			mockCmdFunc: "Test_findChildPidPgrepExtraFields",
			args:        args{findPid: 54321},
			want:        0,
		},
		{
			name:        "PgrepExtraLines",
			mockCmdFunc: "Test_findChildPidPgrepExtraLines",
			args:        args{findPid: 54321},
			want:        0,
		},
		{
			name:        "PgrepNonNumeric",
			mockCmdFunc: "Test_findChildPidPgrepNonNumeric",
			args:        args{findPid: 54321},
			want:        0,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)

			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			got := findChildPid(testCase.args.findPid)
			if got != testCase.want {
				t.Errorf("findChildPid() = %v, want %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest,gocognit
func Test_ensureComDevReadable(t *testing.T) {
	type args struct {
		comDev string
	}

	tests := []struct {
		name             string
		mockCmdFunc      string
		args             args
		wantErr          bool
		wantPath         bool
		wantPathErr      bool
		wantStat         bool
		wantStatErr      bool
		wantStatDir      bool
		wantUIDGidErr    bool
		wantDifferentUID bool
	}{
		{
			name:        "nonsense",
			args:        args{comDev: "someCrap"},
			wantErr:     true,
			wantPath:    false,
			wantPathErr: false,
		},
		{
			name:        "pathError",
			args:        args{comDev: "/dev/nmdm-test2001010101-com1-A"},
			wantErr:     true,
			wantPath:    false,
			wantPathErr: true,
		},
		{
			name:        "pathDoesNotExist",
			args:        args{comDev: "/dev/nmdm-test2001010101-com1-A"},
			wantErr:     true,
			wantPath:    false,
			wantPathErr: false,
		},
		{
			name:        "statErr",
			args:        args{comDev: "/dev/nmdm-test2001010101-com1-A"},
			wantErr:     true,
			wantPath:    true,
			wantPathErr: false,
			wantStat:    true,
			wantStatDir: false,
			wantStatErr: true,
		},
		{
			name:        "statDir",
			args:        args{comDev: "/dev/nmdm-test2001010101-com1-A"},
			wantErr:     true,
			wantPath:    true,
			wantPathErr: false,
			wantStat:    true,
			wantStatDir: true,
			wantStatErr: false,
		},
		{
			name:        "statNil",
			args:        args{comDev: "/dev/nmdm-test2001010101-com1-A"},
			wantErr:     true,
			wantPath:    true,
			wantPathErr: false,
			wantStat:    false,
			wantStatDir: false,
			wantStatErr: false,
		},
		{
			name:          "getUidGidErr",
			args:          args{comDev: "/dev/nmdm-test2001010101-com1-A"},
			wantErr:       true,
			wantPath:      true,
			wantPathErr:   false,
			wantStat:      true,
			wantStatDir:   false,
			wantStatErr:   false,
			wantUIDGidErr: true,
		},
		{
			name:             "ChownError",
			mockCmdFunc:      "Test_ensureComDevReadableChownError",
			args:             args{comDev: "/dev/nmdm-test2001010101-com1-A"},
			wantErr:          true,
			wantPath:         true,
			wantPathErr:      false,
			wantStat:         true,
			wantStatDir:      false,
			wantStatErr:      false,
			wantUIDGidErr:    false,
			wantDifferentUID: true,
		},
		{
			name:             "ChownOk",
			mockCmdFunc:      "Test_ensureComDevReadableChownOk",
			args:             args{comDev: "/dev/nmdm-test2001010101-com1-A"},
			wantErr:          false,
			wantPath:         true,
			wantPathErr:      false,
			wantStat:         true,
			wantStatDir:      false,
			wantStatErr:      false,
			wantUIDGidErr:    false,
			wantDifferentUID: true,
		},
		{
			name:             "NothingToDo",
			mockCmdFunc:      "Test_ensureComDevReadableChownOk",
			args:             args{comDev: "/dev/nmdm-test2001010101-com1-A"},
			wantErr:          false,
			wantPath:         true,
			wantPathErr:      false,
			wantStat:         true,
			wantStatDir:      false,
			wantStatErr:      false,
			wantUIDGidErr:    false,
			wantDifferentUID: false,
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)

			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			pathExistsFunc = func(_ string) (bool, error) {
				if testCase.wantPathErr {
					return true, errors.New("another error") //nolint:goerr113
				}

				if testCase.wantPath {
					return true, nil
				}

				return false, nil
			}

			t.Cleanup(func() { pathExistsFunc = util.PathExists })

			//nolint:nestif
			statFunc = func(_ string) (fs.FileInfo, error) {
				if testCase.wantStatErr {
					return nil, errors.New("some stat error") //nolint:goerr113
				}

				if testCase.wantStatDir {
					statSlash, err := os.Stat("/")
					if err != nil {
						t.Errorf("failed building fake stat: %s", err.Error())
					}

					return statSlash, nil
				} else if testCase.wantStat {
					randFile := filepath.Join("/tmp", RandomString(12))

					_, err := os.Create(randFile)
					if err != nil {
						t.Errorf("failed building fake stat: %s", err.Error())
					}

					err = os.Chmod(randFile, 0x0755)
					if err != nil {
						t.Errorf("failed building fake stat: %s", err.Error())
					}

					statFile, err := os.Stat(randFile)
					if err != nil {
						t.Errorf("failed building fake stat: %s", err.Error())
					}

					return statFile, nil
				}

				return nil, nil //nolint:nilnil
			}

			GetMyUIDGIDFunc = func() (uint32, uint32, error) {
				if testCase.wantUIDGidErr {
					return 0, 0, errors.New("some error") //nolint:goerr113
				}

				myUID, _, err := util.GetMyUIDGID()
				if err != nil {
					t.Errorf("unable to get my real uid when building test: %s", err.Error())
				}

				if testCase.wantDifferentUID {
					return myUID + 1, 0, nil // gid unused
				}

				return myUID, 0, nil // gid unused
			}

			err := ensureComDevReadable(testCase.args.comDev)
			if (err != nil) != testCase.wantErr {
				t.Errorf("ensureComDevReadable() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func Test_findChildProcName(t *testing.T) {
	type args struct {
		startPid uint32
		procName string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		want        uint32
	}{
		{
			name:        "OkDepthZero",
			mockCmdFunc: "Test_findChildProcNameOkDepthZero",
			args:        args{startPid: 12345, procName: "sleep"},
			want:        12345,
		},
		{
			name:        "OkDepthOne",
			mockCmdFunc: "Test_findChildProcNameOkDepthOne",
			args:        args{startPid: 54321, procName: "ls"},
			want:        12345,
		},
		{
			name:        "NotFound",
			mockCmdFunc: "Test_findChildProcNameNotFound",
			args:        args{startPid: 54321, procName: "something"},
			want:        0,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)

			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			got := findChildProcName(testCase.args.startPid, testCase.args.procName)
			if got != testCase.want {
				t.Errorf("findChildProcName() = %v, want %v", got, testCase.want)
			}
		})
	}
}

// test helpers from here down

func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}

	return string(s)
}

//nolint:paralleltest
func Test_findProcNameSleep(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 5 && cmdWithArgs[0] == "/bin/ps" && cmdWithArgs[1] == "--libxo" && cmdWithArgs[2] == "json" && cmdWithArgs[3] == "-p" { //nolint:lll
		fmt.Printf("{\"process-information\": {\"process\": [{\"pid\":\"12345\",\"terminal-name\":\"28 \",\"state\":\"SC+\",\"cpu-time\":\"0:00.00\",\"command\":\"sleep 1024\"}]}}\n") //nolint:lll,forbidigo
		os.Exit(0)
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_findProcNameError(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 5 && cmdWithArgs[0] == "/bin/ps" && cmdWithArgs[1] == "--libxo" && cmdWithArgs[2] == "json" && cmdWithArgs[3] == "-p" { //nolint:lll
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_findProcNameBadJson(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 5 && cmdWithArgs[0] == "/bin/ps" && cmdWithArgs[1] == "--libxo" && cmdWithArgs[2] == "json" && cmdWithArgs[3] == "-p" { //nolint:lll
		fmt.Printf("{\"process-information\": {\"process\": [{\"pid\":\"12345\",\"terminal-name\":\"28 \",\"state\":\"SC+\",\"cpu-time\":\"0:00.00\",\"comm") //nolint:lll,forbidigo
		os.Exit(0)
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_findChildPidSimple(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/bin/pgrep" && cmdWithArgs[1] == "-P" && cmdWithArgs[2] == "54321" { //nolint:lll
		fmt.Printf("12345\n") //nolint:forbidigo
		os.Exit(0)
	}

	for i, v := range os.Args {
		fmt.Printf("arg %d: %s\n", i, v) //nolint:forbidigo
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_findChildPidPgrepError(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/bin/pgrep" && cmdWithArgs[1] == "-P" && cmdWithArgs[2] == "54321" { //nolint:lll
		os.Exit(1)
	}

	for i, v := range os.Args {
		fmt.Printf("arg %d: %s\n", i, v) //nolint:forbidigo
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_findChildPidPgrepExtraFields(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/bin/pgrep" && cmdWithArgs[1] == "-P" && cmdWithArgs[2] == "54321" { //nolint:lll
		fmt.Printf("12345 garbage\n") //nolint:forbidigo
		os.Exit(0)
	}

	for i, v := range os.Args {
		fmt.Printf("arg %d: %s\n", i, v) //nolint:forbidigo
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_findChildPidPgrepExtraLines(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/bin/pgrep" && cmdWithArgs[1] == "-P" && cmdWithArgs[2] == "54321" { //nolint:lll
		fmt.Printf("12345\n12345\n") //nolint:forbidigo
		os.Exit(0)
	}

	for i, v := range os.Args {
		fmt.Printf("arg %d: %s\n", i, v) //nolint:forbidigo
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_findChildPidPgrepNonNumeric(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/bin/pgrep" && cmdWithArgs[1] == "-P" && cmdWithArgs[2] == "54321" { //nolint:lll
		fmt.Printf("12345a\n") //nolint:forbidigo
		os.Exit(0)
	}

	for i, v := range os.Args {
		fmt.Printf("arg %d: %s\n", i, v) //nolint:forbidigo
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_ensureComDevReadableChownError(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/usr/sbin/chown" && cmdWithArgs[2] == "/dev/nmdm-test2001010101-com1-B" { //nolint:lll
		os.Exit(1)
	}

	for i, v := range os.Args {
		fmt.Printf("arg %d: %s\n", i, v) //nolint:forbidigo
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_ensureComDevReadableChownOK(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if len(cmdWithArgs) == 3 && cmdWithArgs[0] == "/usr/sbin/chown" && cmdWithArgs[2] == "/dev/nmdm-test2001010101-com1-B" { //nolint:lll
		os.Exit(0)
	}

	for i, v := range os.Args {
		fmt.Printf("arg %d: %s\n", i, v) //nolint:forbidigo
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_findChildProcNameOkDepthZero(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	// findProcName pid = 12345
	if len(cmdWithArgs) == 5 && cmdWithArgs[0] == "/bin/ps" && cmdWithArgs[1] == "--libxo" && cmdWithArgs[2] == "json" && cmdWithArgs[3] == "-p" { //nolint:lll
		fmt.Printf("{\"process-information\": {\"process\": [{\"pid\":\"12345\",\"terminal-name\":\"28 \",\"state\":\"SC+\",\"cpu-time\":\"0:00.00\",\"command\":\"sleep 1024\"}]}}\n") //nolint:lll,forbidigo
		os.Exit(0)
	}

	for i, v := range os.Args {
		fmt.Printf("arg %d: %s\n", i, v) //nolint:forbidigo
	}

	os.Exit(1)
}

//nolint:paralleltest,cyclop
func Test_findChildProcNameOkDepthOne(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	// findProcName pid = 54321
	if len(cmdWithArgs) == 5 && cmdWithArgs[0] == "/bin/ps" && cmdWithArgs[1] == "--libxo" && cmdWithArgs[2] == "json" && cmdWithArgs[3] == "-p" && cmdWithArgs[4] == "54321" { //nolint:lll
		fmt.Printf("{\"process-information\": {\"process\": [{\"pid\":\"54321\",\"terminal-name\":\"28 \",\"state\":\"SC+\",\"cpu-time\":\"0:00.00\",\"command\":\"sleep 1024\"}]}}\n") //nolint:lll,forbidigo
		os.Exit(0)
	}

	// findChildPid pid = 54321
	if len(cmdWithArgs) == 4 && cmdWithArgs[1] == "/bin/pgrep" && cmdWithArgs[2] == "-P" && cmdWithArgs[3] == "54321" { //nolint:lll
		fmt.Printf("12345\n") //nolint:forbidigo
		os.Exit(0)
	}

	// findProcName pid = 12345
	if len(cmdWithArgs) == 5 && cmdWithArgs[0] == "/bin/ps" && cmdWithArgs[1] == "--libxo" && cmdWithArgs[2] == "json" && cmdWithArgs[3] == "-p" && cmdWithArgs[4] == "12345" { //nolint:lll
		fmt.Printf("{\"process-information\": {\"process\": [{\"pid\":\"12345\",\"terminal-name\":\"28 \",\"state\":\"SC+\",\"cpu-time\":\"0:00.00\",\"command\":\"ls something\"}]}}\n") //nolint:lll,forbidigo
		os.Exit(0)
	}

	fmt.Printf("cmdWithArgs: %+v", cmdWithArgs) //nolint:forbidigo

	for i, v := range os.Args {
		fmt.Printf("arg %d: \"%s\" ", i, v) //nolint:forbidigo
	}

	fmt.Printf("\n") //nolint:forbidigo

	os.Exit(1)
}

//nolint:paralleltest,cyclop
func Test_findChildProcNameNotFound(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	// findProcName pid = 54321
	if len(cmdWithArgs) == 5 && cmdWithArgs[0] == "/bin/ps" && cmdWithArgs[1] == "--libxo" && cmdWithArgs[2] == "json" && cmdWithArgs[3] == "-p" && cmdWithArgs[4] == "54321" { //nolint:lll
		fmt.Printf("{\"process-information\": {\"process\": [{\"pid\":\"54321\",\"terminal-name\":\"28 \",\"state\":\"SC+\",\"cpu-time\":\"0:00.00\",\"command\":\"sleep 1024\"}]}}\n") //nolint:lll,forbidigo
		os.Exit(0)
	}

	// findChildPid pid = 54321
	if len(cmdWithArgs) == 4 && cmdWithArgs[1] == "/bin/pgrep" && cmdWithArgs[2] == "-P" && cmdWithArgs[3] == "54321" { //nolint:lll
		fmt.Printf("12345\n") //nolint:forbidigo
		os.Exit(0)
	}

	// findProcName pid = 12345
	if len(cmdWithArgs) == 5 && cmdWithArgs[0] == "/bin/ps" && cmdWithArgs[1] == "--libxo" && cmdWithArgs[2] == "json" && cmdWithArgs[3] == "-p" && cmdWithArgs[4] == "12345" { //nolint:lll
		fmt.Printf("{\"process-information\": {\"process\": [{\"pid\":\"12345\",\"terminal-name\":\"28 \",\"state\":\"SC+\",\"cpu-time\":\"0:00.00\",\"command\":\"ls something\"}]}}\n") //nolint:lll,forbidigo
		os.Exit(0)
	}

	// findChildPid pid = 12345
	if len(cmdWithArgs) == 4 && cmdWithArgs[1] == "/bin/pgrep" && cmdWithArgs[2] == "-P" && cmdWithArgs[3] == "54321" { //nolint:lll
		os.Exit(0)
	}

	fmt.Printf("cmdWithArgs: %+v", cmdWithArgs) //nolint:forbidigo

	for i, v := range os.Args {
		fmt.Printf("arg %d: \"%s\" ", i, v) //nolint:forbidigo
	}

	fmt.Printf("\n") //nolint:forbidigo

	os.Exit(1)
}
