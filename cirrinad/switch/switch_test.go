package vmswitch

import (
	"database/sql"
	"log"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetAll(t *testing.T) {
	testDB, mock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("An error '%s' was not expected when opening a stub database connection", err)
	}
	defer func(db *sql.DB) {
		_ = db.Close()
	}(testDB)

	mock.ExpectQuery("select sqlite_version()").
		WillReturnRows(sqlmock.NewRows([]string{"sqlite_version()"}).AddRow("3.40.1"))

	gormDB, err := gorm.Open(&sqlite.Dialector{DSN: "testDB", Conn: testDB}, &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Errorf("An error '%s' was not expected when opening gorm database", err)
	}

	instance = &singleton{
		switchDB: gormDB,
	}
	createUpdateTime := time.Now()

	mock.ExpectQuery("^SELECT \\* FROM `switches` WHERE `switches`.`deleted_at` IS NULL$").
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
					"uplink",
				}).
				AddRow(
					"0cb98661-6470-432d-8fa4-5eca3668b494",
					createUpdateTime,
					createUpdateTime,
					nil,
					"bridge0",
					"some if switch description",
					"IF",
					"em1",
				).
				AddRow(
					"76290cc3-7143-4c0b-980f-25f74b12673f",
					createUpdateTime,
					createUpdateTime,
					nil,
					"bnet0",
					"some ng switch description",
					"NG",
					"em0",
				),
		)

	tests := []struct {
		name string
		want []*Switch
	}{
		{
			name: "testGetAllSwitches",
			want: []*Switch{
				{
					Model: gorm.Model{
						ID:        0,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
						DeletedAt: gorm.DeletedAt{},
					},
					ID:          "0cb98661-6470-432d-8fa4-5eca3668b494",
					Name:        "bridge0",
					Description: "some if switch description",
					Type:        "IF",
					Uplink:      "em1",
				},
				{
					Model: gorm.Model{
						ID:        0,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
						DeletedAt: gorm.DeletedAt{},
					},
					ID:          "76290cc3-7143-4c0b-980f-25f74b12673f",
					Name:        "bnet0",
					Description: "some ng switch description",
					Type:        "NG",
					Uplink:      "em0",
				},
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if got := GetAll(); !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("GetAll() = %v, want %v", got, testCase.want)
			}
		})
	}
}
