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
	"cirrina/cirrinad/requests"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/vm"
	"cirrina/cirrinad/vmnic"
)

//nolint:paralleltest,maintidx
func Test_server_GetVMNicVM(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		vmNicID *cirrina.VmNicId
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *cirrina.VMID
		wantErr     bool
	}{
		{
			name:        "nilUuid",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				vmNicID: nil,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "emptyUuid",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "badUuid",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "8a2cce3e-92ab-4efd-9f8f-2e6",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "unknownNic",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("8a2cce3e-92ab-4efd-9f8f-2e68d52d6885").
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
				vmNicID: &cirrina.VmNicId{
					Value: "8a2cce3e-92ab-4efd-9f8f-2e68d52d6885",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "emptyNicName",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("8a2cce3e-92ab-4efd-9f8f-2e68d52d6885").
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
							"8a2cce3e-92ab-4efd-9f8f-2e68d52d6885",
							createUpdateTime,
							createUpdateTime,
							nil,
							"",
							"a description",
							"12:aa:ff:22:aa:55",
							"VIRTIONET",
							"TAP",
							"369a7524-d399-4784-9652-ca584521ed86",
							"",
							false,
							0,
							0,
							nil,
							nil,
							333,
						),
					)
			},
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "8a2cce3e-92ab-4efd-9f8f-2e68d52d6885",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "notAttachedToThisVM",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				testVM := vm.VM{
					ID:          "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
					Name:        "pizzaTestVM",
					Description: "it follows instruction",
					Status:      "STOPPED",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 723,
						},
						VMID: "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}
				vm.List.VMList[testVM.ID] = &testVM

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("8a2cce3e-92ab-4efd-9f8f-2e68d52d6885").
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
							"8a2cce3e-92ab-4efd-9f8f-2e68d52d6885",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"12:aa:ff:22:aa:55",
							"VIRTIONET",
							"TAP",
							"369a7524-d399-4784-9652-ca584521ed86",
							"",
							false,
							0,
							0,
							nil,
							nil,
							0,
						),
					)
			},
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "8a2cce3e-92ab-4efd-9f8f-2e68d52d6885",
				},
			},
			want: &cirrina.VMID{},
		},
		{
			name: "attachedToThisVM",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				testVM := vm.VM{
					ID:          "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
					Name:        "pizzaTestVM",
					Description: "it follows instruction",
					Status:      "STOPPED",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 723,
						},
						VMID: "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}
				vm.List.VMList[testVM.ID] = &testVM

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("8a2cce3e-92ab-4efd-9f8f-2e68d52d6885").
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
							"8a2cce3e-92ab-4efd-9f8f-2e68d52d6885",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"12:aa:ff:22:aa:55",
							"VIRTIONET",
							"TAP",
							"369a7524-d399-4784-9652-ca584521ed86",
							"",
							false,
							0,
							0,
							nil,
							nil,
							723,
						),
					)
			},
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "8a2cce3e-92ab-4efd-9f8f-2e68d52d6885",
				},
			},
			want: &cirrina.VMID{
				Value: "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
			},
		},
		{
			name: "dupeAttach",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				testVM2 := vm.VM{
					ID:          "8813bea4-6fcf-4c79-8b58-de2fbc9eb029",
					Name:        "cheeseTestVM",
					Description: "it is cheesy",
					Status:      "STOPPED",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 723,
						},
						VMID: "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}

				testVM1 := vm.VM{
					ID:          "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
					Name:        "pizzaTestVM",
					Description: "it follows instruction",
					Status:      "STOPPED",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 723,
						},
						VMID: "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}

				vm.List.VMList[testVM1.ID] = &testVM1
				vm.List.VMList[testVM2.ID] = &testVM2

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("8a2cce3e-92ab-4efd-9f8f-2e68d52d6885").
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
							"8a2cce3e-92ab-4efd-9f8f-2e68d52d6885",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aNic",
							"a description",
							"12:aa:ff:22:aa:55",
							"VIRTIONET",
							"TAP",
							"369a7524-d399-4784-9652-ca584521ed86",
							"",
							false,
							0,
							0,
							nil,
							nil,
							723,
						),
					)
			},
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "8a2cce3e-92ab-4efd-9f8f-2e68d52d6885",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// clear out list from other parallel test runs
			vm.List.VMList = map[string]*vm.VM{}

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

			ctx := context.Background()

			testDB, mock := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mock)

			got, err = client.GetVMNicVM(ctx, testCase.args.vmNicID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetVMNicVM() error = %v, wantErr %v", err, testCase.wantErr)

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

//nolint:paralleltest,maintidx
func Test_server_AddVMNic(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		vmNicInfo *cirrina.VmNicInfo
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *cirrina.VmNicId
		wantErr     bool
	}{
		{
			name:        "nilReq",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				vmNicInfo: nil,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "nilName",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				vmNicInfo: &cirrina.VmNicInfo{
					Name: nil,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "invalidName",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				vmNicInfo: &cirrina.VmNicInfo{
					Name: func() *string { name := "!stretch"; return &name }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "validName",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("StretchTheGiraffe").
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
							}))
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `vm_nics` (`created_at`,`updated_at`,`deleted_at`,`description`,`mac`,`net_dev`,`net_type`,`net_dev_type`,`switch_id`,`rate_limit`,`rate_in`,`rate_out`,`inst_bridge`,`inst_epair`,`id`,`name`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`,`config_id`"), //nolint:lll
				).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "", "AUTO", "",
						"VIRTIONET", "TAP", "", false, 0, 0, "", "",
						sqlmock.AnyArg(), "StretchTheGiraffe").
					WillReturnRows(
						sqlmock.NewRows([]string{"id", "name", "config_id"}).
							AddRow("0bd10557-f1ed-4998-a25d-fc883da80a03", "StretchTheGiraffe", 1),
					)
				mock.ExpectCommit()
			},
			args: args{
				vmNicInfo: &cirrina.VmNicInfo{
					Name: func() *string { name := "StretchTheGiraffe"; return &name }(), //nolint:nlreturn
				},
			},
			want: &cirrina.VmNicId{Value: "0bd10557-f1ed-4998-a25d-fc883da80a03"},
		},
		{
			name: "invalidMac",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
			},
			args: args{
				vmNicInfo: &cirrina.VmNicInfo{
					Name:        func() *string { name := "StretchTheGiraffe"; return &name }(),      //nolint:nlreturn
					Description: func() *string { desc := "a description of a nic"; return &desc }(), //nolint:nlreturn
					Mac:         func() *string { mac := "1234123"; return &mac }(),                  //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalidNetDevType",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
			},
			args: args{
				vmNicInfo: &cirrina.VmNicInfo{
					Name:        func() *string { name := "StretchTheGiraffe"; return &name }(),      //nolint:nlreturn
					Description: func() *string { desc := "a description of a nic"; return &desc }(), //nolint:nlreturn
					Mac:         func() *string { mac := "00:22:44:66:88:aa"; return &mac }(),        //nolint:nlreturn
					Netdevtype: func() *cirrina.NetDevType {
						f := cirrina.NetDevType(-1)

						return &f
					}(),
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalidNetType",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
			},
			args: args{
				vmNicInfo: &cirrina.VmNicInfo{
					Name:        func() *string { name := "StretchTheGiraffe"; return &name }(),      //nolint:nlreturn
					Description: func() *string { desc := "a description of a nic"; return &desc }(), //nolint:nlreturn
					Mac:         func() *string { mac := "00:22:44:66:88:aa"; return &mac }(),        //nolint:nlreturn
					Netdevtype: func() *cirrina.NetDevType {
						f := cirrina.NetDevType_TAP

						return &f
					}(),
					Nettype: func() *cirrina.NetType {
						f := cirrina.NetType(-1)

						return &f
					}(),
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalidSwitchID",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
			},
			args: args{
				vmNicInfo: &cirrina.VmNicInfo{
					Name:        func() *string { name := "StretchTheGiraffe"; return &name }(),      //nolint:nlreturn
					Description: func() *string { desc := "a description of a nic"; return &desc }(), //nolint:nlreturn
					Mac:         func() *string { mac := "00:22:44:66:88:aa"; return &mac }(),        //nolint:nlreturn
					Netdevtype: func() *cirrina.NetDevType {
						f := cirrina.NetDevType_TAP

						return &f
					}(),
					Nettype: func() *cirrina.NetType {
						f := cirrina.NetType_VIRTIONET

						return &f
					}(),
					Switchid: func() *string { switchID := "garbage"; return &switchID }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "validSwitchID",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				_switch.Instance = &_switch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("f7df225b-77a7-46f2-ab9f-aabd62001484").
					WillReturnRows(sqlmock.NewRows(
						[]string{
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
							"f7df225b-77a7-46f2-ab9f-aabd62001484",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge0",
							"a simple test bridge",
							"IF",
							"em0",
						),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("StretchTheGiraffe").
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
							}))

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `vm_nics` (`created_at`,`updated_at`,`deleted_at`,`description`,`mac`,`net_dev`,`net_type`,`net_dev_type`,`switch_id`,`rate_limit`,`rate_in`,`rate_out`,`inst_bridge`,`inst_epair`,`id`,`name`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`,`config_id`"), //nolint:lll
				).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "a description of a nic", "00:22:44:66:88:aa", "",
						"VIRTIONET", "TAP", "f7df225b-77a7-46f2-ab9f-aabd62001484", false, 0, 0, "", "",
						sqlmock.AnyArg(), "StretchTheGiraffe").
					WillReturnRows(
						sqlmock.NewRows([]string{"id", "name", "config_id"}).
							AddRow("0bd10557-f1ed-4998-a25d-fc883da80a03", "StretchTheGiraffe", 1),
					)
				mock.ExpectCommit()
			},
			args: args{
				vmNicInfo: &cirrina.VmNicInfo{
					Name:        func() *string { name := "StretchTheGiraffe"; return &name }(),      //nolint:nlreturn
					Description: func() *string { desc := "a description of a nic"; return &desc }(), //nolint:nlreturn
					Mac:         func() *string { mac := "00:22:44:66:88:aa"; return &mac }(),        //nolint:nlreturn
					Netdevtype: func() *cirrina.NetDevType {
						f := cirrina.NetDevType_TAP

						return &f
					}(),
					Nettype: func() *cirrina.NetType {
						f := cirrina.NetType_VIRTIONET

						return &f
					}(),
					Switchid: func() *string { switchID := "f7df225b-77a7-46f2-ab9f-aabd62001484"; return &switchID }(), //nolint:nlreturn,lll
				},
			},
			want: &cirrina.VmNicId{Value: "0bd10557-f1ed-4998-a25d-fc883da80a03"},
		},
		{
			name: "saveDbErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				_switch.Instance = &_switch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("f7df225b-77a7-46f2-ab9f-aabd62001484").
					WillReturnRows(sqlmock.NewRows(
						[]string{
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
							"f7df225b-77a7-46f2-ab9f-aabd62001484",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge0",
							"a simple test bridge",
							"IF",
							"em0",
						),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("StretchTheGiraffe").
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
							}))

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `vm_nics` (`created_at`,`updated_at`,`deleted_at`,`description`,`mac`,`net_dev`,`net_type`,`net_dev_type`,`switch_id`,`rate_limit`,`rate_in`,`rate_out`,`inst_bridge`,`inst_epair`,`id`,`name`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`,`config_id`"), //nolint:lll
				).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "a description of a nic", "00:22:44:66:88:aa", "",
						"VIRTIONET", "TAP", "f7df225b-77a7-46f2-ab9f-aabd62001484", false, 0, 0, "", "",
						sqlmock.AnyArg(), "StretchTheGiraffe").
					WillReturnError(gorm.ErrInvalidData)
				mock.ExpectRollback()
			},
			args: args{
				vmNicInfo: &cirrina.VmNicInfo{
					Name:        func() *string { name := "StretchTheGiraffe"; return &name }(),      //nolint:nlreturn
					Description: func() *string { desc := "a description of a nic"; return &desc }(), //nolint:nlreturn
					Mac:         func() *string { mac := "00:22:44:66:88:aa"; return &mac }(),        //nolint:nlreturn
					Netdevtype: func() *cirrina.NetDevType {
						f := cirrina.NetDevType_TAP

						return &f
					}(),
					Nettype: func() *cirrina.NetType {
						f := cirrina.NetType_VIRTIONET

						return &f
					}(),
					Switchid: func() *string { switchID := "f7df225b-77a7-46f2-ab9f-aabd62001484"; return &switchID }(), //nolint:nlreturn,lll
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "validSwitchIDWithRateLimit",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				_switch.Instance = &_switch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("f7df225b-77a7-46f2-ab9f-aabd62001484").
					WillReturnRows(sqlmock.NewRows(
						[]string{
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
							"f7df225b-77a7-46f2-ab9f-aabd62001484",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge0",
							"a simple test bridge",
							"IF",
							"em0",
						),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE name = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("StretchTheGiraffe").
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
							}))

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `vm_nics` (`created_at`,`updated_at`,`deleted_at`,`description`,`mac`,`net_dev`,`net_type`,`net_dev_type`,`switch_id`,`rate_limit`,`rate_in`,`rate_out`,`inst_bridge`,`inst_epair`,`id`,`name`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`,`config_id`"), //nolint:lll
				).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "a description of a nic", "00:22:44:66:88:aa", "",
						"VIRTIONET", "TAP", "f7df225b-77a7-46f2-ab9f-aabd62001484", true, 0, 0, "", "",
						sqlmock.AnyArg(), "StretchTheGiraffe").
					WillReturnRows(
						sqlmock.NewRows([]string{"id", "name", "config_id"}).
							AddRow("0bd10557-f1ed-4998-a25d-fc883da80a03", "StretchTheGiraffe", 1),
					)
				mock.ExpectCommit()
			},
			args: args{
				vmNicInfo: &cirrina.VmNicInfo{
					Name:        func() *string { name := "StretchTheGiraffe"; return &name }(),      //nolint:nlreturn
					Description: func() *string { desc := "a description of a nic"; return &desc }(), //nolint:nlreturn
					Mac:         func() *string { mac := "00:22:44:66:88:aa"; return &mac }(),        //nolint:nlreturn
					Netdevtype: func() *cirrina.NetDevType {
						f := cirrina.NetDevType_TAP

						return &f
					}(),
					Ratelimit: func() *bool { r := true; return &r }(), //nolint:nlreturn
					Nettype: func() *cirrina.NetType {
						f := cirrina.NetType_VIRTIONET

						return &f
					}(),
					Switchid: func() *string { switchID := "f7df225b-77a7-46f2-ab9f-aabd62001484"; return &switchID }(), //nolint:nlreturn,lll
				},
			},
			want: &cirrina.VmNicId{Value: "0bd10557-f1ed-4998-a25d-fc883da80a03"},
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

			var got *cirrina.VmNicId

			ctx := context.Background()

			got, err = client.AddVMNic(ctx, testCase.args.vmNicInfo)
			if (err != nil) != testCase.wantErr {
				t.Errorf("AddVMNic() error = %v, wantErr %v", err, testCase.wantErr)

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
func Test_server_GetVMNicsAll(t *testing.T) {
	createUpdateTime := time.Now()

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        []string
		wantErr     bool
	}{
		{
			name: "none",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE `vm_nics`.`deleted_at` IS NULL"),
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
			want:    nil,
			wantErr: false,
		},
		{
			name: "one",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE `vm_nics`.`deleted_at` IS NULL"),
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
								"e332414e-177e-4272-87db-e6cc1914d41b",
								createUpdateTime,
								createUpdateTime,
								nil,
								"someVM_int0",
								"some VMs nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"f48a7dbf-31db-4659-845b-33e350123d32",
								"tap0",
								false,
								0,
								0,
								"",
								"",
								123,
							),
					)
			},
			want:    []string{"e332414e-177e-4272-87db-e6cc1914d41b"},
			wantErr: false,
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

			var got []string

			var VMNic *cirrina.VmNicId

			ctx := context.Background()

			res, err = client.GetVMNicsAll(ctx, &cirrina.VmNicsQuery{})
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetVMNicsAll() error = %v, wantErr %v", err, testCase.wantErr)
			}

			for {
				VMNic, err = res.Recv()
				if errors.Is(err, io.EOF) {
					break
				}

				got = append(got, VMNic.GetValue())
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
func Test_server_GetVMNicInfo(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		vmNicID *cirrina.VmNicId
	}

	tests := []struct {
		name        string
		args        args
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        *cirrina.VmNicInfo
		wantErr     bool
	}{
		{
			name:    "invalidID",
			wantErr: true,
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "67523036-a5c8-4975-",
				},
			},
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
			},
		},
		{
			name:    "notFound",
			wantErr: true,
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "67523036-a5c8-4975-8279-db6640182ebf",
				},
			},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("67523036-a5c8-4975-8279-db6640182ebf").
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
						))
			},
		},
		{
			name:    "foundOneNicNoVM",
			wantErr: false,
			want: func() *cirrina.VmNicInfo {
				name := "test2024072701_int0"
				desc := "another daily test nic"
				mac := "AUTO"
				netDevType := cirrina.NetDevType_TAP
				netType := cirrina.NetType_VIRTIONET
				switchID := "4cca9214-bd3e-406f-b988-0167f2a55121"
				rateLimit := false

				var rateInOut uint64

				testNicInfo := cirrina.VmNicInfo{
					Name:        &name,
					Description: &desc,
					Mac:         &mac,
					Netdevtype:  &netDevType,
					Nettype:     &netType,
					Vmid:        nil,
					Switchid:    &switchID,
					Ratelimit:   &rateLimit,
					Ratein:      &rateInOut,
					Rateout:     &rateInOut,
				}

				return &testNicInfo
			}(),
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "67523036-a5c8-4975-8279-db6640182ebf",
				},
			},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("67523036-a5c8-4975-8279-db6640182ebf").
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
								"test2024072701_int0",
								"another daily test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"4cca9214-bd3e-406f-b988-0167f2a55121",
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
		},
		{
			name:    "foundOneNicAttachedToOneVM",
			wantErr: false,
			want: func() *cirrina.VmNicInfo {
				name := "test2024072701_int0"
				desc := "another daily test nic"
				mac := "AUTO"
				netDevType := cirrina.NetDevType_TAP
				netType := cirrina.NetType_VIRTIONET
				switchID := "4cca9214-bd3e-406f-b988-0167f2a55121"
				rateLimit := false
				vmID := "7563edac-3a68-4950-9dec-ca53dd8c7fca"

				var rateInOut uint64

				testNicInfo := cirrina.VmNicInfo{
					Name:        &name,
					Description: &desc,
					Mac:         &mac,
					Netdevtype:  &netDevType,
					Nettype:     &netType,
					Vmid:        &vmID,
					Switchid:    &switchID,
					Ratelimit:   &rateLimit,
					Ratein:      &rateInOut,
					Rateout:     &rateInOut,
				}

				return &testNicInfo
			}(),
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "67523036-a5c8-4975-8279-db6640182ebf",
				},
			},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				testVM1 := vm.VM{
					ID: "7563edac-3a68-4950-9dec-ca53dd8c7fca",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 23,
						},
						VMID: "7563edac-3a68-4950-9dec-ca53dd8c7fca",
						CPU:  2,
						Mem:  1024,
					},
				}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("67523036-a5c8-4975-8279-db6640182ebf").
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
								"test2024072701_int0",
								"another daily test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"4cca9214-bd3e-406f-b988-0167f2a55121",
								"",
								false,
								0,
								0,
								"",
								"",
								23,
							),
					)
			},
		},
		{
			name:    "foundOneNicAttachedToTwoVMs",
			wantErr: true,
			want:    nil,
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "67523036-a5c8-4975-8279-db6640182ebf",
				},
			},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				testVM1 := vm.VM{
					ID: "7563edac-3a68-4950-9dec-ca53dd8c7fca",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 23,
						},
						VMID: "7563edac-3a68-4950-9dec-ca53dd8c7fca",
						CPU:  2,
						Mem:  1024,
					},
				}

				testVM2 := vm.VM{
					ID: "2d29b830-4433-4a4b-a13f-376640b3a8f9",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 23,
						},
						VMID: "2d29b830-4433-4a4b-a13f-376640b3a8f9",
						CPU:  2,
						Mem:  1024,
					},
				}

				vm.List.VMList[testVM1.ID] = &testVM1
				vm.List.VMList[testVM2.ID] = &testVM2

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("67523036-a5c8-4975-8279-db6640182ebf").
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
								"test2024072701_int0",
								"another daily test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"4cca9214-bd3e-406f-b988-0167f2a55121",
								"",
								false,
								0,
								0,
								"",
								"",
								23,
							),
					)
			},
		},
		{
			name:    "foundOneNicNoVMNetTypeE1000",
			wantErr: false,
			want: func() *cirrina.VmNicInfo {
				name := "test2024072701_int0"
				desc := "another daily test nic"
				mac := "AUTO"
				netDevType := cirrina.NetDevType_TAP
				netType := cirrina.NetType_E1000
				switchID := "4cca9214-bd3e-406f-b988-0167f2a55121"
				rateLimit := false

				var rateInOut uint64

				testNicInfo := cirrina.VmNicInfo{
					Name:        &name,
					Description: &desc,
					Mac:         &mac,
					Netdevtype:  &netDevType,
					Nettype:     &netType,
					Vmid:        nil,
					Switchid:    &switchID,
					Ratelimit:   &rateLimit,
					Ratein:      &rateInOut,
					Rateout:     &rateInOut,
				}

				return &testNicInfo
			}(),
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "67523036-a5c8-4975-8279-db6640182ebf",
				},
			},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("67523036-a5c8-4975-8279-db6640182ebf").
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
								"test2024072701_int0",
								"another daily test nic",
								"AUTO",
								"E1000",
								"TAP",
								"4cca9214-bd3e-406f-b988-0167f2a55121",
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
		},
		{
			name:    "foundOneNicNoVMNetDevTypeVMNet",
			wantErr: false,
			want: func() *cirrina.VmNicInfo {
				name := "test2024072701_int0"
				desc := "another daily test nic"
				mac := "AUTO"
				netDevType := cirrina.NetDevType_VMNET
				netType := cirrina.NetType_VIRTIONET
				switchID := "4cca9214-bd3e-406f-b988-0167f2a55121"
				rateLimit := false

				var rateInOut uint64

				testNicInfo := cirrina.VmNicInfo{
					Name:        &name,
					Description: &desc,
					Mac:         &mac,
					Netdevtype:  &netDevType,
					Nettype:     &netType,
					Vmid:        nil,
					Switchid:    &switchID,
					Ratelimit:   &rateLimit,
					Ratein:      &rateInOut,
					Rateout:     &rateInOut,
				}

				return &testNicInfo
			}(),
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "67523036-a5c8-4975-8279-db6640182ebf",
				},
			},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("67523036-a5c8-4975-8279-db6640182ebf").
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
								"test2024072701_int0",
								"another daily test nic",
								"AUTO",
								"VIRTIONET",
								"VMNET",
								"4cca9214-bd3e-406f-b988-0167f2a55121",
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
		},
		{
			name:    "foundOneNicNoVMNetDevTypeNetGraph",
			wantErr: false,
			want: func() *cirrina.VmNicInfo {
				name := "test2024072701_int0"
				desc := "another daily test nic"
				mac := "AUTO"
				netDevType := cirrina.NetDevType_NETGRAPH
				netType := cirrina.NetType_VIRTIONET
				switchID := "4cca9214-bd3e-406f-b988-0167f2a55121"
				rateLimit := false

				var rateInOut uint64

				testNicInfo := cirrina.VmNicInfo{
					Name:        &name,
					Description: &desc,
					Mac:         &mac,
					Netdevtype:  &netDevType,
					Nettype:     &netType,
					Vmid:        nil,
					Switchid:    &switchID,
					Ratelimit:   &rateLimit,
					Ratein:      &rateInOut,
					Rateout:     &rateInOut,
				}

				return &testNicInfo
			}(),
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "67523036-a5c8-4975-8279-db6640182ebf",
				},
			},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("67523036-a5c8-4975-8279-db6640182ebf").
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
								"test2024072701_int0",
								"another daily test nic",
								"AUTO",
								"VIRTIONET",
								"NETGRAPH",
								"4cca9214-bd3e-406f-b988-0167f2a55121",
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
		},
		{
			name:    "foundOneNicNoVMBadNetType",
			wantErr: true,
			want:    nil,
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "67523036-a5c8-4975-8279-db6640182ebf",
				},
			},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("67523036-a5c8-4975-8279-db6640182ebf").
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
								"test2024072701_int0",
								"another daily test nic",
								"AUTO",
								"garbage",
								"TAP",
								"4cca9214-bd3e-406f-b988-0167f2a55121",
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
		},
		{
			name:    "foundOneNicNoVMBadNetDevType",
			wantErr: true,
			want:    nil,
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "67523036-a5c8-4975-8279-db6640182ebf",
				},
			},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("67523036-a5c8-4975-8279-db6640182ebf").
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
								"test2024072701_int0",
								"another daily test nic",
								"AUTO",
								"VIRTIONET",
								"garbage",
								"4cca9214-bd3e-406f-b988-0167f2a55121",
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
		},
		{
			name:    "foundOneNicNoVMBadName",
			wantErr: true,
			want:    nil,
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "67523036-a5c8-4975-8279-db6640182ebf",
				},
			},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("67523036-a5c8-4975-8279-db6640182ebf").
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
								"",
								"another daily test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"4cca9214-bd3e-406f-b988-0167f2a55121",
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
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// clear out list(s) from other parallel test runs
			vm.List.VMList = map[string]*vm.VM{}

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

			var got *cirrina.VmNicInfo

			ctx := context.Background()

			got, err = client.GetVMNicInfo(ctx, testCase.args.vmNicID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetVMNicInfo() error = %v, wantErr %v", err, testCase.wantErr)

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

//nolint:paralleltest,maintidx
func Test_server_SetVMNicSwitch(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		setVMNicSwitchReq *cirrina.SetVmNicSwitchReq
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *cirrina.ReqBool
		wantErr     bool
	}{
		{
			name:        "nilReq",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				setVMNicSwitchReq: nil,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "nilNicID",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				setVMNicSwitchReq: &cirrina.SetVmNicSwitchReq{
					Vmnicid: nil,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "emptyNicID",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				setVMNicSwitchReq: &cirrina.SetVmNicSwitchReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "badNicID",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				setVMNicSwitchReq: &cirrina.SetVmNicSwitchReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "d454100f-3f1c-4679-8a5a-03f6",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "nicIDNotFound",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("d454100f-3f1c-4679-8a5a-03f65de49a08").
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
							}))
			},
			args: args{
				setVMNicSwitchReq: &cirrina.SetVmNicSwitchReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "d454100f-3f1c-4679-8a5a-03f65de49a08",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "emptyNicName",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("d454100f-3f1c-4679-8a5a-03f65de49a08").
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
							}).
							AddRow(
								"d454100f-3f1c-4679-8a5a-03f65de49a08",
								createUpdateTime,
								createUpdateTime,
								nil,
								"",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"5a919407-07dc-4332-825b-3fd65a8804ec",
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
				setVMNicSwitchReq: &cirrina.SetVmNicSwitchReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "d454100f-3f1c-4679-8a5a-03f65de49a08",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "switchIdNotSet",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("d454100f-3f1c-4679-8a5a-03f65de49a08").
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
							}).
							AddRow(
								"d454100f-3f1c-4679-8a5a-03f65de49a08",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2023072702_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"5a919407-07dc-4332-825b-3fd65a8804ec",
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
				setVMNicSwitchReq: &cirrina.SetVmNicSwitchReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "d454100f-3f1c-4679-8a5a-03f65de49a08",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "switchIdEmpty",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("d454100f-3f1c-4679-8a5a-03f65de49a08").
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
							}).
							AddRow(
								"d454100f-3f1c-4679-8a5a-03f65de49a08",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2023072702_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"5a919407-07dc-4332-825b-3fd65a8804ec",
								"",
								false,
								0,
								0,
								"",
								"",
								0,
							),
					)

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(0, "another test nic", "", "", "AUTO", "test2023072702_int0", "", "TAP",
						"VIRTIONET", 0, false, 0, "", sqlmock.AnyArg(),
						"d454100f-3f1c-4679-8a5a-03f65de49a08").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args: args{
				setVMNicSwitchReq: &cirrina.SetVmNicSwitchReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "d454100f-3f1c-4679-8a5a-03f65de49a08",
					},
					Switchid: &cirrina.SwitchId{
						Value: "",
					},
				},
			},
			want:    &cirrina.ReqBool{Success: true},
			wantErr: false,
		},
		{
			name: "switchIdNotEmpty",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				_switch.Instance = &_switch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("d454100f-3f1c-4679-8a5a-03f65de49a08").
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
							}).
							AddRow(
								"d454100f-3f1c-4679-8a5a-03f65de49a08",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2023072702_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"5a919407-07dc-4332-825b-3fd65a8804ec",
								"",
								false,
								0,
								0,
								"",
								"",
								0,
							),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("714ec740-13dd-4696-8469-e3d58eca2468").
					WillReturnRows(sqlmock.NewRows(
						[]string{
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
							"714ec740-13dd-4696-8469-e3d58eca2468",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge0",
							"a simple test bridge",
							"IF",
							"em0",
						),
					)

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(0, "another test nic", "", "", "AUTO", "test2023072702_int0", "", "TAP",
						"VIRTIONET", 0, false, 0, "714ec740-13dd-4696-8469-e3d58eca2468", sqlmock.AnyArg(),
						"d454100f-3f1c-4679-8a5a-03f65de49a08").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args: args{
				setVMNicSwitchReq: &cirrina.SetVmNicSwitchReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "d454100f-3f1c-4679-8a5a-03f65de49a08",
					},
					Switchid: &cirrina.SwitchId{
						Value: "714ec740-13dd-4696-8469-e3d58eca2468",
					},
				},
			},
			want:    &cirrina.ReqBool{Success: true},
			wantErr: false,
		},
		{
			name: "badSwitchId",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				_switch.Instance = &_switch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("d454100f-3f1c-4679-8a5a-03f65de49a08").
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
							}).
							AddRow(
								"d454100f-3f1c-4679-8a5a-03f65de49a08",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2023072702_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"5a919407-07dc-4332-825b-3fd65a8804ec",
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
				setVMNicSwitchReq: &cirrina.SetVmNicSwitchReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "d454100f-3f1c-4679-8a5a-03f65de49a08",
					},
					Switchid: &cirrina.SwitchId{
						Value: "714ec740-13dd-4696-8",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "switchGetErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				_switch.Instance = &_switch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("d454100f-3f1c-4679-8a5a-03f65de49a08").
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
							}).
							AddRow(
								"d454100f-3f1c-4679-8a5a-03f65de49a08",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2023072702_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"5a919407-07dc-4332-825b-3fd65a8804ec",
								"",
								false,
								0,
								0,
								"",
								"",
								0,
							),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("714ec740-13dd-4696-8469-e3d58eca2468").
					WillReturnRows(sqlmock.NewRows(
						[]string{
							"id",
							"created_at",
							"updated_at",
							"deleted_at",
							"name",
							"description",
							"type",
							"uplink",
						}),
					)
			},
			args: args{
				setVMNicSwitchReq: &cirrina.SetVmNicSwitchReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "d454100f-3f1c-4679-8a5a-03f65de49a08",
					},
					Switchid: &cirrina.SwitchId{
						Value: "714ec740-13dd-4696-8469-e3d58eca2468",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "switchNameEmpty",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				_switch.Instance = &_switch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("d454100f-3f1c-4679-8a5a-03f65de49a08").
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
							}).
							AddRow(
								"d454100f-3f1c-4679-8a5a-03f65de49a08",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2023072702_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"5a919407-07dc-4332-825b-3fd65a8804ec",
								"",
								false,
								0,
								0,
								"",
								"",
								0,
							),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("714ec740-13dd-4696-8469-e3d58eca2468").
					WillReturnRows(sqlmock.NewRows(
						[]string{
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
							"714ec740-13dd-4696-8469-e3d58eca2468",
							createUpdateTime,
							createUpdateTime,
							nil,
							"",
							"a simple test bridge",
							"IF",
							"em0",
						),
					)
			},
			args: args{
				setVMNicSwitchReq: &cirrina.SetVmNicSwitchReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "d454100f-3f1c-4679-8a5a-03f65de49a08",
					},
					Switchid: &cirrina.SwitchId{
						Value: "714ec740-13dd-4696-8469-e3d58eca2468",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "setSwitchErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				_switch.Instance = &_switch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("d454100f-3f1c-4679-8a5a-03f65de49a08").
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
							}).
							AddRow(
								"d454100f-3f1c-4679-8a5a-03f65de49a08",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2023072702_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"5a919407-07dc-4332-825b-3fd65a8804ec",
								"",
								false,
								0,
								0,
								"",
								"",
								0,
							),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("714ec740-13dd-4696-8469-e3d58eca2468").
					WillReturnRows(sqlmock.NewRows(
						[]string{
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
							"714ec740-13dd-4696-8469-e3d58eca2468",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge0",
							"a simple test bridge",
							"IF",
							"em0",
						),
					)

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(0, "another test nic", "", "", "AUTO", "test2023072702_int0", "", "TAP",
						"VIRTIONET", 0, false, 0, "714ec740-13dd-4696-8469-e3d58eca2468", sqlmock.AnyArg(),
						"d454100f-3f1c-4679-8a5a-03f65de49a08").
					WillReturnError(gorm.ErrInvalidData)
				mock.ExpectRollback()
			},
			args: args{
				setVMNicSwitchReq: &cirrina.SetVmNicSwitchReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "d454100f-3f1c-4679-8a5a-03f65de49a08",
					},
					Switchid: &cirrina.SwitchId{
						Value: "714ec740-13dd-4696-8469-e3d58eca2468",
					},
				},
			},
			want:    nil,
			wantErr: true,
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

			var got *cirrina.ReqBool

			ctx := context.Background()

			got, err = client.SetVMNicSwitch(ctx, testCase.args.setVMNicSwitchReq)
			if (err != nil) != testCase.wantErr {
				t.Errorf("SetVMNicSwitch() error = %v, wantErr %v", err, testCase.wantErr)

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

//nolint:paralleltest,maintidx
func Test_server_RemoveVMNic(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		vmNicID *cirrina.VmNicId
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *cirrina.ReqBool
		wantErr     bool
	}{
		{
			name:        "nilReq",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				vmNicID: nil,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "emptyNicID",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "badNicID",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "61ba8ba5-8dc3-465a",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "nicNotFound",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("61ba8ba5-8dc3-465a-a482-d043f53b9053").
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
							}))
			},
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "61ba8ba5-8dc3-465a-a482-d043f53b9053",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "nicEmptyName",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("61ba8ba5-8dc3-465a-a482-d043f53b9053").
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
							}).
							AddRow(
								"61ba8ba5-8dc3-465a-a482-d043f53b9053",
								createUpdateTime,
								createUpdateTime,
								nil,
								"",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"5a919407-07dc-4332-825b-3fd65a8804ec",
								"",
								false,
								0,
								0,
								"",
								"",
								0,
							))
			},
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "61ba8ba5-8dc3-465a-a482-d043f53b9053",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "SuccessNoVMs",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("61ba8ba5-8dc3-465a-a482-d043f53b9053").
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
							}).
							AddRow(
								"61ba8ba5-8dc3-465a-a482-d043f53b9053",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2023072702_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"5a919407-07dc-4332-825b-3fd65a8804ec",
								"",
								false,
								0,
								0,
								"",
								"",
								0,
							))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"DELETE FROM `vm_nics` WHERE `vm_nics`.`id` = ?",
					),
				).
					WithArgs("61ba8ba5-8dc3-465a-a482-d043f53b9053").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "61ba8ba5-8dc3-465a-a482-d043f53b9053",
				},
			},
			want:    &cirrina.ReqBool{Success: true},
			wantErr: false,
		},
		{
			name: "saveErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("61ba8ba5-8dc3-465a-a482-d043f53b9053").
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
							}).
							AddRow(
								"61ba8ba5-8dc3-465a-a482-d043f53b9053",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2023072702_int0",
								"another test nic",
								"AUTO",
								"VIRTIONET",
								"TAP",
								"5a919407-07dc-4332-825b-3fd65a8804ec",
								"",
								false,
								0,
								0,
								"",
								"",
								0,
							))
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"DELETE FROM `vm_nics` WHERE `vm_nics`.`id` = ?",
					),
				).
					WithArgs("61ba8ba5-8dc3-465a-a482-d043f53b9053").
					WillReturnError(gorm.ErrInvalidData)
				mock.ExpectRollback()
			},
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "61ba8ba5-8dc3-465a-a482-d043f53b9053",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "nicConnectedToOneVMs",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				testVM1 := vm.VM{
					ID:          "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
					Name:        "pizzaTestVM",
					Description: "it follows instruction",
					Status:      "STOPPED",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 6543,
						},
						VMID: "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}

				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).
					WithArgs("61ba8ba5-8dc3-465a-a482-d043f53b9053").
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
							}).
							AddRow(
								"61ba8ba5-8dc3-465a-a482-d043f53b9053",
								createUpdateTime,
								createUpdateTime,
								nil,
								"aNic",
								"a description",
								"00:11:22:33:44:55",
								"VIRTIONET",
								"TAP",
								"6ad8f637-22ee-43aa-b9d8-10df5fd7f50f",
								"",
								false,
								0,
								0,
								nil,
								nil,
								6543,
							),
					)
			},
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "61ba8ba5-8dc3-465a-a482-d043f53b9053",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "getNicsErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				testVM1 := vm.VM{
					ID:          "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
					Name:        "pizzaTestVM",
					Description: "it follows instruction",
					Status:      "STOPPED",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 6543,
						},
						VMID: "42e72023-0a36-4e1b-aef2-b3fd31ba1d4e",
						CPU:  2,
						Mem:  1024,
					},
					ISOs:  nil,
					Disks: nil,
				}

				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL",
					),
				).
					WithArgs("61ba8ba5-8dc3-465a-a482-d043f53b9053").
					WillReturnError(gorm.ErrInvalidData)
			},
			args: args{
				vmNicID: &cirrina.VmNicId{
					Value: "61ba8ba5-8dc3-465a-a482-d043f53b9053",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// clear out list(s) from other parallel test runs
			disk.List.DiskList = map[string]*disk.Disk{}
			vm.List.VMList = map[string]*vm.VM{}

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

			var got *cirrina.ReqBool

			ctx := context.Background()

			got, err = client.RemoveVMNic(ctx, testCase.args.vmNicID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("RemoveVMNic() error = %v, wantErr %v", err, testCase.wantErr)

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

//nolint:paralleltest,maintidx
func Test_server_UpdateVMNic(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		vmNicInfoUpdate *cirrina.VmNicInfoUpdate
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *cirrina.ReqBool
		wantErr     bool
	}{
		{
			name:        "nilReq",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				vmNicInfoUpdate: nil,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "nilNicID",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				vmNicInfoUpdate: &cirrina.VmNicInfoUpdate{
					Vmnicid: nil,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "emptyNicID",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				vmNicInfoUpdate: &cirrina.VmNicInfoUpdate{
					Vmnicid: &cirrina.VmNicId{
						Value: "",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "badNicID",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				vmNicInfoUpdate: &cirrina.VmNicInfoUpdate{
					Vmnicid: &cirrina.VmNicId{
						Value: "133d38b7-fe09-4f37-9d24-",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "nicNotFound",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("133d38b7-fe09-4f37-9d24-429fbe8589ea").
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
						),
					)
			},
			args: args{
				vmNicInfoUpdate: &cirrina.VmNicInfoUpdate{
					Vmnicid: &cirrina.VmNicId{
						Value: "133d38b7-fe09-4f37-9d24-429fbe8589ea",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "emptyUpdateSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("133d38b7-fe09-4f37-9d24-429fbe8589ea").
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
								"133d38b7-fe09-4f37-9d24-429fbe8589ea",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024050501_int0",
								"another test nic",
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

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(0, "another test nic", "", "", "AUTO", "test2024050501_int0", "", "TAP",
						"VIRTIONET", 0, false, 0, "", sqlmock.AnyArg(), "133d38b7-fe09-4f37-9d24-429fbe8589ea").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args: args{
				vmNicInfoUpdate: &cirrina.VmNicInfoUpdate{
					Vmnicid: &cirrina.VmNicId{
						Value: "133d38b7-fe09-4f37-9d24-429fbe8589ea",
					},
				},
			},
			want:    &cirrina.ReqBool{Success: true},
			wantErr: false,
		},
		{
			name: "totalUpdateSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				_switch.Instance = &_switch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("133d38b7-fe09-4f37-9d24-429fbe8589ea").
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
								"133d38b7-fe09-4f37-9d24-429fbe8589ea",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024050501_int0",
								"another test nic",
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

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("a9c478c6-b949-4cec-8f81-2d1b8363d547").
					WillReturnRows(sqlmock.NewRows(
						[]string{
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
							"a9c478c6-b949-4cec-8f81-2d1b8363d547",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge3",
							"a third simple test bridge",
							"IF",
							"em4",
						),
					)

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(0, "new description", "", "", "00:22:44:66:88:aa", "test2024050502_int0", "", "VMNET",
						"E1000", 62626128, true, 81239012, "a9c478c6-b949-4cec-8f81-2d1b8363d547", sqlmock.AnyArg(), "133d38b7-fe09-4f37-9d24-429fbe8589ea"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args: args{
				vmNicInfoUpdate: &cirrina.VmNicInfoUpdate{
					Vmnicid: &cirrina.VmNicId{
						Value: "133d38b7-fe09-4f37-9d24-429fbe8589ea",
					},
					Name:        func() *string { name := "test2024050502_int0"; return &name }(),                //nolint:nlreturn
					Description: func() *string { desc := "new description"; return &desc }(),                    //nolint:nlreturn
					Mac:         func() *string { mac := "00:22:44:66:88:aa"; return &mac }(),                    //nolint:nlreturn
					Netdevtype:  func() *cirrina.NetDevType { f := cirrina.NetDevType_VMNET; return &f }(),       //nolint:nlreturn
					Nettype:     func() *cirrina.NetType { f := cirrina.NetType_E1000; return &f }(),             //nolint:nlreturn
					Switchid:    func() *string { sid := "a9c478c6-b949-4cec-8f81-2d1b8363d547"; return &sid }(), //nolint:nlreturn
					Ratelimit:   func() *bool { rl := true; return &rl }(),                                       //nolint:nlreturn
					Ratein:      func() *uint64 { var ri uint64 = 62626128; return &ri }(),                       //nolint:nlreturn
					Rateout:     func() *uint64 { var ro uint64 = 81239012; return &ro }(),                       //nolint:nlreturn
				},
			},
			want:    &cirrina.ReqBool{Success: true},
			wantErr: false,
		},
		{
			name: "badNicName",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				_switch.Instance = &_switch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("133d38b7-fe09-4f37-9d24-429fbe8589ea").
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
								"133d38b7-fe09-4f37-9d24-429fbe8589ea",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024050501_int0",
								"another test nic",
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
				vmNicInfoUpdate: &cirrina.VmNicInfoUpdate{
					Vmnicid: &cirrina.VmNicId{
						Value: "133d38b7-fe09-4f37-9d24-429fbe8589ea",
					},
					Name:        func() *string { name := "test2024050502_int0;"; return &name }(),               //nolint:nlreturn
					Description: func() *string { desc := "new description"; return &desc }(),                    //nolint:nlreturn
					Mac:         func() *string { mac := "00:22:44:66:88:aa"; return &mac }(),                    //nolint:nlreturn
					Netdevtype:  func() *cirrina.NetDevType { f := cirrina.NetDevType_VMNET; return &f }(),       //nolint:nlreturn
					Nettype:     func() *cirrina.NetType { f := cirrina.NetType_E1000; return &f }(),             //nolint:nlreturn
					Switchid:    func() *string { sid := "a9c478c6-b949-4cec-8f81-2d1b8363d547"; return &sid }(), //nolint:nlreturn
					Ratelimit:   func() *bool { rl := true; return &rl }(),                                       //nolint:nlreturn
					Ratein:      func() *uint64 { var ri uint64 = 62626128; return &ri }(),                       //nolint:nlreturn
					Rateout:     func() *uint64 { var ro uint64 = 81239012; return &ro }(),                       //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "badMac",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				_switch.Instance = &_switch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("133d38b7-fe09-4f37-9d24-429fbe8589ea").
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
								"133d38b7-fe09-4f37-9d24-429fbe8589ea",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024050501_int0",
								"another test nic",
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
				vmNicInfoUpdate: &cirrina.VmNicInfoUpdate{
					Vmnicid: &cirrina.VmNicId{
						Value: "133d38b7-fe09-4f37-9d24-429fbe8589ea",
					},
					Name:        func() *string { name := "test2024050502_int0"; return &name }(),                //nolint:nlreturn
					Description: func() *string { desc := "new description"; return &desc }(),                    //nolint:nlreturn
					Mac:         func() *string { mac := "00:22:44:66:88"; return &mac }(),                       //nolint:nlreturn
					Netdevtype:  func() *cirrina.NetDevType { f := cirrina.NetDevType_VMNET; return &f }(),       //nolint:nlreturn
					Nettype:     func() *cirrina.NetType { f := cirrina.NetType_E1000; return &f }(),             //nolint:nlreturn
					Switchid:    func() *string { sid := "a9c478c6-b949-4cec-8f81-2d1b8363d547"; return &sid }(), //nolint:nlreturn
					Ratelimit:   func() *bool { rl := true; return &rl }(),                                       //nolint:nlreturn
					Ratein:      func() *uint64 { var ri uint64 = 62626128; return &ri }(),                       //nolint:nlreturn
					Rateout:     func() *uint64 { var ro uint64 = 81239012; return &ro }(),                       //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "badNetDevType",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				_switch.Instance = &_switch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("133d38b7-fe09-4f37-9d24-429fbe8589ea").
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
								"133d38b7-fe09-4f37-9d24-429fbe8589ea",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024050501_int0",
								"another test nic",
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
				vmNicInfoUpdate: &cirrina.VmNicInfoUpdate{
					Vmnicid: &cirrina.VmNicId{
						Value: "133d38b7-fe09-4f37-9d24-429fbe8589ea",
					},
					Name:        func() *string { name := "test2024050502_int0"; return &name }(),                //nolint:nlreturn
					Description: func() *string { desc := "new description"; return &desc }(),                    //nolint:nlreturn
					Mac:         func() *string { mac := "00:22:44:66:88:12"; return &mac }(),                    //nolint:nlreturn
					Netdevtype:  func() *cirrina.NetDevType { f := cirrina.NetDevType(-1); return &f }(),         //nolint:nlreturn
					Nettype:     func() *cirrina.NetType { f := cirrina.NetType_E1000; return &f }(),             //nolint:nlreturn
					Switchid:    func() *string { sid := "a9c478c6-b949-4cec-8f81-2d1b8363d547"; return &sid }(), //nolint:nlreturn
					Ratelimit:   func() *bool { rl := true; return &rl }(),                                       //nolint:nlreturn
					Ratein:      func() *uint64 { var ri uint64 = 62626128; return &ri }(),                       //nolint:nlreturn
					Rateout:     func() *uint64 { var ro uint64 = 81239012; return &ro }(),                       //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "badNetType",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				_switch.Instance = &_switch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("133d38b7-fe09-4f37-9d24-429fbe8589ea").
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
								"133d38b7-fe09-4f37-9d24-429fbe8589ea",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024050501_int0",
								"another test nic",
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
				vmNicInfoUpdate: &cirrina.VmNicInfoUpdate{
					Vmnicid: &cirrina.VmNicId{
						Value: "133d38b7-fe09-4f37-9d24-429fbe8589ea",
					},
					Name:        func() *string { name := "test2024050502_int0"; return &name }(),                //nolint:nlreturn
					Description: func() *string { desc := "new description"; return &desc }(),                    //nolint:nlreturn
					Mac:         func() *string { mac := "00:22:44:66:88:aa"; return &mac }(),                    //nolint:nlreturn
					Netdevtype:  func() *cirrina.NetDevType { f := cirrina.NetDevType_VMNET; return &f }(),       //nolint:nlreturn
					Nettype:     func() *cirrina.NetType { f := cirrina.NetType(-1); return &f }(),               //nolint:nlreturn
					Switchid:    func() *string { sid := "a9c478c6-b949-4cec-8f81-2d1b8363d547"; return &sid }(), //nolint:nlreturn
					Ratelimit:   func() *bool { rl := true; return &rl }(),                                       //nolint:nlreturn
					Ratein:      func() *uint64 { var ri uint64 = 62626128; return &ri }(),                       //nolint:nlreturn
					Rateout:     func() *uint64 { var ro uint64 = 81239012; return &ro }(),                       //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "badSwitchID",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				_switch.Instance = &_switch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("133d38b7-fe09-4f37-9d24-429fbe8589ea").
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
								"133d38b7-fe09-4f37-9d24-429fbe8589ea",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024050501_int0",
								"another test nic",
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
				vmNicInfoUpdate: &cirrina.VmNicInfoUpdate{
					Vmnicid: &cirrina.VmNicId{
						Value: "133d38b7-fe09-4f37-9d24-429fbe8589ea",
					},
					Name:        func() *string { name := "test2024050502_int0"; return &name }(),          //nolint:nlreturn
					Description: func() *string { desc := "new description"; return &desc }(),              //nolint:nlreturn
					Mac:         func() *string { mac := "00:22:44:66:88:aa"; return &mac }(),              //nolint:nlreturn
					Netdevtype:  func() *cirrina.NetDevType { f := cirrina.NetDevType_VMNET; return &f }(), //nolint:nlreturn
					Nettype:     func() *cirrina.NetType { f := cirrina.NetType_E1000; return &f }(),       //nolint:nlreturn
					Switchid:    func() *string { sid := "a9c478c6-b949-4cec-8"; return &sid }(),           //nolint:nlreturn
					Ratelimit:   func() *bool { rl := true; return &rl }(),                                 //nolint:nlreturn
					Ratein:      func() *uint64 { var ri uint64 = 62626128; return &ri }(),                 //nolint:nlreturn
					Rateout:     func() *uint64 { var ro uint64 = 81239012; return &ro }(),                 //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "totalUpdateSaveErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				_switch.Instance = &_switch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("133d38b7-fe09-4f37-9d24-429fbe8589ea").
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
								"133d38b7-fe09-4f37-9d24-429fbe8589ea",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024050501_int0",
								"another test nic",
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

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("a9c478c6-b949-4cec-8f81-2d1b8363d547").
					WillReturnRows(sqlmock.NewRows(
						[]string{
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
							"a9c478c6-b949-4cec-8f81-2d1b8363d547",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bridge3",
							"a third simple test bridge",
							"IF",
							"em4",
						),
					)

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(0, "new description", "", "", "00:22:44:66:88:aa", "test2024050502_int0", "", "VMNET",
						"E1000", 62626128, true, 81239012, "a9c478c6-b949-4cec-8f81-2d1b8363d547", sqlmock.AnyArg(), "133d38b7-fe09-4f37-9d24-429fbe8589ea"). //nolint:lll
					WillReturnError(gorm.ErrInvalidData)
				mock.ExpectRollback()
			},
			args: args{
				vmNicInfoUpdate: &cirrina.VmNicInfoUpdate{
					Vmnicid: &cirrina.VmNicId{
						Value: "133d38b7-fe09-4f37-9d24-429fbe8589ea",
					},
					Name:        func() *string { name := "test2024050502_int0"; return &name }(),                //nolint:nlreturn
					Description: func() *string { desc := "new description"; return &desc }(),                    //nolint:nlreturn
					Mac:         func() *string { mac := "00:22:44:66:88:aa"; return &mac }(),                    //nolint:nlreturn
					Netdevtype:  func() *cirrina.NetDevType { f := cirrina.NetDevType_VMNET; return &f }(),       //nolint:nlreturn
					Nettype:     func() *cirrina.NetType { f := cirrina.NetType_E1000; return &f }(),             //nolint:nlreturn
					Switchid:    func() *string { sid := "a9c478c6-b949-4cec-8f81-2d1b8363d547"; return &sid }(), //nolint:nlreturn
					Ratelimit:   func() *bool { rl := true; return &rl }(),                                       //nolint:nlreturn
					Ratein:      func() *uint64 { var ri uint64 = 62626128; return &ri }(),                       //nolint:nlreturn
					Rateout:     func() *uint64 { var ro uint64 = 81239012; return &ro }(),                       //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
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

			var got *cirrina.ReqBool

			ctx := context.Background()

			got, err = client.UpdateVMNic(ctx, testCase.args.vmNicInfoUpdate)
			if (err != nil) != testCase.wantErr {
				t.Errorf("UpdateVMNic() error = %v, wantErr %v", err, testCase.wantErr)

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

//nolint:paralleltest,maintidx
func Test_server_CloneVMNic(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		cloneReq *cirrina.VmNicCloneReq
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *cirrina.RequestID
		wantErr     bool
	}{
		{
			name:        "nilReq",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				cloneReq: nil,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "nilNicID",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				cloneReq: &cirrina.VmNicCloneReq{
					Vmnicid: nil,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "emptyNicID",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				cloneReq: &cirrina.VmNicCloneReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "nilNewName",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				cloneReq: &cirrina.VmNicCloneReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf",
					},
					NewVmNicName: nil,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "emptyNewName",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				cloneReq: &cirrina.VmNicCloneReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf",
					},
					NewVmNicName: &wrapperspb.StringValue{
						Value: "",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "badUuid",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				cloneReq: &cirrina.VmNicCloneReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "4abdb6d2-a80f-4522-8e87-a4b9",
					},
					NewVmNicName: &wrapperspb.StringValue{
						Value: "test2024050503_int0",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "nicNotFound",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf").
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
						),
					)
			},
			args: args{
				cloneReq: &cirrina.VmNicCloneReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf",
					},
					NewVmNicName: &wrapperspb.StringValue{
						Value: "test2024050503_int0",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "emptyNicNameInDB",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf").
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
								"4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf",
								createUpdateTime,
								createUpdateTime,
								nil,
								"",
								"another test nic",
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
				cloneReq: &cirrina.VmNicCloneReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf",
					},
					NewVmNicName: &wrapperspb.StringValue{
						Value: "test2024050503_int0",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "pendingReqExists",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				requests.Instance = &requests.Singleton{
					ReqDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf").
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
								"4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024072805_int0",
								"another test nic",
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

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL")).
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
							},
						).
							AddRow(
								"60284e9f-69c0-4db8-868d-7a8e24070025",
								createUpdateTime,
								createUpdateTime,
								gorm.DeletedAt{
									Time:  time.Time{},
									Valid: false,
								},
								sql.NullTime{
									Time:  createUpdateTime,
									Valid: true,
								},
								0,
								0,
								"NICCLONE",
								"{\"nic_id\":\"4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf\",\"new_nic_name\":\"test2023102401_int0_clone0\"}",
							),
					)
			},
			args: args{
				cloneReq: &cirrina.VmNicCloneReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf",
					},
					NewVmNicName: &wrapperspb.StringValue{
						Value: "test2024050503_int0",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "reqCreateErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				requests.Instance = &requests.Singleton{
					ReqDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf").
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
								"4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024072805_int0",
								"another test nic",
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

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL")).
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
							},
						),
					)

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `requests` (`created_at`,`updated_at`,`deleted_at`,`started_at`,`successful`,`complete`,`type`,`data`,`id`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`")). //nolint:lll
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, false,
						false, "NICCLONE", "{\"nic_id\":\"4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf\",\"new_nic_name\":\"test2024050503_int0\"}", sqlmock.AnyArg(), //nolint:lll
					).
					WillReturnError(gorm.ErrInvalidData)

				mock.ExpectRollback()
			},
			args: args{
				cloneReq: &cirrina.VmNicCloneReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf",
					},
					NewVmNicName: &wrapperspb.StringValue{
						Value: "test2024050503_int0",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "reqCreateSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}

				requests.Instance = &requests.Singleton{
					ReqDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `vm_nics` WHERE id = ? AND `vm_nics`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf").
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
								"4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2024072805_int0",
								"another test nic",
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

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL")).
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
							},
						),
					)

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `requests` (`created_at`,`updated_at`,`deleted_at`,`started_at`,`successful`,`complete`,`type`,`data`,`id`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`")). //nolint:lll
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, false,
						false, "NICCLONE", "{\"nic_id\":\"4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf\",\"new_nic_name\":\"test2024050503_int0\"}", sqlmock.AnyArg(), //nolint:lll
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow("0874dc01-44ef-4962-b676-ca8824ae5e9f"))
				mock.ExpectCommit()
			},
			args: args{
				cloneReq: &cirrina.VmNicCloneReq{
					Vmnicid: &cirrina.VmNicId{
						Value: "4abdb6d2-a80f-4522-8e87-a4b93cb4ccbf",
					},
					NewVmNicName: &wrapperspb.StringValue{
						Value: "test2024050503_int0",
					},
				},
			},
			want:    &cirrina.RequestID{Value: "0874dc01-44ef-4962-b676-ca8824ae5e9f"},
			wantErr: false,
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

			var got *cirrina.RequestID

			ctx := context.Background()

			got, err = client.CloneVMNic(ctx, testCase.args.cloneReq)
			if (err != nil) != testCase.wantErr {
				t.Errorf("CloneVMNic() error = %v, wantErr %v", err, testCase.wantErr)

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
