package vm

import (
	"testing"

	"github.com/go-test/deep"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/iso"
	"cirrina/cirrinad/util"
)

func TestVM_getKeyboardArg(t *testing.T) {
	type fields struct {
		Config Config
	}

	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "noScreen",
			fields: fields{
				Config: Config{
					Screen:    false,
					KbdLayout: "unused",
				},
			},
			want: []string{},
		},
		{
			name: "default",
			fields: fields{
				Config: Config{
					Screen:    true,
					KbdLayout: "default",
				},
			},
			want: []string{},
		},
		{
			name: "us_unix",
			fields: fields{
				Config: Config{
					Screen:    true,
					KbdLayout: "us_unix",
				},
			},
			want: []string{"-K", "us_unix"},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			vm := &VM{
				Config: testCase.fields.Config,
			}

			got := vm.getKeyboardArg()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestVM_getACPIArg(t *testing.T) {
	type fields struct {
		Config Config
	}

	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name:   "acpiNotSet",
			fields: fields{Config: Config{ACPI: false}},
			want:   []string{},
		},
		{
			name:   "acpiSet",
			fields: fields{Config: Config{ACPI: true}},
			want:   []string{"-A"},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			vm := &VM{
				Config: testCase.fields.Config,
			}

			got := vm.getACPIArg()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestVM_getDPOArg(t *testing.T) {
	type fields struct {
		Config Config
	}

	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name:   "dpoNotSet",
			fields: fields{Config: Config{DestroyPowerOff: false}},
			want:   []string{},
		},
		{
			name:   "dpoSet",
			fields: fields{Config: Config{DestroyPowerOff: true}},
			want:   []string{"-D"},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			vm := &VM{
				Config: testCase.fields.Config,
			}

			got := vm.getDPOArg()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestVM_getEOPArg(t *testing.T) {
	type fields struct {
		Config Config
	}

	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name:   "eopNotSet",
			fields: fields{Config: Config{ExitOnPause: false}},
			want:   []string{},
		},
		{
			name:   "eopSet",
			fields: fields{Config: Config{ExitOnPause: true}},
			want:   []string{"-P"},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			vm := &VM{
				Config: testCase.fields.Config,
			}

			got := vm.getEOPArg()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestVM_getHLTArg(t *testing.T) {
	type fields struct {
		Config Config
	}

	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name:   "hltNotSet",
			fields: fields{Config: Config{UseHLT: false}},
			want:   []string{},
		},
		{
			name:   "hltSet",
			fields: fields{Config: Config{UseHLT: true}},
			want:   []string{"-H"},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			vm := &VM{
				Config: testCase.fields.Config,
			}

			got := vm.getHLTArg()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestVM_getUTCArg(t *testing.T) {
	type fields struct {
		Config Config
	}

	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name:   "utcNotSet",
			fields: fields{Config: Config{UTCTime: false}},
			want:   []string{},
		},
		{
			name:   "utcSet",
			fields: fields{Config: Config{UTCTime: true}},
			want:   []string{"-u"},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			vm := &VM{
				Config: testCase.fields.Config,
			}

			got := vm.getUTCArg()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestVM_getMSRArg(t *testing.T) {
	type fields struct {
		Config Config
	}

	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name:   "msrNotSet",
			fields: fields{Config: Config{IgnoreUnknownMSR: false}},
			want:   []string{},
		},
		{
			name:   "msrSet",
			fields: fields{Config: Config{IgnoreUnknownMSR: true}},
			want:   []string{"-w"},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			vm := &VM{
				Config: testCase.fields.Config,
			}

			got := vm.getMSRArg()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

//nolint:paralleltest
func TestVM_getCPUArg(t *testing.T) {
	type fields struct {
		Config Config
	}

	tests := []struct {
		name                 string
		mockGetHostMaxVMCpus func() (uint16, error)
		fields               fields
		want                 []string
	}{
		{
			name:                 "oneCpus",
			mockGetHostMaxVMCpus: func() (uint16, error) { return 128, nil },
			fields: fields{
				Config: Config{CPU: 1},
			},
			want: []string{"-c", "1"},
		},
		{
			name:                 "twoCpus",
			mockGetHostMaxVMCpus: func() (uint16, error) { return 128, nil },
			fields: fields{
				Config: Config{CPU: 2},
			},
			want: []string{"-c", "2"},
		},
		{
			name:                 "fourCpus",
			mockGetHostMaxVMCpus: func() (uint16, error) { return 128, nil },
			fields: fields{
				Config: Config{CPU: 4},
			},
			want: []string{"-c", "4"},
		},
		{
			name:                 "eightCpus",
			mockGetHostMaxVMCpus: func() (uint16, error) { return 128, nil },
			fields: fields{
				Config: Config{CPU: 8},
			},
			want: []string{"-c", "8"},
		},
		{
			name:                 "sixteenCpus",
			mockGetHostMaxVMCpus: func() (uint16, error) { return 128, nil },
			fields: fields{
				Config: Config{CPU: 16},
			},
			want: []string{"-c", "16"},
		},
		{
			name:                 "thirtyTwoCpus",
			mockGetHostMaxVMCpus: func() (uint16, error) { return 32, nil },
			fields: fields{
				Config: Config{CPU: 32},
			},
			want: []string{"-c", "32"},
		},
		{
			name:                 "sixtyFourCpus",
			mockGetHostMaxVMCpus: func() (uint16, error) { return 32, nil },
			fields: fields{
				Config: Config{CPU: 64},
			},
			want: []string{"-c", "32"},
		},
		{
			name:                 "ifYouGotEmWeWillUseEm",
			mockGetHostMaxVMCpus: func() (uint16, error) { return 1024, nil },
			fields: fields{
				Config: Config{CPU: 1024},
			},
			want: []string{"-c", "1024"},
		},
		{
			name:                 "maxCpusErr",
			mockGetHostMaxVMCpus: func() (uint16, error) { return 0, util.ErrInvalidNumCPUs },
			fields: fields{
				Config: Config{CPU: 2},
			},
			want: []string{},
		},
		{
			name:                 "tooManyCpus",
			mockGetHostMaxVMCpus: func() (uint16, error) { return 4, nil },
			fields: fields{
				Config: Config{CPU: 5},
			},
			want: []string{"-c", "4"},
		},
		{
			name:                 "wayTooManyCpus",
			mockGetHostMaxVMCpus: func() (uint16, error) { return 16, nil },
			fields: fields{
				Config: Config{CPU: 65537},
			},
			want: []string{"-c", "16"},
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			getHostMaxVMCpusFunc = testCase.mockGetHostMaxVMCpus

			t.Cleanup(func() { getHostMaxVMCpusFunc = util.GetHostMaxVMCpus })

			util.GetHostMaxVMCpusFunc = testCase.mockGetHostMaxVMCpus

			t.Cleanup(func() { util.GetHostMaxVMCpusFunc = util.GetHostMaxVMCpus })

			vm := &VM{
				Config: testCase.fields.Config,
			}

			got := vm.getCPUArg()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestVM_getMemArg(t *testing.T) {
	type fields struct {
		Config Config
	}

	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name:   "1G",
			fields: fields{Config: Config{Mem: 1024}},
			want:   []string{"-m", "1024m"},
		},
		{
			name:   "16G",
			fields: fields{Config: Config{Mem: 16384}},
			want:   []string{"-m", "16384m"},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			vm := &VM{
				Config: testCase.fields.Config,
			}

			got := vm.getMemArg()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestVM_getLPCArg(t *testing.T) {
	type args struct {
		slot int
	}

	tests := []struct {
		name     string
		args     args
		want     []string
		wantSlot int
	}{
		{
			name: "theOnlyTestCasePossible",
			args: args{
				slot: 0, // does not matter
			},
			want:     []string{"-s", "31,lpc"},
			wantSlot: 0,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			vm := &VM{}

			got, gotSlot := vm.getLPCArg(testCase.args.slot)

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			diff = deep.Equal(gotSlot, testCase.wantSlot)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestVM_getROMArg(t *testing.T) {
	type fields struct {
		Name   string
		Config Config
	}

	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "noStore",
			fields: fields{
				Name:   "someTestVM",
				Config: Config{StoreUEFIVars: false},
			},
			want: []string{"-l", "bootrom,/usr/local/share/uefi-firmware/BHYVE_UEFI.fd"},
		},
		{
			name: "storeUEFIVars",
			fields: fields{
				Name:   "someTestVM",
				Config: Config{StoreUEFIVars: true},
			},
			want: []string{"-l", "bootrom,/usr/local/share/uefi-firmware/BHYVE_UEFI.fd,/var/tmp/cirrinad/state/someTestVM/BHYVE_UEFI_VARS.fd"}, //nolint:lll
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			config.Config.Rom.Path = "/usr/local/share/uefi-firmware/BHYVE_UEFI.fd"
			config.Config.Disk.VM.Path.State = "/var/tmp/cirrinad/state/"

			vm := &VM{
				Name:   testCase.fields.Name,
				Config: testCase.fields.Config,
			}

			got := vm.getROMArg()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestVM_getWireArg(t *testing.T) {
	type fields struct {
		Config Config
	}

	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name:   "msrNotSet",
			fields: fields{Config: Config{WireGuestMem: false}},
			want:   []string{},
		},
		{
			name:   "msrSet",
			fields: fields{Config: Config{WireGuestMem: true}},
			want:   []string{"-S"},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			vm := &VM{
				Config: testCase.fields.Config,
			}

			got := vm.getWireArg()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestVM_getExtraArg(t *testing.T) {
	type fields struct {
		Config Config
	}

	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "noExtraArgs",
			want: []string{},
		},
		{
			name:   "someExtraArgs",
			fields: fields{Config: Config{ExtraArgs: "-s blah foo"}},
			want:   []string{"-s", "blah", "foo"},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			vm := &VM{
				Config: testCase.fields.Config,
			}

			got := vm.getExtraArg()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestVM_getCDArg(t *testing.T) {
	type fields struct {
		ISOs []*iso.ISO
	}

	type args struct {
		slot int
	}

	tests := []struct {
		name     string
		fields   fields
		args     args
		wantArg  []string
		wantSlot int
	}{
		{
			name:     "noISOs",
			wantArg:  nil,
			wantSlot: 0,
		},
		{
			name: "nilISOItem",
			fields: fields{
				ISOs: []*iso.ISO{nil},
			},
			args:     args{},
			wantArg:  nil,
			wantSlot: 0,
		},
		{
			name: "emptyIsoPath",
			fields: fields{
				ISOs: []*iso.ISO{{
					Name:        "aBusted.iso",
					Description: "some busted iso instance",
					Path:        "",
					Size:        327680,
					Checksum:    "notUsedHere",
				}},
			},
			args:     args{slot: 4},
			wantArg:  []string{"-s", "4:0,ahci,cd:/the/config/path/for/isos/aBusted.iso"},
			wantSlot: 5,
		},
		{
			name: "oneISO",
			fields: fields{
				ISOs: []*iso.ISO{{
					Name:        "someTestThing.iso",
					Description: "a test iso",
					Path:        "/some/path/to/someTestThing.iso",
					Size:        292911919,
					Checksum:    "unusedHere",
				}},
			},
			args: args{
				slot: 2,
			},
			wantArg:  []string{"-s", "2:0,ahci,cd:/some/path/to/someTestThing.iso"},
			wantSlot: 3,
		},
		{
			name: "tooManyIsos",
			fields: fields{
				ISOs: []*iso.ISO{
					{
						Name:        "someTestThing.iso",
						Description: "a test iso",
						Path:        "/some/path/to/someTestThing.iso",
						Size:        292911919,
						Checksum:    "unusedHere",
					},
					{
						Name:        "anotherTestIso.iso",
						Description: "a test iso",
						Path:        "/some/path/to/anotherTestIso.iso",
						Size:        291413919,
						Checksum:    "unusedHere",
					},
					{
						Name:        "thirdTest.iso",
						Description: "a test iso",
						Path:        "/some/path/to/thirdTest.iso",
						Size:        291413111,
						Checksum:    "unusedHere",
					},
				},
			},
			args: args{
				slot: 29,
			},
			wantArg: []string{
				"-s", "29:0,ahci,cd:/some/path/to/someTestThing.iso",
				"-s", "30:0,ahci,cd:/some/path/to/anotherTestIso.iso",
			},
			wantSlot: 31,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			config.Config.Disk.VM.Path.Iso = "/the/config/path/for/isos"

			vm := &VM{
				ISOs: testCase.fields.ISOs,
			}

			gotArg, gotSlot := vm.getCDArg(testCase.args.slot)

			diff := deep.Equal(gotArg, testCase.wantArg)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			diff = deep.Equal(gotSlot, testCase.wantSlot)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}
