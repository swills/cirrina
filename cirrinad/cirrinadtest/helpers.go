package cirrinadtest

import (
	"log"
	"os"

	"github.com/DATA-DOG/go-sqlmock"
	exec "golang.org/x/sys/execabs"
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

// IsTestEnv returns the env is in testing or not
func IsTestEnv() bool {
	return os.Getenv("GO_WANT_HELPER_PROCESS") == "1"
}

// MakeFakeCommand returns the fake exec.Command() function for testing
func MakeFakeCommand(mockFuncName string) func(command string, args ...string) *exec.Cmd {
	return func(command string, args ...string) *exec.Cmd {
		mockArg := "-test.run=" + mockFuncName
		cs := append([]string{mockArg, "--", command}, args...) // -test.run means the self mock function
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "GO_WANT_HELPER_PROCESS=1")

		return cmd
	}
}
