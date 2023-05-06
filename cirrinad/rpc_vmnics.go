package main

import (
	"cirrina/cirrina"
	"cirrina/cirrinad/vm_nics"
	"context"
	"errors"
	"golang.org/x/exp/slog"
)

func (s *server) AddVmNic(_ context.Context, v *cirrina.VmNicInfo) (*cirrina.VmNicId, error) {
	var vmNicInst vm_nics.VmNic
	var vmNicId *cirrina.VmNicId

	if *v.Name == "" {
		return vmNicId, errors.New("invalid nic name")
	}

	vmNicInst.Name = *v.Name
	vmNicInst.Description = *v.Description
	vmNicInst.Mac = *v.Mac

	if *v.Nettype == cirrina.NetType_VIRTIONET {
		vmNicInst.NetType = "VIRTIONET"
	} else if *v.Nettype == cirrina.NetType_E1000 {
		vmNicInst.NetType = "E1000"
	} else {
		return vmNicId, errors.New("invalid net type name")
	}

	if *v.Netdevtype == cirrina.NetDevType_TAP {
		vmNicInst.NetDevType = "TAP"
	} else if *v.Netdevtype == cirrina.NetDevType_VMNET {
		vmNicInst.NetType = "VMNET"
	} else if *v.Netdevtype == cirrina.NetDevType_NETGRAPH {
		vmNicInst.NetType = "NETGRAPH"
	} else {
		return vmNicId, errors.New("invalid net dev type name")
	}

	newVmNicId, err := vm_nics.Create(&vmNicInst)
	if err != nil {
		return &cirrina.VmNicId{}, err
	}
	if newVmNicId != "" {
		return &cirrina.VmNicId{Value: newVmNicId}, nil
	} else {
		return &cirrina.VmNicId{}, errors.New("unknown error creating VmNic")
	}
}

func (s *server) GetVmNicsAll(_ *cirrina.VmNicsQuery, stream cirrina.VMInfo_GetVmNicsAllServer) error {
	var nics []*vm_nics.VmNic
	var pvmnicId cirrina.VmNicId

	nics = vm_nics.GetAll()
	for e := range nics {
		pvmnicId.Value = nics[e].ID
		err := stream.Send(&pvmnicId)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *server) GetVmNicInfo(_ context.Context, v *cirrina.VmNicId) (*cirrina.VmNicInfo, error) {
	var pvmnicinfo cirrina.VmNicInfo
	vmNic, err := vm_nics.GetById(v.Value)
	if err != nil {
		slog.Debug("error getting vmnic", "vm", v.Value, "err", err)
		return &pvmnicinfo, err
	}

	NetTypeVIRTIONET := cirrina.NetType_VIRTIONET
	NetTypeE1000 := cirrina.NetType_E1000

	NetDevTypeTAP := cirrina.NetDevType_TAP
	NetDevTypeVMNET := cirrina.NetDevType_VMNET
	NetDevTypeNETGRAPH := cirrina.NetDevType_NETGRAPH

	pvmnicinfo.Name = &vmNic.Name
	pvmnicinfo.Description = &vmNic.Description
	slog.Debug("GetVmNicInfo", "description", *pvmnicinfo.Description)
	pvmnicinfo.Vmid = &vmNic.VmId

	if vmNic.NetType == "VIRTIONET" {
		pvmnicinfo.Nettype = &NetTypeVIRTIONET
	} else if vmNic.NetType == "E1000" {
		pvmnicinfo.Nettype = &NetTypeE1000
	} else {
		slog.Error("Invalid net type", "vmnicid", vmNic.ID, "nettype", vmNic.NetType)
	}

	if vmNic.NetDevType == "TAP" {
		pvmnicinfo.Netdevtype = &NetDevTypeTAP
	} else if vmNic.NetDevType == "VMNET" {
		pvmnicinfo.Netdevtype = &NetDevTypeVMNET
	} else if vmNic.NetDevType == "NETGRAPH" {
		pvmnicinfo.Netdevtype = &NetDevTypeNETGRAPH
	} else {
		slog.Error("Invalid net dev type", "vmnicid", vmNic.ID, "netdevtype", vmNic.NetDevType)
	}

	pvmnicinfo.Vmid = &vmNic.VmId
	pvmnicinfo.Switchid = &vmNic.SwitchId
	pvmnicinfo.Mac = &vmNic.Mac

	return &pvmnicinfo, nil
}
