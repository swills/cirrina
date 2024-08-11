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
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE `isos`.`deleted_at` IS NULL"),
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
					ID:          "0ecf2f76-d421-4de9-8c55-ee57e0d3b15c",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					DeletedAt:   gorm.DeletedAt{},
					Name:        "FreeBSD-13.1-RELEASE-amd64-dvd1.iso",
					Description: "some description",
					Path:        "/bhyve/isos/FreeBSD-13.1-RELEASE-amd64-dvd1.iso",
					Size:        4621281280,
					Checksum:    "326c7a07a393972d3fcd47deaa08e2b932d9298d96e9b4f63a17a2730f93384abc5feb1f511436dc91fcc8b6f56ed25b43dc91d9cdfc700d2655f7e35420d494", //nolint:lll
				},
				{
					ID:          "ac7d8dc2-df5e-4643-8f2c-9e9064094932",
					CreatedAt:   createUpdateTime,
					UpdatedAt:   createUpdateTime,
					DeletedAt:   gorm.DeletedAt{},
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

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mock.ExpectationsWereMet()
			if err != nil {
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
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
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
				ID:          "ac7d8dc2-df5e-4643-8f2c-9e9064094932",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				DeletedAt:   gorm.DeletedAt{},
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
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
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
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			args:    args{id: "ac7d8dc2-df5e-4643-8f2c-9e9064094932"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "fail3",
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
				}
			},
			args:    args{id: ""},
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

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mock.ExpectationsWereMet()
			if err != nil {
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
			err := validateIso(testCase.args.isoInst)
			if (err != nil) != testCase.wantErr {
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
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
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
				ID:          "ac7d8dc2-df5e-4643-8f2c-9e9064094932",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				DeletedAt:   gorm.DeletedAt{},
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
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
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
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
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

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mock.ExpectationsWereMet()
			if err != nil {
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
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
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
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
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
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
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

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mock.ExpectationsWereMet()
			if err != nil {
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
		ID          string
		CreatedAt   time.Time
		UpdatedAt   time.Time
		DeletedAt   gorm.DeletedAt `gorm:"index"`
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

			got := i.GetPath()
			if got != testCase.want {
				t.Errorf("GetPath() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		isoID string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		wantErr     bool
	}{
		{
			name: "success1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("97737cc1-5890-4148-bf1f-948949b625c2").
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"name",
								"description",
								"size",
								"checksum",
							}).
							AddRow(
								"97737cc1-5890-4148-bf1f-948949b625c2",
								createUpdateTime,
								createUpdateTime,
								nil,
								"some.iso",
								"some test iso",
								123123123123,
								"3db0336f110c24cbf852d1b516888daa077e65ed43dcc7ab1ddf8c5782fed82221bc427e869f79c10e7b612db5b93692318307e6a3388fd2e201ae84e59bdea3", //nolint:lll
							))
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `isos` WHERE `isos`.`id` = ?"),
				).
					WithArgs("97737cc1-5890-4148-bf1f-948949b625c2").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			args:    args{isoID: "97737cc1-5890-4148-bf1f-948949b625c2"},
			wantErr: false,
		},
		{
			name: "fail1",
			mockClosure: func(_ *gorm.DB, _ sqlmock.Sqlmock) {
			},
			args:    args{isoID: ""},
			wantErr: true,
		},
		{
			name: "fail2",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			args:    args{isoID: "97737cc1-5890-4148-bf1f-948949b625c2"},
			wantErr: true,
		},
		{
			name: "fail3",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE id = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("97737cc1-5890-4148-bf1f-948949b625c2").
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"name",
								"description",
								"size",
								"checksum",
							}).
							AddRow(
								"97737cc1-5890-4148-bf1f-948949b625c2",
								createUpdateTime,
								createUpdateTime,
								nil,
								"some.iso",
								"some test iso",
								123123123123,
								"3db0336f110c24cbf852d1b516888daa077e65ed43dcc7ab1ddf8c5782fed82221bc427e869f79c10e7b612db5b93692318307e6a3388fd2e201ae84e59bdea3", //nolint:lll
							))
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `isos` WHERE `isos`.`id` = ?"),
				).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			args:    args{isoID: "97737cc1-5890-4148-bf1f-948949b625c2"},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("diskTest")
			testCase.mockClosure(testDB, mock)

			err := Delete(testCase.args.isoID)

			if (err != nil) != testCase.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, testCase.wantErr)
			}

			mock.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mock.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestISO_Save(t *testing.T) {
	createUpdateTime := time.Now()

	type fields struct {
		ID          string
		CreatedAt   time.Time
		UpdatedAt   time.Time
		DeletedAt   gorm.DeletedAt `gorm:"index"`
		Name        string
		Description string
		Path        string
		Size        uint64
		Checksum    string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields      fields
		wantErr     bool
	}{
		{
			name: "success1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `isos` SET `checksum`=?,`description`=?,`name`=?,`path`=?,`size`=?,`updated_at`=? WHERE `isos`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs(
						"garbage",
						"random iso",
						"some.iso",
						"/some/path/some.iso",
						32768,
						sqlmock.AnyArg(),
						"8c6c9326-bd5f-4c39-a5ec-562bb73391a3",
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:          "8c6c9326-bd5f-4c39-a5ec-562bb73391a3",
				Name:        "some.iso",
				Description: "random iso",
				Path:        "/some/path/some.iso",
				Size:        32768,
				Checksum:    "garbage",
			},
			wantErr: false,
		},
		{
			name: "fail1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `isos` SET `checksum`=?,`description`=?,`name`=?,`path`=?,`size`=?,`updated_at`=? WHERE `isos`.`deleted_at` IS NULL AND `id` = ?", //nolint:lll
					),
				).
					WithArgs(
						"garbage",
						"random iso",
						"some.iso",
						"/some/path/some.iso",
						32768,
						sqlmock.AnyArg(),
						"8c6c9326-bd5f-4c39-a5ec-562bb73391a3",
					).
					WillReturnError(gorm.ErrInvalidField)
				mock.ExpectRollback()
			},
			fields: fields{
				ID:          "8c6c9326-bd5f-4c39-a5ec-562bb73391a3",
				Name:        "some.iso",
				Description: "random iso",
				Path:        "/some/path/some.iso",
				Size:        32768,
				Checksum:    "garbage",
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("isoTest")
			testCase.mockClosure(testDB, mock)

			iso := &ISO{
				ID:          testCase.fields.ID,
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				DeletedAt:   gorm.DeletedAt{},
				Name:        testCase.fields.Name,
				Description: testCase.fields.Description,
				Path:        testCase.fields.Path,
				Size:        testCase.fields.Size,
				Checksum:    testCase.fields.Checksum,
			}

			err := iso.Save()
			if (err != nil) != testCase.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, testCase.wantErr)
			}

			mock.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mock.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

//nolint:maintidx
func TestCreate(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		isoInst *ISO
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		wantPathErr bool
		wantPath    bool
		wantErr     bool
	}{
		{
			name: "fail1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE name = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("some.iso").
					WillReturnError(gorm.ErrInvalidField)
			},
			args: args{
				isoInst: &ISO{
					ID:          "40b149c2-edf7-4bf4-873f-1f5ed74e49f6",
					Name:        "some.iso",
					Description: "a random iso",
					Path:        "/some/path/some.iso",
					Size:        32768,
					Checksum:    "garbage",
				},
			},
			wantErr: true,
		},
		{
			name: "fail2",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE name = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("some.iso").
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
			args: args{
				isoInst: &ISO{
					ID:          "40b149c2-edf7-4bf4-873f-1f5ed74e49f6",
					Name:        "some.iso",
					Description: "a random iso",
					Path:        "/some/path/some.iso",
					Size:        32768,
					Checksum:    "garbage",
				},
			},
			wantErr: true,
		},
		{
			name: "fail3",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE name = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("some.iso").
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
							},
						),
					)
			},
			args: args{
				isoInst: &ISO{
					ID:          "40b149c2-edf7-4bf4-873f-1f5ed74e49f6",
					Name:        "some.iso",
					Description: "a random iso",
					Path:        "/some/path/some.iso",
					Size:        32768,
					Checksum:    "garbage",
				},
			},
			wantPath:    true,
			wantPathErr: true,
			wantErr:     true,
		},
		{
			name: "fail4",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE name = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("some.iso").
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
							},
						),
					)
			},
			args: args{
				isoInst: &ISO{
					ID:          "40b149c2-edf7-4bf4-873f-1f5ed74e49f6",
					Name:        "some.iso",
					Description: "a random iso",
					Path:        "/some/path/some.iso",
					Size:        32768,
					Checksum:    "garbage",
				},
			},
			wantPath:    true,
			wantPathErr: false,
			wantErr:     true,
		},
		{
			name: "fail5",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE name = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("some&bad.iso").
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
							},
						),
					)
			},
			args: args{
				isoInst: &ISO{
					ID:          "40b149c2-edf7-4bf4-873f-1f5ed74e49f6",
					Name:        "some&bad.iso",
					Description: "a random iso",
					Path:        "/some/path/some.iso",
					Size:        32768,
					Checksum:    "garbage",
				},
			},
			wantPath:    false,
			wantPathErr: false,
			wantErr:     true,
		},
		{
			name: "fail6",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE name = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("some.iso").
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
								"config_id",
							},
						),
					)
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `isos` (`created_at`,`updated_at`,`deleted_at`,`description`,`size`,`checksum`,`id`,`name`,`path`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`,`path`", //nolint:lll
					),
				).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						nil,
						"a random iso",
						32768,
						"garbage",
						sqlmock.AnyArg(),
						"some.iso",
						"/some/path/some.iso",
					).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned

				mock.ExpectRollback()
			},
			args: args{
				isoInst: &ISO{
					Name:        "some.iso",
					Description: "a random iso",
					Path:        "/some/path/some.iso",
					Size:        32768,
					Checksum:    "garbage",
				},
			},
			wantPath:    false,
			wantPathErr: false,
			wantErr:     true,
		},
		{
			name: "fail7",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE name = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("some.iso").
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
							},
						),
					)
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `isos` (`created_at`,`updated_at`,`deleted_at`,`description`,`size`,`checksum`,`id`,`name`,`path`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`,`path`", //nolint:lll
					),
				).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						nil,
						"a random iso",
						32768,
						"garbage",
						sqlmock.AnyArg(),
						"some.iso",
						"/some/path/some.iso",
					).
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"name",
								"path",
								"size",
								"checksum",
								"config_id",
							}),
					)
				mock.ExpectCommit()
			},
			args: args{
				isoInst: &ISO{
					Name:        "some.iso",
					Description: "a random iso",
					Path:        "/some/path/some.iso",
					Size:        32768,
					Checksum:    "garbage",
				},
			},
			wantPath:    false,
			wantPathErr: false,
			wantErr:     true,
		},
		{
			name: "success1",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &Singleton{ // prevents parallel testing
					ISODB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `isos` WHERE name = ? AND `isos`.`deleted_at` IS NULL LIMIT 1"),
				).
					WithArgs("some.iso").
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
								"config_id",
							},
						),
					)
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta("INSERT INTO `isos` (`created_at`,`updated_at`,`deleted_at`,`description`,`size`,`checksum`,`id`,`name`,`path`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`,`name`,`path`"), //nolint:lll
				).
					WithArgs(
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						nil,
						"a random iso",
						32768,
						"garbage",
						sqlmock.AnyArg(),
						"some.iso",
						"/some/path/some.iso",
					).
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"name",
								"path",
								"size",
								"checksum",
								"config_id",
							}).
							AddRow(
								"a24e5161-5800-4ed4-95cf-c774b9c5bbd6",
								"some.iso",
								"/some/path/some.iso",
								"1234",
								"abc123",
								"5ebdb5c0-262c-4c70-ad7a-1c48478d5b52",
							),
					)
				mock.ExpectCommit()
			},
			args: args{
				isoInst: &ISO{
					Name:        "some.iso",
					Description: "a random iso",
					Path:        "/some/path/some.iso",
					Size:        32768,
					Checksum:    "garbage",
				},
			},
			wantPath:    false,
			wantPathErr: false,
			wantErr:     false,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("isoTest")
			testCase.mockClosure(testDB, mock)

			pathExistsFunc = func(_ string) (bool, error) {
				if testCase.wantPathErr {
					return true, errors.New("another error") //nolint:goerr113
				}

				if testCase.wantPath {
					return true, nil
				}

				return false, nil
			}

			err := Create(testCase.args.isoInst)
			if (err != nil) != testCase.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, testCase.wantErr)
			}

			mock.ExpectClose()

			db, err := testDB.DB()
			if err != nil {
				t.Error(err)
			}

			err = db.Close()
			if err != nil {
				t.Error(err)
			}

			err = mock.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
