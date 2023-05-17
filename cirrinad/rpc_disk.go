package main

import (
	"cirrina/cirrina"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/vm"
	"context"
	"errors"
	"golang.org/x/exp/slog"
)

func (s *server) GetDisks(_ *cirrina.DisksQuery, stream cirrina.VMInfo_GetDisksServer) error {
	var disks []*disk.Disk
	var DiskId cirrina.DiskId
	disks = disk.GetAll()
	for e := range disks {
		DiskId.Value = disks[e].ID
		err := stream.Send(&DiskId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *server) AddDisk(_ context.Context, i *cirrina.DiskInfo) (*cirrina.DiskId, error) {

	diskInst, err := disk.Create(*i.Name, *i.Description, *i.Size)
	if err != nil {
		return &cirrina.DiskId{}, err
	}
	if diskInst != nil && diskInst.ID != "" {
		return &cirrina.DiskId{Value: diskInst.ID}, nil
	} else {
		return &cirrina.DiskId{}, errors.New("unknown error creating disk")
	}
}

func (s *server) GetDiskInfo(_ context.Context, i *cirrina.DiskId) (*cirrina.DiskInfo, error) {
	var ic cirrina.DiskInfo
	slog.Debug("GetDiskInfo", "disk", i.Value)
	if i.Value == "" {
		return &ic, nil
	}
	diskInst, err := disk.GetById(i.Value)
	if err != nil {
		slog.Error("error getting disk", "disk", i.Value, "err", err)
	}
	ic.Name = &diskInst.Name
	ic.Description = &diskInst.Description
	return &ic, nil
}

func (s *server) RemoveDisk(_ context.Context, i *cirrina.DiskId) (*cirrina.ReqBool, error) {
	slog.Debug("deleting disk", "diskid", i.Value)
	re := cirrina.ReqBool{}
	re.Success = false

	if i.Value == "" {
		return &re, errors.New("disk id must be specified")
	}

	_, err := disk.GetById(i.Value)
	if err != nil {
		slog.Debug("error getting disk, does not exist", "disk", i.Value, "err", err)
		return &re, err
	}

	// check that disk is not in use by a VM
	allVMs := vm.GetAll()
	for _, thisVm := range allVMs {
		slog.Debug("vm checks", "vm", thisVm)
		thisVmDisks, err := thisVm.GetDisks()
		if err != nil {
			return &re, err
		}
		for _, vmDisk := range thisVmDisks {
			if vmDisk.ID == i.Value {
				return &re, errors.New("disk in use by VM")
			}
		}
	}

	res := disk.Delete(i.Value)
	if res != nil {
		slog.Error("error deleting disk", "res", res)
		return &re, errors.New("error deleting disk")
	}

	re.Success = true
	return &re, nil
}
