package rpc

import (
	"cirrina/cirrina"
	"context"
	"errors"
	"io"
	"log"
)

func getSwitchIds(c cirrina.VMInfoClient, ctx context.Context) ([]string, error) {
	var rv []string

	res, err := c.GetSwitches(ctx, &cirrina.SwitchesQuery{})
	if err != nil {
		return []string{}, err
	}

	for {
		VmSwitch, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []string{}, err
		}
		rv = append(rv, VmSwitch.Value)
	}

	return rv, nil
}

func SwitchNameToId(s *string, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
	rv := ""

	switchIds, err := getSwitchIds(c, ctx)
	if err != nil {
		return "", err
	}
	found := false

	for _, switchId := range switchIds {
		res2, err := c.GetSwitchInfo(ctx, &cirrina.SwitchId{Value: switchId})
		if err != nil {
			return "", err
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

func SwitchIdToName(s string, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
	res, err := c.GetSwitchInfo(ctx, &cirrina.SwitchId{Value: s})
	if err != nil {
		return "", err
	}
	return *res.Name, nil
}

func GetSwitches(c cirrina.VMInfoClient, ctx context.Context) ([]string, error) {
	var rv []string
	res, err := c.GetSwitches(ctx, &cirrina.SwitchesQuery{})

	if err != nil {
		return []string{}, err
	}

	for {
		SwitchId, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []string{}, err
		}
		rv = append(rv, SwitchId.Value)
	}
	return rv, nil
}

func AddSwitch(namePtr *string, c cirrina.VMInfoClient, ctx context.Context, descrPtr *string, switchTypePtr *string) (switchId string, err error) {
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

func SetSwitchUplink(c cirrina.VMInfoClient, ctx context.Context, switchIdPtr *string, uplinkNamePtr *string) error {
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
		return err
	}
	return nil
}

func RemoveSwitch(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (err error) {
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

func UpdateSwitch(idPtr *string, c cirrina.VMInfoClient, ctx context.Context, siu *cirrina.SwitchInfoUpdate) (err error) {
	if *idPtr == "" {
		return errors.New("id not specified")
	}
	reqId, err := c.SetSwitchInfo(ctx, siu)
	if err != nil {
		return err
	}
	if !reqId.Success {
		return errors.New("failed to update switch")
	}
	return nil
}

func GetSwitch(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (switchInfo *cirrina.SwitchInfo, err error) {
	if *idPtr == "" {
		return &cirrina.SwitchInfo{}, errors.New("id not specified")
	}
	res, err := c.GetSwitchInfo(ctx, &cirrina.SwitchId{Value: *idPtr})
	if err != nil {
		return &cirrina.SwitchInfo{}, err
	}
	return res, nil
}

func SetVmNicSwitch(c cirrina.VMInfoClient, ctx context.Context, vmNicId string, switchId string) (bool, error) {
	var vmnicid cirrina.VmNicId
	var vmswitchid cirrina.SwitchId

	if vmNicId == "" {
		return false, errors.New("nic id not specified")
	}

	vmnicid.Value = vmNicId
	vmswitchid.Value = switchId

	nicSwitchSettings := cirrina.SetVmNicSwitchReq{
		Vmnicid:  &vmnicid,
		Switchid: &vmswitchid,
	}
	r, err := c.SetVmNicSwitch(ctx, &nicSwitchSettings)
	if err != nil {
		return false, err
	}
	if r.Success {
		return true, nil
	} else {
		return false, nil
	}
}
