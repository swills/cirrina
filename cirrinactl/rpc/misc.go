package rpc

import (
	"cirrina/cirrina"
	"context"
	"errors"
	"io"
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

func GetHostNics(c cirrina.VMInfoClient, ctx context.Context) (rv []*cirrina.NetIf, err error) {
	res, err := c.GetNetInterfaces(ctx, &cirrina.NetInterfacesReq{})
	if err != nil {
		return []*cirrina.NetIf{}, err
	}
	for {
		hostNic, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []*cirrina.NetIf{}, err
		}
		rv = append(rv, hostNic)
	}
	return rv, nil
}
