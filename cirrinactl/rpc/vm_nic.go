package rpc

import (
	"errors"
	"io"

	"google.golang.org/grpc/status"

	"cirrina/cirrina"
)

func GetVMNics(id string) ([]string, error) {
	var err error
	var res cirrina.VMInfo_GetVMNicsClient
	res, err = serverClient.GetVMNics(defaultServerContext, &cirrina.VMID{Value: id})
	if err != nil {
		return []string{}, errors.New(status.Convert(err).Message())
	}
	var rv []string
	for {
		var r2 *cirrina.VmNicId
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

func VMSetNics(id string, nicIds []string) (bool, error) {
	var err error
	j := cirrina.SetNicReq{
		Vmid:    id,
		Vmnicid: nicIds,
	}
	var res *cirrina.ReqBool
	res, err = serverClient.SetVMNics(defaultServerContext, &j)
	if err != nil {
		return false, errors.New(status.Convert(err).Message())
	}

	return res.Success, nil
}
