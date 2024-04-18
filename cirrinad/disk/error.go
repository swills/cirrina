package disk

import "errors"

var errDiskInvalidName = errors.New("invalid disk name")
var errDiskExists = errors.New("disk exists")
var errDiskIDEmptyOrInvalid = errors.New("disk id not specified or invalid")
var errDiskNotFound = errors.New("disk not found")
var errDiskInternalDB = errors.New("internal disk database error")
var errDiskInvalidType = errors.New("invalid disk type")
var errDiskInvalidDevType = errors.New("invalid disk dev type")
var errDiskInvalidSize = errors.New("invalid disk size")
var errDiskZPoolNotConfigured = errors.New("zfs pool not configured")
var errDiskShrinkage = errors.New("new disk smaller than current disk")
var errDiskDupe = errors.New("duplicate disk found")
