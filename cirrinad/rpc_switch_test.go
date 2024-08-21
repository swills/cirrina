package main

import (
	"context"
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
	"gorm.io/gorm"

	"cirrina/cirrina"
	"cirrina/cirrinad/cirrinadtest"
	_switch "cirrina/cirrinad/switch"
)

//nolint:paralleltest
func Test_server_GetSwitchInfo(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		switchID *cirrina.SwitchId
	}

	tests := []struct {
		name        string
		args        args
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        *cirrina.SwitchInfo
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				_switch.Instance = &_switch.Singleton{
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("ce3532e1-2bcc-47d2-b26f-edac51a450da").
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"type",
						"uplink",
					}).AddRow(
						"ce3532e1-2bcc-47d2-b26f-edac51a450da",
						createUpdateTime,
						createUpdateTime,
						nil,
						"bridge0",
						"a test bridge",
						"IF",
						"em2",
					))
			},
			args: args{
				switchID: &cirrina.SwitchId{
					Value: "ce3532e1-2bcc-47d2-b26f-edac51a450da",
				},
			},
			want: &cirrina.SwitchInfo{
				Name:        func() *string { name := "bridge0"; return &name }(),                     //nolint:nlreturn
				Description: func() *string { desc := "a test bridge"; return &desc }(),               //nolint:nlreturn
				Uplink:      func() *string { uplink := "em2"; return &uplink }(),                     //nolint:nlreturn
				SwitchType:  func() *cirrina.SwitchType { st := cirrina.SwitchType_IF; return &st }(), //nolint:nlreturn
			},
		},
		{
			name: "BadID",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				_switch.Instance = &_switch.Singleton{
					SwitchDB: testDB,
				}
			},
			args: args{
				switchID: &cirrina.SwitchId{
					Value: "ce3532e1-2bcc-47d2-b26",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NotFound",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				_switch.Instance = &_switch.Singleton{
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("ce3532e1-2bcc-47d2-b26f-edac51a450da").
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"type",
						"uplink",
					}))
			},
			args: args{
				switchID: &cirrina.SwitchId{
					Value: "ce3532e1-2bcc-47d2-b26f-edac51a450da",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "BadType",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				_switch.Instance = &_switch.Singleton{
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("ce3532e1-2bcc-47d2-b26f-edac51a450da").
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"type",
						"uplink",
					}).AddRow(
						"ce3532e1-2bcc-47d2-b26f-edac51a450da",
						createUpdateTime,
						createUpdateTime,
						nil,
						"bridge0",
						"a test bridge",
						"junk",
						"em2",
					))
			},
			args: args{
				switchID: &cirrina.SwitchId{
					Value: "ce3532e1-2bcc-47d2-b26f-edac51a450da",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("isoTest")

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

			got, err := client.GetSwitchInfo(context.Background(), testCase.args.switchID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetSwitchInfo() error = %v, wantErr %v", err, testCase.wantErr)

				return
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
