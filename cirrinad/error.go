package main

import (
	"errors"
)

var errConfigFileNotFound = errors.New("config file not found")

var (
	errInvalidID   = errors.New("id not specified or invalid")
	errInvalidName = errors.New("name not specified or invalid")
)

var (
	errNotFound         = errors.New("not found")
	errInvalidRequest   = errors.New("invalid request")
	errPendingReqExists = errors.New("pending request already exists")
)

var (
	errInvalidNicID          = errors.New("nic id not specified or invalid")
	errNicInUseByMultipleVMs = errors.New("nic in use by more than one VM")
)

var (
	errSwitchNotFound      = errors.New("switch not found")
	errSwitchInvalidUplink = errors.New("uplink not specified")
	errSwitchUplinkInUse   = errors.New("uplink already in use")
	errSwitchInternalDB    = errors.New("internal switch database error")
)

var (
	errInvalidComDev = errors.New("invalid com dev")
	errComInvalid    = errors.New("invalid com port number")
	errComDevNotSet  = errors.New("com is not set")
)

var (
	errIsoUploadNil      = errors.New("nil isoUploadReq or iso id")
	errIsoUploadSize     = errors.New("iso upload size incorrect")
	errIsoUploadChecksum = errors.New("iso upload checksum incorrect")
	errIsoInUse          = errors.New("ISO in use")
	errISOInternalDB     = errors.New("internal ISO database error")
	errIsoNotFound       = errors.New("ISO not found")
)

var (
	errInvalidDebugPort      = errors.New("invalid debug port")
	errInvalidKeyboardLayout = errors.New("invalid keyboard layout")
	errInvalidSoundDev       = errors.New("invalid sound dev")
	errInvalidVncPort        = errors.New("invalid vnc port")
	errInvalidScreenWidth    = errors.New("invalid screen width")
	errInvalidScreenHeight   = errors.New("invalid screen height")
)

var (
	errInvalidVMState           = errors.New("unknown VM state")
	errInvalidVMStateStop       = errors.New("vm not running")
	errInvalidVMStateDelete     = errors.New("vm not stopped")
	errInvalidVMStateStart      = errors.New("vm not stopped")
	errInvalidVMStateDiskUpload = errors.New("can not upload disk to VM that is not stopped")
)

var (
	errDiskInvalidType        = errors.New("invalid disk type")
	errDiskInvalidDevType     = errors.New("invalid disk dev type")
	errDiskZPoolNotConfigured = errors.New("zfs pool not configured")
	errDiskUsedByTwo          = errors.New("disk used by two VMs")
	errDiskInUse              = errors.New("disk in use")
	errDiskChecksumFailure    = errors.New("disk upload checksum incorrect")
	errDiskSizeFailure        = errors.New("disk upload size incorrect")
	errDiskUpdateGeneric      = errors.New("error updating disk")
	errDiskDeleteGeneric      = errors.New("error deleting disk")
)

var (
	errUnableToMakeTmpFile = errors.New("could not find a tmp file")
	errSTDERRMismatch      = errors.New("stderr prefix mismatch running command")
	errSTDOUTMismatch      = errors.New("stdout prefix mismatch running command")
	errExitCodeMismatch    = errors.New("exitCode mismatch running command")
)

var (
	errVMDupe    = errors.New("VM already exists")
	errReqExists = errors.New("pending request for already exists")
)
