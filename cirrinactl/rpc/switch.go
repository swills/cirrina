package rpc

import (
	"errors"
	"fmt"
	"io"

	"cirrina/cirrina"
)

func getSwitchIDs() ([]string, error) {
	var err error
	var switchIDs []string
	var res cirrina.VMInfo_GetSwitchesClient
	res, err = serverClient.GetSwitches(defaultServerContext, &cirrina.SwitchesQuery{})
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
		switchIDs = append(switchIDs, VMSwitch.Value)
	}

	return switchIDs, nil
}

func SwitchNameToID(thisSwitchID string) (string, error) {
	var err error

	switchName := ""

	var switchIDs []string
	switchIDs, err = getSwitchIDs()
	if err != nil {
		return "", err
	}
	found := false

	for _, switchID := range switchIDs {
		var switchInfo *cirrina.SwitchInfo
		switchInfo, err = serverClient.GetSwitchInfo(defaultServerContext, &cirrina.SwitchId{Value: switchID})
		if err != nil {
			return "", fmt.Errorf("unable to get switch id: %w", err)
		}
		if *switchInfo.Name == thisSwitchID {
			if found {
				return "", errSwitchDuplicate
			}
			found = true
			switchName = switchID
		}
	}

	return switchName, nil
}

func SwitchIDToName(s string) (string, error) {
	var err error

	var res *cirrina.SwitchInfo
	res, err = serverClient.GetSwitchInfo(defaultServerContext, &cirrina.SwitchId{Value: s})
	if err != nil {
		return "", fmt.Errorf("unable to get switch name: %w", err)
	}
	if res.Name != nil {
		return *res.Name, nil
	}

	return "", nil
}

func GetSwitches() ([]string, error) {
	var err error
	var res cirrina.VMInfo_GetSwitchesClient
	res, err = serverClient.GetSwitches(defaultServerContext, &cirrina.SwitchesQuery{})
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
		switchIDs = append(switchIDs, SwitchID.Value)
	}

	return switchIDs, nil
}

func AddSwitch(name string, descrPtr *string, switchTypePtr *string, switchUplinkName *string) (string, error) {
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
	res, err = serverClient.AddSwitch(defaultServerContext, &thisSwitchInfo)
	if err != nil {
		return "", fmt.Errorf("unable to add switch: %w", err)
	}

	return res.Value, nil
}

func SetSwitchUplink(switchID string, uplinkNamePtr *string) error {
	var err error

	if switchID == "" {
		return errSwitchEmptyID
	}

	req := &cirrina.SwitchUplinkReq{}
	si := &cirrina.SwitchId{}
	si.Value = switchID
	req.Switchid = si
	req.Uplink = uplinkNamePtr

	_, err = serverClient.SetSwitchUplink(defaultServerContext, req)
	if err != nil {
		return fmt.Errorf("unable to set switch uplink: %w", err)
	}

	return nil
}

func RemoveSwitch(switchID string) error {
	var err error

	if switchID == "" {
		return errSwitchEmptyID
	}
	var reqID *cirrina.ReqBool
	reqID, err = serverClient.RemoveSwitch(defaultServerContext, &cirrina.SwitchId{Value: switchID})
	if err != nil {
		return fmt.Errorf("unable to remove switch: %w", err)
	}
	if !reqID.Success {
		return errReqFailed
	}

	return nil
}

func UpdateSwitch(switchID string, description *string) error {
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
	reqStat, err = serverClient.SetSwitchInfo(defaultServerContext, &siu)
	if err != nil {
		return fmt.Errorf("unable to update switch: %w", err)
	}
	if !reqStat.Success {
		return errReqFailed
	}

	return nil
}

func GetSwitch(switchID string) (SwitchInfo, error) {
	var err error

	if switchID == "" {
		return SwitchInfo{}, errSwitchEmptyID
	}

	var res *cirrina.SwitchInfo
	res, err = serverClient.GetSwitchInfo(defaultServerContext, &cirrina.SwitchId{Value: switchID})
	if err != nil {
		return SwitchInfo{}, fmt.Errorf("unable to get switch info: %w", err)
	}

	switchType := "unknown"
	if *res.SwitchType == cirrina.SwitchType_IF {
		switchType = "bridge"
	} else if *res.SwitchType == cirrina.SwitchType_NG {
		switchType = "netgraph"
	}

	return SwitchInfo{
		Name:       *res.Name,
		SwitchType: switchType,
		Uplink:     *res.Uplink,
		Descr:      *res.Description,
	}, nil
}

func SetVMNicSwitch(vmNicIDStr string, switchID string) error {
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
	var r *cirrina.ReqBool
	r, err = serverClient.SetVMNicSwitch(defaultServerContext, &nicSwitchSettings)
	if err != nil {
		return fmt.Errorf("unable to set nic switch: %w", err)
	}
	if !r.Success {
		return errReqFailed
	}

	return nil
}
