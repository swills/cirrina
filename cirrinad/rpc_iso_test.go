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
	"cirrina/cirrinad/iso"
	"cirrina/cirrinad/vm"
)

func Test_server_GetISOs(t *testing.T) {
	createUpdateTime := time.Now()

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        []string
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{ // prevents parallel testing
					ISODB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE `isos`.`deleted_at` IS NULL",
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
								"path",
								"size",
								"checksum",
							}).
							AddRow(
								"62bad068-bdaa-484f-9190-22d4f340f645",
								createUpdateTime,
								createUpdateTime,
								nil,
								"someRandomTestIsoForRpcTesting.iso",
								"a totally made up iso",
								"/bhyve/isos/someRandomTestIsoForRpcTesting.iso",
								4621281280,
								"326c7a07a393972d3fcd47deaa08e2b932d9298d96e9b4f63a17a2730f93384abc5feb1f511436dc91fcc8b6f56ed25b43dc91d9cdfc700d2655f7e35420d494", //nolint:lll
							),
					)
			},
			want: []string{"62bad068-bdaa-484f-9190-22d4f340f645"},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

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

			var res cirrina.VMInfo_GetISOsClient

			var got []string

			var VMIso *cirrina.ISOID

			res, err = client.GetISOs(context.Background(), &cirrina.ISOsQuery{})
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetISOs() error = %v, wantErr %v", err, testCase.wantErr)
			}

			for {
				VMIso, err = res.Recv()
				if errors.Is(err, io.EOF) {
					break
				}

				got = append(got, VMIso.GetValue())
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
func Test_server_GetISOInfo(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		isoID *cirrina.ISOID
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *cirrina.ISOInfo
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("ed4d2c9a-10c8-4640-9d90-f95e4bc0c4bb").
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
								"ed4d2c9a-10c8-4640-9d90-f95e4bc0c4bb",
								createUpdateTime,
								createUpdateTime,
								nil,
								"florp.iso",
								"narf",
								"/bhyve/isos/florp.iso",
								4621281280,
								"326c7a07a393972d3fcd47deaa08e2b932d9298d96e9b4f63a17a2730f93384abc5feb1f511436dc91fcc8b6f56ed25b43dc91d9cdfc700d2655f7e35420d494", //nolint:lll
							),
					)
			},
			args: args{
				isoID: &cirrina.ISOID{
					Value: "ed4d2c9a-10c8-4640-9d90-f95e4bc0c4bb",
				},
			},
			want: &cirrina.ISOInfo{
				Name:        func() *string { name := "florp.iso"; return &name }(),          //nolint:nlreturn
				Description: func() *string { desc := "narf"; return &desc }(),               //nolint:nlreturn
				Size:        func() *uint64 { var size uint64 = 4621281280; return &size }(), //nolint:nlreturn
			},
		},
		{
			name: "badUuid",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
			},
			args: args{
				isoID: &cirrina.ISOID{
					Value: "ed4d2c9a-10c8-4",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "notFound",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("ed4d2c9a-10c8-4640-9d90-f95e4bc0c4bb").
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
				isoID: &cirrina.ISOID{
					Value: "ed4d2c9a-10c8-4640-9d90-f95e4bc0c4bb",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "emptyName",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("ed4d2c9a-10c8-4640-9d90-f95e4bc0c4bb").
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
								"ed4d2c9a-10c8-4640-9d90-f95e4bc0c4bb",
								createUpdateTime,
								createUpdateTime,
								nil,
								"",
								"narf",
								"/bhyve/isos/florp.iso",
								4621281280,
								"326c7a07a393972d3fcd47deaa08e2b932d9298d96e9b4f63a17a2730f93384abc5feb1f511436dc91fcc8b6f56ed25b43dc91d9cdfc700d2655f7e35420d494", //nolint:lll
							),
					)
			},
			args: args{
				isoID: &cirrina.ISOID{
					Value: "ed4d2c9a-10c8-4640-9d90-f95e4bc0c4bb",
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

			got, err := client.GetISOInfo(context.Background(), testCase.args.isoID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetISOInfo() error = %v, wantErr %v", err, testCase.wantErr)

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
func Test_server_AddISO(t *testing.T) {
	type args struct {
		isoInfo *cirrina.ISOInfo
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *cirrina.ISOID
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE name = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).WithArgs("narf.iso").
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
							},
						),
					)
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `isos` (`created_at`,`updated_at`,`deleted_at`,`description`,`size`,`checksum`,`id`,`name`,`path`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`,`path`", //nolint:lll
					),
				).
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "a very poit iso", 0, "", sqlmock.AnyArg(), "narf.iso", "/narf.iso",
					).
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"name",
								"path",
							}).
							AddRow(
								"cdedb3c6-ba62-466a-bd3e-452b19726aab",
								"narf.iso",
								"/narf.iso",
							),
					)
				mock.ExpectCommit()
			},
			args: args{
				isoInfo: &cirrina.ISOInfo{
					Name:        func() *string { name := "narf.iso"; return &name }(),         //nolint:nlreturn
					Description: func() *string { desc := "a very poit iso"; return &desc }(),  //nolint:nlreturn
					Size:        func() *uint64 { var size uint64 = 69696969; return &size }(), //nolint:nlreturn
				},
			},
			want: &cirrina.ISOID{
				Value: "cdedb3c6-ba62-466a-bd3e-452b19726aab",
			},
			wantErr: false,
		},
		{
			name: "dbErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE name = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).WithArgs("narf.iso").
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
							},
						),
					)
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `isos` (`created_at`,`updated_at`,`deleted_at`,`description`,`size`,`checksum`,`id`,`name`,`path`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`,`path`", //nolint:lll
					),
				).
					WillReturnError(errInvalidRequest)
				mock.ExpectRollback()
			},
			args: args{
				isoInfo: &cirrina.ISOInfo{
					Name:        func() *string { name := "narf.iso"; return &name }(),         //nolint:nlreturn
					Description: func() *string { desc := "a very poit iso"; return &desc }(),  //nolint:nlreturn
					Size:        func() *uint64 { var size uint64 = 69696969; return &size }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "nilDesc",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE name = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).WithArgs("narf.iso").
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
							},
						),
					)
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `isos` (`created_at`,`updated_at`,`deleted_at`,`description`,`size`,`checksum`,`id`,`name`,`path`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`,`path`", //nolint:lll
					),
				).
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "", 0, "", sqlmock.AnyArg(), "narf.iso", "/narf.iso",
					).
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"name",
								"path",
							}).
							AddRow(
								"cdedb3c6-ba62-466a-bd3e-452b19726aab",
								"narf.iso",
								"/narf.iso",
							),
					)
				mock.ExpectCommit()
			},
			args: args{
				isoInfo: &cirrina.ISOInfo{
					Name:        func() *string { name := "narf.iso"; return &name }(), //nolint:nlreturn
					Description: nil,
					Size:        func() *uint64 { var size uint64 = 69696969; return &size }(), //nolint:nlreturn
				},
			},
			want: &cirrina.ISOID{
				Value: "cdedb3c6-ba62-466a-bd3e-452b19726aab",
			},
			wantErr: false,
		},
		{
			name:        "nilReq",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				isoInfo: &cirrina.ISOInfo{
					Name:        nil,
					Description: nil,
					Size:        nil,
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

			got, err := client.AddISO(context.Background(), testCase.args.isoInfo)
			if (err != nil) != testCase.wantErr {
				t.Errorf("AddISO() error = %v, wantErr %v", err, testCase.wantErr)

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
func Test_server_RemoveISO(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		isoID *cirrina.ISOID
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
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).WithArgs("515df28c-c52d-4fa1-b696-f02f10b1ae1b").
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
							},
						).AddRow(
							"515df28c-c52d-4fa1-b696-f02f10b1ae1b",
							createUpdateTime,
							createUpdateTime,
							nil,
							"machaela.iso",
							"a very normally named iso",
							"/some/stupid/iso/path/machaela.iso",
							"123436789123",
							"totalGarbage",
						),
					)

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT vm_id,iso_id,position FROM `vm_isos` WHERE iso_id LIKE ? LIMIT 1"),
				).
					WithArgs("515df28c-c52d-4fa1-b696-f02f10b1ae1b").
					WillReturnRows(sqlmock.NewRows([]string{"vm_id", "iso_id", "position"}))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `isos` WHERE `isos`.`id` = ?"),
				).
					WithArgs("515df28c-c52d-4fa1-b696-f02f10b1ae1b").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args: args{
				isoID: func() *cirrina.ISOID {
					isoID := cirrina.ISOID{Value: "515df28c-c52d-4fa1-b696-f02f10b1ae1b"}

					return &isoID
				}(),
			},
			want: func() *cirrina.ReqBool { r := cirrina.ReqBool{Success: true}; return &r }(), //nolint:nlreturn
		},
		{
			name: "badUUID",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).WithArgs("515df28c-c52d-4fa1-b696-f02f10b1ae1b").
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
							},
						).AddRow(
							"515df28c-c52d-4fa1-b696-f02f10b1ae1b",
							createUpdateTime,
							createUpdateTime,
							nil,
							"machaela.iso",
							"a very normally named iso",
							"/some/stupid/iso/path/machaela.iso",
							"123436789123",
							"totalGarbage",
						),
					)
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `isos` WHERE `isos`.`id` = ?"),
				).
					WithArgs("515df28c-c52d-4fa1-b696-f02f10b1ae1b").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args: args{
				isoID: func() *cirrina.ISOID {
					isoID := cirrina.ISOID{Value: "515df28c-c52d-4fa"}

					return &isoID
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "isoInUse",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}
				vm.List.VMList = map[string]*vm.VM{}

				testVM1 := vm.VM{
					ID: "7563edac-3a68-4950-9dec-ca53dd8c7fca",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 2,
						},
						VMID: "7563edac-3a68-4950-9dec-ca53dd8c7fca",
						CPU:  2,
						Mem:  1024,
					},
					ISOs: []*iso.ISO{
						{
							ID:          "74793f48-a7ae-4895-b62d-440f9652d8df",
							Name:        "makaela.iso",
							Description: "",
							Path:        "",
							Size:        0,
							Checksum:    "",
						},
					},
					Disks: nil,
				}
				vm.List.VMList[testVM1.ID] = &testVM1

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).WithArgs("74793f48-a7ae-4895-b62d-440f9652d8df").
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
							},
						).AddRow(
							"74793f48-a7ae-4895-b62d-440f9652d8df",
							createUpdateTime,
							createUpdateTime,
							nil,
							"makaela.iso",
							"another normally named iso",
							"/some/stupid/iso/path/makaela.iso",
							"12821789123",
							"totalGarbageStuff",
						),
					)
			},
			args: args{
				isoID: func() *cirrina.ISOID {
					isoID := cirrina.ISOID{Value: "74793f48-a7ae-4895-b62d-440f9652d8df"}

					return &isoID
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).WithArgs("515df28c-c52d-4fa1-b696-f02f10b1ae1b").
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
							},
						).AddRow(
							"515df28c-c52d-4fa1-b696-f02f10b1ae1b",
							createUpdateTime,
							createUpdateTime,
							nil,
							"machaela.iso",
							"a very normally named iso",
							"/some/stupid/iso/path/machaela.iso",
							"123436789123",
							"totalGarbage",
						),
					)
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `isos` WHERE `isos`.`id` = ?"),
				).
					WillReturnError(gorm.ErrInvalidData)
			},
			args: args{
				isoID: func() *cirrina.ISOID {
					isoID := cirrina.ISOID{Value: "515df28c-c52d-4fa1-b696-f02f10b1ae1b"}

					return &isoID
				}(),
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

			got, err := client.RemoveISO(context.Background(), testCase.args.isoID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("AddISO() error = %v, wantErr %v", err, testCase.wantErr)

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
func Test_server_UploadIso(t *testing.T) {
	createUpdateTime := time.Now()

	tests := []struct {
		name                   string
		mockClosure            func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		mockStreamSetupReqFunc func(stream cirrina.VMInfo_UploadIsoClient) error
		mockStreamSendReqFunc  func(stream cirrina.VMInfo_UploadIsoClient) error
		wantErr                bool
		wantSetupError         bool
		wantSendError          bool
	}{
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3753c1dd-48f4-49ca-a415-53a9ee9e2a2f").
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
								"3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
								createUpdateTime,
								createUpdateTime,
								nil,
								"narf.iso",
								"some description",
								"/narf.iso",
								1047048192,
								"259e034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3e3b630ba2", //nolint:lll
							),
					)

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `isos` SET `checksum`=?,`description`=?,`name`=?,`path`=?,`size`=?,`updated_at`=? WHERE `isos`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).WithArgs(
					"41da9689eaf006adb0c7a8c7517b8c4e5f5814978cfbfc297c5e3aa25652042ab1fd940aaf42b87bd775d9d1ed81bcca3571828da4b787e4ed4c91d39ae70da5", //nolint:lll
					"some description",
					"narf.iso",
					"/narf.iso",
					128,
					sqlmock.AnyArg(),
					"3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
				).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				osCreateFunc = func(_ string) (*os.File, error) {
					f, _ := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)

					return f, nil
				}
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadIsoClient) error {
				setupReq := &cirrina.ISOImageRequest{
					Data: &cirrina.ISOImageRequest_Isouploadinfo{
						Isouploadinfo: &cirrina.ISOUploadInfo{
							Isoid: &cirrina.ISOID{
								Value: "3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
							},
							Size:      128,
							Sha512Sum: "41da9689eaf006adb0c7a8c7517b8c4e5f5814978cfbfc297c5e3aa25652042ab1fd940aaf42b87bd775d9d1ed81bcca3571828da4b787e4ed4c91d39ae70da5", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadIsoClient) error {
				dataReq := &cirrina.ISOImageRequest{
					Data: &cirrina.ISOImageRequest_Image{
						Image: []byte{
							0xc3, 0x41, 0xa5, 0x28, 0x6c, 0xc6, 0x05, 0xc1, 0x01, 0x0f, 0xff, 0x30, 0x9e, 0x94, 0x19, 0x21,
							0x73, 0xca, 0x80, 0x81, 0xbb, 0xe7, 0x7d, 0xe6, 0xe2, 0xc3, 0x69, 0xbd, 0xa5, 0xf6, 0x95, 0x28,
							0x9f, 0x98, 0x78, 0xa4, 0x82, 0x2e, 0x18, 0xa0, 0xb2, 0xde, 0xbd, 0x86, 0x2c, 0xfa, 0xb9, 0xc3,
							0xe4, 0xfe, 0x0b, 0x78, 0x27, 0x19, 0x92, 0xe2, 0xf5, 0x1f, 0xea, 0xc1, 0x0a, 0x0c, 0x7d, 0x86,
							0x50, 0x6f, 0xa4, 0x87, 0xda, 0x3d, 0xc6, 0xc1, 0xa0, 0xba, 0x90, 0xe4, 0xec, 0x44, 0x17, 0x79,
							0x1f, 0x04, 0xc4, 0x04, 0x67, 0x55, 0xae, 0x2d, 0xd3, 0x33, 0x80, 0xf2, 0x11, 0x59, 0xf2, 0x6a,
							0x7b, 0xb5, 0xdf, 0xd2, 0xf8, 0xb6, 0x8a, 0xfb, 0xf8, 0x6f, 0x22, 0x6e, 0xdd, 0x09, 0xda, 0x36,
							0xed, 0xae, 0x51, 0x6c, 0xde, 0x2b, 0x58, 0x68, 0x3c, 0x16, 0x2b, 0x99, 0x36, 0x97, 0xa3, 0x25,
						},
					},
				}

				return stream.Send(dataReq)
			},
		},
		{
			name: "badReq",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3753c1dd-48f4-49ca-a415-53a9ee9e2a2f").
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
								"3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
								createUpdateTime,
								createUpdateTime,
								nil,
								"narf.iso",
								"some description",
								"/narf.iso",
								1047048192,
								"259e034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3e3b630ba2", //nolint:lll
							),
					)

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `isos` SET `checksum`=?,`description`=?,`name`=?,`path`=?,`size`=?,`updated_at`=? WHERE `isos`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).WithArgs(
					"41da9689eaf006adb0c7a8c7517b8c4e5f5814978cfbfc297c5e3aa25652042ab1fd940aaf42b87bd775d9d1ed81bcca3571828da4b787e4ed4c91d39ae70da5", //nolint:lll
					"some description",
					"narf.iso",
					"/narf.iso",
					128,
					sqlmock.AnyArg(),
					"3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
				).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				osCreateFunc = func(_ string) (*os.File, error) {
					f, _ := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)

					return f, nil
				}
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadIsoClient) error {
				setupReq := &cirrina.ISOImageRequest{
					Data: &cirrina.ISOImageRequest_Isouploadinfo{
						Isouploadinfo: nil,
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadIsoClient) error {
				dataReq := &cirrina.ISOImageRequest{
					Data: &cirrina.ISOImageRequest_Image{
						Image: []byte{
							0xc3, 0x41, 0xa5, 0x28, 0x6c, 0xc6, 0x05, 0xc1, 0x01, 0x0f, 0xff, 0x30, 0x9e, 0x94, 0x19, 0x21,
							0x73, 0xca, 0x80, 0x81, 0xbb, 0xe7, 0x7d, 0xe6, 0xe2, 0xc3, 0x69, 0xbd, 0xa5, 0xf6, 0x95, 0x28,
							0x9f, 0x98, 0x78, 0xa4, 0x82, 0x2e, 0x18, 0xa0, 0xb2, 0xde, 0xbd, 0x86, 0x2c, 0xfa, 0xb9, 0xc3,
							0xe4, 0xfe, 0x0b, 0x78, 0x27, 0x19, 0x92, 0xe2, 0xf5, 0x1f, 0xea, 0xc1, 0x0a, 0x0c, 0x7d, 0x86,
							0x50, 0x6f, 0xa4, 0x87, 0xda, 0x3d, 0xc6, 0xc1, 0xa0, 0xba, 0x90, 0xe4, 0xec, 0x44, 0x17, 0x79,
							0x1f, 0x04, 0xc4, 0x04, 0x67, 0x55, 0xae, 0x2d, 0xd3, 0x33, 0x80, 0xf2, 0x11, 0x59, 0xf2, 0x6a,
							0x7b, 0xb5, 0xdf, 0xd2, 0xf8, 0xb6, 0x8a, 0xfb, 0xf8, 0x6f, 0x22, 0x6e, 0xdd, 0x09, 0xda, 0x36,
							0xed, 0xae, 0x51, 0x6c, 0xde, 0x2b, 0x58, 0x68, 0x3c, 0x16, 0x2b, 0x99, 0x36, 0x97, 0xa3, 0x25,
						},
					},
				}

				return stream.Send(dataReq)
			},
			wantSetupError: true,
		},
		{
			name: "badUUID",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadIsoClient) error {
				setupReq := &cirrina.ISOImageRequest{
					Data: &cirrina.ISOImageRequest_Isouploadinfo{
						Isouploadinfo: &cirrina.ISOUploadInfo{
							Isoid: &cirrina.ISOID{
								Value: "3753c1dd-48f4-49",
							},
							Size:      128,
							Sha512Sum: "41da9689eaf006adb0c7a8c7517b8c4e5f5814978cfbfc297c5e3aa25652042ab1fd940aaf42b87bd775d9d1ed81bcca3571828da4b787e4ed4c91d39ae70da5", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(_ cirrina.VMInfo_UploadIsoClient) error {
				return nil
			},
			wantSetupError: true,
		},
		{
			name: "isoNotFound",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3753c1dd-48f4-49ca-a415-53a9ee9e2a2f").
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
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadIsoClient) error {
				setupReq := &cirrina.ISOImageRequest{
					Data: &cirrina.ISOImageRequest_Isouploadinfo{
						Isouploadinfo: &cirrina.ISOUploadInfo{
							Isoid: &cirrina.ISOID{
								Value: "3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
							},
							Size:      128,
							Sha512Sum: "41da9689eaf006adb0c7a8c7517b8c4e5f5814978cfbfc297c5e3aa25652042ab1fd940aaf42b87bd775d9d1ed81bcca3571828da4b787e4ed4c91d39ae70da5", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(_ cirrina.VMInfo_UploadIsoClient) error {
				return nil
			},
			wantSetupError: true,
		},
		{
			name: "isoBlankName",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3753c1dd-48f4-49ca-a415-53a9ee9e2a2f").
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
								"3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
								createUpdateTime,
								createUpdateTime,
								nil,
								"",
								"some description",
								"/narf.iso",
								1047048192,
								"259e034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3e3b630ba2", //nolint:lll
							),
					)
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadIsoClient) error {
				setupReq := &cirrina.ISOImageRequest{
					Data: &cirrina.ISOImageRequest_Isouploadinfo{
						Isouploadinfo: &cirrina.ISOUploadInfo{
							Isoid: &cirrina.ISOID{
								Value: "3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
							},
							Size:      128,
							Sha512Sum: "41da9689eaf006adb0c7a8c7517b8c4e5f5814978cfbfc297c5e3aa25652042ab1fd940aaf42b87bd775d9d1ed81bcca3571828da4b787e4ed4c91d39ae70da5", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(_ cirrina.VMInfo_UploadIsoClient) error {
				return nil
			},
			wantSetupError: true,
		},
		{
			name: "osCreateFileFail",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3753c1dd-48f4-49ca-a415-53a9ee9e2a2f").
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
								"3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
								createUpdateTime,
								createUpdateTime,
								nil,
								"narf.iso",
								"some description",
								"/narf.iso",
								1047048192,
								"259e034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3e3b630ba2", //nolint:lll
							),
					)

				osCreateFunc = func(_ string) (*os.File, error) {
					return nil, errors.New("bogus create error") //nolint:goerr113
				}
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadIsoClient) error {
				setupReq := &cirrina.ISOImageRequest{
					Data: &cirrina.ISOImageRequest_Isouploadinfo{
						Isouploadinfo: &cirrina.ISOUploadInfo{
							Isoid: &cirrina.ISOID{
								Value: "3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
							},
							Size:      128,
							Sha512Sum: "41da9689eaf006adb0c7a8c7517b8c4e5f5814978cfbfc297c5e3aa25652042ab1fd940aaf42b87bd775d9d1ed81bcca3571828da4b787e4ed4c91d39ae70da5", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadIsoClient) error {
				dataReq := &cirrina.ISOImageRequest{
					Data: &cirrina.ISOImageRequest_Image{
						Image: []byte{
							0xc3, 0x41, 0xa5, 0x28, 0x6c, 0xc6, 0x05, 0xc1, 0x01, 0x0f, 0xff, 0x30, 0x9e, 0x94, 0x19, 0x21,
							0x73, 0xca, 0x80, 0x81, 0xbb, 0xe7, 0x7d, 0xe6, 0xe2, 0xc3, 0x69, 0xbd, 0xa5, 0xf6, 0x95, 0x28,
							0x9f, 0x98, 0x78, 0xa4, 0x82, 0x2e, 0x18, 0xa0, 0xb2, 0xde, 0xbd, 0x86, 0x2c, 0xfa, 0xb9, 0xc3,
							0xe4, 0xfe, 0x0b, 0x78, 0x27, 0x19, 0x92, 0xe2, 0xf5, 0x1f, 0xea, 0xc1, 0x0a, 0x0c, 0x7d, 0x86,
							0x50, 0x6f, 0xa4, 0x87, 0xda, 0x3d, 0xc6, 0xc1, 0xa0, 0xba, 0x90, 0xe4, 0xec, 0x44, 0x17, 0x79,
							0x1f, 0x04, 0xc4, 0x04, 0x67, 0x55, 0xae, 0x2d, 0xd3, 0x33, 0x80, 0xf2, 0x11, 0x59, 0xf2, 0x6a,
							0x7b, 0xb5, 0xdf, 0xd2, 0xf8, 0xb6, 0x8a, 0xfb, 0xf8, 0x6f, 0x22, 0x6e, 0xdd, 0x09, 0xda, 0x36,
							0xed, 0xae, 0x51, 0x6c, 0xde, 0x2b, 0x58, 0x68, 0x3c, 0x16, 0x2b, 0x99, 0x36, 0x97, 0xa3, 0x25,
						},
					},
				}

				return stream.Send(dataReq)
			},
			wantSetupError: true,
		},
		{
			name: "sizeTooSmall",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3753c1dd-48f4-49ca-a415-53a9ee9e2a2f").
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
								"3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
								createUpdateTime,
								createUpdateTime,
								nil,
								"narf.iso",
								"some description",
								"/narf.iso",
								1047048192,
								"259e034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3e3b630ba2", //nolint:lll
							),
					)

				osCreateFunc = func(_ string) (*os.File, error) {
					f, _ := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)

					return f, nil
				}
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadIsoClient) error {
				setupReq := &cirrina.ISOImageRequest{
					Data: &cirrina.ISOImageRequest_Isouploadinfo{
						Isouploadinfo: &cirrina.ISOUploadInfo{
							Isoid: &cirrina.ISOID{
								Value: "3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
							},
							Size:      64,
							Sha512Sum: "41da9689eaf006adb0c7a8c7517b8c4e5f5814978cfbfc297c5e3aa25652042ab1fd940aaf42b87bd775d9d1ed81bcca3571828da4b787e4ed4c91d39ae70da5", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadIsoClient) error {
				dataReq := &cirrina.ISOImageRequest{
					Data: &cirrina.ISOImageRequest_Image{
						Image: []byte{
							0xc3, 0x41, 0xa5, 0x28, 0x6c, 0xc6, 0x05, 0xc1, 0x01, 0x0f, 0xff, 0x30, 0x9e, 0x94, 0x19, 0x21,
							0x73, 0xca, 0x80, 0x81, 0xbb, 0xe7, 0x7d, 0xe6, 0xe2, 0xc3, 0x69, 0xbd, 0xa5, 0xf6, 0x95, 0x28,
							0x9f, 0x98, 0x78, 0xa4, 0x82, 0x2e, 0x18, 0xa0, 0xb2, 0xde, 0xbd, 0x86, 0x2c, 0xfa, 0xb9, 0xc3,
							0xe4, 0xfe, 0x0b, 0x78, 0x27, 0x19, 0x92, 0xe2, 0xf5, 0x1f, 0xea, 0xc1, 0x0a, 0x0c, 0x7d, 0x86,
							0x50, 0x6f, 0xa4, 0x87, 0xda, 0x3d, 0xc6, 0xc1, 0xa0, 0xba, 0x90, 0xe4, 0xec, 0x44, 0x17, 0x79,
							0x1f, 0x04, 0xc4, 0x04, 0x67, 0x55, 0xae, 0x2d, 0xd3, 0x33, 0x80, 0xf2, 0x11, 0x59, 0xf2, 0x6a,
							0x7b, 0xb5, 0xdf, 0xd2, 0xf8, 0xb6, 0x8a, 0xfb, 0xf8, 0x6f, 0x22, 0x6e, 0xdd, 0x09, 0xda, 0x36,
							0xed, 0xae, 0x51, 0x6c, 0xde, 0x2b, 0x58, 0x68, 0x3c, 0x16, 0x2b, 0x99, 0x36, 0x97, 0xa3, 0x25,
						},
					},
				}
				_ = stream.Send(dataReq)

				return stream.CloseSend()
			},
			wantSendError: true,
		},
		{
			name: "sizeTooLarge",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3753c1dd-48f4-49ca-a415-53a9ee9e2a2f").
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
								"3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
								createUpdateTime,
								createUpdateTime,
								nil,
								"narf.iso",
								"some description",
								"/narf.iso",
								1047048192,
								"259e034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3e3b630ba2", //nolint:lll
							),
					)

				osCreateFunc = func(_ string) (*os.File, error) {
					f, _ := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)

					return f, nil
				}
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadIsoClient) error {
				setupReq := &cirrina.ISOImageRequest{
					Data: &cirrina.ISOImageRequest_Isouploadinfo{
						Isouploadinfo: &cirrina.ISOUploadInfo{
							Isoid: &cirrina.ISOID{
								Value: "3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
							},
							Size:      256,
							Sha512Sum: "41da9689eaf006adb0c7a8c7517b8c4e5f5814978cfbfc297c5e3aa25652042ab1fd940aaf42b87bd775d9d1ed81bcca3571828da4b787e4ed4c91d39ae70da5", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadIsoClient) error {
				dataReq := &cirrina.ISOImageRequest{
					Data: &cirrina.ISOImageRequest_Image{
						Image: []byte{
							0xc3, 0x41, 0xa5, 0x28, 0x6c, 0xc6, 0x05, 0xc1, 0x01, 0x0f, 0xff, 0x30, 0x9e, 0x94, 0x19, 0x21,
							0x73, 0xca, 0x80, 0x81, 0xbb, 0xe7, 0x7d, 0xe6, 0xe2, 0xc3, 0x69, 0xbd, 0xa5, 0xf6, 0x95, 0x28,
							0x9f, 0x98, 0x78, 0xa4, 0x82, 0x2e, 0x18, 0xa0, 0xb2, 0xde, 0xbd, 0x86, 0x2c, 0xfa, 0xb9, 0xc3,
							0xe4, 0xfe, 0x0b, 0x78, 0x27, 0x19, 0x92, 0xe2, 0xf5, 0x1f, 0xea, 0xc1, 0x0a, 0x0c, 0x7d, 0x86,
							0x50, 0x6f, 0xa4, 0x87, 0xda, 0x3d, 0xc6, 0xc1, 0xa0, 0xba, 0x90, 0xe4, 0xec, 0x44, 0x17, 0x79,
							0x1f, 0x04, 0xc4, 0x04, 0x67, 0x55, 0xae, 0x2d, 0xd3, 0x33, 0x80, 0xf2, 0x11, 0x59, 0xf2, 0x6a,
							0x7b, 0xb5, 0xdf, 0xd2, 0xf8, 0xb6, 0x8a, 0xfb, 0xf8, 0x6f, 0x22, 0x6e, 0xdd, 0x09, 0xda, 0x36,
							0xed, 0xae, 0x51, 0x6c, 0xde, 0x2b, 0x58, 0x68, 0x3c, 0x16, 0x2b, 0x99, 0x36, 0x97, 0xa3, 0x25,
						},
					},
				}
				_ = stream.Send(dataReq)

				return stream.CloseSend()
			},
			wantSendError: true,
		},
		{
			name: "dbErr",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3753c1dd-48f4-49ca-a415-53a9ee9e2a2f").
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
								"3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
								createUpdateTime,
								createUpdateTime,
								nil,
								"narf.iso",
								"some description",
								"/narf.iso",
								1047048192,
								"259e034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3e3b630ba2", //nolint:lll
							),
					)

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `isos` SET `checksum`=?,`description`=?,`name`=?,`path`=?,`size`=?,`updated_at`=? WHERE `isos`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).WithArgs(
					"41da9689eaf006adb0c7a8c7517b8c4e5f5814978cfbfc297c5e3aa25652042ab1fd940aaf42b87bd775d9d1ed81bcca3571828da4b787e4ed4c91d39ae70da5", //nolint:lll
					"some description",
					"narf.iso",
					"/narf.iso",
					128,
					sqlmock.AnyArg(),
					"3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
				).
					WillReturnError(errInvalidRequest)

				osCreateFunc = func(_ string) (*os.File, error) {
					f, _ := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)

					return f, nil
				}
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadIsoClient) error {
				setupReq := &cirrina.ISOImageRequest{
					Data: &cirrina.ISOImageRequest_Isouploadinfo{
						Isouploadinfo: &cirrina.ISOUploadInfo{
							Isoid: &cirrina.ISOID{
								Value: "3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
							},
							Size:      128,
							Sha512Sum: "41da9689eaf006adb0c7a8c7517b8c4e5f5814978cfbfc297c5e3aa25652042ab1fd940aaf42b87bd775d9d1ed81bcca3571828da4b787e4ed4c91d39ae70da5", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadIsoClient) error {
				dataReq := &cirrina.ISOImageRequest{
					Data: &cirrina.ISOImageRequest_Image{
						Image: []byte{
							0xc3, 0x41, 0xa5, 0x28, 0x6c, 0xc6, 0x05, 0xc1, 0x01, 0x0f, 0xff, 0x30, 0x9e, 0x94, 0x19, 0x21,
							0x73, 0xca, 0x80, 0x81, 0xbb, 0xe7, 0x7d, 0xe6, 0xe2, 0xc3, 0x69, 0xbd, 0xa5, 0xf6, 0x95, 0x28,
							0x9f, 0x98, 0x78, 0xa4, 0x82, 0x2e, 0x18, 0xa0, 0xb2, 0xde, 0xbd, 0x86, 0x2c, 0xfa, 0xb9, 0xc3,
							0xe4, 0xfe, 0x0b, 0x78, 0x27, 0x19, 0x92, 0xe2, 0xf5, 0x1f, 0xea, 0xc1, 0x0a, 0x0c, 0x7d, 0x86,
							0x50, 0x6f, 0xa4, 0x87, 0xda, 0x3d, 0xc6, 0xc1, 0xa0, 0xba, 0x90, 0xe4, 0xec, 0x44, 0x17, 0x79,
							0x1f, 0x04, 0xc4, 0x04, 0x67, 0x55, 0xae, 0x2d, 0xd3, 0x33, 0x80, 0xf2, 0x11, 0x59, 0xf2, 0x6a,
							0x7b, 0xb5, 0xdf, 0xd2, 0xf8, 0xb6, 0x8a, 0xfb, 0xf8, 0x6f, 0x22, 0x6e, 0xdd, 0x09, 0xda, 0x36,
							0xed, 0xae, 0x51, 0x6c, 0xde, 0x2b, 0x58, 0x68, 0x3c, 0x16, 0x2b, 0x99, 0x36, 0x97, 0xa3, 0x25,
						},
					},
				}

				return stream.Send(dataReq)
			},
			wantErr: true,
		},
		{
			name: "badChecksum",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				iso.Instance = &iso.Singleton{
					ISODB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("3753c1dd-48f4-49ca-a415-53a9ee9e2a2f").
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
								"3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
								createUpdateTime,
								createUpdateTime,
								nil,
								"narf.iso",
								"some description",
								"/narf.iso",
								1047048192,
								"259e034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3e3b630ba2", //nolint:lll
							),
					)

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `isos` SET `checksum`=?,`description`=?,`name`=?,`path`=?,`size`=?,`updated_at`=? WHERE `isos`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).WithArgs(
					"41da9689eaf006adb0c7a8c7517b8c4e5f5814978cfbfc297c5e3aa25652042ab1fd940aaf42b87bd775d9d1ed81bcca3571828da4b787e4ed4c91d39ae70da5", //nolint:lll
					"some description",
					"narf.iso",
					"/narf.iso",
					128,
					sqlmock.AnyArg(),
					"3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
				).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				osCreateFunc = func(_ string) (*os.File, error) {
					f, _ := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)

					return f, nil
				}
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadIsoClient) error {
				setupReq := &cirrina.ISOImageRequest{
					Data: &cirrina.ISOImageRequest_Isouploadinfo{
						Isouploadinfo: &cirrina.ISOUploadInfo{
							Isoid: &cirrina.ISOID{
								Value: "3753c1dd-48f4-49ca-a415-53a9ee9e2a2f",
							},
							Size:      128,
							Sha512Sum: "31da9689eaf006adb0c7a8c7517b8c4e5f5814978cfbfc297c5e3aa25652042ab1fd940aaf42b87bd775d9d1ed81bcca3571828da4b787e4ed4c91d39ae70da5", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadIsoClient) error {
				dataReq := &cirrina.ISOImageRequest{
					Data: &cirrina.ISOImageRequest_Image{
						Image: []byte{
							0xc3, 0x41, 0xa5, 0x28, 0x6c, 0xc6, 0x05, 0xc1, 0x01, 0x0f, 0xff, 0x30, 0x9e, 0x94, 0x19, 0x21,
							0x73, 0xca, 0x80, 0x81, 0xbb, 0xe7, 0x7d, 0xe6, 0xe2, 0xc3, 0x69, 0xbd, 0xa5, 0xf6, 0x95, 0x28,
							0x9f, 0x98, 0x78, 0xa4, 0x82, 0x2e, 0x18, 0xa0, 0xb2, 0xde, 0xbd, 0x86, 0x2c, 0xfa, 0xb9, 0xc3,
							0xe4, 0xfe, 0x0b, 0x78, 0x27, 0x19, 0x92, 0xe2, 0xf5, 0x1f, 0xea, 0xc1, 0x0a, 0x0c, 0x7d, 0x86,
							0x50, 0x6f, 0xa4, 0x87, 0xda, 0x3d, 0xc6, 0xc1, 0xa0, 0xba, 0x90, 0xe4, 0xec, 0x44, 0x17, 0x79,
							0x1f, 0x04, 0xc4, 0x04, 0x67, 0x55, 0xae, 0x2d, 0xd3, 0x33, 0x80, 0xf2, 0x11, 0x59, 0xf2, 0x6a,
							0x7b, 0xb5, 0xdf, 0xd2, 0xf8, 0xb6, 0x8a, 0xfb, 0xf8, 0x6f, 0x22, 0x6e, 0xdd, 0x09, 0xda, 0x36,
							0xed, 0xae, 0x51, 0x6c, 0xde, 0x2b, 0x58, 0x68, 0x3c, 0x16, 0x2b, 0x99, 0x36, 0x97, 0xa3, 0x25,
						},
					},
				}

				return stream.Send(dataReq)
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(t.Name())

			testCase.mockClosure(testDB, mock)

			lis := bufconn.Listen(1024 * 1024)

			testServer := grpc.NewServer()
			reflection.Register(testServer)
			cirrina.RegisterVMInfoServer(testServer, &server{})

			go func() {
				if err := testServer.Serve(lis); err != nil {
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

			stream, _ := client.UploadIso(context.Background())

			_ = testCase.mockStreamSetupReqFunc(stream)

			if testCase.wantSetupError {
				var rb cirrina.ReqBool

				_ = stream.RecvMsg(&rb)

				if rb.GetSuccess() {
					t.Errorf("UploadIso() err = %v, wantSetupErr %v", err, testCase.wantSetupError)
				}

				return
			}

			_ = testCase.mockStreamSendReqFunc(stream)

			if testCase.wantSendError {
				var rb cirrina.ReqBool

				_ = stream.RecvMsg(&rb)

				if rb.GetSuccess() {
					t.Errorf("UploadIso() err = %v, wantSendError %v", err, testCase.wantSendError)
				}

				return
			}

			reply, _ := stream.CloseAndRecv()

			if !reply.GetSuccess() && !testCase.wantErr {
				t.Errorf("UploadIso() success = %v, wantErr %v", reply.GetSuccess(), testCase.wantErr)
			}
		})
	}
}
