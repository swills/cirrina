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
