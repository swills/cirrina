package vm

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/kontera-technologies/go-supervisor/v2"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/util"
)

//nolint:paralleltest
func TestVM_applyResourceLimits(t *testing.T) {
	type fields struct {
		BhyvePid uint32
		proc     *supervisor.Process
		Config   Config
	}

	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "nilProc",
			fields: fields{
				proc: nil,
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(_ *testing.T) {
			vm := &VM{
				BhyvePid: testCase.fields.BhyvePid,
				proc:     testCase.fields.proc,
				Config:   testCase.fields.Config,
			}
			vm.applyResourceLimits()
		})
	}
}

//nolint:paralleltest
func Test_applyResourceLimitWriteIOPS(t *testing.T) {
	type args struct {
		vmPid string
		vm    *VM
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
	}{
		{
			name:        "execOK",
			mockCmdFunc: "Test_applyResourceLimitWriteIOPSSuccess",
			args: args{
				vmPid: "123123",
				vm: &VM{
					log: func() slog.Logger {
						var buf bytes.Buffer

						f := slog.New(slog.NewTextHandler(&buf, nil))

						return *f
					}(),
					Config: Config{
						Pcpu:  10,
						Rbps:  120,
						Wbps:  1234,
						Riops: 123,
						Wiops: 9999,
					},
				},
			},
		},
		{
			name:        "execErr",
			mockCmdFunc: "Test_applyResourceLimitWriteIOPSFail",
			args: args{
				vmPid: "123123",
				vm: &VM{
					log: func() slog.Logger {
						var buf bytes.Buffer

						f := slog.New(slog.NewTextHandler(&buf, nil))

						return *f
					}(),
					Config: Config{
						Pcpu:  10,
						Rbps:  120,
						Wbps:  1234,
						Riops: 123,
						Wiops: 9999,
					},
				},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)

			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			testCase.args.vm.applyResourceLimitWriteIOPS()
		})
	}
}

//nolint:paralleltest
func Test_applyResourceLimitReadIOPS(t *testing.T) {
	type args struct {
		vmPid string
		vm    *VM
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
	}{
		{
			name:        "execOK",
			mockCmdFunc: "Test_applyResourceLimitReadIOPSSuccess",
			args: args{
				vmPid: "123123",
				vm: &VM{
					log: func() slog.Logger {
						var buf bytes.Buffer

						f := slog.New(slog.NewTextHandler(&buf, nil))

						return *f
					}(),
					Config: Config{
						Pcpu:  10,
						Rbps:  120,
						Wbps:  1234,
						Riops: 9999,
						Wiops: 123,
					},
				},
			},
		},
		{
			name:        "execErr",
			mockCmdFunc: "Test_applyResourceLimitReadIOPSFail",
			args: args{
				vmPid: "123123",
				vm: &VM{
					log: func() slog.Logger {
						var buf bytes.Buffer

						f := slog.New(slog.NewTextHandler(&buf, nil))

						return *f
					}(),
					Config: Config{
						Pcpu:  10,
						Rbps:  120,
						Wbps:  1234,
						Riops: 9999,
						Wiops: 123,
					},
				},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(_ *testing.T) {
			testCase.args.vm.applyResourceLimitReadIOPS()
		})
	}
}

//nolint:paralleltest
func Test_applyResourceLimitWriteBPS(t *testing.T) {
	type args struct {
		vmPid string
		vm    *VM
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
	}{
		{
			name:        "execOK",
			mockCmdFunc: "Test_applyResourceLimitWriteBPSSuccess",
			args: args{
				vmPid: "123123",
				vm: &VM{
					log: func() slog.Logger {
						var buf bytes.Buffer

						f := slog.New(slog.NewTextHandler(&buf, nil))

						return *f
					}(),
					Config: Config{
						Pcpu:  10,
						Rbps:  120,
						Wbps:  1234,
						Riops: 123,
						Wiops: 9999,
					},
				},
			},
		},
		{
			name:        "execErr",
			mockCmdFunc: "Test_applyResourceLimitWriteBPSFail",
			args: args{
				vmPid: "123123",
				vm: &VM{
					log: func() slog.Logger {
						var buf bytes.Buffer

						f := slog.New(slog.NewTextHandler(&buf, nil))

						return *f
					}(),
					Config: Config{
						Pcpu:  10,
						Rbps:  120,
						Wbps:  1234,
						Riops: 123,
						Wiops: 9999,
					},
				},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)

			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			testCase.args.vm.applyResourceLimitWriteBPS()
		})
	}
}

//nolint:paralleltest
func Test_applyResourceLimitReadBPS(t *testing.T) {
	type args struct {
		vmPid string
		vm    *VM
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
	}{
		{
			name:        "execOK",
			mockCmdFunc: "Test_applyResourceLimitReadBPSSuccess",
			args: args{
				vmPid: "123123",
				vm: &VM{
					log: func() slog.Logger {
						var buf bytes.Buffer

						f := slog.New(slog.NewTextHandler(&buf, nil))

						return *f
					}(),
					Config: Config{
						Pcpu:  10,
						Rbps:  120,
						Wbps:  1234,
						Riops: 9999,
						Wiops: 123,
					},
				},
			},
		},
		{
			name:        "execErr",
			mockCmdFunc: "Test_applyResourceLimitReadBPSFail",
			args: args{
				vmPid: "123123",
				vm: &VM{
					log: func() slog.Logger {
						var buf bytes.Buffer

						f := slog.New(slog.NewTextHandler(&buf, nil))

						return *f
					}(),
					Config: Config{
						Pcpu:  10,
						Rbps:  120,
						Wbps:  1234,
						Riops: 9999,
						Wiops: 123,
					},
				},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(_ *testing.T) {
			testCase.args.vm.applyResourceLimitReadBPS()
		})
	}
}

//nolint:paralleltest
func Test_applyResourceLimitCPU(t *testing.T) {
	type args struct {
		vmPid string
		vm    *VM
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
	}{
		{
			name:        "execOK",
			mockCmdFunc: "Test_applyResourceLimitCPUSuccess",
			args: args{
				vmPid: "123123",
				vm: &VM{
					log: func() slog.Logger {
						var buf bytes.Buffer

						f := slog.New(slog.NewTextHandler(&buf, nil))

						return *f
					}(),
					Config: Config{
						Pcpu: 9292,
					},
				},
			},
		},
		{
			name:        "execOK",
			mockCmdFunc: "Test_applyResourceLimitCPUFail",
			args: args{
				vmPid: "123123",
				vm: &VM{
					log: func() slog.Logger {
						var buf bytes.Buffer

						f := slog.New(slog.NewTextHandler(&buf, nil))

						return *f
					}(),
					Config: Config{
						Pcpu: 9292,
					},
				},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(_ *testing.T) {
			testCase.args.vm.applyResourceLimitCPU()
		})
	}
}

// test helpers from here down

//nolint:paralleltest
func Test_applyResourceLimitWriteIOPSSuccess(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	for _, v := range os.Args {
		if v == "process:123123:writeiops:throttle=9999" {
			os.Exit(0)
		}
	}

	for _, v := range os.Args {
		fmt.Printf("v: %+v\n", v) //nolint:forbidigo
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_applyResourceLimitWriteIOPSFail(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_applyResourceLimitReadIOPSSuccess(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	for _, v := range os.Args {
		if v == "process:123123:readiops:throttle=9999" {
			os.Exit(0)
		}
	}

	for _, v := range os.Args {
		fmt.Printf("v: %+v\n", v) //nolint:forbidigo
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_applyResourceLimitReadIOPSFail(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_applyResourceLimitWriteBPSSuccess(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	for _, v := range os.Args {
		if v == "process:123123:writebps:throttle=9999" {
			os.Exit(0)
		}
	}

	for _, v := range os.Args {
		fmt.Printf("v: %+v\n", v) //nolint:forbidigo
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_applyResourceLimitWriteBPSFail(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_applyResourceLimitReadBPSSuccess(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	for _, v := range os.Args {
		if v == "process:123123:readbps:throttle=9999" {
			os.Exit(0)
		}
	}

	for _, v := range os.Args {
		fmt.Printf("v: %+v\n", v) //nolint:forbidigo
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_applyResourceLimitReadBPSFail(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_applyResourceLimitCPUSuccess(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	for _, v := range os.Args {
		if v == "process:123123:readbps:throttle=9999" {
			os.Exit(0)
		}
	}

	for _, v := range os.Args {
		fmt.Printf("v: %+v\n", v) //nolint:forbidigo
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_applyResourceLimitCPUFail(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}
