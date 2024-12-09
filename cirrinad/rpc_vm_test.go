package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-test/deep"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"gorm.io/gorm"

	"cirrina/cirrina"
	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/iso"
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
	"cirrina/cirrinad/vmnic"
)

//nolint:paralleltest
func Test_server_GetVMID(t *testing.T) {
	type args struct {
		vmNameReq *wrapperspb.StringValue
	}

	tests := []struct {
		name        string
		mockClosure func()
		args        args
		want        *cirrina.VMID
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "5f90bba5-e830-4be7-b714-2ff8250e2e50",
					Name: "test2024082302",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 339,
						},
						VMID: "5f90bba5-e830-4be7-b714-2ff8250e2e50",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmNameReq: &wrapperspb.StringValue{
					Value: "test2024082302",
				},
			},
			want: func() *cirrina.VMID {
				r := cirrina.VMID{Value: "5f90bba5-e830-4be7-b714-2ff8250e2e50"}

				return &r
			}(),
			wantErr: false,
		},
		{
			name: "NotFound",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "5f90bba5-e830-4be7-b714-2ff8250e2e50",
					Name: "test2024082302",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 339,
						},
						VMID: "5f90bba5-e830-4be7-b714-2ff8250e2e50",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmNameReq: &wrapperspb.StringValue{
					Value: "test2024082300",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "NilReq",
			mockClosure: func() {},
			args: args{
				vmNameReq: nil,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "EmptyStringReq",
			mockClosure: func() {},
			args: args{
				vmNameReq: &wrapperspb.StringValue{
					Value: "",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()

			lis := bufconn.Listen(1024 * 1024)
			s := grpc.NewServer()
			reflection.Register(s)
			cirrina.RegisterVMInfoServer(s, &server{})

			go func() {
				if err := s.Serve(lis); err != nil {
					log.Fatalf("Server exited with error: %v", err)
				}
			}()

			resolver.SetDefaultScheme("passthrough")

			conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}

			defer func(conn *grpc.ClientConn) {
				_ = conn.Close()
			}(conn)

			client := cirrina.NewVMInfoClient(conn)

			var got *cirrina.VMID

			got, err = client.GetVMID(context.Background(), testCase.args.vmNameReq)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetVMID() error = %v, wantErr %v", err, testCase.wantErr)

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
func Test_server_GetVMName(t *testing.T) {
	type args struct {
		vmID *cirrina.VMID
	}

	tests := []struct {
		name        string
		mockClosure func()
		args        args
		want        *wrapperspb.StringValue
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "8a5e7df9-8236-4072-abff-aa8d9765d58f",
					Name: "test2024082303",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 339,
						},
						VMID: "8a5e7df9-8236-4072-abff-aa8d9765d58f",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "8a5e7df9-8236-4072-abff-aa8d9765d58f",
				},
			},
			want: &wrapperspb.StringValue{
				Value: "test2024082303",
			},
			wantErr: false,
		},
		{
			name: "NotFound",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "8a5e7df9-8236-4072-abff-aa8d9765d58f",
					Name: "test2024082303",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 339,
						},
						VMID: "8a5e7df9-8236-4072-abff-aa8d9765d58f",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "d1373974-ca4b-4d2e-b0a1-0f1934361142",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "BadUuid",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "8a5e7df9-8236-4072-abff-aa8d9765d58f",
					Name: "test2024082303",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 339,
						},
						VMID: "8a5e7df9-8236-4072-abff-aa8d9765d58f",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "d1373974-ca4b-4d2e-b0a1-0f1934361",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()

			lis := bufconn.Listen(1024 * 1024)
			s := grpc.NewServer()
			reflection.Register(s)
			cirrina.RegisterVMInfoServer(s, &server{})

			go func() {
				if err := s.Serve(lis); err != nil {
					log.Fatalf("Server exited with error: %v", err)
				}
			}()

			resolver.SetDefaultScheme("passthrough")

			conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}

			defer func(conn *grpc.ClientConn) {
				_ = conn.Close()
			}(conn)

			client := cirrina.NewVMInfoClient(conn)

			var got *wrapperspb.StringValue

			got, err = client.GetVMName(context.Background(), testCase.args.vmID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetVMName() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

//nolint:paralleltest,maintidx
func Test_server_GetVMState(t *testing.T) {
	type args struct {
		vmID *cirrina.VMID
	}

	tests := []struct {
		name        string
		mockClosure func()
		args        args
		want        *cirrina.VMState
		wantErr     bool
	}{
		{
			name: "SuccessStopped",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "176af73d-e4ad-4b55-adf9-f21dc0d68d66",
					Name: "test2024082303",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 339,
						},
						VMID: "176af73d-e4ad-4b55-adf9-f21dc0d68d66",
						CPU:  2,
						Mem:  1024,
					},
					Status:    vm.STOPPED,
					VNCPort:   0,
					DebugPort: 0,
				}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "176af73d-e4ad-4b55-adf9-f21dc0d68d66",
				},
			},
			want: func() *cirrina.VMState {
				vmState := cirrina.VMState{
					Status:    cirrina.VmStatus_STATUS_STOPPED,
					VncPort:   0,
					DebugPort: 0,
				}

				return &vmState
			}(),
			wantErr: false,
		},
		{
			name: "SuccessStarting",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "f91e68e2-716d-4496-b55b-d6a2b6121388",
					Name: "test2024082303",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 339,
						},
						VMID: "f91e68e2-716d-4496-b55b-d6a2b6121388",
						CPU:  2,
						Mem:  1024,
					},
					Status:    vm.STARTING,
					VNCPort:   6900,
					DebugPort: 3434,
				}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "f91e68e2-716d-4496-b55b-d6a2b6121388",
				},
			},
			want: func() *cirrina.VMState {
				vmState := cirrina.VMState{
					Status:    cirrina.VmStatus_STATUS_STARTING,
					VncPort:   6900,
					DebugPort: 3434,
				}

				return &vmState
			}(),
			wantErr: false,
		},
		{
			name: "SuccessRunning",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "a5277e49-6cc0-49a5-a492-6447dd094e4f",
					Name: "test2024082303",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 339,
						},
						VMID: "a5277e49-6cc0-49a5-a492-6447dd094e4f",
						CPU:  2,
						Mem:  1024,
					},
					Status:    vm.RUNNING,
					VNCPort:   6901,
					DebugPort: 3435,
				}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "a5277e49-6cc0-49a5-a492-6447dd094e4f",
				},
			},
			want: func() *cirrina.VMState {
				vmState := cirrina.VMState{
					Status:    cirrina.VmStatus_STATUS_RUNNING,
					VncPort:   6901,
					DebugPort: 3435,
				}

				return &vmState
			}(),
			wantErr: false,
		},
		{
			name: "SuccessStopping",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "0160587e-62af-4166-8e60-47f32d6e481f",
					Name: "test2024082303",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 339,
						},
						VMID: "0160587e-62af-4166-8e60-47f32d6e481f",
						CPU:  2,
						Mem:  1024,
					},
					Status:    vm.STOPPING,
					VNCPort:   6902,
					DebugPort: 3436,
				}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "0160587e-62af-4166-8e60-47f32d6e481f",
				},
			},
			want: func() *cirrina.VMState {
				vmState := cirrina.VMState{
					Status:    cirrina.VmStatus_STATUS_STOPPING,
					VncPort:   6902,
					DebugPort: 3436,
				}

				return &vmState
			}(),
			wantErr: false,
		},
		{
			name: "BadStatus",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "176af73d-e4ad-4b55-adf9-f21dc0d68d66",
					Name: "test2024082303",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 339,
						},
						VMID: "176af73d-e4ad-4b55-adf9-f21dc0d68d66",
						CPU:  2,
						Mem:  1024,
					},
					Status:    vm.StatusType("junk"),
					VNCPort:   0,
					DebugPort: 0,
				}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "176af73d-e4ad-4b55-adf9-f21dc0d68d66",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "EmptyName",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "1f09210f-258f-4765-ab88-24f9dccb61e1",
					Name: "",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 339,
						},
						VMID: "1f09210f-258f-4765-ab88-24f9dccb61e1",
						CPU:  2,
						Mem:  1024,
					},
					Status:    vm.STOPPED,
					VNCPort:   0,
					DebugPort: 0,
				}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "1f09210f-258f-4765-ab88-24f9dccb61e1",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NotFound",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "176af73d-e4ad-4b55-adf9-f21dc0d68d66",
					Name: "test2024082303",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 339,
						},
						VMID: "176af73d-e4ad-4b55-adf9-f21dc0d68d66",
						CPU:  2,
						Mem:  1024,
					},
					Status:    vm.STOPPED,
					VNCPort:   0,
					DebugPort: 0,
				}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "401c0bca-274d-486f-a1f3-4e95ba8c268f",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "BadUuid",
			mockClosure: func() {
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "176af73d-e4ad-4b55-adf9-f21dc0",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()

			lis := bufconn.Listen(1024 * 1024)
			s := grpc.NewServer()
			reflection.Register(s)
			cirrina.RegisterVMInfoServer(s, &server{})

			go func() {
				if err := s.Serve(lis); err != nil {
					log.Fatalf("Server exited with error: %v", err)
				}
			}()

			resolver.SetDefaultScheme("passthrough")

			conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}

			defer func(conn *grpc.ClientConn) {
				_ = conn.Close()
			}(conn)

			client := cirrina.NewVMInfoClient(conn)

			var got *cirrina.VMState

			got, err = client.GetVMState(context.Background(), testCase.args.vmID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetVMState() error = %v, wantErr %v", err, testCase.wantErr)

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
func Test_server_GetVMs(t *testing.T) {
	tests := []struct {
		name        string
		mockClosure func()
		want        []string
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "151d2a7e-ef23-4a25-bf23-2d30e88cd63c",
					Name: "test2024082304",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 339,
						},
						VMID: "151d2a7e-ef23-4a25-bf23-2d30e88cd63c",
						CPU:  2,
						Mem:  1024,
					},
					Status:    vm.STOPPED,
					VNCPort:   0,
					DebugPort: 0,
				}
				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			want:    []string{"151d2a7e-ef23-4a25-bf23-2d30e88cd63c"},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()

			lis := bufconn.Listen(1024 * 1024)
			s := grpc.NewServer()
			reflection.Register(s)
			cirrina.RegisterVMInfoServer(s, &server{})

			go func() {
				if err := s.Serve(lis); err != nil {
					log.Fatalf("Server exited with error: %v", err)
				}
			}()

			resolver.SetDefaultScheme("passthrough")

			conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}

			defer func(conn *grpc.ClientConn) {
				_ = conn.Close()
			}(conn)

			client := cirrina.NewVMInfoClient(conn)

			var vmID *cirrina.VMID

			var got []string

			var res cirrina.VMInfo_GetVMsClient

			res, err = client.GetVMs(context.Background(), &cirrina.VMsQuery{})
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetVMs() error = %v, wantErr %v", err, testCase.wantErr)
			}

			for {
				vmID, err = res.Recv()
				if errors.Is(err, io.EOF) {
					break
				}

				got = append(got, vmID.GetValue())
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

//nolint:paralleltest,maintidx
func Test_server_GetVMConfig(t *testing.T) {
	type args struct {
		vmID *cirrina.VMID
	}

	tests := []struct {
		name        string
		mockClosure func()
		args        args
		want        *cirrina.VMConfig
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:          "168f10f2-5831-4421-ab9b-254be6478016",
					Description: "a test VM",
					Name:        "test2024082401",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 340,
						},
						VMID:             "168f10f2-5831-4421-ab9b-254be6478016",
						CPU:              2,
						Mem:              1024,
						MaxWait:          120,
						Restart:          true,
						RestartDelay:     0,
						Screen:           true,
						ScreenWidth:      1920,
						ScreenHeight:     1080,
						VNCWait:          false,
						VNCPort:          "AUTO",
						Tablet:           true,
						StoreUEFIVars:    true,
						UTCTime:          true,
						HostBridge:       true,
						ACPI:             true,
						UseHLT:           true,
						ExitOnPause:      true,
						WireGuestMem:     false,
						DestroyPowerOff:  true,
						IgnoreUnknownMSR: true,
						KbdLayout:        "default",
						AutoStart:        false,
						Sound:            false,
						SoundIn:          "/dev/dsp0",
						SoundOut:         "/dev/dsp0",
						Com1:             true,
						Com1Dev:          "AUTO",
						Com1Log:          true,
						Com2:             false,
						Com2Dev:          "AUTO",
						Com2Log:          false,
						Com3:             false,
						Com3Dev:          "AUTO",
						Com3Log:          false,
						Com4:             false,
						Com4Dev:          "AUTO",
						Com4Log:          false,
						ExtraArgs:        "",
						Com1Speed:        115200,
						Com2Speed:        0,
						Com3Speed:        0,
						Com4Speed:        0,
						AutoStartDelay:   0,
						Debug:            false,
						DebugWait:        false,
						DebugPort:        "AUTO",
						Priority:         10,
						Protect: sql.NullBool{
							Bool:  false,
							Valid: false,
						},
						Pcpu:  0,
						Rbps:  0,
						Wbps:  0,
						Riops: 0,
						Wiops: 0,
					},
					Status:    vm.STOPPED,
					VNCPort:   0,
					DebugPort: 0,
				}
				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "168f10f2-5831-4421-ab9b-254be6478016",
				},
			},
			wantErr: false,
			want: func() *cirrina.VMConfig {
				testConfig := cirrina.VMConfig{
					Id:             "168f10f2-5831-4421-ab9b-254be6478016",
					Name:           func() *string { r := "test2024082401"; return &r }(), //nolint:nlreturn
					Description:    func() *string { r := "a test VM"; return &r }(),      //nolint:nlreturn
					Cpu:            func() *uint32 { var r uint32 = 2; return &r }(),      //nolint:nlreturn
					Mem:            func() *uint32 { var r uint32 = 1024; return &r }(),   //nolint:nlreturn
					MaxWait:        func() *uint32 { var r uint32 = 120; return &r }(),    //nolint:nlreturn
					Restart:        func() *bool { r := true; return &r }(),               //nolint:nlreturn
					RestartDelay:   func() *uint32 { var r uint32; return &r }(),          //nolint:nlreturn
					Screen:         func() *bool { r := true; return &r }(),               //nolint:nlreturn
					ScreenWidth:    func() *uint32 { var r uint32 = 1920; return &r }(),   //nolint:nlreturn
					ScreenHeight:   func() *uint32 { var r uint32 = 1080; return &r }(),   //nolint:nlreturn
					Vncwait:        func() *bool { r := false; return &r }(),              //nolint:nlreturn
					Wireguestmem:   func() *bool { r := false; return &r }(),              //nolint:nlreturn
					Tablet:         func() *bool { r := true; return &r }(),               //nolint:nlreturn
					Storeuefi:      func() *bool { r := true; return &r }(),               //nolint:nlreturn
					Utc:            func() *bool { r := true; return &r }(),               //nolint:nlreturn
					Hostbridge:     func() *bool { r := true; return &r }(),               //nolint:nlreturn
					Acpi:           func() *bool { r := true; return &r }(),               //nolint:nlreturn
					Hlt:            func() *bool { r := true; return &r }(),               //nolint:nlreturn
					Eop:            func() *bool { r := true; return &r }(),               //nolint:nlreturn
					Dpo:            func() *bool { r := true; return &r }(),               //nolint:nlreturn
					Ium:            func() *bool { r := true; return &r }(),               //nolint:nlreturn
					Vncport:        func() *string { r := "AUTO"; return &r }(),           //nolint:nlreturn
					Keyboard:       func() *string { r := "default"; return &r }(),        //nolint:nlreturn
					Autostart:      func() *bool { r := false; return &r }(),              //nolint:nlreturn
					Sound:          func() *bool { r := false; return &r }(),              //nolint:nlreturn
					SoundIn:        func() *string { r := "/dev/dsp0"; return &r }(),      //nolint:nlreturn
					SoundOut:       func() *string { r := "/dev/dsp0"; return &r }(),      //nolint:nlreturn
					Com1:           func() *bool { r := true; return &r }(),               //nolint:nlreturn
					Com1Dev:        func() *string { r := "AUTO"; return &r }(),           //nolint:nlreturn
					Com2:           func() *bool { r := false; return &r }(),              //nolint:nlreturn
					Com2Dev:        func() *string { r := "AUTO"; return &r }(),           //nolint:nlreturn
					Com3:           func() *bool { r := false; return &r }(),              //nolint:nlreturn
					Com3Dev:        func() *string { r := "AUTO"; return &r }(),           //nolint:nlreturn
					Com4:           func() *bool { r := false; return &r }(),              //nolint:nlreturn
					Com4Dev:        func() *string { r := "AUTO"; return &r }(),           //nolint:nlreturn
					ExtraArgs:      func() *string { r := ""; return &r }(),               //nolint:nlreturn
					Com1Log:        func() *bool { r := true; return &r }(),               //nolint:nlreturn
					Com2Log:        func() *bool { r := false; return &r }(),              //nolint:nlreturn
					Com3Log:        func() *bool { r := false; return &r }(),              //nolint:nlreturn
					Com4Log:        func() *bool { r := false; return &r }(),              //nolint:nlreturn
					Com1Speed:      func() *uint32 { var r uint32 = 115200; return &r }(), //nolint:nlreturn
					Com2Speed:      func() *uint32 { var r uint32; return &r }(),          //nolint:nlreturn
					Com3Speed:      func() *uint32 { var r uint32; return &r }(),          //nolint:nlreturn
					Com4Speed:      func() *uint32 { var r uint32; return &r }(),          //nolint:nlreturn
					AutostartDelay: func() *uint32 { var r uint32; return &r }(),          //nolint:nlreturn
					Debug:          func() *bool { r := false; return &r }(),              //nolint:nlreturn
					DebugWait:      func() *bool { r := false; return &r }(),              //nolint:nlreturn
					DebugPort:      func() *string { r := "AUTO"; return &r }(),           //nolint:nlreturn
					Priority:       func() *int32 { var r int32 = 10; return &r }(),       //nolint:nlreturn
					Protect:        func() *bool { r := false; return &r }(),              //nolint:nlreturn
					Pcpu:           func() *uint32 { var r uint32; return &r }(),          //nolint:nlreturn
					Rbps:           func() *uint32 { var r uint32; return &r }(),          //nolint:nlreturn
					Wbps:           func() *uint32 { var r uint32; return &r }(),          //nolint:nlreturn
					Riops:          func() *uint32 { var r uint32; return &r }(),          //nolint:nlreturn
					Wiops:          func() *uint32 { var r uint32; return &r }(),          //nolint:nlreturn
				}

				return &testConfig
			}(),
		},
		{
			name: "ErrorEmptyName",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "3b02e8e4-1eb1-450b-bf22-589ea2a60edd",
					Name: "",
				}
				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "3b02e8e4-1eb1-450b-bf22-589ea2a60edd",
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "ErrorNotFound",
			mockClosure: func() {
				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "3b02e8e4-1eb1-450b-bf22-589ea2a60ede",
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "ErrorBadUuid",
			mockClosure: func() {
				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "3b02e8e4-1eb1-450b-bf22-589ea2a60",
				},
			},
			wantErr: true,
			want:    nil,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()

			lis := bufconn.Listen(1024 * 1024)
			s := grpc.NewServer()
			reflection.Register(s)
			cirrina.RegisterVMInfoServer(s, &server{})

			go func() {
				if err := s.Serve(lis); err != nil {
					log.Fatalf("Server exited with error: %v", err)
				}
			}()

			resolver.SetDefaultScheme("passthrough")

			conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}

			defer func(conn *grpc.ClientConn) {
				_ = conn.Close()
			}(conn)

			client := cirrina.NewVMInfoClient(conn)

			var got *cirrina.VMConfig

			got, err = client.GetVMConfig(context.Background(), testCase.args.vmID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetVMConfig() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

//nolint:paralleltest,maintidx,gocognit
func Test_server_GetVMNics(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		vmID *cirrina.VMID
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        []string
		wantErr     bool
	}{
		{
			name: "SuccessOneNic",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				testVM1 := vm.VM{
					ID:   "3df50790-cbf0-46aa-b00e-c2b68f1ea165",
					Name: "test2024082402",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 812,
						},
					},
				}
				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).
					WithArgs(812).
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
								"67523036-a5c8-4975-8279-db6640182ebf",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024082402_int0",
								"another daily test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"",
								"",
								false,
								0,
								0,
								"",
								"",
								0,
							),
					)
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "3df50790-cbf0-46aa-b00e-c2b68f1ea165",
				},
			},
			want: []string{"67523036-a5c8-4975-8279-db6640182ebf"},
		},
		{
			name: "ErrorGettingNic",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				testVM1 := vm.VM{
					ID:   "3df50790-cbf0-46aa-b00e-c2b68f1ea165",
					Name: "test2024082402",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 812,
						},
					},
				}
				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).
					WithArgs(812).
					WillReturnError(gorm.ErrInvalidData)
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "3df50790-cbf0-46aa-b00e-c2b68f1ea165",
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "ErrorEmptyName",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				testVM1 := vm.VM{
					ID:   "3df50790-cbf0-46aa-b00e-c2b68f1ea165",
					Name: "",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 812,
						},
					},
				}
				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "3df50790-cbf0-46aa-b00e-c2b68f1ea165",
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "ErrorVMNotFound",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "3df50790-cbf0-46aa-b00e-c2b68f1ea166",
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "ErrorBadID",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "3df50790-cbf0-46aa-b0",
				},
			},
			wantErr: true,
			want:    nil,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mock)

			lis := bufconn.Listen(1024 * 1024)
			s := grpc.NewServer()
			reflection.Register(s)
			cirrina.RegisterVMInfoServer(s, &server{})

			go func() {
				if err := s.Serve(lis); err != nil {
					log.Fatalf("Server exited with error: %v", err)
				}
			}()

			resolver.SetDefaultScheme("passthrough")

			conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}

			defer func(conn *grpc.ClientConn) {
				_ = conn.Close()
			}(conn)

			client := cirrina.NewVMInfoClient(conn)

			var res cirrina.VMInfo_GetVMNicsClient

			var vmNicID *cirrina.VmNicId

			var got []string

			res, err = client.GetVMNics(context.Background(), testCase.args.vmID)
			if err != nil {
				t.Fatalf("GetVMNics() error setting up stream err = %v", err)
			}

			for {
				vmNicID, err = res.Recv()
				if !errors.Is(err, io.EOF) && (err != nil) != testCase.wantErr {
					t.Errorf("GetVMNics() streamErr = %v, wantErr %v", err, testCase.wantErr)
				}

				if err != nil {
					break
				}

				got = append(got, vmNicID.GetValue())
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

//nolint:paralleltest
func Test_server_GetVMDisks(t *testing.T) {
	type args struct {
		vmID *cirrina.VMID
	}

	tests := []struct {
		name        string
		mockClosure func()
		want        []string
		args        args
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "c95064e9-c4c0-4308-85f3-508140490981",
					Name: "test2024082405",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 812,
						},
					},
					Disks: func() []*disk.Disk {
						testDisk := disk.Disk{
							ID: "389640e0-225d-4b86-9e4d-e09b415cf0d7",
						}

						testDiskList := []*disk.Disk{&testDisk}

						return testDiskList
					}(),
				}
				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: func() *cirrina.VMID {
					r := cirrina.VMID{Value: "c95064e9-c4c0-4308-85f3-508140490981"}

					return &r
				}(),
			},
			want:    []string{"389640e0-225d-4b86-9e4d-e09b415cf0d7"},
			wantErr: false,
		},
		{
			name: "DiskIsNil",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "c95064e9-c4c0-4308-85f3-508140490981",
					Name: "test2024082405",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 812,
						},
					},
					Disks: func() []*disk.Disk {
						testDiskList := []*disk.Disk{nil}

						return testDiskList
					}(),
				}
				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: func() *cirrina.VMID {
					r := cirrina.VMID{Value: "c95064e9-c4c0-4308-85f3-508140490981"}

					return &r
				}(),
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "VMNameEmpty",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "c95064e9-c4c0-4308-85f3-508140490981",
					Name: "",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 812,
						},
					},
				}
				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: func() *cirrina.VMID {
					r := cirrina.VMID{Value: "c95064e9-c4c0-4308-85f3-508140490981"}

					return &r
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "VMNotFound",
			mockClosure: func() {
				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				vmID: func() *cirrina.VMID {
					r := cirrina.VMID{Value: "c95064e9-c4c0-4308-85f3-508140490999"}

					return &r
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "BadUuid",
			mockClosure: func() {
				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				vmID: func() *cirrina.VMID {
					r := cirrina.VMID{Value: "c95064e9-c4c0-4308-85f"}

					return &r
				}(),
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()

			lis := bufconn.Listen(1024 * 1024)
			s := grpc.NewServer()
			reflection.Register(s)
			cirrina.RegisterVMInfoServer(s, &server{})

			go func() {
				if err := s.Serve(lis); err != nil {
					log.Fatalf("Server exited with error: %v", err)
				}
			}()

			resolver.SetDefaultScheme("passthrough")

			conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}

			defer func(conn *grpc.ClientConn) {
				_ = conn.Close()
			}(conn)

			client := cirrina.NewVMInfoClient(conn)

			var res cirrina.VMInfo_GetVMDisksClient

			var diskID *cirrina.DiskId

			var got []string

			res, err = client.GetVMDisks(context.Background(), testCase.args.vmID)
			if err != nil {
				t.Fatalf("GetVMDisks() error setting up client, error = %v", err)
			}

			for {
				diskID, err = res.Recv()
				if !errors.Is(err, io.EOF) && (err != nil) != testCase.wantErr {
					t.Errorf("GetVMNics() streamErr = %v, wantErr %v", err, testCase.wantErr)
				}

				if err != nil {
					break
				}

				got = append(got, diskID.GetValue())
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

//nolint:paralleltest
func Test_server_GetVMISOs(t *testing.T) {
	type args struct {
		vmID *cirrina.VMID
	}

	tests := []struct {
		name        string
		mockClosure func()
		want        []string
		args        args
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "c95064e9-c4c0-4308-85f3-508140490981",
					Name: "test2024082405",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 812,
						},
					},
					ISOs: func() []*iso.ISO {
						testIso := iso.ISO{
							ID: "389640e0-225d-4b86-9e4d-e09b415cf0d7",
						}

						testISOList := []*iso.ISO{&testIso}

						return testISOList
					}(),
				}
				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: func() *cirrina.VMID {
					r := cirrina.VMID{Value: "c95064e9-c4c0-4308-85f3-508140490981"}

					return &r
				}(),
			},
			want:    []string{"389640e0-225d-4b86-9e4d-e09b415cf0d7"},
			wantErr: false,
		},
		{
			name: "IsoIsNil",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "c95064e9-c4c0-4308-85f3-508140490981",
					Name: "test2024082405",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 812,
						},
					},
					ISOs: func() []*iso.ISO {
						testISOList := []*iso.ISO{nil}

						return testISOList
					}(),
				}
				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: func() *cirrina.VMID {
					r := cirrina.VMID{Value: "c95064e9-c4c0-4308-85f3-508140490981"}

					return &r
				}(),
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "VMNameEmpty",
			mockClosure: func() {
				testVM1 := vm.VM{
					ID:   "c95064e9-c4c0-4308-85f3-508140490981",
					Name: "",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 812,
						},
					},
				}
				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: func() *cirrina.VMID {
					r := cirrina.VMID{Value: "c95064e9-c4c0-4308-85f3-508140490981"}

					return &r
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "VMNotFound",
			mockClosure: func() {
				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				vmID: func() *cirrina.VMID {
					r := cirrina.VMID{Value: "c95064e9-c4c0-4308-85f3-508140490999"}

					return &r
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "BadUuid",
			mockClosure: func() {
				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				vmID: func() *cirrina.VMID {
					r := cirrina.VMID{Value: "c95064e9-c4c0-4308-85f"}

					return &r
				}(),
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()

			lis := bufconn.Listen(1024 * 1024)
			s := grpc.NewServer()
			reflection.Register(s)
			cirrina.RegisterVMInfoServer(s, &server{})

			go func() {
				if err := s.Serve(lis); err != nil {
					log.Fatalf("Server exited with error: %v", err)
				}
			}()

			resolver.SetDefaultScheme("passthrough")

			conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}

			defer func(conn *grpc.ClientConn) {
				_ = conn.Close()
			}(conn)

			client := cirrina.NewVMInfoClient(conn)

			var res cirrina.VMInfo_GetVMISOsClient

			var isoID *cirrina.ISOID

			var got []string

			res, err = client.GetVMISOs(context.Background(), testCase.args.vmID)
			if err != nil {
				t.Fatalf("GetVMDisks() error setting up client, error = %v", err)
			}

			for {
				isoID, err = res.Recv()
				if !errors.Is(err, io.EOF) && (err != nil) != testCase.wantErr {
					t.Errorf("GetVMNics() streamErr = %v, wantErr %v", err, testCase.wantErr)
				}

				if err != nil {
					break
				}

				got = append(got, isoID.GetValue())
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

//nolint:paralleltest,gocognit
func Test_server_AddVM(t *testing.T) {
	type args struct {
		vmConfig *cirrina.VMConfig
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *cirrina.VMID
		wantErr     bool
		wantPath    bool
		wantPathErr bool
	}{
		{
			name:        "Success",
			mockCmdFunc: "Test_server_AddVMSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `vms` (`created_at`,`updated_at`,`deleted_at`,`name`,`description`,`status`,`bhyve_pid`,`vnc_port`,`debug_port`,`com1_dev`,`com2_dev`,`com3_dev`,`com4_dev`,`id`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING `id`", //nolint:lll
					),
				).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "test2024082406", "", "STOPPED", 0, 0, 0, "", "", "", "", sqlmock.AnyArg()). //nolint:lll
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow("c9478af2-8a18-4a86-8234-5be5ceb80d95"))
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `configs` (`created_at`,`updated_at`,`deleted_at`,`vm_id`,`cpu`,`mem`,`max_wait`,`restart`,`restart_delay`,`screen`,`screen_width`,`screen_height`,`vnc_wait`,`vnc_port`,`tablet`,`store_uefi_vars`,`utc_time`,`host_bridge`,`acpi`,`use_hlt`,`exit_on_pause`,`wire_guest_mem`,`destroy_power_off`,`ignore_unknown_msr`,`kbd_layout`,`auto_start`,`sound`,`sound_in`,`sound_out`,`com1`,`com1_dev`,`com1_log`,`com2`,`com2_dev`,`com2_log`,`com3`,`com3_dev`,`com3_log`,`com4`,`com4_dev`,`com4_log`,`extra_args`,`com1_speed`,`com2_speed`,`com3_speed`,`com4_speed`,`auto_start_delay`,`debug`,`debug_wait`,`debug_port`,`priority`,`protect`,`pcpu`,`rbps`,`wbps`,`riops`,`wiops`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT (`id`) DO UPDATE SET `vm_id`=`excluded`.`vm_id` RETURNING `id`", //nolint:lll
					),
				).WithArgs(
					sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "c9478af2-8a18-4a86-8234-5be5ceb80d95", 1, 128, 120, true, 1, true, 1920, 1080, false, "AUTO", true, true, true, true, true, true, true, false, true, true, "default", false, false, "/dev/dsp0", "/dev/dsp0", true, "AUTO", false, false, "AUTO", false, false, "AUTO", false, false, "AUTO", false, "", 115200, 115200, 115200, 115200, 0, false, false, "AUTO", 0, true, 0, 0, 0, 0, 0). //nolint:lll
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow("489"))
				mock.ExpectCommit()
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Name: func() *string { n := "test2024082406"; return &n }(), //nolint:nlreturn
				},
			},
			want: func() *cirrina.VMID {
				r := cirrina.VMID{Value: "c9478af2-8a18-4a86-8234-5be5ceb80d95"}

				return &r
			}(),
			wantErr:  false,
			wantPath: true,
		},
		{
			name:        "ErrorCreating",
			mockCmdFunc: "Test_server_AddVMSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `vms` (`created_at`,`updated_at`,`deleted_at`,`name`,`description`,`status`,`bhyve_pid`,`vnc_port`,`debug_port`,`com1_dev`,`com2_dev`,`com3_dev`,`com4_dev`,`id`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING `id`", //nolint:lll
					),
				).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "test2024082407", "", "STOPPED", 0, 0, 0, "", "", "", "", sqlmock.AnyArg()). //nolint:lll
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow("c9478af2-8a18-4a86-8234-5be5ceb80d95"))
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `configs` (`created_at`,`updated_at`,`deleted_at`,`vm_id`,`cpu`,`mem`,`max_wait`,`restart`,`restart_delay`,`screen`,`screen_width`,`screen_height`,`vnc_wait`,`vnc_port`,`tablet`,`store_uefi_vars`,`utc_time`,`host_bridge`,`acpi`,`use_hlt`,`exit_on_pause`,`wire_guest_mem`,`destroy_power_off`,`ignore_unknown_msr`,`kbd_layout`,`auto_start`,`sound`,`sound_in`,`sound_out`,`com1`,`com1_dev`,`com1_log`,`com2`,`com2_dev`,`com2_log`,`com3`,`com3_dev`,`com3_log`,`com4`,`com4_dev`,`com4_log`,`extra_args`,`com1_speed`,`com2_speed`,`com3_speed`,`com4_speed`,`auto_start_delay`,`debug`,`debug_wait`,`debug_port`,`priority`,`protect`,`pcpu`,`rbps`,`wbps`,`riops`,`wiops`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT (`id`) DO UPDATE SET `vm_id`=`excluded`.`vm_id` RETURNING `id`", //nolint:lll
					),
				).WithArgs(
					sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "c9478af2-8a18-4a86-8234-5be5ceb80d95", 1, 128, 120, true, 1, true, 1920, 1080, false, "AUTO", true, true, true, true, true, true, true, false, true, true, "default", false, false, "/dev/dsp0", "/dev/dsp0", true, "AUTO", false, false, "AUTO", false, false, "AUTO", false, false, "AUTO", false, "", 115200, 115200, 115200, 115200, 0, false, false, "AUTO", 0, true, 0, 0, 0, 0, 0). //nolint:lll
					WillReturnError(gorm.ErrInvalidData)
				mock.ExpectRollback()
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Name: func() *string { n := "test2024082407"; return &n }(), //nolint:nlreturn
				},
			},
			want:     nil,
			wantErr:  true,
			wantPath: true,
		},
		{
			name:        "ErrorNilName",
			mockCmdFunc: "Test_server_AddVMSuccess",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Name: nil,
				},
			},
			want:     nil,
			wantErr:  true,
			wantPath: true,
		},
		{
			name:        "ErrorInvalidName",
			mockCmdFunc: "Test_server_AddVMSuccess",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Name: func() *string { n := "test202408240!7"; return &n }(), //nolint:nlreturn
				},
			},
			want:     nil,
			wantErr:  true,
			wantPath: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)

			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			vm.PathExistsFunc = func(_ string) (bool, error) {
				if testCase.wantPathErr {
					return true, errors.New("another error") //nolint:goerr113
				}

				if testCase.wantPath {
					return true, nil
				}

				return false, nil
			}

			t.Cleanup(func() { vm.PathExistsFunc = util.PathExists })

			vm.OsOpenFileFunc = func(_ string, _ int, _ os.FileMode) (*os.File, error) {
				o := os.File{}

				return &o, nil
			}

			t.Cleanup(func() { vm.OsOpenFileFunc = os.OpenFile })

			testDB, mockDB := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mockDB)

			lis := bufconn.Listen(1024 * 1024)
			s := grpc.NewServer()
			reflection.Register(s)
			cirrina.RegisterVMInfoServer(s, &server{})

			go func() {
				if err := s.Serve(lis); err != nil {
					log.Fatalf("Server exited with error: %v", err)
				}
			}()

			resolver.SetDefaultScheme("passthrough")

			conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}

			defer func(conn *grpc.ClientConn) {
				_ = conn.Close()
			}(conn)

			client := cirrina.NewVMInfoClient(conn)

			var got *cirrina.VMID

			got, err = client.AddVM(context.Background(), testCase.args.vmConfig)
			if (err != nil) != testCase.wantErr {
				t.Errorf("AddVM() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			mockDB.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mockDB.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

//nolint:paralleltest,maintidx
func Test_server_DeleteVM(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		vmID *cirrina.VMID
	}

	tests := []struct {
		name        string
		args        args
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        *cirrina.RequestID
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:   "0a973853-bd42-476d-8701-be07cab19895",
					Name: "test2024082304",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 340,
						},
						VMID: "0a973853-bd42-476d-8701-be07cab19895",
						CPU:  2,
						Mem:  1024,
					},
					Status:    vm.STOPPED,
					VNCPort:   0,
					DebugPort: 0,
				}
				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL",
					),
				).
					WithArgs(false).
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"started_at",
								"successful",
								"complete",
								"type",
								"data",
							}),
					)

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `requests` (`created_at`,`updated_at`,`deleted_at`,`started_at`,`successful`,`complete`,`type`,`data`,`id`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`", //nolint:lll
					),
				).
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, false, false, "VMDELETE", "{\"vm_id\":\"0a973853-bd42-476d-8701-be07cab19895\"}", sqlmock.AnyArg(), //nolint:lll
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("019185cb-7ebb-7882-8e32-96ca3041d3f4"))
				mock.ExpectCommit()
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "0a973853-bd42-476d-8701-be07cab19895",
				},
			},
			want: func() *cirrina.RequestID {
				r := cirrina.RequestID{Value: "019185cb-7ebb-7882-8e32-96ca3041d3f4"}

				return &r
			}(),
			wantErr: false,
		},
		{
			name: "ErrorSavingRequest",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:   "0a973853-bd42-476d-8701-be07cab19895",
					Name: "test2024082304",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 340,
						},
						VMID: "0a973853-bd42-476d-8701-be07cab19895",
						CPU:  2,
						Mem:  1024,
					},
					Status:    vm.STOPPED,
					VNCPort:   0,
					DebugPort: 0,
				}
				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL",
					),
				).
					WithArgs(false).
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"started_at",
								"successful",
								"complete",
								"type",
								"data",
							}),
					)

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `requests` (`created_at`,`updated_at`,`deleted_at`,`started_at`,`successful`,`complete`,`type`,`data`,`id`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`", //nolint:lll
					),
				).
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, false, false, "VMDELETE", "{\"vm_id\":\"0a973853-bd42-476d-8701-be07cab19895\"}", sqlmock.AnyArg(), //nolint:lll
					).
					WillReturnError(gorm.ErrInvalidData)
				mock.ExpectRollback()
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "0a973853-bd42-476d-8701-be07cab19895",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "VMNotStopped",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:   "0a973853-bd42-476d-8701-be07cab19895",
					Name: "test2024082304",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 340,
						},
						VMID: "0a973853-bd42-476d-8701-be07cab19895",
						CPU:  2,
						Mem:  1024,
					},
					Status:    vm.RUNNING,
					VNCPort:   0,
					DebugPort: 0,
				}
				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL",
					),
				).
					WithArgs(false).
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"started_at",
								"successful",
								"complete",
								"type",
								"data",
							}),
					)
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "0a973853-bd42-476d-8701-be07cab19895",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ReqPendingForVM",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:   "0a973853-bd42-476d-8701-be07cab19895",
					Name: "test2024082304",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 340,
						},
						VMID: "0a973853-bd42-476d-8701-be07cab19895",
						CPU:  2,
						Mem:  1024,
					},
					Status:    vm.STOPPED,
					VNCPort:   0,
					DebugPort: 0,
				}
				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL",
					),
				).
					WithArgs(false).
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"started_at",
								"successful",
								"complete",
								"type",
								"data",
							}).
							AddRow(
								"b27e40d2-63f4-4457-983b-765dfdf9b1da",
								createUpdateTime,
								createUpdateTime,
								nil,
								nil,
								0,
								0,
								"VMSTART",
								"{\"vm_id\":\"0a973853-bd42-476d-8701-be07cab19895\"}",
							),
					)
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "0a973853-bd42-476d-8701-be07cab19895",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "VMNameEmpty",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:   "0a973853-bd42-476d-8701-be07cab19895",
					Name: "",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 340,
						},
						VMID: "0a973853-bd42-476d-8701-be07cab19895",
						CPU:  2,
						Mem:  1024,
					},
					Status:    vm.STOPPED,
					VNCPort:   0,
					DebugPort: 0,
				}
				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "0a973853-bd42-476d-8701-be07cab19895",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "VMNotFound",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "0a973853-bd42-476d-8701-be07cab19895",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "BadUuid",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "0a973853-bd42-476d-8701-be07",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mockDB := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mockDB)

			lis := bufconn.Listen(1024 * 1024)
			s := grpc.NewServer()
			reflection.Register(s)
			cirrina.RegisterVMInfoServer(s, &server{})

			go func() {
				if err := s.Serve(lis); err != nil {
					log.Fatalf("Server exited with error: %v", err)
				}
			}()

			resolver.SetDefaultScheme("passthrough")

			conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}

			defer func(conn *grpc.ClientConn) {
				_ = conn.Close()
			}(conn)

			client := cirrina.NewVMInfoClient(conn)

			var got *cirrina.RequestID

			got, err = client.DeleteVM(context.Background(), testCase.args.vmID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("DeleteVM() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			mockDB.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mockDB.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

//nolint:paralleltest,maintidx
func Test_server_UpdateVM(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		vmConfig *cirrina.VMConfig
	}

	tests := []struct {
		name                  string
		mockCmdFunc           string
		mockClosure           func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		GetKbdLayoutNamesFunc func() []string
		args                  args
		want                  *cirrina.ReqBool
		wantErr               bool
	}{
		{
			name:        "Success",
			mockCmdFunc: "Test_server_UpdateVMSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:          "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					Name:        "test2024082408",
					Description: "test vm description",
					Status:      "STOPPED",
					BhyvePid:    0,
					VNCPort:     0,
					DebugPort:   0,
					Config: vm.Config{
						Model: gorm.Model{
							ID:        876,
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
						},
						VMID:             "f22416b8-4d21-4b29-a9dd-336fc6aca494",
						CPU:              2,
						Mem:              2048,
						MaxWait:          60,
						Restart:          false,
						Screen:           false,
						ScreenWidth:      1024,
						ScreenHeight:     768,
						Sound:            false,
						SoundIn:          "/dev/dsp1",
						SoundOut:         "/dev/dsp1",
						VNCPort:          "AUTO",
						Tablet:           true,
						StoreUEFIVars:    true,
						UTCTime:          true,
						HostBridge:       true,
						ACPI:             true,
						UseHLT:           true,
						ExitOnPause:      true,
						DestroyPowerOff:  true,
						IgnoreUnknownMSR: true,
						KbdLayout:        "default",
						Com1:             true,
						Com1Dev:          "AUTO",
						Com2Dev:          "AUTO",
						Com3Dev:          "AUTO",
						Com4Dev:          "AUTO",
						Com1Speed:        115200,
						Com2Speed:        115200,
						Com3Speed:        115200,
						Com4Speed:        115200,
						AutoStartDelay:   60,
						DebugPort:        "AUTO",
					},
					ISOs:  []*iso.ISO{},
					Disks: []*disk.Disk{},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(
						false,
						true,
						true,
						false,
						true,
						82,
						"/dev/nmdm-test2024082408-c1",
						false,
						9600,
						"/dev/nmdm-test2024082408-c2",
						true,
						9600,
						"/dev/nmdm-test2024082408-c3",
						true,
						9600,
						true,
						"/dev/nmdm-test2024082408-c4",
						true,
						9600,
						4,
						true,
						"7123",
						true,
						false,
						false,
						"-somejunk",
						false,
						false,
						"us_unix",
						1200,
						4096,
						11,
						12,
						true,
						1001,
						true,
						765,
						1003,
						true,
						1080,
						1920,
						true,
						"/dev/dsp0",
						"/dev/dsp0",
						false,
						false,
						false,
						false,
						"7900",
						true,
						1002,
						1004,
						true,
						sqlmock.AnyArg(),
						876).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "a test VM", "test2024082408", 0, sqlmock.AnyArg(), "f22416b8-4d21-4b29-a9dd-336fc6aca494"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("f22416b8-4d21-4b29-a9dd-336fc6aca494").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("f22416b8-4d21-4b29-a9dd-336fc6aca494").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			GetKbdLayoutNamesFunc: func() []string {
				return []string{
					"am",
					"be",
					"be_acc",
					"bg_bds",
					"bg_phonetic",
					"br",
					"br_noacc",
					"centraleuropean",
					"ch",
					"ch_acc",
					"ch_macbook_acc",
					"ch-fr",
					"ch-fr_acc",
					"cz",
					"de",
					"de_acc",
					"de_noacc",
					"default",
					"dk",
					"dk_macbook",
					"ee",
					"es",
					"es_acc",
					"es_dvorak",
					"fi",
					"fr",
					"fr_acc",
					"fr_dvorak",
					"fr_dvorak_acc",
					"fr_macbook",
					"gr",
					"gr_101_acc",
					"gr_elot_acc",
					"hr",
					"hu_101",
					"hu_102",
					"is",
					"is_acc",
					"it",
					"jp",
					"jp_capsctrl",
					"kz_io",
					"kz_kst",
					"latinamerican",
					"latinamerican_acc",
					"lt",
					"nl",
					"no",
					"no_dvorak",
					"nordic_asus-eee",
					"pl_dvorak",
					"pt",
					"pt_acc",
					"ru",
					"ru_shift",
					"ru_win",
					"se",
					"si",
					"tr",
					"tr_f",
					"ua",
					"ua_shift_alt",
					"uk",
					"uk_capsctrl",
					"uk_dvorak",
					"uk_macbook",
					"us_dvorak",
					"us_dvorakl",
					"us_dvorakp",
					"us_dvorakr",
					"us_dvorakx",
					"us_emacs",
					"us_unix",
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Id:             "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					Description:    func() *string { r := "a test VM"; return &r }(),                   //nolint:nlreturn
					Cpu:            func() *uint32 { var r uint32 = 4; return &r }(),                   //nolint:nlreturn
					Mem:            func() *uint32 { var r uint32 = 4096; return &r }(),                //nolint:nlreturn
					MaxWait:        func() *uint32 { var r uint32 = 1200; return &r }(),                //nolint:nlreturn
					Restart:        func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					RestartDelay:   func() *uint32 { var r uint32 = 765; return &r }(),                 //nolint:nlreturn
					Screen:         func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					ScreenWidth:    func() *uint32 { var r uint32 = 1920; return &r }(),                //nolint:nlreturn
					ScreenHeight:   func() *uint32 { var r uint32 = 1080; return &r }(),                //nolint:nlreturn
					Vncwait:        func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Wireguestmem:   func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Tablet:         func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Storeuefi:      func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Utc:            func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Hostbridge:     func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Acpi:           func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Hlt:            func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Eop:            func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Dpo:            func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Ium:            func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Vncport:        func() *string { r := "7900"; return &r }(),                        //nolint:nlreturn
					Keyboard:       func() *string { r := "us_unix"; return &r }(),                     //nolint:nlreturn
					Autostart:      func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Sound:          func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					SoundIn:        func() *string { r := "/dev/dsp0"; return &r }(),                   //nolint:nlreturn
					SoundOut:       func() *string { r := "/dev/dsp0"; return &r }(),                   //nolint:nlreturn
					Com1:           func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Com1Dev:        func() *string { r := "/dev/nmdm-test2024082408-c1"; return &r }(), //nolint:nlreturn
					Com2:           func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Com2Dev:        func() *string { r := "/dev/nmdm-test2024082408-c2"; return &r }(), //nolint:nlreturn
					Com3:           func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Com3Dev:        func() *string { r := "/dev/nmdm-test2024082408-c3"; return &r }(), //nolint:nlreturn
					Com4:           func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Com4Dev:        func() *string { r := "/dev/nmdm-test2024082408-c4"; return &r }(), //nolint:nlreturn
					ExtraArgs:      func() *string { r := "-somejunk"; return &r }(),                   //nolint:nlreturn
					Com1Log:        func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Com2Log:        func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Com3Log:        func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Com4Log:        func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Com1Speed:      func() *uint32 { var r uint32 = 9600; return &r }(),                //nolint:nlreturn
					Com2Speed:      func() *uint32 { var r uint32 = 9600; return &r }(),                //nolint:nlreturn
					Com3Speed:      func() *uint32 { var r uint32 = 9600; return &r }(),                //nolint:nlreturn
					Com4Speed:      func() *uint32 { var r uint32 = 9600; return &r }(),                //nolint:nlreturn
					AutostartDelay: func() *uint32 { var r uint32 = 82; return &r }(),                  //nolint:nlreturn
					Debug:          func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					DebugWait:      func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					DebugPort:      func() *string { r := "7123"; return &r }(),                        //nolint:nlreturn
					Priority:       func() *int32 { var r int32 = 12; return &r }(),                    //nolint:nlreturn
					Protect:        func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Pcpu:           func() *uint32 { var r uint32 = 11; return &r }(),                  //nolint:nlreturn
					Rbps:           func() *uint32 { var r uint32 = 1001; return &r }(),                //nolint:nlreturn
					Wbps:           func() *uint32 { var r uint32 = 1002; return &r }(),                //nolint:nlreturn
					Riops:          func() *uint32 { var r uint32 = 1003; return &r }(),                //nolint:nlreturn
					Wiops:          func() *uint32 { var r uint32 = 1004; return &r }(),                //nolint:nlreturn
				},
			},
			want: func() *cirrina.ReqBool {
				r := cirrina.ReqBool{
					Success: true,
				}

				return &r
			}(),
			wantErr: false,
		},
		{
			name:        "BadDebugPort",
			mockCmdFunc: "Test_server_UpdateVMSuccess",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:          "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					Name:        "test2024082408",
					Description: "test vm description",
					Status:      "STOPPED",
					BhyvePid:    0,
					VNCPort:     0,
					DebugPort:   0,
					Config: vm.Config{
						Model: gorm.Model{
							ID:        876,
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
						},
						VMID:             "f22416b8-4d21-4b29-a9dd-336fc6aca494",
						CPU:              2,
						Mem:              2048,
						MaxWait:          60,
						Restart:          false,
						Screen:           false,
						ScreenWidth:      1024,
						ScreenHeight:     768,
						Sound:            false,
						SoundIn:          "/dev/dsp1",
						SoundOut:         "/dev/dsp1",
						VNCPort:          "AUTO",
						Tablet:           true,
						StoreUEFIVars:    true,
						UTCTime:          true,
						HostBridge:       true,
						ACPI:             true,
						UseHLT:           true,
						ExitOnPause:      true,
						DestroyPowerOff:  true,
						IgnoreUnknownMSR: true,
						KbdLayout:        "default",
						Com1:             true,
						Com1Dev:          "AUTO",
						Com2Dev:          "AUTO",
						Com3Dev:          "AUTO",
						Com4Dev:          "AUTO",
						Com1Speed:        115200,
						Com2Speed:        115200,
						Com3Speed:        115200,
						Com4Speed:        115200,
						AutoStartDelay:   60,
						DebugPort:        "AUTO",
					},
					ISOs:  []*iso.ISO{},
					Disks: []*disk.Disk{},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			GetKbdLayoutNamesFunc: func() []string {
				return []string{
					"am",
					"be",
					"be_acc",
					"bg_bds",
					"bg_phonetic",
					"br",
					"br_noacc",
					"centraleuropean",
					"ch",
					"ch_acc",
					"ch_macbook_acc",
					"ch-fr",
					"ch-fr_acc",
					"cz",
					"de",
					"de_acc",
					"de_noacc",
					"default",
					"dk",
					"dk_macbook",
					"ee",
					"es",
					"es_acc",
					"es_dvorak",
					"fi",
					"fr",
					"fr_acc",
					"fr_dvorak",
					"fr_dvorak_acc",
					"fr_macbook",
					"gr",
					"gr_101_acc",
					"gr_elot_acc",
					"hr",
					"hu_101",
					"hu_102",
					"is",
					"is_acc",
					"it",
					"jp",
					"jp_capsctrl",
					"kz_io",
					"kz_kst",
					"latinamerican",
					"latinamerican_acc",
					"lt",
					"nl",
					"no",
					"no_dvorak",
					"nordic_asus-eee",
					"pl_dvorak",
					"pt",
					"pt_acc",
					"ru",
					"ru_shift",
					"ru_win",
					"se",
					"si",
					"tr",
					"tr_f",
					"ua",
					"ua_shift_alt",
					"uk",
					"uk_capsctrl",
					"uk_dvorak",
					"uk_macbook",
					"us_dvorak",
					"us_dvorakl",
					"us_dvorakp",
					"us_dvorakr",
					"us_dvorakx",
					"us_emacs",
					"us_unix",
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Id:        "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					DebugPort: func() *string { r := "99999"; return &r }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "BadAutostartDelay",
			mockCmdFunc: "Test_server_UpdateVMSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:          "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					Name:        "test2024082408",
					Description: "test vm description",
					Status:      "STOPPED",
					BhyvePid:    0,
					VNCPort:     0,
					DebugPort:   0,
					Config: vm.Config{
						Model: gorm.Model{
							ID:        876,
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
						},
						VMID:             "f22416b8-4d21-4b29-a9dd-336fc6aca494",
						CPU:              2,
						Mem:              2048,
						MaxWait:          60,
						Restart:          false,
						Screen:           false,
						ScreenWidth:      1024,
						ScreenHeight:     768,
						Sound:            false,
						SoundIn:          "/dev/dsp1",
						SoundOut:         "/dev/dsp1",
						VNCPort:          "AUTO",
						Tablet:           true,
						StoreUEFIVars:    true,
						UTCTime:          true,
						HostBridge:       true,
						ACPI:             true,
						UseHLT:           true,
						ExitOnPause:      true,
						DestroyPowerOff:  true,
						IgnoreUnknownMSR: true,
						KbdLayout:        "default",
						Com1:             true,
						Com1Dev:          "AUTO",
						Com2Dev:          "AUTO",
						Com3Dev:          "AUTO",
						Com4Dev:          "AUTO",
						Com1Speed:        115200,
						Com2Speed:        115200,
						Com3Speed:        115200,
						Com4Speed:        115200,
						AutoStartDelay:   60,
						DebugPort:        "AUTO",
					},
					ISOs:  []*iso.ISO{},
					Disks: []*disk.Disk{},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(
						false,
						true,
						true,
						false,
						true,
						3600,
						"/dev/nmdm-test2024082408-c1",
						false,
						9600,
						"/dev/nmdm-test2024082408-c2",
						true,
						9600,
						"/dev/nmdm-test2024082408-c3",
						true,
						9600,
						true,
						"/dev/nmdm-test2024082408-c4",
						true,
						9600,
						4,
						true,
						"7123",
						true,
						false,
						false,
						"-somejunk",
						false,
						false,
						"us_unix",
						1200,
						4096,
						11,
						12,
						true,
						1001,
						true,
						765,
						1003,
						true,
						1080,
						1920,
						true,
						"/dev/dsp0",
						"/dev/dsp0",
						false,
						false,
						false,
						false,
						"7900",
						true,
						1002,
						1004,
						true,
						sqlmock.AnyArg(),
						876).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "a test VM", "test2024082408", 0, sqlmock.AnyArg(), "f22416b8-4d21-4b29-a9dd-336fc6aca494"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("f22416b8-4d21-4b29-a9dd-336fc6aca494").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("f22416b8-4d21-4b29-a9dd-336fc6aca494").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			GetKbdLayoutNamesFunc: func() []string {
				return []string{
					"am",
					"be",
					"be_acc",
					"bg_bds",
					"bg_phonetic",
					"br",
					"br_noacc",
					"centraleuropean",
					"ch",
					"ch_acc",
					"ch_macbook_acc",
					"ch-fr",
					"ch-fr_acc",
					"cz",
					"de",
					"de_acc",
					"de_noacc",
					"default",
					"dk",
					"dk_macbook",
					"ee",
					"es",
					"es_acc",
					"es_dvorak",
					"fi",
					"fr",
					"fr_acc",
					"fr_dvorak",
					"fr_dvorak_acc",
					"fr_macbook",
					"gr",
					"gr_101_acc",
					"gr_elot_acc",
					"hr",
					"hu_101",
					"hu_102",
					"is",
					"is_acc",
					"it",
					"jp",
					"jp_capsctrl",
					"kz_io",
					"kz_kst",
					"latinamerican",
					"latinamerican_acc",
					"lt",
					"nl",
					"no",
					"no_dvorak",
					"nordic_asus-eee",
					"pl_dvorak",
					"pt",
					"pt_acc",
					"ru",
					"ru_shift",
					"ru_win",
					"se",
					"si",
					"tr",
					"tr_f",
					"ua",
					"ua_shift_alt",
					"uk",
					"uk_capsctrl",
					"uk_dvorak",
					"uk_macbook",
					"us_dvorak",
					"us_dvorakl",
					"us_dvorakp",
					"us_dvorakr",
					"us_dvorakx",
					"us_emacs",
					"us_unix",
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Id:             "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					Description:    func() *string { r := "a test VM"; return &r }(),                   //nolint:nlreturn
					Cpu:            func() *uint32 { var r uint32 = 4; return &r }(),                   //nolint:nlreturn
					Mem:            func() *uint32 { var r uint32 = 4096; return &r }(),                //nolint:nlreturn
					MaxWait:        func() *uint32 { var r uint32 = 1200; return &r }(),                //nolint:nlreturn
					Restart:        func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					RestartDelay:   func() *uint32 { var r uint32 = 765; return &r }(),                 //nolint:nlreturn
					Screen:         func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					ScreenWidth:    func() *uint32 { var r uint32 = 1920; return &r }(),                //nolint:nlreturn
					ScreenHeight:   func() *uint32 { var r uint32 = 1080; return &r }(),                //nolint:nlreturn
					Vncwait:        func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Wireguestmem:   func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Tablet:         func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Storeuefi:      func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Utc:            func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Hostbridge:     func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Acpi:           func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Hlt:            func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Eop:            func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Dpo:            func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Ium:            func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Vncport:        func() *string { r := "7900"; return &r }(),                        //nolint:nlreturn
					Keyboard:       func() *string { r := "us_unix"; return &r }(),                     //nolint:nlreturn
					Autostart:      func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Sound:          func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					SoundIn:        func() *string { r := "/dev/dsp0"; return &r }(),                   //nolint:nlreturn
					SoundOut:       func() *string { r := "/dev/dsp0"; return &r }(),                   //nolint:nlreturn
					Com1:           func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Com1Dev:        func() *string { r := "/dev/nmdm-test2024082408-c1"; return &r }(), //nolint:nlreturn
					Com2:           func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Com2Dev:        func() *string { r := "/dev/nmdm-test2024082408-c2"; return &r }(), //nolint:nlreturn
					Com3:           func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Com3Dev:        func() *string { r := "/dev/nmdm-test2024082408-c3"; return &r }(), //nolint:nlreturn
					Com4:           func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Com4Dev:        func() *string { r := "/dev/nmdm-test2024082408-c4"; return &r }(), //nolint:nlreturn
					ExtraArgs:      func() *string { r := "-somejunk"; return &r }(),                   //nolint:nlreturn
					Com1Log:        func() *bool { r := false; return &r }(),                           //nolint:nlreturn
					Com2Log:        func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Com3Log:        func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Com4Log:        func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Com1Speed:      func() *uint32 { var r uint32 = 9600; return &r }(),                //nolint:nlreturn
					Com2Speed:      func() *uint32 { var r uint32 = 9600; return &r }(),                //nolint:nlreturn
					Com3Speed:      func() *uint32 { var r uint32 = 9600; return &r }(),                //nolint:nlreturn
					Com4Speed:      func() *uint32 { var r uint32 = 9600; return &r }(),                //nolint:nlreturn
					AutostartDelay: func() *uint32 { var r uint32 = 99999; return &r }(),               //nolint:nlreturn
					Debug:          func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					DebugWait:      func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					DebugPort:      func() *string { r := "7123"; return &r }(),                        //nolint:nlreturn
					Priority:       func() *int32 { var r int32 = 12; return &r }(),                    //nolint:nlreturn
					Protect:        func() *bool { r := true; return &r }(),                            //nolint:nlreturn
					Pcpu:           func() *uint32 { var r uint32 = 11; return &r }(),                  //nolint:nlreturn
					Rbps:           func() *uint32 { var r uint32 = 1001; return &r }(),                //nolint:nlreturn
					Wbps:           func() *uint32 { var r uint32 = 1002; return &r }(),                //nolint:nlreturn
					Riops:          func() *uint32 { var r uint32 = 1003; return &r }(),                //nolint:nlreturn
					Wiops:          func() *uint32 { var r uint32 = 1004; return &r }(),                //nolint:nlreturn
				},
			},
			want: func() *cirrina.ReqBool {
				r := cirrina.ReqBool{
					Success: true,
				}

				return &r
			}(),
			wantErr: false,
		},
		{
			name:        "BadSoundIn",
			mockCmdFunc: "Test_server_UpdateVMSuccess",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:          "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					Name:        "test2024082408",
					Description: "test vm description",
					Status:      "STOPPED",
					BhyvePid:    0,
					VNCPort:     0,
					DebugPort:   0,
					Config: vm.Config{
						Model: gorm.Model{
							ID:        876,
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
						},
						VMID:             "f22416b8-4d21-4b29-a9dd-336fc6aca494",
						CPU:              2,
						Mem:              2048,
						MaxWait:          60,
						Restart:          false,
						Screen:           false,
						ScreenWidth:      1024,
						ScreenHeight:     768,
						Sound:            false,
						SoundIn:          "/dev/dsp1",
						SoundOut:         "/dev/dsp1",
						VNCPort:          "AUTO",
						Tablet:           true,
						StoreUEFIVars:    true,
						UTCTime:          true,
						HostBridge:       true,
						ACPI:             true,
						UseHLT:           true,
						ExitOnPause:      true,
						DestroyPowerOff:  true,
						IgnoreUnknownMSR: true,
						KbdLayout:        "default",
						Com1:             true,
						Com1Dev:          "AUTO",
						Com2Dev:          "AUTO",
						Com3Dev:          "AUTO",
						Com4Dev:          "AUTO",
						Com1Speed:        115200,
						Com2Speed:        115200,
						Com3Speed:        115200,
						Com4Speed:        115200,
						AutoStartDelay:   60,
						DebugPort:        "AUTO",
					},
					ISOs:  []*iso.ISO{},
					Disks: []*disk.Disk{},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			GetKbdLayoutNamesFunc: func() []string {
				return []string{
					"am",
					"be",
					"be_acc",
					"bg_bds",
					"bg_phonetic",
					"br",
					"br_noacc",
					"centraleuropean",
					"ch",
					"ch_acc",
					"ch_macbook_acc",
					"ch-fr",
					"ch-fr_acc",
					"cz",
					"de",
					"de_acc",
					"de_noacc",
					"default",
					"dk",
					"dk_macbook",
					"ee",
					"es",
					"es_acc",
					"es_dvorak",
					"fi",
					"fr",
					"fr_acc",
					"fr_dvorak",
					"fr_dvorak_acc",
					"fr_macbook",
					"gr",
					"gr_101_acc",
					"gr_elot_acc",
					"hr",
					"hu_101",
					"hu_102",
					"is",
					"is_acc",
					"it",
					"jp",
					"jp_capsctrl",
					"kz_io",
					"kz_kst",
					"latinamerican",
					"latinamerican_acc",
					"lt",
					"nl",
					"no",
					"no_dvorak",
					"nordic_asus-eee",
					"pl_dvorak",
					"pt",
					"pt_acc",
					"ru",
					"ru_shift",
					"ru_win",
					"se",
					"si",
					"tr",
					"tr_f",
					"ua",
					"ua_shift_alt",
					"uk",
					"uk_capsctrl",
					"uk_dvorak",
					"uk_macbook",
					"us_dvorak",
					"us_dvorakl",
					"us_dvorakp",
					"us_dvorakr",
					"us_dvorakx",
					"us_emacs",
					"us_unix",
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Id:      "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					SoundIn: func() *string { r := "junk"; return &r }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "BadSoundOut",
			mockCmdFunc: "Test_server_UpdateVMSuccess",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:          "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					Name:        "test2024082408",
					Description: "test vm description",
					Status:      "STOPPED",
					BhyvePid:    0,
					VNCPort:     0,
					DebugPort:   0,
					Config: vm.Config{
						Model: gorm.Model{
							ID:        876,
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
						},
						VMID:             "f22416b8-4d21-4b29-a9dd-336fc6aca494",
						CPU:              2,
						Mem:              2048,
						MaxWait:          60,
						Restart:          false,
						Screen:           false,
						ScreenWidth:      1024,
						ScreenHeight:     768,
						Sound:            false,
						SoundIn:          "/dev/dsp1",
						SoundOut:         "/dev/dsp1",
						VNCPort:          "AUTO",
						Tablet:           true,
						StoreUEFIVars:    true,
						UTCTime:          true,
						HostBridge:       true,
						ACPI:             true,
						UseHLT:           true,
						ExitOnPause:      true,
						DestroyPowerOff:  true,
						IgnoreUnknownMSR: true,
						KbdLayout:        "default",
						Com1:             true,
						Com1Dev:          "AUTO",
						Com2Dev:          "AUTO",
						Com3Dev:          "AUTO",
						Com4Dev:          "AUTO",
						Com1Speed:        115200,
						Com2Speed:        115200,
						Com3Speed:        115200,
						Com4Speed:        115200,
						AutoStartDelay:   60,
						DebugPort:        "AUTO",
					},
					ISOs:  []*iso.ISO{},
					Disks: []*disk.Disk{},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			GetKbdLayoutNamesFunc: func() []string {
				return []string{
					"am",
					"be",
					"be_acc",
					"bg_bds",
					"bg_phonetic",
					"br",
					"br_noacc",
					"centraleuropean",
					"ch",
					"ch_acc",
					"ch_macbook_acc",
					"ch-fr",
					"ch-fr_acc",
					"cz",
					"de",
					"de_acc",
					"de_noacc",
					"default",
					"dk",
					"dk_macbook",
					"ee",
					"es",
					"es_acc",
					"es_dvorak",
					"fi",
					"fr",
					"fr_acc",
					"fr_dvorak",
					"fr_dvorak_acc",
					"fr_macbook",
					"gr",
					"gr_101_acc",
					"gr_elot_acc",
					"hr",
					"hu_101",
					"hu_102",
					"is",
					"is_acc",
					"it",
					"jp",
					"jp_capsctrl",
					"kz_io",
					"kz_kst",
					"latinamerican",
					"latinamerican_acc",
					"lt",
					"nl",
					"no",
					"no_dvorak",
					"nordic_asus-eee",
					"pl_dvorak",
					"pt",
					"pt_acc",
					"ru",
					"ru_shift",
					"ru_win",
					"se",
					"si",
					"tr",
					"tr_f",
					"ua",
					"ua_shift_alt",
					"uk",
					"uk_capsctrl",
					"uk_dvorak",
					"uk_macbook",
					"us_dvorak",
					"us_dvorakl",
					"us_dvorakp",
					"us_dvorakr",
					"us_dvorakx",
					"us_emacs",
					"us_unix",
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Id:       "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					SoundOut: func() *string { r := "junk"; return &r }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "BadVNCPort",
			mockCmdFunc: "Test_server_UpdateVMSuccess",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:          "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					Name:        "test2024082408",
					Description: "test vm description",
					Status:      "STOPPED",
					BhyvePid:    0,
					VNCPort:     0,
					DebugPort:   0,
					Config: vm.Config{
						Model: gorm.Model{
							ID:        876,
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
						},
						VMID:             "f22416b8-4d21-4b29-a9dd-336fc6aca494",
						CPU:              2,
						Mem:              2048,
						MaxWait:          60,
						Restart:          false,
						Screen:           false,
						ScreenWidth:      1024,
						ScreenHeight:     768,
						Sound:            false,
						SoundIn:          "/dev/dsp1",
						SoundOut:         "/dev/dsp1",
						VNCPort:          "AUTO",
						Tablet:           true,
						StoreUEFIVars:    true,
						UTCTime:          true,
						HostBridge:       true,
						ACPI:             true,
						UseHLT:           true,
						ExitOnPause:      true,
						DestroyPowerOff:  true,
						IgnoreUnknownMSR: true,
						KbdLayout:        "default",
						Com1:             true,
						Com1Dev:          "AUTO",
						Com2Dev:          "AUTO",
						Com3Dev:          "AUTO",
						Com4Dev:          "AUTO",
						Com1Speed:        115200,
						Com2Speed:        115200,
						Com3Speed:        115200,
						Com4Speed:        115200,
						AutoStartDelay:   60,
						DebugPort:        "AUTO",
					},
					ISOs:  []*iso.ISO{},
					Disks: []*disk.Disk{},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			GetKbdLayoutNamesFunc: func() []string {
				return []string{
					"am",
					"be",
					"be_acc",
					"bg_bds",
					"bg_phonetic",
					"br",
					"br_noacc",
					"centraleuropean",
					"ch",
					"ch_acc",
					"ch_macbook_acc",
					"ch-fr",
					"ch-fr_acc",
					"cz",
					"de",
					"de_acc",
					"de_noacc",
					"default",
					"dk",
					"dk_macbook",
					"ee",
					"es",
					"es_acc",
					"es_dvorak",
					"fi",
					"fr",
					"fr_acc",
					"fr_dvorak",
					"fr_dvorak_acc",
					"fr_macbook",
					"gr",
					"gr_101_acc",
					"gr_elot_acc",
					"hr",
					"hu_101",
					"hu_102",
					"is",
					"is_acc",
					"it",
					"jp",
					"jp_capsctrl",
					"kz_io",
					"kz_kst",
					"latinamerican",
					"latinamerican_acc",
					"lt",
					"nl",
					"no",
					"no_dvorak",
					"nordic_asus-eee",
					"pl_dvorak",
					"pt",
					"pt_acc",
					"ru",
					"ru_shift",
					"ru_win",
					"se",
					"si",
					"tr",
					"tr_f",
					"ua",
					"ua_shift_alt",
					"uk",
					"uk_capsctrl",
					"uk_dvorak",
					"uk_macbook",
					"us_dvorak",
					"us_dvorakl",
					"us_dvorakp",
					"us_dvorakr",
					"us_dvorakx",
					"us_emacs",
					"us_unix",
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Id:      "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					Vncport: func() *string { r := "99999"; return &r }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "BadKeymap",
			mockCmdFunc: "Test_server_UpdateVMSuccess",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:          "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					Name:        "test2024082408",
					Description: "test vm description",
					Status:      "STOPPED",
					BhyvePid:    0,
					VNCPort:     0,
					DebugPort:   0,
					Config: vm.Config{
						Model: gorm.Model{
							ID:        876,
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
						},
						VMID:             "f22416b8-4d21-4b29-a9dd-336fc6aca494",
						CPU:              2,
						Mem:              2048,
						MaxWait:          60,
						Restart:          false,
						Screen:           false,
						ScreenWidth:      1024,
						ScreenHeight:     768,
						Sound:            false,
						SoundIn:          "/dev/dsp1",
						SoundOut:         "/dev/dsp1",
						VNCPort:          "AUTO",
						Tablet:           true,
						StoreUEFIVars:    true,
						UTCTime:          true,
						HostBridge:       true,
						ACPI:             true,
						UseHLT:           true,
						ExitOnPause:      true,
						DestroyPowerOff:  true,
						IgnoreUnknownMSR: true,
						KbdLayout:        "default",
						Com1:             true,
						Com1Dev:          "AUTO",
						Com2Dev:          "AUTO",
						Com3Dev:          "AUTO",
						Com4Dev:          "AUTO",
						Com1Speed:        115200,
						Com2Speed:        115200,
						Com3Speed:        115200,
						Com4Speed:        115200,
						AutoStartDelay:   60,
						DebugPort:        "AUTO",
					},
					ISOs:  []*iso.ISO{},
					Disks: []*disk.Disk{},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			GetKbdLayoutNamesFunc: func() []string {
				return []string{
					"am",
					"be",
					"be_acc",
					"bg_bds",
					"bg_phonetic",
					"br",
					"br_noacc",
					"centraleuropean",
					"ch",
					"ch_acc",
					"ch_macbook_acc",
					"ch-fr",
					"ch-fr_acc",
					"cz",
					"de",
					"de_acc",
					"de_noacc",
					"default",
					"dk",
					"dk_macbook",
					"ee",
					"es",
					"es_acc",
					"es_dvorak",
					"fi",
					"fr",
					"fr_acc",
					"fr_dvorak",
					"fr_dvorak_acc",
					"fr_macbook",
					"gr",
					"gr_101_acc",
					"gr_elot_acc",
					"hr",
					"hu_101",
					"hu_102",
					"is",
					"is_acc",
					"it",
					"jp",
					"jp_capsctrl",
					"kz_io",
					"kz_kst",
					"latinamerican",
					"latinamerican_acc",
					"lt",
					"nl",
					"no",
					"no_dvorak",
					"nordic_asus-eee",
					"pl_dvorak",
					"pt",
					"pt_acc",
					"ru",
					"ru_shift",
					"ru_win",
					"se",
					"si",
					"tr",
					"tr_f",
					"ua",
					"ua_shift_alt",
					"uk",
					"uk_capsctrl",
					"uk_dvorak",
					"uk_macbook",
					"us_dvorak",
					"us_dvorakl",
					"us_dvorakp",
					"us_dvorakr",
					"us_dvorakx",
					"us_emacs",
					"us_unix",
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Id:       "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					Keyboard: func() *string { r := "junk"; return &r }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "BadCom1Dev",
			mockCmdFunc: "Test_server_UpdateVMSuccess",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:          "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					Name:        "test2024082408",
					Description: "test vm description",
					Status:      "STOPPED",
					BhyvePid:    0,
					VNCPort:     0,
					DebugPort:   0,
					Config: vm.Config{
						Model: gorm.Model{
							ID:        876,
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
						},
						VMID:             "f22416b8-4d21-4b29-a9dd-336fc6aca494",
						CPU:              2,
						Mem:              2048,
						MaxWait:          60,
						Restart:          false,
						Screen:           false,
						ScreenWidth:      1024,
						ScreenHeight:     768,
						Sound:            false,
						SoundIn:          "/dev/dsp1",
						SoundOut:         "/dev/dsp1",
						VNCPort:          "AUTO",
						Tablet:           true,
						StoreUEFIVars:    true,
						UTCTime:          true,
						HostBridge:       true,
						ACPI:             true,
						UseHLT:           true,
						ExitOnPause:      true,
						DestroyPowerOff:  true,
						IgnoreUnknownMSR: true,
						KbdLayout:        "default",
						Com1:             true,
						Com1Dev:          "AUTO",
						Com2Dev:          "AUTO",
						Com3Dev:          "AUTO",
						Com4Dev:          "AUTO",
						Com1Speed:        115200,
						Com2Speed:        115200,
						Com3Speed:        115200,
						Com4Speed:        115200,
						AutoStartDelay:   60,
						DebugPort:        "AUTO",
					},
					ISOs:  []*iso.ISO{},
					Disks: []*disk.Disk{},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			GetKbdLayoutNamesFunc: func() []string {
				return []string{
					"am",
					"be",
					"be_acc",
					"bg_bds",
					"bg_phonetic",
					"br",
					"br_noacc",
					"centraleuropean",
					"ch",
					"ch_acc",
					"ch_macbook_acc",
					"ch-fr",
					"ch-fr_acc",
					"cz",
					"de",
					"de_acc",
					"de_noacc",
					"default",
					"dk",
					"dk_macbook",
					"ee",
					"es",
					"es_acc",
					"es_dvorak",
					"fi",
					"fr",
					"fr_acc",
					"fr_dvorak",
					"fr_dvorak_acc",
					"fr_macbook",
					"gr",
					"gr_101_acc",
					"gr_elot_acc",
					"hr",
					"hu_101",
					"hu_102",
					"is",
					"is_acc",
					"it",
					"jp",
					"jp_capsctrl",
					"kz_io",
					"kz_kst",
					"latinamerican",
					"latinamerican_acc",
					"lt",
					"nl",
					"no",
					"no_dvorak",
					"nordic_asus-eee",
					"pl_dvorak",
					"pt",
					"pt_acc",
					"ru",
					"ru_shift",
					"ru_win",
					"se",
					"si",
					"tr",
					"tr_f",
					"ua",
					"ua_shift_alt",
					"uk",
					"uk_capsctrl",
					"uk_dvorak",
					"uk_macbook",
					"us_dvorak",
					"us_dvorakl",
					"us_dvorakp",
					"us_dvorakr",
					"us_dvorakx",
					"us_emacs",
					"us_unix",
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Id:      "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					Com1Dev: func() *string { r := "junk"; return &r }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "BadCom2Dev",
			mockCmdFunc: "Test_server_UpdateVMSuccess",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:          "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					Name:        "test2024082408",
					Description: "test vm description",
					Status:      "STOPPED",
					BhyvePid:    0,
					VNCPort:     0,
					DebugPort:   0,
					Config: vm.Config{
						Model: gorm.Model{
							ID:        876,
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
						},
						VMID:             "f22416b8-4d21-4b29-a9dd-336fc6aca494",
						CPU:              2,
						Mem:              2048,
						MaxWait:          60,
						Restart:          false,
						Screen:           false,
						ScreenWidth:      1024,
						ScreenHeight:     768,
						Sound:            false,
						SoundIn:          "/dev/dsp1",
						SoundOut:         "/dev/dsp1",
						VNCPort:          "AUTO",
						Tablet:           true,
						StoreUEFIVars:    true,
						UTCTime:          true,
						HostBridge:       true,
						ACPI:             true,
						UseHLT:           true,
						ExitOnPause:      true,
						DestroyPowerOff:  true,
						IgnoreUnknownMSR: true,
						KbdLayout:        "default",
						Com1:             true,
						Com1Dev:          "AUTO",
						Com2Dev:          "AUTO",
						Com3Dev:          "AUTO",
						Com4Dev:          "AUTO",
						Com1Speed:        115200,
						Com2Speed:        115200,
						Com3Speed:        115200,
						Com4Speed:        115200,
						AutoStartDelay:   60,
						DebugPort:        "AUTO",
					},
					ISOs:  []*iso.ISO{},
					Disks: []*disk.Disk{},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			GetKbdLayoutNamesFunc: func() []string {
				return []string{
					"am",
					"be",
					"be_acc",
					"bg_bds",
					"bg_phonetic",
					"br",
					"br_noacc",
					"centraleuropean",
					"ch",
					"ch_acc",
					"ch_macbook_acc",
					"ch-fr",
					"ch-fr_acc",
					"cz",
					"de",
					"de_acc",
					"de_noacc",
					"default",
					"dk",
					"dk_macbook",
					"ee",
					"es",
					"es_acc",
					"es_dvorak",
					"fi",
					"fr",
					"fr_acc",
					"fr_dvorak",
					"fr_dvorak_acc",
					"fr_macbook",
					"gr",
					"gr_101_acc",
					"gr_elot_acc",
					"hr",
					"hu_101",
					"hu_102",
					"is",
					"is_acc",
					"it",
					"jp",
					"jp_capsctrl",
					"kz_io",
					"kz_kst",
					"latinamerican",
					"latinamerican_acc",
					"lt",
					"nl",
					"no",
					"no_dvorak",
					"nordic_asus-eee",
					"pl_dvorak",
					"pt",
					"pt_acc",
					"ru",
					"ru_shift",
					"ru_win",
					"se",
					"si",
					"tr",
					"tr_f",
					"ua",
					"ua_shift_alt",
					"uk",
					"uk_capsctrl",
					"uk_dvorak",
					"uk_macbook",
					"us_dvorak",
					"us_dvorakl",
					"us_dvorakp",
					"us_dvorakr",
					"us_dvorakx",
					"us_emacs",
					"us_unix",
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Id:      "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					Com2Dev: func() *string { r := "junk"; return &r }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "BadCom3Dev",
			mockCmdFunc: "Test_server_UpdateVMSuccess",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:          "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					Name:        "test2024082408",
					Description: "test vm description",
					Status:      "STOPPED",
					BhyvePid:    0,
					VNCPort:     0,
					DebugPort:   0,
					Config: vm.Config{
						Model: gorm.Model{
							ID:        876,
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
						},
						VMID:             "f22416b8-4d21-4b29-a9dd-336fc6aca494",
						CPU:              2,
						Mem:              2048,
						MaxWait:          60,
						Restart:          false,
						Screen:           false,
						ScreenWidth:      1024,
						ScreenHeight:     768,
						Sound:            false,
						SoundIn:          "/dev/dsp1",
						SoundOut:         "/dev/dsp1",
						VNCPort:          "AUTO",
						Tablet:           true,
						StoreUEFIVars:    true,
						UTCTime:          true,
						HostBridge:       true,
						ACPI:             true,
						UseHLT:           true,
						ExitOnPause:      true,
						DestroyPowerOff:  true,
						IgnoreUnknownMSR: true,
						KbdLayout:        "default",
						Com1:             true,
						Com1Dev:          "AUTO",
						Com2Dev:          "AUTO",
						Com3Dev:          "AUTO",
						Com4Dev:          "AUTO",
						Com1Speed:        115200,
						Com2Speed:        115200,
						Com3Speed:        115200,
						Com4Speed:        115200,
						AutoStartDelay:   60,
						DebugPort:        "AUTO",
					},
					ISOs:  []*iso.ISO{},
					Disks: []*disk.Disk{},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			GetKbdLayoutNamesFunc: func() []string {
				return []string{
					"am",
					"be",
					"be_acc",
					"bg_bds",
					"bg_phonetic",
					"br",
					"br_noacc",
					"centraleuropean",
					"ch",
					"ch_acc",
					"ch_macbook_acc",
					"ch-fr",
					"ch-fr_acc",
					"cz",
					"de",
					"de_acc",
					"de_noacc",
					"default",
					"dk",
					"dk_macbook",
					"ee",
					"es",
					"es_acc",
					"es_dvorak",
					"fi",
					"fr",
					"fr_acc",
					"fr_dvorak",
					"fr_dvorak_acc",
					"fr_macbook",
					"gr",
					"gr_101_acc",
					"gr_elot_acc",
					"hr",
					"hu_101",
					"hu_102",
					"is",
					"is_acc",
					"it",
					"jp",
					"jp_capsctrl",
					"kz_io",
					"kz_kst",
					"latinamerican",
					"latinamerican_acc",
					"lt",
					"nl",
					"no",
					"no_dvorak",
					"nordic_asus-eee",
					"pl_dvorak",
					"pt",
					"pt_acc",
					"ru",
					"ru_shift",
					"ru_win",
					"se",
					"si",
					"tr",
					"tr_f",
					"ua",
					"ua_shift_alt",
					"uk",
					"uk_capsctrl",
					"uk_dvorak",
					"uk_macbook",
					"us_dvorak",
					"us_dvorakl",
					"us_dvorakp",
					"us_dvorakr",
					"us_dvorakx",
					"us_emacs",
					"us_unix",
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Id:      "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					Com3Dev: func() *string { r := "junk"; return &r }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "BadCom4Dev",
			mockCmdFunc: "Test_server_UpdateVMSuccess",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:          "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					Name:        "test2024082408",
					Description: "test vm description",
					Status:      "STOPPED",
					BhyvePid:    0,
					VNCPort:     0,
					DebugPort:   0,
					Config: vm.Config{
						Model: gorm.Model{
							ID:        876,
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
						},
						VMID:             "f22416b8-4d21-4b29-a9dd-336fc6aca494",
						CPU:              2,
						Mem:              2048,
						MaxWait:          60,
						Restart:          false,
						Screen:           false,
						ScreenWidth:      1024,
						ScreenHeight:     768,
						Sound:            false,
						SoundIn:          "/dev/dsp1",
						SoundOut:         "/dev/dsp1",
						VNCPort:          "AUTO",
						Tablet:           true,
						StoreUEFIVars:    true,
						UTCTime:          true,
						HostBridge:       true,
						ACPI:             true,
						UseHLT:           true,
						ExitOnPause:      true,
						DestroyPowerOff:  true,
						IgnoreUnknownMSR: true,
						KbdLayout:        "default",
						Com1:             true,
						Com1Dev:          "AUTO",
						Com2Dev:          "AUTO",
						Com3Dev:          "AUTO",
						Com4Dev:          "AUTO",
						Com1Speed:        115200,
						Com2Speed:        115200,
						Com3Speed:        115200,
						Com4Speed:        115200,
						AutoStartDelay:   60,
						DebugPort:        "AUTO",
					},
					ISOs:  []*iso.ISO{},
					Disks: []*disk.Disk{},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			GetKbdLayoutNamesFunc: func() []string {
				return []string{
					"am",
					"be",
					"be_acc",
					"bg_bds",
					"bg_phonetic",
					"br",
					"br_noacc",
					"centraleuropean",
					"ch",
					"ch_acc",
					"ch_macbook_acc",
					"ch-fr",
					"ch-fr_acc",
					"cz",
					"de",
					"de_acc",
					"de_noacc",
					"default",
					"dk",
					"dk_macbook",
					"ee",
					"es",
					"es_acc",
					"es_dvorak",
					"fi",
					"fr",
					"fr_acc",
					"fr_dvorak",
					"fr_dvorak_acc",
					"fr_macbook",
					"gr",
					"gr_101_acc",
					"gr_elot_acc",
					"hr",
					"hu_101",
					"hu_102",
					"is",
					"is_acc",
					"it",
					"jp",
					"jp_capsctrl",
					"kz_io",
					"kz_kst",
					"latinamerican",
					"latinamerican_acc",
					"lt",
					"nl",
					"no",
					"no_dvorak",
					"nordic_asus-eee",
					"pl_dvorak",
					"pt",
					"pt_acc",
					"ru",
					"ru_shift",
					"ru_win",
					"se",
					"si",
					"tr",
					"tr_f",
					"ua",
					"ua_shift_alt",
					"uk",
					"uk_capsctrl",
					"uk_dvorak",
					"uk_macbook",
					"us_dvorak",
					"us_dvorakl",
					"us_dvorakp",
					"us_dvorakr",
					"us_dvorakx",
					"us_emacs",
					"us_unix",
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Id:      "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					Com4Dev: func() *string { r := "junk"; return &r }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "BadName",
			mockCmdFunc: "Test_server_UpdateVMSuccess",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:          "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					Name:        "test2024082408",
					Description: "test vm description",
					Status:      "STOPPED",
					BhyvePid:    0,
					VNCPort:     0,
					DebugPort:   0,
					Config: vm.Config{
						Model: gorm.Model{
							ID:        876,
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
						},
						VMID:             "f22416b8-4d21-4b29-a9dd-336fc6aca494",
						CPU:              2,
						Mem:              2048,
						MaxWait:          60,
						Restart:          false,
						Screen:           false,
						ScreenWidth:      1024,
						ScreenHeight:     768,
						Sound:            false,
						SoundIn:          "/dev/dsp1",
						SoundOut:         "/dev/dsp1",
						VNCPort:          "AUTO",
						Tablet:           true,
						StoreUEFIVars:    true,
						UTCTime:          true,
						HostBridge:       true,
						ACPI:             true,
						UseHLT:           true,
						ExitOnPause:      true,
						DestroyPowerOff:  true,
						IgnoreUnknownMSR: true,
						KbdLayout:        "default",
						Com1:             true,
						Com1Dev:          "AUTO",
						Com2Dev:          "AUTO",
						Com3Dev:          "AUTO",
						Com4Dev:          "AUTO",
						Com1Speed:        115200,
						Com2Speed:        115200,
						Com3Speed:        115200,
						Com4Speed:        115200,
						AutoStartDelay:   60,
						DebugPort:        "AUTO",
					},
					ISOs:  []*iso.ISO{},
					Disks: []*disk.Disk{},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			GetKbdLayoutNamesFunc: func() []string {
				return []string{
					"am",
					"be",
					"be_acc",
					"bg_bds",
					"bg_phonetic",
					"br",
					"br_noacc",
					"centraleuropean",
					"ch",
					"ch_acc",
					"ch_macbook_acc",
					"ch-fr",
					"ch-fr_acc",
					"cz",
					"de",
					"de_acc",
					"de_noacc",
					"default",
					"dk",
					"dk_macbook",
					"ee",
					"es",
					"es_acc",
					"es_dvorak",
					"fi",
					"fr",
					"fr_acc",
					"fr_dvorak",
					"fr_dvorak_acc",
					"fr_macbook",
					"gr",
					"gr_101_acc",
					"gr_elot_acc",
					"hr",
					"hu_101",
					"hu_102",
					"is",
					"is_acc",
					"it",
					"jp",
					"jp_capsctrl",
					"kz_io",
					"kz_kst",
					"latinamerican",
					"latinamerican_acc",
					"lt",
					"nl",
					"no",
					"no_dvorak",
					"nordic_asus-eee",
					"pl_dvorak",
					"pt",
					"pt_acc",
					"ru",
					"ru_shift",
					"ru_win",
					"se",
					"si",
					"tr",
					"tr_f",
					"ua",
					"ua_shift_alt",
					"uk",
					"uk_capsctrl",
					"uk_dvorak",
					"uk_macbook",
					"us_dvorak",
					"us_dvorakl",
					"us_dvorakp",
					"us_dvorakr",
					"us_dvorakx",
					"us_emacs",
					"us_unix",
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Id:   "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					Name: func() *string { r := "junk!junk"; return &r }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "DupeName",
			mockCmdFunc: "Test_server_UpdateVMSuccess",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:          "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					Name:        "test2024082408",
					Description: "test vm description",
					Status:      "STOPPED",
					BhyvePid:    0,
					VNCPort:     0,
					DebugPort:   0,
					Config: vm.Config{
						Model: gorm.Model{
							ID:        876,
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
						},
						VMID:             "f22416b8-4d21-4b29-a9dd-336fc6aca494",
						CPU:              2,
						Mem:              2048,
						MaxWait:          60,
						Restart:          false,
						Screen:           false,
						ScreenWidth:      1024,
						ScreenHeight:     768,
						Sound:            false,
						SoundIn:          "/dev/dsp1",
						SoundOut:         "/dev/dsp1",
						VNCPort:          "AUTO",
						Tablet:           true,
						StoreUEFIVars:    true,
						UTCTime:          true,
						HostBridge:       true,
						ACPI:             true,
						UseHLT:           true,
						ExitOnPause:      true,
						DestroyPowerOff:  true,
						IgnoreUnknownMSR: true,
						KbdLayout:        "default",
						Com1:             true,
						Com1Dev:          "AUTO",
						Com2Dev:          "AUTO",
						Com3Dev:          "AUTO",
						Com4Dev:          "AUTO",
						Com1Speed:        115200,
						Com2Speed:        115200,
						Com3Speed:        115200,
						Com4Speed:        115200,
						AutoStartDelay:   60,
						DebugPort:        "AUTO",
					},
					ISOs:  []*iso.ISO{},
					Disks: []*disk.Disk{},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			GetKbdLayoutNamesFunc: func() []string {
				return []string{
					"am",
					"be",
					"be_acc",
					"bg_bds",
					"bg_phonetic",
					"br",
					"br_noacc",
					"centraleuropean",
					"ch",
					"ch_acc",
					"ch_macbook_acc",
					"ch-fr",
					"ch-fr_acc",
					"cz",
					"de",
					"de_acc",
					"de_noacc",
					"default",
					"dk",
					"dk_macbook",
					"ee",
					"es",
					"es_acc",
					"es_dvorak",
					"fi",
					"fr",
					"fr_acc",
					"fr_dvorak",
					"fr_dvorak_acc",
					"fr_macbook",
					"gr",
					"gr_101_acc",
					"gr_elot_acc",
					"hr",
					"hu_101",
					"hu_102",
					"is",
					"is_acc",
					"it",
					"jp",
					"jp_capsctrl",
					"kz_io",
					"kz_kst",
					"latinamerican",
					"latinamerican_acc",
					"lt",
					"nl",
					"no",
					"no_dvorak",
					"nordic_asus-eee",
					"pl_dvorak",
					"pt",
					"pt_acc",
					"ru",
					"ru_shift",
					"ru_win",
					"se",
					"si",
					"tr",
					"tr_f",
					"ua",
					"ua_shift_alt",
					"uk",
					"uk_capsctrl",
					"uk_dvorak",
					"uk_macbook",
					"us_dvorak",
					"us_dvorakl",
					"us_dvorakp",
					"us_dvorakr",
					"us_dvorakx",
					"us_emacs",
					"us_unix",
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Id:   "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					Name: func() *string { r := "test2024082408"; return &r }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "NewName",
			mockCmdFunc: "Test_server_UpdateVMSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:          "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					Name:        "test2024082408",
					Description: "test vm description",
					Status:      "STOPPED",
					BhyvePid:    0,
					VNCPort:     0,
					DebugPort:   0,
					Config: vm.Config{
						Model: gorm.Model{
							ID:        876,
							CreatedAt: createUpdateTime,
							UpdatedAt: createUpdateTime,
						},
						VMID:             "f22416b8-4d21-4b29-a9dd-336fc6aca494",
						CPU:              2,
						Mem:              2048,
						MaxWait:          60,
						Restart:          false,
						Screen:           false,
						ScreenWidth:      1024,
						ScreenHeight:     768,
						Sound:            false,
						SoundIn:          "/dev/dsp1",
						SoundOut:         "/dev/dsp1",
						VNCPort:          "AUTO",
						Tablet:           true,
						StoreUEFIVars:    true,
						UTCTime:          true,
						HostBridge:       true,
						ACPI:             true,
						UseHLT:           true,
						ExitOnPause:      true,
						DestroyPowerOff:  true,
						IgnoreUnknownMSR: true,
						KbdLayout:        "default",
						Com1:             true,
						Com1Dev:          "AUTO",
						Com2Dev:          "AUTO",
						Com3Dev:          "AUTO",
						Com4Dev:          "AUTO",
						Com1Speed:        115200,
						Com2Speed:        115200,
						Com3Speed:        115200,
						Com4Speed:        115200,
						AutoStartDelay:   60,
						DebugPort:        "AUTO",
					},
					ISOs:  []*iso.ISO{},
					Disks: []*disk.Disk{},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(
						true,
						false,
						false,
						true,
						false,
						60,
						"AUTO",
						false,
						115200,
						"AUTO",
						false,
						115200,
						"AUTO",
						false,
						115200,
						false,
						"AUTO",
						false,
						115200,
						2,
						false,
						"AUTO",
						false,
						true,
						true,
						"",
						true,
						true,
						"default",
						60, 2048,
						0,
						0,
						nil,
						0,
						false,
						0,
						0,
						false,
						768,
						1024,
						false,
						"/dev/dsp1",
						"/dev/dsp1",
						true,
						true,
						true,
						true,
						"AUTO",
						false,
						0,
						0,
						false,
						sqlmock.AnyArg(),
						876).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "test vm description", "test2024082409", 0, sqlmock.AnyArg(), "f22416b8-4d21-4b29-a9dd-336fc6aca494"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("f22416b8-4d21-4b29-a9dd-336fc6aca494").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("f22416b8-4d21-4b29-a9dd-336fc6aca494").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			GetKbdLayoutNamesFunc: func() []string {
				return []string{
					"am",
					"be",
					"be_acc",
					"bg_bds",
					"bg_phonetic",
					"br",
					"br_noacc",
					"centraleuropean",
					"ch",
					"ch_acc",
					"ch_macbook_acc",
					"ch-fr",
					"ch-fr_acc",
					"cz",
					"de",
					"de_acc",
					"de_noacc",
					"default",
					"dk",
					"dk_macbook",
					"ee",
					"es",
					"es_acc",
					"es_dvorak",
					"fi",
					"fr",
					"fr_acc",
					"fr_dvorak",
					"fr_dvorak_acc",
					"fr_macbook",
					"gr",
					"gr_101_acc",
					"gr_elot_acc",
					"hr",
					"hu_101",
					"hu_102",
					"is",
					"is_acc",
					"it",
					"jp",
					"jp_capsctrl",
					"kz_io",
					"kz_kst",
					"latinamerican",
					"latinamerican_acc",
					"lt",
					"nl",
					"no",
					"no_dvorak",
					"nordic_asus-eee",
					"pl_dvorak",
					"pt",
					"pt_acc",
					"ru",
					"ru_shift",
					"ru_win",
					"se",
					"si",
					"tr",
					"tr_f",
					"ua",
					"ua_shift_alt",
					"uk",
					"uk_capsctrl",
					"uk_dvorak",
					"uk_macbook",
					"us_dvorak",
					"us_dvorakl",
					"us_dvorakp",
					"us_dvorakr",
					"us_dvorakx",
					"us_emacs",
					"us_unix",
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Id:   "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					Name: func() *string { n := "test2024082409"; return &n }(), //nolint:nlreturn
				},
			},
			want: func() *cirrina.ReqBool {
				r := cirrina.ReqBool{
					Success: true,
				}

				return &r
			}(),
			wantErr: false,
		},
		{
			name:        "EmptyExistingName",
			mockCmdFunc: "Test_server_UpdateVMSuccess",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:   "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					Name: "",
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			GetKbdLayoutNamesFunc: func() []string {
				return []string{
					"am",
					"be",
					"be_acc",
					"bg_bds",
					"bg_phonetic",
					"br",
					"br_noacc",
					"centraleuropean",
					"ch",
					"ch_acc",
					"ch_macbook_acc",
					"ch-fr",
					"ch-fr_acc",
					"cz",
					"de",
					"de_acc",
					"de_noacc",
					"default",
					"dk",
					"dk_macbook",
					"ee",
					"es",
					"es_acc",
					"es_dvorak",
					"fi",
					"fr",
					"fr_acc",
					"fr_dvorak",
					"fr_dvorak_acc",
					"fr_macbook",
					"gr",
					"gr_101_acc",
					"gr_elot_acc",
					"hr",
					"hu_101",
					"hu_102",
					"is",
					"is_acc",
					"it",
					"jp",
					"jp_capsctrl",
					"kz_io",
					"kz_kst",
					"latinamerican",
					"latinamerican_acc",
					"lt",
					"nl",
					"no",
					"no_dvorak",
					"nordic_asus-eee",
					"pl_dvorak",
					"pt",
					"pt_acc",
					"ru",
					"ru_shift",
					"ru_win",
					"se",
					"si",
					"tr",
					"tr_f",
					"ua",
					"ua_shift_alt",
					"uk",
					"uk_capsctrl",
					"uk_dvorak",
					"uk_macbook",
					"us_dvorak",
					"us_dvorakl",
					"us_dvorakp",
					"us_dvorakr",
					"us_dvorakx",
					"us_emacs",
					"us_unix",
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Id:   "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					Name: func() *string { n := "test2024082409"; return &n }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "VMNotFound",
			mockCmdFunc: "Test_server_UpdateVMSuccess",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:   "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					Name: "",
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			GetKbdLayoutNamesFunc: func() []string {
				return []string{
					"am",
					"be",
					"be_acc",
					"bg_bds",
					"bg_phonetic",
					"br",
					"br_noacc",
					"centraleuropean",
					"ch",
					"ch_acc",
					"ch_macbook_acc",
					"ch-fr",
					"ch-fr_acc",
					"cz",
					"de",
					"de_acc",
					"de_noacc",
					"default",
					"dk",
					"dk_macbook",
					"ee",
					"es",
					"es_acc",
					"es_dvorak",
					"fi",
					"fr",
					"fr_acc",
					"fr_dvorak",
					"fr_dvorak_acc",
					"fr_macbook",
					"gr",
					"gr_101_acc",
					"gr_elot_acc",
					"hr",
					"hu_101",
					"hu_102",
					"is",
					"is_acc",
					"it",
					"jp",
					"jp_capsctrl",
					"kz_io",
					"kz_kst",
					"latinamerican",
					"latinamerican_acc",
					"lt",
					"nl",
					"no",
					"no_dvorak",
					"nordic_asus-eee",
					"pl_dvorak",
					"pt",
					"pt_acc",
					"ru",
					"ru_shift",
					"ru_win",
					"se",
					"si",
					"tr",
					"tr_f",
					"ua",
					"ua_shift_alt",
					"uk",
					"uk_capsctrl",
					"uk_dvorak",
					"uk_macbook",
					"us_dvorak",
					"us_dvorakl",
					"us_dvorakp",
					"us_dvorakr",
					"us_dvorakx",
					"us_emacs",
					"us_unix",
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Id:   "f22416b8-4d21-4b29-a9dd-336fc6aca495",
					Name: func() *string { n := "test2024082409"; return &n }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "BadUuid",
			mockCmdFunc: "Test_server_UpdateVMSuccess",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:   "f22416b8-4d21-4b29-a9dd-336fc6aca494",
					Name: "",
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			GetKbdLayoutNamesFunc: func() []string {
				return []string{
					"am",
					"be",
					"be_acc",
					"bg_bds",
					"bg_phonetic",
					"br",
					"br_noacc",
					"centraleuropean",
					"ch",
					"ch_acc",
					"ch_macbook_acc",
					"ch-fr",
					"ch-fr_acc",
					"cz",
					"de",
					"de_acc",
					"de_noacc",
					"default",
					"dk",
					"dk_macbook",
					"ee",
					"es",
					"es_acc",
					"es_dvorak",
					"fi",
					"fr",
					"fr_acc",
					"fr_dvorak",
					"fr_dvorak_acc",
					"fr_macbook",
					"gr",
					"gr_101_acc",
					"gr_elot_acc",
					"hr",
					"hu_101",
					"hu_102",
					"is",
					"is_acc",
					"it",
					"jp",
					"jp_capsctrl",
					"kz_io",
					"kz_kst",
					"latinamerican",
					"latinamerican_acc",
					"lt",
					"nl",
					"no",
					"no_dvorak",
					"nordic_asus-eee",
					"pl_dvorak",
					"pt",
					"pt_acc",
					"ru",
					"ru_shift",
					"ru_win",
					"se",
					"si",
					"tr",
					"tr_f",
					"ua",
					"ua_shift_alt",
					"uk",
					"uk_capsctrl",
					"uk_dvorak",
					"uk_macbook",
					"us_dvorak",
					"us_dvorakl",
					"us_dvorakp",
					"us_dvorakr",
					"us_dvorakx",
					"us_emacs",
					"us_unix",
				}
			},
			args: args{
				vmConfig: &cirrina.VMConfig{
					Id:   "f22416b8-4d21-4b29-a9dd-336",
					Name: func() *string { n := "test2024082409"; return &n }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			GetKbdLayoutNamesFunc = testCase.GetKbdLayoutNamesFunc

			t.Cleanup(func() { GetKbdLayoutNamesFunc = GetKbdLayoutNames })

			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)

			util.SetupTestCmd(fakeCommand)

			testDB, mockDB := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mockDB)

			lis := bufconn.Listen(1024 * 1024)
			s := grpc.NewServer()
			reflection.Register(s)
			cirrina.RegisterVMInfoServer(s, &server{})

			go func() {
				if err := s.Serve(lis); err != nil {
					log.Fatalf("Server exited with error: %v", err)
				}
			}()

			resolver.SetDefaultScheme("passthrough")

			conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}

			defer func(conn *grpc.ClientConn) {
				_ = conn.Close()
			}(conn)

			client := cirrina.NewVMInfoClient(conn)

			var got *cirrina.ReqBool

			got, err = client.UpdateVM(context.Background(), testCase.args.vmConfig)
			if (err != nil) != testCase.wantErr {
				t.Errorf("UpdateVM() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			mockDB.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mockDB.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

//nolint:paralleltest,maintidx
func Test_server_SetVMISOs(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		setISOReq *cirrina.SetISOReq
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *cirrina.ReqBool
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				testVM1 := vm.VM{
					ID:     "e908d40c-a4ca-4d72-914e-40489546cf1d",
					Name:   "test2024082410",
					Status: vm.STOPPED,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 8202,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).WithArgs("14cb7716-56a8-4c70-bcd3-2dfbd108e42d").
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"name",
								"description",
								"path",
								"size",
								"checksum",
							}).
							AddRow(
								"14cb7716-56a8-4c70-bcd3-2dfbd108e42d",
								createUpdateTime,
								createUpdateTime,
								nil,
								"someTest3.iso",
								"some iso description",
								"/bhyve/isos/someTest3.iso",
								418819271238,
								"259f034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3aaaa30ba3", //nolint:lll
							),
					)

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(false, false, false, false, false, 0, "", false, 0, "", false, 0, "", false, 0, false, "", false, 0, 0, false, "", false, false, false, "", false, false, "", 0, 0, 0, 0, nil, 0, false, 0, 0, false, 0, 0, false, "", "", false, false, false, false, "", false, 0, 0, false, sqlmock.AnyArg(), 8202). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "", "test2024082410", 0, sqlmock.AnyArg(), "e908d40c-a4ca-4d72-914e-40489546cf1d"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("e908d40c-a4ca-4d72-914e-40489546cf1d").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 27))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_isos` (`vm_id`,`iso_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("e908d40c-a4ca-4d72-914e-40489546cf1d", "14cb7716-56a8-4c70-bcd3-2dfbd108e42d", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("e908d40c-a4ca-4d72-914e-40489546cf1d").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			args: args{
				setISOReq: &cirrina.SetISOReq{
					Id:    "e908d40c-a4ca-4d72-914e-40489546cf1d",
					Isoid: []string{"14cb7716-56a8-4c70-bcd3-2dfbd108e42d"},
				},
			},
			want:    func() *cirrina.ReqBool { r := cirrina.ReqBool{Success: true}; return &r }(), //nolint:nlreturn
			wantErr: false,
		},
		{
			name: "ErrorSaving",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				testVM1 := vm.VM{
					ID:     "e908d40c-a4ca-4d72-914e-40489546cf1d",
					Name:   "test2024082410",
					Status: vm.STOPPED,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 8202,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).WithArgs("14cb7716-56a8-4c70-bcd3-2dfbd108e42d").
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"name",
								"description",
								"path",
								"size",
								"checksum",
							}).
							AddRow(
								"14cb7716-56a8-4c70-bcd3-2dfbd108e42d",
								createUpdateTime,
								createUpdateTime,
								nil,
								"someTest3.iso",
								"some iso description",
								"/bhyve/isos/someTest3.iso",
								418819271238,
								"259f034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3aaaa30ba3", //nolint:lll
							),
					)

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(false, false, false, false, false, 0, "", false, 0, "", false, 0, "", false, 0, false, "", false, 0, 0, false, "", false, false, false, "", false, false, "", 0, 0, 0, 0, nil, 0, false, 0, 0, false, 0, 0, false, "", "", false, false, false, false, "", false, 0, 0, false, sqlmock.AnyArg(), 8202). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "", "test2024082410", 0, sqlmock.AnyArg(), "e908d40c-a4ca-4d72-914e-40489546cf1d"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("e908d40c-a4ca-4d72-914e-40489546cf1d").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 27))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_isos` (`vm_id`,`iso_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("e908d40c-a4ca-4d72-914e-40489546cf1d", "14cb7716-56a8-4c70-bcd3-2dfbd108e42d", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("e908d40c-a4ca-4d72-914e-40489546cf1d").
					WillReturnError(gorm.ErrInvalidData)
			},
			args: args{
				setISOReq: &cirrina.SetISOReq{
					Id:    "e908d40c-a4ca-4d72-914e-40489546cf1d",
					Isoid: []string{"14cb7716-56a8-4c70-bcd3-2dfbd108e42d"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorGettingISO",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				testVM1 := vm.VM{
					ID:     "e908d40c-a4ca-4d72-914e-40489546cf1d",
					Name:   "test2024082410",
					Status: vm.STOPPED,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 8202,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).WithArgs("14cb7716-56a8-4c70-bcd3-2dfbd108e42c").
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"name",
								"description",
								"path",
								"size",
								"checksum",
							}),
					)
			},
			args: args{
				setISOReq: &cirrina.SetISOReq{
					Id:    "e908d40c-a4ca-4d72-914e-40489546cf1d",
					Isoid: []string{"14cb7716-56a8-4c70-bcd3-2dfbd108e42c"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "EmptyExistingVMName",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				testVM1 := vm.VM{
					ID:     "e908d40c-a4ca-4d72-914e-40489546cf1d",
					Name:   "",
					Status: vm.STOPPED,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 8202,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				setISOReq: &cirrina.SetISOReq{
					Id:    "e908d40c-a4ca-4d72-914e-40489546cf1d",
					Isoid: []string{"14cb7716-56a8-4c70-bcd3-2dfbd108e42c"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "VMDoesNotExist",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				setISOReq: &cirrina.SetISOReq{
					Id:    "e908d40c-a4ca-4d72-914e-40489546cf1d",
					Isoid: []string{"14cb7716-56a8-4c70-bcd3-2dfbd108e42c"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "BadUuid",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				setISOReq: &cirrina.SetISOReq{
					Id:    "e908d40c-a4ca-4d72-914e-40489546cf",
					Isoid: []string{"14cb7716-56a8-4c70-bcd3-2dfbd108e42c"},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mockDB := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mockDB)

			lis := bufconn.Listen(1024 * 1024)
			s := grpc.NewServer()
			reflection.Register(s)
			cirrina.RegisterVMInfoServer(s, &server{})

			go func() {
				if err := s.Serve(lis); err != nil {
					log.Fatalf("Server exited with error: %v", err)
				}
			}()

			resolver.SetDefaultScheme("passthrough")

			conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}

			defer func(conn *grpc.ClientConn) {
				_ = conn.Close()
			}(conn)

			client := cirrina.NewVMInfoClient(conn)

			var got *cirrina.ReqBool

			got, err = client.SetVMISOs(context.Background(), testCase.args.setISOReq)
			if (err != nil) != testCase.wantErr {
				t.Errorf("SetVMISOs() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			mockDB.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mockDB.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

//nolint:paralleltest,maintidx
func Test_server_SetVMNics(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		setNicReq *cirrina.SetNicReq
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *cirrina.ReqBool
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				testVM1 := vm.VM{
					ID:     "25ebf487-ee33-41c5-88f5-117dadfa9b4f",
					Name:   "test2024082501",
					Status: vm.STOPPED,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 692,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).WithArgs(692).
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

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1",
					),
				).WithArgs("f8501be4-271c-4f4e-8cda-803e4e8a97fe").
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
					}).
						AddRow(
							"f8501be4-271c-4f4e-8cda-803e4e8a97fe",
							createUpdateTime,
							createUpdateTime,
							nil,
							"test2024082501_int0",
							"a test description",
							"00:22:44:8a:7b:6c",
							"VIRTIONET",
							"TAP",
							"4392676b-4705-4fd1-bf95-f9e235760fd4",
							"",
							false,
							0,
							0,
							nil,
							nil,
							0,
						),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).WithArgs(692).
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

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1",
					),
				).WithArgs("f8501be4-271c-4f4e-8cda-803e4e8a97fe").
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
					}).
						AddRow(
							"f8501be4-271c-4f4e-8cda-803e4e8a97fe",
							createUpdateTime,
							createUpdateTime,
							nil,
							"test2024082501_int0",
							"a test description",
							"00:22:44:8a:7b:6c",
							"VIRTIONET",
							"TAP",
							"4392676b-4705-4fd1-bf95-f9e235760fd4",
							"",
							false,
							0,
							0,
							nil,
							nil,
							0,
						),
					)

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(692, "a test description", "", "", "00:22:44:8a:7b:6c", "test2024082501_int0", "", "TAP",
						"VIRTIONET", 0, false, 0, "4392676b-4705-4fd1-bf95-f9e235760fd4", sqlmock.AnyArg(),
						"f8501be4-271c-4f4e-8cda-803e4e8a97fe").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args: args{
				setNicReq: &cirrina.SetNicReq{
					Vmid:    "25ebf487-ee33-41c5-88f5-117dadfa9b4f",
					Vmnicid: []string{"f8501be4-271c-4f4e-8cda-803e4e8a97fe"},
				},
			},
			want:    func() *cirrina.ReqBool { r := cirrina.ReqBool{Success: true}; return &r }(), //nolint:nlreturn
			wantErr: false,
		},
		{
			name: "ErrorSaving",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				testVM1 := vm.VM{
					ID:     "25ebf487-ee33-41c5-88f5-117dadfa9b4f",
					Name:   "test2024082501",
					Status: vm.STOPPED,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 692,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).WithArgs(692).
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

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1",
					),
				).WithArgs("f8501be4-271c-4f4e-8cda-803e4e8a97fe").
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
					}).
						AddRow(
							"f8501be4-271c-4f4e-8cda-803e4e8a97fe",
							createUpdateTime,
							createUpdateTime,
							nil,
							"test2024082501_int0",
							"a test description",
							"00:22:44:8a:7b:6c",
							"VIRTIONET",
							"TAP",
							"4392676b-4705-4fd1-bf95-f9e235760fd4",
							"",
							false,
							0,
							0,
							nil,
							nil,
							0,
						),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).WithArgs(692).
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

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1",
					),
				).WithArgs("f8501be4-271c-4f4e-8cda-803e4e8a97fe").
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
					}).
						AddRow(
							"f8501be4-271c-4f4e-8cda-803e4e8a97fe",
							createUpdateTime,
							createUpdateTime,
							nil,
							"test2024082501_int0",
							"a test description",
							"00:22:44:8a:7b:6c",
							"VIRTIONET",
							"TAP",
							"4392676b-4705-4fd1-bf95-f9e235760fd4",
							"",
							false,
							0,
							0,
							nil,
							nil,
							0,
						),
					)

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(692, "a test description", "", "", "00:22:44:8a:7b:6c", "test2024082501_int0", "", "TAP",
						"VIRTIONET", 0, false, 0, "4392676b-4705-4fd1-bf95-f9e235760fd4", sqlmock.AnyArg(),
						"f8501be4-271c-4f4e-8cda-803e4e8a97fe").
					WillReturnError(gorm.ErrInvalidData)
				mock.ExpectRollback()
			},
			args: args{
				setNicReq: &cirrina.SetNicReq{
					Vmid:    "25ebf487-ee33-41c5-88f5-117dadfa9b4f",
					Vmnicid: []string{"f8501be4-271c-4f4e-8cda-803e4e8a97fe"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorExistingVMName",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				testVM1 := vm.VM{
					ID:     "25ebf487-ee33-41c5-88f5-117dadfa9b4f",
					Name:   "",
					Status: vm.STOPPED,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 692,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				setNicReq: &cirrina.SetNicReq{
					Vmid:    "25ebf487-ee33-41c5-88f5-117dadfa9b4f",
					Vmnicid: []string{"f8501be4-271c-4f4e-8cda-803e4e8a97fe"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorVMNotFound",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				setNicReq: &cirrina.SetNicReq{
					Vmid:    "06e010bc-00d0-43e8-8c1c-446ef3a20c02",
					Vmnicid: []string{"f8501be4-271c-4f4e-8cda-803e4e8a97fe"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorInvalidVMUuid",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				setNicReq: &cirrina.SetNicReq{
					Vmid:    "06e010bc-00d0-43e8-8c1c-446",
					Vmnicid: []string{"f8501be4-271c-4f4e-8cda-803e4e8a97fe"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorInvalidNicUuid",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{ // prevents parallel testing
					VMDB: testDB,
				}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				testVM1 := vm.VM{
					ID:     "25ebf487-ee33-41c5-88f5-117dadfa9b4f",
					Name:   "test2024082501",
					Status: vm.STOPPED,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 692,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).WithArgs(692).
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
			args: args{
				setNicReq: &cirrina.SetNicReq{
					Vmid:    "25ebf487-ee33-41c5-88f5-117dadfa9b4f",
					Vmnicid: []string{"f8501be4-271c-4f4e-8cda-803"},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mockDB := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mockDB)

			lis := bufconn.Listen(1024 * 1024)
			s := grpc.NewServer()
			reflection.Register(s)
			cirrina.RegisterVMInfoServer(s, &server{})

			go func() {
				if err := s.Serve(lis); err != nil {
					log.Fatalf("Server exited with error: %v", err)
				}
			}()

			resolver.SetDefaultScheme("passthrough")

			conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}

			defer func(conn *grpc.ClientConn) {
				_ = conn.Close()
			}(conn)

			client := cirrina.NewVMInfoClient(conn)

			var got *cirrina.ReqBool

			got, err = client.SetVMNics(context.Background(), testCase.args.setNicReq)
			if (err != nil) != testCase.wantErr {
				t.Errorf("SetVMNics() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			mockDB.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mockDB.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

//nolint:paralleltest,maintidx
func Test_server_SetVMDisks(t *testing.T) {
	type args struct {
		setDiskReq *cirrina.SetDiskReq
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *cirrina.ReqBool
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				disk.Instance = &disk.Singleton{DiskDB: testDB}

				testVM1 := vm.VM{
					ID:     "ccf34989-0da6-497d-a5b7-1aee352dcac1",
					Name:   "test2024082502",
					Status: vm.STOPPED,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 693,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				testDisk1 := disk.Disk{
					ID:          "f462d4b3-9d41-4630-98f2-4bbb8cce6eed",
					Name:        "test2024082502_hd0",
					Description: "test disk",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}

				disk.List.DiskList[testDisk1.ID] = &testDisk1

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(false, false, false, false, false, 0, "", false, 0, "", false, 0, "", false, 0, false, "", false, 0, 0, false, "", false, false, false, "", false, false, "", 0, 0, 0, 0, nil, 0, false, 0, 0, false, 0, 0, false, "", "", false, false, false, false, "", false, 0, 0, false, sqlmock.AnyArg(), 693). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "", "test2024082502", 0, sqlmock.AnyArg(), "ccf34989-0da6-497d-a5b7-1aee352dcac1"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("ccf34989-0da6-497d-a5b7-1aee352dcac1").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("ccf34989-0da6-497d-a5b7-1aee352dcac1").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_disks` (`vm_id`,`disk_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("ccf34989-0da6-497d-a5b7-1aee352dcac1", "f462d4b3-9d41-4630-98f2-4bbb8cce6eed", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args: args{
				setDiskReq: &cirrina.SetDiskReq{
					Id:     "ccf34989-0da6-497d-a5b7-1aee352dcac1",
					Diskid: []string{"f462d4b3-9d41-4630-98f2-4bbb8cce6eed"},
				},
			},
			want:    func() *cirrina.ReqBool { r := cirrina.ReqBool{Success: true}; return &r }(), //nolint:nlreturn
			wantErr: false,
		},
		{
			name: "ErrorSaving",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				disk.Instance = &disk.Singleton{DiskDB: testDB}

				testVM1 := vm.VM{
					ID:     "ccf34989-0da6-497d-a5b7-1aee352dcac1",
					Name:   "test2024082502",
					Status: vm.STOPPED,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 693,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				testDisk1 := disk.Disk{
					ID:          "f462d4b3-9d41-4630-98f2-4bbb8cce6eed",
					Name:        "test2024082502_hd0",
					Description: "test disk",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}

				disk.List.DiskList[testDisk1.ID] = &testDisk1

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(false, false, false, false, false, 0, "", false, 0, "", false, 0, "", false, 0, false, "", false, 0, 0, false, "", false, false, false, "", false, false, "", 0, 0, 0, 0, nil, 0, false, 0, 0, false, 0, 0, false, "", "", false, false, false, false, "", false, 0, 0, false, sqlmock.AnyArg(), 693). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "", "test2024082502", 0, sqlmock.AnyArg(), "ccf34989-0da6-497d-a5b7-1aee352dcac1"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("ccf34989-0da6-497d-a5b7-1aee352dcac1").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("ccf34989-0da6-497d-a5b7-1aee352dcac1").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_disks` (`vm_id`,`disk_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("ccf34989-0da6-497d-a5b7-1aee352dcac1", "f462d4b3-9d41-4630-98f2-4bbb8cce6eed", 0).
					WillReturnError(gorm.ErrInvalidData)
				mock.ExpectRollback()
			},
			args: args{
				setDiskReq: &cirrina.SetDiskReq{
					Id:     "ccf34989-0da6-497d-a5b7-1aee352dcac1",
					Diskid: []string{"f462d4b3-9d41-4630-98f2-4bbb8cce6eed"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorEmptyExistingVMName",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				disk.Instance = &disk.Singleton{DiskDB: testDB}

				testVM1 := vm.VM{
					ID:     "ccf34989-0da6-497d-a5b7-1aee352dcac1",
					Name:   "",
					Status: vm.STOPPED,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 693,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				testDisk1 := disk.Disk{
					ID:          "f462d4b3-9d41-4630-98f2-4bbb8cce6eed",
					Name:        "test2024082502_hd0",
					Description: "test disk",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}

				disk.List.DiskList[testDisk1.ID] = &testDisk1
			},
			args: args{
				setDiskReq: &cirrina.SetDiskReq{
					Id:     "ccf34989-0da6-497d-a5b7-1aee352dcac1",
					Diskid: []string{"f462d4b3-9d41-4630-98f2-4bbb8cce6eed"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorVMNotFound",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				disk.Instance = &disk.Singleton{DiskDB: testDB}

				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				setDiskReq: &cirrina.SetDiskReq{
					Id:     "ccf34989-0da6-497d-a5b7-1aee352dcac2",
					Diskid: []string{"f462d4b3-9d41-4630-98f2-4bbb8cce6eed"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorVMBadUuid",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				disk.Instance = &disk.Singleton{DiskDB: testDB}

				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				setDiskReq: &cirrina.SetDiskReq{
					Id:     "ccf34989-0da6-497d-a5b7-1ae",
					Diskid: []string{"f462d4b3-9d41-4630-98f2-4bbb8cce6eed"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorDiskBadUuid",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				disk.Instance = &disk.Singleton{DiskDB: testDB}

				testVM1 := vm.VM{
					ID:     "ccf34989-0da6-497d-a5b7-1aee352dcac1",
					Name:   "test2024082502",
					Status: vm.STOPPED,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 693,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				testDisk1 := disk.Disk{
					ID:          "f462d4b3-9d41-4630-98f2-4bbb8cce6eed",
					Name:        "test2024082502_hd0",
					Description: "test disk",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}

				disk.List.DiskList[testDisk1.ID] = &testDisk1
			},
			args: args{
				setDiskReq: &cirrina.SetDiskReq{
					Id:     "ccf34989-0da6-497d-a5b7-1aee352dcac1",
					Diskid: []string{"f462d4b3-9d41-4630-98f2-4bbb8cce"},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mockDB := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mockDB)

			lis := bufconn.Listen(1024 * 1024)
			s := grpc.NewServer()
			reflection.Register(s)
			cirrina.RegisterVMInfoServer(s, &server{})

			go func() {
				if err := s.Serve(lis); err != nil {
					log.Fatalf("Server exited with error: %v", err)
				}
			}()

			resolver.SetDefaultScheme("passthrough")

			conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}

			defer func(conn *grpc.ClientConn) {
				_ = conn.Close()
			}(conn)

			client := cirrina.NewVMInfoClient(conn)

			var got *cirrina.ReqBool

			got, err = client.SetVMDisks(context.Background(), testCase.args.setDiskReq)
			if (err != nil) != testCase.wantErr {
				t.Errorf("SetVMDisks() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			mockDB.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mockDB.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

//nolint:paralleltest,maintidx
func Test_server_StopVM(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		vmID *cirrina.VMID
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *cirrina.RequestID
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:     "ef7f4777-5a93-4159-b3b2-807860929ddc",
					Name:   "test2024082503",
					Status: vm.RUNNING,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 694,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL",
					),
				).
					WithArgs(false).
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"started_at",
								"successful",
								"complete",
								"type",
								"data",
							}),
					)

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `requests` (`created_at`,`updated_at`,`deleted_at`,`started_at`,`successful`,`complete`,`type`,`data`,`id`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`", //nolint:lll
					),
				).
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, false, false, "VMSTOP", "{\"vm_id\":\"ef7f4777-5a93-4159-b3b2-807860929ddc\"}", sqlmock.AnyArg(), //nolint:lll
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("92b56d70-7001-4598-8895-761791234678"))
				mock.ExpectCommit()
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "ef7f4777-5a93-4159-b3b2-807860929ddc",
				},
			},
			want: func() *cirrina.RequestID {
				r := cirrina.RequestID{Value: "92b56d70-7001-4598-8895-761791234678"}

				return &r
			}(),
			wantErr: false,
		},
		{
			name: "ErrorSaving",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:     "ef7f4777-5a93-4159-b3b2-807860929ddc",
					Name:   "test2024082503",
					Status: vm.RUNNING,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 694,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL",
					),
				).
					WithArgs(false).
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"started_at",
								"successful",
								"complete",
								"type",
								"data",
							}),
					)

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `requests` (`created_at`,`updated_at`,`deleted_at`,`started_at`,`successful`,`complete`,`type`,`data`,`id`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`", //nolint:lll
					),
				).
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, false, false, "VMSTOP", "{\"vm_id\":\"ef7f4777-5a93-4159-b3b2-807860929ddc\"}", sqlmock.AnyArg(), //nolint:lll
					).
					WillReturnError(gorm.ErrInvalidData)
				mock.ExpectRollback()
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "ef7f4777-5a93-4159-b3b2-807860929ddc",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorVMNotRunning",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:     "ef7f4777-5a93-4159-b3b2-807860929ddc",
					Name:   "test2024082503",
					Status: vm.STOPPED,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 694,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL",
					),
				).
					WithArgs(false).
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"started_at",
								"successful",
								"complete",
								"type",
								"data",
							}),
					)
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "ef7f4777-5a93-4159-b3b2-807860929ddc",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorPendingRequestExists",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:     "ef7f4777-5a93-4159-b3b2-807860929ddc",
					Name:   "test2024082503",
					Status: vm.RUNNING,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 694,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL",
					),
				).
					WithArgs(false).
					WillReturnRows(
						sqlmock.NewRows([]string{
							"id",
							"created_at",
							"updated_at",
							"deleted_at",
							"started_at",
							"successful",
							"complete",
							"type",
							"data"}).
							AddRow(
								"0662d1f4-1a62-4b42-8d3d-844f37d3c35a",
								createUpdateTime,
								createUpdateTime,
								nil,
								time.Time{},
								0,
								0,
								"VMSTART",
								"{\"vm_id\":\"ef7f4777-5a93-4159-b3b2-807860929ddc\"}"))
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "ef7f4777-5a93-4159-b3b2-807860929ddc",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorExistingVMNameEmpty",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:     "ef7f4777-5a93-4159-b3b2-807860929ddd",
					Name:   "",
					Status: vm.RUNNING,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 694,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "ef7f4777-5a93-4159-b3b2-807860929ddd",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorVMNotFound",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "ef7f4777-5a93-4159-b3b2-807860929dde",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorVMBadUuid",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "ef7f4777-5a93-4159-b3b2-80786",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mockDB := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mockDB)

			lis := bufconn.Listen(1024 * 1024)
			s := grpc.NewServer()
			reflection.Register(s)
			cirrina.RegisterVMInfoServer(s, &server{})

			go func() {
				if err := s.Serve(lis); err != nil {
					log.Fatalf("Server exited with error: %v", err)
				}
			}()

			resolver.SetDefaultScheme("passthrough")

			conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}

			defer func(conn *grpc.ClientConn) {
				_ = conn.Close()
			}(conn)

			client := cirrina.NewVMInfoClient(conn)

			var got *cirrina.RequestID

			got, err = client.StopVM(context.Background(), testCase.args.vmID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("StopVM() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			mockDB.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mockDB.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

//nolint:paralleltest,maintidx
func Test_server_StartVM(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		vmID *cirrina.VMID
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *cirrina.RequestID
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:     "46153591-b8b1-419f-8bdb-d82981abb115",
					Name:   "test2024082504",
					Status: vm.STOPPED,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 695,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL",
					),
				).
					WithArgs(false).
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"started_at",
								"successful",
								"complete",
								"type",
								"data",
							}),
					)

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `requests` (`created_at`,`updated_at`,`deleted_at`,`started_at`,`successful`,`complete`,`type`,`data`,`id`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`", //nolint:lll
					),
				).
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, false, false, "VMSTART", "{\"vm_id\":\"46153591-b8b1-419f-8bdb-d82981abb115\"}", sqlmock.AnyArg(), //nolint:lll
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("e382c142-c38e-44b5-8746-72e282b2f6b8"))
				mock.ExpectCommit()
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "46153591-b8b1-419f-8bdb-d82981abb115",
				},
			},
			want: func() *cirrina.RequestID {
				r := cirrina.RequestID{Value: "e382c142-c38e-44b5-8746-72e282b2f6b8"}

				return &r
			}(),
			wantErr: false,
		},
		{
			name: "ErrorSaving",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:     "46153591-b8b1-419f-8bdb-d82981abb115",
					Name:   "test2024082504",
					Status: vm.STOPPED,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 695,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL",
					),
				).
					WithArgs(false).
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"started_at",
								"successful",
								"complete",
								"type",
								"data",
							}),
					)

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `requests` (`created_at`,`updated_at`,`deleted_at`,`started_at`,`successful`,`complete`,`type`,`data`,`id`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`", //nolint:lll
					),
				).
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, false, false, "VMSTART", "{\"vm_id\":\"46153591-b8b1-419f-8bdb-d82981abb115\"}", sqlmock.AnyArg(), //nolint:lll
					).
					WillReturnError(gorm.ErrInvalidData)
				mock.ExpectRollback()
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "46153591-b8b1-419f-8bdb-d82981abb115",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorVMNotStopped",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:     "46153591-b8b1-419f-8bdb-d82981abb115",
					Name:   "test2024082504",
					Status: vm.RUNNING,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 695,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL",
					),
				).
					WithArgs(false).
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"started_at",
								"successful",
								"complete",
								"type",
								"data",
							}),
					)
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "46153591-b8b1-419f-8bdb-d82981abb115",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorPendingReqExists",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:     "46153591-b8b1-419f-8bdb-d82981abb115",
					Name:   "test2024082504",
					Status: vm.RUNNING,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 694,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL",
					),
				).
					WithArgs(false).
					WillReturnRows(
						sqlmock.NewRows([]string{
							"id",
							"created_at",
							"updated_at",
							"deleted_at",
							"started_at",
							"successful",
							"complete",
							"type",
							"data"}).
							AddRow(
								"f8e19a4a-e6a3-4582-80c8-7387db7c4fe7",
								createUpdateTime,
								createUpdateTime,
								nil,
								time.Time{},
								0,
								0,
								"VMSTART",
								"{\"vm_id\":\"46153591-b8b1-419f-8bdb-d82981abb115\"}"))
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "46153591-b8b1-419f-8bdb-d82981abb115",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorExistingVMNameEmpty",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				testVM1 := vm.VM{
					ID:     "46153591-b8b1-419f-8bdb-d82981abb115",
					Name:   "",
					Status: vm.STOPPED,
					Config: vm.Config{
						Model: gorm.Model{
							ID: 694,
						},
					},
				}

				vm.List.VMList = map[string]*vm.VM{}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "46153591-b8b1-419f-8bdb-d82981abb115",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorVMNotFound",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "46153591-b8b1-419f-8bdb-d82981abb116",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ErrorBadVMUuid",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vm.Instance = &vm.Singleton{VMDB: testDB}
				requests.Instance = &requests.Singleton{ReqDB: testDB}

				vm.List.VMList = map[string]*vm.VM{}
			},
			args: args{
				vmID: &cirrina.VMID{
					Value: "46153591-b8b1-419f-8bdb-d8",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mockDB := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mockDB)

			lis := bufconn.Listen(1024 * 1024)
			s := grpc.NewServer()
			reflection.Register(s)
			cirrina.RegisterVMInfoServer(s, &server{})

			go func() {
				if err := s.Serve(lis); err != nil {
					log.Fatalf("Server exited with error: %v", err)
				}
			}()

			resolver.SetDefaultScheme("passthrough")

			conn, err := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("Failed to dial bufnet: %v", err)
			}

			defer func(conn *grpc.ClientConn) {
				_ = conn.Close()
			}(conn)

			client := cirrina.NewVMInfoClient(conn)

			var got *cirrina.RequestID

			got, err = client.StartVM(context.Background(), testCase.args.vmID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("StartVM() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			mockDB.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mockDB.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

// test helpers from here down

//nolint:paralleltest
func Test_server_AddVMSuccess(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_server_UpdateVMSuccess(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	for _, v := range os.Args {
		if v == "hw.vmm.maxcpu" {
			fmt.Printf("64") //nolint:forbidigo
			os.Exit(0)
		}
	}

	os.Exit(0)
}
