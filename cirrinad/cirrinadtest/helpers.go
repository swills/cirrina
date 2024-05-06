package cirrinadtest

import (
	"log"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewMockDB(testDSN string) (*gorm.DB, sqlmock.Sqlmock) {
	testDB, mock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("An error '%s' was not expected when opening a stub database connection", err)
	}

	mock.ExpectQuery("select sqlite_version()").
		WillReturnRows(sqlmock.NewRows([]string{"sqlite_version()"}).AddRow("3.40.1"))

	gormDB, err := gorm.Open(
		&sqlite.Dialector{
			DSN:  testDSN,
			Conn: testDB,
		},
		&gorm.Config{
			DisableAutomaticPing: true,
		},
	)
	if err != nil {
		log.Fatalf("An error '%s' was not expected when opening gorm database", err)
	}

	return gormDB, mock
}
