package rpc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"cirrina/cirrina"
)

func GetHostNics(ctx context.Context) ([]string, error) {
	var err error

	var hostNics []string

	var res cirrina.VMInfo_GetNetInterfacesClient

	res, err = serverClient.GetNetInterfaces(ctx, &cirrina.NetInterfacesReq{})
	if err != nil {
		return []string{}, fmt.Errorf("unable to get host nics: %w", err)
	}

	for {
		hostNic, err := res.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return []string{}, fmt.Errorf("unable to get host nics: %w", err)
		}

		hostNics = append(hostNics, hostNic.GetInterfaceName())
	}

	return hostNics, nil
}

func GetHostVersion(ctx context.Context) (string, error) {
	var version string

	var err error

	var res *wrapperspb.StringValue

	res, err = serverClient.GetVersion(ctx, &emptypb.Empty{})
	if err != nil {
		return "", fmt.Errorf("unable to get host version: %w", err)
	}

	version = res.GetValue()

	return version, nil
}
