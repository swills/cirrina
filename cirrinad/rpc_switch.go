package main

import (
	"cirrina/cirrina"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/exp/slog"
	"strconv"
	"strings"
)
import "context"

func (s *server) AddSwitch(_ context.Context, i *cirrina.SwitchInfo) (*cirrina.SwitchId, error) {
	var switchType string
	defaultSwitchType := cirrina.SwitchType_IF
	defaultSwitchDescription := ""
	// TODO handle uplink (i.Uplink)

	if i.Name == nil || !util.ValidSwitchName(*i.Name) {
		return &cirrina.SwitchId{}, errors.New("invalid name")
	}

	if i.Description == nil {
		i.Description = &defaultSwitchDescription
	}

	if i.SwitchType == nil {
		i.SwitchType = &defaultSwitchType
	}

	if *i.SwitchType == cirrina.SwitchType_IF {
		switchType = "IF"
		// TODO check that same uplink isn't used for another switch of same type
		if !strings.HasPrefix(*i.Name, "bridge") {
			slog.Error("invalid name", "name", *i.Name)
			return &cirrina.SwitchId{Value: ""}, errors.New("invalid name")
		}

		bridgeNumStr := strings.TrimPrefix(*i.Name, "bridge")
		bridgeNum, err := strconv.Atoi(bridgeNumStr)
		if err != nil {
			slog.Error("invalid bridge name", "name", *i.Name)
			return &cirrina.SwitchId{Value: ""}, errors.New("invalid bridge name")

		}
		bridgeNumFormattedString := strconv.FormatInt(int64(bridgeNum), 10)
		// Check for silly things like "0123"
		if bridgeNumStr != bridgeNumFormattedString {
			slog.Error("invalid name", "name", *i.Name)
			return &cirrina.SwitchId{Value: ""}, errors.New("invalid name")
		}

	} else if *i.SwitchType == cirrina.SwitchType_NG {
		switchType = "NG"
		// TODO check that same uplink isn't used for another switch of same type
		if !strings.HasPrefix(*i.Name, "bnet") {
			slog.Error("invalid bridge name", "name", *i.Name)
			return &cirrina.SwitchId{Value: ""}, errors.New("invalid name")
		}

		bridgeNumStr := strings.TrimPrefix(*i.Name, "bnet")
		bridgeNum, err := strconv.Atoi(bridgeNumStr)
		if err != nil {
			slog.Error("invalid bridge name", "name", *i.Name)
			return &cirrina.SwitchId{Value: ""}, errors.New("invalid bridge name")

		}
		bridgeNumFormattedString := strconv.FormatInt(int64(bridgeNum), 10)
		// Check for silly things like "0123"
		if bridgeNumStr != bridgeNumFormattedString {
			slog.Error("invalid name", "name", *i.Name)
			return &cirrina.SwitchId{Value: ""}, errors.New("invalid name")
		}
	} else {
		return &cirrina.SwitchId{}, errors.New("invalid type")
	}

	switchInst, err := _switch.Create(*i.Name, *i.Description, switchType)
	if err != nil {
		return &cirrina.SwitchId{}, err
	}
	if switchInst != nil && switchInst.ID != "" {
		if switchInst.Type == "IF" {
			slog.Debug("creating if bridge", "name", switchInst.Name)
			err := _switch.BuildIfBridge(switchInst)
			if err != nil {
				slog.Error("error creating if bridge", "err", err)
				// already created in db, so ignore system state and proceed on...
				return &cirrina.SwitchId{Value: switchInst.ID}, nil
			}
		} else if switchInst.Type == "NG" {
			slog.Debug("creating ng bridge", "name", switchInst.Name)
			err := _switch.BuildNgBridge(switchInst)
			if err != nil {
				slog.Error("error creating ng bridge", "err", err)
				// already created in db, so ignore system state and proceed on...
				return &cirrina.SwitchId{Value: switchInst.ID}, nil
			}
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

	switchUuid, err := uuid.Parse(v.Value)
	if err != nil {
		return &pvmswitchinfo, errors.New("id not specified or invalid")
	}

	vmSwitch, err := _switch.GetById(switchUuid.String())
	if err != nil {
		slog.Error("error getting switch info", "switch", v.Value, "err", err)
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

	switchUuid, err := uuid.Parse(si.Value)
	if err != nil {
		return &re, errors.New("id not specified or invalid")
	}

	switchInst, err := _switch.GetById(switchUuid.String())
	if err != nil {
		return &re, errors.New("not found")
	}

	err2 := _switch.CheckSwitchInUse(si.Value)
	if err2 != nil {
		slog.Debug("attempted to delete switch which is in use",
			"switch", si.Value,
			"switch_name", switchInst.Name,
		)
		return &re, errors.New("switch in use")
	}

	if switchInst.Type == "IF" {
		err := _switch.DestroyIfBridge(switchInst.Name, true)
		if err != nil {
			return &re, err
		}
	} else if switchInst.Type == "NG" {
		err := _switch.DestroyNgBridge(switchInst.Name)
		if err != nil {
			slog.Error("switch removal failure")
			return &re, err
		}
	} else {
		return &re, errors.New("invalid switch type")
	}
	slog.Debug("RemoveSwitch", "switchid", si.Value)
	err = _switch.Delete(si.Value)
	if err != nil {
		return &re, err
	}
	re.Success = true
	return &re, nil
}

func (s *server) SetSwitchUplink(_ context.Context, su *cirrina.SwitchUplinkReq) (*cirrina.ReqBool, error) {
	var r cirrina.ReqBool
	r.Success = false

	if su.Switchid == nil {
		return &r, errors.New("id not specified or invalid")
	}

	switchUuid, err := uuid.Parse(su.Switchid.Value)
	if err != nil {
		return &r, errors.New("id not specified or invalid")
	}

	if su.Uplink == nil {
		return &r, errors.New("uplink not specified")
	}

	uplink := *su.Uplink
	slog.Debug("SetSwitchUplink", "switch", su.Switchid.Value, "uplink", uplink)
	switchInst, err := _switch.GetById(switchUuid.String())
	if err != nil {
		return &r, err
	}
	if uplink == "" {
		if switchInst.Uplink != "" {
			slog.Debug("SetSwitchUplink", "msg", "unsetting switch uplink", "switchInst", switchInst)
			if err = switchInst.UnsetUplink(); err != nil {
				return &r, err
			}
		}
	} else {
		switchList := _switch.GetAll()
		for _, sw := range switchList {
			if sw.ID != switchInst.ID && sw.Type == switchInst.Type && sw.Uplink == uplink {
				slog.Error("SetSwitchUplink uplink already in use by another switch",
					"uplink", uplink,
					"name", sw.Name,
				)
				errorString := fmt.Sprintf("uplink already in use by %v", sw.Name)
				return &r, errors.New(errorString)
			}
		}
		if switchInst.Uplink != uplink {
			slog.Debug("SetSwitchUplink", "msg", "unsetting switch uplink", "switchInst", switchInst)
			// ignore error here because it may not be set so removing it can fail
			_ = switchInst.UnsetUplink()
			slog.Debug("SetSwitchUplink", "msg", "setting switch uplink", "switchInst", switchInst)
			if err = switchInst.SetUplink(uplink); err != nil {
				return &r, err
			}
		} else {
			slog.Debug("SetSwitchUplink", "msg", "re-setting switch uplink", "switchInst", switchInst)
			if err = switchInst.UnsetUplink(); err != nil {
				return &r, err
			}
			if err = switchInst.SetUplink(uplink); err != nil {
				return &r, err
			}
		}
	}
	r.Success = true
	return &r, nil
}

func (s *server) SetSwitchInfo(c context.Context, siu *cirrina.SwitchInfoUpdate) (*cirrina.ReqBool, error) {
	var re cirrina.ReqBool
	re.Success = false

	if siu.Id == "" {
		return &re, errors.New("id not specified or invalid")
	}

	switchUuid, err := uuid.Parse(siu.Id)
	if err != nil {
		return &re, errors.New("id not specified or invalid")
	}

	switchInst, err := _switch.GetById(switchUuid.String())
	if err != nil {
		return &re, err
	}

	if siu.Description != nil {
		switchInst.Description = *siu.Description
	}

	err = switchInst.Save()
	if err != nil {
		return &re, errors.New("failed to update switch")
	}
	re.Success = true

	return &re, nil
}
