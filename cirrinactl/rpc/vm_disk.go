package rpc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"cirrina/cirrina"
)

func GetVMDisks(ctx context.Context, vmID string) ([]string, error) {
	var err error

	var diskIDs []string

	var getVMDisksClient cirrina.VMInfo_GetVMDisksClient

	getVMDisksClient, err = serverClient.GetVMDisks(ctx, &cirrina.VMID{Value: vmID})
	if err != nil {
		return []string{}, fmt.Errorf("unable to get VM disks: %w", err)
	}

	for {
		var diskID *cirrina.DiskId

		diskID, err = getVMDisksClient.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return []string{}, fmt.Errorf("unable to get VM disks: %w", err)
		}

		diskIDs = append(diskIDs, diskID.GetValue())
	}

	return diskIDs, nil
}

func VMSetDisks(ctx context.Context, id string, diskIDs []string) (bool, error) {
	var err error

	setDiskReq := cirrina.SetDiskReq{
		Id:     id,
		Diskid: diskIDs,
	}

	var res *cirrina.ReqBool

	res, err = serverClient.SetVMDisks(ctx, &setDiskReq)
	if err != nil {
		return false, fmt.Errorf("unable to set VM disks: %w", err)
	}

	return res.GetSuccess(), nil
}
