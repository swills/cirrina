package rpc

import (
	"errors"
	"io"

	"google.golang.org/grpc/status"

	"cirrina/cirrina"
)

func GetVMDisks(id string) ([]string, error) {
	var err error
	var rv []string
	var getVMDisksClient cirrina.VMInfo_GetVMDisksClient
	getVMDisksClient, err = serverClient.GetVMDisks(defaultServerContext, &cirrina.VMID{Value: id})
	if err != nil {
		return []string{}, errors.New(status.Convert(err).Message())
	}
	for {
		var diskID *cirrina.DiskId
		diskID, err = getVMDisksClient.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return []string{}, errors.New(status.Convert(err).Message())
		}
		rv = append(rv, diskID.Value)
	}

	return rv, nil
}

func VMSetDisks(id string, diskIds []string) (bool, error) {
	var err error
	setDiskReq := cirrina.SetDiskReq{
		Id:     id,
		Diskid: diskIds,
	}
	var res *cirrina.ReqBool
	res, err = serverClient.SetVMDisks(defaultServerContext, &setDiskReq)
	if err != nil {
		return false, errors.New(status.Convert(err).Message())
	}

	return res.Success, nil
}
