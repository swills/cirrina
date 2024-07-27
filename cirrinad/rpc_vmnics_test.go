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
		testCase := testCase
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

			testDB, mock := cirrinadtest.NewMockDB("switchTest")
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
	}

	for _, testCase := range tests {
		testCase := testCase
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
