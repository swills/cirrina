package vmswitch

import "errors"

var errSwitchInternalChecking = errors.New("error checking if switch uplink in use by another bridge")
var errSwitchInvalidName = errors.New("invalid name")
var errSwitchInvalidUplink = errors.New("invalid switch uplink name")
var errSwitchInvalidNetDevEmpty = errors.New("netDev can't be empty")
var errSwitchNotFound = errors.New("not found")
var errSwitchExists = errors.New("switch exists")
var errSwitchInvalidID = errors.New("switch id invalid")
var errSwitchInUse = errors.New("switch in use")
var errSwitchInvalidType = errors.New("unknown switch type")
var errSwitchUplinkInUse = errors.New("uplink already used")
var errSwitchUplinkWrongType = errors.New("uplink switch has wrong type")
var errSwitchInternalDB = errors.New("internal nic database error")
var errSwitchInvalidBridgeNameIF = errors.New("invalid bridge name, bridge name must start with \"bridge\"")
var errSwitchInvalidBridgeDupe = errors.New("duplicate bridge")
var errSwitchInvalidBridgeNameNG = errors.New("invalid bridge name, bridge name must start with \"bnet\"")
var errSwitchFailDummy = errors.New("failed to create ng bridge: could not get dummy bridge name")
