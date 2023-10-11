package config

import (
	"flag"
	"fmt"
	"github.com/jinzhu/configor"
	"golang.org/x/exp/slog"
	"net"
	"os"
	"strings"
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
		Ip   string `default:"0.0.0.0"`
		Port uint   `default:"5900"`
	}
	Debug struct {
		Ip   string `default:"0.0.0.0"`
		Port uint   `default:"2828"`
	}
}{}

func validateConfig() {
	macTest := Config.Network.Mac.Oui
	hwAddr, err := net.ParseMAC(macTest + ":ff:ff:ff")
	if err != nil {
		slog.Error("validateConfig", "hwAddr", hwAddr)
		fmt.Printf("invalid MAC OUI %s: %s\n", macTest, err)
		os.Exit(1)
	}
	hwAddrSlc := strings.ToLower(hwAddr.String())
	if hwAddrSlc == "ff:ff:ff:ff:ff:ff" {
		fmt.Printf("invalid MAC OUI %s: may not use potentially broadcast OUI\n", macTest)
		os.Exit(1)
	}
	// https://cgit.freebsd.org/src/tree/usr.sbin/bhyve/net_utils.c?id=1d386b48a555f61cb7325543adbbb5c3f3407a66#n56
	// could maybe convert to hex and check but meh
	if string(hwAddrSlc[1]) == "1" || string(hwAddrSlc[1]) == "3" || string(hwAddrSlc[1]) == "5" || string(hwAddrSlc[1]) == "7" || string(hwAddrSlc[1]) == "9" || string(hwAddrSlc[1]) == "b" || string(hwAddrSlc[1]) == "d" || string(hwAddrSlc[1]) == "f" {
		fmt.Printf("invalid MAC OUI %s: may not use multicast OUI\n", macTest)
		os.Exit(1)
	}
}

func init() {
	flag.Parse()
	err := configor.Load(&Config, *configFile)
	if err != nil {
		slog.Error("config loading failed", "err", err)
		return
	}
	validateConfig()
}
