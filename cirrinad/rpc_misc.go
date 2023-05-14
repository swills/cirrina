package main

import (
	"cirrina/cirrina"
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/vm"
	"context"
	"golang.org/x/exp/slog"
	"net"
	"strings"
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
	var phNic cirrina.NetIf
	var netDevs []string
	netInterfaces, _ := net.Interfaces()
	for _, inter := range netInterfaces {
		netDevs = append(netDevs, inter.Name)
	}

	for e, nic := range netDevs {
		slog.Debug("netdev", "e", e, "nic", nic)
		if strings.HasPrefix(nic, "lo") {
			continue
		}
		if strings.HasPrefix(nic, "bridge") {
			continue
		}
		if strings.HasPrefix(nic, "tap") {
			continue
		}
		if strings.HasPrefix(nic, "vmnet") {
			continue
		}
		phNic.InterfaceName = nic
		err := st.Send(&phNic)
		if err != nil {
			return err
		}
	}

	return nil
}
