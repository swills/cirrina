package vmnic

import "errors"

var (
	errNicExists               = errors.New("VMNic exists or not valid")
	errNicInternalDB           = errors.New("internal nic database error")
	errInvalidMac              = errors.New("invalid MAC address")
	errInvalidNetDevType       = errors.New("bad net dev type")
	errInvalidNetType          = errors.New("bad net type")
	errInvalidNetworkRateLimit = errors.New("bad network rate limit")
	errInvalidNicName          = errors.New("invalid name")
	errInvalidMacBroadcast     = errors.New("may not use broadcast MAC address")
	errInvalidMacMulticast     = errors.New("may not use multicast MAC address")
)
