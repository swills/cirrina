package handlers

import "errors"

var (
	ErrRemoveDisk = errors.New("error removing disk")
	ErrRemoveISO  = errors.New("error removing ISO")
)
