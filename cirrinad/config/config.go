package config

type ConfigType struct {
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
			Size string
		}
	}
	Log struct {
		Path  string
		Level string
	}
	Network struct {
		Grpc struct {
			// TODO separate settings for IPv4 and IPv6 IP
			Ip   string
			Port uint
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
		Ip   string `default:"0.0.0.0"`
		Port uint   `default:"5900"`
	}
	Debug struct {
		// TODO separate settings for IPv4 and IPv6 IP
		Ip   string `default:"0.0.0.0"`
		Port uint   `default:"2828"`
	}
}

var Config ConfigType
