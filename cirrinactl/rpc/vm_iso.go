package rpc

import (
	"errors"
	"io"

	"google.golang.org/grpc/status"

	"cirrina/cirrina"
)

func GetVMIsos(id string) ([]string, error) {
	var err error
	var res cirrina.VMInfo_GetVMISOsClient
	res, err = serverClient.GetVMISOs(defaultServerContext, &cirrina.VMID{Value: id})
	if err != nil {
		return []string{}, errors.New(status.Convert(err).Message())
	}
	var rv []string
	for {
		var r2 *cirrina.ISOID
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

func VMSetIsos(id string, isoIds []string) (bool, error) {
	var err error
	j := cirrina.SetISOReq{
		Id:    id,
		Isoid: isoIds,
	}
	var res *cirrina.ReqBool
	res, err = serverClient.SetVMISOs(defaultServerContext, &j)
	if err != nil {
		return false, errors.New(status.Convert(err).Message())
	}

	return res.Success, nil
}
