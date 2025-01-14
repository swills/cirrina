package util

import (
	"strconv"

	"github.com/spf13/cast"
)

var (
	listenHost            = "localhost"
	listenPort     uint16 = 8888
	websockifyPort uint16 = 7900
)

func SetListenHost(lh string) {
	if lh != "" {
		listenHost = lh
	}
}

func SetListenPort(listenPortString string) {
	var err error

	var listenPort64 uint64

	if listenPortString != "" {
		listenPort64, err = strconv.ParseUint(listenPortString, 10, 16)
		if err == nil {
			listenPort = cast.ToUint16(listenPort64)
		}
	}
}

func SetWebsockifyPort(websockifyPortString string) {
	var err error

	if websockifyPortString != "" {
		var cirrinaWebsockifyPortTemp uint64

		cirrinaWebsockifyPortTemp, err = strconv.ParseUint(websockifyPortString, 10, 16)
		if err == nil {
			websockifyPort = cast.ToUint16(cirrinaWebsockifyPortTemp)
		}
	}
}

func GetListenHost() string {
	return listenHost
}

func GetListenPort() uint16 {
	return listenPort
}

func GetWebsockifyPort() uint16 {
	return websockifyPort
}
