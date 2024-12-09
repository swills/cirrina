package disk

import (
	"database/sql"
	"errors"
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
	"cirrina/cirrinad/util"
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
				Instance = &Singleton{ // prevents parallel testing
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
							).
							AddRow(
								"41ae49ee-6e7e-47c2-aebb-671f2dbac4a2",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2023061001_15",
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
					Name:        "test2023061001_14",
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
					Name:        "test2023061001_15",
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
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(t.Name())
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
			got := diskTypeValid(tt.args.diskType)
			if got != tt.want {
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
			got := diskDevTypeValid(tt.args.diskDevType)
			if got != tt.want {
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

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// ensure no leftover values from other tests
			config.Config.Disk.VM.Path.Zpool = ""
			// clear out list(s) from other parallel test runs
			List.DiskList = map[string]*Disk{}

			err := testCase.args.diskInst.validate()
			if (err != nil) != testCase.wantErr {
				t.Errorf("validateDisk() error = %v, wantErr %v", err, testCase.wantErr)
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
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(t.Name())

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

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mock.ExpectationsWereMet()
			if err != nil {
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
			// clear out list(s) from other parallel test runs
			List.DiskList = map[string]*Disk{}

			testCase.mockClosure()

			got, err := GetByName(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetByName() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
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
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()
			d := &Disk{
				Name:    testCase.fields.Name,
				DevType: testCase.fields.DevType,
			}

			got := d.GetPath()
			if got != testCase.want {
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
				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
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
				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
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
				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
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
				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
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
				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
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
				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
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
				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
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
				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
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
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mock)

			testDisk := &Disk{
				ID:          testCase.testDisk.ID,
				Name:        testCase.testDisk.Name,
				Description: testCase.testDisk.Description,
				Type:        testCase.testDisk.Type,
				DevType:     testCase.testDisk.DevType,
				DiskCache:   testCase.testDisk.DiskCache,
				DiskDirect:  testCase.testDisk.DiskDirect,
			}

			err := testDisk.Save()
			if (err != nil) != testCase.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, testCase.wantErr)
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

func TestDelete(t *testing.T) {
	type args struct {
		diskInst *Disk
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

				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT vm_id,disk_id,position FROM `vm_disks` WHERE disk_id LIKE ? LIMIT 1"),
				).
					WithArgs("e89be82f-25c7-42b9-823a-df432e64320e").
					WillReturnRows(sqlmock.NewRows([]string{"vm_id", "disk_id", "position"}))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `disks` WHERE `disks`.`id` = ?"),
				).
					WithArgs("e89be82f-25c7-42b9-823a-df432e64320e").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args: args{
				diskInst: &Disk{ID: "e89be82f-25c7-42b9-823a-df432e64320e"},
			},
			wantErr: false,
		},
		{
			name: "DiskIsInUse",
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

				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT vm_id,disk_id,position FROM `vm_disks` WHERE disk_id LIKE ? LIMIT 1"),
				).
					WithArgs("e89be82f-25c7-42b9-823a-df432e64320e").
					WillReturnRows(
						sqlmock.NewRows([]string{"vm_id", "disk_id", "position"}).
							AddRow(
								"438e601f-fceb-4a4e-bde3-5e7ec45f4d08",
								"e89be82f-25c7-42b9-823a-df432e64320e",
								0,
							))
			},
			args: args{
				diskInst: &Disk{ID: "e89be82f-25c7-42b9-823a-df432e64320e"},
			},
			wantErr: true,
		},
		{
			name: "error1",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
			},
			args: args{
				diskInst: &Disk{ID: ""},
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

				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
				}
			},
			args: args{
				diskInst: &Disk{ID: "2e1f3582-7bc1-4a7a-beeb-1651e371435e"},
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

				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT vm_id,disk_id,position FROM `vm_disks` WHERE disk_id LIKE ? LIMIT 1"),
				).
					WithArgs("e89be82f-25c7-42b9-823a-df432e64320e").
					WillReturnRows(sqlmock.NewRows([]string{"vm_id", "disk_id", "position"}))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `disks` WHERE `disks`.`id` = ?"),
				).
					WithArgs("e89be82f-25c7-42b9-823a-df432e64320e").
					WillReturnResult(sqlmock.NewResult(1, 0))
				mock.ExpectCommit()
			},
			args: args{
				diskInst: &Disk{ID: "e89be82f-25c7-42b9-823a-df432e64320e"},
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mock)

			err := testCase.args.diskInst.Delete()
			if (err != nil) != testCase.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, testCase.wantErr)
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

func Test_diskExists(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		diskInst *Disk
	}

	tests := []struct {
		name          string
		args          args
		getByNameFunc func(string) (*Disk, error)
		mockClosure   func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want          bool
		wantErr       bool
	}{
		{
			name: "GetByNameErr",
			args: args{&Disk{
				Name: "someDisk",
			}},
			getByNameFunc: func(_ string) (*Disk, error) {
				return nil, errors.New("some bogus error") //nolint:goerr113
			},
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
			},
			want:    true,
			wantErr: true,
		},
		{
			name: "GetByNameOK",
			args: args{&Disk{
				Name: "someDisk",
			}},
			getByNameFunc: func(_ string) (*Disk, error) {
				return &Disk{Name: "someDisk"}, nil
			},
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "NotFoundDB",
			args: args{&Disk{
				Name: "someDisk",
			}},
			getByNameFunc: func(_ string) (*Disk, error) {
				return nil, errDiskNotFound
			},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
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
							).
							AddRow(
								"41ae49ee-6e7e-47c2-aebb-671f2dbac4a2",
								createUpdateTime,
								createUpdateTime,
								nil,
								"test2023061001_15",
								"another virtual hard disk image",
								"NVME",
								"FILE",
								1,
								0,
							),
					)
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "FoundDB",
			args: args{&Disk{
				Name: "someDisk",
			}},
			getByNameFunc: func(_ string) (*Disk, error) {
				return nil, errDiskNotFound
			},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
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
							).
							AddRow(
								"41ae49ee-6e7e-47c2-aebb-671f2dbac4a2",
								createUpdateTime,
								createUpdateTime,
								nil,
								"someDisk",
								"another virtual hard disk image",
								"NVME",
								"FILE",
								1,
								0,
							),
					)
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mock)

			GetByNameFunc = testCase.getByNameFunc

			t.Cleanup(func() { GetByNameFunc = GetByName })

			got, err := diskExistsCacheDB(testCase.args.diskInst)
			if (err != nil) != testCase.wantErr {
				t.Errorf("diskExistsCacheDB() error = %v, wantErr %v", err, testCase.wantErr)

				return
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

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
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
		diskExistsCacheDBFunc func(d *Disk) (bool, error)
		diskValidateFunc      func(d *Disk) error
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
				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
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
				config.Config.Disk.VM.Path.Zpool = "someBogusZpool"
				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
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
				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
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
				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
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
		t.Run(testCase.name, func(t *testing.T) {
			diskExistsCacheDBFunc = testCase.diskExistsCacheDBFunc

			t.Cleanup(func() { diskExistsCacheDBFunc = diskExistsCacheDB })

			ctrl := gomock.NewController(t)
			fileMock := NewMockFileInfoFetcher(ctrl)
			zfsMock := NewMockZfsVolInfoFetcher(ctrl)

			testDB, mockDB := cirrinadtest.NewMockDB(t.Name())

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

func TestDisk_VerifyExists(t *testing.T) {
	type fields struct {
		ID          string
		CreatedAt   time.Time
		UpdatedAt   time.Time
		DeletedAt   gorm.DeletedAt
		Name        string
		Description string
		Type        string
		DevType     string
		DiskCache   sql.NullBool
		DiskDirect  sql.NullBool
	}

	tests := []struct {
		name        string
		mockClosure func()
		fields      fields
		want        bool
		wantErr     bool
		wantPath    bool
		wantPathErr bool
	}{
		{
			name: "FileOk",
			mockClosure: func() {
				config.Config.Disk.VM.Path.Image = "/some/path"
			},
			fields: fields{
				DevType: "FILE",
			},
			want:        true,
			wantErr:     false,
			wantPath:    true,
			wantPathErr: false,
		},
		{
			name: "ZVOLOk",
			mockClosure: func() {
				config.Config.Disk.VM.Path.Image = "/some/path"
			},
			fields: fields{
				DevType: "ZVOL",
			},
			want:        true,
			wantErr:     false,
			wantPath:    true,
			wantPathErr: false,
		},
		{
			name: "FileErr",
			mockClosure: func() {
				config.Config.Disk.VM.Path.Image = "/some/path"
			},
			fields: fields{
				DevType: "FILE",
			},
			want:        true,
			wantErr:     true,
			wantPath:    true,
			wantPathErr: true,
		},
		{
			name: "ZVOLErr",
			mockClosure: func() {
				config.Config.Disk.VM.Path.Image = "/some/path"
			},
			fields: fields{
				DevType: "ZVOL",
			},
			want:        true,
			wantErr:     true,
			wantPath:    true,
			wantPathErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()

			PathExistsFunc = func(_ string) (bool, error) {
				if testCase.wantPathErr {
					return true, errors.New("another error") //nolint:goerr113
				}

				if testCase.wantPath {
					return true, nil
				}

				return false, nil
			}

			t.Cleanup(func() { PathExistsFunc = util.PathExists })

			testDisk := &Disk{
				DevType: testCase.fields.DevType,
			}

			got, err := testDisk.VerifyExists()
			if (err != nil) != testCase.wantErr {
				t.Errorf("VerifyExists() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("VerifyExists() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestDisk_initOneDisk(t *testing.T) {
	createUpdateTime := time.Now()

	type fields struct {
		ID          string
		CreatedAt   time.Time
		UpdatedAt   time.Time
		Name        string
		Description string
		Type        string
		DevType     string
		DiskCache   sql.NullBool
		DiskDirect  sql.NullBool
	}

	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "Success",
			fields: fields{
				ID:          "73b99544-6950-4456-b762-fc940b79018e",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081102_hd0",
				Description: "test disk",
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
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(_ *testing.T) {
			testDisk := &Disk{
				ID:          testCase.fields.ID,
				CreatedAt:   testCase.fields.CreatedAt,
				UpdatedAt:   testCase.fields.UpdatedAt,
				Name:        testCase.fields.Name,
				Description: testCase.fields.Description,
				Type:        testCase.fields.Type,
				DevType:     testCase.fields.DevType,
				DiskCache:   testCase.fields.DiskCache,
				DiskDirect:  testCase.fields.DiskDirect,
			}
			testDisk.initOneDisk()
		})
	}
}

func TestDisk_initOneDiskNil(t *testing.T) {
	var testDisk *Disk

	t.Run("initOneDiskNil", func(_ *testing.T) {
		testDisk.initOneDisk()
	})
}

func TestDisk_InUse(t *testing.T) {
	type fields struct {
		ID          string
		CreatedAt   time.Time
		UpdatedAt   time.Time
		DeletedAt   gorm.DeletedAt
		Name        string
		Description string
		Type        string
		DevType     string
		DiskCache   sql.NullBool
		DiskDirect  sql.NullBool
	}

	tests := []struct {
		name        string
		fields      fields
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        bool
	}{
		{
			name: "DiskUsed",
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

				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT vm_id,disk_id,position FROM `vm_disks` WHERE disk_id LIKE ? LIMIT 1",
					),
				).
					WithArgs("9685398e-4d72-4585-b76f-d5bba4efe2d2").
					WillReturnRows(
						sqlmock.NewRows([]string{"vm_id", "disk_id", "position"}).
							AddRow(
								"b5c59d40-9c9f-418d-916e-dc36168b1775",
								"9685398e-4d72-4585-b76f-d5bba4efe2d2",
								0,
							),
					)
			},
			fields: fields{ID: "9685398e-4d72-4585-b76f-d5bba4efe2d2"},
			want:   true,
		},
		{
			name: "DiskNotUsed",
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

				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT vm_id,disk_id,position FROM `vm_disks` WHERE disk_id LIKE ? LIMIT 1",
					),
				).
					WithArgs("17cc455c-5193-4627-939a-3123880620b9").
					WillReturnRows(
						sqlmock.NewRows([]string{"vm_id", "disk_id", "position"}),
					)
			},
			fields: fields{ID: "17cc455c-5193-4627-939a-3123880620b9"},
			want:   false,
		},
		{
			name: "ErrFailSafe",
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

				Instance = &Singleton{ // prevents parallel testing
					DiskDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT vm_id,disk_id,position FROM `vm_disks` WHERE disk_id LIKE ? LIMIT 1",
					),
				).
					WithArgs("6a6e8abc-bbe0-43f9-9327-182f07ae57e2").
					WillReturnError(gorm.ErrInvalidData)
			},
			fields: fields{ID: "6a6e8abc-bbe0-43f9-9327-182f07ae57e2"},
			want:   true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDisk := &Disk{
				ID:          testCase.fields.ID,
				CreatedAt:   testCase.fields.CreatedAt,
				UpdatedAt:   testCase.fields.UpdatedAt,
				DeletedAt:   testCase.fields.DeletedAt,
				Name:        testCase.fields.Name,
				Description: testCase.fields.Description,
				Type:        testCase.fields.Type,
				DevType:     testCase.fields.DevType,
				DiskCache:   testCase.fields.DiskCache,
				DiskDirect:  testCase.fields.DiskDirect,
			}

			testDB, mock := cirrinadtest.NewMockDB(t.Name())
			testCase.mockClosure(testDB, mock)

			got := testDisk.InUse()
			if got != testCase.want {
				t.Errorf("InUse() = %v, want %v", got, testCase.want)
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
