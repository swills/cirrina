package vmswitch

import "errors"

var (
	errSwitchInternalChecking      = errors.New("error checking if switch uplink in use by another bridge")
	ErrSwitchInvalidName           = errors.New("invalid name")
	errSwitchInvalidUplink         = errors.New("invalid switch uplink name")
	errSwitchNotFound              = errors.New("switch not found")
	errSwitchInvalidID             = errors.New("switch id invalid")
	ErrSwitchExists                = errors.New("switch exists")
	ErrSwitchInUse                 = errors.New("switch in use")
	ErrSwitchInvalidType           = errors.New("unknown switch type")
	errSwitchUplinkInUse           = errors.New("uplink already used")
	errSwitchUplinkWrongType       = errors.New("uplink switch has wrong type")
	errSwitchInternalDB            = errors.New("internal switch database error")
	errSwitchInvalidBridgeNameIF   = errors.New("invalid bridge name, bridge name must start with \"bridge\"")
	errSwitchFailDummy             = errors.New("failed to create ng bridge: could not get dummy bridge name")
	errSwitchUnknownNicDevType     = errors.New("unknown nic dev type")
	ErrSwitchDoesNotExist          = errors.New("switch does not exist")
	ErrSwitchInterfaceDoesNotExist = errors.New("interface does not exist")
)
