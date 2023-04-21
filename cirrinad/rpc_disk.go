package main

import (
	"cirrina/cirrina"
	"cirrina/cirrinad/disk"
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
