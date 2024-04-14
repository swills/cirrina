package rpc

import (
	"errors"
	"io"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"cirrina/cirrina"
)

func GetHostNics() (rv []*cirrina.NetIf, err error) {
	var res cirrina.VMInfo_GetNetInterfacesClient
	res, err = serverClient.GetNetInterfaces(defaultServerContext, &cirrina.NetInterfacesReq{})
	if err != nil {
		return []*cirrina.NetIf{}, errors.New(status.Convert(err).Message())
	}
	for {
		hostNic, err := res.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return []*cirrina.NetIf{}, errors.New(status.Convert(err).Message())
		}
		rv = append(rv, hostNic)
	}

	return rv, nil
}

func GetHostVersion() (version string, err error) {
	var res *wrapperspb.StringValue
	res, err = serverClient.GetVersion(defaultServerContext, &emptypb.Empty{})
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	version = res.Value

	return version, nil
}
