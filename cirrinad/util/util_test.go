package util

import (
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	exec "golang.org/x/sys/execabs"
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
		{
			name: "nil1",
			args: args{elems: nil, v: 6900},
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

var netstatOutOK1 = `
{
  "statistics": {
    "socket": [
      {
        "protocol": "tcp4",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "local": {
          "address": "*",
          "port": "22"
        },
        "remote": {
          "address": "*",
          "port": "*"
        },
        "tcp-state": "LISTEN     "
      },
      {
        "protocol": "tcp6",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "local": {
          "address": "*",
          "port": "22"
        },
        "remote": {
          "address": "*",
          "port": "*"
        },
        "tcp-state": "LISTEN     "
      },
      {
        "protocol": "udp4",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "local": {
          "address": "*",
          "port": "514"
        },
        "remote": {
          "address": "*",
          "port": "*"
        }
      },
      {
        "protocol": "udp6",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "local": {
          "address": "*",
          "port": "514"
        },
        "remote": {
          "address": "*",
          "port": "*"
        }
      },
      {
        "address": "fffff80002cc9700",
        "type": "stream",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "vnode": "0",
        "connection": "fffff80002cc9800",
        "first-reference": "0",
        "next-reference": "0"
      },
      {
        "address": "fffff80002cc9800",
        "type": "stream",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "vnode": "0",
        "connection": "fffff80002cc9700",
        "first-reference": "0",
        "next-reference": "0"
      },
      {
        "address": "fffff80002da4700",
        "type": "stream",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "vnode": "fffff80002d33540",
        "connection": "0",
        "first-reference": "0",
        "next-reference": "0",
        "path": "/var/run/devd.pipe"
      },
      {
        "address": "fffff80002da4600",
        "type": "dgram",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "vnode": "0",
        "connection": "fffff80002da4c00",
        "first-reference": "0",
        "next-reference": "fffff80002da4500"
      },
      {
        "address": "fffff80002da4500",
        "type": "dgram",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "vnode": "0",
        "connection": "fffff80002da4c00",
        "first-reference": "0",
        "next-reference": "0"
      },
      {
        "address": "fffff80002da4a00",
        "type": "dgram",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "vnode": "fffff8006d6bca80",
        "connection": "0",
        "first-reference": "0",
        "next-reference": "0",
        "path": "/var/run/logpriv"
      },
      {
        "address": "fffff80002da4c00",
        "type": "dgram",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "vnode": "fffff8006d6bcc40",
        "connection": "0",
        "first-reference": "fffff80002da4600",
        "next-reference": "0",
        "path": "/var/run/log"
      },
      {
        "address": "fffff80002da4900",
        "type": "seqpac",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "vnode": "fffff80002d33380",
        "connection": "0",
        "first-reference": "0",
        "next-reference": "0",
        "path": "/var/run/devd.seqpacket.pipe"
      }
    ]
  }
}
`
var netstatOutOK2 = `
{
  "statistics": {
    "socket": [
      {
        "protocol": "tcp4",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "local": {
          "address": "*",
          "port": "22"
        },
        "remote": {
          "address": "*",
          "port": "*"
        },
        "tcp-state": "LISTEN     "
      }
    ]
  }
}
`
var netstatOutBad1 = `
{  
  "statistics": {
    "socket": [ 
      { 
        "protocol": "tcp4",   
        "receive-bytes-waiting": 0,    
        "send-bytes-waiting": 0,   
        "local": {   
          "address": "*",   
          "port": "22"   
        },      
        "remote": { 
          "address": "*",    
          "port": "*"    
        },     
        "tcp-state": "LISTEN     "     
      }   ,
      1
    ] 
  }
}`
var netstatOutBad2 = `
{
  "statistics": {
    "socket": [
      {
        "protocol": "tcp6",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "local": {
          "address": "::1",
          "port": "22"
        },
        "remote": {
          "address": "::1",
          "port": "61720"
        },
        "tcp-state": "ESTABLISHED"
      },
      {
        "protocol": "tcp6",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "local": {
          "address": "::1",
          "port": "61720"
        },
        "remote": {
          "address": "::1",
          "port": "22"
        },
        "tcp-state": "ESTABLISHED"
      },
      {
        "protocol": "tcp4",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "local": {
          "address": "*",
          "port": "22"
        },
        "remote": {
          "address": "*",
          "port": "*"
        },
        "tcp-state": "LISTEN     "
      },
      {
        "protocol": "tcp6",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "local": {
          "address": "*",
          "port": "22"
        },
        "remote": {
          "address": "*",
          "port": "*"
        },
        "tcp-state": "LISTEN     "
      },
      {
        "protocol": "udp4",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "local": {
          "address": "*",
          "port": "514"
        },
        "remote": {
          "address": "*",
          "port": "*"
        }
      },
      {
        "protocol": "udp6",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "local": {
          "address": "*",
          "port": "514"
        },
        "remote": {
          "address": "*",
          "port": "*"
        }
      },
      {
        "address": "fffff80002cc9c00",
        "type": "stream",
        "receive-bytes-waiting": 0,
        "send-bytes-waiting": 0,
        "vnode": "0",
        "connection": "fffff80002cc9a00",
        "first-reference": "0",
        "next-reference": "0"
      }
    ]
  }
}
`
var netstatOutBad3 = `
{  
  "statistics": {
    "socket": [ 
      { 
        "protocol": "tcp4",   
        "receive-bytes-waiting": 0,    
        "send-bytes-waiting": 0,   
        "local": {   
          "address": "*",   
          "port": "twentytwo"   
        },      
        "remote": { 
          "address": "*",    
          "port": "*"    
        },     
        "tcp-state": "LISTEN     "     
      }   ,
      1
    ] 
  }
}`
var netstatOutBad4 = `
{  
  "statistics": {
    "socket": [ 
      { 
        "protocol": "tcp4",   
        "receive-bytes-waiting": 0,    
        "send-bytes-waiting": 0,   
        "local": {   
          "address": "*",   
          "port": 22   
        },      
        "remote": { 
          "address": "*",    
          "port": "*"    
        },     
        "tcp-state": "LISTEN     "     
      }   ,
      1
    ] 
  }
}`
var netstatOutBad5 = `
{  
  "statistics": {
    "socket": [ 
      { 
        "protocol": "tcp4",   
        "receive-bytes-waiting": 0,    
        "send-bytes-waiting": 0,   
        "local": {   
          "address": "*"   
        },      
        "remote": { 
          "address": "*",    
          "port": "*"    
        },     
        "tcp-state": "LISTEN     "     
      }   ,
      1
    ] 
  }
}`
var netstatOutBad6 = `
{  
  "statistics": {
    "socket": [ 
      { 
        "protocol": "tcp4",   
        "receive-bytes-waiting": 0,    
        "send-bytes-waiting": 0,   
        "remote": { 
          "address": "*",    
          "port": "*"    
        },     
        "tcp-state": "LISTEN     "     
      }   ,
      1
    ] 
  }
}`
var netstatOutBad7 = `
{  
  "statistics": {
    "socket": [ 
      { 
        "protocol": "tcp4",   
        "receive-bytes-waiting": 0,    
        "send-bytes-waiting": 0,   
        "remote": { 
          "address": "*",    
          "port": "*"    
        }     
      }   ,
      1
    ] 
  }
}`

func Test_parseNetstatJSONOutput(t *testing.T) {
	type args struct {
		netstatOutput []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []int
		wantErr bool
	}{
		{
			name:    "generic1",
			args:    args{netstatOutput: []byte(netstatOutOK1)},
			want:    []int{22},
			wantErr: false,
		},
		{
			name:    "generic2",
			args:    args{netstatOutput: []byte(netstatOutOK2)},
			want:    []int{22},
			wantErr: false,
		},
		{
			name:    "invalid1",
			args:    args{netstatOutput: []byte("")},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid2",
			args:    args{netstatOutput: []byte("{\"something\": 1}")},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid3",
			args:    args{netstatOutput: []byte("{\"statistics\": {\"socket\": 2}}")},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid4",
			args:    args{netstatOutput: []byte(netstatOutBad1)},
			want:    []int{22},
			wantErr: false,
		},
		{
			name:    "invalid5",
			args:    args{netstatOutput: []byte(netstatOutBad2)},
			want:    []int{22},
			wantErr: false,
		},
		{
			name:    "invalid6",
			args:    args{netstatOutput: []byte(netstatOutBad3)},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "invalid7",
			args:    args{netstatOutput: []byte(netstatOutBad4)},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "invalid8",
			args:    args{netstatOutput: []byte(netstatOutBad5)},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "invalid9",
			args:    args{netstatOutput: []byte(netstatOutBad6)},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "invalid10",
			args:    args{netstatOutput: []byte(netstatOutBad7)},
			want:    nil,
			wantErr: false,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := parseNetstatJSONOutput(testCase.args.netstatOutput)
			if (err != nil) != testCase.wantErr {
				t.Errorf("parseNetstatJSONOutput() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}
			if !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("parseNetstatJSONOutput() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestPidExists(t *testing.T) {
	type args struct {
		pid int
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "findinit",
			args:    args{pid: 1},
			want:    true,
			wantErr: false,
		},
		{
			name:    "invalidPid1",
			args:    args{pid: -11},
			want:    false,
			wantErr: true,
		},
		{
			name:    "invalidPid2",
			args:    args{pid: 99999999},
			want:    false,
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := PidExists(testCase.args.pid)
			if (err != nil) != testCase.wantErr {
				t.Errorf("PidExists() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}
			if got != testCase.want {
				t.Errorf("PidExists() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestPidExistsSleeping(t *testing.T) {
	var err error
	sleepCmd := exec.Command("/bin/sleep", "42")
	err = sleepCmd.Start()
	if err != nil {
		t.Fail()
	}
	sleepPid := sleepCmd.Process.Pid

	type args struct {
		pid int
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "sleepTest",
			args:    args{sleepPid},
			want:    true,
			wantErr: false,
		},
	}
	var got bool

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err = PidExists(testCase.args.pid)
			if (err != nil) != testCase.wantErr {
				t.Errorf("PidExists() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}
			if got != testCase.want {
				t.Errorf("PidExists() got = %v, want %v", got, testCase.want)
			}
		})
	}
	err = sleepCmd.Process.Kill()
	if err != nil {
		t.Fail()
	}
	err = sleepCmd.Wait()
	if err != nil {
		t.Fail()
	}
}

func TestPidExistsSleepingExited(t *testing.T) {
	sleepCmd := exec.Command("/bin/sleep", "42")
	err := sleepCmd.Start()
	if err != nil {
		t.Fail()
	}
	sleepPid := sleepCmd.Process.Pid

	type args struct {
		pid int
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "sleepTest",
			args:    args{sleepPid},
			want:    false,
			wantErr: false,
		},
	}

	err = sleepCmd.Process.Kill()
	if err != nil {
		t.Fail()
	}
	err = sleepCmd.Wait()
	if err != nil {
		t.Fail()
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := PidExists(testCase.args.pid)
			if (err != nil) != testCase.wantErr {
				t.Errorf("PidExists() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}
			if got != testCase.want {
				t.Errorf("PidExists() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}

	return string(s)
}

func TestPathExists(t *testing.T) {
	testPathExistsTmpDir, err := os.MkdirTemp("/tmp/", "cirrinaTestPathExists")
	if err != nil {
		log.Fatal(err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fail()
		}
	}(testPathExistsTmpDir) // clean up

	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "findslash",
			args:    args{path: "/"},
			want:    true,
			wantErr: false,
		},
		{
			name:    "findtmp",
			args:    args{path: "/tmp"},
			want:    true,
			wantErr: false,
		},
		{
			name:    "findrandompath",
			args:    args{path: "/tmp/" + RandomString(10)},
			want:    false,
			wantErr: false,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := PathExists(testCase.args.path)
			if (err != nil) != testCase.wantErr {
				t.Errorf("PathExists() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}
			if got != testCase.want {
				t.Errorf("PathExists() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestOSReadDir(t *testing.T) {
	testOSReadDirPath1, err := os.MkdirTemp("/tmp/", "cirrinaTestOSReadDir1")
	if err != nil {
		log.Fatal(err)
	}
	defer func(path string) {
		err2 := os.RemoveAll(path)
		if err2 != nil {
			t.Fail()
		}
	}(testOSReadDirPath1) // clean up

	testOSReadDirPath2, err := os.MkdirTemp("/tmp/", "cirrinaTestOSReadDir2")
	if err != nil {
		t.Fail()
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fail()
		}
	}(testOSReadDirPath2) // clean up

	file := filepath.Join(testOSReadDirPath2, "tmpfile")
	if err := os.WriteFile(file, []byte("content"), 0666); err != nil {
		t.Fail()
	}

	type args struct {
		root string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name:    "readEmpty",
			args:    args{root: testOSReadDirPath1},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "readEmpty",
			args:    args{root: testOSReadDirPath2},
			want:    []string{"tmpfile"},
			wantErr: false,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := OSReadDir(testCase.args.root)
			if (err != nil) != testCase.wantErr {
				t.Errorf("OSReadDir() error = %v, wantErr %v", err, testCase.wantErr)

				return
			}
			if !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("OSReadDir() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestIsValidTCPPort(t *testing.T) {
	type args struct {
		tcpPort uint
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "valid1",
			args: args{tcpPort: 123},
			want: true,
		},
		{
			name: "invalid1",
			args: args{tcpPort: 65536},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidTCPPort(tt.args.tcpPort); got != tt.want {
				t.Errorf("IsValidTCPPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidIP(t *testing.T) {
	type args struct {
		ipAddress string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "valid1",
			args: args{ipAddress: "10.0.1.1"},
			want: true,
		},
		{
			name: "valid2",
			args: args{ipAddress: "192.168.0.1"},
			want: true,
		},
		{
			name: "invalid1",
			args: args{ipAddress: "912.861.1.0"},
			want: false,
		},
		{
			name: "invalid2",
			args: args{ipAddress: "asdf"},
			want: false,
		},
		{
			name: "valid3",
			args: args{ipAddress: "2001:db8::68"},
			want: true,
		},
		{
			name: "valid4",
			args: args{ipAddress: "::ffff:192.0.2.1"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidIP(tt.args.ipAddress); got != tt.want {
				t.Errorf("IsValidIP() = %v, want %v", got, tt.want)
			}
		})
	}
}
