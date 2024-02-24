package rpc

import (
	"cirrina/cirrina"
	"errors"
	"google.golang.org/grpc/status"
)

func ReqStat(id string) (ReqStatus, error) {
	var err error

	if id == "" {
		return ReqStatus{}, errors.New("id not specified")
	}
	var res *cirrina.ReqStatus
	res, err = serverClient.RequestStatus(defaultServerContext, &cirrina.RequestID{Value: id})
	if err != nil {
		return ReqStatus{}, errors.New(status.Convert(err).Message())
	}
	rv := ReqStatus{
		Complete: res.Complete,
		Success:  res.Success,
	}
	return rv, nil
}
