package main

import (
	"cirrina/cirrina"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/vm"
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
	vmNicInst.SwitchId = *v.Switchid

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
		vmNicInst.NetDevType = "VMNET"
	} else if *v.Netdevtype == cirrina.NetDevType_NETGRAPH {
		vmNicInst.NetDevType = "NETGRAPH"
	} else {
		return vmNicId, errors.New("invalid net dev type name")
	}

	if vmNicInst.SwitchId != "" {
		switchInst, err := _switch.GetById(vmNicInst.SwitchId)
		if err != nil {
			return vmNicId, errors.New("bad switch id")
		}
		if vmNicInst.NetDevType == "TAP" || vmNicInst.NetDevType == "VMNET" {
			if switchInst.Type != "IF" {
				return vmNicId, errors.New("uplink switch has wrong type")
			}
		} else if vmNicInst.NetDevType == "NETGRAPH" {
			if switchInst.Type != "NG" {
				return vmNicId, errors.New("uplink switch has wrong type")
			}
		}
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

	pvmnicinfo.Switchid = &vmNic.SwitchId
	pvmnicinfo.Mac = &vmNic.Mac

	return &pvmnicinfo, nil
}
func (s *server) SetVmNicSwitch(_ context.Context, v *cirrina.SetVmNicSwitchReq) (*cirrina.ReqBool, error) {
	var r cirrina.ReqBool
	r.Success = false

	if v.Vmnicid.Value == "" {
		return &r, errors.New("nic ID not specified")
	}
	if v.Switchid.Value == "" {
		return &r, errors.New("switch ID not specified")
	}

	vmNic, err := vm_nics.GetById(v.Vmnicid.Value)
	if err != nil {
		return &r, err
	}
	_, err = _switch.GetById(v.Switchid.Value)
	if err != nil {
		return &r, err
	}
	err = vmNic.SetSwitch(v.Switchid.Value)
	if err != nil {
		return &r, err
	}
	r.Success = true
	return &r, nil
}

func (s *server) RemoveVmNic(_ context.Context, vn *cirrina.VmNicId) (*cirrina.ReqBool, error) {
	var re cirrina.ReqBool
	re.Success = false
	slog.Debug("RemoveVmNic", "vmnic", vn.Value)

	allVms := vm.GetAll()
	for _, aVm := range allVms {
		nics, err := aVm.GetNics()
		if err != nil {
			return &re, nil
		}
		for _, aNic := range nics {
			if aNic.ID == vn.Value {
				return &re, errors.New("nic in use")
			}
		}
	}

	vmNic, err := vm_nics.GetById(vn.Value)
	if err != nil {
		return &re, err
	}
	err = vmNic.Delete()
	if err != nil {
		return &re, err
	}
	re.Success = true
	return &re, nil
}
