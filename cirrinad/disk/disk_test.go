package disk

import (
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
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

			if got := GetAllDB(); !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("GetAllDB() = %v, want %v", got, testCase.want)
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
