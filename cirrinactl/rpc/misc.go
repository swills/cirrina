package rpc

import (
	"cirrina/cirrina"
	"context"
	"errors"
)

func ReqStat(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (r *cirrina.ReqStatus, err error) {
	if *idPtr == "" {
		return &cirrina.ReqStatus{}, errors.New("id not specified")
	}
	res, err := c.RequestStatus(ctx, &cirrina.RequestID{Value: *idPtr})
	if err != nil {
		return &cirrina.ReqStatus{}, err
	}
	return res, nil
}
