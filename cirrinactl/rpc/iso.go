package rpc

import (
	"cirrina/cirrina"
	"context"
)

func AddIso(j *cirrina.ISOInfo, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
	res, err := c.AddISO(ctx, j)
	if err != nil {
		return "", err
	}

	return res.Value, nil
}
