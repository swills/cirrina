package main

import (
	"errors"
)

var errConfigFileNotFound = errors.New("config file not found")

var errInvalidID = errors.New("id not specified or invalid")
var errInvalidName = errors.New("name not specified or invalid")

var errReqInvalid = errors.New("nil diskUploadReq or disk id")

var errNotFound = errors.New("not found")
var errInvalidRequest = errors.New("invalid request")
var errPendingReqExists = errors.New("pending request already exists")

var errInvalidNicID = errors.New("nic id not specified or invalid")
var errNicInUseByMultipleVMs = errors.New("nic in use by more than one VM")
var errNicInUse = errors.New("nic in use")
var errNicUnknown = errors.New("unknown error creating VMNic")

var errSwitchNotFound = errors.New("switch not found")
var errSwitchInUse = errors.New("switch in use")
var errSwitchInvalidType = errors.New("invalid switch type")
var errSwitchInvalidUplink = errors.New("uplink not specified")
var errSwitchUplinkInUse = errors.New("uplink already in use")
var errSwitchInternalDB = errors.New("internal switch database error")
var errSwitchInvalidName = errors.New("invalid bridge name")

var errInvalidComDev = errors.New("invalid com dev")
var errComInvalid = errors.New("invalid com port number")
var errComDevNotSet = errors.New("com is not set")

var errIsoUploadNil = errors.New("nil isoUploadReq or iso id")
var errIsoUploadSize = errors.New("iso upload size incorrect")
var errIsoUploadChecksum = errors.New("iso upload checksum incorrect")
var errIsoInUse = errors.New("ISO in use")

var errInvalidDebugPort = errors.New("invalid debug port")
var errInvalidKeyboardLayout = errors.New("invalid keyboard layout")
var errInvalidSoundDev = errors.New("invalid sound dev")
var errInvalidVncPort = errors.New("invalid vnc port")

var errInvalidVMState = errors.New("unknown VM state")
var errInvalidVMStateStop = errors.New("vm not running")
var errInvalidVMStateDelete = errors.New("vm not stopped")
var errInvalidVMStateStart = errors.New("vm not stopped")
var errInvalidVMStateDiskUpload = errors.New("can not upload disk to VM that is not stopped")

var errDiskInvalidType = errors.New("invalid disk type")
var errDiskInvalidDevType = errors.New("invalid disk dev type")
var errDiskZPoolNotConfigured = errors.New("zfs pool not configured")
var errDiskInUse = errors.New("disk already attached to another VM")
var errDiskChecksumFailure = errors.New("disk upload checksum incorrect")
var errDiskSizeFailure = errors.New("disk upload size incorrect")
var errDiskCreateGeneric = errors.New("error creating disk")
var errDiskUpdateGeneric = errors.New("error updating disk")
var errDiskDeleteGeneric = errors.New("error deleting disk")

var errUnableToMakeTmpFile = errors.New("could not find a tmp file")
var errSTDERRMismatch = errors.New("stderr prefix mismatch running command")
var errSTDOUTMismatch = errors.New("stdout prefix mismatch running command")
var errExitCodeMismatch = errors.New("exitCode mismatch running command")

var errISOInternalDB = errors.New("internal ISO database error")

var errVMDupe = errors.New("VM already exists")
var errReqExists = errors.New("pending request for already exists")
