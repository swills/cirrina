package rpc

import (
	"errors"
	"fmt"
	"io"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"cirrina/cirrina"
)

func GetHostNics() ([]*cirrina.NetIf, error) {
	var err error
	var hostNics []*cirrina.NetIf
	var res cirrina.VMInfo_GetNetInterfacesClient
	res, err = serverClient.GetNetInterfaces(defaultServerContext, &cirrina.NetInterfacesReq{})
	if err != nil {
		return []*cirrina.NetIf{}, fmt.Errorf("unable to get host nics: %w", err)
	}
	for {
		hostNic, err := res.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return []*cirrina.NetIf{}, fmt.Errorf("unable to get host nics: %w", err)
		}
		hostNics = append(hostNics, hostNic)
	}

	return hostNics, nil
}

func GetHostVersion() (string, error) {
	var version string
	var err error
	var res *wrapperspb.StringValue
	res, err = serverClient.GetVersion(defaultServerContext, &emptypb.Empty{})
	if err != nil {
		return "", fmt.Errorf("unable to get host version: %w", err)
	}
	version = res.Value

	return version, nil
}
