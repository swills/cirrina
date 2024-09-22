package config

type Info struct {
	Sys struct {
		Sudo        string
		PidFilePath string
	}
	DB struct {
		Path string
	}
	Disk struct {
		VM struct {
			Path struct {
				Image string
				State string
				Iso   string
				Zpool string
			}
		}
		Default struct {
			Size string `default:"1g"`
		}
	}
	Log struct {
		Path  string
		Level string
	}
	Network struct {
		Grpc struct {
			// TODO separate settings for IPv4 and IPv6 IP
			IP      string
			Port    uint
			Timeout uint64 `default:"60"` // in seconds
		}
		Mac struct {
			Oui string
		}
	}
	Rom struct {
		Path string
		Vars struct {
			Template string
		}
	}
	Vnc struct {
		// TODO separate settings for IPv4 and IPv6 IP
		IP   string `default:"0.0.0.0"`
		Port uint   `default:"5900"`
	}
	Debug struct {
		// TODO separate settings for IPv4 and IPv6 IP
		IP   string `default:"0.0.0.0"`
		Port uint   `default:"2828"`
	}
	Metrics struct {
		Enabled bool   `default:"false"`
		Host    string `default:""`
		Port    uint   `default:"2223"`
	}
}

var Config Info
