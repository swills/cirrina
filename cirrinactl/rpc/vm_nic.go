package rpc

import (
	"errors"
	"fmt"
	"io"

	"cirrina/cirrina"
)

func GetVMNics(id string) ([]string, error) {
	var err error
	var res cirrina.VMInfo_GetVMNicsClient
	res, err = serverClient.GetVMNics(defaultServerContext, &cirrina.VMID{Value: id})
	if err != nil {
		return []string{}, fmt.Errorf("unable to get VM nics: %w", err)
	}
	var vmNics []string
	for {
		var res2 *cirrina.VmNicId
		res2, err = res.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return []string{}, fmt.Errorf("unable to get VM nics: %w", err)
		}
		vmNics = append(vmNics, res2.Value)
	}

	return vmNics, nil
}

func VMSetNics(id string, nicIDs []string) (bool, error) {
	var err error
	j := cirrina.SetNicReq{
		Vmid:    id,
		Vmnicid: nicIDs,
	}
	var res *cirrina.ReqBool
	res, err = serverClient.SetVMNics(defaultServerContext, &j)
	if err != nil {
		return false, fmt.Errorf("unable to set VM nics: %w", err)
	}

	return res.Success, nil
}
