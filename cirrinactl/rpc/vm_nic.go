package rpc

import (
	"cirrina/cirrina"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"io"
)

func GetVmNics(id string) ([]string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return []string{}, err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var res cirrina.VMInfo_GetVmNicsClient
	res, err = c.GetVmNics(ctx, &cirrina.VMID{Value: id})
	if err != nil {
		return []string{}, errors.New(status.Convert(err).Message())
	}
	var rv []string
	for {
		var r2 *cirrina.VmNicId
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

func VmSetNics(id string, nicIds []string) (bool, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return false, err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	j := cirrina.SetNicReq{
		Vmid:    id,
		Vmnicid: nicIds,
	}
	var res *cirrina.ReqBool
	res, err = c.SetVmNics(ctx, &j)
	if err != nil {
		return false, errors.New(status.Convert(err).Message())
	}
	return res.Success, nil
}
