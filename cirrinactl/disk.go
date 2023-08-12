package main

import (
	"cirrina/cirrina"
	"context"
	"errors"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"io"
	"log"
	"os"
)

func addDisk(namePtr *string, c cirrina.VMInfoClient, ctx context.Context, descrPtr *string, sizePtr *string, diskTypePtr *string) (diskId string, err error) {
	var thisDiskType cirrina.DiskType

	if *namePtr == "" {
		return "", errors.New("name not specified")
	}

	if *diskTypePtr == "" {
		return "", errors.New("disk type not specified")
	}

	if *diskTypePtr == "NVME" || *diskTypePtr == "nvme" {
		thisDiskType = cirrina.DiskType_NVME
	} else if *diskTypePtr == "AHCI" || *diskTypePtr == "ahci" || *diskTypePtr == "ahcihd" {
		thisDiskType = cirrina.DiskType_AHCIHD
	} else if *diskTypePtr == "VIRTIOBLK" || *diskTypePtr == "virtioblk" || *diskTypePtr == "virtio-blk" {
		thisDiskType = cirrina.DiskType_VIRTIOBLK
	} else {
		return "", errors.New("invalid disk type specified")
	}

	res, err := c.AddDisk(ctx, &cirrina.DiskInfo{
		Name:        namePtr,
		Description: descrPtr,
		Size:        sizePtr,
		DiskType:    &thisDiskType,
	})
	if err != nil {
		return "", err
	}
	return res.Value, nil
}

func getDisks(c cirrina.VMInfoClient, ctx context.Context) {
	res, err := c.GetDisks(ctx, &cirrina.DisksQuery{})
	if err != nil {
		log.Fatalf("could not get disks: %v", err)
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{"NAME", "UUID", "SIZE", "TYPE", "DESCRIPTION"})
	t.SetStyle(myTableStyle)
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 3, Align: text.AlignLeft, AlignHeader: text.AlignLeft},
	})
	for {
		VmDisk, err := res.Recv()
		if err == io.EOF {
			break
		}
		res2, err := c.GetDiskInfo(ctx, &cirrina.DiskId{Value: VmDisk.Value})
		if err != nil {
			log.Fatalf("could not get disks: %v", err)
			return
		}

		diskType := "unknown"
		if *res2.DiskType == cirrina.DiskType_NVME {
			diskType = "nvme"
		} else if *res2.DiskType == cirrina.DiskType_AHCIHD {
			diskType = "ahcihd"
		} else if *res2.DiskType == cirrina.DiskType_VIRTIOBLK {
			diskType = "virtio-blk"
		}

		t.AppendRow(table.Row{
			*res2.Name,
			VmDisk.Value,
			*res2.SizeNum,
			diskType,
			*res2.Description,
		})
	}
	t.Render()
}
