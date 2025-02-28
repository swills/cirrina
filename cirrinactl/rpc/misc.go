package rpc

import (
	"context"
	"fmt"

	"cirrina/cirrina"
)

func ReqStat(ctx context.Context, reqID string) (ReqStatus, error) {
	var err error

	if reqID == "" {
		return ReqStatus{}, errReqEmpty
	}

	var res *cirrina.ReqStatus

	res, err = serverClient.RequestStatus(ctx, &cirrina.RequestID{Value: reqID})
	if err != nil {
		return ReqStatus{}, fmt.Errorf("request error: %w", err)
	}

	rv := ReqStatus{
		Complete: res.GetComplete(),
		Success:  res.GetSuccess(),
	}

	return rv, nil
}
