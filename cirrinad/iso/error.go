package iso

import "errors"

var (
	ErrIsoInvalidName      = errors.New("invalid iso name")
	errIsoExists           = errors.New("iso exists")
	errIsoInternalDB       = errors.New("internal iso database error")
	errIsoIDEmptyOrInvalid = errors.New("iso id not specified or invalid")
	errIsoNotFound         = errors.New("iso not found")
	errIsoInUse            = errors.New("ISO in use")
)
