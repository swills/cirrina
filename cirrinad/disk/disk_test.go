package disk

import (
	"database/sql"
	"errors"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-test/deep"
	"github.com/mattn/go-sqlite3"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/config"
)

func TestGetAllDB(t *testing.T) {
	createUpdateTime := time.Now()

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        []*Disk
	}{
		{
			name: "testDisksGetAllDB",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					diskDB: testDB,
				}
				mock.ExpectQuery("^SELECT \\* FROM `disks` WHERE `disks`.`deleted_at` IS NULL$").
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
								"test2023061001_14.img",
								"a virtual hard disk image",
								"NVME",
								"FILE",
								1,
								0,
							).
							AddRow(
								"41ae49ee-6e7e-47c2-aebb-671f2dbac4a2",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2023061001_15.img",
								"another virtual hard disk image",
								"NVME",
								"FILE",
								1,
								0,
							),
					)
			},
			want: []*Disk{
				{
					ID:        "20d3098f-7ccf-484e-bed4-757940a3c775",
					CreatedAt: createUpdateTime,
					UpdatedAt: createUpdateTime,
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
					Name:        "test2023061001_14.img",
					Description: "a virtual hard disk image",
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
				},
				{
					ID:        "41ae49ee-6e7e-47c2-aebb-671f2dbac4a2",
					CreatedAt: createUpdateTime,
					UpdatedAt: createUpdateTime,
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
					Name:        "test2023061001_15.img",
					Description: "another virtual hard disk image",
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
				},
			},
		},
	}
	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("diskTest")
			testCase.mockClosure(testDB, mock)

			got := GetAllDB()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			mock.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			if err = db.Close(); err != nil {
				t.Error(err)
			}

			if err = mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func Test_diskTypeValid(t *testing.T) {
	type args struct {
		diskType string
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "diskTypeValidNVME",
			args: args{diskType: "NVME"},
			want: true,
		},
		{
			name: "diskTypeValidAHCIHD",
			args: args{diskType: "AHCI-HD"},
			want: true,
		},
		{
			name: "diskTypeValidVIRTIOBLK",
			args: args{diskType: "VIRTIO-BLK"},
			want: true,
		},
		{
			name: "diskTypeInvalidJunk",
			args: args{diskType: "something"},
			want: false,
		},
		{
			name: "diskTypeInvalidEmpty",
			args: args{diskType: "something"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := diskTypeValid(tt.args.diskType); got != tt.want {
				t.Errorf("diskTypeValid() = %v, wantFetch %v", got, tt.want)
			}
		})
	}
}

func Test_diskDevTypeValid(t *testing.T) {
	type args struct {
		diskDevType string
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "diskDevTypeValidFile",
			args: args{diskDevType: "FILE"},
			want: true,
		},
		{
			name: "diskDevTypeValidZVOL",
			args: args{diskDevType: "ZVOL"},
			want: true,
		},
		{
			name: "diskDevTypeInvalidJunk",
			args: args{diskDevType: "junk"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := diskDevTypeValid(tt.args.diskDevType); got != tt.want {
				t.Errorf("diskDevTypeValid() = %v, wantFetch %v", got, tt.want)
			}
		})
	}
}

func Test_validateDisk(t *testing.T) {
	type args struct {
		diskInst *Disk
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "validateDiskValid1",
			args: args{diskInst: &Disk{
				Name:        "someCoolDisk1",
				Description: "a totally cool test disk",
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
			}},
			wantErr: false,
		},
		{
			name: "validateDiskInvalid0",
			args: args{diskInst: &Disk{
				Name:        "someCoolDisk1",
				Description: "a totally cool test disk",
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
			}},
			wantErr: true,
		},
		{
			name: "validateDiskInvalid1",
			args: args{diskInst: &Disk{
				Name:        ".someCoolDisk1",
				Description: "a totally cool test disk",
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
			}},
			wantErr: true,
		},
		{
			name: "validateDiskInvalid2",
			args: args{diskInst: &Disk{
				Name:        "someCoolDisk1",
				Description: "a totally cool test disk",
				Type:        "junk",
				DevType:     "FILE",
				DiskCache: sql.NullBool{
					Bool:  true,
					Valid: true,
				},
				DiskDirect: sql.NullBool{
					Bool:  false,
					Valid: true,
				},
			}},
			wantErr: true,
		},
		{
			name: "validateDiskInvalid3",
			args: args{diskInst: &Disk{
				Name:        "someCoolDisk1",
				Description: "a totally cool test disk",
				Type:        "NVME",
				DevType:     "junk",
				DiskCache: sql.NullBool{
					Bool:  true,
					Valid: true,
				},
				DiskDirect: sql.NullBool{
					Bool:  false,
					Valid: true,
				},
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateDisk(tt.args.diskInst); (err != nil) != tt.wantErr {
				t.Errorf("validateDisk() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetByID(t *testing.T) {
	type args struct {
		diskID string
	}

	tests := []struct {
		name        string
		mockClosure func()
		args        args
		want        *Disk
		wantErr     bool
	}{
		{
			name: "Valid1",
			mockClosure: func() {
				diskInst := &Disk{
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
				List.DiskList[diskInst.ID] = diskInst
			},
			args: args{diskID: "0d4a0338-0b68-4645-b99d-9cbb30df272d"},
			want: &Disk{
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
			},
			wantErr: false,
		},
		{
			name: "Invalid1",
			mockClosure: func() {
				diskInst := &Disk{
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
				List.DiskList[diskInst.ID] = diskInst
			},
			args:    args{diskID: "a3f817df-26d4-4955-97e6-6e7732b03c5d"},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "Invalid2",
			mockClosure: func() {},
			args:        args{diskID: ""},
			want:        nil,
			wantErr:     true,
		},
	}
	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("diskTest")

			testCase.mockClosure()

			got, err := GetByID(testCase.args.diskID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			mock.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			if err = db.Close(); err != nil {
				t.Error(err)
			}

			if err = mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestGetByName(t *testing.T) {
	type args struct {
		name string
	}

	tests := []struct {
		name        string
		mockClosure func()
		args        args
		want        *Disk
		wantErr     bool
	}{
		{
			name: "Valid1",
			mockClosure: func() {
				diskInst := &Disk{
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
				List.DiskList[diskInst.ID] = diskInst
			},
			args: args{name: "aDisk"},
			want: &Disk{
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
			},
			wantErr: false,
		},
		{
			name:        "Invalid1",
			mockClosure: func() {},
			args:        args{name: "blah"},
			want:        nil,
			wantErr:     true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := GetByName(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetByName() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("GetByName() got = %v, wantFetch %v", got, testCase.want)
			}
		})
	}
}

func TestDisk_GetPath(t *testing.T) {
	type fields struct {
		Name    string
		DevType string
	}

	tests := []struct {
		name        string
		mockClosure func()
		fields      fields
		want        string
	}{
		{
			name: "Valid1",
			mockClosure: func() {
				config.Config.Disk.VM.Path.Image = "/some/path"
			},
			fields: fields{
				Name:    "someDisk",
				DevType: "FILE",
			},
			want: "/some/path/someDisk.img",
		},
		{
			name: "Valid2",
			mockClosure: func() {
				config.Config.Disk.VM.Path.Zpool = "somePool/dataSet"
			},
			fields: fields{
				Name:    "someDisk",
				DevType: "ZVOL",
			},
			want: "somePool/dataSet/someDisk",
		},
		{
			name: "Invalid1",
			mockClosure: func() {
			},
			fields: fields{
				DevType: "",
			},
			want: "",
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()
			d := &Disk{
				Name:    testCase.fields.Name,
				DevType: testCase.fields.DevType,
			}

			if got := d.GetPath(); got != testCase.want {
				t.Errorf("GetPath() = %v, wantFetch %v", got, testCase.want)
			}
		})
	}
}

//nolint:maintidx
func TestDisk_Save(t *testing.T) {
	type diskFields struct {
		ID          string
		Name        string
		Description string
		Type        string
		DevType     string
		DiskCache   sql.NullBool
		DiskDirect  sql.NullBool
	}

	tests := []struct {
		name        string
		testDisk    diskFields
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		wantErr     bool
	}{
		{
			name: "success1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					diskDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `disks` SET `description`=?,`dev_type`=?,`disk_cache`=?,`disk_direct`=?,`name`=?,`type`=?,`updated_at`=? WHERE `disks`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("some test disk", "FILE", true, false, "aDisk", "NVME", sqlmock.AnyArg(), "89609970-ae8d-4ccd-a71c-9f69fd8e12cd"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			testDisk: diskFields{
				ID:          "89609970-ae8d-4ccd-a71c-9f69fd8e12cd",
				Name:        "aDisk",
				Description: "some test disk",
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
			},
			wantErr: false,
		},
		{
			name: "success2",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					diskDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `disks` SET `description`=?,`dev_type`=?,`disk_cache`=?,`disk_direct`=?,`name`=?,`type`=?,`updated_at`=? WHERE `disks`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("some test disk", "FILE", true, false, "aDisk", "AHCI-HD", sqlmock.AnyArg(), "89609970-ae8d-4ccd-a71c-9f69fd8e12cd"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			testDisk: diskFields{
				ID:          "89609970-ae8d-4ccd-a71c-9f69fd8e12cd",
				Name:        "aDisk",
				Description: "some test disk",
				Type:        "AHCI-HD",
				DevType:     "FILE",
				DiskCache: sql.NullBool{
					Bool:  true,
					Valid: true,
				},
				DiskDirect: sql.NullBool{
					Bool:  false,
					Valid: true,
				},
			},
			wantErr: false,
		},
		{
			name: "success3",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					diskDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `disks` SET `description`=?,`dev_type`=?,`disk_cache`=?,`disk_direct`=?,`name`=?,`type`=?,`updated_at`=? WHERE `disks`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("some test disk", "FILE", true, false, "aDisk", "VIRTIO-BLK", sqlmock.AnyArg(), "89609970-ae8d-4ccd-a71c-9f69fd8e12cd"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			testDisk: diskFields{
				ID:          "89609970-ae8d-4ccd-a71c-9f69fd8e12cd",
				Name:        "aDisk",
				Description: "some test disk",
				Type:        "VIRTIO-BLK",
				DevType:     "FILE",
				DiskCache: sql.NullBool{
					Bool:  true,
					Valid: true,
				},
				DiskDirect: sql.NullBool{
					Bool:  false,
					Valid: true,
				},
			},
			wantErr: false,
		},
		{
			name: "success4",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					diskDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `disks` SET `description`=?,`dev_type`=?,`disk_cache`=?,`disk_direct`=?,`name`=?,`type`=?,`updated_at`=? WHERE `disks`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("some test disk", "ZVOL", true, false, "aDisk", "NVME", sqlmock.AnyArg(), "89609970-ae8d-4ccd-a71c-9f69fd8e12cd"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			testDisk: diskFields{
				ID:          "89609970-ae8d-4ccd-a71c-9f69fd8e12cd",
				Name:        "aDisk",
				Description: "some test disk",
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
			},
			wantErr: false,
		},
		{
			name: "success5",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					diskDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(

					regexp.QuoteMeta(
						"UPDATE `disks` SET `description`=?,`dev_type`=?,`disk_cache`=?,`disk_direct`=?,`name`=?,`type`=?,`updated_at`=? WHERE `disks`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("some test disk", "ZVOL", true, false, "aDisk", "AHCI-HD", sqlmock.AnyArg(), "89609970-ae8d-4ccd-a71c-9f69fd8e12cd"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			testDisk: diskFields{
				ID:          "89609970-ae8d-4ccd-a71c-9f69fd8e12cd",
				Name:        "aDisk",
				Description: "some test disk",
				Type:        "AHCI-HD",
				DevType:     "ZVOL",
				DiskCache: sql.NullBool{
					Bool:  true,
					Valid: true,
				},
				DiskDirect: sql.NullBool{
					Bool:  false,
					Valid: true,
				},
			},
			wantErr: false,
		},
		{
			name: "success6",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					diskDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `disks` SET `description`=?,`dev_type`=?,`disk_cache`=?,`disk_direct`=?,`name`=?,`type`=?,`updated_at`=? WHERE `disks`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("some test disk", "ZVOL", true, false, "aDisk", "VIRTIO-BLK", sqlmock.AnyArg(), "89609970-ae8d-4ccd-a71c-9f69fd8e12cd"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			testDisk: diskFields{
				ID:          "89609970-ae8d-4ccd-a71c-9f69fd8e12cd",
				Name:        "aDisk",
				Description: "some test disk",
				Type:        "VIRTIO-BLK",
				DevType:     "ZVOL",
				DiskCache: sql.NullBool{
					Bool:  true,
					Valid: true,
				},
				DiskDirect: sql.NullBool{
					Bool:  false,
					Valid: true,
				},
			},
			wantErr: false,
		},
		{
			name: "error1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					diskDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `disks` SET `description`=?,`dev_type`=?,`disk_cache`=?,`disk_direct`=?,`name`=?,`type`=?,`updated_at`=? WHERE `disks`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("some test disk", "FILE", true, false, "aDisk", "NVME", sqlmock.AnyArg(), "89609970-ae8d-4ccd-a71c-9f69fd8e12cd"). //nolint:lll
					// does not matter what error is returned
					WillReturnError(gorm.ErrInvalidField)
				mock.ExpectRollback()
			},
			testDisk: diskFields{
				ID:          "89609970-ae8d-4ccd-a71c-9f69fd8e12cd",
				Name:        "aDisk",
				Description: "some test disk",
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
			},
			wantErr: true,
		},
		{
			name: "error2",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					diskDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `disks` SET `description`=?,`dev_type`=?,`disk_cache`=?,`disk_direct`=?,`name`=?,`type`=?,`updated_at`=? WHERE `disks`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs("some test disk", "ZVOL", true, false, "aDisk", "junk", sqlmock.AnyArg(), "89609970-ae8d-4ccd-a71c-9f69fd8e12cd"). //nolint:lll
					WillReturnError(sqlite3.ErrConstraintCheck)
				mock.ExpectRollback()
			},
			testDisk: diskFields{
				ID:          "89609970-ae8d-4ccd-a71c-9f69fd8e12cd",
				Name:        "aDisk",
				Description: "some test disk",
				Type:        "junk",
				DevType:     "ZVOL",
				DiskCache: sql.NullBool{
					Bool:  true,
					Valid: true,
				},
				DiskDirect: sql.NullBool{
					Bool:  false,
					Valid: true,
				},
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("diskTest")
			testCase.mockClosure(testDB, mock)

			aDisk := &Disk{
				ID:          testCase.testDisk.ID,
				Name:        testCase.testDisk.Name,
				Description: testCase.testDisk.Description,
				Type:        testCase.testDisk.Type,
				DevType:     testCase.testDisk.DevType,
				DiskCache:   testCase.testDisk.DiskCache,
				DiskDirect:  testCase.testDisk.DiskDirect,
			}

			if err := aDisk.Save(); (err != nil) != testCase.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, testCase.wantErr)
			}

			mock.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			if err = db.Close(); err != nil {
				t.Error(err)
			}

			if err = mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	type args struct {
		diskID string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		wantErr     bool
	}{
		{
			name: "success1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				diskInst := &Disk{
					ID:          "e89be82f-25c7-42b9-823a-df432e64320e",
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
				List.DiskList[diskInst.ID] = diskInst

				instance = &singleton{ // prevents parallel testing
					diskDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `disks` SET `deleted_at`=? WHERE `disks`.`id` = ? AND `disks`.`deleted_at` IS NULL"),
				).
					WithArgs(sqlmock.AnyArg(), "e89be82f-25c7-42b9-823a-df432e64320e").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args: args{
				diskID: "e89be82f-25c7-42b9-823a-df432e64320e",
			},
			wantErr: false,
		},
		{
			name: "error1",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
			},
			args: args{
				diskID: "",
			},
			wantErr: true,
		},
		{
			name: "error2",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				diskInst := &Disk{
					ID:          "e89be82f-25c7-42b9-823a-df432e64320e",
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
				List.DiskList[diskInst.ID] = diskInst

				instance = &singleton{ // prevents parallel testing
					diskDB: testDB,
				}
			},
			args: args{
				diskID: "2e1f3582-7bc1-4a7a-beeb-1651e371435e",
			},
			wantErr: true,
		},
		{
			name: "error3",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				diskInst := &Disk{
					ID:          "e89be82f-25c7-42b9-823a-df432e64320e",
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
				List.DiskList[diskInst.ID] = diskInst

				instance = &singleton{ // prevents parallel testing
					diskDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `disks` SET `deleted_at`=? WHERE `disks`.`id` = ? AND `disks`.`deleted_at` IS NULL"),
				).
					WithArgs(sqlmock.AnyArg(), "e89be82f-25c7-42b9-823a-df432e64320e").
					WillReturnResult(sqlmock.NewResult(1, 0))
				mock.ExpectCommit()
			},
			args: args{
				diskID: "e89be82f-25c7-42b9-823a-df432e64320e",
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("diskTest")
			testCase.mockClosure(testDB, mock)

			err := Delete(testCase.args.diskID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, testCase.wantErr)
			}

			mock.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			if err = db.Close(); err != nil {
				t.Error(err)
			}

			if err = mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func Test_diskExists(t *testing.T) {
	type args struct {
		diskInst *Disk
	}

	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := diskExistsCacheDB(testCase.args.diskInst)
			if (err != nil) != testCase.wantErr {
				t.Errorf("diskExistsCacheDB() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("diskExistsCacheDB() got = %v, wantFetch %v", got, testCase.want)
			}
		})
	}
}

//nolint:maintidx
func TestCreate(t *testing.T) {
	type args struct {
		diskInst *Disk
		size     string
	}

	tests := []struct {
		name                  string
		args                  args
		mockClosure           func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		diskExistsCacheDBFunc func(diskInst *Disk) (bool, error)
		diskValidateFunc      func(diskInst *Disk) error
		wantExists            bool
		wantExistsErr         bool
		wantCreateErr         bool
		wantErr               bool
	}{
		{
			name:                  "fileExistsCacheOrDB",
			args:                  args{diskInst: &Disk{Name: "someDisk", DevType: "FILE"}, size: "2g"},
			mockClosure:           func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			diskExistsCacheDBFunc: func(*Disk) (bool, error) { return true, nil },
			diskValidateFunc:      func(*Disk) error { return nil },
			wantErr:               true,
		},
		{
			name:                  "zvolExistsCacheOrDB",
			args:                  args{diskInst: &Disk{Name: "someDisk", DevType: "ZVOL"}, size: "2g"},
			mockClosure:           func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			diskExistsCacheDBFunc: func(*Disk) (bool, error) { return true, nil },
			diskValidateFunc:      func(*Disk) error { return nil },
			wantErr:               true,
		},
		{
			name:                  "badType",
			args:                  args{diskInst: &Disk{Name: "someDisk", DevType: "asdf"}, size: "2g"},
			mockClosure:           func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			diskExistsCacheDBFunc: func(*Disk) (bool, error) { return true, nil },
			diskValidateFunc:      func(*Disk) error { return nil },
			wantErr:               true,
		},
		{
			name:                  "errorCheckingExistsCacheMem",
			args:                  args{diskInst: &Disk{Name: "someDisk", DevType: "ZVOL"}, size: "2g"},
			mockClosure:           func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			diskExistsCacheDBFunc: func(*Disk) (bool, error) { return true, errors.New("some bogus error") }, //nolint:goerr113
			diskValidateFunc:      func(*Disk) error { return nil },
			wantErr:               true,
		},
		{
			name:                  "errorCheckingExistsFile",
			args:                  args{diskInst: &Disk{Name: "someDisk", DevType: "FILE"}, size: "2g"},
			mockClosure:           func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			diskExistsCacheDBFunc: func(*Disk) (bool, error) { return false, nil },
			diskValidateFunc:      func(*Disk) error { return nil },
			wantExists:            true,
			wantExistsErr:         true,
			wantErr:               true,
		},
		{
			name:                  "errorCheckingExistsZVOL",
			args:                  args{diskInst: &Disk{Name: "someDisk", DevType: "ZVOL"}, size: "2g"},
			mockClosure:           func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			diskExistsCacheDBFunc: func(*Disk) (bool, error) { return false, nil },
			diskValidateFunc:      func(*Disk) error { return nil },
			wantExists:            true,
			wantExistsErr:         true,
			wantErr:               true,
		},
		{
			name:                  "existsFile",
			args:                  args{diskInst: &Disk{Name: "someDisk", DevType: "FILE"}, size: "2g"},
			mockClosure:           func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			diskExistsCacheDBFunc: func(*Disk) (bool, error) { return false, nil },
			diskValidateFunc:      func(*Disk) error { return nil },
			wantExists:            true,
			wantExistsErr:         false,
			wantErr:               true,
		},
		{
			name:                  "existsZFS",
			args:                  args{diskInst: &Disk{Name: "someDisk", DevType: "ZVOL"}, size: "2g"},
			mockClosure:           func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			diskExistsCacheDBFunc: func(*Disk) (bool, error) { return false, nil },
			diskValidateFunc:      func(*Disk) error { return nil },
			wantExists:            true,
			wantExistsErr:         false,
			wantErr:               true,
		},
		{
			name:                  "invalidDisk",
			args:                  args{diskInst: &Disk{Name: "someDisk", DevType: "ZVOL"}, size: "2g"},
			mockClosure:           func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			diskExistsCacheDBFunc: func(*Disk) (bool, error) { return false, nil },
			diskValidateFunc:      func(*Disk) error { return errors.New("bogus invalid disk error") }, //nolint:goerr113
			wantExists:            false,
			wantExistsErr:         false,
			wantErr:               true,
		},
		{
			name:                  "badDiskSize",
			args:                  args{diskInst: &Disk{Name: "someDisk", DevType: "ZVOL"}, size: "123z"},
			mockClosure:           func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			diskExistsCacheDBFunc: func(*Disk) (bool, error) { return false, nil },
			diskValidateFunc:      func(*Disk) error { return nil },
			wantExists:            false,
			wantExistsErr:         false,
			wantErr:               true,
		},
		{
			name:                  "badCreateFile",
			args:                  args{diskInst: &Disk{Name: "someDisk", DevType: "FILE"}, size: "1g"},
			mockClosure:           func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			diskExistsCacheDBFunc: func(*Disk) (bool, error) { return false, nil },
			diskValidateFunc:      func(*Disk) error { return nil },
			wantExists:            false,
			wantExistsErr:         false,
			wantCreateErr:         true,
			wantErr:               true,
		},
		{
			name:                  "badCreateZVOL",
			args:                  args{diskInst: &Disk{Name: "someDisk", DevType: "ZVOL"}, size: "1g"},
			mockClosure:           func(_ *gorm.DB, _ sqlmock.Sqlmock) {},
			diskExistsCacheDBFunc: func(*Disk) (bool, error) { return false, nil },
			diskValidateFunc:      func(*Disk) error { return nil },
			wantExists:            false,
			wantExistsErr:         false,
			wantCreateErr:         true,
			wantErr:               true,
		},
		{
			name: "badDbFile",
			args: args{diskInst: &Disk{
				Name:        "someDisk",
				Description: "a test disk",
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
			}, size: "1g"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					diskDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta("INSERT INTO `disks` (`created_at`,`updated_at`,`deleted_at`,`description`,`type`,`dev_type`,`disk_cache`,`disk_direct`,`id`,`name`) VALUES (?,?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`")). //nolint:lll
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil,
						"a test disk", "NVME", "FILE", true, false, sqlmock.AnyArg(), "someDisk",
					).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
				mock.ExpectRollback()
			},
			diskExistsCacheDBFunc: func(*Disk) (bool, error) { return false, nil },
			diskValidateFunc:      func(*Disk) error { return nil },
			wantExists:            false,
			wantExistsErr:         false,
			wantCreateErr:         false,
			wantErr:               true,
		},
		{
			name: "badDbZVOL",
			args: args{diskInst: &Disk{
				Name:        "someDisk",
				Description: "a test disk",
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
			}, size: "1g"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					diskDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta("INSERT INTO `disks` (`created_at`,`updated_at`,`deleted_at`,`description`,`type`,`dev_type`,`disk_cache`,`disk_direct`,`id`,`name`) VALUES (?,?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`")). //nolint:lll
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil,
						"a test disk", "NVME", "ZVOL", true, false, sqlmock.AnyArg(), "someDisk",
					).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
				mock.ExpectRollback()
			},
			diskExistsCacheDBFunc: func(*Disk) (bool, error) { return false, nil },
			diskValidateFunc:      func(*Disk) error { return nil },
			wantExists:            false,
			wantExistsErr:         false,
			wantCreateErr:         false,
			wantErr:               true,
		},
		{
			name: "badDbResRows",
			args: args{diskInst: &Disk{
				Name:        "someDisk",
				Description: "a test disk",
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
			}, size: "1g"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					diskDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta("INSERT INTO `disks` (`created_at`,`updated_at`,`deleted_at`,`description`,`type`,`dev_type`,`disk_cache`,`disk_direct`,`id`,`name`) VALUES (?,?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`")). //nolint:lll
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil,
						"a test disk", "NVME", "FILE", true, false, sqlmock.AnyArg(), "someDisk",
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))
				mock.ExpectCommit()
			},
			diskExistsCacheDBFunc: func(*Disk) (bool, error) { return false, nil },
			diskValidateFunc:      func(*Disk) error { return nil },
			wantExists:            false,
			wantExistsErr:         false,
			wantCreateErr:         false,
			wantErr:               true,
		},
		{
			name: "success",
			args: args{diskInst: &Disk{
				Name:        "someDisk",
				Description: "a test disk",
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
			}, size: "1g"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					diskDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta("INSERT INTO `disks` (`created_at`,`updated_at`,`deleted_at`,`description`,`type`,`dev_type`,`disk_cache`,`disk_direct`,`id`,`name`) VALUES (?,?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`")). //nolint:lll
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil,
						"a test disk", "NVME", "FILE", true, false, sqlmock.AnyArg(), "someDisk",
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow("c916ca6e-eb6b-400c-86ec-824b84ae71d3"))
				mock.ExpectCommit()
			},
			diskExistsCacheDBFunc: func(*Disk) (bool, error) { return false, nil },
			diskValidateFunc:      func(*Disk) error { return nil },
			wantExists:            false,
			wantExistsErr:         false,
			wantCreateErr:         false,
			wantErr:               false,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			diskExistsCacheDBFunc = testCase.diskExistsCacheDBFunc

			t.Cleanup(func() { diskExistsCacheDBFunc = diskExistsCacheDB })

			validateDiskFunc = testCase.diskValidateFunc

			t.Cleanup(func() { validateDiskFunc = validateDisk })

			ctrl := gomock.NewController(t)
			fileMock := NewMockFileInfoFetcher(ctrl)
			zfsMock := NewMockZfsVolInfoFetcher(ctrl)

			testDB, mockDB := cirrinadtest.NewMockDB("diskTest")

			testCase.mockClosure(testDB, mockDB)

			FileInfoFetcherImpl = fileMock

			t.Cleanup(func() { FileInfoFetcherImpl = FileInfoCmds{} })

			ZfsInfoFetcherImpl = zfsMock

			t.Cleanup(func() { ZfsInfoFetcherImpl = ZfsVolInfoCmds{} })

			var existsErr error

			var createErr error

			if testCase.wantExistsErr {
				existsErr = errors.New("bogus exists error") //nolint:goerr113
			}

			if testCase.wantCreateErr {
				createErr = errors.New("bogus create error") //nolint:goerr113
			}

			if testCase.args.diskInst.DevType == "FILE" {
				fileMock.EXPECT().CheckExists(gomock.Any()).MaxTimes(1).Return(testCase.wantExists, existsErr)
				fileMock.EXPECT().Add(gomock.Any(), gomock.Any()).MaxTimes(1).Return(createErr)
			}

			if testCase.args.diskInst.DevType == "ZVOL" {
				zfsMock.EXPECT().CheckExists(gomock.Any()).MaxTimes(1).Return(testCase.wantExists, existsErr)
				zfsMock.EXPECT().Add(gomock.Any(), gomock.Any()).MaxTimes(1).Return(createErr)
			}

			err := Create(testCase.args.diskInst, testCase.args.size)
			if (err != nil) != testCase.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, testCase.wantErr)
			}

			mockDB.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			if err = db.Close(); err != nil {
				t.Error(err)
			}

			if err = mockDB.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
