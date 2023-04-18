package config

import (
	"github.com/jinzhu/configor"
	"log"
)

var Config = struct {
	Disk struct {
		VM struct {
			Path struct {
				Image string
				State string
				Iso   string
			}
		}
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
	err := configor.Load(&Config, "config.yml")
	if err != nil {
		log.Fatalf("Config loading failed: %v", err)
		return
	}
}
