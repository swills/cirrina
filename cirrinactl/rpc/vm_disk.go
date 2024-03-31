package rpc

import (
	"cirrina/cirrina"
	"errors"
	"io"

	"google.golang.org/grpc/status"
)

func GetVmDisks(id string) ([]string, error) {
	var err error
	var res cirrina.VMInfo_GetVmDisksClient
	res, err = serverClient.GetVmDisks(defaultServerContext, &cirrina.VMID{Value: id})
	if err != nil {
		return []string{}, errors.New(status.Convert(err).Message())
	}

	var rv []string
	for {
		var r2 *cirrina.DiskId
		r2, err = res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []string{}, errors.New(status.Convert(err).Message())
		}
		rv = append(rv, r2.Value)
	}
	return rv, nil
}

func VmSetDisks(id string, diskIds []string) (bool, error) {
	var err error
	j := cirrina.SetDiskReq{
		Id:     id,
		Diskid: diskIds,
	}

	var res *cirrina.ReqBool
	res, err = serverClient.SetVmDisks(defaultServerContext, &j)
	if err != nil {
		return false, errors.New(status.Convert(err).Message())
	}
	return res.Success, nil
}
