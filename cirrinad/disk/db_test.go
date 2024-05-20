package disk

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"cirrina/cirrinad/cirrinadtest"
)

//nolint:paralleltest
func TestDisk_BeforeCreate(t *testing.T) {
	testDB, _ := cirrinadtest.NewMockDB("requestTest")

	type fields struct {
		Model       gorm.Model
		ID          string
		Name        string
		Description string
		Type        string
		DevType     string
		DiskCache   sql.NullBool
		DiskDirect  sql.NullBool
	}

	type args struct {
		in0 *gorm.DB
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "success1",
			fields: fields{
				Model: gorm.Model{
					ID:        0,
					CreatedAt: time.Time{},
					UpdatedAt: time.Time{},
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
				},
				ID:          "",
				Name:        "someDisk",
				Description: "a good disk",
				Type:        "NVME",
				DevType:     "FILE",
				DiskCache: sql.NullBool{
					Bool:  false,
					Valid: true,
				},
				DiskDirect: sql.NullBool{
					Bool:  false,
					Valid: true,
				},
			},
			args:    args{in0: testDB},
			wantErr: false,
		},
		{
			name: "fail1",
			fields: fields{
				Model: gorm.Model{
					ID:        0,
					CreatedAt: time.Time{},
					UpdatedAt: time.Time{},
					DeletedAt: gorm.DeletedAt{
						Time:  time.Time{},
						Valid: false,
					},
				},
				ID:          "",
				Name:        "",
				Description: "a good disk",
				Type:        "NVME",
				DevType:     "FILE",
				DiskCache: sql.NullBool{
					Bool:  false,
					Valid: true,
				},
				DiskDirect: sql.NullBool{
					Bool:  false,
					Valid: true,
				},
			},
			args:    args{in0: testDB},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture

		t.Run(testCase.name, func(t *testing.T) {
			testDisk := &Disk{
				Model:       testCase.fields.Model,
				ID:          testCase.fields.ID,
				Name:        testCase.fields.Name,
				Description: testCase.fields.Description,
				Type:        testCase.fields.Type,
				DevType:     testCase.fields.DevType,
				DiskCache:   testCase.fields.DiskCache,
				DiskDirect:  testCase.fields.DiskDirect,
			}

			if testDisk.ID != "" {
				t.Error("test bug, uuid is not empty before test")
			}

			err := testDisk.BeforeCreate(testCase.args.in0)
			if (err != nil) != testCase.wantErr {
				t.Errorf("BeforeCreate() error = %v, wantErr %v", err, testCase.wantErr)
			}

			if testDisk.ID == "" {
				t.Fatalf("ID empty after create")
			}

			_, err = uuid.Parse(testDisk.ID)
			if err != nil {
				t.Fatalf("error parsing uuid: %s", err.Error())
			}
		})
	}
}
