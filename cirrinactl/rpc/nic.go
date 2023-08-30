package rpc

import (
	"cirrina/cirrina"
	"context"
	"errors"
	"io"
)

func AddNic(c cirrina.VMInfoClient, ctx context.Context, thisVmNic *cirrina.VmNicInfo) (*cirrina.VmNicId, error) {
	if *thisVmNic.Name == "" {
		return &cirrina.VmNicId{}, errors.New("nic name not specified")
	}
	return c.AddVmNic(ctx, thisVmNic)
}

func RmNic(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (bool, error) {
	if *idPtr == "" {
		return false, errors.New("id not specified")
	}
	reqId, err := c.RemoveVmNic(ctx, &cirrina.VmNicId{Value: *idPtr})
	if err != nil {
		return false, err
	}
	if reqId.Success {
		return true, nil
	} else {
		return false, nil
	}
}

func GetVmNicInfo(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (*cirrina.VmNicInfo, error) {
	res, err := c.GetVmNicInfo(ctx, &cirrina.VmNicId{Value: *idPtr})
	if err != nil {
		return &cirrina.VmNicInfo{}, err
	}
	return res, nil
}

func NicNameToId(namePtr *string, c cirrina.VMInfoClient, ctx context.Context) (nicId string, err error) {
	if namePtr == nil || *namePtr == "" {
		return "", errors.New("nic name not specified")
	}

	nicIds, err := GetVmNicsAll(c, ctx)
	if err != nil {
		return "", err
	}

	found := false
	for _, aNicId := range nicIds {
		res, err := GetVmNicInfo(&aNicId, c, ctx)
		if err != nil {
			return "", err
		}
		if *res.Name == *namePtr {
			if found {
				return "", errors.New("duplicate nic found")
			}
			found = true
			nicId = aNicId
		}
	}
	if !found {
		return "", errors.New("nic not found")
	}
	return nicId, nil
}

func NicIdToName(s string, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
	res, err := c.GetVmNicInfo(ctx, &cirrina.VmNicId{Value: s})
	print("")
	if err != nil {
		return "", err
	}
	return *res.Name, nil
}

func GetVmNicOne(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
	var rv string
	res, err := c.GetVmNics(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		return "", err
	}
	found := false
	for {
		VMNicId, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if found {
			return "", errors.New("duplicate nic")
		} else {
			found = true
			rv = VMNicId.Value
		}
	}
	return rv, nil
}

func GetVmNicsAll(c cirrina.VMInfoClient, ctx context.Context) ([]string, error) {
	var rv []string

	res, err := c.GetVmNicsAll(ctx, &cirrina.VmNicsQuery{})
	if err != nil {
		return []string{}, err
	}

	for {
		VMNicId, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []string{}, err
		}
		rv = append(rv, VMNicId.Value)
	}
	return rv, nil
}
