package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net"
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
			testDB, mockDB := cirrinadtest.NewMockDB("diskTest")
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
		testCase := testCase
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
