package vm

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-test/deep"
	"gorm.io/gorm"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/iso"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vmnic"
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

//nolint:paralleltest
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

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
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

//nolint:paralleltest
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

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
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

//nolint:paralleltest
func Test_getMac(t *testing.T) {
	type args struct {
		thisNic vmnic.VMNic
		thisVM  *VM
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Auto",
			args: args{
				thisNic: vmnic.VMNic{
					ID:   "f865c0c5-4a06-40c6-b066-2c10c81691d1",
					Name: "test2024080901_int0",
					Mac:  "AUTO",
				},
				thisVM: &VM{
					ID:   "58b45d43-b1f1-47fd-a94a-d877a89ec34f",
					Name: "test2024080901",
				},
			},
			want: "d9:81:b2:3d:a7:a2",
		},
		{
			name: "Specified",
			args: args{
				thisNic: vmnic.VMNic{
					ID:   "f865c0c5-4a06-40c6-b066-2c10c81691d1",
					Name: "test2024080901_int0",
					Mac:  "00:22:44:AA:BB:CC",
				},
				thisVM: &VM{
					ID:   "58b45d43-b1f1-47fd-a94a-d877a89ec34f",
					Name: "test2024080901",
				},
			},
			want: "00:22:44:AA:BB:CC",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			config.Config.Network.Mac.Oui = "d9:81:b2"
			got := getMac(testCase.args.thisNic, testCase.args.thisVM)

			if got != testCase.want {
				t.Errorf("getMac() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func Test_addComArgs(t *testing.T) {
	type args struct {
		com1Arg []string
		args    []string
		com2Arg []string
		com3Arg []string
		com4Arg []string
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "none",
			args: args{
				args:    nil,
				com1Arg: nil,
				com2Arg: nil,
				com3Arg: nil,
				com4Arg: nil,
			},
			want: nil,
		},
		{
			name: "comOne",
			args: args{
				args:    nil,
				com1Arg: []string{"-l", "com1,/dev/nmdm-test2024060101-com1-A"},
				com2Arg: nil,
				com3Arg: nil,
				com4Arg: nil,
			},
			want: []string{"-l", "com1,/dev/nmdm-test2024060101-com1-A"},
		},
		{
			name: "comTwo",
			args: args{
				args:    nil,
				com1Arg: nil,
				com2Arg: []string{"-l", "com2,/dev/nmdm-test2024060101-com2-A"},
				com3Arg: nil,
				com4Arg: nil,
			},
			want: []string{"-l", "com2,/dev/nmdm-test2024060101-com2-A"},
		},
		{
			name: "comThree",
			args: args{
				args:    nil,
				com1Arg: nil,
				com2Arg: nil,
				com3Arg: []string{"-l", "com3,/dev/nmdm-test2024060101-com3-A"},
				com4Arg: nil,
			},
			want: []string{"-l", "com3,/dev/nmdm-test2024060101-com3-A"},
		},
		{
			name: "comFour",
			args: args{
				args:    nil,
				com1Arg: nil,
				com2Arg: nil,
				com3Arg: nil,
				com4Arg: []string{"-l", "com4,/dev/nmdm-test2024060101-com4-A"},
			},
			want: []string{"-l", "com4,/dev/nmdm-test2024060101-com4-A"},
		},
		{
			name: "comAll",
			args: args{
				args:    nil,
				com1Arg: []string{"-l", "com1,/dev/nmdm-test2024060101-com1-A"},
				com2Arg: []string{"-l", "com2,/dev/nmdm-test2024060101-com2-A"},
				com3Arg: []string{"-l", "com3,/dev/nmdm-test2024060101-com3-A"},
				com4Arg: []string{"-l", "com4,/dev/nmdm-test2024060101-com4-A"},
			},
			want: []string{
				"-l", "com1,/dev/nmdm-test2024060101-com1-A",
				"-l", "com2,/dev/nmdm-test2024060101-com2-A",
				"-l", "com3,/dev/nmdm-test2024060101-com3-A",
				"-l", "com4,/dev/nmdm-test2024060101-com4-A",
			},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := addComArgs(
				testCase.args.args,
				testCase.args.com1Arg,
				testCase.args.com2Arg,
				testCase.args.com3Arg,
				testCase.args.com4Arg,
			)

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

//nolint:paralleltest,maintidx
func TestVM_getDiskArg(t *testing.T) {
	type fields struct {
		Disks []*disk.Disk
	}

	type args struct {
		slot int
	}

	tests := []struct {
		name        string
		mockClosure func()
		fields      fields
		args        args
		wantArgs    []string
		wantSlot    int
		wantPath    bool
		wantPathErr bool
	}{
		{
			name:        "None",
			mockClosure: func() {},
			fields: fields{
				Disks: nil,
			},
			args: args{
				slot: 3,
			},
			wantArgs: nil,
			wantSlot: 3,
		},
		{
			name: "OneDiskNVMEFile",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
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
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,nvme,test2024081001_hd0.img"},
			wantSlot: 4,
		},
		{
			name: "OneDiskNVMEFileNoCache",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,nvme,test2024081001_hd0.img,nocache"},
			wantSlot: 4,
		},
		{
			name: "OneDiskNVMEFileDirect",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "NVME",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,nvme,test2024081001_hd0.img,direct"},
			wantSlot: 4,
		},
		{
			name: "OneDiskAHCIFile",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "AHCI-HD",
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
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,ahci-hd,test2024081001_hd0.img"},
			wantSlot: 4,
		},
		{
			name: "OneDiskAHCIFileNoCache",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "AHCI-HD",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,ahci-hd,test2024081001_hd0.img,nocache"},
			wantSlot: 4,
		},
		{
			name: "OneDiskAHCIFileDirect",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "AHCI-HD",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,ahci-hd,test2024081001_hd0.img,direct"},
			wantSlot: 4,
		},
		{
			name: "OneDiskVirtIOFile",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "VIRTIO-BLK",
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
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,virtio-blk,test2024081001_hd0.img"},
			wantSlot: 4,
		},
		{
			name: "OneDiskVirtIOFileNoCache",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "VIRTIO-BLK",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,virtio-blk,test2024081001_hd0.img,nocache"},
			wantSlot: 4,
		},
		{
			name: "OneDiskVirtIOFileDirect",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "VIRTIO-BLK",
					DevType:     "FILE",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,virtio-blk,test2024081001_hd0.img,direct"},
			wantSlot: 4,
		},
		{
			name: "OneDiskNVMEZVOL",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "NVME",
					DevType:     "ZVOL",
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
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,nvme,/dev/zvol/test2024081001_hd0"},
			wantSlot: 4,
		},
		{
			name: "OneDiskNVMEZVOLNoCache",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "NVME",
					DevType:     "ZVOL",
					DiskCache: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,nvme,/dev/zvol/test2024081001_hd0,nocache"},
			wantSlot: 4,
		},
		{
			name: "OneDiskNVMEZVOLDirect",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "NVME",
					DevType:     "ZVOL",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,nvme,/dev/zvol/test2024081001_hd0,direct"},
			wantSlot: 4,
		},
		{
			name: "OneDiskAHCIZVOL",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "AHCI-HD",
					DevType:     "ZVOL",
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
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,ahci-hd,/dev/zvol/test2024081001_hd0"},
			wantSlot: 4,
		},
		{
			name: "OneDiskAHCIZVOLNoCache",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "AHCI-HD",
					DevType:     "ZVOL",
					DiskCache: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,ahci-hd,/dev/zvol/test2024081001_hd0,nocache"},
			wantSlot: 4,
		},
		{
			name: "OneDiskAHCIZVOLDirect",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "AHCI-HD",
					DevType:     "ZVOL",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,ahci-hd,/dev/zvol/test2024081001_hd0,direct"},
			wantSlot: 4,
		},
		{
			name: "OneDiskVirtIOZVOL",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "VIRTIO-BLK",
					DevType:     "ZVOL",
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
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,virtio-blk,/dev/zvol/test2024081001_hd0"},
			wantSlot: 4,
		},
		{
			name: "OneDiskVirtIOZVOLNoCache",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "VIRTIO-BLK",
					DevType:     "ZVOL",
					DiskCache: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  false,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,virtio-blk,/dev/zvol/test2024081001_hd0,nocache"},
			wantSlot: 4,
		},
		{
			name: "OneDiskVirtIOZVOLDirect",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "VIRTIO-BLK",
					DevType:     "ZVOL",
					DiskCache: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
					DiskDirect: sql.NullBool{
						Bool:  true,
						Valid: true,
					},
				}
				disk.List.DiskList[diskInst.ID] = diskInst
			},
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: []string{"-s", "3,virtio-blk,/dev/zvol/test2024081001_hd0,direct"},
			wantSlot: 4,
		},
		{
			name: "ErrorCheckingExists",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
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
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPathErr: true,
			wantArgs:    nil,
			wantSlot:    3,
		},
		{
			name: "DiskDoesNotExist",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
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
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: false,
			wantArgs: nil,
			wantSlot: 3,
		},
		{
			name: "BadType",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
					Type:        "someGarbage",
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
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: nil,
			wantSlot: 3,
		},
		{
			name: "NilDisk",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
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
			fields: fields{
				Disks: []*disk.Disk{
					nil,
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: nil,
			wantSlot: 3,
		},
		{
			name: "EmptyDiskID",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
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
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: nil,
			wantSlot: 3,
		},
		{
			name: "BadDiskID",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
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
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a3",
					},
				},
			},
			args: args{
				slot: 3,
			},
			wantPath: true,
			wantArgs: nil,
			wantSlot: 3,
		},
		{
			name: "TooManyDisks",
			mockClosure: func() {
				diskInst := &disk.Disk{
					ID:          "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					Name:        "test2024081001_hd0",
					Description: "some test disk",
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
			fields: fields{
				Disks: []*disk.Disk{
					{
						ID: "25b5e67d-915d-4b0e-bb3a-42f3233510a2",
					},
				},
			},
			args: args{
				slot: 31,
			},
			wantPath: true,
			wantArgs: nil,
			wantSlot: 31,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			disk.List.DiskList = map[string]*disk.Disk{}

			testDB, mock := cirrinadtest.NewMockDB("diskTest")

			testCase.mockClosure()

			disk.PathExistsFunc = func(_ string) (bool, error) {
				if testCase.wantPathErr {
					return true, errors.New("another error") //nolint:goerr113
				}

				if testCase.wantPath {
					return true, nil
				}

				return false, nil
			}

			vm := &VM{
				Disks: testCase.fields.Disks,
			}
			gotArgs, gotSlot := vm.getDiskArg(testCase.args.slot)

			diff := deep.Equal(gotArgs, testCase.wantArgs)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			diff = deep.Equal(gotSlot, testCase.wantSlot)
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

//nolint:paralleltest
func TestVM_getSoundArg(t *testing.T) {
	type fields struct {
		Config Config
	}

	type args struct {
		slot int
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantArgs    []string
		wantSlot    int
		wantPath    bool
		wantPathErr bool
	}{
		{
			name: "NoSound",
			fields: fields{
				Config: Config{
					Sound:    false,
					SoundIn:  "/dev/dsp40",
					SoundOut: "/dev/dsp41",
				},
			},
			args: args{
				slot: 3,
			},
			wantArgs: []string{},
			wantSlot: 3,
		},
		{
			name: "YesSound",
			fields: fields{
				Config: Config{
					Sound:    true,
					SoundIn:  "/dev/dsp42",
					SoundOut: "/dev/dsp43",
				},
			},
			args: args{
				slot: 3,
			},
			wantArgs:    []string{"-s", "3,hda,play=/dev/dsp43,rec=/dev/dsp42"},
			wantSlot:    4,
			wantPathErr: false,
			wantPath:    true,
		},
		{
			name: "YesSoundOutNonexistent",
			fields: fields{
				Config: Config{
					Sound:    true,
					SoundIn:  "/dev/dsp44",
					SoundOut: "/dev/dsp45",
				},
			},
			args: args{
				slot: 3,
			},
			wantArgs:    []string{"-s", "3,hda,rec=/dev/dsp44"},
			wantSlot:    4,
			wantPathErr: false,
			wantPath:    true,
		},
		{
			name: "YesSoundInNonexistent",
			fields: fields{
				Config: Config{
					Sound:    true,
					SoundIn:  "/dev/dsp45",
					SoundOut: "/dev/dsp44",
				},
			},
			args: args{
				slot: 3,
			},
			wantArgs:    []string{"-s", "3,hda,play=/dev/dsp44"},
			wantSlot:    4,
			wantPathErr: false,
			wantPath:    true,
		},
		{
			name: "YesSoundBothNonexistent",
			fields: fields{
				Config: Config{
					Sound:    true,
					SoundIn:  "/dev/dsp46",
					SoundOut: "/dev/dsp47",
				},
			},
			args: args{
				slot: 3,
			},
			wantArgs:    nil,
			wantSlot:    3,
			wantPathErr: false,
			wantPath:    false,
		},
		{
			name: "YesSoundOutExistError",
			fields: fields{
				Config: Config{
					Sound:    true,
					SoundIn:  "/dev/dsp46",
					SoundOut: "/dev/dsp48",
				},
			},
			args: args{
				slot: 3,
			},
			wantArgs:    nil,
			wantSlot:    3,
			wantPathErr: false,
			wantPath:    false,
		},
		{
			name: "YesSoundInExistError",
			fields: fields{
				Config: Config{
					Sound:    true,
					SoundIn:  "/dev/dsp48",
					SoundOut: "/dev/dsp46",
				},
			},
			args: args{
				slot: 3,
			},
			wantArgs:    nil,
			wantSlot:    3,
			wantPathErr: false,
			wantPath:    false,
		},
		{
			name: "YesSoundBothExistError",
			fields: fields{
				Config: Config{
					Sound:    true,
					SoundIn:  "/dev/dsp49",
					SoundOut: "/dev/dsp49",
				},
			},
			args: args{
				slot: 3,
			},
			wantArgs:    nil,
			wantSlot:    3,
			wantPathErr: true,
			wantPath:    false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testVM := &VM{
				Config: testCase.fields.Config,
			}

			PathExistsFunc = func(testPath string) (bool, error) {
				if testCase.wantPathErr {
					return true, errors.New("another error") //nolint:goerr113
				}

				if strings.Contains(testPath, "dsp48") {
					return false, errors.New("sound error") //nolint:goerr113
				}

				if strings.Contains(testPath, "dsp45") {
					return false, nil
				}

				if testCase.wantPath {
					return true, nil
				}

				return false, nil
			}

			t.Cleanup(func() { PathExistsFunc = util.PathExists })

			gotArgs, gotSlot := testVM.getSoundArg(testCase.args.slot)

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

//nolint:paralleltest
func TestVM_getDebugArg(t *testing.T) {
	type fields struct {
		ID          string
		CreatedAt   time.Time
		UpdatedAt   time.Time
		DeletedAt   gorm.DeletedAt
		Name        string
		Description string
		Status      StatusType
		BhyvePid    uint32
		VNCPort     int32
		DebugPort   int32
		Config      Config
		ISOs        []*iso.ISO
		Disks       []*disk.Disk
		Com1Dev     string
		Com2Dev     string
		Com3Dev     string
		Com4Dev     string
		Com1write   bool
		Com2write   bool
		Com3write   bool
		Com4write   bool
	}

	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "Success",
			fields: fields{
				ID:        "",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{
					Time:  time.Time{},
					Valid: false,
				},
				Name:        "",
				Description: "",
				Status:      "",
				BhyvePid:    0,
				VNCPort:     0,
				DebugPort:   0,
				Config: Config{
					Model: gorm.Model{
						ID:        0,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID:             "",
					CPU:              0,
					Mem:              0,
					MaxWait:          0,
					Restart:          false,
					RestartDelay:     0,
					Screen:           false,
					ScreenWidth:      0,
					ScreenHeight:     0,
					VNCWait:          false,
					VNCPort:          "",
					Tablet:           false,
					StoreUEFIVars:    false,
					UTCTime:          false,
					HostBridge:       false,
					ACPI:             false,
					UseHLT:           false,
					ExitOnPause:      false,
					WireGuestMem:     false,
					DestroyPowerOff:  false,
					IgnoreUnknownMSR: false,
					KbdLayout:        "",
					AutoStart:        false,
					Sound:            false,
					SoundIn:          "",
					SoundOut:         "",
					Com1:             false,
					Com1Dev:          "",
					Com1Log:          false,
					Com2:             false,
					Com2Dev:          "",
					Com2Log:          false,
					Com3:             false,
					Com3Dev:          "",
					Com3Log:          false,
					Com4:             false,
					Com4Dev:          "",
					Com4Log:          false,
					ExtraArgs:        "",
					Com1Speed:        0,
					Com2Speed:        0,
					Com3Speed:        0,
					Com4Speed:        0,
					AutoStartDelay:   0,
					Debug:            true,
					DebugWait:        false,
					DebugPort:        "4444",
				},
			},
			want: []string{"-G", ":4444"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testVM := &VM{
				ID:          testCase.fields.ID,
				CreatedAt:   testCase.fields.CreatedAt,
				UpdatedAt:   testCase.fields.UpdatedAt,
				DeletedAt:   testCase.fields.DeletedAt,
				Name:        testCase.fields.Name,
				Description: testCase.fields.Description,
				Status:      testCase.fields.Status,
				BhyvePid:    testCase.fields.BhyvePid,
				VNCPort:     testCase.fields.VNCPort,
				DebugPort:   testCase.fields.DebugPort,
				Config:      testCase.fields.Config,
				ISOs:        testCase.fields.ISOs,
				Disks:       testCase.fields.Disks,
				Com1Dev:     testCase.fields.Com1Dev,
				Com2Dev:     testCase.fields.Com2Dev,
				Com3Dev:     testCase.fields.Com3Dev,
				Com4Dev:     testCase.fields.Com4Dev,
				Com1write:   testCase.fields.Com1write,
				Com2write:   testCase.fields.Com2write,
				Com3write:   testCase.fields.Com3write,
				Com4write:   testCase.fields.Com4write,
			}

			got := testVM.getDebugArg()

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

//nolint:paralleltest,maintidx
func TestVM_getVideoArg(t *testing.T) {
	createUpdateTime := time.Now()

	type fields struct {
		ID          string
		CreatedAt   time.Time
		UpdatedAt   time.Time
		DeletedAt   gorm.DeletedAt
		Name        string
		Description string
		Status      StatusType
		BhyvePid    uint32
		VNCPort     int32
		DebugPort   int32
		Config      Config
		ISOs        []*iso.ISO
		Disks       []*disk.Disk
		Com1Dev     string
		Com2Dev     string
		Com3Dev     string
		Com4Dev     string
		Com1write   bool
		Com2write   bool
		Com3write   bool
		Com4write   bool
	}

	type args struct {
		slot int
	}

	tests := []struct {
		name            string
		mockVMClosure   func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		mockGetPortFunc func(int, []int) (int, error)
		fields          fields
		args            args
		wantArgs        []string
		wantSlot        int
	}{
		{
			name: "SuccessAutoNoWait",
			mockVMClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 60, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "default", 60, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "AUTO", false, 0, 0, false, sqlmock.AnyArg(), 81). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "test vm", "test2024081103", 7901, sqlmock.AnyArg(), "5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 27))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_isos` (`vm_id`,`iso_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75", "59445e39-a842-467c-9cb8-4bd4b0a529c7", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_disks` (`vm_id`,`disk_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75", "0e7af864-ed45-4256-947e-871f6ba3a3ac", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			mockGetPortFunc: func(_ int, _ []int) (int, error) {
				return 7901, nil
			},
			fields: fields{
				ID:          "5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081103",
				Description: "test vm",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID:        81,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
					},
					VMID:             "5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75",
					CPU:              2,
					Mem:              2048,
					MaxWait:          60,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					Sound:            false,
					SoundIn:          "/dev/dsp0",
					SoundOut:         "/dev/dsp0",
					VNCPort:          "AUTO",
					Tablet:           true,
					StoreUEFIVars:    true,
					UTCTime:          true,
					HostBridge:       true,
					ACPI:             true,
					UseHLT:           true,
					ExitOnPause:      true,
					DestroyPowerOff:  true,
					IgnoreUnknownMSR: true,
					KbdLayout:        "default",
					Com1:             true,
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        115200,
					Com2Speed:        115200,
					Com3Speed:        115200,
					Com4Speed:        115200,
					AutoStartDelay:   60,
					DebugPort:        "AUTO",
				},
				ISOs: []*iso.ISO{
					{
						ID: "59445e39-a842-467c-9cb8-4bd4b0a529c7",
					},
				},
				Disks: []*disk.Disk{
					{
						ID: "0e7af864-ed45-4256-947e-871f6ba3a3ac",
					},
				},
			},
			args: args{
				slot: 6,
			},
			wantArgs: []string{"-s", "6,fbuf,w=1920,h=1080,tcp=:7901"},
			wantSlot: 7,
		},
		{
			name: "SuccessSpecifiedNoWait",
			mockVMClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 60, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "default", 60, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "8901", false, 0, 0, false, sqlmock.AnyArg(), 81). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "test vm", "test2024081103", 8901, sqlmock.AnyArg(), "5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 27))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_isos` (`vm_id`,`iso_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75", "59445e39-a842-467c-9cb8-4bd4b0a529c7", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_disks` (`vm_id`,`disk_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75", "0e7af864-ed45-4256-947e-871f6ba3a3ac", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			mockGetPortFunc: func(_ int, _ []int) (int, error) {
				return 7901, nil
			},
			fields: fields{
				ID:          "5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081103",
				Description: "test vm",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID:        81,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
					},
					VMID:             "5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75",
					CPU:              2,
					Mem:              2048,
					MaxWait:          60,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					Sound:            false,
					SoundIn:          "/dev/dsp0",
					SoundOut:         "/dev/dsp0",
					VNCPort:          "8901",
					Tablet:           true,
					StoreUEFIVars:    true,
					UTCTime:          true,
					HostBridge:       true,
					ACPI:             true,
					UseHLT:           true,
					ExitOnPause:      true,
					DestroyPowerOff:  true,
					IgnoreUnknownMSR: true,
					KbdLayout:        "default",
					Com1:             true,
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        115200,
					Com2Speed:        115200,
					Com3Speed:        115200,
					Com4Speed:        115200,
					AutoStartDelay:   60,
					DebugPort:        "AUTO",
				},
				ISOs: []*iso.ISO{
					{
						ID: "59445e39-a842-467c-9cb8-4bd4b0a529c7",
					},
				},
				Disks: []*disk.Disk{
					{
						ID: "0e7af864-ed45-4256-947e-871f6ba3a3ac",
					},
				},
			},
			args: args{
				slot: 6,
			},
			wantArgs: []string{"-s", "6,fbuf,w=1920,h=1080,tcp=:8901"},
			wantSlot: 7,
		},
		{
			name: "SuccessAutoWait",
			mockVMClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 60, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "default", 60, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "AUTO", true, 0, 0, false, sqlmock.AnyArg(), 81). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "test vm", "test2024081103", 7901, sqlmock.AnyArg(), "5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 27))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_isos` (`vm_id`,`iso_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75", "59445e39-a842-467c-9cb8-4bd4b0a529c7", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_disks` (`vm_id`,`disk_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75", "0e7af864-ed45-4256-947e-871f6ba3a3ac", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			mockGetPortFunc: func(_ int, _ []int) (int, error) {
				return 7901, nil
			},
			fields: fields{
				ID:          "5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081103",
				Description: "test vm",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID:        81,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
					},
					VMID:             "5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75",
					CPU:              2,
					Mem:              2048,
					MaxWait:          60,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					Sound:            false,
					SoundIn:          "/dev/dsp0",
					SoundOut:         "/dev/dsp0",
					VNCPort:          "AUTO",
					VNCWait:          true,
					Tablet:           true,
					StoreUEFIVars:    true,
					UTCTime:          true,
					HostBridge:       true,
					ACPI:             true,
					UseHLT:           true,
					ExitOnPause:      true,
					DestroyPowerOff:  true,
					IgnoreUnknownMSR: true,
					KbdLayout:        "default",
					Com1:             true,
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        115200,
					Com2Speed:        115200,
					Com3Speed:        115200,
					Com4Speed:        115200,
					AutoStartDelay:   60,
					DebugPort:        "AUTO",
				},
				ISOs: []*iso.ISO{
					{
						ID: "59445e39-a842-467c-9cb8-4bd4b0a529c7",
					},
				},
				Disks: []*disk.Disk{
					{
						ID: "0e7af864-ed45-4256-947e-871f6ba3a3ac",
					},
				},
			},
			args: args{
				slot: 6,
			},
			wantArgs: []string{"-s", "6,fbuf,w=1920,h=1080,tcp=:7901,wait"},
			wantSlot: 7,
		},
		{
			name: "SuccessSpecifiedWait",
			mockVMClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `configs` SET `com1`=?,`com2`=?,`com3`=?,`acpi`=?,`auto_start`=?,`auto_start_delay`=?,`com1_dev`=?,`com1_log`=?,`com1_speed`=?,`com2_dev`=?,`com2_log`=?,`com2_speed`=?,`com3_dev`=?,`com3_log`=?,`com3_speed`=?,`com4`=?,`com4_dev`=?,`com4_log`=?,`com4_speed`=?,`cpu`=?,`debug`=?,`debug_port`=?,`debug_wait`=?,`destroy_power_off`=?,`exit_on_pause`=?,`extra_args`=?,`host_bridge`=?,`ignore_unknown_msr`=?,`kbd_layout`=?,`max_wait`=?,`mem`=?,`pcpu`=?,`priority`=?,`protect`=?,`rbps`=?,`restart`=?,`restart_delay`=?,`riops`=?,`screen`=?,`screen_height`=?,`screen_width`=?,`sound`=?,`sound_in`=?,`sound_out`=?,`store_uefi_vars`=?,`tablet`=?,`use_hlt`=?,`utc_time`=?,`vnc_port`=?,`vnc_wait`=?,`wbps`=?,`wiops`=?,`wire_guest_mem`=?,`updated_at`=? WHERE `configs`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(true, false, false, true, false, 60, "AUTO", false, 115200, "AUTO", false, 115200, "AUTO", false, 115200, false, "AUTO", false, 115200, 2, false, "AUTO", false, true, true, "", true, true, "default", 60, 2048, 0, 0, nil, 0, true, 0, 0, true, 1080, 1920, false, "/dev/dsp0", "/dev/dsp0", true, true, true, true, "8901", true, 0, 0, false, sqlmock.AnyArg(), 81). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta(
						"UPDATE `vms` SET `com1_dev`=?,`com2_dev`=?,`com3_dev`=?,`com4_dev`=?,`debug_port`=?,`description`=?,`name`=?,`vnc_port`=?,`updated_at`=? WHERE `vms`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs("", "", "", "", 0, "test vm", "test2024081103", 8901, sqlmock.AnyArg(), "5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75"). //nolint:lll
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_isos` WHERE `vm_id` = ?"),
				).
					WithArgs("5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75").
					// does not matter how many rows are returned, we wipe all isos from the VM
					// unconditionally and add the ones we want to have
					WillReturnResult(sqlmock.NewResult(1, 27))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_isos` (`vm_id`,`iso_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75", "59445e39-a842-467c-9cb8-4bd4b0a529c7", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()

				mock.ExpectExec(
					regexp.QuoteMeta("DELETE FROM `vm_disks` WHERE `vm_id` = ?"),
				).
					WithArgs("5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO `vm_disks` (`vm_id`,`disk_id`, `position`) VALUES (?,?,?)"),
				).
					WithArgs("5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75", "0e7af864-ed45-4256-947e-871f6ba3a3ac", 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			mockGetPortFunc: func(_ int, _ []int) (int, error) {
				return 7901, nil
			},
			fields: fields{
				ID:          "5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081103",
				Description: "test vm",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID:        81,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
					},
					VMID:             "5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75",
					CPU:              2,
					Mem:              2048,
					MaxWait:          60,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					Sound:            false,
					SoundIn:          "/dev/dsp0",
					SoundOut:         "/dev/dsp0",
					VNCPort:          "8901",
					VNCWait:          true,
					Tablet:           true,
					StoreUEFIVars:    true,
					UTCTime:          true,
					HostBridge:       true,
					ACPI:             true,
					UseHLT:           true,
					ExitOnPause:      true,
					DestroyPowerOff:  true,
					IgnoreUnknownMSR: true,
					KbdLayout:        "default",
					Com1:             true,
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        115200,
					Com2Speed:        115200,
					Com3Speed:        115200,
					Com4Speed:        115200,
					AutoStartDelay:   60,
					DebugPort:        "AUTO",
				},
				ISOs: []*iso.ISO{
					{
						ID: "59445e39-a842-467c-9cb8-4bd4b0a529c7",
					},
				},
				Disks: []*disk.Disk{
					{
						ID: "0e7af864-ed45-4256-947e-871f6ba3a3ac",
					},
				},
			},
			args: args{
				slot: 6,
			},
			wantArgs: []string{"-s", "6,fbuf,w=1920,h=1080,tcp=:8901,wait"},
			wantSlot: 7,
		},
		{
			name: "NoScreen",
			mockVMClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}
			},
			mockGetPortFunc: func(_ int, _ []int) (int, error) {
				return 7901, nil
			},
			fields: fields{
				ID:          "5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081103",
				Description: "test vm",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID:        81,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
					},
					VMID:             "5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75",
					CPU:              2,
					Mem:              2048,
					MaxWait:          60,
					Restart:          true,
					Screen:           false,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					Sound:            false,
					SoundIn:          "/dev/dsp0",
					SoundOut:         "/dev/dsp0",
					VNCPort:          "AUTO",
					Tablet:           true,
					StoreUEFIVars:    true,
					UTCTime:          true,
					HostBridge:       true,
					ACPI:             true,
					UseHLT:           true,
					ExitOnPause:      true,
					DestroyPowerOff:  true,
					IgnoreUnknownMSR: true,
					KbdLayout:        "default",
					Com1:             true,
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        115200,
					Com2Speed:        115200,
					Com3Speed:        115200,
					Com4Speed:        115200,
					AutoStartDelay:   60,
					DebugPort:        "AUTO",
				},
				ISOs: []*iso.ISO{
					{
						ID: "59445e39-a842-467c-9cb8-4bd4b0a529c7",
					},
				},
				Disks: []*disk.Disk{
					{
						ID: "0e7af864-ed45-4256-947e-871f6ba3a3ac",
					},
				},
			},
			args: args{
				slot: 6,
			},
			wantArgs: []string{},
			wantSlot: 6,
		},
		{
			name: "GetFreeTCPPortFuncError",
			mockVMClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}
			},
			mockGetPortFunc: func(_ int, _ []int) (int, error) {
				return 0, errors.New("some random error") //nolint:goerr113
			},
			fields: fields{
				ID:          "5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "test2024081103",
				Description: "test vm",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID:        81,
						CreatedAt: createUpdateTime,
						UpdatedAt: createUpdateTime,
					},
					VMID:             "5d9a8fe3-d0aa-430d-bc1b-99f3f0c5eb75",
					CPU:              2,
					Mem:              2048,
					MaxWait:          60,
					Restart:          true,
					Screen:           true,
					ScreenWidth:      1920,
					ScreenHeight:     1080,
					Sound:            false,
					SoundIn:          "/dev/dsp0",
					SoundOut:         "/dev/dsp0",
					VNCPort:          "AUTO",
					Tablet:           true,
					StoreUEFIVars:    true,
					UTCTime:          true,
					HostBridge:       true,
					ACPI:             true,
					UseHLT:           true,
					ExitOnPause:      true,
					DestroyPowerOff:  true,
					IgnoreUnknownMSR: true,
					KbdLayout:        "default",
					Com1:             true,
					Com1Dev:          "AUTO",
					Com2Dev:          "AUTO",
					Com3Dev:          "AUTO",
					Com4Dev:          "AUTO",
					Com1Speed:        115200,
					Com2Speed:        115200,
					Com3Speed:        115200,
					Com4Speed:        115200,
					AutoStartDelay:   60,
					DebugPort:        "AUTO",
				},
				ISOs: []*iso.ISO{
					{
						ID: "59445e39-a842-467c-9cb8-4bd4b0a529c7",
					},
				},
				Disks: []*disk.Disk{
					{
						ID: "0e7af864-ed45-4256-947e-871f6ba3a3ac",
					},
				},
			},
			args: args{
				slot: 6,
			},
			wantArgs: []string{},
			wantSlot: 6,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testDB, mock := cirrinadtest.NewMockDB("vmTest")
			testCase.mockVMClosure(testDB, mock)

			GetFreeTCPPortFunc = testCase.mockGetPortFunc

			t.Cleanup(func() { GetFreeTCPPortFunc = util.GetFreeTCPPort })

			testVM := &VM{
				ID:          testCase.fields.ID,
				CreatedAt:   testCase.fields.CreatedAt,
				UpdatedAt:   testCase.fields.UpdatedAt,
				DeletedAt:   testCase.fields.DeletedAt,
				Name:        testCase.fields.Name,
				Description: testCase.fields.Description,
				Status:      testCase.fields.Status,
				BhyvePid:    testCase.fields.BhyvePid,
				VNCPort:     testCase.fields.VNCPort,
				DebugPort:   testCase.fields.DebugPort,
				Config:      testCase.fields.Config,
				ISOs:        testCase.fields.ISOs,
				Disks:       testCase.fields.Disks,
				Com1Dev:     testCase.fields.Com1Dev,
				Com2Dev:     testCase.fields.Com2Dev,
				Com3Dev:     testCase.fields.Com3Dev,
				Com4Dev:     testCase.fields.Com4Dev,
				Com1write:   testCase.fields.Com1write,
				Com2write:   testCase.fields.Com2write,
				Com3write:   testCase.fields.Com3write,
				Com4write:   testCase.fields.Com4write,
			}

			gotArgs, gotSlot := testVM.getVideoArg(testCase.args.slot)

			diff := deep.Equal(gotArgs, testCase.wantArgs)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			diff = deep.Equal(gotSlot, testCase.wantSlot)
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

//nolint:paralleltest
func Test_getNetDevTypeArg(t *testing.T) {
	createUpdateTime := time.Now()

	type args struct {
		netDevType string
		switchID   string
		vmName     string
	}

	tests := []struct {
		name            string
		mockCmdFunc     string
		mockVMClosure   func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		hostIntStubFunc func() ([]net.Interface, error)
		args            args
		wantNetDev      string
		wantNetDevArg   string
		wantErr         bool
	}{
		{
			name:        "badType",
			mockCmdFunc: "Test_getNetDevTypeArgSuccess",
			mockVMClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}
			},
			hostIntStubFunc: StubHostInterfacesSuccess1,
			args: args{
				netDevType: "junk",
				switchID:   "64bdfe13-7f85-4add-9a8c-7a28deb32193",
				vmName:     "unused",
			},
			wantNetDev:    "",
			wantNetDevArg: "",
			wantErr:       true,
		},
		{
			name:        "tap",
			mockCmdFunc: "Test_getNetDevTypeArgSuccess",
			mockVMClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}
			},
			hostIntStubFunc: StubHostInterfacesSuccess1,
			args: args{
				netDevType: "TAP",
				switchID:   "64bdfe13-7f85-4add-9a8c-7a28deb32193",
				vmName:     "unused",
			},
			wantNetDev:    "tap0",
			wantNetDevArg: "tap0",
			wantErr:       false,
		},
		{
			name:        "vmnet",
			mockCmdFunc: "Test_getNetDevTypeArgSuccess",
			mockVMClosure: func(testDB *gorm.DB, _ sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}
			},
			hostIntStubFunc: StubHostInterfacesSuccess1,
			args: args{
				netDevType: "VMNET",
				switchID:   "64bdfe13-7f85-4add-9a8c-7a28deb32193",
				vmName:     "unused",
			},
			wantNetDev:    "vmnet0",
			wantNetDevArg: "vmnet0",
			wantErr:       false,
		},
		{
			name:        "netgraph",
			mockCmdFunc: "Test_getNetDevTypeArgSuccess",
			mockVMClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}
				_switch.Instance = &_switch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("64bdfe13-7f85-4add-9a8c-7a28deb32193").
					WillReturnRows(sqlmock.NewRows(
						[]string{
							"id",
							"created_at",
							"updated_at",
							"deleted_at",
							"name",
							"description",
							"type",
							"uplink",
						}).
						AddRow(
							"64bdfe13-7f85-4add-9a8c-7a28deb32193",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bnet0",
							"some ng switch description",
							"NG",
							"em0",
						))
			},
			hostIntStubFunc: StubHostInterfacesSuccess1,
			args: args{
				netDevType: "NETGRAPH",
				switchID:   "64bdfe13-7f85-4add-9a8c-7a28deb32193",
				vmName:     "someTestVM",
			},
			wantNetDev:    "bnet0,link2",
			wantNetDevArg: "netgraph,path=bnet0:,peerhook=link2,socket=someTestVM",
			wantErr:       false,
		},
		{
			name:        "netgraphError",
			mockCmdFunc: "Test_getNetDevTypeArgError1",
			mockVMClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				Instance = &singleton{ // prevents parallel testing
					vmDB: testDB,
				}
				_switch.Instance = &_switch.Singleton{ // prevents parallel testing
					SwitchDB: testDB,
				}

				mock.ExpectQuery(
					regexp.QuoteMeta(
						"SELECT * FROM `switches` WHERE id = ? AND `switches`.`deleted_at` IS NULL LIMIT 1",
					),
				).
					WithArgs("64bdfe13-7f85-4add-9a8c-7a28deb32193").
					WillReturnRows(sqlmock.NewRows(
						[]string{
							"id",
							"created_at",
							"updated_at",
							"deleted_at",
							"name",
							"description",
							"type",
							"uplink",
						}).
						AddRow(
							"64bdfe13-7f85-4add-9a8c-7a28deb32193",
							createUpdateTime,
							createUpdateTime,
							nil,
							"bnet0",
							"some ng switch description",
							"NG",
							"em0",
						))
			},
			hostIntStubFunc: StubHostInterfacesSuccess1,
			args: args{
				netDevType: "NETGRAPH",
				switchID:   "64bdfe13-7f85-4add-9a8c-7a28deb32193",
				vmName:     "someTestVM",
			},
			wantNetDev:    "",
			wantNetDevArg: "",
			wantErr:       true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			NetInterfacesFunc = testCase.hostIntStubFunc

			t.Cleanup(func() { NetInterfacesFunc = net.Interfaces })

			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)
			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			testDB, mock := cirrinadtest.NewMockDB("vmTest")
			testCase.mockVMClosure(testDB, mock)

			gotNetDev, gotNetDevArg, err := getNetDevTypeArg(
				testCase.args.netDevType, testCase.args.switchID, testCase.args.vmName,
			)
			if (err != nil) != testCase.wantErr {
				t.Errorf("getNetDevTypeArg() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if gotNetDev != testCase.wantNetDev {
				t.Errorf("getNetDevTypeArg() gotNetDev = %v, wantNetDev %v", gotNetDev, testCase.wantNetDev)
			}

			if gotNetDevArg != testCase.wantNetDevArg {
				t.Errorf("getNetDevTypeArg() gotNetDevArg = %v, wantNetDevArg %v", gotNetDevArg, testCase.wantNetDevArg)
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

//nolint:paralleltest,maintidx
func TestVM_getNetArgs(t *testing.T) {
	createUpdateTime := time.Now()

	type fields struct {
		ID          string
		CreatedAt   time.Time
		UpdatedAt   time.Time
		DeletedAt   gorm.DeletedAt
		Name        string
		Description string
		Status      StatusType
		BhyvePid    uint32
		VNCPort     int32
		DebugPort   int32
		Config      Config
		ISOs        []*iso.ISO
		Disks       []*disk.Disk
		Com1Dev     string
		Com2Dev     string
		Com3Dev     string
		Com4Dev     string
		Com1write   bool
		Com2write   bool
		Com3write   bool
		Com4write   bool
	}

	type args struct {
		slot int
	}

	tests := []struct {
		name            string
		hostIntStubFunc func() ([]net.Interface, error)
		mockClosure     func(testDB *gorm.DB, mock sqlmock.Sqlmock)
		fields          fields
		args            args
		wantNetArgs     []string
		wantSlot        int
	}{
		{
			name:            "noNics",
			hostIntStubFunc: StubHostInterfacesSuccess1,
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(226).
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"mac",
						"net_type",
						"net_dev_type",
						"switch_id",
						"net_dev",
						"rate_limit",
						"rate_in",
						"rate_out",
						"inst_bridge",
						"inst_epair",
						"config_id",
					}))
			},
			fields: fields{
				ID:          "1324e9c6-cc63-4f53-9a16-6fc74b0b24d5",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "sundayVM",
				Description: "a test VM created on a sunday",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID:        226,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID: "1324e9c6-cc63-4f53-9a16-6fc74b0b24d5",
					CPU:  2,
					Mem:  2048,
				},
			},
			args: args{
				slot: 4,
			},
			wantNetArgs: nil,
			wantSlot:    4,
		},
		{
			name:            "oneNic",
			hostIntStubFunc: StubHostInterfacesSuccess1,
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(226).
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"mac",
						"net_type",
						"net_dev_type",
						"switch_id",
						"net_dev",
						"rate_limit",
						"rate_in",
						"rate_out",
						"inst_bridge",
						"inst_epair",
						"config_id",
					}).
						AddRow(
							"375ab4fb-a829-432a-bb58-ba38aa76498a",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aSundayNic",
							"a cool description of a sunday evening NIC",
							"00:11:22:55:44:33",
							"VIRTIONET",
							"TAP",
							"a81d4b08-3912-4831-8965-9e70ce4321f1",
							"",
							false,
							0,
							0,
							nil,
							nil,
							226,
						))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(226, "a cool description of a sunday evening NIC", "", "", "00:11:22:55:44:33", "aSundayNic", "tap0", "TAP", //nolint:lll
						"VIRTIONET", 0, false, 0, "a81d4b08-3912-4831-8965-9e70ce4321f1", sqlmock.AnyArg(),
						"375ab4fb-a829-432a-bb58-ba38aa76498a").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			fields: fields{
				ID:          "1324e9c6-cc63-4f53-9a16-6fc74b0b24d5",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "sundayVM",
				Description: "a test VM created on a sunday",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID:        226,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID: "1324e9c6-cc63-4f53-9a16-6fc74b0b24d5",
					CPU:  2,
					Mem:  2048,
				},
			},
			args: args{
				slot: 4,
			},
			wantNetArgs: []string{"-s", "4,virtio-net,tap0,mac=00:11:22:55:44:33"},
			wantSlot:    5,
		},
		{
			name:            "saveError",
			hostIntStubFunc: StubHostInterfacesSuccess1,
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(226).
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"mac",
						"net_type",
						"net_dev_type",
						"switch_id",
						"net_dev",
						"rate_limit",
						"rate_in",
						"rate_out",
						"inst_bridge",
						"inst_epair",
						"config_id",
					}).
						AddRow(
							"375ab4fb-a829-432a-bb58-ba38aa76498a",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aSundayNic",
							"a cool description of a sunday evening NIC",
							"00:11:22:55:44:33",
							"VIRTIONET",
							"TAP",
							"a81d4b08-3912-4831-8965-9e70ce4321f1",
							"",
							false,
							0,
							0,
							nil,
							nil,
							226,
						))

				mock.ExpectBegin()
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE `vm_nics` SET `config_id`=?,`description`=?,`inst_bridge`=?,`inst_epair`=?,`mac`=?,`name`=?,`net_dev`=?,`net_dev_type`=?,`net_type`=?,`rate_in`=?,`rate_limit`=?,`rate_out`=?,`switch_id`=?,`updated_at`=? WHERE `vm_nics`.`deleted_at` IS NULL AND `id` = ?"), //nolint:lll
				).
					WithArgs(226, "a cool description of a sunday evening NIC", "", "", "00:11:22:55:44:33", "aSundayNic", "tap0", "TAP", //nolint:lll
						"VIRTIONET", 0, false, 0, "a81d4b08-3912-4831-8965-9e70ce4321f1", sqlmock.AnyArg(),
						"375ab4fb-a829-432a-bb58-ba38aa76498a").
					WillReturnError(gorm.ErrInvalidData)
				mock.ExpectRollback()
			},
			fields: fields{
				ID:          "1324e9c6-cc63-4f53-9a16-6fc74b0b24d5",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "sundayVM",
				Description: "a test VM created on a sunday",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID:        226,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID: "1324e9c6-cc63-4f53-9a16-6fc74b0b24d5",
					CPU:  2,
					Mem:  2048,
				},
			},
			args: args{
				slot: 4,
			},
			wantNetArgs: []string{},
			wantSlot:    4,
		},
		{
			name:            "getNetDevTypeArgError",
			hostIntStubFunc: StubHostInterfacesSuccess1,
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(226).
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"mac",
						"net_type",
						"net_dev_type",
						"switch_id",
						"net_dev",
						"rate_limit",
						"rate_in",
						"rate_out",
						"inst_bridge",
						"inst_epair",
						"config_id",
					}).
						AddRow(
							"375ab4fb-a829-432a-bb58-ba38aa76498a",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aSundayNic",
							"a cool description of a sunday evening NIC",
							"00:11:22:55:44:33",
							"VIRTIONET",
							"junk",
							"a81d4b08-3912-4831-8965-9e70ce4321f1",
							"",
							false,
							0,
							0,
							nil,
							nil,
							226,
						))
			},
			fields: fields{
				ID:          "1324e9c6-cc63-4f53-9a16-6fc74b0b24d5",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "sundayVM",
				Description: "a test VM created on a sunday",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID:        226,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID: "1324e9c6-cc63-4f53-9a16-6fc74b0b24d5",
					CPU:  2,
					Mem:  2048,
				},
			},
			args: args{
				slot: 4,
			},
			wantNetArgs: []string{},
			wantSlot:    4,
		},
		{
			name:            "getNetTypeArgError",
			hostIntStubFunc: StubHostInterfacesSuccess1,
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(226).
					WillReturnRows(sqlmock.NewRows([]string{
						"id",
						"created_at",
						"updated_at",
						"deleted_at",
						"name",
						"description",
						"mac",
						"net_type",
						"net_dev_type",
						"switch_id",
						"net_dev",
						"rate_limit",
						"rate_in",
						"rate_out",
						"inst_bridge",
						"inst_epair",
						"config_id",
					}).
						AddRow(
							"375ab4fb-a829-432a-bb58-ba38aa76498a",
							createUpdateTime,
							createUpdateTime,
							nil,
							"aSundayNic",
							"a cool description of a sunday evening NIC",
							"00:11:22:55:44:33",
							"junkHere",
							"TAP",
							"a81d4b08-3912-4831-8965-9e70ce4321f1",
							"",
							false,
							0,
							0,
							nil,
							nil,
							226,
						))
			},
			fields: fields{
				ID:          "1324e9c6-cc63-4f53-9a16-6fc74b0b24d5",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "sundayVM",
				Description: "a test VM created on a sunday",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID:        226,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID: "1324e9c6-cc63-4f53-9a16-6fc74b0b24d5",
					CPU:  2,
					Mem:  2048,
				},
			},
			args: args{
				slot: 4,
			},
			wantNetArgs: []string{},
			wantSlot:    4,
		},
		{
			name:            "getNicsErr",
			hostIntStubFunc: StubHostInterfacesSuccess1,
			mockClosure: func(testDB *gorm.DB, mock sqlmock.Sqlmock) {
				vmnic.Instance = &vmnic.Singleton{ // prevents parallel testing
					VMNicDB: testDB,
				}
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM `vm_nics` WHERE config_id = ? AND `vm_nics`.`deleted_at` IS NULL"),
				).
					WithArgs(226).
					WillReturnError(gorm.ErrInvalidData)
			},
			fields: fields{
				ID:          "1324e9c6-cc63-4f53-9a16-6fc74b0b24d5",
				CreatedAt:   createUpdateTime,
				UpdatedAt:   createUpdateTime,
				Name:        "sundayVM",
				Description: "a test VM created on a sunday",
				Status:      "STOPPED",
				Config: Config{
					Model: gorm.Model{
						ID:        226,
						CreatedAt: time.Time{},
						UpdatedAt: time.Time{},
						DeletedAt: gorm.DeletedAt{
							Time:  time.Time{},
							Valid: false,
						},
					},
					VMID: "1324e9c6-cc63-4f53-9a16-6fc74b0b24d5",
					CPU:  2,
					Mem:  2048,
				},
			},
			args: args{
				slot: 4,
			},
			wantNetArgs: []string{},
			wantSlot:    4,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			NetInterfacesFunc = testCase.hostIntStubFunc

			t.Cleanup(func() { NetInterfacesFunc = net.Interfaces })

			testDB, mock := cirrinadtest.NewMockDB("nicTest")

			testCase.mockClosure(testDB, mock)

			testVM := &VM{
				ID:          testCase.fields.ID,
				CreatedAt:   testCase.fields.CreatedAt,
				UpdatedAt:   testCase.fields.UpdatedAt,
				DeletedAt:   testCase.fields.DeletedAt,
				Name:        testCase.fields.Name,
				Description: testCase.fields.Description,
				Status:      testCase.fields.Status,
				BhyvePid:    testCase.fields.BhyvePid,
				VNCPort:     testCase.fields.VNCPort,
				DebugPort:   testCase.fields.DebugPort,
				Config:      testCase.fields.Config,
				ISOs:        testCase.fields.ISOs,
				Disks:       testCase.fields.Disks,
				Com1Dev:     testCase.fields.Com1Dev,
				Com2Dev:     testCase.fields.Com2Dev,
				Com3Dev:     testCase.fields.Com3Dev,
				Com4Dev:     testCase.fields.Com4Dev,
				Com1write:   testCase.fields.Com1write,
				Com2write:   testCase.fields.Com2write,
				Com3write:   testCase.fields.Com3write,
				Com4write:   testCase.fields.Com4write,
			}

			gotNetARgs, gotSlot := testVM.getNetArgs(testCase.args.slot)

			diff := deep.Equal(gotNetARgs, testCase.wantNetArgs)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}

			diff = deep.Equal(gotSlot, testCase.wantSlot)
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

// test helpers from here down

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

//nolint:paralleltest
func Test_getNetDevTypeArgSuccess(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	ngctlOutput := `  Name: bnet0           Type: bridge          ID: 0000000b   Num hooks: 2
  Local hook      Peer name       Peer type    Peer ID         Peer hook      
  ----------      ---------       ---------    -------         ---------      
  link1           em0             ether        00000002        upper          
  link0           em0             ether        00000002        lower          
`

	fmt.Print(ngctlOutput) //nolint:forbidigo
	os.Exit(0)
}

//nolint:paralleltest
func Test_getNetDevTypeArgError1(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	os.Exit(1)
}
