package main

import (
	"cirrina/cirrina"
	_switch "cirrina/cirrinad/switch"
	"errors"
	"golang.org/x/exp/slog"
)
import "context"

func (s *server) AddSwitch(_ context.Context, i *cirrina.SwitchInfo) (*cirrina.SwitchId, error) {
	var switchType string

	if *i.SwitchType == cirrina.SwitchType_IF {
		switchType = "IF"
	} else if *i.SwitchType == cirrina.SwitchType_NG {
		switchType = "NG"
	} else {
		return &cirrina.SwitchId{}, errors.New("invalid switch type")
	}

	switchInst, err := _switch.Create(*i.Name, *i.Description, switchType)
	if err != nil {
		return &cirrina.SwitchId{}, err
	}
	if switchInst != nil && switchInst.ID != "" {
		slog.Debug("creating if bridge", "name", switchInst.Name)
		err := _switch.BuildIfBridge(switchInst)
		if err != nil {
			slog.Error("error creating if bridge", "err", err)
			// already created in db, so ignore system state and proceed on...
			return &cirrina.SwitchId{Value: switchInst.ID}, nil
		}
		return &cirrina.SwitchId{Value: switchInst.ID}, nil
	} else {
		return &cirrina.SwitchId{}, errors.New("unknown error creating switch")
	}
}

func (s *server) GetSwitches(_ *cirrina.SwitchesQuery, stream cirrina.VMInfo_GetSwitchesServer) error {
	var switches []*_switch.Switch
	var pSwitchId cirrina.SwitchId

	switches = _switch.GetAll()
	for e := range switches {
		pSwitchId.Value = switches[e].ID
		err := stream.Send(&pSwitchId)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *server) GetSwitchInfo(_ context.Context, v *cirrina.SwitchId) (*cirrina.SwitchInfo, error) {
	var pvmswitchinfo cirrina.SwitchInfo

	vmSwitch, err := _switch.GetById(v.Value)
	if err != nil {
		slog.Debug("error getting switch info", "switch", v.Value, "err", err)
		return &pvmswitchinfo, err
	}

	pvmswitchinfo.Name = &vmSwitch.Name
	pvmswitchinfo.Description = &vmSwitch.Description
	pvmswitchinfo.Uplink = &vmSwitch.Uplink

	SwitchTypeIf := cirrina.SwitchType_IF
	SwitchTypeNg := cirrina.SwitchType_NG

	if vmSwitch.Type == "IF" {
		pvmswitchinfo.SwitchType = &SwitchTypeIf
	} else if vmSwitch.Type == "NG" {
		pvmswitchinfo.SwitchType = &SwitchTypeNg
	} else {
		slog.Error("GetSwitchInfo bad switch type", "switchid", vmSwitch.ID, "type", vmSwitch.Type)
	}
	return &pvmswitchinfo, nil
}

func (s *server) RemoveSwitch(_ context.Context, si *cirrina.SwitchId) (*cirrina.ReqBool, error) {
	var re cirrina.ReqBool
	re.Success = false

	slog.Debug("RemoveSwitch", "switchid", si.Value)
	err := _switch.Delete(si.Value)
	if err != nil {
		return &re, err
	}
	re.Success = true
	return &re, nil
}

func (s *server) SetSwitchUplink(_ context.Context, su *cirrina.SwitchUplinkReq) (*cirrina.ReqBool, error) {
	var r cirrina.ReqBool
	r.Success = false
	thisSwitch := su.Switchid.Value
	uplink := *su.Uplink
	slog.Debug("SetSwitchUplink", "switch", thisSwitch, "uplink", uplink)
	switchInst, err := _switch.GetById(thisSwitch)
	if err != nil {
		return &r, err
	}
	if uplink == "" {
		if err = switchInst.UnsetUplink(); err != nil {
			return &r, err
		}

	} else {
		if err = switchInst.SetUplink(uplink); err != nil {
			return &r, err
		}
	}
	r.Success = true
	return &r, nil
}
