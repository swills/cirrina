package rpc

import (
	"errors"
)

var errReqFailed = errors.New("failed")
var errReqEmpty = errors.New("request ID not specified")
var errInvalidServerResponse = errors.New("invalid server response")
var errNotFound = errors.New("not found")
var errInternalError = errors.New("internal error")

var errDiskEmptyName = errors.New("disk name not specified")
var errDiskEmptyID = errors.New("disk id not specified")
var errDiskDuplicate = errors.New("duplicate disk found")
var errDiskTypeUnknown = errors.New("invalid disk type specified")
var errDiskDevTypeUnknown = errors.New("invalid disk dev type specified")

var errIsoEmptyName = errors.New("iso name not specified")
var errIsoEmptyID = errors.New("iso id not specified")
var errIsoDuplicate = errors.New("duplicate iso found")

var errNicEmptyID = errors.New("nic id not specified")
var errNicEmptyName = errors.New("nic name not specified")
var errNicNotFound = errors.New("nic not found")
var errNicDuplicate = errors.New("duplicate nic found")
var errNicInvalidType = errors.New("invalid nic type must be either VIRTIONET or E1000")
var errNicInvalidDevType = errors.New("invalid nic dev type must be one of TAP, VMNET or NETGRAPH")

var errSwitchEmptyID = errors.New("switch id not specified")
var errSwitchEmptyName = errors.New("switch name not specified")
var errSwitchTypeEmpty = errors.New("switch type not specified")
var errSwitchTypeInvalid = errors.New("switch type must be one of: IF, bridge, NG, netgraph")
var errSwitchDuplicate = errors.New("duplicate switch found")

var errVMNotFound = errors.New("VM not found")
var errVMEmptyID = errors.New("VM ID not specified")
var errVMEmptyName = errors.New("VM name not specified")
