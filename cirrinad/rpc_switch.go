package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"cirrina/cirrina"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
)

func (s *server) AddSwitch(_ context.Context, switchInfo *cirrina.SwitchInfo) (*cirrina.SwitchId, error) {
	defaultSwitchType := cirrina.SwitchType_IF
	defaultSwitchDescription := ""

	if switchInfo.Name == nil || !util.ValidSwitchName(switchInfo.GetName()) {
		return &cirrina.SwitchId{}, errInvalidName
	}

	if switchInfo.Description == nil {
		switchInfo.Description = &defaultSwitchDescription
	}

	if switchInfo.SwitchType == nil {
		switchInfo.SwitchType = &defaultSwitchType
	}

	err := validateNewSwitch(switchInfo)
	if err != nil {
		return &cirrina.SwitchId{}, err
	}
	switchType, err := mapSwitchTypeTypeToDBString(switchInfo.GetSwitchType())
	if err != nil {
		return &cirrina.SwitchId{}, err
	}

	switchInst, err := _switch.Create(
		switchInfo.GetName(), switchInfo.GetDescription(), switchType, switchInfo.GetUplink(),
	)
	if err != nil {
		return &cirrina.SwitchId{}, fmt.Errorf("error creating switch: %w", err)
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
			return fmt.Errorf("error sending to stream: %w", err)
		}
	}

	return nil
}

func (s *server) GetSwitchInfo(_ context.Context, switchID *cirrina.SwitchId) (*cirrina.SwitchInfo, error) {
	var switchInfo cirrina.SwitchInfo

	switchUUID, err := uuid.Parse(switchID.GetValue())
	if err != nil {
		return &cirrina.SwitchInfo{}, errInvalidID
	}

	vmSwitch, err := _switch.GetByID(switchUUID.String())
	if err != nil {
		slog.Error("error getting switch info", "switch", switchID.GetValue(), "err", err)

		return &cirrina.SwitchInfo{}, fmt.Errorf("error getting switch info: %w", err)
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

func (s *server) RemoveSwitch(_ context.Context, switchID *cirrina.SwitchId) (*cirrina.ReqBool, error) {
	var res cirrina.ReqBool
	res.Success = false

	switchUUID, err := uuid.Parse(switchID.GetValue())
	if err != nil {
		return &res, errInvalidID
	}

	switchInst, err := _switch.GetByID(switchUUID.String())
	if err != nil {
		return &res, errNotFound
	}

	err2 := _switch.CheckSwitchInUse(switchID.GetValue())
	if err2 != nil {
		slog.Debug("attempted to delete switch which is in use",
			"switch", switchID.GetValue(),
			"switch_name", switchInst.Name,
		)

		return &res, errSwitchInUse
	}

	switch switchInst.Type {
	case "IF":
		err = _switch.DestroyIfBridge(switchInst.Name, true)
		if err != nil {
			return &res, fmt.Errorf("error destroying bridge: %w", err)
		}
	case "NG":
		err = _switch.DestroyNgBridge(switchInst.Name)
		if err != nil {
			slog.Error("switch removal failure")

			return &res, fmt.Errorf("error destroying bridge: %w", err)
		}
	default:
		return &res, errSwitchInvalidType
	}
	slog.Debug("RemoveSwitch", "switchid", switchID.GetValue())
	err = _switch.Delete(switchID.GetValue())
	if err != nil {
		return &res, fmt.Errorf("error deleting bridge: %w", err)
	}
	res.Success = true

	return &res, nil
}

func (s *server) SetSwitchUplink(_ context.Context,
	switchUplinkReq *cirrina.SwitchUplinkReq,
) (*cirrina.ReqBool, error) {
	var res cirrina.ReqBool
	res.Success = false

	if switchUplinkReq.GetSwitchid() == nil {
		return &res, errInvalidID
	}

	switchUUID, err := uuid.Parse(switchUplinkReq.GetSwitchid().GetValue())
	if err != nil {
		return &res, errInvalidID
	}

	if switchUplinkReq.Uplink == nil {
		return &res, errSwitchInvalidUplink
	}

	uplink := switchUplinkReq.GetUplink()
	slog.Debug("SetSwitchUplink", "switch", switchUplinkReq.GetSwitchid().GetValue(), "uplink", uplink)
	switchInst, err := _switch.GetByID(switchUUID.String())
	if err != nil {
		return &res, fmt.Errorf("error getting switch: %w", err)
	}

	if uplink == "" {
		if switchInst.Uplink != "" {
			slog.Debug("SetSwitchUplink", "msg", "unsetting switch uplink", "switchInst", switchInst)
			if err = switchInst.UnsetUplink(); err != nil {
				return &res, fmt.Errorf("error unsetting swtich uplink: %w", err)
			}
		}
		res.Success = true

		return &res, nil
	}

	if uplinkInUse(switchInst, uplink) {
		slog.Error("SetSwitchUplink uplink already in use by another switch",
			"uplink", uplink,
			"name", switchInst.Name,
		)

		return &res, errSwitchUplinkInUse
	}
	if switchInst.Uplink != uplink {
		slog.Debug("SetSwitchUplink", "msg", "unsetting switch uplink", "switchInst", switchInst)
		// ignore error here because it may not be set so removing it can fail
		_ = switchInst.UnsetUplink()
		slog.Debug("SetSwitchUplink", "msg", "setting switch uplink", "switchInst", switchInst)
		if err = switchInst.SetUplink(uplink); err != nil {
			return &res, fmt.Errorf("error setting switch uplink: %w", err)
		}
	} else {
		slog.Debug("SetSwitchUplink", "msg", "re-setting switch uplink", "switchInst", switchInst)
		if err = switchInst.UnsetUplink(); err != nil {
			return &res, fmt.Errorf("error unsetting switch uplink: %w", err)
		}
		if err = switchInst.SetUplink(uplink); err != nil {
			return &res, fmt.Errorf("error setting switch uplink: %w", err)
		}
	}
	res.Success = true

	return &res, nil
}

func (s *server) SetSwitchInfo(_ context.Context,
	switchInfoUpdate *cirrina.SwitchInfoUpdate,
) (*cirrina.ReqBool, error) {
	var res cirrina.ReqBool
	res.Success = false

	if switchInfoUpdate.GetId() == "" {
		return &res, errInvalidID
	}

	switchUUID, err := uuid.Parse(switchInfoUpdate.GetId())
	if err != nil {
		return &res, errInvalidID
	}

	switchInst, err := _switch.GetByID(switchUUID.String())
	if err != nil {
		return &res, fmt.Errorf("error getting switch ID: %w", err)
	}

	if switchInfoUpdate.Description != nil {
		switchInst.Description = switchInfoUpdate.GetDescription()
	}

	err = switchInst.Save()
	if err != nil {
		return &res, errSwitchInternalDB
	}
	res.Success = true

	return &res, nil
}

// uplinkInUse check if the uplink is in use by a switch other than this one
func uplinkInUse(vmSwitch *_switch.Switch, uplinkName string) bool {
	switchList := _switch.GetAll()
	for _, sw := range switchList {
		if sw.ID != vmSwitch.ID && sw.Type == vmSwitch.Type && sw.Uplink == uplinkName {
			return true
		}
	}

	return false
}

func validateNewSwitch(switchInfo *cirrina.SwitchInfo) error {
	if switchInfo == nil || switchInfo.SwitchType == nil {
		return errSwitchInvalidType
	}
	switch switchInfo.GetSwitchType() {
	case cirrina.SwitchType_IF:
		return validateIfSwitch(switchInfo)
	case cirrina.SwitchType_NG:
		return validateNgSwitch(switchInfo)
	default:
		return errSwitchInvalidType
	}
}

func validateNgSwitch(switchInfo *cirrina.SwitchInfo) error {
	// it can't be a member of another bridge of same type already
	if switchInfo.Uplink != nil && switchInfo.GetUplink() != "" {
		alreadyUsed, err := _switch.MemberUsedByNgBridge(switchInfo.GetUplink())
		if err != nil {
			slog.Error("error checking if member already used", "err", err)

			return fmt.Errorf("error checking if member already used: %w", err)
		}
		if alreadyUsed {
			return errSwitchUplinkInUse
		}
	}
	if !strings.HasPrefix(switchInfo.GetName(), "bnet") {
		slog.Error("invalid bridge name", "name", switchInfo.GetName())

		return errSwitchInvalidName
	}

	bridgeNumStr := strings.TrimPrefix(switchInfo.GetName(), "bnet")
	bridgeNum, err := strconv.Atoi(bridgeNumStr)
	if err != nil {
		slog.Error("invalid bridge name", "name", switchInfo.GetName())

		return fmt.Errorf("error checking switch name: %w", err)
	}
	bridgeNumFormattedString := strconv.FormatInt(int64(bridgeNum), 10)
	// Check for silly things like "0123"
	if bridgeNumStr != bridgeNumFormattedString {
		slog.Error("invalid name", "name", switchInfo.GetName())

		return errSwitchInvalidName
	}

	return nil
}

func validateIfSwitch(switchInfo *cirrina.SwitchInfo) error {
	// it can't be a member of another bridge of same type already
	if switchInfo.Uplink != nil && switchInfo.GetUplink() != "" {
		alreadyUsed, err := _switch.MemberUsedByIfBridge(switchInfo.GetUplink())
		if err != nil {
			slog.Error("error checking if member already used", "err", err)

			return fmt.Errorf("error checking if switch uplink in use by another bridge: %w", err)
		}
		if alreadyUsed {
			return errSwitchUplinkInUse
		}
	}
	if !strings.HasPrefix(switchInfo.GetName(), "bridge") {
		slog.Error("invalid name", "name", switchInfo.GetName())

		return errSwitchInvalidName
	}

	bridgeNumStr := strings.TrimPrefix(switchInfo.GetName(), "bridge")
	bridgeNum, err := strconv.Atoi(bridgeNumStr)
	if err != nil {
		slog.Error("invalid bridge name", "name", switchInfo.GetName())

		return fmt.Errorf("error checking switch name: %w", err)
	}
	bridgeNumFormattedString := strconv.FormatInt(int64(bridgeNum), 10)
	// Check for silly things like "0123"
	if bridgeNumStr != bridgeNumFormattedString {
		slog.Error("invalid name", "name", switchInfo.GetName())

		return errSwitchInvalidName
	}

	return nil
}

func bringUpNewSwitch(switchInst *_switch.Switch) (*cirrina.SwitchId, error) {
	if switchInst == nil || switchInst.ID != "" {
		return &cirrina.SwitchId{}, errInvalidID
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

		return &cirrina.SwitchId{}, errSwitchInvalidType
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
		return "", errSwitchInvalidType
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
		return nil, errSwitchInvalidType
	}
}
