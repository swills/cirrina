package util

import (
	"cirrina/cirrina"
	"cirrina/cirrinactl/rpc"
	"context"
	"errors"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"google.golang.org/grpc/status"
	"os"
	"sort"
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

func GetDisks(c cirrina.VMInfoClient, ctx context.Context, useHumanize bool) (err error) {
	res, err := rpc.GetDisks(c, ctx)
	if err != nil {
		return err
	}

	var names []string
	type ThisDiskInfo struct {
		id       string
		diskType string
		size     uint64
		usage    uint64
		vm       string
		descr    string
	}

	diskInfos := make(map[string]ThisDiskInfo)

	for _, id := range res {
		res2, err := rpc.GetDiskInfo(id, c, ctx)
		if err != nil {
			return err
		}

		aDiskType := "unknown"
		if *res2.DiskType == cirrina.DiskType_NVME {
			aDiskType = "nvme"
		} else if *res2.DiskType == cirrina.DiskType_AHCIHD {
			aDiskType = "ahcihd"
		} else if *res2.DiskType == cirrina.DiskType_VIRTIOBLK {
			aDiskType = "virtio-blk"
		}

		vmName, err := rpc.DiskGetVm(&id, c, ctx)
		if err != nil {
			return err
		}

		aVmInfo := ThisDiskInfo{
			id:       id,
			diskType: aDiskType,
			size:     *res2.SizeNum,
			usage:    *res2.UsageNum,
			vm:       vmName,
			descr:    *res2.Description,
		}
		diskInfos[*res2.Name] = aVmInfo
		names = append(names, *res2.Name)

	}

	sort.Strings(names)

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"NAME", "UUID", "TYPE", "SIZE", "USAGE", "VM", "DESCRIPTION"})
	t.SetStyle(myTableStyle)
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 4, Align: text.AlignRight, AlignHeader: text.AlignRight},
		{Number: 5, Align: text.AlignRight, AlignHeader: text.AlignRight},
	})
	for _, name := range names {
		if useHumanize {
			t.AppendRow(table.Row{
				name,
				diskInfos[name].id,
				diskInfos[name].diskType,
				humanize.IBytes(diskInfos[name].size),
				humanize.IBytes(diskInfos[name].usage),
				diskInfos[name].vm,
				diskInfos[name].descr,
			})
		} else {
			t.AppendRow(table.Row{
				name,
				diskInfos[name].id,
				diskInfos[name].diskType,
				diskInfos[name].size,
				diskInfos[name].usage,
				diskInfos[name].vm,
				diskInfos[name].descr,
			})
		}
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
