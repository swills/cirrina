package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"testing"

	"github.com/go-test/deep"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"gorm.io/gorm"

	"cirrina/cirrina"
	"cirrina/cirrinad/vm"
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
