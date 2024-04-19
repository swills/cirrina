package disk

import "errors"

var (
	errDiskInvalidName        = errors.New("invalid disk name")
	errDiskExists             = errors.New("disk exists")
	errDiskIDEmptyOrInvalid   = errors.New("disk id not specified or invalid")
	errDiskNotFound           = errors.New("disk not found")
	errDiskInternalDB         = errors.New("internal disk database error")
	errDiskInvalidType        = errors.New("invalid disk type")
	errDiskInvalidDevType     = errors.New("invalid disk dev type")
	errDiskZPoolNotConfigured = errors.New("zfs pool not configured")
	errDiskShrinkage          = errors.New("new disk smaller than current disk")
	errDiskDupe               = errors.New("duplicate disk found")
)
