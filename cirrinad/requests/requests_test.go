package requests

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-test/deep"
	"gorm.io/gorm"

	"cirrina/cirrinad/cirrinadtest"
)

func TestGetByID(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		id string
	}

	tests := []struct {
		name        string
		args        args
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        Request
		wantErr     bool
	}{
		{
			name: "testRequestsGetByIDSuccess",
			args: args{id: "4aecbcd1-c39c-48e6-9a45-4a1abe06821f"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectQuery(
					"^SELECT \\* FROM `requests` WHERE `requests`.`id` = \\? AND `requests`.`deleted_at` IS NULL LIMIT 1$"). //nolint:lll
					WithArgs("4aecbcd1-c39c-48e6-9a45-4a1abe06821f").
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"started_at",
								"successful",
								"complete",
								"type",
								"data",
							}).
							AddRow(
								"4aecbcd1-c39c-48e6-9a45-4a1abe06821f",
								createUpdateTime,
								createUpdateTime,
								nil,
								sql.NullTime{
									Time:  createUpdateTime,
									Valid: true,
								},
								1,
								1,
								"VMSTART",
								"{\"vm_id\":\"49bd57aa-611e-4cf4-a7b7-2e71470c9aeb\"}",
							),
					)
			},
			want: Request{
				Model: gorm.Model{
					ID:        0,
					CreatedAt: createUpdateTime,
					UpdatedAt: createUpdateTime,
					DeletedAt: gorm.DeletedAt{},
				},
				ID: "4aecbcd1-c39c-48e6-9a45-4a1abe06821f",
				StartedAt: sql.NullTime{
					Time:  createUpdateTime,
					Valid: true,
				},
				Successful: true,
				Complete:   true,
				Type:       "VMSTART",
				Data:       "{\"vm_id\":\"49bd57aa-611e-4cf4-a7b7-2e71470c9aeb\"}",
			},
			wantErr: false,
		},
		{
			name: "testGetByID_error",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectQuery(
					"^SELECT \\* FROM `requests` WHERE `requests`.`id` = \\? AND `requests`.`deleted_at` IS NULL LIMIT 1$"). //nolint:lll
					WithArgs("cd48e86e-8b1a-4870-b1ec-337d1f1df37d").
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
			},
			args:    args{id: "cd48e86e-8b1a-4870-b1ec-337d1f1df37d"},
			want:    Request{},
			wantErr: true,
		},
		{
			name: "testRequestsGetByIDNotFound",
			args: args{id: "db945c03-c8f5-4c5d-91ec-da826646d227"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectQuery(
					"^SELECT \\* FROM `requests` WHERE `requests`.`id` = \\? AND `requests`.`deleted_at` IS NULL LIMIT 1$"). //nolint:lll
					WithArgs("db945c03-c8f5-4c5d-91ec-da826646d227").
					WillReturnRows(
						sqlmock.NewRows(
							[]string{
								"id",
								"created_at",
								"updated_at",
								"deleted_at",
								"started_at",
								"successful",
								"complete",
								"type",
								"data",
							},
						),
					)
			},
			want:    Request{},
			wantErr: true,
		},
		{
			name: "testRequestsGetByIDEmpty",
			args: args{id: ""},
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
			},
			want:    Request{},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("requestTest")
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
