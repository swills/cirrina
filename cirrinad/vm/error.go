package vm

import "errors"

var (
	errVMUnknownDiskType   = errors.New("unknown disk type")
	errVMUnknownNetType    = errors.New("unknown net type")
	errVMUnknownNetDevType = errors.New("unknown net dev type")
	errVMNotFound          = errors.New("not found")
	errVMDupe              = errors.New("VM already exists")
	errFailedParsing       = errors.New("failed parsing output")
)

var (
	errVMInvalidComDev     = errors.New("invalid com dev")
	errVMComDevIsDir       = errors.New("error checking com dev readable: comReadDev is directory")
	errVMComDevNonexistent = errors.New("comDev does not exists)")
)

var (
	errVMTypeFailure           = errors.New("type failure")
	errVMTypeConversionFailure = errors.New("failed converting comReadFileInfo to Stat_t")
)

var (
	errVMInvalidName      = errors.New("invalid name")
	errVMInternalDB       = errors.New("internal VM database error")
	errVMNotStopped       = errors.New("VM must be stopped first")
	errVMStopFail         = errors.New("stop failed")
	errVMIDEmptyOrInvalid = errors.New("VM ID not specified or invalid")
)

var (
	errVMComInvalid   = errors.New("invalid com port number")
	errVMComDevNotSet = errors.New("com port enabled but comDev not set")
)

var (
	errVMDiskNotFound = errors.New("disk not found")
	errVMDiskInvalid  = errors.New("disk id not specified or invalid")
	errVMDiskDupe     = errors.New("disk may only be added once")
	errVMDiskAttached = errors.New("disk already attached")
)

var (
	errVMNicDupe     = errors.New("nic may only be added once")
	errVMNicAttached = errors.New("nic already attached")
)
