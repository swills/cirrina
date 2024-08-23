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

	"cirrina/cirrina"
	"cirrina/cirrinad/util"
)

//nolint:paralleltest
func Test_server_GetVersion(t *testing.T) {
	tests := []struct {
		name        string
		want        *wrapperspb.StringValue
		mockClosure func()
		wantErr     bool
	}{
		{
			name:        "SuccessNotSet",
			want:        func() *wrapperspb.StringValue { v := "unknown"; return wrapperspb.String(v) }(), //nolint:nlreturn
			mockClosure: func() {},
			wantErr:     false,
		},
		{
			name: "SuccessDev",
			want: func() *wrapperspb.StringValue { v := "dev"; return wrapperspb.String(v) }(), //nolint:nlreturn
			mockClosure: func() {
				mainVersion = "dev"
			},
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

			var got *wrapperspb.StringValue

			got, err = client.GetVersion(context.Background(), nil)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetVersion() error = %v, wantErr %v", err, testCase.wantErr)

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
func Test_server_GetNetInterfaces(t *testing.T) {
	tests := []struct {
		name                string
		hostIntStubFunc     func() ([]net.Interface, error)
		getIntGroupStubFunc func(string) ([]string, error)
		want                []string
		wantErr             bool
	}{
		{
			name:                "Success",
			hostIntStubFunc:     StubHostInterfacesSuccess3,
			getIntGroupStubFunc: StubGetHostIntGroupSuccess3,
			want:                []string{"re0", "re1", "re2"},
			wantErr:             false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			util.NetInterfacesFunc = testCase.hostIntStubFunc

			t.Cleanup(func() { util.NetInterfacesFunc = net.Interfaces })

			util.GetIntGroupsFunc = testCase.getIntGroupStubFunc

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

			var res cirrina.VMInfo_GetNetInterfacesClient

			var NetIf *cirrina.NetIf

			var got []string

			res, err = client.GetNetInterfaces(context.Background(), &cirrina.NetInterfacesReq{})

			if (err != nil) != testCase.wantErr {
				t.Errorf("GetNetInterfaces() error = %v, wantErr %v", err, testCase.wantErr)
			}

			for {
				NetIf, err = res.Recv()
				if errors.Is(err, io.EOF) {
					break
				}

				got = append(got, NetIf.GetInterfaceName())
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

// test helpers from here down

func StubHostInterfacesSuccess3() ([]net.Interface, error) {
	return []net.Interface{
		{
			Index:        1,
			MTU:          1500,
			Name:         "re0",
			HardwareAddr: net.HardwareAddr{0xff, 0xdd, 0xcc, 0x28, 0x73, 0x3e},
			Flags:        0x33,
		},
		{
			Index:        2,
			MTU:          16384,
			Name:         "lo0",
			HardwareAddr: net.HardwareAddr(nil),
			Flags:        0x35,
		},
		{
			Index:        3,
			MTU:          1500,
			Name:         "re1",
			HardwareAddr: net.HardwareAddr{0xff, 0xdd, 0xcc, 0x91, 0x7a, 0x71},
			Flags:        0x33,
		},
		{
			Index:        4,
			MTU:          1500,
			Name:         "re2",
			HardwareAddr: net.HardwareAddr{0xab, 0xcd, 0xef, 0x01, 0x23, 0x34},
			Flags:        0x33,
		},
	}, nil
}

func StubGetHostIntGroupSuccess3(intName string) ([]string, error) {
	switch intName {
	case "re0":
		return []string{}, nil
	case "re1":
		return []string{}, nil
	case "re2":
		return []string{}, nil
	case "lo0":
		return []string{"lo"}, nil
	default:
		return nil, nil
	}
}
