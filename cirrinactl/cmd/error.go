package cmd

import "errors"

var (
	errDiskEmptyName = errors.New("empty disk name")
	errDiskNotFound  = errors.New("disk not found")
	errDiskInUse     = errors.New("unable to upload disk used by running VM")
)

var (
	errIsoEmptyName = errors.New("empty ISO name")
	errIsoNotFound  = errors.New("ISO not found")
)

var (
	errNicEmptyName = errors.New("empty NIC name")
	errNicNotFound  = errors.New("NIC not found")
)

var (
	errSwitchEmptyName = errors.New("empty switch name")
	errSwitchNotFound  = errors.New("switch not found")
)

var (
	errVMEmptyName     = errors.New("empty VM name")
	errVMNotFound      = errors.New("VM not found")
	errVMInUseStop     = errors.New("VM must be stopped in order to be destroyed")
	errVMNotRunning    = errors.New("VM not running")
	errVMNotStopped    = errors.New("VM must be stopped in order to be started")
	errVMUnknownFormat = errors.New("unknown output format")
)

var errReqFailed = errors.New("failed")

var errHostNotAvailable = errors.New("host not available")
