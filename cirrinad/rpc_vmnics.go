package main

import (
	"cirrina/cirrina"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
	"cirrina/cirrinad/vm_nics"
	"context"
	"errors"
	"github.com/google/uuid"
	"log/slog"
	"net"
)

func (s *server) AddVmNic(_ context.Context, v *cirrina.VmNicInfo) (*cirrina.VmNicId, error) {
	var vmNicInst vm_nics.VmNic
	var vmNicId *cirrina.VmNicId

	reflect := v.ProtoReflect()

	if v.Name == nil || !util.ValidNicName(*v.Name) {
		return vmNicId, errors.New("invalid name")
	}
	vmNicInst.Name = *v.Name
	if isOptionPassed(reflect, "description") {
		vmNicInst.Description = *v.Description
	}
	if isOptionPassed(reflect, "mac") {
		if *v.Mac == "AUTO" {
			vmNicInst.Mac = *v.Mac
		} else {
			isBroadcast, err := util.MacIsBroadcast(*v.Mac)
			if err != nil {
				return vmNicId, errors.New("invalid MAC address")
			}
			if isBroadcast {
				return vmNicId, errors.New("may not use broadcast MAC address")
			}
			isMulticast, err := util.MacIsMulticast(*v.Mac)
			if err != nil {
				return vmNicId, errors.New("invalid MAC address")
			}
			if isMulticast {
				return vmNicId, errors.New("may not use multicast MAC address")
			}
			newMac, err := net.ParseMAC(*v.Mac)
			vmNicInst.Mac = newMac.String()
		}
	}
	if isOptionPassed(reflect, "switchid") {
		if *v.Switchid == "" {
			vmNicInst.SwitchId = ""
		} else {
			switchUuid, err := uuid.Parse(*v.Switchid)
			if err != nil {
				return vmNicId, errors.New("switch id invalid")
			}
			switchInst, err := _switch.GetById(switchUuid.String())
			if err != nil {
				slog.Debug("error getting switch id",
					"id", vmNicInst.SwitchId,
					"err", err,
				)
				return vmNicId, errors.New("switch id invalid")
			}
			if switchInst.Name == "" {
				return vmNicId, errors.New("switch id invalid")
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
			vmNicInst.SwitchId = switchUuid.String()
		}
	}

	if isOptionPassed(reflect, "nettype") {
		if *v.Nettype == cirrina.NetType_VIRTIONET {
			vmNicInst.NetType = "VIRTIONET"
		} else if *v.Nettype == cirrina.NetType_E1000 {
			vmNicInst.NetType = "E1000"
		} else {
			return vmNicId, errors.New("invalid net type name")
		}
	}
	if isOptionPassed(reflect, "netdevtype") {
		if *v.Netdevtype == cirrina.NetDevType_TAP {
			vmNicInst.NetDevType = "TAP"
		} else if *v.Netdevtype == cirrina.NetDevType_VMNET {
			vmNicInst.NetDevType = "VMNET"
		} else if *v.Netdevtype == cirrina.NetDevType_NETGRAPH {
			vmNicInst.NetDevType = "NETGRAPH"
		} else {
			return vmNicId, errors.New("invalid net dev type name")
		}
		if *v.Netdevtype == cirrina.NetDevType_TAP || *v.Netdevtype == cirrina.NetDevType_VMNET {
			slog.Debug("AddVmNic", "msg", "checking rate limiting")
			r := v.ProtoReflect()
			if isOptionPassed(r, "ratelimit") &&
				isOptionPassed(r, "ratein") &&
				isOptionPassed(r, "rateout") {
				vmNicInst.RateLimit = *v.Ratelimit
				vmNicInst.RateIn = *v.Ratein
				vmNicInst.RateOut = *v.Rateout
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

	nicUuid, err := uuid.Parse(v.Value)
	if err != nil {
		return &pvmnicinfo, errors.New("id not specified or invalid")
	}
	vmNic, err := vm_nics.GetById(nicUuid.String())
	if err != nil {
		slog.Error("error getting nic", "vm", v.Value, "err", err)
		return &pvmnicinfo, errors.New("not found")
	}
	if vmNic.Name == "" {
		return &pvmnicinfo, errors.New("not found")
	}

	NetTypeVIRTIONET := cirrina.NetType_VIRTIONET
	NetTypeE1000 := cirrina.NetType_E1000

	NetDevTypeTAP := cirrina.NetDevType_TAP
	NetDevTypeVMNET := cirrina.NetDevType_VMNET
	NetDevTypeNETGRAPH := cirrina.NetDevType_NETGRAPH

	pvmnicinfo.Name = &vmNic.Name
	pvmnicinfo.Description = &vmNic.Description

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
	pvmnicinfo.Ratelimit = &vmNic.RateLimit
	pvmnicinfo.Ratein = &vmNic.RateIn
	pvmnicinfo.Rateout = &vmNic.RateOut

	return &pvmnicinfo, nil
}

func (s *server) SetVmNicSwitch(_ context.Context, v *cirrina.SetVmNicSwitchReq) (*cirrina.ReqBool, error) {
	var r cirrina.ReqBool
	r.Success = false

	if v.Vmnicid == nil || v.Vmnicid.Value == "" {
		return &r, errors.New("nic id not specified or invalid")
	}
	nicUuid, err := uuid.Parse(v.Vmnicid.Value)
	if err != nil {
		return &r, errors.New("nic id not specified or invalid")
	}
	vmNic, err := vm_nics.GetById(nicUuid.String())
	if err != nil {
		slog.Error("error getting nic", "vm", v.Vmnicid.Value, "err", err)
		return &r, errors.New("nic not found")
	}
	if vmNic.Name == "" {
		return &r, errors.New("nic not found")
	}

	if v.Switchid == nil {
		return &r, errors.New("switch id not specified or invalid")
	}

	var switchId string
	if v.Switchid.Value == "" {
		switchId = ""
	} else {
		switchUuid, err := uuid.Parse(v.Switchid.Value)
		if err != nil {
			return &r, errors.New("id not specified or invalid")
		}
		vmSwitch, err := _switch.GetById(switchUuid.String())
		if err != nil {
			slog.Error("error getting switch info", "switch", v.Switchid.Value, "err", err)
			return &r, errors.New("switch not found")
		}
		if vmSwitch.Name == "" {
			return &r, errors.New("switch not found")
		}
		switchId = vmSwitch.ID
	}

	err = vmNic.SetSwitch(switchId)
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

	nicUuid, err := uuid.Parse(vn.Value)
	if err != nil {
		return &re, errors.New("id not specified or invalid")
	}
	vmNic, err := vm_nics.GetById(nicUuid.String())
	if err != nil {
		slog.Error("error getting nic", "vm", vn.Value, "err", err)
		return &re, errors.New("not found")
	}
	if vmNic.Name == "" {
		return &re, errors.New("not found")
	}

	allVms := vm.GetAll()
	for _, aVm := range allVms {
		nics, err := aVm.GetNics()
		if err != nil {
			return &re, nil
		}
		for _, aNic := range nics {
			if aNic.ID == nicUuid.String() {
				return &re, errors.New("nic in use")
			}
		}
	}

	err = vmNic.Delete()
	if err != nil {
		return &re, err
	}
	re.Success = true
	return &re, nil
}

func (s *server) GetVmNicVm(_ context.Context, i *cirrina.VmNicId) (v *cirrina.VMID, err error) {
	slog.Debug("GetVmNicVm finding VM for nic", "nicid", i.Value)
	var pvmId cirrina.VMID

	nicUuid, err := uuid.Parse(i.Value)
	if err != nil {
		return &pvmId, errors.New("id not specified or invalid")
	}
	vmNic, err := vm_nics.GetById(nicUuid.String())
	if err != nil {
		slog.Error("error getting nic", "vm", i.Value, "err", err)
		return &pvmId, errors.New("not found")
	}
	if vmNic.Name == "" {
		return &pvmId, errors.New("not found")
	}

	allVMs := vm.GetAll()
	found := false
	for _, thisVm := range allVMs {
		thisVmNics, err := thisVm.GetNics()
		if err != nil {
			return nil, err
		}
		for _, vmNic := range thisVmNics {
			if vmNic.ID == nicUuid.String() {
				if found == true {
					slog.Error("GetVmNicVm nic in use by more than one VM",
						"nicid", nicUuid.String(),
						"vmid", thisVm.ID,
					)
					return nil, errors.New("nic in use by more than one VM")
				}
				found = true
				pvmId.Value = thisVm.ID
			}
		}
	}

	return &pvmId, nil
}

func (s *server) UpdateVmNic(_ context.Context, _ *cirrina.VmNicInfoUpdate) (_ *cirrina.ReqBool, _ error) {
	var re cirrina.ReqBool
	re.Success = false
	return &re, errors.New("not implemented yet")
}
