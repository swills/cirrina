package rpc

import (
	"cirrina/cirrina"
	"context"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"io"
)

func GetHostNics() (rv []*cirrina.NetIf, err error) {
	var conn *grpc.ClientConn
	var c cirrina.VMInfoClient
	var ctx context.Context
	var cancel context.CancelFunc
	conn, c, ctx, cancel, err = SetupConn()
	if err != nil {
		return []*cirrina.NetIf{}, err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var res cirrina.VMInfo_GetNetInterfacesClient
	res, err = c.GetNetInterfaces(ctx, &cirrina.NetInterfacesReq{})
	if err != nil {
		return []*cirrina.NetIf{}, errors.New(status.Convert(err).Message())
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

func GetHostVersion() (version string, err error) {
	var conn *grpc.ClientConn
	var c cirrina.VMInfoClient
	var ctx context.Context
	var cancel context.CancelFunc
	conn, c, ctx, cancel, err = SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var res *wrapperspb.StringValue
	res, err = c.GetVersion(ctx, &emptypb.Empty{})
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	version = res.Value
	return version, nil
}
