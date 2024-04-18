package vmnic

import "errors"

var errNicExists = errors.New("VMNic exists or not valid")
var errNicInternalDB = errors.New("internal nic database error")
var errInvalidMac = errors.New("invalid MAC address")
var errInvalidNetDevType = errors.New("bad net dev type")
var errInvalidNetType = errors.New("bad net type")
var errInvalidNetworkRateLimit = errors.New("bad network rate limit")
var errInvalidNicName = errors.New("invalid name")
var errInvalidMacBroadcast = errors.New("may not use broadcast MAC address")
var errInvalidMacMulticast = errors.New("may not use multicast MAC address")
