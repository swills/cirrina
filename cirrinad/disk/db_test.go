package disk

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestDisk_BeforeCreate(t *testing.T) {
	type fields struct {
		ID          string
		CreatedAt   time.Time
		UpdatedAt   time.Time
		DeletedAt   gorm.DeletedAt `gorm:"index"`
		Name        string
		Description string
		Type        string
		DevType     string
		DiskCache   sql.NullBool
		DiskDirect  sql.NullBool
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "SuccessIDNotSet",
			fields: fields{
				ID:        "",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
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
			wantErr: false,
		},
		{
			name: "SuccessIDJunk",
			fields: fields{
				ID:        "asdfasdfasdf",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
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
			wantErr: false,
		},
		{
			name: "SuccessIDSet",
			fields: fields{
				ID:        "ab6d066c-bf9b-4ff8-b5ba-023fc846b8e4",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
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
			wantErr: false,
		},
		{
			name: "SuccessIDWrongFormat",
			fields: fields{
				ID:        "3b47565fce714ec9819b40df75b074f4",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
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
			wantErr: false,
		},
		{
			name: "FailNameNotSet",
			fields: fields{
				ID:        "5dcf2bc4-4d6d-419f-8eb5-8b1812f59ade",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
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
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testDisk := &Disk{
				ID:          testCase.fields.ID,
				Name:        testCase.fields.Name,
				Description: testCase.fields.Description,
				Type:        testCase.fields.Type,
				DevType:     testCase.fields.DevType,
				DiskCache:   testCase.fields.DiskCache,
				DiskDirect:  testCase.fields.DiskDirect,
			}

			err := testDisk.BeforeCreate(nil)
			if (err != nil) != testCase.wantErr {
				t.Errorf("BeforeCreate() error = %v, wantErr %v", err, testCase.wantErr)
			}

			if testCase.wantErr {
				return
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

func TestDisk_BeforeCreateNilReceiver(t *testing.T) {
	t.Parallel()

	t.Run("NilReceiver", func(t *testing.T) {
		t.Parallel()

		testDisk := (*Disk)(nil)

		err := testDisk.BeforeCreate(nil)
		if err == nil {
			t.Errorf("BeforeCreate() nil receiver did not return error, error = %v", err)
		}
	})
}
