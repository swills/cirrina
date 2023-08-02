package main

import (
	"cirrina/cirrina"
	"context"
	"fmt"
	"github.com/jedib0t/go-pretty/table"
	"io"
	"log"
	"os"
)

func addSwitch(namePtr *string, c cirrina.VMInfoClient, ctx context.Context, descrPtr *string, switchTypePtr *string) {
	var thisSwitchType cirrina.SwitchType
	if *namePtr == "" {
		log.Fatalf("Name not specified")
		return
	}
	if *switchTypePtr == "" {
		log.Fatalf("Switch type not specified")
		return
	}
	if *switchTypePtr == "IF" {
		thisSwitchType = cirrina.SwitchType_IF
	} else if *switchTypePtr == "NG" {
		thisSwitchType = cirrina.SwitchType_NG
	} else {
		log.Fatalf("Switch type must be either \"IF\" or \"NG\"")
		return
	}

	log.Printf("Creating switch %v type %v", *namePtr, *switchTypePtr)

	var thisSwitchInfo cirrina.SwitchInfo
	thisSwitchInfo.Name = namePtr
	thisSwitchInfo.Description = descrPtr
	thisSwitchInfo.SwitchType = &thisSwitchType

	res, err := c.AddSwitch(ctx, &thisSwitchInfo)
	if err != nil {
		log.Fatalf("could not create switch: %v", err)
		return
	}
	fmt.Printf("Created switch %v\n", res.Value)
}

func setSwitchUplink(c cirrina.VMInfoClient, ctx context.Context, switchIdPtr *string, uplinkNamePtr *string) {
	if *switchIdPtr == "" {
		log.Fatalf("switch id not specified")
		return
	}

	req := &cirrina.SwitchUplinkReq{}
	si := &cirrina.SwitchId{}
	si.Value = *switchIdPtr
	req.Switchid = si
	req.Uplink = uplinkNamePtr

	res, err := c.SetSwitchUplink(ctx, req)
	if err != nil {
		log.Fatalf("could not set switch uplink: %v", err)
	}
	if res.Success {
		fmt.Printf("Switch uplink set successful")
	} else {
		fmt.Printf("Switch uplink set failed")
	}

}

func rmSwitch(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	reqId, err := c.RemoveSwitch(ctx, &cirrina.SwitchId{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not delete switch: %v", err)
	}
	if reqId.Success {
		fmt.Printf("Deleted successful")
	} else {
		fmt.Printf("Delete failed")
	}
}

func getSwitch(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	res, err := c.GetSwitchInfo(ctx, &cirrina.SwitchId{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
	}
	fmt.Printf(
		"name: %v "+
			"description: %v "+
			"type: %v "+
			"uplink: %v"+
			"\n",
		*res.Name,
		*res.Description,
		*res.SwitchType,
		*res.Uplink,
	)

}

func getSwitches(c cirrina.VMInfoClient, ctx context.Context) {
	res, err := c.GetSwitches(ctx, &cirrina.SwitchesQuery{})

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{"NAME", "UUID", "TYPE", "UPLINK", "DESCRIPTION"})
	t.SetStyle(myTableStyle)

	if err != nil {
		log.Fatalf("could not get Switches: %v", err)
		return
	}
	for {
		VmSwitch, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("GetSwitches failed: %v", err)
		}
		res2, err := c.GetSwitchInfo(ctx, &cirrina.SwitchId{Value: VmSwitch.Value})
		if err != nil {
			log.Fatalf("could not get VM: %v", err)
		}
		switchType := "Unknown"
		if *res2.SwitchType == cirrina.SwitchType_IF {
			switchType = "bridge"
		} else if *res2.SwitchType == cirrina.SwitchType_NG {
			switchType = "netgraph"
		}
		t.AppendRow(table.Row{*res2.Name, VmSwitch.Value, switchType, *res2.Uplink, *res2.Description})
	}
	t.Render()
}

func setVmNicSwitch(c cirrina.VMInfoClient, ctx context.Context, vmNicId string, switchId string) {
	var vmnicid cirrina.VmNicId
	var vmswitchid cirrina.SwitchId

	if vmNicId == "" {
		log.Fatalf("vm NIC ID not specified")
		return
	}
	if switchId == "" {
		log.Fatalf("Switch ID not specified")
		return
	}

	vmnicid.Value = vmNicId
	vmswitchid.Value = switchId

	nicSwitchSettings := cirrina.SetVmNicSwitchReq{
		Vmnicid:  &vmnicid,
		Switchid: &vmswitchid,
	}
	r, err := c.SetVmNicSwitch(ctx, &nicSwitchSettings)
	if err != nil {
		log.Fatalf("could not set vm nic switch: %v", err)
	}
	if r.Success {
		log.Printf("Set VM Nic switch connection")
	} else {
		log.Printf("Failed to set vmNic switch")
	}
}
