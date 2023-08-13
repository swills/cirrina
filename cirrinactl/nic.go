package main

import (
	"cirrina/cirrina"
	"context"
	"errors"
	"fmt"
	"github.com/jedib0t/go-pretty/table"
	"io"
	"log"
	"os"
)

func addVmNic(name *string, c cirrina.VMInfoClient, ctx context.Context, descrptr *string, nettypeptr *string, netdevtypeptr *string, macPtr *string, switchIdPtr *string) (nicId string, err error) {
	var thisVmNic cirrina.VmNicInfo
	var thisNetType cirrina.NetType
	var thisNetDevType cirrina.NetDevType

	thisVmNic.Name = name
	thisVmNic.Description = descrptr
	thisVmNic.Mac = macPtr
	thisVmNic.Switchid = switchIdPtr

	if *nettypeptr == "VIRTIONET" || *nettypeptr == "virtio-net" {
		thisNetType = cirrina.NetType_VIRTIONET
	} else if *nettypeptr == "E1000" || *nettypeptr == "e1000" {
		thisNetType = cirrina.NetType_E1000
	} else {
		return "", errors.New("net type must be either VIRTIONET or E1000")
	}
	if *netdevtypeptr == "TAP" || *netdevtypeptr == "tap" {
		thisNetDevType = cirrina.NetDevType_TAP
	} else if *nettypeptr == "VMNET" || *nettypeptr == "vmnet" {
		thisNetDevType = cirrina.NetDevType_VMNET
	} else if *nettypeptr == "NETGRAPH" || *nettypeptr == "netgraph" {
		thisNetDevType = cirrina.NetDevType_NETGRAPH
	} else {
		return "", errors.New("net dev type must be either TAP or VMNET or NETGRAPH")
	}

	thisVmNic.Nettype = &thisNetType
	thisVmNic.Netdevtype = &thisNetDevType

	res, err := c.AddVmNic(ctx, &thisVmNic)
	if err != nil {
		return "", err
	}
	return res.Value, nil
}

func rmVmNic(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	reqId, err := c.RemoveVmNic(ctx, &cirrina.VmNicId{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not delete switch: %v", err)
	}
	if reqId.Success {
		fmt.Printf("Deleted successful")
	} else {
		fmt.Printf("Delete failed")
	}
}

func getVmNics(c cirrina.VMInfoClient, ctx context.Context, idPtr *string) {
	if *idPtr == "" {
		getVmNicsAll(c, ctx)
	} else {
		getVmNicsOne(c, ctx, idPtr)
	}

}

func getVmNicsOne(c cirrina.VMInfoClient, ctx context.Context, idPtr *string) {
	res, err := c.GetVmNics(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not get VmNics: %v", err)
		return
	}
	for {
		VMNicId, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("GetVmNiss failed: %v", err)
		}
		fmt.Printf("VmNic: id: %v\n", VMNicId.Value)
	}

}

func getVmNicsAll(c cirrina.VMInfoClient, ctx context.Context) {
	res, err := c.GetVmNicsAll(ctx, &cirrina.VmNicsQuery{})
	if err != nil {
		log.Fatalf("could not get VmNics: %v", err)
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{"NAME", "UUID", "NETDEVTYPE", "NETTYPE", "RATELIMITED", "DESCRIPTION"})
	t.SetStyle(myTableStyle)

	for {
		VMNicId, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("GetVmNiss failed: %v", err)
		}
		res2, err := c.GetVmNicInfo(ctx, &cirrina.VmNicId{Value: VMNicId.Value})
		if err != nil {
			log.Fatalf("could not get VM: %v", err)
		}

		netDevType := "unknown"
		if *res2.Netdevtype == cirrina.NetDevType_TAP {
			netDevType = "tap"
		} else if *res2.Netdevtype == cirrina.NetDevType_VMNET {
			netDevType = "vmnet"
		} else if *res2.Netdevtype == cirrina.NetDevType_NETGRAPH {
			netDevType = "netgraph"
		}

		netType := "unknown"
		if *res2.Nettype == cirrina.NetType_VIRTIONET {
			netType = "virtio-net"
		} else if *res2.Nettype == cirrina.NetType_E1000 {
			netType = "e1000"
		}

		rateLimited := "unknown"
		if *res2.Ratelimit {
			rateLimited = "yes"
		} else {
			rateLimited = "no"
		}

		t.AppendRow(table.Row{
			*res2.Name,
			VMNicId.Value,
			netDevType,
			netType,
			rateLimited,
			*res2.Description,
		})
	}
	t.Render()
}

func getVmNic(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	var netTypeString string
	var netDevTypeString string
	var descriptionStr string

	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	res, err := c.GetVmNicInfo(ctx, &cirrina.VmNicId{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
	}

	if res.Description != nil {
		descriptionStr = *res.Description
	}

	if *res.Nettype == cirrina.NetType_VIRTIONET {
		netTypeString = "VirtioNet"
	} else if *res.Nettype == cirrina.NetType_E1000 {
		netTypeString = "E1000"
	}

	if *res.Netdevtype == cirrina.NetDevType_TAP {
		netDevTypeString = "TAP"
	} else if *res.Netdevtype == cirrina.NetDevType_VMNET {
		netDevTypeString = "VMNet"
	} else if *res.Netdevtype == cirrina.NetDevType_NETGRAPH {
		netDevTypeString = "Netgraph"
	}

	fmt.Printf(
		"name: %v "+
			"desc: %v "+
			"Mac: %v "+
			"Net_type: %v "+
			"Net_dev_type: %v "+
			"switch_id: %v "+
			"\n",
		*res.Name,
		descriptionStr,
		*res.Mac,
		netTypeString,
		netDevTypeString,
		*res.Switchid,
	)
}

func setVmNicVm(_ cirrina.VMInfoClient, _ context.Context) {

}

func deleteNic(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (err error) {
	if idPtr == nil || *idPtr == "" {
		return errors.New("disk id not specified")
	}
	_, err = c.RemoveVmNic(ctx, &cirrina.VmNicId{Value: *idPtr})
	if err != nil {
		return err
	}
	return nil
}

func getNicIds(c cirrina.VMInfoClient, ctx context.Context) (ids []string, err error) {
	res, err := c.GetVmNicsAll(ctx, &cirrina.VmNicsQuery{})
	if err != nil {
		return []string{}, err
	}
	for {
		aNic, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []string{}, err
		}
		ids = append(ids, aNic.Value)
	}
	return ids, nil
}

func getNicInfo(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (aNic *cirrina.VmNicInfo, err error) {
	if idPtr == nil || *idPtr == "" {
		return &cirrina.VmNicInfo{}, errors.New("nic id not specified")
	}
	res, err := c.GetVmNicInfo(ctx, &cirrina.VmNicId{Value: *idPtr})
	if err != nil {
		return &cirrina.VmNicInfo{}, err
	}
	return res, nil
}

func getNicByName(namePtr *string, c cirrina.VMInfoClient, ctx context.Context) (nicId string, err error) {
	if namePtr == nil || *namePtr == "" {
		return "", errors.New("disk name not specified")
	}

	nicIds, err := getNicIds(c, ctx)
	if err != nil {
		return "", err
	}

	found := false
	for _, aNicId := range nicIds {
		res, err := getNicInfo(&aNicId, c, ctx)
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
		return "", errors.New("disk not found")
	}
	return nicId, nil
}
