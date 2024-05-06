package iso

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-test/deep"
	"gorm.io/gorm"

	"cirrina/cirrinad/cirrinadtest"
)

func TestGetAll(t *testing.T) {
	createUpdateTime := time.Now()

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        []*ISO
	}{
		{
			name: "testGetAllIsos",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					isoDB: testDB,
				}
				mock.ExpectQuery("^SELECT \\* FROM `isos` WHERE `isos`.`deleted_at` IS NULL$").
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"name",
								"description",
								"path",
								"size",
								"checksum",
							}).
							AddRow(
								"0ecf2f76-d421-4de9-8c55-ee57e0d3b15c",
								createUpdateTime,
								createUpdateTime,
								nil,
								"FreeBSD-13.1-RELEASE-amd64-dvd1.iso",
								"some description",
								"/bhyve/isos/FreeBSD-13.1-RELEASE-amd64-dvd1.iso",
								4621281280,
								"326c7a07a393972d3fcd47deaa08e2b932d9298d96e9b4f63a17a2730f93384abc5feb1f511436dc91fcc8b6f56ed25b43dc91d9cdfc700d2655f7e35420d494", //nolint:lll
							).
							AddRow(
								"ac7d8dc2-df5e-4643-8f2c-9e9064094932",
								createUpdateTime,
								createUpdateTime,
								nil,
								"FreeBSD-13.1-RELEASE-amd64-disc1.iso",
								"some description",
								"/bhyve/isos/FreeBSD-13.1-RELEASE-amd64-disc1.iso",
								1047048192,
								"259e034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3e3b630ba2", //nolint:lll
							),
					)
			},
			want: []*ISO{
				{
					Model: gorm.Model{
						ID:        0,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
						DeletedAt: gorm.DeletedAt{},
					},
					ID:          "0ecf2f76-d421-4de9-8c55-ee57e0d3b15c",
					Name:        "FreeBSD-13.1-RELEASE-amd64-dvd1.iso",
					Description: "some description",
					Path:        "/bhyve/isos/FreeBSD-13.1-RELEASE-amd64-dvd1.iso",
					Size:        4621281280,
					Checksum:    "326c7a07a393972d3fcd47deaa08e2b932d9298d96e9b4f63a17a2730f93384abc5feb1f511436dc91fcc8b6f56ed25b43dc91d9cdfc700d2655f7e35420d494", //nolint:lll
				},
				{
					Model: gorm.Model{
						ID:        0,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
						DeletedAt: gorm.DeletedAt{},
					},
					ID:          "ac7d8dc2-df5e-4643-8f2c-9e9064094932",
					Name:        "FreeBSD-13.1-RELEASE-amd64-disc1.iso",
					Description: "some description",
					Path:        "/bhyve/isos/FreeBSD-13.1-RELEASE-amd64-disc1.iso",
					Size:        1047048192,
					Checksum:    "259e034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3e3b630ba2", //nolint:lll
				},
			},
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("isoTest")
			testCase.mockClosure(testDB, mock)

			got := GetAll()

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
