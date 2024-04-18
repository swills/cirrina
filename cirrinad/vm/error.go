package vm

import "errors"

var errVMUnknownDiskType = errors.New("unknown disk type")
var errVMNotFound = errors.New("not found")
var errVMDupe = errors.New("VM already exists")

var errVMInvalidComDev = errors.New("invalid com dev")
var errVMComDevIsDir = errors.New("error checking com dev readable: comReadDev is directory")
var errVMComDevNonexistent = errors.New("comDev does not exists)")

var errVMTypeFailure = errors.New("type failure")
var errVMTypeConversionFailure = errors.New("failed converting comReadFileInfo to Stat_t")

var errVMInvalidName = errors.New("invalid name")
var errVMInternalDB = errors.New("internal VM database error")
var errVMNotStopped = errors.New("VM must be stopped first")
var errVMAlreadyStopped = errors.New("VM already stopped")
var errVMStopFail = errors.New("stop failed")

var errVMSwitchNICMismatch = errors.New("bridge/interface type mismatch")

var errVMIsoInvalid = errors.New("iso id not specified or invalid")
var errVMIsoNotFound = errors.New("iso not found")

var errVMComInvalid = errors.New("invalid com port number")
var errVMComDevNotSet = errors.New("com port enabled but comDev not set")

var errVMDiskNotFound = errors.New("disk not found")
var errVMDiskInvalid = errors.New("disk id not specified or invalid")
var errVMDiskDupe = errors.New("disk may only be added once")
var errVMDiskAttached = errors.New("disk already attached")

var errVMNICInvalid = errors.New("nic id not specified or invalid")
var errVMNICNotFound = errors.New("nic not found")
var errVMNicDupe = errors.New("nic may only be added once")
var errVMNicAttached = errors.New("nic already attached")
