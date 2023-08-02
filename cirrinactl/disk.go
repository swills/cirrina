package main

import (
	"cirrina/cirrina"
	"context"
	"fmt"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"io"
	"log"
	"os"
)

func addDisk(namePtr *string, c cirrina.VMInfoClient, ctx context.Context, descrPtr *string, sizePtr *string, diskTypePtr *string) {
	var thisDiskType cirrina.DiskType

	if *namePtr == "" {
		log.Fatalf("Name not specified")
		return
	}

	if *diskTypePtr == "" {
		log.Fatalf("Disk type not specified")
		return
	}
	if *diskTypePtr == "NVME" {
		thisDiskType = cirrina.DiskType_NVME
	} else if *diskTypePtr == "AHCI" {
		thisDiskType = cirrina.DiskType_AHCIHD
	} else if *diskTypePtr == "VIRTIOBLK" {
		thisDiskType = cirrina.DiskType_VIRTIOBLK
	} else {
		log.Fatalf("Invalid disk type specified")
		return
	}

	res, err := c.AddDisk(ctx, &cirrina.DiskInfo{
		Name:        namePtr,
		Description: descrPtr,
		Size:        sizePtr,
		DiskType:    &thisDiskType,
	})
	if err != nil {
		log.Fatalf("could not create Disk: %v", err)
		return
	}
	fmt.Printf("Created Disk %v\n", res.Value)

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
