package disk

import (
	"errors"
	"os"
	"os/user"
	"syscall"
	"testing"
	"testing/fstest"

	"github.com/go-test/deep"
	"go.uber.org/mock/gomock"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/util"
)

func TestNewFileInfoService(t *testing.T) {
	type args struct {
		impl FileInfoFetcher
	}

	tests := []struct {
		name string
		args args
		want FileInfoService
	}{
		{
			name: "success1",
			args: args{impl: nil},
			want: FileInfoService{
				FileInfoImpl: &FileInfoCmds{},
			},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := NewFileInfoService(testCase.args.impl)

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestFileInfoService_GetSize(t *testing.T) {
	type fields struct {
		FileInfoImpl FileInfoFetcher
	}

	type args struct {
		volumeName string
	}

	tests := []struct {
		name              string
		mockSizeReturnVal uint64
		mockSizeReturnErr error
		fields            fields
		args              args
		want              uint64
		wantErr           bool
	}{
		{
			name:              "success1",
			mockSizeReturnVal: 1073741824,
			mockSizeReturnErr: nil,
			args: args{
				volumeName: "someVolumeName",
			},
			want:    1073741824,
			wantErr: false,
		},
		{
			name:              "error1",
			mockSizeReturnVal: 0,
			mockSizeReturnErr: errDiskNotFound,
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
			mock := NewMockFileInfoFetcher(ctrl)

			diskService := NewFileInfoService(mock)

			mock.EXPECT().FetchFileSize(testCase.args.volumeName).
				Return(testCase.mockSizeReturnVal, testCase.mockSizeReturnErr).
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

func TestFileInfoService_GetUsage(t *testing.T) {
	type fields struct {
		FileInfoImpl FileInfoFetcher
	}

	type args struct {
		volumeName string
	}

	tests := []struct {
		name               string
		mockUsageReturnVal uint64
		mockUsageReturnErr error
		fields             fields
		args               args
		want               uint64
		wantErr            bool
	}{
		{
			name:               "success1",
			mockUsageReturnVal: 662609920,
			mockUsageReturnErr: nil,
			args: args{
				volumeName: "someVolumeName",
			},
			want:    662609920,
			wantErr: false,
		},
		{
			name:               "error1",
			mockUsageReturnVal: 0,
			mockUsageReturnErr: errDiskNotFound,
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
			mock := NewMockFileInfoFetcher(ctrl)

			fileInfoService := NewFileInfoService(mock)

			mock.EXPECT().FetchFileUsage(testCase.args.volumeName).
				Return(testCase.mockUsageReturnVal, testCase.mockUsageReturnErr).
				MaxTimes(1)

			got, err := fileInfoService.GetUsage(testCase.args.volumeName)
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

func TestFileInfoService_SetSize(t *testing.T) {
	type args struct {
		name    string
		newSize uint64
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "success1",
			args:    args{name: "aDisk", newSize: 1024},
			wantErr: false,
		},
		{
			name:    "fail1",
			args:    args{name: "aDisk", newSize: 1024},
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			mock := NewMockFileInfoFetcher(ctrl)

			fileInfoService := NewFileInfoService(mock)

			var mockError error

			if testCase.wantErr {
				mockError = errors.New("another error") //nolint:goerr113
			}

			mock.EXPECT().ApplyFileSize(testCase.args.name, testCase.args.newSize).
				Return(mockError)

			err := fileInfoService.SetSize(testCase.args.name, testCase.args.newSize)
			if (err != nil) != testCase.wantErr {
				t.Errorf("SetSize() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestFileInfoService_Exists(t *testing.T) {
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
			args:    args{name: "aDisk"},
			want:    true,
			wantErr: false,
		},
		{
			name:    "success2",
			args:    args{name: "aDisk"},
			want:    false,
			wantErr: false,
		},
		{
			name:    "fail1",
			args:    args{name: "aDisk"},
			want:    true,
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mock := NewMockFileInfoFetcher(ctrl)

			fileInfoService := NewFileInfoService(mock)

			var mockError error

			if testCase.wantErr {
				mockError = errors.New("another error") //nolint:goerr113
			}

			mock.EXPECT().CheckExists(testCase.args.name).
				Return(testCase.want, mockError)

			got, err := fileInfoService.Exists(testCase.args.name)
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

func TestFileInfoService_Create(t *testing.T) {
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
			args:    args{name: "anotherDisk", size: 1024 * 1024},
			wantErr: false,
		},
		{
			name:    "fail1",
			args:    args{name: "anotherDisk", size: 1024 * 1024},
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mock := NewMockFileInfoFetcher(ctrl)

			fileInfoService := NewFileInfoService(mock)

			var mockError error

			if testCase.wantErr {
				mockError = errors.New("another error") //nolint:goerr113
			}

			mock.EXPECT().Add(testCase.args.name, testCase.args.size).
				Return(mockError)

			err := fileInfoService.Create(testCase.args.name, testCase.args.size)
			if (err != nil) != testCase.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestFileInfoService_GetAll(t *testing.T) {
	tests := []struct {
		name    string
		want    []string
		wantErr bool
	}{
		{
			name:    "success1",
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "success2",
			want:    []string{"diskA", "diskB"},
			wantErr: false,
		},
		{
			name:    "fail1",
			want:    nil,
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mock := NewMockFileInfoFetcher(ctrl)

			fileInfoService := NewFileInfoService(mock)

			var mockError error

			if testCase.wantErr {
				mockError = errors.New("another error") //nolint:goerr113
			}

			mock.EXPECT().FetchAll().
				Return(testCase.want, mockError)

			got, err := fileInfoService.GetAll()
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
func TestFileInfoCmds_FetchFileSize(t *testing.T) {
	type args struct {
		name string
	}

	tests := []struct {
		name    string
		testFS  fstest.MapFS
		args    args
		want    uint64
		wantErr bool
	}{
		{
			name: "success1",
			testFS: fstest.MapFS{
				"someDisk": {
					Data: []byte("some data"),
				},
			},
			args: args{
				name: "someDisk",
			},
			want:    9,
			wantErr: false,
		},
		{
			name: "fail1",
			testFS: fstest.MapFS{
				"someDisk": {
					Data: []byte("some data"),
				},
			},
			args: args{
				name: "someDisk",
			},
			want:    0,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockMyFS := NewMockLocalFileSystem(ctrl)

			testFileInfo, err := testCase.testFS.Stat("someDisk")
			if err != nil {
				t.Fatalf("failed setting up test: %s", err.Error())
			}

			var statErr error

			if testCase.wantErr {
				statErr = errors.New("something went wrong") //nolint:goerr113
			}

			mockMyFS.EXPECT().
				Stat(testCase.args.name).
				Return(testFileInfo, statErr).
				Times(1)

			f := FileInfoCmds{}
			myFS = mockMyFS

			got, err := f.FetchFileSize(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("FetchFileSize() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("FetchFileSize() got = %v, wantFetch %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func TestFileInfoCmds_FetchFileUsage(t *testing.T) {
	type args struct {
		name string
	}

	tests := []struct {
		name     string
		statFunc func(path string, st *syscall.Stat_t) (err error)
		args     args
		want     uint64
		wantErr  bool
	}{
		{
			name: "success1",
			statFunc: func(_ string, st *syscall.Stat_t) error {
				st.Blocks = 10

				return nil
			},
			args: args{
				name: "someDisk",
			},
			want:    5120,
			wantErr: false,
		},
		{
			name: "fail1",
			statFunc: func(_ string, _ *syscall.Stat_t) error {
				return errors.New("something went wrong") //nolint:goerr113
			},
			args: args{
				name: "someDisk",
			},
			want:    0,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			myStat = testCase.statFunc
			f := FileInfoCmds{}

			got, err := f.FetchFileUsage(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("FetchFileUsage() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("FetchFileUsage() got = %v, wantFetch %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func TestFileInfoCmds_ApplyFileSize(t *testing.T) {
	type args struct {
		name    string
		newSize uint64
	}

	tests := []struct {
		name                 string
		mockCmdFunc          string
		checkExistWant       bool
		checkExistWantErr    bool
		fetchFileSizeWant    uint64
		fetchFileSizeWantErr bool
		args                 args
		wantErr              bool
	}{
		{
			name:                 "success1",
			mockCmdFunc:          "TestFileInfoCmds_ApplyFileSizeSuccess1",
			checkExistWant:       true,
			checkExistWantErr:    false,
			fetchFileSizeWant:    1048576,
			fetchFileSizeWantErr: false,
			args:                 args{name: "someDisk", newSize: 2097152},
			wantErr:              false,
		},
		{
			name:                 "success2",
			mockCmdFunc:          "TestFileInfoCmds_ApplyFileSizeSuccess1",
			checkExistWant:       true,
			checkExistWantErr:    false,
			fetchFileSizeWant:    2097152,
			fetchFileSizeWantErr: false,
			args:                 args{name: "someDisk", newSize: 2097152},
			wantErr:              false,
		},
		{
			name:                 "fail1",
			mockCmdFunc:          "TestFileInfoCmds_ApplyFileSizeSuccess1",
			checkExistWant:       true,
			checkExistWantErr:    true,
			fetchFileSizeWant:    1048576,
			fetchFileSizeWantErr: false,
			args:                 args{name: "someDisk", newSize: 2097152},
			wantErr:              true,
		},
		{
			name:                 "fail2",
			mockCmdFunc:          "TestFileInfoCmds_ApplyFileSizeSuccess1",
			checkExistWant:       false,
			checkExistWantErr:    false,
			fetchFileSizeWant:    1048576,
			fetchFileSizeWantErr: false,
			args:                 args{name: "someDisk", newSize: 2097152},
			wantErr:              true,
		},
		{
			name:                 "fail3",
			mockCmdFunc:          "TestFileInfoCmds_ApplyFileSizeSuccess1",
			checkExistWant:       true,
			checkExistWantErr:    false,
			fetchFileSizeWant:    0,
			fetchFileSizeWantErr: true,
			args:                 args{name: "someDisk", newSize: 2097152},
			wantErr:              true,
		},
		{
			name:                 "fail4",
			mockCmdFunc:          "TestFileInfoCmds_ApplyFileSizeSuccess1",
			checkExistWant:       true,
			checkExistWantErr:    false,
			fetchFileSizeWant:    2097152,
			fetchFileSizeWantErr: false,
			args:                 args{name: "someDisk", newSize: 1048576},
			wantErr:              true,
		},
		{
			name:                 "fail5",
			mockCmdFunc:          "TestFileInfoCmds_ApplyFileSizeFail1",
			checkExistWant:       true,
			checkExistWantErr:    false,
			fetchFileSizeWant:    1048576,
			fetchFileSizeWantErr: false,
			args:                 args{name: "someDisk", newSize: 2097152},
			wantErr:              true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)

			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			fileInfoCmds := FileInfoCmds{}

			checkExistsFunc = func(_ FileInfoCmds, _ string) (bool, error) {
				var ceErr error
				if testCase.checkExistWantErr {
					ceErr = errors.New("checkExists returned error") //nolint:goerr113
				}

				return testCase.checkExistWant, ceErr
			}

			t.Cleanup(func() { checkExistsFunc = FileInfoCmds.CheckExists })

			fetchFileSizeFunc = func(_ FileInfoCmds, _ string) (uint64, error) {
				var ffsErr error

				if testCase.fetchFileSizeWantErr {
					ffsErr = errors.New("FetchFileSize returned and error") //nolint:goerr113
				}

				return testCase.fetchFileSizeWant, ffsErr
			}

			t.Cleanup(func() { fetchFileSizeFunc = FileInfoCmds.FetchFileSize })

			err := fileInfoCmds.ApplyFileSize(testCase.args.name, testCase.args.newSize)

			if (err != nil) != testCase.wantErr {
				t.Errorf("ApplyFileSize() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func TestFileInfoCmds_CheckExists(t *testing.T) {
	type args struct {
		name string
	}

	tests := []struct {
		name        string
		args        args
		want        bool
		wantErr     bool
		wantPath    bool
		wantPathErr bool
	}{
		{
			name:        "success1",
			args:        args{name: "someVolumeName"},
			want:        true,
			wantErr:     false,
			wantPath:    true,
			wantPathErr: false,
		},
		{
			name:        "success2",
			args:        args{name: "someVolumeName"},
			want:        false,
			wantErr:     false,
			wantPath:    false,
			wantPathErr: false,
		},
		{
			name:        "fail1",
			args:        args{name: "someVolumeName"},
			want:        true,
			wantErr:     true,
			wantPath:    false,
			wantPathErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			fileInfoCmds := FileInfoCmds{}

			PathExistsFunc = func(_ string) (bool, error) {
				if testCase.wantPathErr {
					return true, errors.New("another error") //nolint:goerr113
				}

				if testCase.wantPath {
					return true, nil
				}

				return false, nil
			}

			got, err := fileInfoCmds.CheckExists(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("CheckExists() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("CheckExists() got = %v, wantFetch %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func TestFileInfoCmds_Add(t *testing.T) {
	type args struct {
		name string
		size uint64
	}

	tests := []struct {
		name                string
		mockCmdFunc         string
		mockCurrentUserFunc func() (*user.User, error)
		args                args
		wantErr             bool
	}{
		{
			name:        "success1",
			mockCmdFunc: "TestFileInfoCmds_AddSuccess1",
			mockCurrentUserFunc: func() (*user.User, error) {
				mockUser := user.User{
					Uid:      "1001",
					Gid:      "1001",
					Username: "username",
					Name:     "username",
					HomeDir:  "/home/username",
				}

				return &mockUser, nil
			},
			args:    args{name: "someDisk", size: 2097152},
			wantErr: false,
		},
		{
			name:        "fail1",
			mockCmdFunc: "TestFileInfoCmds_AddFail1",
			mockCurrentUserFunc: func() (*user.User, error) {
				mockUser := user.User{
					Uid:      "1001",
					Gid:      "1001",
					Username: "username",
					Name:     "username",
					HomeDir:  "/home/username",
				}

				return &mockUser, nil
			},
			args:    args{name: "someDisk", size: 2097152},
			wantErr: true,
		},
		{
			name:        "fail2",
			mockCmdFunc: "TestFileInfoCmds_AddSuccess1",
			mockCurrentUserFunc: func() (*user.User, error) {
				mockUser := user.User{
					Uid:      "1001",
					Gid:      "1001",
					Username: "username",
					Name:     "username",
					HomeDir:  "/home/username",
				}

				return &mockUser, errors.New("some user error") //nolint:goerr113
			},
			args:    args{name: "someDisk", size: 2097152},
			wantErr: true,
		},
		{
			name:        "fail2",
			mockCmdFunc: "TestFileInfoCmds_AddFail2",
			mockCurrentUserFunc: func() (*user.User, error) {
				mockUser := user.User{
					Uid:      "1001",
					Gid:      "1001",
					Username: "username",
					Name:     "username",
					HomeDir:  "/home/username",
				}

				return &mockUser, nil
			},
			args:    args{name: "someDisk", size: 2097152},
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

			currentUserFunc = testCase.mockCurrentUserFunc

			t.Cleanup(func() { currentUserFunc = user.Current })

			f := FileInfoCmds{}
			err := f.Add(testCase.args.name, testCase.args.size)

			if (err != nil) != testCase.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func TestFileInfoCmds_FetchAll(t *testing.T) {
	tests := []struct {
		name                  string
		mockUtilOSReadDirFunc func(string) ([]string, error)
		want                  []string
		wantErr               bool
	}{
		{
			name: "success1",
			mockUtilOSReadDirFunc: func(_ string) ([]string, error) {
				return []string{"someDisk"}, nil
			},
			want:    []string{"someDisk"},
			wantErr: false,
		},
		{
			name: "fail1",
			mockUtilOSReadDirFunc: func(_ string) ([]string, error) {
				return nil, errors.New("some os readDir error") //nolint:goerr113
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			utilOSReadDirFunc = testCase.mockUtilOSReadDirFunc

			t.Cleanup(func() { utilOSReadDirFunc = util.OSReadDir })

			f := FileInfoCmds{}

			got, err := f.FetchAll()
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

// test helpers from here down

//nolint:paralleltest
func TestFileInfoCmds_ApplyFileSizeSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]
	cmd := cmdWithArgs[0]

	if cmd != "/usr/bin/truncate" {
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestFileInfoCmds_ApplyFileSizeFail1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]
	cmd := cmdWithArgs[0]

	if cmd != "/usr/bin/truncate" {
		os.Exit(0)
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestFileInfoCmds_AddSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]
	cmd := cmdWithArgs[0]

	if cmd == "/usr/bin/truncate" {
		if cmdWithArgs[len(cmdWithArgs)-1] == "someDisk" && cmdWithArgs[len(cmdWithArgs)-2] == "2097152" {
			os.Exit(0)
		}

		os.Exit(1)
	}

	if cmd == "/usr/sbin/chown" {
		if cmdWithArgs[len(cmdWithArgs)-1] == "someDisk" {
			os.Exit(0)
		}
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestFileInfoCmds_AddFail1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]
	cmd := cmdWithArgs[0]

	if cmd == "/usr/bin/truncate" {
		if cmdWithArgs[len(cmdWithArgs)-1] == "someDisk" && cmdWithArgs[len(cmdWithArgs)-2] == "2097152" {
			os.Exit(1)
		}

		os.Exit(0)
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestFileInfoCmds_AddFail2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]
	cmd := cmdWithArgs[0]

	if cmd == "/usr/bin/truncate" {
		os.Exit(0)
	}

	if cmd == "/usr/sbin/chown" {
		if cmdWithArgs[len(cmdWithArgs)-1] == "someDisk" {
			os.Exit(1)
		}

		os.Exit(0)
	}

	os.Exit(0)
}
