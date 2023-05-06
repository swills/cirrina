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
