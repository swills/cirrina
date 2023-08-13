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

func addSwitch(namePtr *string, c cirrina.VMInfoClient, ctx context.Context, descrPtr *string, switchTypePtr *string) (switchId string, err error) {
	var thisSwitchType cirrina.SwitchType
	if *namePtr == "" {
		log.Fatalf("Name not specified")
		return
	}
	if *switchTypePtr == "" {
		return "", errors.New("switch type not specified")
	}
	if *switchTypePtr == "IF" || *switchTypePtr == "bridge" {
		thisSwitchType = cirrina.SwitchType_IF
	} else if *switchTypePtr == "NG" || *switchTypePtr == "netgraph" {
		thisSwitchType = cirrina.SwitchType_NG
	} else {
		return "", errors.New("switch type must be one of: IF, bridge, NG, netgraph")
	}

	var thisSwitchInfo cirrina.SwitchInfo
	thisSwitchInfo.Name = namePtr
	thisSwitchInfo.Description = descrPtr
	thisSwitchInfo.SwitchType = &thisSwitchType

	res, err := c.AddSwitch(ctx, &thisSwitchInfo)
	if err != nil {
		return "", err
	}
	return res.Value, nil
}

func setSwitchUplink(c cirrina.VMInfoClient, ctx context.Context, switchIdPtr *string, uplinkNamePtr *string) error {
	if *switchIdPtr == "" {
		return errors.New("switch id not specified")
	}

	req := &cirrina.SwitchUplinkReq{}
	si := &cirrina.SwitchId{}
	si.Value = *switchIdPtr
	req.Switchid = si
	req.Uplink = uplinkNamePtr

	_, err := c.SetSwitchUplink(ctx, req)
	if err != nil {
		em := fmt.Sprintf("could not set switch uplink: %s", err.Error())
		return errors.New(em)
	}
	return nil
}

func rmSwitch(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (err error) {
	if *idPtr == "" {
		return errors.New("id not specified")
	}
	reqId, err := c.RemoveSwitch(ctx, &cirrina.SwitchId{Value: *idPtr})
	if err != nil {
		return err
	}
	if !reqId.Success {
		return errors.New("failed to delete switch")
	}
	return nil
}

func getSwitch(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (switchInfo *cirrina.SwitchInfo, err error) {
	if *idPtr == "" {
		return &cirrina.SwitchInfo{}, errors.New("id not specified")
	}
	res, err := c.GetSwitchInfo(ctx, &cirrina.SwitchId{Value: *idPtr})
	if err != nil {
		return &cirrina.SwitchInfo{}, err
	}
	return res, nil
}

func getSwitchIds(c cirrina.VMInfoClient, ctx context.Context) ([]string, error) {
	var rv []string

	res, err := c.GetSwitches(ctx, &cirrina.SwitchesQuery{})
	if err != nil {
		em := fmt.Sprintf("could not get switches %s", err.Error())
		return []string{}, errors.New(em)
	}

	for {
		VmSwitch, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			em := fmt.Sprintf("error getting switches: %s", err.Error())
			return []string{}, errors.New(em)
		}
		rv = append(rv, VmSwitch.Value)
	}

	return rv, nil
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

func getSwitchByName(s *string, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
	rv := ""

	switchIds, err := getSwitchIds(c, ctx)
	if err != nil {
		em := fmt.Sprintf("error getting switches: %s", err.Error())
		return "", errors.New(em)
	}
	found := false

	for _, switchId := range switchIds {
		res2, err := c.GetSwitchInfo(ctx, &cirrina.SwitchId{Value: switchId})
		if err != nil {
			em := fmt.Sprintf("error getting switches: %s", err.Error())
			return "", errors.New(em)
		}
		if *res2.Name == *s {
			if found {
				return "", errors.New("duplicate switch found")
			} else {
				found = true
				rv = switchId
			}
		}

	}

	return rv, nil
}
