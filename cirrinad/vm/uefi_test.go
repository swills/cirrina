package vm

import (
	"errors"
	"strings"
	"testing"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

//nolint:paralleltest
func TestVM_createUefiVarsFile(t *testing.T) {
	type fields struct {
		Name string
	}

	tests := []struct {
		name        string
		fields      fields
		wantPath    bool
		wantPathErr bool
		wantFile    bool
		wantFileErr bool
	}{
		{
			name: "dirExistsErr",
			fields: fields{
				Name: "someVM",
			},
			wantPathErr: true,
		},
		{
			name: "dirAlreadyExists",
			fields: fields{
				Name: "someVM",
			},
			wantPath: true,
		},
		{
			name: "fileExistsErr",
			fields: fields{
				Name: "someVM",
			},
			wantPath:    true,
			wantFileErr: true,
		},
		{
			name: "fileAlreadyExists",
			fields: fields{
				Name: "someVM",
			},
			wantPath: true,
			wantFile: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			PathExistsFunc = func(s string) (bool, error) {
				if strings.Contains(s, "BHYVE_UEFI_VARS.fd") {
					if testCase.wantFileErr {
						return true, errors.New("another error") //nolint:goerr113
					}

					if testCase.wantFile {
						return true, nil
					}
				}

				if testCase.wantPathErr {
					return true, errors.New("another error") //nolint:goerr113
				}

				if testCase.wantPath {
					return true, nil
				}

				return false, nil
			}

			config.Config.Disk.VM.Path.State = "/var/tmp/cirrinad/state/"

			t.Cleanup(func() { PathExistsFunc = util.PathExists })

			vm := &VM{
				Name: testCase.fields.Name,
			}
			vm.createUefiVarsFile()
		})
	}
}

//nolint:paralleltest
func TestVM_DeleteUEFIState(t *testing.T) {
	type fields struct {
		Name string
	}

	tests := []struct {
		name        string
		fields      fields
		wantErr     bool
		wantPath    bool
		wantPathErr bool
	}{
		{
			name: "doesNotExist",
			fields: fields{
				Name: "anotherVM",
			},
			wantErr: false,
		},
		{
			name: "existsErr",
			fields: fields{
				Name: "anotherVM",
			},
			wantErr:     true,
			wantPathErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			PathExistsFunc = func(_ string) (bool, error) {
				if testCase.wantPathErr {
					return true, errors.New("another error") //nolint:goerr113
				}

				if testCase.wantPath {
					return true, nil
				}

				return false, nil
			}

			config.Config.Disk.VM.Path.State = "/var/tmp/cirrinad/state/"

			t.Cleanup(func() { PathExistsFunc = util.PathExists })

			vm := &VM{
				Name: testCase.fields.Name,
			}

			err := vm.DeleteUEFIState()
			if (err != nil) != testCase.wantErr {
				t.Errorf("DeleteUEFIState() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}
