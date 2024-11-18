package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"cirrina/cirrina"
	_switch "cirrina/cirrinad/switch"
)

func (s *server) AddSwitch(_ context.Context, switchInfo *cirrina.SwitchInfo) (*cirrina.SwitchId, error) {
	switchType, err := mapSwitchTypeTypeToDBString(switchInfo.GetSwitchType())
	if err != nil {
		return nil, err
	}

	switchInst := &_switch.Switch{
		Name:        switchInfo.GetName(),
		Description: switchInfo.GetDescription(),
		Type:        switchType,
		Uplink:      switchInfo.GetUplink(),
	}

	err = _switch.Create(switchInst)
	if err != nil {
		return nil, fmt.Errorf("error creating switch: %w", err)
	}

	return &cirrina.SwitchId{Value: switchInst.ID}, nil
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
		err = _switch.DestroyIfSwitch(switchInst.Name, true)
		if err != nil {
			return &res, fmt.Errorf("error destroying bridge: %w", err)
		}
	case "NG":
		err = _switch.DestroyNgSwitch(switchInst.Name)
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

func validateSetSwitchUplinkRequest(switchUplinkReq *cirrina.SwitchUplinkReq) (*_switch.Switch, error) {
	var err error

	var switchUUID uuid.UUID

	var switchInst *_switch.Switch

	if switchUplinkReq.GetSwitchid() == nil {
		return nil, errInvalidID
	}

	switchUUID, err = uuid.Parse(switchUplinkReq.GetSwitchid().GetValue())
	if err != nil {
		return nil, errInvalidID
	}

	switchInst, err = _switch.GetByID(switchUUID.String())
	if err != nil {
		return nil, fmt.Errorf("error getting switch: %w", err)
	}

	if switchUplinkReq.Uplink == nil {
		return nil, errSwitchInvalidUplink
	}

	return switchInst, nil
}

func (s *server) SetSwitchUplink(_ context.Context, switchUplinkReq *cirrina.SwitchUplinkReq) (*cirrina.ReqBool, error) { //nolint:lll
	var res cirrina.ReqBool

	var err error

	var switchInst *_switch.Switch

	res.Success = false

	switchInst, err = validateSetSwitchUplinkRequest(switchUplinkReq)
	if err != nil {
		return &res, err
	}

	uplink := switchUplinkReq.GetUplink()
	slog.Debug("SetSwitchUplink", "switch", switchUplinkReq.GetSwitchid().GetValue(), "uplink", uplink)

	if uplink == "" {
		if switchInst.Uplink != "" {
			slog.Debug("SetSwitchUplink", "msg", "unsetting switch uplink", "switchInst", switchInst)

			err = switchInst.UnsetUplink()
			if err != nil {
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

		err = switchInst.SetUplink(uplink)
		if err != nil {
			return &res, fmt.Errorf("error setting switch uplink: %w", err)
		}
	} else {
		slog.Debug("SetSwitchUplink", "msg", "re-setting switch uplink", "switchInst", switchInst)

		err = switchInst.UnsetUplink()
		if err != nil {
			return &res, fmt.Errorf("error unsetting switch uplink: %w", err)
		}

		err = switchInst.SetUplink(uplink)
		if err != nil {
			return &res, fmt.Errorf("error setting switch uplink: %w", err)
		}
	}

	res.Success = true

	return &res, nil
}

func (s *server) SetSwitchInfo(_ context.Context, switchInfoUpdate *cirrina.SwitchInfoUpdate) (*cirrina.ReqBool, error) { //nolint:lll
	var res cirrina.ReqBool
	res.Success = false

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
