package vm

import (
	"database/sql"
	"testing"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/disk"
)

//nolint:paralleltest
func Test_diskAttached(t *testing.T) {
	type args struct {
		aDisk  string
		thisVM *VM
	}

	tests := []struct {
		name        string
		mockClosure func()
		args        args
		want        bool
	}{
		{
			name: "Fail1",
			mockClosure: func() {
				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
			},
			args: args{
				aDisk:  "someDisk",
				thisVM: nil,
			},
			want: false,
		},
		{
			name: "Fail2",
			mockClosure: func() {
				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
			},
			args: args{
				aDisk: "",
				thisVM: &VM{
					ID:          "f143252e-9eb2-43c6-b1c6-8f2d274474a2",
					Name:        "someTestVM",
					Description: "test Vm of the day",
					Status:      "STOPPED",
				},
			},
			want: false,
		},
		{
			name: "Success1",
			mockClosure: func() {
				testVM := VM{
					ID:          "f143252e-9eb2-43c6-b1c6-8f2d274474a2",
					Name:        "someTestVM",
					Description: "test Vm of the day",
					Status:      "STOPPED",
					Disks: []*disk.Disk{
						{
							ID: "b9c51f58-ef3a-425b-80ab-7f67486c0931",
						},
					},
				}

				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM
			},
			args: args{
				aDisk: "b9c51f58-ef3a-425b-80ab-7f67486c0931",
				thisVM: &VM{
					ID:          "a7c48313-de26-472d-a7aa-38f19a7aa794",
					Name:        "someTestVM",
					Description: "test Vm of the day",
					Status:      "STOPPED",
					Disks: []*disk.Disk{
						{
							ID: "b9c51f58-ef3a-425b-80ab-7f67486c0931",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Success2",
			mockClosure: func() {
				testVM := VM{
					ID:          "f143252e-9eb2-43c6-b1c6-8f2d274474a2",
					Name:        "someTestVM",
					Description: "test Vm of the day",
					Status:      "STOPPED",
					Disks:       []*disk.Disk{nil},
				}

				// clear out list from other parallel test runs
				List.VMList = map[string]*VM{}
				List.VMList[testVM.ID] = &testVM
			},
			args: args{
				aDisk: "b9c51f58-ef3a-425b-80ab-7f67486c0931",
				thisVM: &VM{
					ID:          "a7c48313-de26-472d-a7aa-38f19a7aa794",
					Name:        "someTestVM",
					Description: "test Vm of the day",
					Status:      "STOPPED",
					Disks: []*disk.Disk{
						{
							ID: "b9c51f58-ef3a-425b-80ab-7f67486c0931",
						},
					},
				},
			},
			want: false,
		},
	}

	//nolint:paralleltest
	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockClosure()

			got := diskAttached(testCase.args.aDisk, testCase.args.thisVM)
			if got != testCase.want {
				t.Errorf("diskAttached() = %v, want %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func Test_validateDisks(t *testing.T) {
	type args struct {
		diskids []string
		thisVM  *VM
	}

	tests := []struct {
		name        string
		mockClosure func()
		args        args
		wantErr     bool
	}{
		{
			name:        "Empty",
			mockClosure: func() {},
			args:        args{diskids: []string{}, thisVM: &VM{}},
			wantErr:     false,
		},
		{
			name:        "BadUUID",
			mockClosure: func() {},
			args:        args{diskids: []string{"80acc7c8-b55d-415"}, thisVM: &VM{}},
			wantErr:     true,
		},
		{
			name: "EmptyVM",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args:    args{diskids: []string{"80acc7c8-b55d-415c-8a9d-2b02608a4894"}, thisVM: &VM{}},
			wantErr: true,
		},
		{
			name: "EmptyDiskName",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args:    args{diskids: []string{"0d4a0338-0b68-4645-b99d-9cbb30df272d"}, thisVM: &VM{}},
			wantErr: true,
		},
		{
			name: "DiskNotInUse",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args:    args{diskids: []string{"0d4a0338-0b68-4645-b99d-9cbb30df272d"}, thisVM: &VM{}},
			wantErr: false,
		},
		{
			name: "DiskDupe",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			args: args{
				diskids: []string{
					"0d4a0338-0b68-4645-b99d-9cbb30df272d",
					"0d4a0338-0b68-4645-b99d-9cbb30df272d",
				},
				thisVM: &VM{},
			},
			wantErr: true,
		},
		{
			name: "DiskAlreadyInUse",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "0d4a0338-0b68-4645-b99d-9cbb30df272d",
					Name:        "aDisk",
					Description: "a description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst

				testVM := VM{
					ID:          "f143252e-9eb2-43c6-b1c6-8f2d274474a2",
					Name:        "someTestVM",
					Description: "test Vm of the day",
					Status:      "STOPPED",
					Disks: []*disk.Disk{
						{
							ID: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
						},
					},
				}
				List.VMList[testVM.ID] = &testVM
			},
			args: args{
				diskids: []string{"0d4a0338-0b68-4645-b99d-9cbb30df272d"},
				thisVM: &VM{
					ID:          "22a719c6-a4e6-4824-88c2-de5b946e228c",
					Name:        "notTheSame",
					Description: "a completely different VM",
					Status:      "STOPPED",
					Disks: []*disk.Disk{
						{
							ID: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Normal",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "7091c957-3720-4d41-804b-25b443e60cb8",
					Name:        "aNewDisk",
					Description: "a new description",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst

				testVM := VM{
					ID:          "f143252e-9eb2-43c6-b1c6-8f2d274474a2",
					Name:        "someTestVM",
					Description: "test Vm of the day",
					Status:      "STOPPED",
					Disks: []*disk.Disk{
						{
							ID: "0d4a0338-0b68-4645-b99d-9cbb30df272d",
						},
					},
				}
				List.VMList[testVM.ID] = &testVM
			},
			args: args{
				diskids: []string{"7091c957-3720-4d41-804b-25b443e60cb8"},
				thisVM: &VM{
					ID:          "22a719c6-a4e6-4824-88c2-de5b946e228c",
					Name:        "notTheSame",
					Description: "a completely different VM",
					Status:      "STOPPED",
					Disks: []*disk.Disk{
						{
							ID: "cf5b91af-0f24-4991-a5d7-8a21c5c483d8",
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("diskTest")

			testCase.mockClosure()

			err := validateDisks(testCase.args.diskids, testCase.args.thisVM)
			if (err != nil) != testCase.wantErr {
				t.Errorf("validateDisks() error = %v, wantErr %v", err, testCase.wantErr)
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
