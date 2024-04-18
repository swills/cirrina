package rpc

import (
	"fmt"

	"cirrina/cirrina"
)

func ReqStat(id string) (ReqStatus, error) {
	var err error

	if id == "" {
		return ReqStatus{}, errReqEmpty
	}
	var res *cirrina.ReqStatus
	res, err = serverClient.RequestStatus(defaultServerContext, &cirrina.RequestID{Value: id})
	if err != nil {
		return ReqStatus{}, fmt.Errorf("request error: %w", err)
	}
	rv := ReqStatus{
		Complete: res.Complete,
		Success:  res.Success,
	}

	return rv, nil
}
