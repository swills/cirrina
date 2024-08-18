package iso

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestISO_BeforeCreate(t *testing.T) {
	type fields struct {
		ID          string
		CreatedAt   time.Time
		UpdatedAt   time.Time
		DeletedAt   gorm.DeletedAt
		Name        string
		Description string
		Path        string
		Size        uint64
		Checksum    string
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
				Name:        "test.iso",
				Description: "a test iso",
				Path:        "/some/path/which/is/unused/by/the/test.iso",
				Size:        782712837,
				Checksum:    "someJunk",
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
				Name:        "test.iso",
				Description: "a test iso",
				Path:        "/some/path/which/is/unused/by/the/test.iso",
				Size:        782712837,
				Checksum:    "someJunk",
			},
			wantErr: false,
		},
		{
			name: "SuccessIDWrongFormat",
			fields: fields{
				ID:        "b5d3d9d05bf94329b28fe1a5297d3e65",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "test.iso",
				Description: "a test iso",
				Path:        "/some/path/which/is/unused/by/the/test.iso",
				Size:        782712837,
				Checksum:    "someJunk",
			},
			wantErr: false,
		},
		{
			name: "SuccessIDSet",
			fields: fields{
				ID:        "edcc7f49-8aba-444d-837a-218f10312c96",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "test.iso",
				Description: "a test iso",
				Path:        "/some/path/which/is/unused/by/the/test.iso",
				Size:        782712837,
				Checksum:    "someJunk",
			},
			wantErr: false,
		},
		{
			name: "FailNameNotSet",
			fields: fields{
				ID:        "edcc7f49-8aba-444d-837a-218f10312c96",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "",
				Description: "a test iso",
				Path:        "/some/path/which/is/unused/by/the/test.iso",
				Size:        782712837,
				Checksum:    "someJunk",
			},
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			testISO := &ISO{
				ID:          testCase.fields.ID,
				CreatedAt:   testCase.fields.CreatedAt,
				UpdatedAt:   testCase.fields.UpdatedAt,
				DeletedAt:   testCase.fields.DeletedAt,
				Name:        testCase.fields.Name,
				Description: testCase.fields.Description,
				Path:        testCase.fields.Path,
				Size:        testCase.fields.Size,
				Checksum:    testCase.fields.Checksum,
			}

			err := testISO.BeforeCreate(nil)

			if (err != nil) != testCase.wantErr {
				t.Errorf("BeforeCreate() error = %v, wantErr %v", err, testCase.wantErr)
			}

			if testCase.wantErr {
				return
			}

			if testISO.ID == "" {
				t.Fatalf("ID empty after create")
			}

			_, err = uuid.Parse(testISO.ID)
			if err != nil {
				t.Fatalf("error parsing uuid: %s", err.Error())
			}
		})
	}
}

func TestISO_BeforeCreateNilReceiver(t *testing.T) {
	t.Parallel()

	t.Run("NilReceiver", func(t *testing.T) {
		t.Parallel()

		testISO := (*ISO)(nil)

		err := testISO.BeforeCreate(nil)
		if err == nil {
			t.Errorf("BeforeCreate() nil receiver did not return error, error = %v", err)
		}
	})
}
