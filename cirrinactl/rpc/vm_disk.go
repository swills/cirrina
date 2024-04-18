package rpc

import (
	"errors"
	"fmt"
	"io"

	"cirrina/cirrina"
)

func GetVMDisks(id string) ([]string, error) {
	var err error
	var rv []string
	var getVMDisksClient cirrina.VMInfo_GetVMDisksClient
	getVMDisksClient, err = serverClient.GetVMDisks(defaultServerContext, &cirrina.VMID{Value: id})
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
		rv = append(rv, diskID.Value)
	}

	return rv, nil
}

func VMSetDisks(id string, diskIDs []string) (bool, error) {
	var err error
	setDiskReq := cirrina.SetDiskReq{
		Id:     id,
		Diskid: diskIDs,
	}
	var res *cirrina.ReqBool
	res, err = serverClient.SetVMDisks(defaultServerContext, &setDiskReq)
	if err != nil {
		return false, fmt.Errorf("unable to set VM disks: %w", err)
	}

	return res.Success, nil
}
