package rpc

import (
	"errors"
	"io"

	"google.golang.org/grpc/status"

	"cirrina/cirrina"
)

func GetVMIsos(id string) ([]string, error) {
	var err error
	var rv []string
	var getVMISOsClient cirrina.VMInfo_GetVMISOsClient
	getVMISOsClient, err = serverClient.GetVMISOs(defaultServerContext, &cirrina.VMID{Value: id})
	if err != nil {
		return []string{}, errors.New(status.Convert(err).Message())
	}
	for {
		var isoid *cirrina.ISOID
		isoid, err = getVMISOsClient.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return []string{}, errors.New(status.Convert(err).Message())
		}
		rv = append(rv, isoid.Value)
	}

	return rv, nil
}

func VMSetIsos(id string, isoIds []string) (bool, error) {
	var err error
	setISOReq := cirrina.SetISOReq{
		Id:    id,
		Isoid: isoIds,
	}
	var res *cirrina.ReqBool
	res, err = serverClient.SetVMISOs(defaultServerContext, &setISOReq)
	if err != nil {
		return false, errors.New(status.Convert(err).Message())
	}

	return res.Success, nil
}
