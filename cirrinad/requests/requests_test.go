package requests

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-test/deep"
	"github.com/mattn/go-sqlite3"
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
				ID:        "4aecbcd1-c39c-48e6-9a45-4a1abe06821f",
				CreatedAt: createUpdateTime,
				UpdatedAt: createUpdateTime,
				DeletedAt: gorm.DeletedAt{},
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

func TestCreateVMReq(t *testing.T) { //nolint:maintidx
	type args struct {
		requestType reqType
		vmID        string
	}

	tests := []struct {
		name         string
		args         args
		mockClosure  func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want         Request
		wantErr      bool
		checkErrType bool
		wantErrType  error
	}{
		{
			name: "testRequestVMStartSuccess",
			args: args{requestType: VMSTART, vmID: "f2d857d8-7625-47da-9545-e339f0468856"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `requests` (`created_at`,`updated_at`,`deleted_at`,`started_at`,`successful`,`complete`,`type`,`data`,`id`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`")). //nolint:lll
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, false,
						false, "VMSTART", "{\"vm_id\":\"f2d857d8-7625-47da-9545-e339f0468856\"}", sqlmock.AnyArg(),
					).
					// gorm asks the db to return the id but does not check that it matches what gorm set it
					// to, so we can fake it and return any value we like
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow("f2943275-2b6d-48a0-9e85-7ee6baa64c37"))
				mock.ExpectCommit()
			},
			want: Request{
				ID:         "f2943275-2b6d-48a0-9e85-7ee6baa64c37",
				CreatedAt:  time.Time{},
				UpdatedAt:  time.Time{},
				DeletedAt:  gorm.DeletedAt{},
				StartedAt:  sql.NullTime{},
				Successful: false,
				Complete:   false,
				Type:       "VMSTART",
				Data:       "{\"vm_id\":\"f2d857d8-7625-47da-9545-e339f0468856\"}",
			},
			wantErr: false,
		},
		{
			name: "testRequestVMStopSuccess",
			args: args{requestType: VMSTOP, vmID: "f2d857d8-7625-47da-9545-e339f0468856"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `requests` (`created_at`,`updated_at`,`deleted_at`,`started_at`,`successful`,`complete`,`type`,`data`,`id`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`")). //nolint:lll
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, false, false,
						"VMSTOP", "{\"vm_id\":\"f2d857d8-7625-47da-9545-e339f0468856\"}", sqlmock.AnyArg()).
					// gorm asks the db to return the id but does not check that it matches what gorm set it
					// to, so we can fake it and return any value we like
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow("f2943275-2b6d-48a0-9e85-7ee6baa64c37"))
				mock.ExpectCommit()
			},
			want: Request{
				ID:         "f2943275-2b6d-48a0-9e85-7ee6baa64c37",
				CreatedAt:  time.Time{},
				UpdatedAt:  time.Time{},
				DeletedAt:  gorm.DeletedAt{},
				StartedAt:  sql.NullTime{},
				Successful: false,
				Complete:   false,
				Type:       "VMSTOP",
				Data:       "{\"vm_id\":\"f2d857d8-7625-47da-9545-e339f0468856\"}",
			},
			wantErr: false,
		},
		{
			name: "testRequestVMDeleteSuccess",
			args: args{requestType: VMDELETE, vmID: "f2d857d8-7625-47da-9545-e339f0468856"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `requests` (`created_at`,`updated_at`,`deleted_at`,`started_at`,`successful`,`complete`,`type`,`data`,`id`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`")). //nolint:lll
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, false, false,
						"VMDELETE", "{\"vm_id\":\"f2d857d8-7625-47da-9545-e339f0468856\"}", sqlmock.AnyArg()).
					// gorm asks the db to return the id but does not check that it matches what gorm set it
					// to, so we can fake it and return any value we like
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow("f2943275-2b6d-48a0-9e85-7ee6baa64c37"))
				mock.ExpectCommit()
			},
			want: Request{
				ID:         "f2943275-2b6d-48a0-9e85-7ee6baa64c37",
				CreatedAt:  time.Time{},
				UpdatedAt:  time.Time{},
				DeletedAt:  gorm.DeletedAt{},
				StartedAt:  sql.NullTime{},
				Successful: false,
				Complete:   false,
				Type:       "VMDELETE",
				Data:       "{\"vm_id\":\"f2d857d8-7625-47da-9545-e339f0468856\"}",
			},
			wantErr: false,
		},
		{
			name: "testtRequestVMStartError",
			args: args{requestType: VMSTART, vmID: "f2d857d8-7625-47da-9545-e339f0468856"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `requests` (`created_at`,`updated_at`,`deleted_at`,`started_at`,`successful`,`complete`,`type`,`data`,`id`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`")). //nolint:lll
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, false, false,
										"VMSTART", "{\"vm_id\":\"f2d857d8-7625-47da-9545-e339f0468856\"}", sqlmock.AnyArg()).
					WillReturnError(gorm.ErrInvalidField) // does not matter what error is returned
				mock.ExpectRollback()
			},
			want:    Request{},
			wantErr: true,
		},
		{
			name: "testRequestVMStartSuccessWrongNumberOfRows",
			args: args{requestType: VMSTART, vmID: "f2d857d8-7625-47da-9545-e339f0468856"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `requests` (`created_at`,`updated_at`,`deleted_at`,`started_at`,`successful`,`complete`,`type`,`data`,`id`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`")). //nolint:lll
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, false, false,
						"VMSTART", "{\"vm_id\":\"f2d857d8-7625-47da-9545-e339f0468856\"}", sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))
				mock.ExpectCommit()
			},
			want:    Request{},
			wantErr: true,
		},
		{
			name: "testRequestEmptyVMID",
			args: args{requestType: VMSTART, vmID: ""},
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
			},
			want:    Request{},
			wantErr: true,
		},
		{
			name: "testRequestErrorBadType",
			args: args{requestType: "blah", vmID: "3d245c57-a68e-41d9-adfa-a365d91f20eb"},
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
			},
			want:         Request{},
			wantErr:      true,
			checkErrType: true,
			wantErrType:  errInvalidRequest,
		},
		{
			name: "testRequestVMStartDupe",
			args: args{requestType: VMSTART, vmID: "f2d857d8-7625-47da-9545-e339f0468856"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `requests` (`created_at`,`updated_at`,`deleted_at`,`started_at`,`successful`,`complete`,`type`,`data`,`id`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`")). //nolint:lll
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, false,
						false, "VMSTART", "{\"vm_id\":\"f2d857d8-7625-47da-9545-e339f0468856\"}", sqlmock.AnyArg(),
					).
					WillReturnError(sqlite3.ErrConstraintUnique)
				mock.ExpectRollback()
			},
			want:    Request{},
			wantErr: true,
		},
		{
			name: "testRequestInvalidVMID",
			args: args{vmID: "somegarbage"},
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

			got, err := CreateVMReq(testCase.args.requestType, testCase.args.vmID)
			if (err != nil) != testCase.wantErr {
				t.Errorf("CreateVMReq() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if testCase.wantErr && testCase.checkErrType {
				if err == nil || !errors.Is(err, testCase.wantErrType) {
					t.Errorf("error type was wrong, expected %s, got %s", errInvalidRequest, err)
				}
			}

			// zero out the time since we know it's going to vary and don't care
			got.CreatedAt = time.Time{}
			got.UpdatedAt = time.Time{}

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

func Test_validVMReqType(t *testing.T) {
	type args struct {
		aReqType reqType
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "validVMReqTypeVMStart",
			args: args{aReqType: VMSTART},
			want: true,
		},
		{
			name: "validVMReqTypeVMStop",
			args: args{aReqType: VMSTOP},
			want: true,
		},
		{
			name: "validVMReqTypeVMStart",
			args: args{aReqType: VMDELETE},
			want: true,
		},
		{
			name: "validVMReqTypeVMStart",
			args: args{aReqType: NICCLONE},
			want: false,
		},
		{
			name: "validVMReqTypeVMStart",
			args: args{aReqType: "somegarbage"},
			want: false,
		},
		{
			name: "validVMReqTypeVMStart",
			args: args{aReqType: VMSTART},
			want: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := validVMReqType(testCase.args.aReqType); got != testCase.want {
				t.Errorf("validVMReqType() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestCreateNicCloneReq(t *testing.T) {
	type args struct {
		nicID   string
		newName string
	}

	tests := []struct {
		name        string
		args        args
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        Request
		wantErr     bool
	}{
		{
			name: "testRequestNICCloneSuccess",
			args: args{nicID: "f2d857d8-7625-47da-9545-e339f0468856", newName: "somenic"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `requests` (`created_at`,`updated_at`,`deleted_at`,`started_at`,`successful`,`complete`,`type`,`data`,`id`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`")). //nolint:lll
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, false,
						false, "NICCLONE", "{\"nic_id\":\"f2d857d8-7625-47da-9545-e339f0468856\",\"new_nic_name\":\"somenic\"}", sqlmock.AnyArg(), //nolint:lll
					).
					// gorm asks the db to return the id but does not check that it matches what gorm set it
					// to, so we can fake it and return any value we like
					WillReturnRows(sqlmock.NewRows([]string{"id"}).
						AddRow("f2943275-2b6d-48a0-9e85-7ee6baa64c37"))
				mock.ExpectCommit()
			},
			want: Request{
				ID:         "f2943275-2b6d-48a0-9e85-7ee6baa64c37",
				CreatedAt:  time.Time{},
				UpdatedAt:  time.Time{},
				DeletedAt:  gorm.DeletedAt{},
				StartedAt:  sql.NullTime{},
				Successful: false,
				Complete:   false,
				Type:       "NICCLONE",
				Data:       "{\"nic_id\":\"f2d857d8-7625-47da-9545-e339f0468856\",\"new_nic_name\":\"somenic\"}",
			},
			wantErr: false,
		},
		{
			name: "testRequestNICCloneEmptyNICID",
			args: args{nicID: "", newName: "somenic"},
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
			},
			want:    Request{},
			wantErr: true,
		},
		{
			name: "testRequestNICCloneEmptyNewNICName",
			args: args{nicID: "f2d857d8-7625-47da-9545-e339f0468856", newName: ""},
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
			},
			want:    Request{},
			wantErr: true,
		},
		{
			name: "testRequestNICCloneInvalidVMID",
			args: args{nicID: "moregarbage", newName: "somenic"},
			mockClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
			},
			want:    Request{},
			wantErr: true,
		},
		{name: "testRequestNICCloneWrongNumberOfRows",
			args: args{nicID: "f2d857d8-7625-47da-9545-e339f0468856", newName: "somenic"},
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"INSERT INTO `requests` (`created_at`,`updated_at`,`deleted_at`,`started_at`,`successful`,`complete`,`type`,`data`,`id`) VALUES (?,?,?,?,?,?,?,?,?) RETURNING `id`")). //nolint:lll
					WithArgs(
						sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, false,
						false, "NICCLONE", "{\"nic_id\":\"f2d857d8-7625-47da-9545-e339f0468856\",\"new_nic_name\":\"somenic\"}", sqlmock.AnyArg(), //nolint:lll
					).
					// gorm asks the db to return the id but does not check that it matches what gorm set it
					// to, so we can fake it and return any value we like
					WillReturnRows(sqlmock.NewRows([]string{"id"}))

				mock.ExpectCommit()
			},
			want:    Request{},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("requestTest")
			testCase.mockClosure(testDB, mock)
			got, err := CreateNicCloneReq(testCase.args.nicID, testCase.args.newName)

			if (err != nil) != testCase.wantErr {
				t.Errorf("CreateNicCloneReq() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}
			// zero out the time since we know it's going to vary and don't care
			got.CreatedAt = time.Time{}
			got.UpdatedAt = time.Time{}

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

func TestGetUnStarted(t *testing.T) {
	createUpdateTime := time.Now()

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		want        Request
	}{
		{
			name: "TestGetUnstartedSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `requests` WHERE started_at IS NULL AND `requests`.`deleted_at` IS NULL LIMIT 1")).                               //nolint:lll
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "started_at", "successful", "complete", "type", "data"}). //nolint:lll
																								AddRow("018f5b9e-91ce-7854-b614-22057b03558a", createUpdateTime, createUpdateTime, nil, nil, 0, 0, "VMSTART", "{\"vm_id\":\"f5b761a1-8193-4db3-a914-b37edc848d29\"}"), //nolint:lll
					)
			},
			want: Request{
				ID:         "018f5b9e-91ce-7854-b614-22057b03558a",
				CreatedAt:  createUpdateTime,
				UpdatedAt:  createUpdateTime,
				DeletedAt:  gorm.DeletedAt{},
				StartedAt:  sql.NullTime{},
				Successful: false,
				Complete:   false,
				Type:       VMSTART,
				Data:       "{\"vm_id\":\"f5b761a1-8193-4db3-a914-b37edc848d29\"}"},
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("requestTest")
			testCase.mockClosure(testDB, mock)

			got := GetUnStarted()

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

func TestRequest_Start(t *testing.T) {
	createUpdateTime := time.Now()

	type fields struct {
		ID         string
		CreatedAt  time.Time
		UpdatedAt  time.Time
		DeletedAt  gorm.DeletedAt
		StartedAt  sql.NullTime
		Successful bool
		Complete   bool
		Type       reqType
		Data       string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields      fields
	}{
		{
			name: "TestRequestStartSuccessful",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
																																"UPDATE `requests` SET `id`=?,`created_at`=?,`updated_at`=?,`started_at`=?,`type`=?,`data`=? WHERE `requests`.`deleted_at` IS NULL AND `id` = ?")). //nolint:lll
					WithArgs("7a691194-64e7-45d1-ae2a-a064271d7c24", createUpdateTime, sqlmock.AnyArg(), sqlmock.AnyArg(), "VMSTART", "{\"vm_id\":\"f5b761a1-8193-4db3-a914-b37edc848d29\"}", "7a691194-64e7-45d1-ae2a-a064271d7c24"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:        "7a691194-64e7-45d1-ae2a-a064271d7c24",
				CreatedAt: createUpdateTime,
				UpdatedAt: createUpdateTime,
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				StartedAt:  sql.NullTime{},
				Successful: false,
				Complete:   false,
				Type:       "VMSTART",
				Data:       "{\"vm_id\":\"f5b761a1-8193-4db3-a914-b37edc848d29\"}",
			},
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("requestTest")
			testCase.mockClosure(testDB, mock)

			req := &Request{
				ID:         testCase.fields.ID,
				CreatedAt:  testCase.fields.CreatedAt,
				UpdatedAt:  testCase.fields.UpdatedAt,
				DeletedAt:  testCase.fields.DeletedAt,
				StartedAt:  testCase.fields.StartedAt,
				Successful: testCase.fields.Successful,
				Complete:   testCase.fields.Complete,
				Type:       testCase.fields.Type,
				Data:       testCase.fields.Data,
			}

			req.Start()

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

func TestRequest_Succeeded(t *testing.T) {
	createUpdateTime := time.Now()

	type fields struct {
		ID         string
		CreatedAt  time.Time
		UpdatedAt  time.Time
		DeletedAt  gorm.DeletedAt
		StartedAt  sql.NullTime
		Successful bool
		Complete   bool
		Type       reqType
		Data       string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields      fields
	}{
		{
			name: "TestRequestSucceededSuccessful",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `requests` SET `updated_at`=?,`successful`=?,`complete`=? WHERE `requests`.`deleted_at` IS NULL AND `id` = ?")). //nolint:lll
					WithArgs(sqlmock.AnyArg(), true, true, "7a691194-64e7-45d1-ae2a-a064271d7c24").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:        "7a691194-64e7-45d1-ae2a-a064271d7c24",
				CreatedAt: createUpdateTime,
				UpdatedAt: createUpdateTime,
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				StartedAt:  sql.NullTime{},
				Successful: false,
				Complete:   false,
				Type:       "VMSTART",
				Data:       "{\"vm_id\":\"f5b761a1-8193-4db3-a914-b37edc848d29\"}",
			},
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("requestTest")
			testCase.mockClosure(testDB, mock)

			req := &Request{
				ID:         testCase.fields.ID,
				CreatedAt:  testCase.fields.CreatedAt,
				UpdatedAt:  testCase.fields.UpdatedAt,
				DeletedAt:  testCase.fields.DeletedAt,
				StartedAt:  testCase.fields.StartedAt,
				Successful: testCase.fields.Successful,
				Complete:   testCase.fields.Complete,
				Type:       testCase.fields.Type,
				Data:       testCase.fields.Data,
			}

			req.Succeeded()

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

func TestRequest_Failed(t *testing.T) {
	createUpdateTime := time.Now()

	type fields struct {
		ID         string
		CreatedAt  time.Time
		UpdatedAt  time.Time
		DeletedAt  gorm.DeletedAt
		StartedAt  sql.NullTime
		Successful bool
		Complete   bool
		Type       reqType
		Data       string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields      fields
	}{
		{
			name: "TestRequestFailedSuccessful",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `requests` SET `updated_at`=?,`complete`=? WHERE `requests`.`deleted_at` IS NULL AND `id` = ?")). //nolint:lll
					WithArgs(sqlmock.AnyArg(), true, "7a691194-64e7-45d1-ae2a-a064271d7c24").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:        "7a691194-64e7-45d1-ae2a-a064271d7c24",
				CreatedAt: createUpdateTime,
				UpdatedAt: createUpdateTime,
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				StartedAt:  sql.NullTime{},
				Successful: false,
				Complete:   false,
				Type:       "VMSTART",
				Data:       "{\"vm_id\":\"f5b761a1-8193-4db3-a914-b37edc848d29\"}",
			},
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("requestTest")
			testCase.mockClosure(testDB, mock)

			req := &Request{
				ID:         testCase.fields.ID,
				CreatedAt:  testCase.fields.CreatedAt,
				UpdatedAt:  testCase.fields.UpdatedAt,
				DeletedAt:  testCase.fields.DeletedAt,
				StartedAt:  testCase.fields.StartedAt,
				Successful: testCase.fields.Successful,
				Complete:   testCase.fields.Complete,
				Type:       testCase.fields.Type,
				Data:       testCase.fields.Data,
			}

			req.Failed()

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

func TestPendingReqExists(t *testing.T) { //nolint:maintidx
	createUpdateTime := time.Now()

	type args struct {
		objID string
	}

	tests := []struct {
		name        string
		mockClosure func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		args        args
		want        []string
	}{
		{
			name: "TestPendingReqExistsVMStartSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL")).
					WithArgs(false).
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
						).
							AddRow(
								"60284e9f-69c0-4db8-868d-7a8e24070025",
								createUpdateTime,
								createUpdateTime,
								gorm.DeletedAt{
									Time:  time.Time{},
									Valid: false,
								},
								sql.NullTime{
									Time:  createUpdateTime,
									Valid: true,
								},
								0,
								0,
								"VMSTART",
								"{\"vm_id\":\"a4d76261-c8e0-4310-8a26-c34819e939fa\"}",
							),
					)
			},
			args: args{objID: "a4d76261-c8e0-4310-8a26-c34819e939fa"},
			want: []string{"60284e9f-69c0-4db8-868d-7a8e24070025"},
		},
		{
			name: "TestPendingReqExistsVMStopSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL")).
					WithArgs(false).
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
						).
							AddRow(
								"60284e9f-69c0-4db8-868d-7a8e24070025",
								createUpdateTime,
								createUpdateTime,
								gorm.DeletedAt{
									Time:  time.Time{},
									Valid: false,
								},
								sql.NullTime{
									Time:  createUpdateTime,
									Valid: true,
								},
								0,
								0,
								"VMSTOP",
								"{\"vm_id\":\"a4d76261-c8e0-4310-8a26-c34819e939fa\"}",
							),
					)
			},
			args: args{objID: "a4d76261-c8e0-4310-8a26-c34819e939fa"},
			want: []string{"60284e9f-69c0-4db8-868d-7a8e24070025"},
		},
		{
			name: "TestPendingReqExistsVMDeleteSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL")).
					WithArgs(false).
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
						).
							AddRow(
								"60284e9f-69c0-4db8-868d-7a8e24070025",
								createUpdateTime,
								createUpdateTime,
								gorm.DeletedAt{
									Time:  time.Time{},
									Valid: false,
								},
								sql.NullTime{
									Time:  createUpdateTime,
									Valid: true,
								},
								0,
								0,
								"VMDELETE",
								"{\"vm_id\":\"a4d76261-c8e0-4310-8a26-c34819e939fa\"}",
							),
					)
			},
			args: args{objID: "a4d76261-c8e0-4310-8a26-c34819e939fa"},
			want: []string{"60284e9f-69c0-4db8-868d-7a8e24070025"},
		},
		{
			name: "TestPendingReqExistsVMStartBadJSON",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL")).
					WithArgs(false).
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
						).
							AddRow(
								"60284e9f-69c0-4db8-868d-7a8e24070025",
								createUpdateTime,
								createUpdateTime,
								gorm.DeletedAt{
									Time:  time.Time{},
									Valid: false,
								},
								sql.NullTime{
									Time:  createUpdateTime,
									Valid: true,
								},
								0,
								0,
								"VMSTART",
								"junk",
							),
					)
			},
			args: args{objID: "a4d76261-c8e0-4310-8a26-c34819e939fa"},
			want: nil,
		},
		{
			name: "TestPendingReqExistsNICCloneSuccess",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL")).
					WithArgs(false).
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
						).
							AddRow(
								"60284e9f-69c0-4db8-868d-7a8e24070025",
								createUpdateTime,
								createUpdateTime,
								gorm.DeletedAt{
									Time:  time.Time{},
									Valid: false,
								},
								sql.NullTime{
									Time:  createUpdateTime,
									Valid: true,
								},
								0,
								0,
								"NICCLONE",
								"{\"nic_id\":\"b0dd54ed-d905-4b7f-a5d9-9ace4c99c302\",\"new_nic_name\":\"test2023102401_int0_clone0\"}",
							),
					)
			},
			args: args{objID: "b0dd54ed-d905-4b7f-a5d9-9ace4c99c302"},
			want: []string{"60284e9f-69c0-4db8-868d-7a8e24070025"},
		},
		{
			name: "TestPendingReqExistsNICCloneBadJSON",
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				instance = &singleton{ // prevents parallel testing
					reqDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `requests` WHERE `complete` = ? AND `requests`.`deleted_at` IS NULL")).
					WithArgs(false).
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
						).
							AddRow(
								"60284e9f-69c0-4db8-868d-7a8e24070025",
								createUpdateTime,
								createUpdateTime,
								gorm.DeletedAt{
									Time:  time.Time{},
									Valid: false,
								},
								sql.NullTime{
									Time:  createUpdateTime,
									Valid: true,
								},
								0,
								0,
								"NICCLONE",
								"junk",
							),
					)
			},
			args: args{objID: "b0dd54ed-d905-4b7f-a5d9-9ace4c99c302"},
			want: nil,
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("requestTest")
			testCase.mockClosure(testDB, mock)

			got := PendingReqExists(testCase.args.objID)

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
