package util

import "errors"

var (
	errSocketNotFound  = errors.New("failed parsing output, socket not found")
	errFailedParsing   = errors.New("failed parsing output, statistics not found")
	errInvalidMac      = errors.New("invalid MAC address")
	errInvalidNumCPUs  = errors.New("invalid max number of CPUs")
	errMissingTCPStat  = errors.New("missing tcp-stat")
	errNoListenSocket  = errors.New("not a listen socket")
	errNoTCPSocket     = errors.New("not a tcp socket")
	errNoListenPort    = errors.New("port is not a listen port")
	errInvalidPort     = errors.New("tcp port failed to convert to int")
	errPortNotFound    = errors.New("tcp port not found")
	errPortNotParsable = errors.New("tcp port not parsable")
	errSTDERRNotEmpty  = errors.New("stderr is not empty")
	errInvalidPid      = errors.New("invalid PID")
	errInvalidDiskSize = errors.New("invalid disk size")
)
