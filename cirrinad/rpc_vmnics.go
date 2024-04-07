package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"cirrina/cirrina"
	"cirrina/cirrinad/requests"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
	"cirrina/cirrinad/vm_nics"
)

func (s *server) AddVmNic(_ context.Context, v *cirrina.VmNicInfo) (*cirrina.VmNicId, error) {
	var vmNicInst vm_nics.VmNic
	var vmNicId *cirrina.VmNicId
	var err error

	if v.Name == nil || !util.ValidNicName(*v.Name) {
		return vmNicId, errors.New("invalid name")
	}
	vmNicInst.Name = *v.Name

	if v.Description != nil {
		vmNicInst.Description = *v.Description
	}
	if v.Mac != nil {
		vmNicInst.Mac, err = vm_nics.ParseMac(*v.Mac)
		if err != nil {
			return vmNicId, err
		}
	}
	if v.Netdevtype != nil {
		newNetDevType, err := vm_nics.ParseNetDevType(*v.Netdevtype)
		if err != nil {
			return vmNicId, err
		}
		vmNicInst.NetDevType = newNetDevType
	}
	if v.Nettype != nil {
		newNetType, err := vm_nics.ParseNetType(*v.Nettype)
		if err != nil {
			return vmNicId, err
		}
		vmNicInst.NetType = newNetType
	}
	if v.Switchid != nil {
		newSwitchId, err := _switch.ParseSwitchId(*v.Switchid, vmNicInst.NetType)
		if err != nil {
			return vmNicId, err
		}
		vmNicInst.SwitchId = newSwitchId
	}
	vmNicParseRateLimit(&vmNicInst, v.Ratelimit, v.Ratein, v.Rateout)
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

	switch vmNic.NetType {
	case "VIRTIONET":
		pvmnicinfo.Nettype = &NetTypeVIRTIONET
	case "E1000":
		pvmnicinfo.Nettype = &NetTypeE1000
	default:
		slog.Error("Invalid net type", "vmnicid", vmNic.ID, "nettype", vmNic.NetType)
	}

	switch vmNic.NetDevType {
	case "TAP":
		pvmnicinfo.Netdevtype = &NetDevTypeTAP
	case "VMNET":
		pvmnicinfo.Netdevtype = &NetDevTypeVMNET
	case "NETGRAPH":
		pvmnicinfo.Netdevtype = &NetDevTypeNETGRAPH
	default:
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
			return &re, err
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
		if thisVm.Config.ID == vmNic.ConfigID {
			if found {
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

	return &pvmId, nil
}

func updateReqIsValid(v *cirrina.VmNicInfoUpdate) (*vm_nics.VmNic, bool) {
	var err error
	var vmNicInst *vm_nics.VmNic
	if v == nil || v.Vmnicid == nil || v.Vmnicid.Value == "" {
		return &vm_nics.VmNic{}, false
	}
	nicUuid, err := uuid.Parse(v.Vmnicid.Value)
	if err != nil {
		return &vm_nics.VmNic{}, false
	}

	vmNicInst, err = vm_nics.GetById(nicUuid.String())
	if err != nil {
		return &vm_nics.VmNic{}, false
	}

	return vmNicInst, true
}

func (s *server) UpdateVmNic(_ context.Context, v *cirrina.VmNicInfoUpdate) (*cirrina.ReqBool, error) {
	var re cirrina.ReqBool
	var err error

	vmNicInst, isValid := updateReqIsValid(v)

	if !isValid {
		return &re, errors.New("request error")
	}

	if v.Name != nil {
		if !util.ValidNicName(*v.Name) {
			return &re, errors.New("invalid name")
		}
		vmNicInst.Name = *v.Name
	}
	if v.Description != nil {
		vmNicInst.Description = *v.Description
	}
	if v.Mac != nil {
		var newMac string
		newMac, err = vm_nics.ParseMac(*v.Mac)
		if err != nil {
			return &re, err
		}
		vmNicInst.Mac = newMac
	}
	if v.Netdevtype != nil {
		newNetDevType, err := vm_nics.ParseNetDevType(*v.Netdevtype)
		if err != nil {
			return &re, err
		}
		vmNicInst.NetDevType = newNetDevType
	}
	if v.Nettype != nil {
		newNetType, err := vm_nics.ParseNetType(*v.Nettype)
		if err != nil {
			return &re, err
		}
		vmNicInst.NetType = newNetType
	}
	if v.Switchid != nil {
		newSwitchId, err := _switch.ParseSwitchId(*v.Switchid, vmNicInst.NetType)
		if err != nil {
			return &re, err
		}
		vmNicInst.SwitchId = newSwitchId
	}
	vmNicParseRateLimit(vmNicInst, v.Ratelimit, v.Ratein, v.Rateout)

	err = vmNicInst.Save()
	if err != nil {
		return &re, err
	}

	re.Success = true

	return &re, nil
}

func (s *server) CloneVmNic(_ context.Context, cloneReq *cirrina.VmNicCloneReq) (*cirrina.RequestID, error) {
	if cloneReq == nil || cloneReq.Vmnicid == nil || cloneReq.Vmnicid.Value == "" ||
		cloneReq.NewVmNicName == nil || cloneReq.NewVmNicName.String() == "" {
		return &cirrina.RequestID{}, errors.New("request error")
	}

	nicUuid, err := uuid.Parse(cloneReq.Vmnicid.Value)
	if err != nil {
		return &cirrina.RequestID{}, errors.New("request error")
	}

	vmNicInst, err := vm_nics.GetById(nicUuid.String())
	if err != nil {
		slog.Error("error finding clone nic", "vm", cloneReq.Vmnicid.Value, "err", err)

		return &cirrina.RequestID{}, errors.New("not found")
	}
	if vmNicInst.Name == "" {
		return &cirrina.RequestID{}, errors.New("not found")
	}
	pendingReqIds := requests.PendingReqExists(nicUuid.String())
	if len(pendingReqIds) > 0 {
		return &cirrina.RequestID{}, fmt.Errorf("pending request for %v already exists", cloneReq.Vmnicid.Value)
	}
	newReq, err := requests.CreateNicCloneReq(
		nicUuid.String(), cloneReq.NewVmNicName.Value,
	)
	if err != nil {
		return &cirrina.RequestID{}, err
	}

	return &cirrina.RequestID{Value: newReq.ID}, nil
}

// vmNicParseRateLimit helper for dealing with rate limiting
func vmNicParseRateLimit(vmNicInst *vm_nics.VmNic, rateLimit *bool, rateIn *uint64, rateOut *uint64) {
	// can only set rate limiting on IF type devs (TAP and VMNET), not netgraph devs
	if vmNicInst.NetDevType == "TAP" || vmNicInst.NetDevType == "VMNET" {
		vmNicParseRateLimitIf(vmNicInst, rateLimit, rateIn, rateOut)
	} else {
		vmNicParseRateLimitNg(vmNicInst)
	}
}

func vmNicParseRateLimitNg(vmNicInst *vm_nics.VmNic) {
	vmNicInst.RateLimit = false
	vmNicInst.RateIn = 0
	vmNicInst.RateOut = 0
}

func vmNicParseRateLimitIf(vmNicInst *vm_nics.VmNic, rateLimit *bool, rateIn *uint64, rateOut *uint64) {
	if rateLimit != nil {
		vmNicInst.RateLimit = *rateLimit
	}
	if vmNicInst.RateLimit {
		if rateIn != nil {
			vmNicInst.RateIn = *rateIn
		}
		if rateOut != nil {
			vmNicInst.RateOut = *rateOut
		}
	} else { // rate limit disabled, force to zero
		vmNicInst.RateIn = 0
		vmNicInst.RateOut = 0
	}
}
