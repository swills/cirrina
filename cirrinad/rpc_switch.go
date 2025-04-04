package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	epb "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cirrina/cirrina"
	_switch "cirrina/cirrinad/switch"
)

func (s *server) AddSwitch(_ context.Context, switchInfo *cirrina.SwitchInfo) (*cirrina.SwitchId, error) {
	switchType, err := _switch.MapSwitchTypeTypeToDBString(switchInfo.GetSwitchType())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "switch type invalid")
	}

	switchInst := &_switch.Switch{
		Name:        switchInfo.GetName(),
		Description: switchInfo.GetDescription(),
		Type:        switchType,
		Uplink:      switchInfo.GetUplink(),
	}

	err = _switch.Create(switchInst)
	if err != nil {
		switch {
		case errors.Is(err, _switch.ErrSwitchExists):
			return nil, status.Error(codes.AlreadyExists, "switch by that name already exists")
		case errors.Is(err, _switch.ErrSwitchInvalidName):
			return nil, status.Error(codes.InvalidArgument, "switch name invalid")
		case errors.Is(err, _switch.ErrSwitchInvalidUplink):
			return nil, status.Error(codes.FailedPrecondition, "switch uplink does not exist")
		case errors.Is(err, _switch.ErrSwitchUplinkInUse):
			return nil, status.Error(codes.FailedPrecondition, "switch uplink in use by another switch")
		default:
			return nil, status.Errorf(codes.Internal, "internal error creating switch: %s", err)
		}
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

	switchInfo.SwitchType, err = _switch.MapSwitchTypeDBStringToType(vmSwitch.Type)
	if err != nil {
		return &cirrina.SwitchInfo{}, fmt.Errorf("internal error: %w", err)
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

	err = switchInst.Delete()
	if err != nil {
		if errors.Is(err, _switch.ErrSwitchInUse) {
			errStatus := status.New(codes.FailedPrecondition, "switch in use")
			errDetails, err2 := errStatus.WithDetails(
				&epb.PreconditionFailure{
					Violations: []*epb.PreconditionFailure_Violation{{
						Subject:     fmt.Sprintf("name: %s, id:%s", switchInst.Name, switchInst.ID),
						Description: "Switch is in use as uplink by existing VM NIC(s)",
					}},
				},
			)

			if err2 != nil {
				return &res, errStatus.Err()
			}

			return &res, errDetails.Err()
		}

		return &res, fmt.Errorf("%w", err)
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

	var err error

	var switchUUID uuid.UUID

	var switchInst *_switch.Switch

	res.Success = false

	switchUUID, err = uuid.Parse(switchInfoUpdate.GetId())
	if err != nil {
		return &res, errInvalidID
	}

	switchInst, err = _switch.GetByID(switchUUID.String())
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

// uplinkInUse checks if the uplink is in use by a switch other than this one
func uplinkInUse(vmSwitch *_switch.Switch, uplinkName string) bool {
	switchList := _switch.GetAll()
	for _, sw := range switchList {
		if sw.ID != vmSwitch.ID && sw.Type == vmSwitch.Type && sw.Uplink == uplinkName {
			return true
		}
	}

	return false
}
