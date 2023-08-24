package rpc

import (
	"cirrina/cirrina"
	"context"
	"errors"
	"io"
)

func AddDisk(aDiskInfo *cirrina.DiskInfo, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
	res, err := c.AddDisk(ctx, aDiskInfo)
	if err != nil {
		return "", err
	}
	return res.Value, nil
}

func GetDiskInfo(j string, c cirrina.VMInfoClient, ctx context.Context) (*cirrina.DiskInfo, error) {
	k, err := c.GetDiskInfo(ctx, &cirrina.DiskId{Value: j})
	if err != nil {
		return nil, err
	}

	return k, nil
}

func GetDisks(c cirrina.VMInfoClient, ctx context.Context) ([]string, error) {
	var rv []string

	res, err := c.GetDisks(ctx, &cirrina.DisksQuery{})
	if err != nil {
		return []string{}, err
	}

	for {
		VmDisk, err := res.Recv()
		if err == io.EOF {
			break
		}

		rv = append(rv, VmDisk.Value)
	}

	return rv, err
}

func RmDisk(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (bool, error) {
	res, err := c.RemoveDisk(ctx, &cirrina.DiskId{Value: *idPtr})
	if err != nil {
		return false, err
	}
	return res.Success, err
}

func GetDiskByName(namePtr *string, c cirrina.VMInfoClient, ctx context.Context) (diskId string, err error) {
	if namePtr == nil || *namePtr == "" {
		return "", errors.New("disk name not specified")
	}

	diskIds, err := GetDisks(c, ctx)
	if err != nil {
		return "", err
	}

	found := false
	for _, aDiskId := range diskIds {
		res, err := GetDiskInfo(aDiskId, c, ctx)
		if err != nil {
			return "", err
		}
		if err != nil {
			return "", err
		}
		if *res.Name == *namePtr {
			if found {
				return "", errors.New("duplicate disk found")
			}
			found = true
			diskId = aDiskId
		}
	}
	if !found {
		return "", errors.New("disk not found")
	}
	return diskId, nil
}
