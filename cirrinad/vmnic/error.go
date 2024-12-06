package vmnic

import "errors"

var (
	errNicExists            = errors.New("nic exists or not valid")
	errNicInternalDB        = errors.New("internal nic database error")
	errInvalidMac           = errors.New("invalid MAC address")
	errInvalidNetDevType    = errors.New("bad net dev type")
	errInvalidNetType       = errors.New("bad net type")
	errInvalidMacBroadcast  = errors.New("may not use broadcast MAC address")
	errInvalidMacMulticast  = errors.New("may not use multicast MAC address")
	ErrInvalidNicName       = errors.New("invalid name")
	ErrNicNotFound          = errors.New("nic not found")
	ErrInvalidNic           = errors.New("invalid NIC")
	ErrNicInUse             = errors.New("nic in use")
	ErrNicUnknownNetDevType = errors.New("unknown net dev type")
)
