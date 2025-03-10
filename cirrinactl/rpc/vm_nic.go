package rpc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"cirrina/cirrina"
)

func GetVMNics(ctx context.Context, id string) ([]string, error) {
	var err error

	var res cirrina.VMInfo_GetVMNicsClient

	res, err = serverClient.GetVMNics(ctx, &cirrina.VMID{Value: id})
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

		vmNics = append(vmNics, res2.GetValue())
	}

	return vmNics, nil
}

func VMSetNics(ctx context.Context, id string, nicIDs []string) (bool, error) {
	var err error

	setNicReq := cirrina.SetNicReq{
		Vmid:    id,
		Vmnicid: nicIDs,
	}

	var res *cirrina.ReqBool

	res, err = serverClient.SetVMNics(ctx, &setNicReq)
	if err != nil {
		return false, fmt.Errorf("unable to set VM nics: %w", err)
	}

	return res.GetSuccess(), nil
}

func GetVMNicName(ctx context.Context, nicID string) (string, error) {
	var err error

	if nicID == "" {
		return "", errNicEmptyID
	}

	var res *wrapperspb.StringValue

	res, err = serverClient.GetVMNicName(ctx, &cirrina.VmNicId{Value: nicID})
	if err != nil {
		return "", fmt.Errorf("unable to get VM name: %w", err)
	}

	return res.GetValue(), nil
}

func GetVMNicID(ctx context.Context, name string) (string, error) {
	var err error

	if name == "" {
		return "", errNicEmptyName
	}

	var res *cirrina.VmNicId

	res, err = serverClient.GetVMNicID(ctx, wrapperspb.String(name))
	if err != nil {
		return "", fmt.Errorf("unable to get NIC ID: %w", err)
	}

	return res.GetValue(), nil
}
