package config

import (
	"flag"
	"github.com/jinzhu/configor"
	"golang.org/x/exp/slog"
)

var configFile = flag.String("config", "config.yml", "Config File")

var Config = struct {
	Sys struct {
		Sudo string
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
			Ip   string `default:"0.0.0.0"`
			Port uint   `default:"50051"`
		}
		// We use the "00:18:25" private OUI from
		// https://standards-oui.ieee.org/oui/oui.txt
		// as default, because why not?
		// but you can customize it
		// you probably want to stick to the non-multicast ones from that file
		// grep -i private oui.txt | grep -Ei base | grep -v '^.[13579BDF]' | grep -vi limited | grep -vi ltd
		Mac struct {
			Oui string `default:"00:18:25"`
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
}{}

func init() {
	flag.Parse()
	err := configor.Load(&Config, *configFile)
	if err != nil {
		slog.Error("config loading failed", "err", err)
		return
	}
}
