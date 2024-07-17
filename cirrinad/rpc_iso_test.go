package main

import (
	"context"
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
	"gorm.io/gorm"

	"cirrina/cirrina"
	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/iso"
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
		testCase := testCase
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
