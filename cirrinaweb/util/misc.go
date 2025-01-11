package util

import (
	"strconv"
)

var (
	listenHost            = "localhost"
	listenPort     uint64 = 8888
	websockifyPort uint64 = 7900
)

func SetListenHost(lh string) {
	if lh != "" {
		listenHost = lh
	}
}

func SetListenPort(lp string) {
	var err error

	if lp != "" {
		listenPort, err = strconv.ParseUint(lp, 10, 16)
		if err != nil || listenPort > 65536 {
			listenPort = 8888
		}
	} else {
		listenPort = 8888
	}
}

func SetWebsockifyPort(websockifyPortString string) {
	var err error

	if websockifyPortString != "" {
		var cirrinaWebsockifyPortTemp uint64

		cirrinaWebsockifyPortTemp, err = strconv.ParseUint(websockifyPortString, 10, 16)
		if err == nil {
			websockifyPort = cirrinaWebsockifyPortTemp
		}
	}
}

func GetListenHost() string {
	return listenHost
}

func GetListenPort() uint64 {
	return listenPort
}

func GetWebsockifyPort() uint64 {
	return websockifyPort
}
