package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"cirrina/cirrina"
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
)

func (s *server) RequestStatus(_ context.Context, r *cirrina.RequestID) (*cirrina.ReqStatus, error) {
	reqUUID, err := uuid.Parse(r.Value)
	if err != nil {
		return &cirrina.ReqStatus{}, errInvalidID
	}
	rs, err := requests.GetByID(reqUUID.String())
	if err != nil {
		slog.Error("ReqStatus error getting req", "vm", r.Value, "err", err)

		return &cirrina.ReqStatus{}, errNotFound
	}
	if rs.ID == "" {
		return &cirrina.ReqStatus{}, errNotFound
	}
	res := &cirrina.ReqStatus{
		Complete: rs.Complete,
		Success:  rs.Successful,
	}

	return res, nil
}

func (s *server) ClearUEFIState(_ context.Context, v *cirrina.VMID) (*cirrina.ReqBool, error) {
	re := cirrina.ReqBool{}
	re.Success = false

	vmUUID, err := uuid.Parse(v.Value)
	if err != nil {
		return &re, errInvalidID
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("ClearUEFIState error getting vm", "vm", v.Value, "err", err)

		return &re, errNotFound
	}
	if vmInst.Name == "" {
		slog.Debug("vm not found")

		return &re, errNotFound
	}
	err = vmInst.DeleteUEFIState()
	if err != nil {
		return &re, fmt.Errorf("error deleting UEFI state: %w", err)
	}
	re.Success = true

	return &re, nil
}

func (s *server) GetNetInterfaces(_ *cirrina.NetInterfacesReq, stream cirrina.VMInfo_GetNetInterfacesServer) error {
	netDevs := util.GetHostInterfaces()

	for _, nic := range netDevs {
		err := stream.Send(&cirrina.NetIf{InterfaceName: nic})
		if err != nil {
			return fmt.Errorf("error sending to stream: %w", err)
		}
	}

	return nil
}

func (s *server) GetVersion(_ context.Context, _ *emptypb.Empty) (_ *wrapperspb.StringValue, _ error) {
	return wrapperspb.String(mainVersion), nil
}
