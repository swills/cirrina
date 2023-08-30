package rpc

import (
	"cirrina/cirrina"
	"context"
	"errors"
	"io"
)

func AddIso(j *cirrina.ISOInfo, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
	res, err := c.AddISO(ctx, j)
	if err != nil {
		return "", err
	}

	return res.Value, nil
}

func GetIsoIds(c cirrina.VMInfoClient, ctx context.Context) (ids []string, err error) {
	res, err := c.GetISOs(ctx, &cirrina.ISOsQuery{})
	if err != nil {
		return []string{}, err
	}
	for {
		VM, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []string{}, err
		}
		ids = append(ids, VM.Value)
	}
	return ids, nil
}

func GetIsoInfo(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (isoInfo *cirrina.ISOInfo, err error) {
	if *idPtr == "" {
		return &cirrina.ISOInfo{}, errors.New("iso id not specified")
	}
	isoInfo, err = c.GetISOInfo(ctx, &cirrina.ISOID{Value: *idPtr})
	if err != nil {
		return &cirrina.ISOInfo{}, err
	}
	return isoInfo, nil
}

func IsoNameToId(namePtr *string, c cirrina.VMInfoClient, ctx context.Context) (isoId string, err error) {
	if namePtr == nil || *namePtr == "" {
		return "", errors.New("iso name not specified")
	}

	isoIds, err := GetIsoIds(c, ctx)
	if err != nil {
		return "", err
	}

	found := false
	for _, aIsoId := range isoIds {
		res, err := GetIsoInfo(&aIsoId, c, ctx)
		if err != nil {
			return "", err
		}
		if err != nil {
			return "", err
		}
		if *res.Name == *namePtr {
			if found {
				return "", errors.New("duplicate iso found")
			}
			found = true
			isoId = aIsoId
		}
	}
	if !found {
		return "", errors.New("iso not found")
	}
	return isoId, nil
}
