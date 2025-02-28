package rpc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"cirrina/cirrina"
)

func getSwitchIDs(ctx context.Context) ([]string, error) {
	var err error

	var switchIDs []string

	var res cirrina.VMInfo_GetSwitchesClient

	res, err = serverClient.GetSwitches(ctx, &cirrina.SwitchesQuery{})
	if err != nil {
		return []string{}, fmt.Errorf("unable to get switch IDs: %w", err)
	}

	for {
		var VMSwitch *cirrina.SwitchId

		VMSwitch, err = res.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return []string{}, fmt.Errorf("unable to get switch IDs: %w", err)
		}

		switchIDs = append(switchIDs, VMSwitch.GetValue())
	}

	return switchIDs, nil
}

func SwitchNameToID(ctx context.Context, thisSwitchID string) (string, error) {
	var err error

	switchName := ""

	var switchIDs []string

	switchIDs, err = getSwitchIDs(ctx)
	if err != nil {
		return "", err
	}

	found := false

	for _, switchID := range switchIDs {
		var switchInfo *cirrina.SwitchInfo

		switchInfo, err = serverClient.GetSwitchInfo(ctx, &cirrina.SwitchId{Value: switchID})
		if err != nil {
			return "", fmt.Errorf("unable to get switch id: %w", err)
		}

		if switchInfo.GetName() == thisSwitchID {
			if found {
				return "", errSwitchDuplicate
			}

			found = true
			switchName = switchID
		}
	}

	return switchName, nil
}

func SwitchIDToName(ctx context.Context, switchID string) (string, error) {
	var err error

	var res *cirrina.SwitchInfo

	res, err = serverClient.GetSwitchInfo(ctx, &cirrina.SwitchId{Value: switchID})
	if err != nil {
		return "", fmt.Errorf("unable to get switch name: %w", err)
	}

	return res.GetName(), nil
}

func GetSwitches(ctx context.Context) ([]string, error) {
	var err error

	var res cirrina.VMInfo_GetSwitchesClient

	res, err = serverClient.GetSwitches(ctx, &cirrina.SwitchesQuery{})
	if err != nil {
		return []string{}, fmt.Errorf("unable to get switches: %w", err)
	}

	var switchIDs []string

	for {
		var SwitchID *cirrina.SwitchId

		SwitchID, err = res.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return []string{}, fmt.Errorf("unable to get switches: %w", err)
		}

		switchIDs = append(switchIDs, SwitchID.GetValue())
	}

	return switchIDs, nil
}

func AddSwitch(ctx context.Context, name string, descrPtr *string, switchTypePtr *string, switchUplinkName *string) (string, error) { //nolint:lll
	var err error

	var thisSwitchType cirrina.SwitchType

	if name == "" {
		return "", errSwitchEmptyName
	}

	if *switchTypePtr == "" {
		return "", errSwitchTypeEmpty
	}

	switch {
	case *switchTypePtr == "IF" || *switchTypePtr == "bridge":
		thisSwitchType = cirrina.SwitchType_IF
	case *switchTypePtr == "NG" || *switchTypePtr == "netgraph":
		thisSwitchType = cirrina.SwitchType_NG
	default:
		return "", errSwitchTypeInvalid
	}

	var thisSwitchInfo cirrina.SwitchInfo
	thisSwitchInfo.Name = &name
	thisSwitchInfo.Description = descrPtr
	thisSwitchInfo.SwitchType = &thisSwitchType
	thisSwitchInfo.Uplink = switchUplinkName

	var res *cirrina.SwitchId

	res, err = serverClient.AddSwitch(ctx, &thisSwitchInfo)
	if err != nil {
		return "", fmt.Errorf("unable to add switch: %w", err)
	}

	return res.GetValue(), nil
}

func SetSwitchUplink(ctx context.Context, switchID string, uplinkNamePtr *string) error {
	var err error

	if switchID == "" {
		return errSwitchEmptyID
	}

	req := &cirrina.SwitchUplinkReq{}
	si := &cirrina.SwitchId{}
	si.Value = switchID
	req.Switchid = si
	req.Uplink = uplinkNamePtr

	_, err = serverClient.SetSwitchUplink(ctx, req)
	if err != nil {
		return fmt.Errorf("unable to set switch uplink: %w", err)
	}

	return nil
}

func DeleteSwitch(ctx context.Context, switchID string) error {
	var err error

	if switchID == "" {
		return errSwitchEmptyID
	}

	var reqID *cirrina.ReqBool

	reqID, err = serverClient.RemoveSwitch(ctx, &cirrina.SwitchId{Value: switchID})
	if err != nil {
		return fmt.Errorf("unable to remove switch: %w", err)
	}

	if !reqID.GetSuccess() {
		return errReqFailed
	}

	return nil
}

func UpdateSwitch(ctx context.Context, switchID string, description *string) error {
	if switchID == "" {
		return errSwitchEmptyID
	}

	var err error

	siu := cirrina.SwitchInfoUpdate{
		Id: switchID,
	}

	if description != nil {
		siu.Description = description
	}

	var reqStat *cirrina.ReqBool

	reqStat, err = serverClient.SetSwitchInfo(ctx, &siu)
	if err != nil {
		return fmt.Errorf("unable to update switch: %w", err)
	}

	if !reqStat.GetSuccess() {
		return errReqFailed
	}

	return nil
}

func GetSwitch(ctx context.Context, switchID string) (SwitchInfo, error) {
	var err error

	if switchID == "" {
		return SwitchInfo{}, errSwitchEmptyID
	}

	var res *cirrina.SwitchInfo

	res, err = serverClient.GetSwitchInfo(ctx, &cirrina.SwitchId{Value: switchID})
	if err != nil {
		return SwitchInfo{}, fmt.Errorf("unable to get switch info: %w", err)
	}

	switchType := "unknown"
	if res.GetSwitchType() == cirrina.SwitchType_IF {
		switchType = "bridge"
	} else if res.GetSwitchType() == cirrina.SwitchType_NG {
		switchType = "netgraph"
	}

	return SwitchInfo{
		Name:       res.GetName(),
		SwitchType: switchType,
		Uplink:     res.GetUplink(),
		Descr:      res.GetDescription(),
	}, nil
}

func SetVMNicSwitch(ctx context.Context, vmNicIDStr string, switchID string) error {
	if vmNicIDStr == "" {
		return errNicEmptyID
	}

	var err error

	var vmNicID cirrina.VmNicId
	vmNicID.Value = vmNicIDStr

	var vmSwitchID cirrina.SwitchId
	vmSwitchID.Value = switchID

	nicSwitchSettings := cirrina.SetVmNicSwitchReq{
		Vmnicid:  &vmNicID,
		Switchid: &vmSwitchID,
	}

	var reqBool *cirrina.ReqBool

	reqBool, err = serverClient.SetVMNicSwitch(ctx, &nicSwitchSettings)
	if err != nil {
		return fmt.Errorf("unable to set nic switch: %w", err)
	}

	if !reqBool.GetSuccess() {
		return errReqFailed
	}

	return nil
}
