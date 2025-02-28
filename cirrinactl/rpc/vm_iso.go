package rpc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"cirrina/cirrina"
)

func GetVMIsos(ctx context.Context, vmID string) ([]string, error) {
	var err error

	var isoIDs []string

	var getVMISOsClient cirrina.VMInfo_GetVMISOsClient

	getVMISOsClient, err = serverClient.GetVMISOs(ctx, &cirrina.VMID{Value: vmID})
	if err != nil {
		return []string{}, fmt.Errorf("unable to get VM isos: %w", err)
	}

	for {
		var isoID *cirrina.ISOID

		isoID, err = getVMISOsClient.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return []string{}, fmt.Errorf("unable to get VM isos: %w", err)
		}

		isoIDs = append(isoIDs, isoID.GetValue())
	}

	return isoIDs, nil
}

func VMSetIsos(ctx context.Context, id string, isoIDs []string) (bool, error) {
	var err error

	setISOReq := cirrina.SetISOReq{
		Id:    id,
		Isoid: isoIDs,
	}

	var res *cirrina.ReqBool

	res, err = serverClient.SetVMISOs(ctx, &setISOReq)
	if err != nil {
		return false, fmt.Errorf("unable to set VM isos: %w", err)
	}

	return res.GetSuccess(), nil
}
