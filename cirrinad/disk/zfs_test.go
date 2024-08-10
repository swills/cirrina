package disk

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/go-test/deep"
	"go.uber.org/mock/gomock"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/util"
)

func TestNewZfsVolService(t *testing.T) {
	type args struct {
		impl ZfsVolInfoFetcher
	}

	tests := []struct {
		name string
		args args
		want ZfsVolService
	}{
		{
			name: "success1",
			args: args{impl: nil},
			want: ZfsVolService{
				ZvolInfoImpl: &ZfsVolInfoCmds{},
			},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			diskService := NewZfsVolInfoService(testCase.args.impl)
			diff := deep.Equal(diskService, testCase.want)

			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestZfsVolService_GetSize(t *testing.T) {
	type args struct {
		volumeName string
	}

	tests := []struct {
		name                          string
		mockGetZfsVolumeSizeReturnVal uint64
		mockGetZfsVolumeSizeReturnErr error
		args                          args
		want                          uint64
		wantErr                       bool
	}{
		{
			name:                          "success1",
			mockGetZfsVolumeSizeReturnVal: 1073741824,
			mockGetZfsVolumeSizeReturnErr: nil,
			args: args{
				volumeName: "someVolumeName",
			},
			want:    1073741824,
			wantErr: false,
		},
		{
			name:                          "error1",
			mockGetZfsVolumeSizeReturnVal: 0,
			mockGetZfsVolumeSizeReturnErr: errDiskNotFound,
			args: args{
				volumeName: "someVolumeName",
			},
			want:    0,
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mock := NewMockZfsVolInfoFetcher(ctrl)

			diskService := NewZfsVolInfoService(mock)

			mock.EXPECT().FetchZfsVolumeSize(testCase.args.volumeName).
				Return(testCase.mockGetZfsVolumeSizeReturnVal, testCase.mockGetZfsVolumeSizeReturnErr).
				MaxTimes(1)

			got, err := diskService.GetSize(testCase.args.volumeName)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetSize() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("GetSize() got = %v, wantFetch %v", got, testCase.want)
			}
		})
	}
}

func TestZfsVolService_GetUsage(t *testing.T) {
	type args struct {
		volumeName string
	}

	tests := []struct {
		name                            string
		mockGetZfsVolUsageReturnVal     uint64
		mockGetZfsVolUsageSizeReturnErr error
		args                            args
		want                            uint64
		wantErr                         bool
	}{
		{
			name:                            "success1",
			mockGetZfsVolUsageReturnVal:     662609920,
			mockGetZfsVolUsageSizeReturnErr: nil,
			args: args{
				volumeName: "someVolumeName",
			},
			want:    662609920,
			wantErr: false,
		},
		{
			name:                            "error1",
			mockGetZfsVolUsageReturnVal:     0,
			mockGetZfsVolUsageSizeReturnErr: errDiskNotFound,
			args: args{
				volumeName: "someVolumeName",
			},
			want:    0,
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mock := NewMockZfsVolInfoFetcher(ctrl)

			diskService := NewZfsVolInfoService(mock)

			mock.EXPECT().FetchZfsVolumeUsage(testCase.args.volumeName).
				Return(testCase.mockGetZfsVolUsageReturnVal, testCase.mockGetZfsVolUsageSizeReturnErr).
				MaxTimes(1)

			got, err := diskService.GetUsage(testCase.args.volumeName)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetUsage() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("GetUsage() got = %v, wantFetch %v", got, testCase.want)
			}
		})
	}
}

func TestZfsVolService_SetSize(t *testing.T) {
	type args struct {
		volumeName string
		volSize    uint64
	}

	tests := []struct {
		name                            string
		mockGetZfsVolumeSizeReturnVal   uint64
		mockGetZfsVolumeSizeReturnErr   error
		mockGetZfsVolBlockSizeReturnVal uint64
		mockGetZfsVolBlockSizeReturnErr error
		args                            args
		wantErr                         bool
	}{
		{
			name:                            "success1",
			mockGetZfsVolumeSizeReturnVal:   1073741824,
			mockGetZfsVolumeSizeReturnErr:   nil,
			mockGetZfsVolBlockSizeReturnVal: 16384,
			mockGetZfsVolBlockSizeReturnErr: nil,
			args: args{
				volumeName: "someVolume",
				volSize:    2147483648,
			},
			wantErr: false,
		},
		{
			name:                            "errorGetVolSize1",
			mockGetZfsVolumeSizeReturnVal:   0,
			mockGetZfsVolumeSizeReturnErr:   errDiskNotFound,
			mockGetZfsVolBlockSizeReturnVal: 16384,
			mockGetZfsVolBlockSizeReturnErr: nil,
			args: args{
				volumeName: "someVolume",
				volSize:    2147483648,
			},
			wantErr: true,
		},
		{
			name:                            "errorGetVolSize2",
			mockGetZfsVolumeSizeReturnVal:   2147483648,
			mockGetZfsVolumeSizeReturnErr:   nil,
			mockGetZfsVolBlockSizeReturnVal: 16384,
			mockGetZfsVolBlockSizeReturnErr: nil,
			args: args{
				volumeName: "someVolume",
				volSize:    2147483648,
			},
			wantErr: false,
		},
		{
			name:                            "errorGetVolBlockSize1",
			mockGetZfsVolumeSizeReturnVal:   1073741824,
			mockGetZfsVolumeSizeReturnErr:   nil,
			mockGetZfsVolBlockSizeReturnVal: 0,
			mockGetZfsVolBlockSizeReturnErr: errDiskNotFound,
			args: args{
				volumeName: "someVolume",
				volSize:    2147483648,
			},
			wantErr: true,
		},
		{
			name:                            "success2",
			mockGetZfsVolumeSizeReturnVal:   1073741824,
			mockGetZfsVolumeSizeReturnErr:   nil,
			mockGetZfsVolBlockSizeReturnVal: 16384,
			mockGetZfsVolBlockSizeReturnErr: nil,
			args: args{
				volumeName: "someVolume",
				volSize:    2147483647,
			},
			wantErr: false,
		},
		{
			name:                            "errorShrinkage",
			mockGetZfsVolumeSizeReturnVal:   2147483648,
			mockGetZfsVolumeSizeReturnErr:   nil,
			mockGetZfsVolBlockSizeReturnVal: 16384,
			mockGetZfsVolBlockSizeReturnErr: nil,
			args: args{
				volumeName: "someVolume",
				volSize:    1073741824,
			},
			wantErr: true,
		},
		{
			name:                            "errorExit",
			mockGetZfsVolumeSizeReturnVal:   1073741824,
			mockGetZfsVolumeSizeReturnErr:   nil,
			mockGetZfsVolBlockSizeReturnVal: 16384,
			mockGetZfsVolBlockSizeReturnErr: nil,
			args: args{
				volumeName: "someVolume",
				volSize:    2147483648,
			},
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			wantNewVolSize := func(startVal uint64, blockSize uint64) uint64 {
				newVal := startVal
				if blockSize == 0 {
					return newVal
				}

				mod := startVal % blockSize

				if mod != 0 {
					ads := blockSize - mod
					newVal += ads
				}

				return newVal
			}(testCase.args.volSize, testCase.mockGetZfsVolBlockSizeReturnVal)

			mock := NewMockZfsVolInfoFetcher(ctrl)

			mock.EXPECT().FetchZfsVolumeSize(testCase.args.volumeName).
				Return(testCase.mockGetZfsVolumeSizeReturnVal, testCase.mockGetZfsVolumeSizeReturnErr)
			mock.EXPECT().FetchZfsVolBlockSize(testCase.args.volumeName).
				Return(testCase.mockGetZfsVolBlockSizeReturnVal, testCase.mockGetZfsVolBlockSizeReturnErr).
				MaxTimes(1)
			mock.EXPECT().ApplyZfsVolumeSize(testCase.args.volumeName, wantNewVolSize).
				MaxTimes(1).
				DoAndReturn(func(_ string, _ uint64) error {
					if testCase.wantErr {
						return errDiskNotFound
					}

					return nil
				})

			diskService := NewZfsVolInfoService(mock)

			err := diskService.SetSize(testCase.args.volumeName, testCase.args.volSize)
			if (err != nil) != testCase.wantErr {
				t.Errorf("SetSize() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func TestZfsVolService_Exists(t *testing.T) {
	type args struct {
		name string
	}

	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "success1",
			args:    args{"someVol"},
			want:    true,
			wantErr: false,
		},
		{
			name:    "fail1",
			args:    args{"someVol"},
			want:    true,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := NewMockZfsVolInfoFetcher(ctrl)

			var mockError error

			if testCase.wantErr {
				mockError = errors.New("random error") //nolint:goerr113
			}

			mock.EXPECT().CheckExists(testCase.args.name).
				Return(testCase.want, mockError).
				Times(1)

			diskService := NewZfsVolInfoService(mock)

			got, err := diskService.Exists(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("Exists() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("Exists() got = %v, wantFetch %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func TestZfsVolService_Create(t *testing.T) {
	type args struct {
		name string
		size uint64
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "success1",
			args:    args{"aVol", 1073741824},
			wantErr: false,
		},
		{
			name:    "success1",
			args:    args{"aVol", 1073741824},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := NewMockZfsVolInfoFetcher(ctrl)

			var mockError error

			if testCase.wantErr {
				mockError = errors.New("random error") //nolint:goerr113
			}

			mock.EXPECT().Add(testCase.args.name, testCase.args.size).
				Return(mockError).
				Times(1)

			diskService := NewZfsVolInfoService(mock)

			err := diskService.Create(testCase.args.name, testCase.args.size)
			if (err != nil) != testCase.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestZfsVolService_GetAll(t *testing.T) {
	tests := []struct {
		name                  string
		mockFetchAllReturnVal []string
		mockGetAllReturnErr   error
		want                  []string
		wantErr               bool
	}{
		{
			name:                  "success1",
			mockFetchAllReturnVal: []string{"aDisk", "anotherDisk"},
			mockGetAllReturnErr:   nil,
			want:                  []string{"aDisk", "anotherDisk"},
			wantErr:               false,
		},
		{
			name:                  "fail1",
			mockFetchAllReturnVal: nil,
			mockGetAllReturnErr:   errors.New("some error"), //nolint:goerr113
			want:                  nil,
			wantErr:               true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mock := NewMockZfsVolInfoFetcher(ctrl)
			diskService := NewZfsVolInfoService(mock)

			mock.EXPECT().FetchAll().
				Return(testCase.mockFetchAllReturnVal, testCase.mockGetAllReturnErr).
				Times(1)

			got, err := diskService.GetAll()
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetAll() error = %v, wantErr %v", err, testCase.wantErr)

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
func TestZfsVolInfoCmds_FetchZfsVolumeSize(t *testing.T) {
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
			mockCmdFunc: "TestFetchZfsVolumeSizeSuccess1",
			args:        args{volumeName: "someVolumeName"},
			want:        2147483648,
			wantErr:     false,
		},
		{
			name:        "errorExit",
			mockCmdFunc: "TestFetchZfsVolumeSizeErrorExit",
			args:        args{volumeName: "someVolumeName"},
			want:        0,
			wantErr:     true,
		},
		{
			name:        "errorExit",
			mockCmdFunc: "TestFetchZfsVolumeSizeErrorFields",
			args:        args{volumeName: "someVolumeName"},
			want:        0,
			wantErr:     true,
		},
		{
			name:        "errorParse",
			mockCmdFunc: "TestFetchZfsVolumeSizeErrorParse",
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

			n := ZfsVolService{
				ZvolInfoImpl: &ZfsVolInfoCmds{},
			}

			got, err := n.ZvolInfoImpl.FetchZfsVolumeSize(testCase.args.volumeName)
			if (err != nil) != testCase.wantErr {
				t.Errorf("FetchZfsVolumeSize() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("FetchZfsVolumeSize() got = %v, wantFetch %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func TestZfsVolInfoCmds_FetchVolumeUsage(t *testing.T) {
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
			mockCmdFunc: "TestFetchZfsVolumeUsageSuccess1",
			args:        args{volumeName: "someVolumeName"},
			want:        662609920,
			wantErr:     false,
		},
		{
			name:        "errorExec",
			mockCmdFunc: "TestFetchZfsVolumeUsageErrorExec",
			args:        args{volumeName: "someVolumeName"},
			want:        0,
			wantErr:     true,
		},
		{
			name:        "errorParse",
			mockCmdFunc: "TestFetchZfsVolumeUsageErrorParse",
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

			n := ZfsVolService{
				ZvolInfoImpl: &ZfsVolInfoCmds{},
			}

			got, err := n.ZvolInfoImpl.FetchZfsVolumeUsage(testCase.args.volumeName)

			if (err != nil) != testCase.wantErr {
				t.Errorf("FetchZfsVolumeUsage() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("FetchZfsVolumeUsage() got = %v, wantFetch %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func TestZfsVolInfoCmds_FetchBlockSize(t *testing.T) {
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
			mockCmdFunc: "TestFetchZfsVolBlockSizeSuccess1",
			args:        args{volumeName: "cirrinad0/disk/test2024021425_hd5"},
			want:        16384,
			wantErr:     false,
		},
		{
			name:        "errorExit",
			mockCmdFunc: "TestFetchZfsVolBlockSizeErrorExit",
			args:        args{volumeName: "cirrinad0/disk/test2024021425_hd5"},
			want:        0,
			wantErr:     true,
		},
		{
			name:        "errorDupe",
			mockCmdFunc: "TestFetchZfsVolBlockSizeErrorDupe",
			args:        args{volumeName: "cirrinad0/disk/test2024021425_hd5"},
			want:        0,
			wantErr:     true,
		},
		{
			name:        "errorFields",
			mockCmdFunc: "TestFetchZfsVolBlockSizeErrorFields",
			args:        args{volumeName: "cirrinad0/disk/test2024021425_hd5"},
			want:        0,
			wantErr:     true,
		},
		{
			name:        "errorNotFound",
			mockCmdFunc: "TestFetchZfsVolBlockSizeErrorNotFound",
			args:        args{volumeName: "cirrinad0/disk/test2024021425_hd5"},
			want:        0,
			wantErr:     true,
		},
		{
			name:        "errorUintParse",
			mockCmdFunc: "TestFetchZfsVolBlockSizeErrorUintParse",
			args:        args{volumeName: "cirrinad0/disk/test2024021425_hd5"},
			want:        0,
			wantErr:     true,
		},
		{
			name:        "errorUintParse",
			mockCmdFunc: "TestFetchZfsVolBlockSizeErrorZero",
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

			n := ZfsVolService{
				ZvolInfoImpl: &ZfsVolInfoCmds{},
			}

			got, err := n.ZvolInfoImpl.FetchZfsVolBlockSize(testCase.args.volumeName)
			if (err != nil) != testCase.wantErr {
				t.Errorf("FetchZfsVolBlockSize() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("FetchZfsVolBlockSize() got = %v, wantFetch %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func TestZfsVolInfoCmds_ApplyZfsVolumeSize(t *testing.T) {
	type args struct {
		volumeName string
		volSize    uint64
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestApplyZfsVolumeSizeSuccess1",
			args: args{
				volumeName: "someVolume",
				volSize:    2147483648,
			},
			wantErr: false,
		},
		{
			name:        "errorExit",
			mockCmdFunc: "TestApplyZfsVolumeSizeExitError",
			args: args{
				volumeName: "someVolume",
				volSize:    2147483648,
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

			e := &ZfsVolInfoCmds{}

			err := e.ApplyZfsVolumeSize(testCase.args.volumeName, testCase.args.volSize)
			if (err != nil) != testCase.wantErr {
				t.Errorf("ApplyZfsVolumeSize() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func TestZfsVolInfoCmds_CheckExists(t *testing.T) {
	type args struct {
		name string
	}

	tests := []struct {
		name         string
		args         args
		want         bool
		wantFetch    bool
		wantFetchErr bool
		wantPath     bool
		wantPathErr  bool
		wantErr      bool
	}{
		{
			name:         "success1",
			args:         args{name: "someVolumeName"},
			want:         true,
			wantErr:      false,
			wantFetch:    true,
			wantFetchErr: false,
			wantPath:     false,
			wantPathErr:  false,
		},
		{
			name:         "success2",
			args:         args{name: "someVolumeName"},
			want:         true,
			wantErr:      false,
			wantFetch:    false,
			wantFetchErr: false,
			wantPath:     true,
			wantPathErr:  false,
		},
		{
			name:         "TestZfsVolInfoCmds_CheckExistsFail1",
			args:         args{name: "someVolumeName"},
			want:         false,
			wantErr:      false,
			wantFetch:    false,
			wantFetchErr: false,
			wantPath:     false,
			wantPathErr:  false,
		},
		{
			name:         "TestZfsVolInfoCmds_CheckExistsFail2",
			args:         args{name: "someVolumeName"},
			want:         true,
			wantErr:      true,
			wantFetch:    false,
			wantFetchErr: true,
			wantPath:     false,
			wantPathErr:  false,
		},
		{
			name:         "TestZfsVolInfoCmds_CheckExistsFail3",
			args:         args{name: "someVolumeName"},
			want:         true,
			wantErr:      true,
			wantFetch:    false,
			wantFetchErr: false,
			wantPath:     false,
			wantPathErr:  true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			zfsVolInfoCmds := ZfsVolInfoCmds{}

			fetchAllFunc = func(_ ZfsVolInfoCmds) ([]string, error) {
				if testCase.wantFetchErr {
					return nil, errors.New("some error") //nolint:goerr113
				}

				if testCase.wantFetch {
					return []string{testCase.args.name}, nil
				}

				return nil, nil
			}

			PathExistsFunc = func(_ string) (bool, error) {
				if testCase.wantPathErr {
					return true, errors.New("another error") //nolint:goerr113
				}

				if testCase.wantPath {
					return true, nil
				}

				return false, nil
			}

			got, err := zfsVolInfoCmds.CheckExists(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("CheckExists() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("CheckExists() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func TestZfsVolInfoCmds_Add(t *testing.T) {
	type args struct {
		volName string
		size    uint64
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestZfsVolInfoCmds_AddSuccess1",
			args: args{
				volName: "someVolume",
				size:    2147483648,
			},
			wantErr: false,
		},
		{
			name:        "fail1",
			mockCmdFunc: "TestZfsVolInfoCmds_AddFail1",
			args: args{
				volName: "someVolume",
				size:    2147483648,
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

			e := ZfsVolInfoCmds{}

			err := e.Add(testCase.args.volName, testCase.args.size)
			if (err != nil) != testCase.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func TestZfsVolInfoCmds_FetchAll(t *testing.T) {
	tests := []struct {
		name        string
		mockCmdFunc string
		want        []string
		wantErr     bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestFetchAllZfsVolumesSuccess1",
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

			e := ZfsVolInfoCmds{}

			got, err := e.FetchAll()
			if (err != nil) != testCase.wantErr {
				t.Errorf("FetchAll() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

// test helpers from here on down

//nolint:paralleltest
func TestFetchAllZfsVolumesSuccess1(_ *testing.T) {
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
func TestFetchZfsVolumeSizeErrorParse(_ *testing.T) {
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
func TestFetchZfsVolumeSizeErrorFields(_ *testing.T) {
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
func TestFetchZfsVolumeSizeErrorExit(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestFetchZfsVolumeSizeSuccess1(_ *testing.T) {
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
func TestFetchZfsVolumeUsageSuccess1(_ *testing.T) {
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
func TestFetchZfsVolumeUsageErrorExec(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestFetchZfsVolumeUsageErrorParse(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Printf("number") //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func TestFetchZfsVolBlockSizeErrorUintParse(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Print("cirrinad0/disk/test2024021425_hd5       volblocksize    one   default\n") //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func TestFetchZfsVolBlockSizeErrorNotFound(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Print("cannot open 'cirrinad0/disk/test2024021425_hd5': dataset does not exist\n") //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func TestFetchZfsVolBlockSizeErrorFields(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Print("cirrinad0/disk/test2024021425_hd5       volblocksize    16384\n") //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func TestFetchZfsVolBlockSizeErrorExit(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestFetchZfsVolBlockSizeErrorDupe(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Print("cirrinad0/disk/test2024021425_hd5       volblocksize    16384   default\n") //nolint:forbidigo
	fmt.Print("cirrinad0/disk/test2024021425_hd5       volblocksize    16384   default\n") //nolint:forbidigo

	os.Exit(0)
}

//nolint:paralleltest
func TestFetchZfsVolBlockSizeErrorZero(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Print("cirrinad0/disk/test2024021425_hd5       volblocksize    0   default\n") //nolint:forbidigo

	os.Exit(0)
}

//nolint:paralleltest
func TestFetchZfsVolBlockSizeSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Print("cirrinad0/disk/test2024021425_hd5       volblocksize    16384   default\n") //nolint:forbidigo

	os.Exit(0)
}

//nolint:paralleltest
func TestApplyZfsVolumeSizeSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestApplyZfsVolumeSizeExitError(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestZfsVolInfoCmds_AddSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]
	cmd := cmdWithArgs[0]

	if cmd != "/sbin/zfs" {
		os.Exit(1)
	}

	if cmdWithArgs[len(cmdWithArgs)-1] != "someVolume" {
		os.Exit(1)
	}

	if cmdWithArgs[len(cmdWithArgs)-3] != "2147483648" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestZfsVolInfoCmds_AddFail1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestZfsVolInfoCmds_CheckExistsSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]
	cmd := cmdWithArgs[0]

	if cmd != "/sbin/zfs" {
		os.Exit(1)
	}

	if cmdWithArgs[len(cmdWithArgs)-1] != "someVolume" {
		os.Exit(1)
	}

	if cmdWithArgs[len(cmdWithArgs)-3] != "2147483648" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestZfsVolInfoCmds_CheckExistsSuccess2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestZfsVolInfoCmds_CheckExistsFail1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}
