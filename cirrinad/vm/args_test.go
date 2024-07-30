package vm

import (
	"testing"

	"github.com/go-test/deep"
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
