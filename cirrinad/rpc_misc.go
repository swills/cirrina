package main

import (
	"cirrina/cirrina"
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
	"context"
	"errors"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"log/slog"
)

func (s *server) RequestStatus(_ context.Context, r *cirrina.RequestID) (*cirrina.ReqStatus, error) {
	util.Trace()
	reqUuid, err := uuid.Parse(r.Value)
	if err != nil {
		return &cirrina.ReqStatus{}, errors.New("invalid id")
	}
	rs, err := requests.GetByID(reqUuid.String())
	if err != nil {
		slog.Error("ReqStatus error getting req", "vm", r.Value, "err", err)
		return &cirrina.ReqStatus{}, errors.New("not found")
	}
	if rs.ID == "" {
		return &cirrina.ReqStatus{}, errors.New("not found")
	}
	res := &cirrina.ReqStatus{
		Complete: rs.Complete,
		Success:  rs.Successful,
	}
	return res, nil
}

func (s *server) ClearUEFIState(_ context.Context, v *cirrina.VMID) (*cirrina.ReqBool, error) {
	util.Trace()
	re := cirrina.ReqBool{}
	re.Success = false

	vmUuid, err := uuid.Parse(v.Value)
	if err != nil {
		return &re, errors.New("invalid id")
	}
	vmInst, err := vm.GetById(vmUuid.String())
	if err != nil {
		slog.Error("ClearUEFIState error getting vm", "vm", v.Value, "err", err)
		return &re, errors.New("not found")
	}
	if vmInst.Name == "" {
		slog.Debug("vm not found")
		return &re, errors.New("not found")
	}
	err = vmInst.DeleteUEFIState()
	if err != nil {
		return &re, err
	}
	re.Success = true
	return &re, nil
}

func (s *server) GetNetInterfaces(_ *cirrina.NetInterfacesReq, st cirrina.VMInfo_GetNetInterfacesServer) error {
	util.Trace()
	netDevs := util.GetHostInterfaces()

	for _, nic := range netDevs {
		err := st.Send(&cirrina.NetIf{InterfaceName: nic})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *server) GetVersion(_ context.Context, _ *emptypb.Empty) (_ *wrapperspb.StringValue, _ error) {
	util.Trace()
	return wrapperspb.String(mainVersion), nil
}
