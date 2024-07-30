package main

import (
	"testing"

	"github.com/go-test/deep"

	"cirrina/cirrina"
)

func Test_mapSwitchTypeTypeToDBString(t *testing.T) {
	type args struct {
		switchType cirrina.SwitchType
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "junk",
			args: args{
				switchType: cirrina.SwitchType(-1),
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "IF",
			args: args{
				switchType: cirrina.SwitchType_IF,
			},
			want:    "IF",
			wantErr: false,
		},
		{
			name: "NG",
			args: args{
				switchType: cirrina.SwitchType_NG,
			},
			want:    "NG",
			wantErr: false,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := mapSwitchTypeTypeToDBString(testCase.args.switchType)
			if (err != nil) != testCase.wantErr {
				t.Errorf("mapSwitchTypeTypeToDBString() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("mapSwitchTypeTypeToDBString() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func Test_mapSwitchTypeDBStringToType(t *testing.T) {
	type args struct {
		switchType string
	}

	tests := []struct {
		name    string
		args    args
		want    *cirrina.SwitchType
		wantErr bool
	}{
		{
			name: "junk",
			args: args{
				switchType: "junk",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "IF",
			args: args{
				switchType: "IF",
			},
			want:    func() *cirrina.SwitchType { n := cirrina.SwitchType_IF; return &n }(), //nolint:nlreturn
			wantErr: false,
		},
		{
			name: "NG",
			args: args{
				switchType: "NG",
			},
			want:    func() *cirrina.SwitchType { n := cirrina.SwitchType_NG; return &n }(), //nolint:nlreturn
			wantErr: false,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := mapSwitchTypeDBStringToType(testCase.args.switchType)
			if (err != nil) != testCase.wantErr {
				t.Errorf("mapSwitchTypeDBStringToType() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}
