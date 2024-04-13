package rpc

import (
	"errors"
	"io"

	"google.golang.org/grpc/status"

	"cirrina/cirrina"
)

func GetVMDisks(id string) ([]string, error) {
	var err error
	var res cirrina.VMInfo_GetVMDisksClient
	res, err = serverClient.GetVMDisks(defaultServerContext, &cirrina.VMID{Value: id})
	if err != nil {
		return []string{}, errors.New(status.Convert(err).Message())
	}

	var rv []string
	for {
		var r2 *cirrina.DiskId
		r2, err = res.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return []string{}, errors.New(status.Convert(err).Message())
		}
		rv = append(rv, r2.Value)
	}

	return rv, nil
}

func VMSetDisks(id string, diskIds []string) (bool, error) {
	var err error
	j := cirrina.SetDiskReq{
		Id:     id,
		Diskid: diskIds,
	}

	var res *cirrina.ReqBool
	res, err = serverClient.SetVMDisks(defaultServerContext, &j)
	if err != nil {
		return false, errors.New(status.Convert(err).Message())
	}

	return res.Success, nil
}
