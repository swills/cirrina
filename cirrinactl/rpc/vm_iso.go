package rpc

import (
	"cirrina/cirrina"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"io"
)

func GetVmIsos(id string) ([]string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return []string{}, err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var res cirrina.VMInfo_GetVmISOsClient
	res, err = c.GetVmISOs(ctx, &cirrina.VMID{Value: id})
	if err != nil {
		return []string{}, errors.New(status.Convert(err).Message())
	}
	var rv []string
	for {
		var r2 *cirrina.ISOID
		r2, err = res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []string{}, errors.New(status.Convert(err).Message())
		}
		rv = append(rv, r2.Value)
	}
	return rv, nil
}

func VmSetIsos(id string, isoIds []string) (bool, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return false, err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	j := cirrina.SetISOReq{
		Id:    id,
		Isoid: isoIds,
	}
	var res *cirrina.ReqBool
	res, err = c.SetVmISOs(ctx, &j)
	if err != nil {
		return false, errors.New(status.Convert(err).Message())
	}
	return res.Success, nil
}
