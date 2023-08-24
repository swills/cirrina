package rpc

import (
	"cirrina/cirrina"
	"context"
	"errors"
	"io"
)

func AddVM(namePtr *string, c cirrina.VMInfoClient, ctx context.Context, descrPtr *string, cpuPtr *uint32, memPtr *uint32) (reqId string, err error) {
	if *namePtr == "" {
		return "", errors.New("name not specified")
	}
	res, err := c.AddVM(ctx, &cirrina.VMConfig{
		Name:        namePtr,
		Description: descrPtr,
		Cpu:         cpuPtr,
		Mem:         memPtr,
	})
	if err != nil {
		return "", err
	}
	return res.Value, nil
}

func DeleteVM(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
	if *idPtr == "" {
		return "", errors.New("id not specified")
	}
	reqId, err := c.DeleteVM(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		return "", err
	}
	return reqId.Value, nil
}

func StopVM(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
	if *idPtr == "" {
		return "", errors.New("id not specified")
	}
	reqId, err := c.StopVM(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		return "", err
	}
	return reqId.Value, nil
}

func StartVM(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
	if *idPtr == "" {
		return "", errors.New("id not specified")
	}
	reqId, err := c.StartVM(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		return "", err
	}
	return reqId.Value, nil
}

func GetVMConfig(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (*cirrina.VMConfig, error) {
	if *idPtr == "" {
		return &cirrina.VMConfig{}, errors.New("id not specified")
	}
	res, err := c.GetVMConfig(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		return &cirrina.VMConfig{}, err
	}
	return res, nil
}

func GetVmIds(c cirrina.VMInfoClient, ctx context.Context) (ids []string, err error) {
	res, err := c.GetVMs(ctx, &cirrina.VMsQuery{})
	if err != nil {
		return ids, err
	}
	for {
		VM, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []string{}, err
		}
		ids = append(ids, VM.Value)
	}
	return ids, nil
}

func GetVMState(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
	if *idPtr == "" {
		return "", errors.New("id not specified")
	}
	res, err := c.GetVMState(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		return "", err
	}
	var vmstate string
	switch res.Status {
	case cirrina.VmStatus_STATUS_STOPPED:
		vmstate = "stopped"
	case cirrina.VmStatus_STATUS_STARTING:
		vmstate = "starting"
	case cirrina.VmStatus_STATUS_RUNNING:
		vmstate = "running"
	case cirrina.VmStatus_STATUS_STOPPING:
		vmstate = "stopping"
	}
	return vmstate, nil
}

func VmRunning(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (bool, error) {
	r, err := GetVMState(idPtr, c, ctx)
	if err != nil {
		return false, err
	}
	if r == "running" {
		return true, nil
	}
	return false, nil
}

func VmStopped(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (bool, error) {
	r, err := GetVMState(idPtr, c, ctx)
	if err != nil {
		return false, err
	}
	if r == "stopped" {
		return true, nil
	}
	return false, nil

}

func VmNameToId(name string, c cirrina.VMInfoClient, ctx context.Context) (rid string, err error) {
	found := false
	ids, err := GetVmIds(c, ctx)
	if err != nil {
		return "", err
	}
	for _, id := range ids {
		res, err := GetVMConfig(&id, c, ctx)
		if err != nil {
			return "", err
		}
		if *res.Name == name {
			if found == true {
				return "", errors.New("duplicate VM name")
			} else {
				found = true
				rid = id
			}
		}
	}
	return rid, nil
}

func UpdateVMConfig(newConfig *cirrina.VMConfig, c cirrina.VMInfoClient, ctx context.Context) error {
	_, err := c.UpdateVM(ctx, newConfig)
	return err
}
