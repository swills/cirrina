package rpc

import (
	"cirrina/cirrina"
	"context"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"io"
)

func getSwitchIds(c cirrina.VMInfoClient, ctx context.Context) ([]string, error) {
	var err error
	var rv []string
	var res cirrina.VMInfo_GetSwitchesClient
	res, err = c.GetSwitches(ctx, &cirrina.SwitchesQuery{})
	if err != nil {
		return []string{}, errors.New(status.Convert(err).Message())
	}

	for {
		var VmSwitch *cirrina.SwitchId
		VmSwitch, err = res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []string{}, errors.New(status.Convert(err).Message())
		}
		rv = append(rv, VmSwitch.Value)
	}

	return rv, nil
}

func SwitchNameToId(s string) (string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	rv := ""

	var switchIds []string
	switchIds, err = getSwitchIds(c, ctx)
	if err != nil {
		return "", err
	}
	found := false

	for _, switchId := range switchIds {
		var switchInfo *cirrina.SwitchInfo
		switchInfo, err = c.GetSwitchInfo(ctx, &cirrina.SwitchId{Value: switchId})
		if err != nil {
			return "", errors.New(status.Convert(err).Message())
		}
		if *switchInfo.Name == s {
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

func SwitchIdToName(s string) (string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var res *cirrina.SwitchInfo
	res, err = c.GetSwitchInfo(ctx, &cirrina.SwitchId{Value: s})
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	return *res.Name, nil
}

func GetSwitches() ([]string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return []string{}, err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var res cirrina.VMInfo_GetSwitchesClient
	res, err = c.GetSwitches(ctx, &cirrina.SwitchesQuery{})

	if err != nil {
		return []string{}, errors.New(status.Convert(err).Message())
	}

	var rv []string
	for {
		var SwitchId *cirrina.SwitchId
		SwitchId, err = res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []string{}, errors.New(status.Convert(err).Message())
		}
		rv = append(rv, SwitchId.Value)
	}
	return rv, nil
}

func AddSwitch(name string, descrPtr *string, switchTypePtr *string) (string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var thisSwitchType cirrina.SwitchType
	if name == "" {
		return "", errors.New("switch name not specified")
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
	thisSwitchInfo.Name = &name
	thisSwitchInfo.Description = descrPtr
	thisSwitchInfo.SwitchType = &thisSwitchType
	var res *cirrina.SwitchId
	res, err = c.AddSwitch(ctx, &thisSwitchInfo)
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	return res.Value, nil
}

func SetSwitchUplink(switchId string, uplinkNamePtr *string) error {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	if switchId == "" {
		return errors.New("switch id not specified")
	}

	req := &cirrina.SwitchUplinkReq{}
	si := &cirrina.SwitchId{}
	si.Value = switchId
	req.Switchid = si
	req.Uplink = uplinkNamePtr

	_, err = c.SetSwitchUplink(ctx, req)
	if err != nil {
		return errors.New(status.Convert(err).Message())
	}
	return nil
}

func RemoveSwitch(id string) error {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	if id == "" {
		return errors.New("id not specified")
	}
	var reqId *cirrina.ReqBool
	reqId, err = c.RemoveSwitch(ctx, &cirrina.SwitchId{Value: id})
	if err != nil {
		return errors.New(status.Convert(err).Message())
	}
	if !reqId.Success {
		return errors.New("failed to delete switch")
	}
	return nil
}

func UpdateSwitch(id string, description *string) error {
	if id == "" {
		return errors.New("id not specified")
	}
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	siu := cirrina.SwitchInfoUpdate{
		Id: id,
	}

	if description != nil {
		siu.Description = description
	}
	var reqId *cirrina.ReqBool
	reqId, err = c.SetSwitchInfo(ctx, &siu)
	if err != nil {
		return errors.New(status.Convert(err).Message())
	}
	if !reqId.Success {
		return errors.New("failed to update switch")
	}
	return nil
}

func GetSwitch(id string) (SwitchInfo, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return SwitchInfo{}, err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	if id == "" {
		return SwitchInfo{}, errors.New("id not specified")
	}

	var res *cirrina.SwitchInfo
	res, err = c.GetSwitchInfo(ctx, &cirrina.SwitchId{Value: id})
	if err != nil {
		return SwitchInfo{}, errors.New(status.Convert(err).Message())
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

func SetVmNicSwitch(vmNicIdStr string, switchId string) error {
	if vmNicIdStr == "" {
		return errors.New("nic id not specified")
	}
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var vmNicId cirrina.VmNicId
	vmNicId.Value = vmNicIdStr
	var vmSwitchId cirrina.SwitchId
	vmSwitchId.Value = switchId

	nicSwitchSettings := cirrina.SetVmNicSwitchReq{
		Vmnicid:  &vmNicId,
		Switchid: &vmSwitchId,
	}
	var r *cirrina.ReqBool
	r, err = c.SetVmNicSwitch(ctx, &nicSwitchSettings)
	if err != nil {
		return errors.New(status.Convert(err).Message())
	}
	if !r.Success {
		return errors.New("failed to add nic to switch")
	}
	return nil
}
