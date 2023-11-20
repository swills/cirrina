package util

import (
	"cirrina/cirrina"
	"cirrina/cirrinactl/rpc"
	"context"
	"errors"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/table"
	"google.golang.org/grpc/status"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
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

//func GetVmNicsOne(c cirrina.VMInfoClient, ctx context.Context, idPtr *string) {
//	res, err := rpc.GetVmNicOne(idPtr, c, ctx)
//	if err != nil {
//		log.Fatalf("could not get Nic: %v", err)
//		return
//	}
//	fmt.Printf("VmNic: id: %v\n", res)
//}

func GetVmNicsAll(c cirrina.VMInfoClient, ctx context.Context, useHumanize bool) {
	res, err := rpc.GetVmNicsAll(c, ctx)
	if err != nil {
		log.Fatalf("could not get VmNics: %v", err)
		return
	}

	var names []string
	type ThisNicInfo struct {
		id          string
		mac         string
		nettype     string
		netdevtype  string
		uplink      string
		vm          string
		ratelimited string
		ratein      string
		rateout     string
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
		var rateins string
		var rateouts string
		if *res2.Ratelimit {
			rateLimited = "yes"
			if useHumanize {
				rateins = humanize.Bytes(*res2.Ratein)
				rateins = strings.Replace(rateins, "B", "b", 1) + "ps"
				rateouts = humanize.Bytes(*res2.Rateout)
				rateouts = strings.Replace(rateouts, "B", "b", 1) + "ps"
			} else {
				rateins = strconv.FormatUint(*res2.Ratein, 10)
				rateouts = strconv.FormatUint(*res2.Rateout, 10)
			}
		} else {
			rateLimited = "no"
		}

		uplinkName := ""
		if res2.Switchid != nil && *res2.Switchid != "" {
			uplinkName, err = rpc.SwitchIdToName(*res2.Switchid, c, ctx)
			if err != nil {
				log.Fatalf("could not get VmNics: %v", err)
				return
			}
		}

		vmName, err := rpc.NicGetVm(&id, c, ctx)
		if err != nil {
			log.Fatalf("could not get VmNic VM: %v", err)
			return
		}

		aIsoInfo := ThisNicInfo{
			id:          id,
			mac:         *res2.Mac,
			nettype:     netType,
			netdevtype:  netDevType,
			uplink:      uplinkName,
			vm:          vmName,
			ratelimited: rateLimited,
			ratein:      rateins,
			rateout:     rateouts,
			descr:       *res2.Description,
		}
		nicInfos[*res2.Name] = aIsoInfo
		names = append(names, *res2.Name)
	}

	sort.Strings(names)

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{"NAME", "UUID", "MAC", "TYPE", "DEVTYPE", "UPLINK", "VM", "RATE-LIMITED", "RATE-IN", "RATE-OUT", "DESCRIPTION"})
	t.SetStyle(myTableStyle)
	for _, name := range names {
		t.AppendRow(table.Row{
			name,
			nicInfos[name].id,
			nicInfos[name].mac,
			nicInfos[name].nettype,
			nicInfos[name].netdevtype,
			nicInfos[name].uplink,
			nicInfos[name].vm,
			nicInfos[name].ratelimited,
			nicInfos[name].ratein,
			nicInfos[name].rateout,
			nicInfos[name].descr,
		})

	}

	t.Render()
}

//func GetVmNic(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
//	var netTypeString string
//	var netDevTypeString string
//	var descriptionStr string
//
//	if *idPtr == "" {
//		log.Fatalf("ID not specified")
//		return
//	}
//	res, err := rpc.GetVmNicInfo(idPtr, c, ctx)
//	if err != nil {
//		log.Fatalf("could not get VM: %v", err)
//	}
//
//	if res.Description != nil {
//		descriptionStr = *res.Description
//	}
//
//	if *res.Nettype == cirrina.NetType_VIRTIONET {
//		netTypeString = "VirtioNet"
//	} else if *res.Nettype == cirrina.NetType_E1000 {
//		netTypeString = "E1000"
//	}
//
//	if *res.Netdevtype == cirrina.NetDevType_TAP {
//		netDevTypeString = "TAP"
//	} else if *res.Netdevtype == cirrina.NetDevType_VMNET {
//		netDevTypeString = "VMNet"
//	} else if *res.Netdevtype == cirrina.NetDevType_NETGRAPH {
//		netDevTypeString = "Netgraph"
//	}
//
//	fmt.Printf(
//		"name: %v "+
//			"desc: %v "+
//			"Mac: %v "+
//			"Net_type: %v "+
//			"Net_dev_type: %v "+
//			"switch_id: %v "+
//			"\n",
//		*res.Name,
//		descriptionStr,
//		*res.Mac,
//		netTypeString,
//		netDevTypeString,
//		*res.Switchid,
//	)
//}

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
