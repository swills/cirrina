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

	"cirrina/cirrina"
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
