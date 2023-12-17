package rpc

import (
	"cirrina/cirrina"
	"context"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func ReqStat(id string) (ReqStatus, error) {
	var conn *grpc.ClientConn
	var c cirrina.VMInfoClient
	var ctx context.Context
	var cancel context.CancelFunc
	var err error
	conn, c, ctx, cancel, err = SetupConn()
	if err != nil {
		return ReqStatus{}, err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	if id == "" {
		return ReqStatus{}, errors.New("id not specified")
	}
	var res *cirrina.ReqStatus
	res, err = c.RequestStatus(ctx, &cirrina.RequestID{Value: id})
	if err != nil {
		return ReqStatus{}, errors.New(status.Convert(err).Message())
	}
	rv := ReqStatus{
		Complete: res.Complete,
		Success:  res.Success,
	}
	return rv, nil
}
