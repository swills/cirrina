package util

import (
	"testing"
)

func Test_parseDiskSizeSuffix(t *testing.T) {
	type args struct {
		diskSize string
	}
	tests := []struct {
		name            string
		args            args
		wantTrimmedsize string
		wantMultiplier  uint64
	}{
		{
			name:            "1",
			args:            args{diskSize: "1"},
			wantTrimmedsize: "1",
			wantMultiplier:  1,
		},
		{
			name:            "2b",
			args:            args{diskSize: "2b"},
			wantTrimmedsize: "2",
			wantMultiplier:  1,
		},
		{
			name:            "3B",
			args:            args{diskSize: "3B"},
			wantTrimmedsize: "3",
			wantMultiplier:  1,
		},
		{
			name:            "4k",
			args:            args{diskSize: "4k"},
			wantTrimmedsize: "4",
			wantMultiplier:  1024,
		},
		{
			name:            "5K",
			args:            args{diskSize: "5K"},
			wantTrimmedsize: "5",
			wantMultiplier:  1024,
		},
		{
			name:            "6m",
			args:            args{diskSize: "6m"},
			wantTrimmedsize: "6",
			wantMultiplier:  1024 * 1024,
		},
		{
			name:            "7M",
			args:            args{diskSize: "7M"},
			wantTrimmedsize: "7",
			wantMultiplier:  1024 * 1024,
		},
		{
			name:            "8g",
			args:            args{diskSize: "8g"},
			wantTrimmedsize: "8",
			wantMultiplier:  1024 * 1024 * 1024,
		},
		{
			name:            "9G",
			args:            args{diskSize: "9G"},
			wantTrimmedsize: "9",
			wantMultiplier:  1024 * 1024 * 1024,
		},
		{
			name:            "10t",
			args:            args{diskSize: "10t"},
			wantTrimmedsize: "10",
			wantMultiplier:  1024 * 1024 * 1024 * 1024,
		},
		{
			name:            "11T",
			args:            args{diskSize: "11T"},
			wantTrimmedsize: "11",
			wantMultiplier:  1024 * 1024 * 1024 * 1024,
		},
		{
			name:            "12asdf",
			args:            args{diskSize: "12asdf"},
			wantTrimmedsize: "12asdf",
			wantMultiplier:  1,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			trimmedSize, multiplier := parseDiskSizeSuffix(testCase.args.diskSize)
			if trimmedSize != testCase.wantTrimmedsize {
				t.Errorf("parseDiskSizeSuffix() trimmedSize = %v, want_trimmedSize %v", trimmedSize, testCase.wantTrimmedsize)
			}
			if multiplier != testCase.wantMultiplier {
				t.Errorf("parseDiskSizeSuffix() multiplier = %v, want_trimmedSize %v", multiplier, testCase.wantMultiplier)
			}
		})
	}
}

func Test_ParseDiskSize(t *testing.T) {
	type args struct {
		diskSize string
	}
	tests := []struct {
		name    string
		args    args
		want    uint64
		wantErr bool
	}{
		{
			name:    "1024M",
			args:    args{diskSize: "1024M"},
			want:    1024 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "1024T",
			args:    args{diskSize: "1024T"},
			want:    0,
			wantErr: true,
		},
		{
			name:    "overflow1",
			args:    args{diskSize: "2345678901T"},
			want:    0,
			wantErr: true,
		},
		{
			name:    "10T",
			args:    args{diskSize: "10T"},
			want:    10 * 1024 * 1024 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "10asdf",
			args:    args{diskSize: "10asdf"},
			want:    0,
			wantErr: true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := ParseDiskSize(testCase.args.diskSize)
			if (err != nil) != testCase.wantErr {
				t.Errorf("ParseDiskSize() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}
			if got != testCase.want {
				t.Errorf("ParseDiskSize() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func Test_multiplyWillOverflow(t *testing.T) {
	type args struct {
		xVal uint64
		yVal uint64
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nooverflow1",
			args: args{xVal: 2345, yVal: 6789},
			want: false,
		},
		{
			name: "nooverflow2",
			args: args{xVal: 1, yVal: 6789},
			want: false,
		},
		{
			name: "nooverflow3",
			args: args{xVal: 1234, yVal: 1},
			want: false,
		},
		{
			name: "nooverflow4",
			args: args{xVal: 2345678, yVal: 9012345},
			want: false,
		},
		{
			name: "overflow5",
			args: args{xVal: 2345678901, yVal: 9012345678},
			want: true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if got := multiplyWillOverflow(testCase.args.xVal, testCase.args.yVal); got != testCase.want {
				t.Errorf("multiplyWillOverflow() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestMacIsBroadcast(t *testing.T) {
	type args struct {
		macAddress string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "validMac",
			args:    args{"00:11:22:33:44:55"},
			want:    false,
			wantErr: false,
		},
		{
			name:    "invalidMac",
			args:    args{"00:11:22:33:44:55:66"},
			want:    false,
			wantErr: true,
		},
		{
			name:    "broadcastMac",
			args:    args{"FF:FF:FF:FF:FF:FF"},
			want:    true,
			wantErr: false,
		},
		{
			name:    "sillyInfiniband",
			args:    args{"00-00-00-00-fe-80-00-00-00-00-00-00-02-00-5e-10-00-00-00-01"},
			want:    false,
			wantErr: true,
		},
		{
			name:    "aMulticastMac",
			args:    args{"11:22:33:44:55:66"},
			want:    false,
			wantErr: false,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := MacIsBroadcast(testCase.args.macAddress)
			if (err != nil) != testCase.wantErr {
				t.Errorf("MacIsBroadcast() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}
			if got != testCase.want {
				t.Errorf("MacIsBroadcast() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestMacIsMulticast(t *testing.T) {
	type args struct {
		macAddress string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "validMac",
			args:    args{"00:11:22:33:44:55"},
			want:    false,
			wantErr: false,
		},
		{
			name:    "invalidMac",
			args:    args{"00:11:22:33:44:55:66"},
			want:    false,
			wantErr: true,
		},
		{
			name:    "broadcastMac",
			args:    args{"FF:FF:FF:FF:FF:FF"},
			want:    true,
			wantErr: false,
		},
		{
			name:    "sillyInfiniband",
			args:    args{"00-00-00-00-fe-80-00-00-00-00-00-00-02-00-5e-10-00-00-00-01"},
			want:    false,
			wantErr: true,
		},
		{
			name:    "aMulticastMac",
			args:    args{"11:22:33:44:55:66"},
			want:    true,
			wantErr: false,
		},
		{
			name:    "unicodeMac",
			args:    args{"00:11:22:33:44:аа"},
			want:    false,
			wantErr: true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := MacIsMulticast(testCase.args.macAddress)
			if (err != nil) != testCase.wantErr {
				t.Errorf("MacIsMulticast() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}
			if got != testCase.want {
				t.Errorf("MacIsMulticast() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestValidVMName(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "empty",
			args: args{""},
			want: false,
		},
		{
			name: "validupper",
			args: args{"A"},
			want: true,
		},
		{
			name: "validupper2",
			args: args{"THEQUICKBROWNFOXJUMPSOVERTHELAZYDOG"},
			want: true,
		},
		{
			name: "validlower",
			args: args{"a"},
			want: true,
		},
		{
			name: "validlower2",
			args: args{"thequickbrownfoxjumpsoverthelazydog"},
			want: true,
		},
		{
			name: "validlower3",
			args: args{"abc"},
			want: true,
		},
		{
			name: "validnumber",
			args: args{"1"},
			want: true,
		},
		{
			name: "validnumber2",
			args: args{"0123456789THEQUICKBROWNFOXJUMPSOVERTHELAZYDOGthequickbrownfoxjumpsoverthelazydog"},
			want: true,
		},
		{
			name: "validunder",
			args: args{"_"},
			want: true,
		},
		{
			name: "validunder2",
			args: args{"_0123456789"},
			want: true,
		},
		{
			name: "validunder3",
			args: args{"_0123456789thequickbrownfoxjumpsoverthelazydog"},
			want: true,
		},
		{
			name: "validnumber4",
			args: args{"_0123456789thequickbrownfoxjumpsoverthelazydogTHEQUICKBROWNFOXJUMPSOVERTHELAZYDOG"},
			want: true,
		},
		{
			name: "validdash1",
			args: args{"-_0123456789"},
			want: true,
		},
		{
			name: "validdash2",
			args: args{"-_0123456789thequickbrownfoxjumpsoverthelazydog"},
			want: true,
		},
		{
			name: "validdash3",
			args: args{"-_0123456789thequickbrownfoxjumpsoverthelazydogTHEQUICKBROWNFOXJUMPSOVERTHELAZYDOG"},
			want: true,
		},
		{
			name: "validdash4",
			args: args{"--a-__a-a-__90123"},
			want: true,
		},
		{
			name: "invalid1",
			args: args{"abc9123asdf-@"},
			want: false,
		},
		{
			name: "invalid2",
			args: args{"abc9123asdf-#"},
			want: false,
		},
		{
			name: "invalid3",
			args: args{"abc9123asdf-)"},
			want: false,
		},
		{
			name: "invalid4",
			args: args{"abc9123asdf-("},
			want: false,
		},
		{
			name: "invalid5",
			args: args{"abc9123asdf-&"},
			want: false,
		},
		{
			name: "invalid6",
			args: args{"abc9123asdf-$"},
			want: false,
		},
		{
			name: "invalid7",
			args: args{"abc9123asdf-$"},
			want: false,
		},
		{
			name: "invalid8",
			args: args{"abc9123 asdf-$"},
			want: false,
		},
		{
			name: "invalid9",
			args: args{"ab.c"},
			want: false,
		},
		{
			name: "invalid10",
			args: args{"a..b"},
			want: false,
		},
		{
			name: "invalidunicode1",
			args: args{"aа"},
			want: false,
		},
		{
			name: "invalidunicode2",
			args: args{"с"},
			want: false,
		},
		{
			name: "invalidunicode3",
			args: args{"ԁ"},
			want: false,
		},
		{
			name: "invalidunicode4",
			args: args{"ո"},
			want: false,
		},
		{
			name: "invalidunicode5",
			args: args{"κ"},
			want: false,
		},
		{
			name: "invalidunicodesnowman",
			args: args{"☃︎"},
			want: false,
		},
		{
			name: "invalidslash",
			args: args{"/"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidVMName(tt.args.name); got != tt.want {
				t.Errorf("ValidVMName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsInt(t *testing.T) {
	type args struct {
		elems []int
		v     int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "simple1",
			args: args{elems: []int{1, 2, 3}, v: 1},
			want: true,
		},
		{
			name: "simple2",
			args: args{elems: []int{1, 2, 34, 123, 71293812, 321}, v: 34},
			want: true,
		},
		{
			name: "simple3",
			args: args{elems: []int{110, 1, 2, 3, 34, 7281}, v: 7281},
			want: true,
		},
		{
			name: "simple4",
			args: args{elems: []int{110, 1, 2, 3, 34, 7281}, v: 7282},
			want: false,
		},
		{
			name: "simple5",
			args: args{elems: []int{110, 1, -2, 3, 34, 7281}, v: -2},
			want: true,
		},
		{
			name: "simple6",
			args: args{elems: []int{110, 1, -2, 3, 34, 7281}, v: -4},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsInt(tt.args.elems, tt.args.v); got != tt.want {
				t.Errorf("ContainsInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsStr(t *testing.T) {
	type args struct {
		elems []string
		v     string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "simple1",
			args: args{elems: []string{"a", "b", "c"}, v: "a"},
			want: true,
		},
		{
			name: "simple2",
			args: args{elems: []string{"abc"}, v: "a"},
			want: false,
		},
		{
			name: "simple3",
			args: args{elems: []string{"abc", "def", "ghi"}, v: "def"},
			want: true,
		},
		{
			name: "simple4",
			args: args{elems: []string{"аbc", "def", "ghi"}, v: "abc"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsStr(tt.args.elems, tt.args.v); got != tt.want {
				t.Errorf("ContainsStr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidNicName(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "empty",
			args: args{""},
			want: false,
		},
		{
			name: "validupper",
			args: args{"A"},
			want: true,
		},
		{
			name: "validupper2",
			args: args{"THEQUICKBROWNFOXJUMPSOVERTHELAZYDOG"},
			want: true,
		},
		{
			name: "validlower",
			args: args{"a"},
			want: true,
		},
		{
			name: "validlower2",
			args: args{"thequickbrownfoxjumpsoverthelazydog"},
			want: true,
		},
		{
			name: "validlower3",
			args: args{"abc"},
			want: true,
		},
		{
			name: "validnumber",
			args: args{"1"},
			want: true,
		},
		{
			name: "validnumber2",
			args: args{"0123456789THEQUICKBROWNFOXJUMPSOVERTHELAZYDOGthequickbrownfoxjumpsoverthelazydog"},
			want: true,
		},
		{
			name: "validunder",
			args: args{"_"},
			want: true,
		},
		{
			name: "validunder2",
			args: args{"_0123456789"},
			want: true,
		},
		{
			name: "validunder3",
			args: args{"_0123456789thequickbrownfoxjumpsoverthelazydog"},
			want: true,
		},
		{
			name: "validnumber4",
			args: args{"_0123456789thequickbrownfoxjumpsoverthelazydogTHEQUICKBROWNFOXJUMPSOVERTHELAZYDOG"},
			want: true,
		},
		{
			name: "validdash1",
			args: args{"-_0123456789"},
			want: true,
		},
		{
			name: "validdash2",
			args: args{"-_0123456789thequickbrownfoxjumpsoverthelazydog"},
			want: true,
		},
		{
			name: "validdash3",
			args: args{"-_0123456789thequickbrownfoxjumpsoverthelazydogTHEQUICKBROWNFOXJUMPSOVERTHELAZYDOG"},
			want: true,
		},
		{
			name: "validdash4",
			args: args{"--a-__a-a-__90123"},
			want: true,
		},
		{
			name: "invalid1",
			args: args{"abc9123asdf-@"},
			want: false,
		},
		{
			name: "invalid2",
			args: args{"abc9123asdf-#"},
			want: false,
		},
		{
			name: "invalid3",
			args: args{"abc9123asdf-)"},
			want: false,
		},
		{
			name: "invalid4",
			args: args{"abc9123asdf-("},
			want: false,
		},
		{
			name: "invalid5",
			args: args{"abc9123asdf-&"},
			want: false,
		},
		{
			name: "invalid6",
			args: args{"abc9123asdf-$"},
			want: false,
		},
		{
			name: "invalid7",
			args: args{"abc9123asdf-$"},
			want: false,
		},
		{
			name: "invalid8",
			args: args{"abc9123 asdf-$"},
			want: false,
		},
		{
			name: "invalid9",
			args: args{"ab.c"},
			want: false,
		},
		{
			name: "invalid10",
			args: args{"a..b"},
			want: false,
		},
		{
			name: "invalidunicode1",
			args: args{"aа"},
			want: false,
		},
		{
			name: "invalidunicode2",
			args: args{"с"},
			want: false,
		},
		{
			name: "invalidunicode3",
			args: args{"ԁ"},
			want: false,
		},
		{
			name: "invalidunicode4",
			args: args{"ո"},
			want: false,
		},
		{
			name: "invalidunicode5",
			args: args{"κ"},
			want: false,
		},
		{
			name: "invalidunicodesnowman",
			args: args{"☃︎"},
			want: false,
		},
		{
			name: "invalidslash",
			args: args{"/"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidNicName(tt.args.name); got != tt.want {
				t.Errorf("ValidNicName() = %v, want %v", got, tt.want)
			}
		})
	}
}
