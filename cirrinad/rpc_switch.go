package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"cirrina/cirrina"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
)

func (s *server) AddSwitch(_ context.Context, i *cirrina.SwitchInfo) (*cirrina.SwitchId, error) {
	defaultSwitchType := cirrina.SwitchType_IF
	defaultSwitchDescription := ""

	if i.Name == nil || !util.ValidSwitchName(*i.Name) {
		return &cirrina.SwitchId{}, errors.New("invalid name")
	}

	if i.Description == nil {
		i.Description = &defaultSwitchDescription
	}

	if i.SwitchType == nil {
		i.SwitchType = &defaultSwitchType
	}

	err := validateNewSwitch(i)
	if err != nil {
		return &cirrina.SwitchId{}, err
	}
	switchType, err := mapSwitchTypeTypeToDBString(*i.SwitchType)
	if err != nil {
		return &cirrina.SwitchId{}, err
	}

	switchInst, err := _switch.Create(*i.Name, *i.Description, switchType, *i.Uplink)
	if err != nil {
		return &cirrina.SwitchId{}, err
	}

	return bringUpNewSwitch(switchInst)
}

func (s *server) GetSwitches(_ *cirrina.SwitchesQuery, stream cirrina.VMInfo_GetSwitchesServer) error {
	var switches []*_switch.Switch
	var pSwitchID cirrina.SwitchId

	switches = _switch.GetAll()
	for e := range switches {
		pSwitchID.Value = switches[e].ID
		err := stream.Send(&pSwitchID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *server) GetSwitchInfo(_ context.Context, v *cirrina.SwitchId) (*cirrina.SwitchInfo, error) {
	var switchInfo cirrina.SwitchInfo

	switchUUID, err := uuid.Parse(v.Value)
	if err != nil {
		return &cirrina.SwitchInfo{}, errors.New("id not specified or invalid")
	}

	vmSwitch, err := _switch.GetByID(switchUUID.String())
	if err != nil {
		slog.Error("error getting switch info", "switch", v.Value, "err", err)

		return &cirrina.SwitchInfo{}, err
	}

	switchInfo.Name = &vmSwitch.Name
	switchInfo.Description = &vmSwitch.Description
	switchInfo.Uplink = &vmSwitch.Uplink
	switchInfo.SwitchType, err = mapSwitchTypeDBStringToType(vmSwitch.Type)
	if err != nil {
		return &cirrina.SwitchInfo{}, err
	}

	return &switchInfo, nil
}

func (s *server) RemoveSwitch(_ context.Context, si *cirrina.SwitchId) (*cirrina.ReqBool, error) {
	var re cirrina.ReqBool
	re.Success = false

	switchUUID, err := uuid.Parse(si.Value)
	if err != nil {
		return &re, errors.New("id not specified or invalid")
	}

	switchInst, err := _switch.GetByID(switchUUID.String())
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

	switch switchInst.Type {
	case "IF":
		err := _switch.DestroyIfBridge(switchInst.Name, true)
		if err != nil {
			return &re, err
		}
	case "NG":
		err := _switch.DestroyNgBridge(switchInst.Name)
		if err != nil {
			slog.Error("switch removal failure")

			return &re, err
		}
	default:
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

	switchUUID, err := uuid.Parse(su.Switchid.Value)
	if err != nil {
		return &r, errors.New("id not specified or invalid")
	}

	if su.Uplink == nil {
		return &r, errors.New("uplink not specified")
	}

	uplink := *su.Uplink
	slog.Debug("SetSwitchUplink", "switch", su.Switchid.Value, "uplink", uplink)
	switchInst, err := _switch.GetByID(switchUUID.String())
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
		r.Success = true

		return &r, nil
	}

	if uplinkInUse(switchInst.ID, uplink, switchInst.Type) {
		slog.Error("SetSwitchUplink uplink already in use by another switch",
			"uplink", uplink,
			"name", switchInst.Name,
		)
		errorString := fmt.Sprintf("uplink already in use by %v", switchInst.Name)

		return &r, errors.New(errorString)
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
	r.Success = true

	return &r, nil
}

func (s *server) SetSwitchInfo(_ context.Context, siu *cirrina.SwitchInfoUpdate) (*cirrina.ReqBool, error) {
	var re cirrina.ReqBool
	re.Success = false

	if siu.Id == "" {
		return &re, errors.New("id not specified or invalid")
	}

	switchUUID, err := uuid.Parse(siu.Id)
	if err != nil {
		return &re, errors.New("id not specified or invalid")
	}

	switchInst, err := _switch.GetByID(switchUUID.String())
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

// uplinkInUse check if the uplink is in use by a switch other than this one
func uplinkInUse(id string, uplink string, t string) bool {
	switchList := _switch.GetAll()
	for _, sw := range switchList {
		if sw.ID != id && sw.Type == t && sw.Uplink == uplink {
			return true
		}
	}

	return false
}

func validateNewSwitch(i *cirrina.SwitchInfo) error {
	if i == nil || i.SwitchType == nil {
		return errors.New("invalid type")
	}
	switch *i.SwitchType {
	case cirrina.SwitchType_IF:
		return validateIfSwitch(i)
	case cirrina.SwitchType_NG:
		return validateNgSwitch(i)
	default:
		return errors.New("invalid type")
	}
}

func validateNgSwitch(i *cirrina.SwitchInfo) error {
	// it can't be a member of another bridge of same type already
	if i.Uplink != nil && *i.Uplink != "" {
		alreadyUsed, err := _switch.MemberUsedByNgBridge(*i.Uplink)
		if err != nil {
			slog.Error("error checking if member already used", "err", err)

			return errors.New("error checking if switch uplink in use by another bridge")
		}
		if alreadyUsed {
			return errors.New("uplink already in use by another bridge of same type (NG)")
		}
	}
	if !strings.HasPrefix(*i.Name, "bnet") {
		slog.Error("invalid bridge name", "name", *i.Name)

		return errors.New("invalid name")
	}

	bridgeNumStr := strings.TrimPrefix(*i.Name, "bnet")
	bridgeNum, err := strconv.Atoi(bridgeNumStr)
	if err != nil {
		slog.Error("invalid bridge name", "name", *i.Name)

		return errors.New("invalid bridge name")
	}
	bridgeNumFormattedString := strconv.FormatInt(int64(bridgeNum), 10)
	// Check for silly things like "0123"
	if bridgeNumStr != bridgeNumFormattedString {
		slog.Error("invalid name", "name", *i.Name)

		return errors.New("invalid name")
	}

	return nil
}

func validateIfSwitch(i *cirrina.SwitchInfo) error {
	// it can't be a member of another bridge of same type already
	if i.Uplink != nil && *i.Uplink != "" {
		alreadyUsed, err := _switch.MemberUsedByIfBridge(*i.Uplink)
		if err != nil {
			slog.Error("error checking if member already used", "err", err)

			return errors.New("error checking if switch uplink in use by another bridge")
		}
		if alreadyUsed {
			return errors.New("uplink already in use by another bridge of same type (IF)")
		}
	}
	if !strings.HasPrefix(*i.Name, "bridge") {
		slog.Error("invalid name", "name", *i.Name)

		return errors.New("invalid name")
	}

	bridgeNumStr := strings.TrimPrefix(*i.Name, "bridge")
	bridgeNum, err := strconv.Atoi(bridgeNumStr)
	if err != nil {
		slog.Error("invalid bridge name", "name", *i.Name)

		return errors.New("invalid bridge name")
	}
	bridgeNumFormattedString := strconv.FormatInt(int64(bridgeNum), 10)
	// Check for silly things like "0123"
	if bridgeNumStr != bridgeNumFormattedString {
		slog.Error("invalid name", "name", *i.Name)

		return errors.New("invalid name")
	}

	return nil
}

func bringUpNewSwitch(switchInst *_switch.Switch) (*cirrina.SwitchId, error) {
	if switchInst == nil || switchInst.ID != "" {
		return &cirrina.SwitchId{}, errors.New("unknown error creating switch")
	}
	switch switchInst.Type {
	case "IF":
		slog.Debug("creating if bridge", "name", switchInst.Name)
		err := _switch.BuildIfBridge(switchInst)
		if err != nil {
			slog.Error("error creating if bridge", "err", err)
			// already created in db, so ignore system state and proceed on...
			return &cirrina.SwitchId{Value: switchInst.ID}, nil
		}
	case "NG":
		slog.Debug("creating ng bridge", "name", switchInst.Name)
		err := _switch.BuildNgBridge(switchInst)
		if err != nil {
			slog.Error("error creating ng bridge", "err", err)
			// already created in db, so ignore system state and proceed on...
			return &cirrina.SwitchId{Value: switchInst.ID}, nil
		}
	default:
		slog.Error("unknown switch type bringing up new switch")

		return &cirrina.SwitchId{}, errors.New("unknown switch type creating switch")
	}

	return &cirrina.SwitchId{Value: switchInst.ID}, nil
}

func mapSwitchTypeTypeToDBString(switchType cirrina.SwitchType) (string, error) {
	switch switchType {
	case cirrina.SwitchType_IF:
		return "IF", nil
	case cirrina.SwitchType_NG:
		return "NG", nil
	default:
		return "", errors.New("invalid type")
	}
}

func mapSwitchTypeDBStringToType(switchType string) (*cirrina.SwitchType, error) {
	SwitchTypeIf := cirrina.SwitchType_IF
	SwitchTypeNg := cirrina.SwitchType_NG
	switch switchType {
	case "IF":
		return &SwitchTypeIf, nil
	case "NG":
		return &SwitchTypeNg, nil
	default:
		return nil, errors.New("invalid switch type")
	}
}
