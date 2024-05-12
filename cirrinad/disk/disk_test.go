package disk

import (
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-test/deep"
	"gorm.io/gorm"

	"cirrina/cirrinad/cirrinadtest"
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
					Model: gorm.Model{
						ID:        0,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
						DeletedAt: gorm.DeletedAt{},
					},
					ID:          "20d3098f-7ccf-484e-bed4-757940a3c775",
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
					Model: gorm.Model{
						ID:        0,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
						DeletedAt: gorm.DeletedAt{},
					},
					ID:          "41ae49ee-6e7e-47c2-aebb-671f2dbac4a2",
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
				t.Errorf("diskTypeValid() = %v, want %v", got, tt.want)
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
				t.Errorf("diskDevTypeValid() = %v, want %v", got, tt.want)
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
				t.Errorf("GetByName() got = %v, want %v", got, testCase.want)
			}
		})
	}
}
