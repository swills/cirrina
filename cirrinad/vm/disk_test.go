package vm

import (
	"testing"

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
