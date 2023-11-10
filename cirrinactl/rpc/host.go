package rpc

import (
	"cirrina/cirrina"
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
)

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

func GetHostVersion(c cirrina.VMInfoClient, ctx context.Context) (version string, err error) {
	res, err := c.GetVersion(ctx, &emptypb.Empty{})
	if err != nil {
		return "", err
	}
	version = res.Value
	return version, nil
}
