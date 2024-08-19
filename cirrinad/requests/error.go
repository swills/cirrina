package requests

import "errors"

var errRequestCreateFailure = errors.New("failed to create request")
var errRequestNotFound = errors.New("request not found")
var ErrInvalidRequest = errors.New("invalid request")
