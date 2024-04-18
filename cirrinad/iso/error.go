package iso

import "errors"

var errIsoInvalidName = errors.New("invalid iso name")
var errIsoExists = errors.New("iso exists")
var errIsoInternalDB = errors.New("internal iso database error")
var errIsoIDEmptyOrInvalid = errors.New("iso id not specified or invalid")
var errIsoNotFound = errors.New("iso not found")
