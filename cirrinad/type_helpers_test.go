package main

import (
	"testing"

	"github.com/go-test/deep"

	"cirrina/cirrina"
)

func Test_mapDiskDevTypeTypeToDBString(t *testing.T) {
	type args struct {
		diskDevType cirrina.DiskDevType
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "file",
			args:    args{diskDevType: cirrina.DiskDevType_FILE},
			want:    "FILE",
			wantErr: false,
		},
		{
			name:    "zvol",
			args:    args{diskDevType: cirrina.DiskDevType_ZVOL},
			want:    "ZVOL",
			wantErr: false,
		},
		{
			name:    "error",
			args:    args{diskDevType: -1},
			want:    "",
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := mapDiskDevTypeTypeToDBString(testCase.args.diskDevType)
			if (err != nil) != testCase.wantErr {
				t.Errorf("mapDiskDevTypeTypeToDBString() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func Test_mapDiskDevTypeDBStringToType(t *testing.T) {
	type args struct {
		diskDevType string
	}

	tests := []struct {
		name    string
		args    args
		want    *cirrina.DiskDevType
		wantErr bool
	}{
		{
			name:    "file",
			args:    args{diskDevType: "FILE"},
			want:    func() *cirrina.DiskDevType { f := cirrina.DiskDevType_FILE; return &f }(), //nolint:nlreturn
			wantErr: false,
		},
		{
			name:    "zvol",
			args:    args{diskDevType: "ZVOL"},
			want:    func() *cirrina.DiskDevType { f := cirrina.DiskDevType_ZVOL; return &f }(), //nolint:nlreturn
			wantErr: false,
		},
		{
			name:    "error",
			args:    args{diskDevType: "garbage"},
			want:    nil,
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := mapDiskDevTypeDBStringToType(testCase.args.diskDevType)
			if (err != nil) != testCase.wantErr {
				t.Errorf("mapDiskDevTypeDBStringToType() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}

func Test_mapDiskTypeTypeToDBString(t *testing.T) {
	type args struct {
		diskType cirrina.DiskType
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "nvme",
			args:    args{diskType: cirrina.DiskType_NVME},
			want:    "NVME",
			wantErr: false,
		},
		{
			name:    "ahcihd",
			args:    args{diskType: cirrina.DiskType_AHCIHD},
			want:    "AHCI-HD",
			wantErr: false,
		},
		{
			name:    "virtioblk",
			args:    args{diskType: cirrina.DiskType_VIRTIOBLK},
			want:    "VIRTIO-BLK",
			wantErr: false,
		},
		{
			name:    "error",
			args:    args{diskType: -1},
			want:    "",
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := mapDiskTypeTypeToDBString(testCase.args.diskType)
			if (err != nil) != testCase.wantErr {
				t.Errorf("mapDiskTypeTypeToDBString() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			if got != testCase.want {
				t.Errorf("mapDiskTypeTypeToDBString() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func Test_mapDiskTypeDBStringToType(t *testing.T) {
	type args struct {
		diskType string
	}

	tests := []struct {
		name    string
		args    args
		want    *cirrina.DiskType
		wantErr bool
	}{
		{
			name:    "nvme",
			args:    args{diskType: "NVME"},
			want:    func() *cirrina.DiskType { f := cirrina.DiskType_NVME; return &f }(), //nolint:nlreturn
			wantErr: false,
		},
		{
			name:    "ahcihd",
			args:    args{diskType: "AHCI-HD"},
			want:    func() *cirrina.DiskType { f := cirrina.DiskType_AHCIHD; return &f }(), //nolint:nlreturn
			wantErr: false,
		},
		{
			name:    "virtioblk",
			args:    args{diskType: "VIRTIO-BLK"},
			want:    func() *cirrina.DiskType { f := cirrina.DiskType_VIRTIOBLK; return &f }(), //nolint:nlreturn
			wantErr: false,
		},
		{
			name:    "error",
			args:    args{diskType: "garbage"},
			want:    nil,
			wantErr: true,
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := mapDiskTypeDBStringToType(testCase.args.diskType)
			if (err != nil) != testCase.wantErr {
				t.Errorf("mapDiskTypeDBStringToType() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}

			diff := deep.Equal(got, testCase.want)
			if diff != nil {
				t.Errorf("compare failed: %v", diff)
			}
		})
	}
}
