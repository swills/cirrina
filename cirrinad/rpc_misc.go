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

func (s *server) RequestStatus(_ context.Context, requestID *cirrina.RequestID) (*cirrina.ReqStatus, error) {
	reqUUID, err := uuid.Parse(requestID.GetValue())
	if err != nil {
		return nil, errInvalidID
	}
	request, err := requests.GetByID(reqUUID.String())
	if err != nil {
		slog.Error("ReqStatus error getting req", "vm", requestID.GetValue(), "err", err)

		return nil, errNotFound
	}
	if request.ID == "" {
		return &cirrina.ReqStatus{}, errNotFound
	}
	res := &cirrina.ReqStatus{
		Complete: request.Complete,
		Success:  request.Successful,
	}

	return res, nil
}

func (s *server) ClearUEFIState(_ context.Context, vmID *cirrina.VMID) (*cirrina.ReqBool, error) {
	res := cirrina.ReqBool{}
	res.Success = false

	vmUUID, err := uuid.Parse(vmID.GetValue())
	if err != nil {
		return &res, errInvalidID
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("ClearUEFIState error getting vm", "vm", vmID.GetValue(), "err", err)

		return &res, errNotFound
	}
	if vmInst.Name == "" {
		slog.Debug("vm not found")

		return &res, errNotFound
	}
	err = vmInst.DeleteUEFIState()
	if err != nil {
		return &res, fmt.Errorf("error deleting UEFI state: %w", err)
	}
	res.Success = true

	return &res, nil
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
