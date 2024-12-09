package main

import (
	"context"
	"database/sql"
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
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/gorm"

	"cirrina/cirrina"
	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/vm"
)

//nolint:paralleltest,maintidx,gocognit
func Test_server_AddDisk(t *testing.T) {
	createUpdateTime := time.Now()
	diskDevTypeFile := cirrina.DiskDevType_FILE
	diskTypeNVME := cirrina.DiskType_NVME
	// diskDevTypeZVol := cirrina.DiskDevType_ZVOL

	type args struct {
		diskInfo *cirrina.DiskInfo
	}

	tests := []struct {
		name          string
		mockClosure   func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args          args
		want          *cirrina.DiskId
		wantExists    bool
		wantExistsErr bool
		wantCreateErr bool
		wantErr       bool
	}{
		{
			name: "nilName",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
			},
			args: args{
				diskInfo: &cirrina.DiskInfo{
					Name:        nil,
					Description: nil,
					Size:        nil,
					DiskType:    nil,
					Usage:       nil,
					SizeNum:     nil,
					UsageNum:    nil,
					DiskDevType: nil,
					Cache:       nil,
					Direct:      nil,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "nilSize",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				disk.Instance = &disk.Singleton{ // prevents parallel testing
					DiskDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `disks` WHERE `disks`.`deleted_at` IS NULL"),
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
								"dev_type",
								"disk_cache",
								"disk_direct",
							}).
							AddRow(
								"20d3098f-7ccf-484e-bed4-757940a3c775",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2023061001_14",
								"a virtual hard disk image",
								"NVME",
								"FILE",
								1,
								0,
							),
					)
			},
			args: args{
				diskInfo: &cirrina.DiskInfo{
					Name:        func() *string { name := "someDisk"; return &name }(), //nolint:nlreturn
					Description: nil,
					Size:        nil,
					DiskType:    &diskTypeNVME,
					Usage:       nil,
					SizeNum:     nil,
					UsageNum:    nil,
					DiskDevType: nil,
					Cache:       nil,
					Direct:      nil,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "nilType",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				disk.Instance = &disk.Singleton{ // prevents parallel testing
					DiskDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `disks` WHERE `disks`.`deleted_at` IS NULL"),
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
								"dev_type",
								"disk_cache",
								"disk_direct",
							}).
							AddRow(
								"20d3098f-7ccf-484e-bed4-757940a3c775",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2023061001_14",
								"a virtual hard disk image",
								"NVME",
								"FILE",
								1,
								0,
							),
					)

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta("INSERT INTO `disks` (`created_at`,`updated_at`,`deleted_at`,`description`,`type`,`dev_type`,`disk_cache`,`disk_direct`,`id`,`name`) VALUES (?,?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`")). //nolint:lll
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil,
						"", "NVME", "FILE", true, false, sqlmock.AnyArg(), "someDisk",
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow("c916ca6e-eb6b-400c-86ec-824b84ae71d3"))
				mock.ExpectCommit()
			},
			args: args{
				diskInfo: &cirrina.DiskInfo{
					Name:        func() *string { name := "someDisk"; return &name }(), //nolint:nlreturn
					Description: nil,
					Size:        nil,
					DiskType:    nil,
					Usage:       nil,
					SizeNum:     nil,
					UsageNum:    nil,
					DiskDevType: &diskDevTypeFile,
					Cache:       nil,
					Direct:      nil,
				},
			},
			want: &cirrina.DiskId{
				Value: "c916ca6e-eb6b-400c-86ec-824b84ae71d3",
			},
			wantErr: false,
		},
		{
			name: "successFile",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				disk.Instance = &disk.Singleton{ // prevents parallel testing
					DiskDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `disks` WHERE `disks`.`deleted_at` IS NULL"),
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
								"dev_type",
								"disk_cache",
								"disk_direct",
							}).
							AddRow(
								"20d3098f-7ccf-484e-bed4-757940a3c775",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2023061001_14",
								"a virtual hard disk image",
								"NVME",
								"FILE",
								1,
								0,
							),
					)

				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta("INSERT INTO `disks` (`created_at`,`updated_at`,`deleted_at`,`description`,`type`,`dev_type`,`disk_cache`,`disk_direct`,`id`,`name`) VALUES (?,?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`")). //nolint:lll
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil,
						"", "NVME", "FILE", true, false, sqlmock.AnyArg(), "someDisk2",
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow("c916ca6e-eb6b-400c-86ec-824b84ae71d3"))
				mock.ExpectCommit()
			},
			args: args{
				diskInfo: &cirrina.DiskInfo{
					Name:        func() *string { name := "someDisk2"; return &name }(), //nolint:nlreturn
					Description: nil,
					Size:        nil,
					DiskType:    &diskTypeNVME,
					Usage:       nil,
					SizeNum:     nil,
					UsageNum:    nil,
					DiskDevType: &diskDevTypeFile,
					Cache:       nil,
					Direct:      nil,
				},
			},
			want: &cirrina.DiskId{
				Value: "c916ca6e-eb6b-400c-86ec-824b84ae71d3",
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mockDB := cirrinadtest.NewMockDB(t.Name())
			config.Config.Disk.Default.Size = "1G"

			ctrl := gomock.NewController(t)

			fileMock := disk.NewMockFileInfoFetcher(ctrl)
			disk.FileInfoFetcherImpl = fileMock

			t.Cleanup(func() { disk.FileInfoFetcherImpl = disk.FileInfoCmds{} })

			zfsMock := disk.NewMockZfsVolInfoFetcher(ctrl)
			disk.ZfsInfoFetcherImpl = zfsMock

			t.Cleanup(func() { disk.ZfsInfoFetcherImpl = disk.ZfsVolInfoCmds{} })

			testCase.mockClosure(testDB, mockDB)

			var existsErr error

			var createErr error

			if testCase.wantExistsErr {
				existsErr = errors.New("bogus exists error") //nolint:goerr113
			}

			if testCase.wantCreateErr {
				createErr = errors.New("bogus create error") //nolint:goerr113
			}

			// file is default type
			if testCase.args.diskInfo.GetDiskDevType() == cirrina.DiskDevType_FILE {
				fileMock.EXPECT().CheckExists(gomock.Any()).MaxTimes(1).Return(testCase.wantExists, existsErr)
				fileMock.EXPECT().Add(gomock.Any(), gomock.Any()).MaxTimes(1).Return(createErr)
			}

			if testCase.args.diskInfo.GetDiskDevType() == cirrina.DiskDevType_ZVOL {
				zfsMock.EXPECT().CheckExists(gomock.Any()).MaxTimes(1).Return(testCase.wantExists, existsErr)
				zfsMock.EXPECT().Add(gomock.Any(), gomock.Any()).MaxTimes(1).Return(createErr)
			}

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

			got, err := client.AddDisk(context.Background(), testCase.args.diskInfo)
			if (err != nil) != testCase.wantErr {
				t.Errorf("AddDisk() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			mockDB.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mockDB.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

//nolint:paralleltest,maintidx
func Test_server_GetDiskInfo(t *testing.T) {
	type args struct {
		diskID *cirrina.DiskId
	}

	tests := []struct {
		name           string
		mockClosure    func()
		args           args
		want           *cirrina.DiskInfo
		wantErr        bool
		wantExists     bool
		wantExistsErr  bool
		wantSize       uint64
		wantSizeErr    bool
		wantZvolConfig bool
	}{
		{
			name:        "invalidRequest",
			mockClosure: func() {},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "94e8241c-2aff-473",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "diskNotFound",
			mockClosure: func() {},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "94e8241c-2aff-4768-9975-9f67d2467ad3",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalidDiskCacheInCache",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "94e8241c-2aff-4768-9975-9f67d2467ad3",
					Name:        "aDisk",
					Description: "a description",
					Type:        "garbage",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: false,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "94e8241c-2aff-4768-9975-9f67d2467ad3",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalidDiskDirectInCache",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "94e8241c-2aff-4768-9975-9f67d2467ad3",
					Name:        "aDisk",
					Description: "a description",
					Type:        "garbage",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: false,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "94e8241c-2aff-4768-9975-9f67d2467ad3",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalidDiskTypeInCache",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "94e8241c-2aff-4768-9975-9f67d2467ad3",
					Name:        "aDisk",
					Description: "a description",
					Type:        "garbage",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "94e8241c-2aff-4768-9975-9f67d2467ad3",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalidDiskDevTypeInCache",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "94e8241c-2aff-4768-9975-9f67d2467ad3",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "garbage",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "94e8241c-2aff-4768-9975-9f67d2467ad3",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "diskDoesNotExistInSystemFile",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "94e8241c-2aff-4768-9975-9f67d2467ad3",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "94e8241c-2aff-4768-9975-9f67d2467ad3",
				},
			},
			want:        nil,
			wantErr:     true,
			wantSizeErr: true,
		},
		{
			name: "diskDoesNotExistInSystemZVOLUnconfig",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "94e8241c-2aff-4768-9975-9f67d2467ad3",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "ZVOL",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "94e8241c-2aff-4768-9975-9f67d2467ad3",
				},
			},
			want:        nil,
			wantErr:     true,
			wantSizeErr: true,
		},
		{
			name: "diskDoesNotExistInSystemZVOL",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "94e8241c-2aff-4768-9975-9f67d2467ad3",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "ZVOL",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "94e8241c-2aff-4768-9975-9f67d2467ad3",
				},
			},
			want:           nil,
			wantErr:        true,
			wantSizeErr:    true,
			wantZvolConfig: true,
		},
		{
			name: "Success",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "94e8241c-2aff-4768-9975-9f67d2467ad3",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "94e8241c-2aff-4768-9975-9f67d2467ad3",
				},
			},
			want: &cirrina.DiskInfo{
				Name:        func() *string { name := "aDisk"; return &name }(),                                             //nolint:nlreturn,lll
				Description: func() *string { desc := "a description"; return &desc }(),                                     //nolint:nlreturn,lll
				Size:        func() *string { size := "712717171"; return &size }(),                                         //nolint:nlreturn,lll
				DiskType:    func() *cirrina.DiskType { diskType := cirrina.DiskType_NVME; return &diskType }(),             //nolint:nlreturn,lll
				DiskDevType: func() *cirrina.DiskDevType { diskDevType := cirrina.DiskDevType_FILE; return &diskDevType }(), //nolint:nlreturn,lll
				Usage:       func() *string { size := "712717171"; return &size }(),                                         //nolint:nlreturn,lll
				SizeNum:     func() *uint64 { var size uint64 = 712717171; return &size }(),                                 //nolint:nlreturn,lll
				UsageNum:    func() *uint64 { var size uint64 = 712717171; return &size }(),                                 //nolint:nlreturn,lll
				Cache:       func() *bool { cache := true; return &cache }(),                                                //nolint:nlreturn,lll
				Direct:      func() *bool { direct := false; return &direct }(),                                             //nolint:nlreturn,lll
			},
			wantExists: true,
			wantSize:   712717171,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			config.Config.Disk.Default.Size = "1G"
			if testCase.wantZvolConfig {
				config.Config.Disk.VM.Path.Zpool = "/some/bogus/path"
			}

			ctrl := gomock.NewController(t)

			fileMock := disk.NewMockFileInfoFetcher(ctrl)
			disk.FileInfoFetcherImpl = fileMock

			t.Cleanup(func() { disk.FileInfoFetcherImpl = disk.FileInfoCmds{} })

			zfsMock := disk.NewMockZfsVolInfoFetcher(ctrl)
			disk.ZfsInfoFetcherImpl = zfsMock

			t.Cleanup(func() { disk.ZfsInfoFetcherImpl = disk.ZfsVolInfoCmds{} })

			testCase.mockClosure()

			var sizeErr error

			if testCase.wantSizeErr {
				sizeErr = errors.New("bogus size error") //nolint:goerr113
			}

			// file is default type
			diskVal := disk.List.DiskList[testCase.args.diskID.GetValue()]
			if diskVal != nil && diskVal.DevType == "FILE" {
				fileMock.EXPECT().FetchFileSize(gomock.Any()).MaxTimes(1).Return(testCase.wantSize, sizeErr)
				fileMock.EXPECT().FetchFileUsage(gomock.Any()).MaxTimes(1).Return(testCase.wantSize, sizeErr)
			}

			if diskVal != nil && diskVal.DevType == "ZVOL" {
				zfsMock.EXPECT().FetchZfsVolumeSize(gomock.Any()).MaxTimes(1).Return(testCase.wantSize, sizeErr)
			}

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

			got, err := client.GetDiskInfo(context.Background(), testCase.args.diskID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetDiskInfo() error = %v, wantErr %v", err, testCase.wantErr)

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
func Test_server_RemoveDisk(t *testing.T) {
	type args struct {
		diskID *cirrina.DiskId
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *cirrina.ReqBool
		wantErr     bool
	}{
		{
			name: "badUUID",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "garbage",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "doesNotExist",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "fb81d8bc-7b66-4172-aec1-d633a5043d2b",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "diskNameEmpty",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "getDiskVMErrorUsedByTwo",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst

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
					ISOs: nil,
					Disks: []*disk.Disk{{
						ID: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					}},
				}
				testVM2 := vm.VM{
					ID: "4a8bae96-632c-48d1-aee7-6c428639004c",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 3,
						},
						VMID: "4a8bae96-632c-48d1-aee7-6c428639004c",
						CPU:  2,
						Mem:  1024,
					},
					ISOs: nil,
					Disks: []*disk.Disk{{
						ID: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					}},
				}
				vm.List.VMList[testVM1.ID] = &testVM1
				vm.List.VMList[testVM2.ID] = &testVM2
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "getDiskVMErrorInUse",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst

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
					ISOs: nil,
					Disks: []*disk.Disk{{
						ID: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					}},
				}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "diskDeleteError",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				disk.Instance = &disk.Singleton{ // prevents parallel testing
					DiskDB: testDB,
				}

				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}

				disk.List.DiskList[diskInst.ID] = diskInst

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `disks` SET `deleted_at`=? WHERE `disks`.`id` = ? AND `disks`.`deleted_at` IS NULL"),
				).
					WithArgs(sqlmock.AnyArg(), "0d4a0338-0b68-4645-b99d-9cbb30df272d").
					WillReturnError(errInvalidRequest)
				mock.ExpectRollback()
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				disk.Instance = &disk.Singleton{ // prevents parallel testing
					DiskDB: testDB,
				}

				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}

				disk.List.DiskList[diskInst.ID] = diskInst

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT vm_id,disk_id,position FROM `vm_disks` WHERE disk_id LIKE ? LIMIT 1"),
				).
					WithArgs("0d4a0338-0b68-4645-b99d-9cbb30df272d").
					WillReturnRows(sqlmock.NewRows([]string{"vm_id", "disk_id", "position"}))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `disks` WHERE `disks`.`id` = ?"),
				).
					WithArgs("0d4a0338-0b68-4645-b99d-9cbb30df272d").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
				},
			},
			want:    &cirrina.ReqBool{Success: true},
			wantErr: false,
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

			got, err := client.RemoveDisk(context.Background(), testCase.args.diskID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("RemoveDisk() error = %v, wantErr %v", err, testCase.wantErr)

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
func Test_server_GetDiskVM(t *testing.T) {
	type args struct {
		diskID *cirrina.DiskId
	}

	tests := []struct {
		name        string
		mockClosure func()
		args        args
		want        *cirrina.VMID
		wantErr     bool
	}{
		{
			name: "badUUID",
			mockClosure: func() {
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "garbage",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "doesNotExist",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "fb81d8bc-7b66-4172-aec1-d633a5043d2b",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "diskNameEmpty",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "getDiskVMErrorUsedByTwo",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst

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
					ISOs: nil,
					Disks: []*disk.Disk{{
						ID: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					}},
				}
				testVM2 := vm.VM{
					ID: "4a8bae96-632c-48d1-aee7-6c428639004c",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 3,
						},
						VMID: "4a8bae96-632c-48d1-aee7-6c428639004c",
						CPU:  2,
						Mem:  1024,
					},
					ISOs: nil,
					Disks: []*disk.Disk{{
						ID: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					}},
				}
				vm.List.VMList[testVM1.ID] = &testVM1
				vm.List.VMList[testVM2.ID] = &testVM2
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Success",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}

				disk.List.DiskList[diskInst.ID] = diskInst
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
					ISOs: nil,
					Disks: []*disk.Disk{{
						ID: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					}},
				}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			args: args{
				diskID: &cirrina.DiskId{
					Value: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
				},
			},
			want: &cirrina.VMID{
				Value: "7563edac-3a68-4950-9dec-ca53dd8c7fca",
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// clear out list(s) from other parallel test runs
			disk.List.DiskList = map[string]*disk.Disk{}
			vm.List.VMList = map[string]*vm.VM{}

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

			got, err := client.GetDiskVM(context.Background(), testCase.args.diskID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("RemoveDisk() error = %v, wantErr %v", err, testCase.wantErr)

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
func Test_server_GetDisks(t *testing.T) {
	tests := []struct {
		name        string
		mockClosure func()
		want        []string
		wantErr     bool
	}{
		{
			name: "Success",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}

				disk.List.DiskList[diskInst.ID] = diskInst
			},
			want: []string{"0d4a0338-0b68-4645-b99d-9cbb30df272d"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// clear out list(s) from other parallel test runs
			disk.List.DiskList = map[string]*disk.Disk{}
			vm.List.VMList = map[string]*vm.VM{}

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

			var res cirrina.VMInfo_GetDisksClient

			var got []string

			var VMDisk *cirrina.DiskId

			ctx := context.Background()

			res, err = client.GetDisks(ctx, &cirrina.DisksQuery{})

			if (err != nil) != testCase.wantErr {
				t.Errorf("GetDisks() error = %v, wantErr %v", err, testCase.wantErr)
			}

			for {
				VMDisk, err = res.Recv()
				if errors.Is(err, io.EOF) {
					break
				}

				got = append(got, VMDisk.GetValue())
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

//nolint:paralleltest,maintidx
func Test_server_SetDiskInfo(t *testing.T) {
	type args struct {
		diu *cirrina.DiskInfoUpdate
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *cirrina.ReqBool
		wantErr     bool
	}{
		{
			name:        "emptyUUID",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				diu: &cirrina.DiskInfoUpdate{
					Id: "",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "badUUID",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				diu: &cirrina.DiskInfoUpdate{
					Id: "garbage",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "getByIDError",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			args: args{
				diu: &cirrina.DiskInfoUpdate{
					Id: "7746cd3b-8ec9-4b04-b7be-2fd79f580dc4",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "badDiskType",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				diskInst := &disk.Disk{
					ID:          "7746cd3b-8ec9-4b04-b7be-2fd79f580dc4",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: false,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diu: &cirrina.DiskInfoUpdate{
					Id: "7746cd3b-8ec9-4b04-b7be-2fd79f580dc4",
					Description: func() *string {
						desc := "some description"

						return &desc
					}(),
					DiskType: func() *cirrina.DiskType {
						f := cirrina.DiskType(-1)

						return &f
					}(),
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "badDiskDevType",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				diskInst := &disk.Disk{
					ID:          "7746cd3b-8ec9-4b04-b7be-2fd79f580dc4",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: false,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diu: &cirrina.DiskInfoUpdate{
					Id: "7746cd3b-8ec9-4b04-b7be-2fd79f580dc4",
					Description: func() *string {
						desc := "some description"

						return &desc
					}(),
					DiskDevType: func() *cirrina.DiskDevType {
						f := cirrina.DiskDevType(-1)

						return &f
					}(),
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "dbError",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				disk.Instance = &disk.Singleton{ // prevents parallel testing
					DiskDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `disks` SET `description`=?,`dev_type`=?,`disk_cache`=?,`disk_direct`=?,`name`=?,`type`=?,`updated_at`=? WHERE `disks`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("some new description", "FILE", true, false, "yetAnotherDisk", "NVME", sqlmock.AnyArg(), "7746cd3b-8ec9-4b04-b7be-2fd79f580dc4"). //nolint:lll
					WillReturnError(errInvalidRequest)
				mock.ExpectCommit()

				diskInst := &disk.Disk{
					ID:          "7746cd3b-8ec9-4b04-b7be-2fd79f580dc4",
					Name:        "yetAnotherDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: false,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diu: &cirrina.DiskInfoUpdate{
					Id: "7746cd3b-8ec9-4b04-b7be-2fd79f580dc4",
					Description: func() *string {
						desc := "some new description"

						return &desc
					}(),
					DiskType: func() *cirrina.DiskType {
						f := cirrina.DiskType_NVME

						return &f
					}(),
					DiskDevType: func() *cirrina.DiskDevType {
						f := cirrina.DiskDevType_FILE

						return &f
					}(),
					Cache:  func() *bool { f := true; return &f }(),  //nolint:nlreturn
					Direct: func() *bool { f := false; return &f }(), //nolint:nlreturn
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Success",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				disk.Instance = &disk.Singleton{ // prevents parallel testing
					DiskDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `disks` SET `description`=?,`dev_type`=?,`disk_cache`=?,`disk_direct`=?,`name`=?,`type`=?,`updated_at`=? WHERE `disks`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("some new description", "FILE", true, false, "yetAnotherDisk", "NVME", sqlmock.AnyArg(), "7746cd3b-8ec9-4b04-b7be-2fd79f580dc4"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				diskInst := &disk.Disk{
					ID:          "7746cd3b-8ec9-4b04-b7be-2fd79f580dc4",
					Name:        "yetAnotherDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: false,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diu: &cirrina.DiskInfoUpdate{
					Id: "7746cd3b-8ec9-4b04-b7be-2fd79f580dc4",
					Description: func() *string {
						desc := "some new description"

						return &desc
					}(),
					DiskType: func() *cirrina.DiskType {
						f := cirrina.DiskType_NVME

						return &f
					}(),
					DiskDevType: func() *cirrina.DiskDevType {
						f := cirrina.DiskDevType_FILE

						return &f
					}(),
					Cache:  func() *bool { f := true; return &f }(),  //nolint:nlreturn
					Direct: func() *bool { f := false; return &f }(), //nolint:nlreturn
				},
			},
			want:    &cirrina.ReqBool{Success: true},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// clear out list(s) from other parallel test runs
			disk.List.DiskList = map[string]*disk.Disk{}

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

			got, err := client.SetDiskInfo(context.Background(), testCase.args.diu)
			if (err != nil) != testCase.wantErr {
				t.Errorf("SetDiskInfo() error = %v, wantErr %v", err, testCase.wantErr)

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
func Test_server_UploadDisk(t *testing.T) {
	tests := []struct {
		name                   string
		mockClosure            func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		mockStreamSetupReqFunc func(stream cirrina.VMInfo_UploadDiskClient) error
		mockStreamSendReqFunc  func(stream cirrina.VMInfo_UploadDiskClient) error
		wantErr                bool
		wantSetupError         bool
		wantSendError          bool
	}{
		{
			name: "Success",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				osCreateFunc = func(_ string) (*os.File, error) {
					f, _ := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)

					return f, nil
				}

				osOpenFileFunc = func(_ string, _ int, _ os.FileMode) (*os.File, error) {
					return os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)
				}

				diskInst := &disk.Disk{
					ID:          "dd29c150-b1ed-4518-bd49-c09a6c5ed431",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				setupReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Diskuploadinfo{
						Diskuploadinfo: &cirrina.DiskUploadInfo{
							Diskid:    &cirrina.DiskId{Value: "dd29c150-b1ed-4518-bd49-c09a6c5ed431"},
							Size:      128,
							Sha512Sum: "9c5dd1250baddae1c12a54f8782dc8903065aa53408000a72cef0868d2914b6a5285f4c7b3ddb493f758515ba906fafc7491db6157c0d164f028cfdc35b9fe89", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				dataReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Image{
						Image: []byte{
							0x62, 0xf3, 0x4c, 0x65, 0xc4, 0x32, 0x0e, 0x1d, 0xf6, 0x34, 0xb3, 0x5c, 0xaf, 0x48, 0x32, 0x2a,
							0x0b, 0x03, 0xda, 0x72, 0x23, 0x30, 0xcf, 0x4f, 0xb8, 0x10, 0x05, 0x0c, 0x13, 0xc4, 0xf8, 0x28,
							0x91, 0x48, 0xc4, 0x55, 0x63, 0x62, 0xba, 0x5d, 0xdb, 0xa5, 0x1b, 0xd3, 0x7c, 0x5c, 0x76, 0x63,
							0x56, 0x9c, 0x10, 0x68, 0xcc, 0xea, 0x04, 0x79, 0x42, 0x88, 0x9d, 0xcb, 0xa5, 0xbf, 0xf1, 0x2d,
							0x3c, 0xce, 0x99, 0xaa, 0x77, 0xca, 0x84, 0xa6, 0x7c, 0x40, 0xf7, 0x4f, 0xc4, 0xfb, 0xca, 0xe7,
							0x15, 0x79, 0x3e, 0x21, 0x93, 0x70, 0x9a, 0xab, 0xf5, 0xa6, 0x7b, 0x3f, 0x43, 0xb2, 0xd0, 0xac,
							0xb9, 0xd1, 0x63, 0x7d, 0x77, 0xe8, 0x47, 0x6f, 0x46, 0x23, 0x26, 0x87, 0x1a, 0x9c, 0x33, 0x58,
							0xa3, 0x9b, 0x22, 0x48, 0xb6, 0xcd, 0x9b, 0xd3, 0x80, 0x2c, 0x1f, 0x33, 0x8b, 0x31, 0x0d, 0x82,
						},
					},
				}

				return stream.Send(dataReq)
			},
		},
		{
			name: "badreq",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				osCreateFunc = func(_ string) (*os.File, error) {
					f, _ := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)

					return f, nil
				}

				osOpenFileFunc = func(_ string, _ int, _ os.FileMode) (*os.File, error) {
					return os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)
				}

				diskInst := &disk.Disk{
					ID:          "dd29c150-b1ed-4518-bd49-c09a6c5ed431",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				setupReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Diskuploadinfo{
						Diskuploadinfo: &cirrina.DiskUploadInfo{
							Diskid:    &cirrina.DiskId{Value: ""},
							Size:      0,
							Sha512Sum: "",
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				dataReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Image{
						Image: []byte{
							0x62, 0xf3, 0x4c, 0x65, 0xc4, 0x32, 0x0e, 0x1d, 0xf6, 0x34, 0xb3, 0x5c, 0xaf, 0x48, 0x32, 0x2a,
							0x0b, 0x03, 0xda, 0x72, 0x23, 0x30, 0xcf, 0x4f, 0xb8, 0x10, 0x05, 0x0c, 0x13, 0xc4, 0xf8, 0x28,
							0x91, 0x48, 0xc4, 0x55, 0x63, 0x62, 0xba, 0x5d, 0xdb, 0xa5, 0x1b, 0xd3, 0x7c, 0x5c, 0x76, 0x63,
							0x56, 0x9c, 0x10, 0x68, 0xcc, 0xea, 0x04, 0x79, 0x42, 0x88, 0x9d, 0xcb, 0xa5, 0xbf, 0xf1, 0x2d,
							0x3c, 0xce, 0x99, 0xaa, 0x77, 0xca, 0x84, 0xa6, 0x7c, 0x40, 0xf7, 0x4f, 0xc4, 0xfb, 0xca, 0xe7,
							0x15, 0x79, 0x3e, 0x21, 0x93, 0x70, 0x9a, 0xab, 0xf5, 0xa6, 0x7b, 0x3f, 0x43, 0xb2, 0xd0, 0xac,
							0xb9, 0xd1, 0x63, 0x7d, 0x77, 0xe8, 0x47, 0x6f, 0x46, 0x23, 0x26, 0x87, 0x1a, 0x9c, 0x33, 0x58,
							0xa3, 0x9b, 0x22, 0x48, 0xb6, 0xcd, 0x9b, 0xd3, 0x80, 0x2c, 0x1f, 0x33, 0x8b, 0x31, 0x0d, 0x82,
						},
					},
				}

				return stream.Send(dataReq)
			},
			wantSetupError: true,
		},
		{
			name: "saveFailFile",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				osCreateFunc = func(_ string) (*os.File, error) {
					return nil, errors.New("bogus create error") //nolint:goerr113
				}

				osOpenFileFunc = func(_ string, _ int, _ os.FileMode) (*os.File, error) {
					return nil, errors.New("bogus open error") //nolint:goerr113
				}

				diskInst := &disk.Disk{
					ID:          "dd29c150-b1ed-4518-bd49-c09a6c5ed431",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				setupReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Diskuploadinfo{
						Diskuploadinfo: &cirrina.DiskUploadInfo{
							Diskid:    &cirrina.DiskId{Value: "dd29c150-b1ed-4518-bd49-c09a6c5ed431"},
							Size:      128,
							Sha512Sum: "9c5dd1250baddae1c12a54f8782dc8903065aa53408000a72cef0868d2914b6a5285f4c7b3ddb493f758515ba906fafc7491db6157c0d164f028cfdc35b9fe89", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				dataReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Image{
						Image: []byte{
							0x62, 0xf3, 0x4c, 0x65, 0xc4, 0x32, 0x0e, 0x1d, 0xf6, 0x34, 0xb3, 0x5c, 0xaf, 0x48, 0x32, 0x2a,
							0x0b, 0x03, 0xda, 0x72, 0x23, 0x30, 0xcf, 0x4f, 0xb8, 0x10, 0x05, 0x0c, 0x13, 0xc4, 0xf8, 0x28,
							0x91, 0x48, 0xc4, 0x55, 0x63, 0x62, 0xba, 0x5d, 0xdb, 0xa5, 0x1b, 0xd3, 0x7c, 0x5c, 0x76, 0x63,
							0x56, 0x9c, 0x10, 0x68, 0xcc, 0xea, 0x04, 0x79, 0x42, 0x88, 0x9d, 0xcb, 0xa5, 0xbf, 0xf1, 0x2d,
							0x3c, 0xce, 0x99, 0xaa, 0x77, 0xca, 0x84, 0xa6, 0x7c, 0x40, 0xf7, 0x4f, 0xc4, 0xfb, 0xca, 0xe7,
							0x15, 0x79, 0x3e, 0x21, 0x93, 0x70, 0x9a, 0xab, 0xf5, 0xa6, 0x7b, 0x3f, 0x43, 0xb2, 0xd0, 0xac,
							0xb9, 0xd1, 0x63, 0x7d, 0x77, 0xe8, 0x47, 0x6f, 0x46, 0x23, 0x26, 0x87, 0x1a, 0x9c, 0x33, 0x58,
							0xa3, 0x9b, 0x22, 0x48, 0xb6, 0xcd, 0x9b, 0xd3, 0x80, 0x2c, 0x1f, 0x33, 0x8b, 0x31, 0x0d, 0x82,
						},
					},
				}

				return stream.Send(dataReq)
			},
			wantSetupError: true,
		},
		{
			name: "missingDisk",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				osCreateFunc = func(_ string) (*os.File, error) {
					f, _ := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)

					return f, nil
				}

				osOpenFileFunc = func(_ string, _ int, _ os.FileMode) (*os.File, error) {
					return os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)
				}

				diskInst := &disk.Disk{
					ID:          "dd29c150-b1ed-4518-bd49-c09a6c5ed431",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				setupReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Diskuploadinfo{
						Diskuploadinfo: &cirrina.DiskUploadInfo{
							Diskid:    &cirrina.DiskId{Value: "a5137fb0-05c6-4551-856e-79a6b6d1608d"},
							Size:      128,
							Sha512Sum: "9c5dd1250baddae1c12a54f8782dc8903065aa53408000a72cef0868d2914b6a5285f4c7b3ddb493f758515ba906fafc7491db6157c0d164f028cfdc35b9fe89", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				dataReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Image{
						Image: []byte{
							0x62, 0xf3, 0x4c, 0x65, 0xc4, 0x32, 0x0e, 0x1d, 0xf6, 0x34, 0xb3, 0x5c, 0xaf, 0x48, 0x32, 0x2a,
							0x0b, 0x03, 0xda, 0x72, 0x23, 0x30, 0xcf, 0x4f, 0xb8, 0x10, 0x05, 0x0c, 0x13, 0xc4, 0xf8, 0x28,
							0x91, 0x48, 0xc4, 0x55, 0x63, 0x62, 0xba, 0x5d, 0xdb, 0xa5, 0x1b, 0xd3, 0x7c, 0x5c, 0x76, 0x63,
							0x56, 0x9c, 0x10, 0x68, 0xcc, 0xea, 0x04, 0x79, 0x42, 0x88, 0x9d, 0xcb, 0xa5, 0xbf, 0xf1, 0x2d,
							0x3c, 0xce, 0x99, 0xaa, 0x77, 0xca, 0x84, 0xa6, 0x7c, 0x40, 0xf7, 0x4f, 0xc4, 0xfb, 0xca, 0xe7,
							0x15, 0x79, 0x3e, 0x21, 0x93, 0x70, 0x9a, 0xab, 0xf5, 0xa6, 0x7b, 0x3f, 0x43, 0xb2, 0xd0, 0xac,
							0xb9, 0xd1, 0x63, 0x7d, 0x77, 0xe8, 0x47, 0x6f, 0x46, 0x23, 0x26, 0x87, 0x1a, 0x9c, 0x33, 0x58,
							0xa3, 0x9b, 0x22, 0x48, 0xb6, 0xcd, 0x9b, 0xd3, 0x80, 0x2c, 0x1f, 0x33, 0x8b, 0x31, 0x0d, 0x82,
						},
					},
				}

				return stream.Send(dataReq)
			},
			wantSetupError: true,
		},
		{
			name: "diskNoName",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				osCreateFunc = func(_ string) (*os.File, error) {
					f, _ := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)

					return f, nil
				}

				osOpenFileFunc = func(_ string, _ int, _ os.FileMode) (*os.File, error) {
					return os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)
				}

				diskInst := &disk.Disk{
					ID:          "a5137fb0-05c6-4551-856e-79a6b6d1608d",
					Name:        "",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				setupReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Diskuploadinfo{
						Diskuploadinfo: &cirrina.DiskUploadInfo{
							Diskid:    &cirrina.DiskId{Value: "a5137fb0-05c6-4551-856e-79a6b6d1608d"},
							Size:      128,
							Sha512Sum: "9c5dd1250baddae1c12a54f8782dc8903065aa53408000a72cef0868d2914b6a5285f4c7b3ddb493f758515ba906fafc7491db6157c0d164f028cfdc35b9fe89", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				dataReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Image{
						Image: []byte{
							0x62, 0xf3, 0x4c, 0x65, 0xc4, 0x32, 0x0e, 0x1d, 0xf6, 0x34, 0xb3, 0x5c, 0xaf, 0x48, 0x32, 0x2a,
							0x0b, 0x03, 0xda, 0x72, 0x23, 0x30, 0xcf, 0x4f, 0xb8, 0x10, 0x05, 0x0c, 0x13, 0xc4, 0xf8, 0x28,
							0x91, 0x48, 0xc4, 0x55, 0x63, 0x62, 0xba, 0x5d, 0xdb, 0xa5, 0x1b, 0xd3, 0x7c, 0x5c, 0x76, 0x63,
							0x56, 0x9c, 0x10, 0x68, 0xcc, 0xea, 0x04, 0x79, 0x42, 0x88, 0x9d, 0xcb, 0xa5, 0xbf, 0xf1, 0x2d,
							0x3c, 0xce, 0x99, 0xaa, 0x77, 0xca, 0x84, 0xa6, 0x7c, 0x40, 0xf7, 0x4f, 0xc4, 0xfb, 0xca, 0xe7,
							0x15, 0x79, 0x3e, 0x21, 0x93, 0x70, 0x9a, 0xab, 0xf5, 0xa6, 0x7b, 0x3f, 0x43, 0xb2, 0xd0, 0xac,
							0xb9, 0xd1, 0x63, 0x7d, 0x77, 0xe8, 0x47, 0x6f, 0x46, 0x23, 0x26, 0x87, 0x1a, 0x9c, 0x33, 0x58,
							0xa3, 0x9b, 0x22, 0x48, 0xb6, 0xcd, 0x9b, 0xd3, 0x80, 0x2c, 0x1f, 0x33, 0x8b, 0x31, 0x0d, 0x82,
						},
					},
				}

				return stream.Send(dataReq)
			},
			wantSetupError: true,
		},
		{
			name: "diskUsedByTwoVMs",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				osCreateFunc = func(_ string) (*os.File, error) {
					f, _ := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)

					return f, nil
				}

				osOpenFileFunc = func(_ string, _ int, _ os.FileMode) (*os.File, error) {
					return os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)
				}

				diskInst := &disk.Disk{
					ID:          "dd29c150-b1ed-4518-bd49-c09a6c5ed431",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst

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
					ISOs: nil,
					Disks: []*disk.Disk{{
						ID: "dd29c150-b1ed-4518-bd49-c09a6c5ed431",
					}},
				}
				testVM2 := vm.VM{
					ID: "4a8bae96-632c-48d1-aee7-6c428639004c",
					Config: vm.Config{
						Model: gorm.Model{
							ID: 3,
						},
						VMID: "4a8bae96-632c-48d1-aee7-6c428639004c",
						CPU:  2,
						Mem:  1024,
					},
					ISOs: nil,
					Disks: []*disk.Disk{{
						ID: "dd29c150-b1ed-4518-bd49-c09a6c5ed431",
					}},
				}
				vm.List.VMList[testVM1.ID] = &testVM1
				vm.List.VMList[testVM2.ID] = &testVM2
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				setupReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Diskuploadinfo{
						Diskuploadinfo: &cirrina.DiskUploadInfo{
							Diskid:    &cirrina.DiskId{Value: "dd29c150-b1ed-4518-bd49-c09a6c5ed431"},
							Size:      128,
							Sha512Sum: "9c5dd1250baddae1c12a54f8782dc8903065aa53408000a72cef0868d2914b6a5285f4c7b3ddb493f758515ba906fafc7491db6157c0d164f028cfdc35b9fe89", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				dataReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Image{
						Image: []byte{
							0x62, 0xf3, 0x4c, 0x65, 0xc4, 0x32, 0x0e, 0x1d, 0xf6, 0x34, 0xb3, 0x5c, 0xaf, 0x48, 0x32, 0x2a,
							0x0b, 0x03, 0xda, 0x72, 0x23, 0x30, 0xcf, 0x4f, 0xb8, 0x10, 0x05, 0x0c, 0x13, 0xc4, 0xf8, 0x28,
							0x91, 0x48, 0xc4, 0x55, 0x63, 0x62, 0xba, 0x5d, 0xdb, 0xa5, 0x1b, 0xd3, 0x7c, 0x5c, 0x76, 0x63,
							0x56, 0x9c, 0x10, 0x68, 0xcc, 0xea, 0x04, 0x79, 0x42, 0x88, 0x9d, 0xcb, 0xa5, 0xbf, 0xf1, 0x2d,
							0x3c, 0xce, 0x99, 0xaa, 0x77, 0xca, 0x84, 0xa6, 0x7c, 0x40, 0xf7, 0x4f, 0xc4, 0xfb, 0xca, 0xe7,
							0x15, 0x79, 0x3e, 0x21, 0x93, 0x70, 0x9a, 0xab, 0xf5, 0xa6, 0x7b, 0x3f, 0x43, 0xb2, 0xd0, 0xac,
							0xb9, 0xd1, 0x63, 0x7d, 0x77, 0xe8, 0x47, 0x6f, 0x46, 0x23, 0x26, 0x87, 0x1a, 0x9c, 0x33, 0x58,
							0xa3, 0x9b, 0x22, 0x48, 0xb6, 0xcd, 0x9b, 0xd3, 0x80, 0x2c, 0x1f, 0x33, 0x8b, 0x31, 0x0d, 0x82,
						},
					},
				}

				return stream.Send(dataReq)
			},
			wantSetupError: true,
		},
		{
			name: "diskUsedByRunningVM",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				osCreateFunc = func(_ string) (*os.File, error) {
					f, _ := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)

					return f, nil
				}

				osOpenFileFunc = func(_ string, _ int, _ os.FileMode) (*os.File, error) {
					return os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)
				}

				diskInst := &disk.Disk{
					ID:          "dd29c150-b1ed-4518-bd49-c09a6c5ed431",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst

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
					ISOs: nil,
					Disks: []*disk.Disk{
						{
							ID: "dd29c150-b1ed-4518-bd49-c09a6c5ed431",
						},
					},
					Status: "RUNNING",
				}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				setupReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Diskuploadinfo{
						Diskuploadinfo: &cirrina.DiskUploadInfo{
							Diskid:    &cirrina.DiskId{Value: "dd29c150-b1ed-4518-bd49-c09a6c5ed431"},
							Size:      128,
							Sha512Sum: "9c5dd1250baddae1c12a54f8782dc8903065aa53408000a72cef0868d2914b6a5285f4c7b3ddb493f758515ba906fafc7491db6157c0d164f028cfdc35b9fe89", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				dataReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Image{
						Image: []byte{
							0x62, 0xf3, 0x4c, 0x65, 0xc4, 0x32, 0x0e, 0x1d, 0xf6, 0x34, 0xb3, 0x5c, 0xaf, 0x48, 0x32, 0x2a,
							0x0b, 0x03, 0xda, 0x72, 0x23, 0x30, 0xcf, 0x4f, 0xb8, 0x10, 0x05, 0x0c, 0x13, 0xc4, 0xf8, 0x28,
							0x91, 0x48, 0xc4, 0x55, 0x63, 0x62, 0xba, 0x5d, 0xdb, 0xa5, 0x1b, 0xd3, 0x7c, 0x5c, 0x76, 0x63,
							0x56, 0x9c, 0x10, 0x68, 0xcc, 0xea, 0x04, 0x79, 0x42, 0x88, 0x9d, 0xcb, 0xa5, 0xbf, 0xf1, 0x2d,
							0x3c, 0xce, 0x99, 0xaa, 0x77, 0xca, 0x84, 0xa6, 0x7c, 0x40, 0xf7, 0x4f, 0xc4, 0xfb, 0xca, 0xe7,
							0x15, 0x79, 0x3e, 0x21, 0x93, 0x70, 0x9a, 0xab, 0xf5, 0xa6, 0x7b, 0x3f, 0x43, 0xb2, 0xd0, 0xac,
							0xb9, 0xd1, 0x63, 0x7d, 0x77, 0xe8, 0x47, 0x6f, 0x46, 0x23, 0x26, 0x87, 0x1a, 0x9c, 0x33, 0x58,
							0xa3, 0x9b, 0x22, 0x48, 0xb6, 0xcd, 0x9b, 0xd3, 0x80, 0x2c, 0x1f, 0x33, 0x8b, 0x31, 0x0d, 0x82,
						},
					},
				}

				return stream.Send(dataReq)
			},
			wantSetupError: true,
		},
		{
			name: "diskUsedByStartingVM",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				osCreateFunc = func(_ string) (*os.File, error) {
					f, _ := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)

					return f, nil
				}

				osOpenFileFunc = func(_ string, _ int, _ os.FileMode) (*os.File, error) {
					return os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)
				}

				diskInst := &disk.Disk{
					ID:          "dd29c150-b1ed-4518-bd49-c09a6c5ed431",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst

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
					ISOs: nil,
					Disks: []*disk.Disk{
						{
							ID: "dd29c150-b1ed-4518-bd49-c09a6c5ed431",
						},
					},
					Status: "STARTING",
				}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				setupReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Diskuploadinfo{
						Diskuploadinfo: &cirrina.DiskUploadInfo{
							Diskid:    &cirrina.DiskId{Value: "dd29c150-b1ed-4518-bd49-c09a6c5ed431"},
							Size:      128,
							Sha512Sum: "9c5dd1250baddae1c12a54f8782dc8903065aa53408000a72cef0868d2914b6a5285f4c7b3ddb493f758515ba906fafc7491db6157c0d164f028cfdc35b9fe89", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				dataReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Image{
						Image: []byte{
							0x62, 0xf3, 0x4c, 0x65, 0xc4, 0x32, 0x0e, 0x1d, 0xf6, 0x34, 0xb3, 0x5c, 0xaf, 0x48, 0x32, 0x2a,
							0x0b, 0x03, 0xda, 0x72, 0x23, 0x30, 0xcf, 0x4f, 0xb8, 0x10, 0x05, 0x0c, 0x13, 0xc4, 0xf8, 0x28,
							0x91, 0x48, 0xc4, 0x55, 0x63, 0x62, 0xba, 0x5d, 0xdb, 0xa5, 0x1b, 0xd3, 0x7c, 0x5c, 0x76, 0x63,
							0x56, 0x9c, 0x10, 0x68, 0xcc, 0xea, 0x04, 0x79, 0x42, 0x88, 0x9d, 0xcb, 0xa5, 0xbf, 0xf1, 0x2d,
							0x3c, 0xce, 0x99, 0xaa, 0x77, 0xca, 0x84, 0xa6, 0x7c, 0x40, 0xf7, 0x4f, 0xc4, 0xfb, 0xca, 0xe7,
							0x15, 0x79, 0x3e, 0x21, 0x93, 0x70, 0x9a, 0xab, 0xf5, 0xa6, 0x7b, 0x3f, 0x43, 0xb2, 0xd0, 0xac,
							0xb9, 0xd1, 0x63, 0x7d, 0x77, 0xe8, 0x47, 0x6f, 0x46, 0x23, 0x26, 0x87, 0x1a, 0x9c, 0x33, 0x58,
							0xa3, 0x9b, 0x22, 0x48, 0xb6, 0xcd, 0x9b, 0xd3, 0x80, 0x2c, 0x1f, 0x33, 0x8b, 0x31, 0x0d, 0x82,
						},
					},
				}

				return stream.Send(dataReq)
			},
			wantSetupError: true,
		},
		{
			name: "diskUsedByStoppingVM",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				osCreateFunc = func(_ string) (*os.File, error) {
					f, _ := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)

					return f, nil
				}

				osOpenFileFunc = func(_ string, _ int, _ os.FileMode) (*os.File, error) {
					return os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)
				}

				diskInst := &disk.Disk{
					ID:          "dd29c150-b1ed-4518-bd49-c09a6c5ed431",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst

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
					ISOs: nil,
					Disks: []*disk.Disk{
						{
							ID: "dd29c150-b1ed-4518-bd49-c09a6c5ed431",
						},
					},
					Status: "STOPPING",
				}
				vm.List.VMList[testVM1.ID] = &testVM1
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				setupReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Diskuploadinfo{
						Diskuploadinfo: &cirrina.DiskUploadInfo{
							Diskid:    &cirrina.DiskId{Value: "dd29c150-b1ed-4518-bd49-c09a6c5ed431"},
							Size:      128,
							Sha512Sum: "9c5dd1250baddae1c12a54f8782dc8903065aa53408000a72cef0868d2914b6a5285f4c7b3ddb493f758515ba906fafc7491db6157c0d164f028cfdc35b9fe89", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				dataReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Image{
						Image: []byte{
							0x62, 0xf3, 0x4c, 0x65, 0xc4, 0x32, 0x0e, 0x1d, 0xf6, 0x34, 0xb3, 0x5c, 0xaf, 0x48, 0x32, 0x2a,
							0x0b, 0x03, 0xda, 0x72, 0x23, 0x30, 0xcf, 0x4f, 0xb8, 0x10, 0x05, 0x0c, 0x13, 0xc4, 0xf8, 0x28,
							0x91, 0x48, 0xc4, 0x55, 0x63, 0x62, 0xba, 0x5d, 0xdb, 0xa5, 0x1b, 0xd3, 0x7c, 0x5c, 0x76, 0x63,
							0x56, 0x9c, 0x10, 0x68, 0xcc, 0xea, 0x04, 0x79, 0x42, 0x88, 0x9d, 0xcb, 0xa5, 0xbf, 0xf1, 0x2d,
							0x3c, 0xce, 0x99, 0xaa, 0x77, 0xca, 0x84, 0xa6, 0x7c, 0x40, 0xf7, 0x4f, 0xc4, 0xfb, 0xca, 0xe7,
							0x15, 0x79, 0x3e, 0x21, 0x93, 0x70, 0x9a, 0xab, 0xf5, 0xa6, 0x7b, 0x3f, 0x43, 0xb2, 0xd0, 0xac,
							0xb9, 0xd1, 0x63, 0x7d, 0x77, 0xe8, 0x47, 0x6f, 0x46, 0x23, 0x26, 0x87, 0x1a, 0x9c, 0x33, 0x58,
							0xa3, 0x9b, 0x22, 0x48, 0xb6, 0xcd, 0x9b, 0xd3, 0x80, 0x2c, 0x1f, 0x33, 0x8b, 0x31, 0x0d, 0x82,
						},
					},
				}

				return stream.Send(dataReq)
			},
			wantSetupError: true,
		},
		{
			name: "sizeTooSmall",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				osCreateFunc = func(_ string) (*os.File, error) {
					f, _ := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)

					return f, nil
				}

				osOpenFileFunc = func(_ string, _ int, _ os.FileMode) (*os.File, error) {
					return os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)
				}

				diskInst := &disk.Disk{
					ID:          "dd29c150-b1ed-4518-bd49-c09a6c5ed431",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				setupReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Diskuploadinfo{
						Diskuploadinfo: &cirrina.DiskUploadInfo{
							Diskid:    &cirrina.DiskId{Value: "dd29c150-b1ed-4518-bd49-c09a6c5ed431"},
							Size:      64,
							Sha512Sum: "9c5dd1250baddae1c12a54f8782dc8903065aa53408000a72cef0868d2914b6a5285f4c7b3ddb493f758515ba906fafc7491db6157c0d164f028cfdc35b9fe89", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				dataReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Image{
						Image: []byte{
							0x62, 0xf3, 0x4c, 0x65, 0xc4, 0x32, 0x0e, 0x1d, 0xf6, 0x34, 0xb3, 0x5c, 0xaf, 0x48, 0x32, 0x2a,
							0x0b, 0x03, 0xda, 0x72, 0x23, 0x30, 0xcf, 0x4f, 0xb8, 0x10, 0x05, 0x0c, 0x13, 0xc4, 0xf8, 0x28,
							0x91, 0x48, 0xc4, 0x55, 0x63, 0x62, 0xba, 0x5d, 0xdb, 0xa5, 0x1b, 0xd3, 0x7c, 0x5c, 0x76, 0x63,
							0x56, 0x9c, 0x10, 0x68, 0xcc, 0xea, 0x04, 0x79, 0x42, 0x88, 0x9d, 0xcb, 0xa5, 0xbf, 0xf1, 0x2d,
							0x3c, 0xce, 0x99, 0xaa, 0x77, 0xca, 0x84, 0xa6, 0x7c, 0x40, 0xf7, 0x4f, 0xc4, 0xfb, 0xca, 0xe7,
							0x15, 0x79, 0x3e, 0x21, 0x93, 0x70, 0x9a, 0xab, 0xf5, 0xa6, 0x7b, 0x3f, 0x43, 0xb2, 0xd0, 0xac,
							0xb9, 0xd1, 0x63, 0x7d, 0x77, 0xe8, 0x47, 0x6f, 0x46, 0x23, 0x26, 0x87, 0x1a, 0x9c, 0x33, 0x58,
							0xa3, 0x9b, 0x22, 0x48, 0xb6, 0xcd, 0x9b, 0xd3, 0x80, 0x2c, 0x1f, 0x33, 0x8b, 0x31, 0x0d, 0x82,
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
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				osCreateFunc = func(_ string) (*os.File, error) {
					f, _ := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)

					return f, nil
				}

				osOpenFileFunc = func(_ string, _ int, _ os.FileMode) (*os.File, error) {
					return os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)
				}

				diskInst := &disk.Disk{
					ID:          "dd29c150-b1ed-4518-bd49-c09a6c5ed431",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				setupReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Diskuploadinfo{
						Diskuploadinfo: &cirrina.DiskUploadInfo{
							Diskid:    &cirrina.DiskId{Value: "dd29c150-b1ed-4518-bd49-c09a6c5ed431"},
							Size:      256,
							Sha512Sum: "9c5dd1250baddae1c12a54f8782dc8903065aa53408000a72cef0868d2914b6a5285f4c7b3ddb493f758515ba906fafc7491db6157c0d164f028cfdc35b9fe89", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				dataReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Image{
						Image: []byte{
							0x62, 0xf3, 0x4c, 0x65, 0xc4, 0x32, 0x0e, 0x1d, 0xf6, 0x34, 0xb3, 0x5c, 0xaf, 0x48, 0x32, 0x2a,
							0x0b, 0x03, 0xda, 0x72, 0x23, 0x30, 0xcf, 0x4f, 0xb8, 0x10, 0x05, 0x0c, 0x13, 0xc4, 0xf8, 0x28,
							0x91, 0x48, 0xc4, 0x55, 0x63, 0x62, 0xba, 0x5d, 0xdb, 0xa5, 0x1b, 0xd3, 0x7c, 0x5c, 0x76, 0x63,
							0x56, 0x9c, 0x10, 0x68, 0xcc, 0xea, 0x04, 0x79, 0x42, 0x88, 0x9d, 0xcb, 0xa5, 0xbf, 0xf1, 0x2d,
							0x3c, 0xce, 0x99, 0xaa, 0x77, 0xca, 0x84, 0xa6, 0x7c, 0x40, 0xf7, 0x4f, 0xc4, 0xfb, 0xca, 0xe7,
							0x15, 0x79, 0x3e, 0x21, 0x93, 0x70, 0x9a, 0xab, 0xf5, 0xa6, 0x7b, 0x3f, 0x43, 0xb2, 0xd0, 0xac,
							0xb9, 0xd1, 0x63, 0x7d, 0x77, 0xe8, 0x47, 0x6f, 0x46, 0x23, 0x26, 0x87, 0x1a, 0x9c, 0x33, 0x58,
							0xa3, 0x9b, 0x22, 0x48, 0xb6, 0xcd, 0x9b, 0xd3, 0x80, 0x2c, 0x1f, 0x33, 0x8b, 0x31, 0x0d, 0x82,
						},
					},
				}

				_ = stream.Send(dataReq)

				return stream.CloseSend()
			},
			wantSendError: true,
		},
		{
			name: "badChecksum",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
				osCreateFunc = func(_ string) (*os.File, error) {
					f, _ := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)

					return f, nil
				}

				osOpenFileFunc = func(_ string, _ int, _ os.FileMode) (*os.File, error) {
					return os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0644)
				}

				diskInst := &disk.Disk{
					ID:          "dd29c150-b1ed-4518-bd49-c09a6c5ed431",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			mockStreamSetupReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				setupReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Diskuploadinfo{
						Diskuploadinfo: &cirrina.DiskUploadInfo{
							Diskid:    &cirrina.DiskId{Value: "dd29c150-b1ed-4518-bd49-c09a6c5ed431"},
							Size:      128,
							Sha512Sum: "9c5dd1250baddae1c12a54f8782dc8903065aa53408000a72cef0868d2914b6a5285f4c7b3ddb493f758515ba906fafc7491db6157c0d164f028cfdc35b9fe89", //nolint:lll
						},
					},
				}

				return stream.Send(setupReq)
			},
			mockStreamSendReqFunc: func(stream cirrina.VMInfo_UploadDiskClient) error {
				dataReq := &cirrina.DiskImageRequest{
					Data: &cirrina.DiskImageRequest_Image{
						Image: []byte{
							0x00, 0xf3, 0x4c, 0x65, 0xc4, 0x32, 0x0e, 0x1d, 0xf6, 0x34, 0xb3, 0x5c, 0xaf, 0x48, 0x32, 0x2a,
							0x0b, 0x03, 0xda, 0x72, 0x23, 0x30, 0xcf, 0x4f, 0xb8, 0x10, 0x05, 0x0c, 0x13, 0xc4, 0xf8, 0x28,
							0x91, 0x48, 0xc4, 0x55, 0x63, 0x62, 0xba, 0x5d, 0xdb, 0xa5, 0x1b, 0xd3, 0x7c, 0x5c, 0x76, 0x63,
							0x56, 0x9c, 0x10, 0x68, 0xcc, 0xea, 0x04, 0x79, 0x42, 0x88, 0x9d, 0xcb, 0xa5, 0xbf, 0xf1, 0x2d,
							0x3c, 0xce, 0x99, 0xaa, 0x77, 0xca, 0x84, 0xa6, 0x7c, 0x40, 0xf7, 0x4f, 0xc4, 0xfb, 0xca, 0xe7,
							0x15, 0x79, 0x3e, 0x21, 0x93, 0x70, 0x9a, 0xab, 0xf5, 0xa6, 0x7b, 0x3f, 0x43, 0xb2, 0xd0, 0xac,
							0xb9, 0xd1, 0x63, 0x7d, 0x77, 0xe8, 0x47, 0x6f, 0x46, 0x23, 0x26, 0x87, 0x1a, 0x9c, 0x33, 0x58,
							0xa3, 0x9b, 0x22, 0x48, 0xb6, 0xcd, 0x9b, 0xd3, 0x80, 0x2c, 0x1f, 0x33, 0x8b, 0x31, 0x0d, 0x82,
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
			// clear out list(s) from other parallel test runs
			disk.List.DiskList = map[string]*disk.Disk{}
			vm.List.VMList = map[string]*vm.VM{}

			testDB, mock := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mock)

			lis := bufconn.Listen(1024 * 1024)

			testServer := grpc.NewServer()
			reflection.Register(testServer)
			cirrina.RegisterVMInfoServer(testServer, &server{})

			go func() {
				err := testServer.Serve(lis)
				if err != nil {
					log.Fatalf("Server exited with error: %v", err)
				}
			}()

			defer testServer.Stop()

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

			stream, _ := client.UploadDisk(context.Background())

			_ = testCase.mockStreamSetupReqFunc(stream)

			if testCase.wantSetupError {
				var rb cirrina.ReqBool

				_ = stream.RecvMsg(&rb)

				if rb.GetSuccess() {
					t.Errorf("UploadDisk() err = %v, wantSetupErr %v", err, testCase.wantSetupError)
				}

				return
			}

			_ = testCase.mockStreamSendReqFunc(stream)

			if testCase.wantSendError {
				var rb cirrina.ReqBool

				_ = stream.RecvMsg(&rb)

				if rb.GetSuccess() {
					t.Errorf("UploadDisk() err = %v, wantSendError %v", err, testCase.wantSendError)
				}

				return
			}

			reply, _ := stream.CloseAndRecv()

			if !reply.GetSuccess() && !testCase.wantErr {
				t.Errorf("UploadDisk() success = %v, wantErr %v", reply.GetSuccess(), testCase.wantErr)
			}
		})
	}
}

func Test_validateDiskReq(t *testing.T) {
	type args struct {
		diskUploadReq *cirrina.DiskUploadInfo
	}

	tests := []struct {
		name    string
		args    args
		want    *disk.Disk
		wantErr bool
	}{
		{
			name:    "nilReq",
			args:    args{diskUploadReq: nil},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "nilDiskID",
			args:    args{diskUploadReq: &cirrina.DiskUploadInfo{Diskid: nil}},
			want:    nil,
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := validateDiskReq(testCase.args.diskUploadReq)
			if (err != nil) != testCase.wantErr {
				t.Errorf("validateDiskReq() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}
