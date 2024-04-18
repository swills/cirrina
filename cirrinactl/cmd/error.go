package cmd

import "errors"

var errDiskEmptyName = errors.New("empty disk name")
var errDiskNotFound = errors.New("disk not found")
var errDiskInUse = errors.New("unable to upload disk used by running VM")

var errIsoEmptyName = errors.New("empty ISO name")
var errIsoNotFound = errors.New("ISO not found")

var errNicEmptyName = errors.New("empty NIC name")
var errNicNotFound = errors.New("NIC not found")

var errSwitchEmptyName = errors.New("empty switch name")
var errSwitchNotFound = errors.New("switch not found")

var errVMEmptyName = errors.New("empty VM name")
var errVMNotFound = errors.New("VM not found")
var errVMInUseStop = errors.New("VM must be stopped in order to be destroyed")
var errVMNotRunning = errors.New("VM not running")
var errVMNotStopped = errors.New("VM must be stopped in order to be started")
var errVMUnknownFormat = errors.New("unknown output format")

var errReqFailed = errors.New("failed")

var errHostNotAvailable = errors.New("host not available")
