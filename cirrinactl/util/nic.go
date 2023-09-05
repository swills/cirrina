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
	"sort"
	"strconv"
)

func AddVmNic(name *string, c cirrina.VMInfoClient, ctx context.Context, descrptr *string,
	nettypeptr *string, netdevtypeptr *string, macPtr *string, switchIdPtr *string, rateLimit *bool,
	rateIn *uint64, rateOut *uint64) (nicId string, err error) {
	var thisVmNic cirrina.VmNicInfo
	var thisNetType cirrina.NetType
	var thisNetDevType cirrina.NetDevType

	thisVmNic.Name = name
	thisVmNic.Description = descrptr
	thisVmNic.Mac = macPtr
	thisVmNic.Switchid = switchIdPtr
	thisVmNic.Ratelimit = rateLimit
	thisVmNic.Ratein = rateIn
	thisVmNic.Rateout = rateOut

	if *nettypeptr == "VIRTIONET" || *nettypeptr == "virtio-net" {
		thisNetType = cirrina.NetType_VIRTIONET
	} else if *nettypeptr == "E1000" || *nettypeptr == "e1000" {
		thisNetType = cirrina.NetType_E1000
	} else {
		return "", errors.New("net type must be either VIRTIONET or E1000")
	}
	if *netdevtypeptr == "TAP" || *netdevtypeptr == "tap" {
		thisNetDevType = cirrina.NetDevType_TAP
	} else if *netdevtypeptr == "VMNET" || *netdevtypeptr == "vmnet" {
		thisNetDevType = cirrina.NetDevType_VMNET
	} else if *netdevtypeptr == "NETGRAPH" || *netdevtypeptr == "netgraph" {
		thisNetDevType = cirrina.NetDevType_NETGRAPH
	} else {
		return "", errors.New("net dev type must be one of TAP or VMNET or NETGRAPH")
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

	t.AppendHeader(table.Row{"NAME", "UUID", "NETDEVTYPE", "NETTYPE", "RATELIMITED", "RATE-IN", "RATE-OUT", "UPLINK", "DESCRIPTION"})
	t.SetStyle(myTableStyle)

	var names []string
	type ThisNicInfo struct {
		id          string
		nettype     string
		netdevtype  string
		ratelimited string
		ratein      string
		rateout     string
		uplink      string
		descr       string
	}

	nicInfos := make(map[string]ThisNicInfo)

	for _, id := range res {
		res2, err := rpc.GetVmNicInfo(&id, c, ctx)
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

		rateins := strconv.FormatUint(*res2.Ratein, 10)
		rateouts := strconv.FormatUint(*res2.Rateout, 10)

		uplinkName := ""
		if res2.Switchid != nil && *res2.Switchid != "" {
			uplinkName, err = rpc.SwitchIdToName(*res2.Switchid, c, ctx)
			if err != nil {
				log.Fatalf("could not get VmNics: %v", err)
				return
			}
		}

		aIsoInfo := ThisNicInfo{
			id:          id,
			nettype:     netType,
			netdevtype:  netDevType,
			ratelimited: rateLimited,
			ratein:      rateins,
			rateout:     rateouts,
			uplink:      uplinkName,
			descr:       *res2.Description,
		}
		nicInfos[*res2.Name] = aIsoInfo
		names = append(names, *res2.Name)
	}

	sort.Strings(names)

	for _, a := range names {
		t.AppendRow(table.Row{
			a,
			nicInfos[a].id,
			nicInfos[a].netdevtype,
			nicInfos[a].nettype,
			nicInfos[a].ratelimited,
			nicInfos[a].ratein,
			nicInfos[a].rateout,
			nicInfos[a].uplink,
			nicInfos[a].descr,
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
