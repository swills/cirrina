package rpc

import (
	"errors"
)

var (
	errReqFailed             = errors.New("failed")
	errReqEmpty              = errors.New("request ID not specified")
	errInvalidServerResponse = errors.New("invalid server response")
	errNotFound              = errors.New("not found")
	errInternalError         = errors.New("internal error")
)

var (
	errDiskEmptyName      = errors.New("disk name not specified")
	errDiskEmptyID        = errors.New("disk id not specified")
	errDiskDuplicate      = errors.New("duplicate disk found")
	errDiskTypeUnknown    = errors.New("invalid disk type specified")
	errDiskDevTypeUnknown = errors.New("invalid disk dev type specified")
)

var (
	errIsoEmptyName = errors.New("iso name not specified")
	errIsoEmptyID   = errors.New("iso id not specified")
	errIsoDuplicate = errors.New("duplicate iso found")
)

var (
	errNicEmptyID        = errors.New("nic id not specified")
	errNicEmptyName      = errors.New("nic name not specified")
	errNicNotFound       = errors.New("nic not found")
	errNicDuplicate      = errors.New("duplicate nic found")
	errNicInvalidType    = errors.New("invalid nic type must be either VIRTIONET or E1000")
	errNicInvalidDevType = errors.New("invalid nic dev type must be one of TAP, VMNET or NETGRAPH")
)

var (
	errSwitchEmptyID     = errors.New("switch id not specified")
	errSwitchEmptyName   = errors.New("switch name not specified")
	errSwitchTypeEmpty   = errors.New("switch type not specified")
	errSwitchTypeInvalid = errors.New("switch type must be one of: IF, bridge, NG, netgraph")
	errSwitchDuplicate   = errors.New("duplicate switch found")
)

var (
	errVMNotFound  = errors.New("VM not found")
	errVMEmptyID   = errors.New("VM ID not specified")
	errVMEmptyName = errors.New("VM name not specified")
)
