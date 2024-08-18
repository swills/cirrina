package epair

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/go-test/deep"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/util"
)

//nolint:paralleltest
func TestCreateEpair(t *testing.T) {
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
			name:        "fail1",
			mockCmdFunc: "TestCreateEpairFail1",
			args:        args{name: ""},
			wantErr:     true,
		},
		{
			name:        "fail2",
			mockCmdFunc: "TestCreateEpairFail2",
			args:        args{name: "epair32767"},
			wantErr:     true,
		},
		{
			name:        "fail3",
			mockCmdFunc: "TestCreateEpairFail3",
			args:        args{name: "epair32767"},
			wantErr:     true,
		},
		{
			name:        "fail4",
			mockCmdFunc: "TestCreateEpairFail4",
			args:        args{name: "epair32767"},
			wantErr:     true,
		},
		{
			name:        "success1",
			mockCmdFunc: "TestCreateEpairSuccess1",
			args:        args{name: "epair32767"},
			wantErr:     false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := CreateEpair(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("CreateEpair() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func TestDestroyEpair(t *testing.T) {
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
			name:        "fail1",
			mockCmdFunc: "TestDestroyEpairFail1",
			args:        args{name: "blah"},
			wantErr:     true,
		},
		{
			name:        "fail2",
			mockCmdFunc: "TestDestroyEpairFail1",
			args:        args{name: ""},
			wantErr:     true,
		},
		{
			name:        "success1",
			mockCmdFunc: "TestDestroyEpairSuccess1",
			args:        args{name: "epair32767"},
			wantErr:     false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := DestroyEpair(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("DestroyEpair() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func TestGetDummyEpairName(t *testing.T) {
	tests := []struct {
		name        string
		mockCmdFunc string
		want        string
	}{
		{
			name:        "success1",
			mockCmdFunc: "Test_getAllEpairSuccess1",
			want:        "epair32766",
		},
		{
			name:        "success2",
			mockCmdFunc: "TestGetDummyEpairNameSuccess2",
			want:        "epair32767",
		},
		{
			name:        "success3",
			mockCmdFunc: "TestGetDummyEpairNameSuccess3",
			want:        "epair16384",
		},
		{
			name:        "fail1",
			mockCmdFunc: "Test_getAllEpairFail1",
			want:        "",
		},
		{
			name:        "fail2",
			mockCmdFunc: "TestGetDummyEpairNameFail2",
			want:        "",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			got := GetDummyEpairName()

			if got != testCase.want {
				t.Errorf("GetDummyEpairName() = %v, want %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func TestNgCreatePipeWithRateLimit(t *testing.T) {
	type args struct {
		name string
		rate uint64
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		wantErr     bool
	}{
		{
			name:        "fail1",
			mockCmdFunc: "TestNgCreatePipeWithRateLimitFail1",
			args:        args{},
			wantErr:     true,
		},
		{
			name:        "fail2",
			mockCmdFunc: "TestNgCreatePipeWithRateLimitFail2",
			args:        args{name: "something", rate: 123456},
			wantErr:     true,
		},
		{
			name:        "fail3",
			mockCmdFunc: "TestNgCreatePipeWithRateLimitFail3",
			args:        args{name: "something", rate: 123456},
			wantErr:     true,
		},
		{
			name:        "fail4",
			mockCmdFunc: "TestNgCreatePipeWithRateLimitFail4",
			args:        args{name: "something", rate: 123456},
			wantErr:     true,
		},
		{
			name:        "success1",
			mockCmdFunc: "TestNgCreatePipeWithRateLimitSuccess1",
			args:        args{name: "something", rate: 123456},
			wantErr:     false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := NgCreatePipeWithRateLimit(testCase.args.name, testCase.args.rate)
			if (err != nil) != testCase.wantErr {
				t.Errorf("NgCreatePipeWithRateLimit() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func TestNgDestroyPipe(t *testing.T) {
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
			name:        "fail1",
			mockCmdFunc: "TestNgDestroyPipeFail1",
			args: args{
				name: "",
			},
			wantErr: true,
		},
		{
			name:        "success1",
			mockCmdFunc: "TestNgDestroyPipeSuccess1",
			args: args{
				name: "",
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := NgDestroyPipe(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("NgDestroyPipe() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func TestSetRateLimit(t *testing.T) {
	type args struct {
		name    string
		rateIn  uint64
		rateOut uint64
	}

	tests := []struct {
		name        string
		args        args
		mockCmdFunc string
		wantErr     bool
	}{
		{
			name:        "fail1",
			mockCmdFunc: "TestSetRateLimitFail1",
			args: args{
				name:    "",
				rateIn:  0,
				rateOut: 0,
			},
			wantErr: true,
		},
		{
			name:        "fail2",
			mockCmdFunc: "TestSetRateLimitFail2",
			args: args{
				name:    "something",
				rateIn:  123456,
				rateOut: 654321,
			},
			wantErr: true,
		},
		{
			name:        "success1",
			mockCmdFunc: "TestSetRateLimitSuccess1",
			args: args{
				name:    "something",
				rateIn:  123456,
				rateOut: 654321,
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			err := SetRateLimit(testCase.args.name, testCase.args.rateIn, testCase.args.rateOut)
			if (err != nil) != testCase.wantErr {
				t.Errorf("SetRateLimit() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

//nolint:paralleltest
func Test_getAllEpair(t *testing.T) {
	tests := []struct {
		name        string
		mockCmdFunc string
		want        []string
		wantErr     bool
	}{
		{
			name:        "fail1",
			mockCmdFunc: "Test_getAllEpairFail1",
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "empty1",
			mockCmdFunc: "Test_getAllEpairEmpty1",
			want:        nil,
			wantErr:     false,
		},
		{
			name:        "success1",
			mockCmdFunc: "Test_getAllEpairSuccess1",
			want:        []string{"epair32767"},
			wantErr:     false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			got, err := getAllEpair()
			if (err != nil) != testCase.wantErr {
				t.Errorf("getAllEpair() error = %v, wantErr %v", err, testCase.wantErr)

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
func Test_getAllEpairFail1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest,forbidigo
func Test_getAllEpairEmpty1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Printf("\n")
	fmt.Printf("asdf asdf\n")
	fmt.Printf("epair123b\n")
	fmt.Printf("epair123c\n")

	os.Exit(0)
}

//nolint:paralleltest,forbidigo
func Test_getAllEpairSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	fmt.Printf("epair32767b\nepair32767a\n")

	os.Exit(0)
}

//nolint:paralleltest
func TestGetDummyEpairNameSuccess2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest,forbidigo
func TestGetDummyEpairNameSuccess3(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	for i := 32767; i >= 16385; i-- {
		fmt.Printf("epair%db\nepair%da\n", i, i)
	}

	os.Exit(0)
}

//nolint:paralleltest,forbidigo
func TestGetDummyEpairNameFail2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	for i := 32767; i >= 0; i-- {
		fmt.Printf("epair%db\nepair%da\n", i, i)
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestDestroyEpairFail1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestDestroyEpairSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestCreateEpairFail1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestCreateEpairFail2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestCreateEpairFail3(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if cmdWithArgs[2] == "create" {
		os.Exit(0)
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestCreateEpairFail4(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if cmdWithArgs[2] == "create" {
		os.Exit(0)
	}

	if strings.HasSuffix(cmdWithArgs[1], "a") {
		os.Exit(0)
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestCreateEpairSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if cmdWithArgs[2] == "create" {
		os.Exit(0)
	}

	if strings.HasSuffix(cmdWithArgs[1], "a") {
		os.Exit(0)
	}

	if strings.HasSuffix(cmdWithArgs[1], "b") {
		os.Exit(0)
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestNgCreatePipeWithRateLimitFail1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestNgCreatePipeWithRateLimitFail2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if cmdWithArgs[1] == "mkpeer" {
		os.Exit(0)
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestNgCreatePipeWithRateLimitFail3(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if cmdWithArgs[1] == "mkpeer" || cmdWithArgs[1] == "name" {
		os.Exit(0)
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestNgCreatePipeWithRateLimitFail4(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if cmdWithArgs[1] == "mkpeer" || cmdWithArgs[1] == "name" || cmdWithArgs[1] == "connect" {
		os.Exit(0)
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestNgCreatePipeWithRateLimitSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	if cmdWithArgs[1] == "mkpeer" || cmdWithArgs[1] == "name" || cmdWithArgs[1] == "connect" || cmdWithArgs[1] == "msg" {
		os.Exit(0)
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestNgDestroyPipeFail1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestNgDestroyPipeSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func TestSetRateLimitFail1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestSetRateLimitFail2(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[4:]

	for index := range cmdWithArgs {
		nameParts := strings.Split(cmdWithArgs[index], ":")
		if len(nameParts) > 1 {
			if strings.HasSuffix(nameParts[0], "a") {
				os.Exit(0)
			}
		}

		nameNoPipe := strings.Split(cmdWithArgs[index], "_pipe")
		if len(nameNoPipe) > 1 {
			if strings.HasSuffix(nameNoPipe[0], "a") {
				os.Exit(0)
			}
		}
	}

	os.Exit(1)
}

//nolint:paralleltest
func TestSetRateLimitSuccess1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}
