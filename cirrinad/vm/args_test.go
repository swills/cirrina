package vm

import (
	"database/sql"
	"net"
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

func TestVM_getHostBridgeArg(t *testing.T) {
	type fields struct {
		Config Config
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
			name: "noHostBridge",
			fields: fields{
				Config: Config{
					HostBridge: false,
				},
			},
			args:     args{slot: 2},
			wantArg:  []string{},
			wantSlot: 2,
		},
		{
			name: "yesHostBridge",
			fields: fields{
				Config: Config{
					HostBridge: true,
				},
			},
			args:     args{slot: 2},
			wantArg:  []string{"-s", "2,hostbridge"},
			wantSlot: 3,
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

			gotArg, gotSlot := vm.getHostBridgeArg(testCase.args.slot)

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

func Test_getCom(t *testing.T) {
	type args struct {
		comDev string
		vmName string
		num    int
	}

	tests := []struct {
		name     string
		args     args
		wantArgs []string
		wantNmdm string
	}{
		{
			name: "autoCom1",
			args: args{
				comDev: "AUTO",
				vmName: "testVM",
				num:    1,
			},
			wantNmdm: "/dev/nmdm-testVM-com1-A",
			wantArgs: []string{"-l", "com1,/dev/nmdm-testVM-com1-A"},
		},
		{
			name: "autoCom2",
			args: args{
				comDev: "AUTO",
				vmName: "testVM",
				num:    2,
			},
			wantNmdm: "/dev/nmdm-testVM-com2-A",
			wantArgs: []string{"-l", "com2,/dev/nmdm-testVM-com2-A"},
		},
		{
			name: "autoCom3",
			args: args{
				comDev: "AUTO",
				vmName: "testVM",
				num:    3,
			},
			wantNmdm: "/dev/nmdm-testVM-com3-A",
			wantArgs: []string{"-l", "com3,/dev/nmdm-testVM-com3-A"},
		},
		{
			name: "autoCom4",
			args: args{
				comDev: "AUTO",
				vmName: "testVM",
				num:    4,
			},
			wantNmdm: "/dev/nmdm-testVM-com4-A",
			wantArgs: []string{"-l", "com4,/dev/nmdm-testVM-com4-A"},
		},
		{
			name: "specifyCom1",
			args: args{
				comDev: "/dev/nmdm-somethingA",
				vmName: "testVM",
				num:    1,
			},
			wantNmdm: "/dev/nmdm-somethingA",
			wantArgs: []string{"-l", "com1,/dev/nmdm-somethingA"},
		},
		{
			name: "specifyCom2",
			args: args{
				comDev: "/dev/nmdm-somethingA",
				vmName: "testVM",
				num:    2,
			},
			wantNmdm: "/dev/nmdm-somethingA",
			wantArgs: []string{"-l", "com2,/dev/nmdm-somethingA"},
		},
		{
			name: "specifyCom3",
			args: args{
				comDev: "/dev/nmdm-somethingA",
				vmName: "testVM",
				num:    3,
			},
			wantNmdm: "/dev/nmdm-somethingA",
			wantArgs: []string{"-l", "com3,/dev/nmdm-somethingA"},
		},
		{
			name: "specifyCom4",
			args: args{
				comDev: "/dev/nmdm-somethingA",
				vmName: "testVM",
				num:    4,
			},
			wantNmdm: "/dev/nmdm-somethingA",
			wantArgs: []string{"-l", "com4,/dev/nmdm-somethingA"},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			gotArgs, gotNmdm := getCom(testCase.args.comDev, testCase.args.vmName, testCase.args.num)

			diff := deep.Equal(gotArgs, testCase.wantArgs)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			diff = deep.Equal(gotNmdm, testCase.wantNmdm)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func TestVM_getTabletArg(t *testing.T) {
	type fields struct {
		Config Config
	}

	type args struct {
		slot int
	}

	tests := []struct {
		name     string
		fields   fields
		args     args
		wantArgs []string
		wantSlot int
	}{
		{
			name: "noScreenOrTablet",
			fields: fields{
				Config: Config{
					Screen: false,
					Tablet: false,
				},
			},
			args: args{
				slot: 16,
			},
			wantArgs: []string{},
			wantSlot: 16,
		},
		{
			name: "screenNoTablet",
			fields: fields{
				Config: Config{
					Screen: true,
					Tablet: false,
				},
			},
			args: args{
				slot: 16,
			},
			wantArgs: []string{},
			wantSlot: 16,
		},
		{
			name: "screenAndTablet",
			fields: fields{
				Config: Config{
					Screen: true,
					Tablet: true,
				},
			},
			args: args{
				slot: 16,
			},
			wantArgs: []string{"-s", "16,xhci,tablet"},
			wantSlot: 17,
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

			gotArgs, gotSlot := vm.getTabletArg(testCase.args.slot)

			diff := deep.Equal(gotArgs, testCase.wantArgs)
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

func Test_addPriorityArgs(t *testing.T) {
	type args struct {
		vm   *VM
		args []string
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "noPriority",
			args: args{
				vm: &VM{
					Config: Config{
						Priority: 0,
					},
				},
				args: []string{"/usr/bin/protect"},
			},
			want: []string{"/usr/bin/protect"},
		},
		{
			name: "priorityTen",
			args: args{
				vm: &VM{
					Config: Config{
						Priority: 10,
					},
				},
				args: []string{"/usr/bin/protect"},
			},
			want: []string{"/usr/bin/protect", "/usr/bin/nice", "-n", "10"},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := addPriorityArgs(testCase.args.vm, testCase.args.args)

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func Test_addProtectArgs(t *testing.T) {
	type args struct {
		vm   *VM
		args []string
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "noProtect",
			args: args{
				vm: &VM{
					Config: Config{
						Protect: sql.NullBool{
							Bool:  false,
							Valid: true,
						},
					},
				},
				args: []string{},
			},
			want: []string{},
		},
		{
			name: "yesProtect",
			args: args{
				vm: &VM{
					Config: Config{
						Protect: sql.NullBool{
							Bool:  true,
							Valid: true,
						},
					},
				},
				args: []string{},
			},
			want: []string{"/usr/bin/protect"},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := addProtectArgs(testCase.args.vm, testCase.args.args)

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func Test_getNetTypeArg(t *testing.T) {
	type args struct {
		netType string
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "virtionet",
			args: args{
				netType: "VIRTIONET",
			},
			want:    "virtio-net",
			wantErr: false,
		},
		{
			name: "e1000",
			args: args{
				netType: "E1000",
			},
			want:    "e1000",
			wantErr: false,
		},
		{
			name: "junk",
			args: args{
				netType: "someJunk",
			},
			want:    "",
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := getNetTypeArg(testCase.args.netType)

			if (err != nil) != testCase.wantErr {
				t.Errorf("getNetTypeArg() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("getNetTypeArg() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func Test_getTapDev(t *testing.T) {
	tests := []struct {
		name            string
		hostIntStubFunc func() ([]net.Interface, error)
		wantNetDev      string
		wantNetDevArg   string
	}{
		{
			name:            "noInterfaces",
			hostIntStubFunc: StubHostInterfacesSuccess1,
			wantNetDev:      "tap0",
			wantNetDevArg:   "tap0",
		},
		{
			name:            "oneTap",
			hostIntStubFunc: StubHostInterfacesSuccess2,
			wantNetDev:      "tap1",
			wantNetDevArg:   "tap1",
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			NetInterfacesFunc = testCase.hostIntStubFunc

			t.Cleanup(func() { NetInterfacesFunc = net.Interfaces })

			getNetDev, gotNetDevArg := getTapDev()
			if getNetDev != testCase.wantNetDev {
				t.Errorf("getTapDev() got = %v, want %v", getNetDev, testCase.wantNetDev)
			}

			if gotNetDevArg != testCase.wantNetDevArg {
				t.Errorf("getTapDev() got1 = %v, want %v", gotNetDevArg, testCase.wantNetDevArg)
			}
		})
	}
}

func Test_getVmnetDev(t *testing.T) {
	tests := []struct {
		name            string
		hostIntStubFunc func() ([]net.Interface, error)
		wantNetDev      string
		wantNetDevArg   string
	}{
		{
			name:            "noVmNets",
			hostIntStubFunc: StubHostInterfacesSuccess3,
			wantNetDev:      "vmnet0",
			wantNetDevArg:   "vmnet0",
		},
		{
			name:            "oneVmNets",
			hostIntStubFunc: StubHostInterfacesSuccess4,
			wantNetDev:      "vmnet1",
			wantNetDevArg:   "vmnet1",
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			NetInterfacesFunc = testCase.hostIntStubFunc

			t.Cleanup(func() { NetInterfacesFunc = net.Interfaces })

			gotNetDev, gotNetDevArg := getVmnetDev()

			if gotNetDev != testCase.wantNetDev {
				t.Errorf("getVmnetDev() gotNetDev = %v, want %v", gotNetDev, testCase.wantNetDev)
			}

			if gotNetDevArg != testCase.wantNetDev {
				t.Errorf("getVmnetDev() gotNetDevArg = %v, want %v", gotNetDevArg, testCase.wantNetDev)
			}
		})
	}
}

//nolint:paralleltest
func Test_getComArgs(t *testing.T) {
	type args struct {
		aVM *VM
	}

	tests := []struct {
		name        string
		args        args
		wantCom1Arg []string
		wantCom2Arg []string
		wantCom3Arg []string
		wantCom4Arg []string
		wantCom1Dev string
		wantCom2Dev string
		wantCom3Dev string
		wantCom4Dev string
	}{
		{
			name: "noCom",
			args: args{
				aVM: &VM{
					Name: "someTestVM",
					Config: Config{
						Com1:    false,
						Com1Dev: "AUTO",
						Com2:    false,
						Com2Dev: "AUTO",
						Com3:    false,
						Com3Dev: "AUTO",
						Com4:    false,
						Com4Dev: "AUTO",
					},
					Com1Dev: "",
					Com2Dev: "",
					Com3Dev: "",
					Com4Dev: "",
				},
			},
			wantCom1Arg: nil,
			wantCom2Arg: nil,
			wantCom3Arg: nil,
			wantCom4Arg: nil,
			wantCom1Dev: "",
			wantCom2Dev: "",
			wantCom3Dev: "",
			wantCom4Dev: "",
		},
		{
			name: "onlyCom1",
			args: args{
				aVM: &VM{
					Name: "someTestVM",
					Config: Config{
						Com1:    true,
						Com1Dev: "AUTO",
						Com2:    false,
						Com2Dev: "AUTO",
						Com3:    false,
						Com3Dev: "AUTO",
						Com4:    false,
						Com4Dev: "AUTO",
					},
					Com1Dev: "",
					Com2Dev: "",
					Com3Dev: "",
					Com4Dev: "",
				},
			},
			wantCom1Arg: []string{"-l", "com1,/dev/nmdm-someTestVM-com1-A"},
			wantCom2Arg: nil,
			wantCom3Arg: nil,
			wantCom4Arg: nil,
			wantCom1Dev: "/dev/nmdm-someTestVM-com1-A",
			wantCom2Dev: "",
			wantCom3Dev: "",
			wantCom4Dev: "",
		},
		{
			name: "onlyCom2",
			args: args{
				aVM: &VM{
					Name: "someTestVM",
					Config: Config{
						Com1:    false,
						Com1Dev: "AUTO",
						Com2:    true,
						Com2Dev: "AUTO",
						Com3:    false,
						Com3Dev: "AUTO",
						Com4:    false,
						Com4Dev: "AUTO",
					},
					Com1Dev: "",
					Com2Dev: "",
					Com3Dev: "",
					Com4Dev: "",
				},
			},
			wantCom1Arg: nil,
			wantCom2Arg: []string{"-l", "com2,/dev/nmdm-someTestVM-com2-A"},
			wantCom3Arg: nil,
			wantCom4Arg: nil,
			wantCom1Dev: "",
			wantCom2Dev: "/dev/nmdm-someTestVM-com2-A",
			wantCom3Dev: "",
			wantCom4Dev: "",
		},
		{
			name: "onlyCom3",
			args: args{
				aVM: &VM{
					Name: "someTestVM",
					Config: Config{
						Com1:    false,
						Com1Dev: "AUTO",
						Com2:    false,
						Com2Dev: "AUTO",
						Com3:    true,
						Com3Dev: "AUTO",
						Com4:    false,
						Com4Dev: "AUTO",
					},
					Com1Dev: "",
					Com2Dev: "",
					Com3Dev: "",
					Com4Dev: "",
				},
			},
			wantCom1Arg: nil,
			wantCom2Arg: nil,
			wantCom3Arg: []string{"-l", "com3,/dev/nmdm-someTestVM-com3-A"},
			wantCom4Arg: nil,
			wantCom1Dev: "",
			wantCom2Dev: "",
			wantCom3Dev: "/dev/nmdm-someTestVM-com3-A",
			wantCom4Dev: "",
		},
		{
			name: "onlyCom4",
			args: args{
				aVM: &VM{
					Name: "someTestVM",
					Config: Config{
						Com1:    false,
						Com1Dev: "AUTO",
						Com2:    false,
						Com2Dev: "AUTO",
						Com3:    false,
						Com3Dev: "AUTO",
						Com4:    true,
						Com4Dev: "AUTO",
					},
					Com1Dev: "",
					Com2Dev: "",
					Com3Dev: "",
					Com4Dev: "",
				},
			},
			wantCom1Arg: nil,
			wantCom2Arg: nil,
			wantCom3Arg: nil,
			wantCom4Arg: []string{"-l", "com4,/dev/nmdm-someTestVM-com4-A"},
			wantCom1Dev: "",
			wantCom2Dev: "",
			wantCom3Dev: "",
			wantCom4Dev: "/dev/nmdm-someTestVM-com4-A",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			List.VMList[testCase.args.aVM.ID] = testCase.args.aVM

			gotCom1Arg, gotCom2Arg, gotCom3Arg, gotCom4Arg := getComArgs(testCase.args.aVM)

			diff := deep.Equal(gotCom1Arg, testCase.wantCom1Arg)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			diff = deep.Equal(gotCom2Arg, testCase.wantCom2Arg)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			diff = deep.Equal(gotCom3Arg, testCase.wantCom3Arg)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			diff = deep.Equal(gotCom4Arg, testCase.wantCom4Arg)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			diff = deep.Equal(testCase.args.aVM.Com1Dev, testCase.wantCom1Dev)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			diff = deep.Equal(testCase.args.aVM.Com2Dev, testCase.wantCom2Dev)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			diff = deep.Equal(testCase.args.aVM.Com3Dev, testCase.wantCom3Dev)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			diff = deep.Equal(testCase.args.aVM.Com4Dev, testCase.wantCom4Dev)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func StubHostInterfacesSuccess1() ([]net.Interface, error) {
	return []net.Interface{
		{
			Index:        1,
			MTU:          1500,
			Name:         "abc0",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x28, 0x73, 0x3e},
			Flags:        0x33,
		},
		{
			Index:        2,
			MTU:          1500,
			Name:         "def0",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x32, 0x6e, 0x6},
			Flags:        0x33,
		},
		{
			Index:        3,
			MTU:          16384,
			Name:         "lo0",
			HardwareAddr: net.HardwareAddr(nil),
			Flags:        0x35,
		},
	}, nil
}

func StubHostInterfacesSuccess2() ([]net.Interface, error) {
	return []net.Interface{
		{
			Index:        1,
			MTU:          1500,
			Name:         "tap0",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x28, 0x73, 0x3e},
			Flags:        0x33,
		},
		{
			Index:        2,
			MTU:          1500,
			Name:         "def0",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x32, 0x6e, 0x6},
			Flags:        0x33,
		},
		{
			Index:        3,
			MTU:          16384,
			Name:         "lo0",
			HardwareAddr: net.HardwareAddr(nil),
			Flags:        0x35,
		},
	}, nil
}

func StubHostInterfacesSuccess3() ([]net.Interface, error) {
	return []net.Interface{
		{
			Index:        1,
			MTU:          1500,
			Name:         "abc0",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x28, 0x73, 0x3e},
			Flags:        0x33,
		},
		{
			Index:        2,
			MTU:          1500,
			Name:         "def0",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x32, 0x6e, 0x6},
			Flags:        0x33,
		},
		{
			Index:        3,
			MTU:          16384,
			Name:         "lo0",
			HardwareAddr: net.HardwareAddr(nil),
			Flags:        0x35,
		},
	}, nil
}

func StubHostInterfacesSuccess4() ([]net.Interface, error) {
	return []net.Interface{
		{
			Index:        1,
			MTU:          1500,
			Name:         "vmnet0",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x28, 0x73, 0x3e},
			Flags:        0x33,
		},
		{
			Index:        2,
			MTU:          1500,
			Name:         "def0",
			HardwareAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x32, 0x6e, 0x6},
			Flags:        0x33,
		},
		{
			Index:        3,
			MTU:          16384,
			Name:         "lo0",
			HardwareAddr: net.HardwareAddr(nil),
			Flags:        0x35,
		},
	}, nil
}
