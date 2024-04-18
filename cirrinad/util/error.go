package util

import "errors"

var errSocketNotFound = errors.New("failed parsing output, socket not found")
var errFailedParsing = errors.New("failed parsing output, statistics not found")
var errInvalidMac = errors.New("invalid MAC address")
var errInvalidNumCPUs = errors.New("invalid max number of CPUs")
var errMissingTCPStat = errors.New("missing tcp-stat")
var errNoListenSocket = errors.New("not a listen socket")
var errNoTCPSocket = errors.New("not a tcp socket")
var errNoListenPort = errors.New("port is not a listen port")
var errInvalidPort = errors.New("tcp port failed to convert to int")
var errPortNotFound = errors.New("tcp port not found")
var errPortNotParsable = errors.New("tcp port not parsable")
var errSTDERRNotEmpty = errors.New("stderr is not empty")
var errInvalidPid = errors.New("invalid PID")
var errInvalidDiskSize = errors.New("invalid disk size")
