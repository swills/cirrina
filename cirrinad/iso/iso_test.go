package iso

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-test/deep"
	"gorm.io/gorm"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/config"
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

func TestGetByID(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		id string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *ISO
		wantErr     bool
	}{
		{
			name: "Success1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					isoDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
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
								"path",
								"size",
								"checksum",
							}).
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
			args: args{id: "ac7d8dc2-df5e-4643-8f2c-9e9064094932"},
			want: &ISO{
				Model: gorm.Model{
					CreatedAt: createUpdateTime,
					UpdatedAt: createUpdateTime,
				},
				ID:          "ac7d8dc2-df5e-4643-8f2c-9e9064094932",
				Name:        "FreeBSD-13.1-RELEASE-amd64-disc1.iso",
				Description: "some description",
				Path:        "/bhyve/isos/FreeBSD-13.1-RELEASE-amd64-disc1.iso",
				Size:        1047048192,
				Checksum:    "259e034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3e3b630ba2", //nolint:lll
			},
			wantErr: false,
		},
		{
			name: "fail1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					isoDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
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
								"path",
								"size",
								"checksum",
							}),
					)
			},
			args:    args{id: "ac7d8dc2-df5e-4643-8f2c-9e9064094932"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "fail2",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					isoDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			args:    args{id: "ac7d8dc2-df5e-4643-8f2c-9e9064094932"},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("isoTest")
			testCase.mockClosure(testDB, mock)

			got, err := GetByID(testCase.args.id)
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

func Test_validateIso(t *testing.T) {
	type args struct {
		isoInst *ISO
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success1",
			args: args{&ISO{
				Name: "asdfasdf",
			}},
			wantErr: false,
		},
		{
			name: "fail1",
			args: args{&ISO{
				Name: "asdfasd;f",
			}},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if err := validateIso(testCase.args.isoInst); (err != nil) != testCase.wantErr {
				t.Errorf("validateIso() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestGetByName(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		name string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        *ISO
		wantErr     bool
	}{
		{
			name: "success1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					isoDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE name = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
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
								"path",
								"size",
								"checksum",
							}).
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
			args: args{name: "FreeBSD-13.1-RELEASE-amd64-disc1.iso"},
			want: &ISO{
				Model: gorm.Model{
					CreatedAt: createUpdateTime,
					UpdatedAt: createUpdateTime,
				},
				ID:          "ac7d8dc2-df5e-4643-8f2c-9e9064094932",
				Name:        "FreeBSD-13.1-RELEASE-amd64-disc1.iso",
				Description: "some description",
				Path:        "/bhyve/isos/FreeBSD-13.1-RELEASE-amd64-disc1.iso",
				Size:        1047048192,
				Checksum:    "259e034731c1493740a5a9f2933716c479746360f570312ea44ed9b7b59ed9131284c5f9fe8db13f8f4e10f312033db1447ff2900d65bfefbf5cfb3e3b630ba2", //nolint:lll
			},
			wantErr: false,
		},
		{
			name: "fail1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					isoDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE name = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			args:    args{name: "FreeBSD-13.1-RELEASE-amd64-disc1.iso"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "fail2",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					isoDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE name = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
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
								"path",
								"size",
								"checksum",
							}),
					)
			},
			args:    args{name: "FreeBSD-13.1-RELEASE-amd64-disc1.iso"},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("isoTest")
			testCase.mockClosure(testDB, mock)

			got, err := GetByName(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("GetByName() error = %v, wantErr %v", err, testCase.wantErr)

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

func Test_isoExistsDB(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		isoName string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		wantPathErr bool
		wantPath    bool
		want        bool
		wantErr     bool
	}{
		{
			name: "fail1",
			args: args{"someIso.iso"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					isoDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE name = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
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
								"path",
								"size",
								"checksum",
							}),
					)
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "fail2",
			args: args{"someIso.iso"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					isoDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE name = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			want:    true,
			wantErr: true,
		},
		{
			name: "success",
			args: args{"FreeBSD-13.1-RELEASE-amd64-disc1.iso"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					isoDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE name = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
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
								"path",
								"size",
								"checksum",
							}).
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
			want:    true,
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("isoTest")
			testCase.mockClosure(testDB, mock)

			got, err := isoExistsDB(testCase.args.isoName)
			if (err != nil) != testCase.wantErr {
				t.Errorf("isoExistsDB() error = %v, wantErr %v", err, testCase.wantErr)

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

func Test_isoExistsFS(t *testing.T) {
	type args struct {
		name string
	}

	tests := []struct {
		name        string
		args        args
		wantPath    bool
		wantPathErr bool
		want        bool
		wantErr     bool
	}{
		{
			name:        "fail1",
			args:        args{name: "someIso.iso"},
			wantPath:    true,
			wantPathErr: true,
			want:        true,
			wantErr:     true,
		},
		{
			name:        "fail2",
			args:        args{name: "someIso.iso"},
			wantPath:    true,
			wantPathErr: false,
			want:        true,
			wantErr:     false,
		},
		{
			name:        "success1",
			args:        args{name: "someIso.iso"},
			wantPath:    false,
			wantPathErr: false,
			want:        false,
			wantErr:     false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			pathExistsFunc = func(_ string) (bool, error) {
				if testCase.wantPathErr {
					return true, errors.New("another error") //nolint:goerr113
				}

				if testCase.wantPath {
					return true, nil
				}

				return false, nil
			}

			got, err := isoExistsFS(testCase.args.name)
			if (err != nil) != testCase.wantErr {
				t.Errorf("isoExistsFS() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("isoExistsFS() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestISO_GetPath(t *testing.T) {
	type fields struct {
		Model       gorm.Model
		ID          string
		Name        string
		Description string
		Path        string
		Size        uint64
		Checksum    string
	}

	tests := []struct {
		name        string
		mockClosure func()
		fields      fields
		want        string
	}{
		{
			name: "valid1",
			mockClosure: func() {
				config.Config.Disk.VM.Path.Iso = "/some/path"
			},
			fields: fields{
				Name: "some.iso",
			},
			want: "/some/path/some.iso",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()
			i := &ISO{
				Name: testCase.fields.Name,
			}

			if got := i.GetPath(); got != testCase.want {
				t.Errorf("GetPath() = %v, want %v", got, testCase.want)
			}
		})
	}
}
