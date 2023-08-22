package main

import (
	"cirrina/cirrina"
	"context"
	"io"
)

func getHostNics(c cirrina.VMInfoClient, ctx context.Context) (rv []*cirrina.NetIf, err error) {
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
