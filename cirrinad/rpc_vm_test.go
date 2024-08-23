package main

import (
	"context"
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
