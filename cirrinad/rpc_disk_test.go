package main

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-test/deep"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"

	"cirrina/cirrina"
	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
)

//nolint:paralleltest,maintidx
func Test_server_AddDisk(t *testing.T) {
	createUpdateTime := time.Now()
	diskDevTypeFile := cirrina.DiskDevType_FILE
	diskTypeNVME := cirrina.DiskType_NVME
	// diskDevTypeZVol := cirrina.DiskDevType_ZVOL

	type fields struct {
		UnimplementedVMInfoServer cirrina.UnimplementedVMInfoServer
	}

	type args struct {
		in0      context.Context //nolint:containedctx
		diskInfo *cirrina.DiskInfo
	}

	tests := []struct {
		name          string
		mockClosure   func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields        fields
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
			fields: fields{
				UnimplementedVMInfoServer: cirrina.UnimplementedVMInfoServer{},
			},
			args: args{
				in0: nil,
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
			want: &cirrina.DiskId{
				Value: "",
			},
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
			fields: fields{
				UnimplementedVMInfoServer: cirrina.UnimplementedVMInfoServer{},
			},
			args: args{
				in0: nil,
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
			fields: fields{
				UnimplementedVMInfoServer: cirrina.UnimplementedVMInfoServer{},
			},
			args: args{
				in0: nil,
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
			fields: fields{
				UnimplementedVMInfoServer: cirrina.UnimplementedVMInfoServer{},
			},
			args: args{
				in0: nil,
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
			if testCase.args.diskInfo.DiskDevType == nil || *testCase.args.diskInfo.DiskDevType == cirrina.DiskDevType_FILE { //nolint:protogetter,lll
				fileMock.EXPECT().CheckExists(gomock.Any()).MaxTimes(1).Return(testCase.wantExists, existsErr)
				fileMock.EXPECT().Add(gomock.Any(), gomock.Any()).MaxTimes(1).Return(createErr)
			}

			if testCase.args.diskInfo.DiskDevType != nil && *testCase.args.diskInfo.DiskDevType == cirrina.DiskDevType_ZVOL { //nolint:protogetter,lll
				zfsMock.EXPECT().CheckExists(gomock.Any()).MaxTimes(1).Return(testCase.wantExists, existsErr)
				zfsMock.EXPECT().Add(gomock.Any(), gomock.Any()).MaxTimes(1).Return(createErr)
			}

			s := &server{
				UnimplementedVMInfoServer: testCase.fields.UnimplementedVMInfoServer,
			}

			got, err := s.AddDisk(testCase.args.in0, testCase.args.diskInfo)
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
