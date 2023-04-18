package config

import (
	"flag"
	"github.com/jinzhu/configor"
	"log"
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
			}
		}
	}
	Log struct {
		Path string
	}

	Network struct {
		Grpc struct {
			Ip   string `default:"0.0.0.0"`
			Port uint   `default:"50051"`
		}
		Interface string
		Bridge    string
	}
	Rom struct {
		Path string
		Vars struct {
			Template string
		}
	}
	Vnc struct {
		Ip   string `default:"0.0.0.0"`
		Port uint   `default:"5900"`
	}
}{}

func init() {
	flag.Parse()
	err := configor.Load(&Config, *configFile)
	if err != nil {
		log.Fatalf("Config loading failed: %v", err)
		return
	}
}
