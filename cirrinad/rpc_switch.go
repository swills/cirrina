package main

import (
	"cirrina/cirrina"
	_switch "cirrina/cirrinad/switch"
	"errors"
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
