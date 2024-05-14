package vm

import "testing"

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
