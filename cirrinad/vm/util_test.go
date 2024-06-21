package vm

import (
	"fmt"
	"os"
	"testing"

	"cirrina/cirrinad/cirrinadtest"
	"cirrina/cirrinad/util"
)

func Test_parsePsJSONOutput(t *testing.T) {
	type args struct {
		psJSONOutput []byte
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "valid1",
			args:    args{psJSONOutput: []byte(`{"process-information": {"process": [{"pid":"83821","terminal-name":"27 ","state":"I","cpu-time":"0:00.02","command":"/usr/local/bin/sudo /usr/bin/protect /usr/sbin/bhyve -U 50f994e3-5c30-4d4d-a330-f5c46106cffe -A -H -P -D -w -u -l bootrom,/usr/local/share/uefi-firmware/BHYVE_UEFI.fd,/var/tmp/cirrinad/state/something"}]}}`)}, //nolint:lll
			want:    "/usr/local/bin/sudo",
			wantErr: false,
		},
		{
			name:    "valid2",
			args:    args{psJSONOutput: []byte(`{"process-information": {"process": [{"pid":"71004","terminal-name":"28 ","state":"S","cpu-time":"0:00.03","command":"/usr/sbin/bhyve -U f5b761a1-8193-4db3-a914-b37edc848d29 -A -H -P -D -w -u -l bootrom,/usr/local/share/uefi-firmware/BHYVE_UEFI.fd,/var/tmp/cirrinad/state/something"}]}}`)}, //nolint:lll
			want:    "/usr/sbin/bhyve",
			wantErr: false,
		},
		{
			name:    "valid3",
			args:    args{psJSONOutput: []byte(`{"process-information": {"process": [{"pid":"85540","terminal-name":"28 ","state":"SC","cpu-time":"1:41.54","command":"bhyve: test2024010401 (bhyve)"}]}}`)}, //nolint:lll
			want:    "bhyve:",
			wantErr: false,
		},
		{
			name:    "invalid1",
			args:    args{psJSONOutput: []byte(``)},
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid2",
			args:    args{psJSONOutput: []byte(`{"process-information": 1}`)},
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid3",
			args:    args{psJSONOutput: []byte(`{"process-information": {"blah": 1}}`)},
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid4",
			args:    args{psJSONOutput: []byte(`{"process-information": {"process": [1,2]}}`)},
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid5",
			args:    args{psJSONOutput: []byte(`{"process-information": {"process": [1]}}`)},
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid6",
			args:    args{psJSONOutput: []byte(`{"process-information": {"process": [{"number": 1}]}}`)},
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid7",
			args:    args{psJSONOutput: []byte(`{"process-information": {"process": [{"pid":"83821","terminal-name":"27 ","state":"I","cpu-time":"0:00.02","command":123}]}}`)}, //nolint:lll
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid8",
			args:    args{psJSONOutput: []byte(`{"process-information": {"process": [{"pid":"83821","terminal-name":"27 ","state":"I","cpu-time":"0:00.02","command":""}]}}`)}, //nolint:lll
			want:    "",
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := parsePsJSONOutput(testCase.args.psJSONOutput)
			if (err != nil) != testCase.wantErr {
				t.Errorf("parsePsJSONOutput() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("parsePsJSONOutput() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest
func Test_findProcName(t *testing.T) {
	type args struct {
		pid uint32
	}

	tests := []struct {
		name        string
		mockCmdFunc string
		args        args
		want        string
	}{
		{
			name:        "Sleep",
			mockCmdFunc: "Test_findProcNameSleep",
			args:        args{pid: 12345},
			want:        "sleep",
		},
		{
			name:        "Error",
			mockCmdFunc: "Test_findProcNameError",
			args:        args{pid: 12345},
			want:        "",
		},
		{
			name:        "BadJson",
			mockCmdFunc: "Test_findProcNameBadJson",
			args:        args{pid: 12345},
			want:        "",
		},
	}

	for _, testCase := range tests {
		testCase := testCase // shadow to avoid loop variable capture
		t.Run(testCase.name, func(t *testing.T) {
			// prevents parallel testing
			fakeCommand := cirrinadtest.MakeFakeCommand(testCase.mockCmdFunc)

			util.SetupTestCmd(fakeCommand)

			t.Cleanup(func() { util.TearDownTestCmd() })

			got := findProcName(testCase.args.pid)
			if got != testCase.want {
				t.Errorf("findProcName() = %v, want %v", got, testCase.want)
			}
		})
	}
}

// test helpers from here down

//nolint:paralleltest
func Test_findProcNameSleep(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 5 && cmdWithArgs[0] == "/bin/ps" && cmdWithArgs[1] == "--libxo" && cmdWithArgs[2] == "json" && cmdWithArgs[3] == "-p" { //nolint:lll
		fmt.Printf("{\"process-information\": {\"process\": [{\"pid\":\"12345\",\"terminal-name\":\"28 \",\"state\":\"SC+\",\"cpu-time\":\"0:00.00\",\"command\":\"sleep 1024\"}]}}\n") //nolint:lll,forbidigo
		os.Exit(0)
	}

	os.Exit(1)
}

//nolint:paralleltest
func Test_findProcNameError(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 5 && cmdWithArgs[0] == "/bin/ps" && cmdWithArgs[1] == "--libxo" && cmdWithArgs[2] == "json" && cmdWithArgs[3] == "-p" { //nolint:lll
		os.Exit(1)
	}

	os.Exit(0)
}

//nolint:paralleltest
func Test_findProcNameBadJson(_ *testing.T) {
	if !cirrinadtest.IsTestEnv() {
		return
	}

	cmdWithArgs := os.Args[3:]

	if len(cmdWithArgs) == 5 && cmdWithArgs[0] == "/bin/ps" && cmdWithArgs[1] == "--libxo" && cmdWithArgs[2] == "json" && cmdWithArgs[3] == "-p" { //nolint:lll
		fmt.Printf("{\"process-information\": {\"process\": [{\"pid\":\"12345\",\"terminal-name\":\"28 \",\"state\":\"SC+\",\"cpu-time\":\"0:00.00\",\"comm") //nolint:lll,forbidigo
		os.Exit(0)
	}

	os.Exit(1)
}
