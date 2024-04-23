package main

import "testing"

func Test_nicCloneValidateMac(t *testing.T) {
	type args struct {
		newMac string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "validMac",
			args:    args{"00:11:22:33:44:55"},
			wantErr: false,
		},
		{
			name:    "invalidMac",
			args:    args{"00:11:22:33:44:55:66"},
			wantErr: true,
		},
		{
			name:    "broadcastMac",
			args:    args{"FF:FF:FF:FF:FF:FF"},
			wantErr: true,
		},
		{
			name:    "sillyInfiniband",
			args:    args{"00-00-00-00-fe-80-00-00-00-00-00-00-02-00-5e-10-00-00-00-01"},
			wantErr: true,
		},
		{
			name:    "aMulticastMac",
			args:    args{"11:22:33:44:55:66"},
			wantErr: true,
		},
		{
			name:    "empty",
			args:    args{""},
			wantErr: false,
		},
		{
			name:    "AUTO",
			args:    args{"AUTO"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := nicCloneValidateMac(tt.args.newMac); (err != nil) != tt.wantErr {
				t.Errorf("nicCloneValidateMac() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
