package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log"
	"net"
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
			testDB, mock := cirrinadtest.NewMockDB(testCase.name)
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
