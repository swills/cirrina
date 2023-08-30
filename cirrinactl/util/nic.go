package util

import (
	"cirrina/cirrina"
	"cirrina/cirrinactl/rpc"
	"context"
	"errors"
	"fmt"
	"github.com/jedib0t/go-pretty/table"
	"google.golang.org/grpc/status"
	"log"
	"os"
)

func AddVmNic(name *string, c cirrina.VMInfoClient, ctx context.Context, descrptr *string, nettypeptr *string, netdevtypeptr *string, macPtr *string, switchIdPtr *string) (nicId string, err error) {
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

	res, err := rpc.AddNic(c, ctx, &thisVmNic)
	if err != nil {
		return "", err
	}
	return res.Value, nil
}

func RmVmNic(name *string, c cirrina.VMInfoClient, ctx context.Context) {
	nicId, err := rpc.NicNameToId(name, c, ctx)
	if err != nil {
		s := status.Convert(err)
		fmt.Printf("error: could not delete nic: %s\n", s.Message())
		return
	}
	res, err := rpc.RmNic(&nicId, c, ctx)
	if err != nil {
		log.Fatalf("could not delete switch: %v", err)
	}
	if !res {
		fmt.Printf("Delete failed\n")
	}
}

func GetVmNicsOne(c cirrina.VMInfoClient, ctx context.Context, idPtr *string) {
	res, err := rpc.GetVmNicOne(idPtr, c, ctx)
	if err != nil {
		log.Fatalf("could not get Nic: %v", err)
		return
	}
	fmt.Printf("VmNic: id: %v\n", res)
}

func GetVmNicsAll(c cirrina.VMInfoClient, ctx context.Context) {
	res, err := rpc.GetVmNicsAll(c, ctx)
	if err != nil {
		log.Fatalf("could not get VmNics: %v", err)
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{"NAME", "UUID", "NETDEVTYPE", "NETTYPE", "RATELIMITED", "DESCRIPTION"})
	t.SetStyle(myTableStyle)

	for _, r := range res {
		res2, err := rpc.GetVmNicInfo(&r, c, ctx)
		if err != nil {
			log.Fatalf("could not get VmNics: %v", err)
			return
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
			r,
			netDevType,
			netType,
			rateLimited,
			*res2.Description,
		})
	}
	t.Render()
}

func GetVmNic(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	var netTypeString string
	var netDevTypeString string
	var descriptionStr string

	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	res, err := rpc.GetVmNicInfo(idPtr, c, ctx)
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

func NicSetSwitch(nicId string, switchId string, c cirrina.VMInfoClient, ctx context.Context) {
	res, err := rpc.SetVmNicSwitch(c, ctx, nicId, switchId)
	if err != nil {
		s := status.Convert(err)
		fmt.Printf("error111: could not add nic to switch: %s\n", s.Message())
	}
	if res {
		fmt.Printf("Added NIC to switch\n")
	} else {
		fmt.Printf("Failed to add NIC to switch\n")
	}
}
