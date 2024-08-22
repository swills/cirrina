package main

import (
	"context"
	"errors"
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
	"gorm.io/gorm"

	"cirrina/cirrina"
	"cirrina/cirrinad/cirrinadtest"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vmnic"
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
			testDB, mock := cirrinadtest.NewMockDB("testDB")

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

//nolint:paralleltest
func Test_server_GetSwitches(t *testing.T) {
	createUpdateTime := time.Now()

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        []string
		wantErr     bool
	}{
		{
			name: "None",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				_switch.Instance = &_switch.Singleton{
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE `switches`.`deleted_at` IS NULL",
					),
				).
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
			want:    nil,
			wantErr: false,
		},
		{
			name: "One",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				_switch.Instance = &_switch.Singleton{
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE `switches`.`deleted_at` IS NULL",
					),
				).
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"name",
								"description",
								"type",
								"uplink",
							},
						).AddRow(
							"2c973451-bd41-4681-a147-c3636cfe0aac",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge27",
							"the 27th test bridge or something",
							"IF",
							"re0",
						),
					)
			},
			want:    []string{"2c973451-bd41-4681-a147-c3636cfe0aac"},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("testDB")

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

			var res cirrina.VMInfo_GetSwitchesClient

			var got []string

			var VMSwitch *cirrina.SwitchId

			res, err = client.GetSwitches(context.Background(), &cirrina.SwitchesQuery{})

			if (err != nil) != testCase.wantErr {
				t.Errorf("GetISOs() error = %v, wantErr %v", err, testCase.wantErr)
			}

			for {
				VMSwitch, err = res.Recv()
				if errors.Is(err, io.EOF) {
					break
				}

				got = append(got, VMSwitch.GetValue())
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

//nolint:paralleltest,maintidx
func Test_server_RemoveSwitch(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		switchID *cirrina.SwitchId
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *cirrina.ReqBool
		wantErr     bool
	}{
		{
			name:        "SuccessIF",
			mockCmdFunc: "Test_server_RemoveSwitchSuccessIF",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				_switch.Instance = &_switch.Singleton{
					SwitchDB: testDB,
				}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3d595921-b225-49f7-b8eb-c416cfd1ea63").
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"type",
						"uplink",
					}).
						AddRow(
							"3d595921-b225-49f7-b8eb-c416cfd1ea63",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge0",
							"a switch",
							"IF",
							"re3",
						),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE `vm_nics`.`deleted_at` IS NULL",
					),
				).
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
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3d595921-b225-49f7-b8eb-c416cfd1ea63").
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"type",
						"uplink",
					}).
						AddRow(
							"3d595921-b225-49f7-b8eb-c416cfd1ea63",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge0",
							"a switch",
							"IF",
							"re3",
						),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE `vm_nics`.`deleted_at` IS NULL",
					),
				).
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

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"DELETE FROM `switches` WHERE `switches`.`id` = ?",
					),
				).
					WithArgs("3d595921-b225-49f7-b8eb-c416cfd1ea63").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args: args{
				switchID: &cirrina.SwitchId{
					Value: "3d595921-b225-49f7-b8eb-c416cfd1ea63",
				},
			},
			want:    func() *cirrina.ReqBool { r := cirrina.ReqBool{Success: true}; return &r }(), //nolint:nlreturn
			wantErr: false,
		},
		{
			name:        "SuccessNG",
			mockCmdFunc: "Test_server_RemoveSwitchSuccessNG",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				_switch.Instance = &_switch.Singleton{
					SwitchDB: testDB,
				}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3d595921-b225-49f7-b8eb-c416cfd1ea63").
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"type",
						"uplink",
					}).
						AddRow(
							"3d595921-b225-49f7-b8eb-c416cfd1ea63",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bnet0",
							"a switch",
							"NG",
							"re3",
						),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE `vm_nics`.`deleted_at` IS NULL",
					),
				).
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
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3d595921-b225-49f7-b8eb-c416cfd1ea63").
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"type",
						"uplink",
					}).
						AddRow(
							"3d595921-b225-49f7-b8eb-c416cfd1ea63",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bnet0",
							"a switch",
							"NG",
							"re3",
						),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE `vm_nics`.`deleted_at` IS NULL",
					),
				).
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

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"DELETE FROM `switches` WHERE `switches`.`id` = ?",
					),
				).
					WithArgs("3d595921-b225-49f7-b8eb-c416cfd1ea63").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args: args{
				switchID: &cirrina.SwitchId{
					Value: "3d595921-b225-49f7-b8eb-c416cfd1ea63",
				},
			},
			want:    func() *cirrina.ReqBool { r := cirrina.ReqBool{Success: true}; return &r }(), //nolint:nlreturn
			wantErr: false,
		},
		{
			name:        "ErrorDeleting",
			mockCmdFunc: "Test_server_RemoveSwitchSuccessIF",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				_switch.Instance = &_switch.Singleton{
					SwitchDB: testDB,
				}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3d595921-b225-49f7-b8eb-c416cfd1ea63").
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"type",
						"uplink",
					}).
						AddRow(
							"3d595921-b225-49f7-b8eb-c416cfd1ea63",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge0",
							"a switch",
							"IF",
							"re3",
						),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE `vm_nics`.`deleted_at` IS NULL",
					),
				).
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
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3d595921-b225-49f7-b8eb-c416cfd1ea63").
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"type",
						"uplink",
					}).
						AddRow(
							"3d595921-b225-49f7-b8eb-c416cfd1ea63",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge0",
							"a switch",
							"IF",
							"re3",
						),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE `vm_nics`.`deleted_at` IS NULL",
					),
				).
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

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"DELETE FROM `switches` WHERE `switches`.`id` = ?",
					),
				).
					WithArgs("3d595921-b225-49f7-b8eb-c416cfd1ea63").
					WillReturnError(gorm.ErrInvalidData)
				mock.ExpectRollback()
			},
			args: args{
				switchID: &cirrina.SwitchId{
					Value: "3d595921-b225-49f7-b8eb-c416cfd1ea63",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "ErrorInvalidType",
			mockCmdFunc: "Test_server_RemoveSwitchSuccessIF",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				_switch.Instance = &_switch.Singleton{
					SwitchDB: testDB,
				}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3d595921-b225-49f7-b8eb-c416cfd1ea63").
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"type",
						"uplink",
					}).
						AddRow(
							"3d595921-b225-49f7-b8eb-c416cfd1ea63",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge0",
							"a switch",
							"junk",
							"re3",
						),
					)
			},
			args: args{
				switchID: &cirrina.SwitchId{
					Value: "3d595921-b225-49f7-b8eb-c416cfd1ea63",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "SuccessIFErrDeleting",
			mockCmdFunc: "Test_server_RemoveSwitchSuccessIFErrDeleting",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				_switch.Instance = &_switch.Singleton{
					SwitchDB: testDB,
				}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3d595921-b225-49f7-b8eb-c416cfd1ea63").
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"type",
						"uplink",
					}).
						AddRow(
							"3d595921-b225-49f7-b8eb-c416cfd1ea63",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge0",
							"a switch",
							"IF",
							"re3",
						),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE `vm_nics`.`deleted_at` IS NULL",
					),
				).
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
				switchID: &cirrina.SwitchId{
					Value: "3d595921-b225-49f7-b8eb-c416cfd1ea63",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "SuccessNGErrDeleting",
			mockCmdFunc: "Test_server_RemoveSwitchSuccessNGErrDeleting",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				_switch.Instance = &_switch.Singleton{
					SwitchDB: testDB,
				}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3d595921-b225-49f7-b8eb-c416cfd1ea63").
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"type",
						"uplink",
					}).
						AddRow(
							"3d595921-b225-49f7-b8eb-c416cfd1ea63",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bnet0",
							"a switch",
							"NG",
							"re3",
						),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE `vm_nics`.`deleted_at` IS NULL",
					),
				).
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
				switchID: &cirrina.SwitchId{
					Value: "3d595921-b225-49f7-b8eb-c416cfd1ea63",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "ErrSwitchInUse",
			mockCmdFunc: "Test_server_RemoveSwitchSuccessIF",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				_switch.Instance = &_switch.Singleton{
					SwitchDB: testDB,
				}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3d595921-b225-49f7-b8eb-c416cfd1ea63").
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"type",
						"uplink",
					}).
						AddRow(
							"3d595921-b225-49f7-b8eb-c416cfd1ea63",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge0",
							"a switch",
							"IF",
							"re3",
						),
					)
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE `vm_nics`.`deleted_at` IS NULL",
					),
				).
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
					}).AddRow(
						"1168e13e-14f7-4dc7-9838-07cbd50d03f9",
						createUpdateTime,
						createUpdateTime,
						nil,
						"test2024082201_int0",
						"test nic",
						"AUTO",
						"IF",
						"VIRTIO-BLK",
						"3d595921-b225-49f7-b8eb-c416cfd1ea63",
						"",
						false,
						1024,
						1024,
						"",
						"",
						1,
					))
			},
			args: args{
				switchID: &cirrina.SwitchId{
					Value: "3d595921-b225-49f7-b8eb-c416cfd1ea63",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "ErrGetSwitch",
			mockCmdFunc: "Test_server_RemoveSwitchSuccessIF",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				_switch.Instance = &_switch.Singleton{
					SwitchDB: testDB,
				}
				vmnic.Instance = &vmnic.Singleton{VMNicDB: testDB}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3d595921-b225-49f7-b8eb-c416cfd1ea63").
					WillReturnError(gorm.ErrInvalidData)
			},
			args: args{
				switchID: &cirrina.SwitchId{
					Value: "3d595921-b225-49f7-b8eb-c416cfd1ea63",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "ErrBadUuid",
			mockCmdFunc: "Test_server_RemoveSwitchSuccessIF",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
			},
			args: args{
				switchID: &cirrina.SwitchId{
					Value: "3d595921-b225-",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)

			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			testDB, mock := cirrinadtest.NewMockDB("testDB")

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

			var got *cirrina.ReqBool

			got, err = client.RemoveSwitch(context.Background(), testCase.args.switchID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("RemoveSwitch() error = %v, wantErr %v", err, testCase.wantErr)

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

// test helpers from here down

//nolint:paralleltest
func Test_server_RemoveSwitchSuccessIF(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_server_RemoveSwitchSuccessNG(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_server_RemoveSwitchSuccessIFErrDeleting(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_server_RemoveSwitchSuccessNGErrDeleting(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}
