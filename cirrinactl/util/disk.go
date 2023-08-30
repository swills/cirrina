package util

import (
	"cirrina/cirrina"
	"cirrina/cirrinactl/rpc"
	"context"
	"errors"
	"fmt"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"google.golang.org/grpc/status"
	"os"
)

func AddDisk(namePtr *string, c cirrina.VMInfoClient, ctx context.Context, descrPtr *string, sizePtr *string, diskTypePtr *string) (diskId string, err error) {
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

	aDiskInfo := &cirrina.DiskInfo{
		Name:        namePtr,
		Description: descrPtr,
		Size:        sizePtr,
		DiskType:    &thisDiskType,
	}

	res, err := rpc.AddDisk(aDiskInfo, c, ctx)
	return res, err
}

func GetDisks(c cirrina.VMInfoClient, ctx context.Context) (err error) {
	res, err := rpc.GetDisks(c, ctx)
	if err != nil {
		return err
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{"NAME", "UUID", "SIZE", "TYPE", "DESCRIPTION"})
	t.SetStyle(myTableStyle)
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 3, Align: text.AlignLeft, AlignHeader: text.AlignLeft},
	})

	for _, r := range res {
		res2, err := rpc.GetDiskInfo(r, c, ctx)
		if err != nil {
			return err
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
			r,
			*res2.SizeNum,
			diskType,
			*res2.Description,
		})
	}
	t.Render()
	return nil
}

func RmDisk(name string, c cirrina.VMInfoClient, ctx context.Context) {
	diskId, err := rpc.DiskNameToId(&name, c, ctx)
	if err != nil {
		s := status.Convert(err)
		fmt.Printf("error: could not delete disk: %s\n", s.Message())
		return
	}
	_, err = rpc.RmDisk(&diskId, c, ctx)
	if err != nil {
		s := status.Convert(err)
		fmt.Printf("error: could not delete disk: %s\n", s.Message())
		return
	}
}
