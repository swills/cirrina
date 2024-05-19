package disk

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/util"
)

//nolint:paralleltest
func TestGetAllZfsVolumesSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	successOutput := `cirrinad0/disk/test2024051402_hd0
cirrinad0/disk/test2024051402_hd1
cirrinad0/disk/test2024051402_hd2
`

	fmt.Print(successOutput) //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func TestGetAllZfsVolumesErrorFields(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Print("forced error wrong number of fields") //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func TestGetAllZfsVolumesErrorExec(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Print("forced error exec") //nolint:forbidigo
	os.Exit(1)
}

//nolint:paralleltest
func TestGetAllZfsVolumes(t *testing.T) {
	tests := []struct {
		name        string
		mockCmdFunc string
		want        []string
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestGetAllZfsVolumesSuccess1",
			want: []string{
				"cirrinad0/disk/test2024051402_hd0",
				"cirrinad0/disk/test2024051402_hd1",
				"cirrinad0/disk/test2024051402_hd2",
			},
			wantErr: false,
		},
		{
			name:        "errorExec",
			mockCmdFunc: "TestGetAllZfsVolumesErrorExec",
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "errorFields",
			mockCmdFunc: "TestGetAllZfsVolumesErrorFields",
			want:        nil,
			wantErr:     false,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			got, err := GetAllZfsVolumes()
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetAllZfsVolumes() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("GetAllZfsVolumes() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func TestGetZfsVolumeSizeErrorParse(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]
	cmd := cmdWithArgs[0]

	if cmd != "/sbin/zfs" {
		os.Exit(1)
	}

	if cmdWithArgs[len(cmdWithArgs)-1] != "someVolumeName" {
		os.Exit(1)
	}

	fmt.Print("four\n") //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func TestGetZfsVolumeSizeErrorFields(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	cmd := cmdWithArgs[0]
	if cmd != "/sbin/zfs" {
		os.Exit(1)
	}

	if cmdWithArgs[len(cmdWithArgs)-1] != "someVolumeName" {
		os.Exit(1)
	}

	fmt.Print("21474 83648\n") //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func TestGetZfsVolumeSizeErrorExit(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestGetZfsVolumeSizeSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	cmd := cmdWithArgs[0]
	if cmd != "/sbin/zfs" {
		os.Exit(1)
	}

	if cmdWithArgs[len(cmdWithArgs)-1] != "someVolumeName" {
		os.Exit(1)
	}

	fmt.Print("2147483648\n") //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func TestGetZfsVolumeSize(t *testing.T) {
	type args struct {
		volumeName string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		want        uint64
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestGetZfsVolumeSizeSuccess1",
			args:        args{volumeName: "someVolumeName"},
			want:        2147483648,
			wantErr:     false,
		},
		{
			name:        "errorExit",
			mockCmdFunc: "TestGetZfsVolumeSizeErrorExit",
			args:        args{volumeName: "someVolumeName"},
			want:        0,
			wantErr:     true,
		},
		{
			name:        "errorExit",
			mockCmdFunc: "TestGetZfsVolumeSizeErrorFields",
			args:        args{volumeName: "someVolumeName"},
			want:        0,
			wantErr:     true,
		},
		{
			name:        "errorParse",
			mockCmdFunc: "TestGetZfsVolumeSizeErrorParse",
			args:        args{volumeName: "someVolumeName"},
			want:        0,
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

			got, err := GetZfsVolumeSize(testCase.args.volumeName)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetZfsVolumeSize() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("GetZfsVolumeSize() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func TestGetZfsVolumeUsageSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	cmd := cmdWithArgs[0]
	if cmd != "/sbin/zfs" {
		os.Exit(1)
	}

	if cmdWithArgs[len(cmdWithArgs)-1] != "someVolumeName" {
		os.Exit(1)
	}

	fmt.Print("662609920\n") //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func TestGetZfsVolumeUsageErrorExec(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestGetZfsVolumeUsageErrorParse(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Printf("number") //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func TestGetZfsVolumeUsage(t *testing.T) {
	type args struct {
		volumeName string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		want        uint64
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestGetZfsVolumeUsageSuccess1",
			args:        args{volumeName: "someVolumeName"},
			want:        662609920,
			wantErr:     false,
		},
		{
			name:        "errorExec",
			mockCmdFunc: "TestGetZfsVolumeUsageErrorExec",
			args:        args{volumeName: "someVolumeName"},
			want:        0,
			wantErr:     true,
		},
		{
			name:        "errorParse",
			mockCmdFunc: "TestGetZfsVolumeUsageErrorParse",
			args:        args{volumeName: "someVolumeName"},
			want:        0,
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

			got, err := GetZfsVolumeUsage(testCase.args.volumeName)

			if (err != nil) != testCase.wantErr {
				t.Errorf("GetZfsVolumeUsage() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("GetZfsVolumeUsage() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func TestGetZfsVolBlockSizeErrorUintParse(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Print("cirrinad0/disk/test2024021425_hd5       volblocksize    one   default\n") //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func TestGetZfsVolBlockSizeErrorNotFound(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Print("cannot open 'cirrinad0/disk/test2024021425_hd5': dataset does not exist\n") //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func TestGetZfsVolBlockSizeErrorFields(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Print("cirrinad0/disk/test2024021425_hd5       volblocksize    16384\n") //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func TestGetZfsVolBlockSizeErrorExit(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestGetZfsVolBlockSizeErrorDupe(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Print("cirrinad0/disk/test2024021425_hd5       volblocksize    16384   default\n") //nolint:forbidigo
	fmt.Print("cirrinad0/disk/test2024021425_hd5       volblocksize    16384   default\n") //nolint:forbidigo

	os.Exit(0)
}

//nolint:paralleltest
func TestGetZfsVolBlockSizeSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Print("cirrinad0/disk/test2024021425_hd5       volblocksize    16384   default\n") //nolint:forbidigo

	os.Exit(0)
}

//nolint:paralleltest
func TestGetZfsVolBlockSize(t *testing.T) {
	type args struct {
		volumeName string
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		want        uint64
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestGetZfsVolBlockSizeSuccess1",
			args:        args{volumeName: "cirrinad0/disk/test2024021425_hd5"},
			want:        16384,
			wantErr:     false,
		},
		{
			name:        "errorExit",
			mockCmdFunc: "TestGetZfsVolBlockSizeErrorExit",
			args:        args{volumeName: "cirrinad0/disk/test2024021425_hd5"},
			want:        0,
			wantErr:     true,
		},
		{
			name:        "errorDupe",
			mockCmdFunc: "TestGetZfsVolBlockSizeErrorDupe",
			args:        args{volumeName: "cirrinad0/disk/test2024021425_hd5"},
			want:        0,
			wantErr:     true,
		},
		{
			name:        "errorFields",
			mockCmdFunc: "TestGetZfsVolBlockSizeErrorFields",
			args:        args{volumeName: "cirrinad0/disk/test2024021425_hd5"},
			want:        0,
			wantErr:     true,
		},
		{
			name:        "errorNotFound",
			mockCmdFunc: "TestGetZfsVolBlockSizeErrorNotFound",
			args:        args{volumeName: "cirrinad0/disk/test2024021425_hd5"},
			want:        0,
			wantErr:     true,
		},
		{
			name:        "errorUintParse",
			mockCmdFunc: "TestGetZfsVolBlockSizeErrorUintParse",
			args:        args{volumeName: "cirrinad0/disk/test2024021425_hd5"},
			want:        0,
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

			got, err := GetZfsVolBlockSize(testCase.args.volumeName)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetZfsVolBlockSize() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("GetZfsVolBlockSize() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func TestSetZfsVolumeSizeSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestSetZfsVolumeSizeExitError(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestSetZfsVolumeSize(t *testing.T) {
	type args struct {
		volumeName string
		volSize    uint64
	}

	tests := []struct {
		name                       string
		mockCmdFunc                string
		mockGetZfsVolumeSizeFunc   func(string) (uint64, error)
		mockGetZfsVolBlockSizeFunc func(string) (uint64, error)
		args                       args
		wantErr                    bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestSetZfsVolumeSizeSuccess1",
			mockGetZfsVolumeSizeFunc: func(string) (uint64, error) {
				return 1073741824, nil
			},
			mockGetZfsVolBlockSizeFunc: func(string) (uint64, error) {
				return 16384, nil
			},
			args: args{
				volumeName: "someVolume",
				volSize:    2147483648,
			},
			wantErr: false,
		},
		{
			name:        "errorGetVolSize1",
			mockCmdFunc: "TestSetZfsVolumeSizeSuccess1",
			mockGetZfsVolumeSizeFunc: func(string) (uint64, error) {
				return 0, errDiskNotFound
			},
			mockGetZfsVolBlockSizeFunc: func(string) (uint64, error) {
				return 16384, nil
			},
			args: args{
				volumeName: "someVolume",
				volSize:    2147483648,
			},
			wantErr: true,
		},
		{
			name:        "errorGetVolSize2",
			mockCmdFunc: "TestSetZfsVolumeSizeSuccess1",
			mockGetZfsVolumeSizeFunc: func(string) (uint64, error) {
				return 2147483648, nil
			},
			mockGetZfsVolBlockSizeFunc: func(string) (uint64, error) {
				return 16384, nil
			},
			args: args{
				volumeName: "someVolume",
				volSize:    2147483648,
			},
			wantErr: false,
		},
		{
			name:        "errorGetVolBlockSize1",
			mockCmdFunc: "TestSetZfsVolumeSizeSuccess1",
			mockGetZfsVolumeSizeFunc: func(string) (uint64, error) {
				return 1073741824, nil
			},
			mockGetZfsVolBlockSizeFunc: func(string) (uint64, error) {
				return 0, errDiskNotFound
			},
			args: args{
				volumeName: "someVolume",
				volSize:    2147483648,
			},
			wantErr: true,
		},
		{
			name:        "success2",
			mockCmdFunc: "TestSetZfsVolumeSizeSuccess1",
			mockGetZfsVolumeSizeFunc: func(string) (uint64, error) {
				return 1073741824, nil
			},
			mockGetZfsVolBlockSizeFunc: func(string) (uint64, error) {
				return 16384, nil
			},
			args: args{
				volumeName: "someVolume",
				volSize:    2147483647,
			},
			wantErr: false,
		},
		{
			name:        "errorShrinkage",
			mockCmdFunc: "TestSetZfsVolumeSizeSuccess1",
			mockGetZfsVolumeSizeFunc: func(string) (uint64, error) {
				return 2147483648, nil
			},
			mockGetZfsVolBlockSizeFunc: func(string) (uint64, error) {
				return 16384, nil
			},
			args: args{
				volumeName: "someVolume",
				volSize:    1073741824,
			},
			wantErr: true,
		},
		{
			name:        "errorExit",
			mockCmdFunc: "TestSetZfsVolumeSizeExitError",
			mockGetZfsVolumeSizeFunc: func(string) (uint64, error) {
				return 1073741824, nil
			},
			mockGetZfsVolBlockSizeFunc: func(string) (uint64, error) {
				return 16384, nil
			},
			args: args{
				volumeName: "someVolume",
				volSize:    2147483648,
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

			GetZfsVolumeSizeFunc = testCase.mockGetZfsVolumeSizeFunc

			t.Cleanup(func() { GetZfsVolumeSizeFunc = GetZfsVolumeSize })

			GetZfsVolBlockSizeFunc = testCase.mockGetZfsVolBlockSizeFunc

			t.Cleanup(func() { GetZfsVolBlockSizeFunc = GetZfsVolBlockSize })

			err := SetZfsVolumeSize(testCase.args.volumeName, testCase.args.volSize)
			if (err != nil) != testCase.wantErr {
				t.Errorf("SetZfsVolumeSize() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}
