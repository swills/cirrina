package rpc

import (
	"fmt"

	"cirrina/cirrina"
)

func ReqStat(reqID string) (ReqStatus, error) {
	var err error

	if reqID == "" {
		return ReqStatus{}, errReqEmpty
	}
	var res *cirrina.ReqStatus
	res, err = serverClient.RequestStatus(defaultServerContext, &cirrina.RequestID{Value: reqID})
	if err != nil {
		return ReqStatus{}, fmt.Errorf("request error: %w", err)
	}
	rv := ReqStatus{
		Complete: res.GetComplete(),
		Success:  res.GetSuccess(),
	}

	return rv, nil
}
