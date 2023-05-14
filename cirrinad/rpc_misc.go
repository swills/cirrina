package main

import (
	"cirrina/cirrina"
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
	"context"
	"golang.org/x/exp/slog"
)

func (s *server) RequestStatus(_ context.Context, r *cirrina.RequestID) (*cirrina.ReqStatus, error) {
	rs, err := requests.GetByID(r.Value)
	if err != nil {
		return &cirrina.ReqStatus{}, err
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
	vmInst, err := vm.GetById(v.Value)
	if err != nil {
		slog.Error("error getting vm", "vm", v.Value, "err", err)
		return &re, err
	}
	err = vmInst.DeleteUEFIState()
	if err != nil {
		return &re, err
	}
	re.Success = true
	return &re, nil
}

func (s *server) GetNetInterfaces(_ *cirrina.NetInterfacesReq, st cirrina.VMInfo_GetNetInterfacesServer) error {
	netDevs := util.GetHostInterfaces()

	for _, nic := range netDevs {
		err := st.Send(&cirrina.NetIf{InterfaceName: nic})
		if err != nil {
			return err
		}
	}

	return nil
}
