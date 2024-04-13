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
	"cirrina/cirrinad/vmnic"
)

func (s *server) AddVMNic(_ context.Context, v *cirrina.VmNicInfo) (*cirrina.VmNicId, error) {
	var vmNicInst vmnic.VMNic
	var vmNicID *cirrina.VmNicId
	var err error

	if v.Name == nil || !util.ValidNicName(*v.Name) {
		return vmNicID, errors.New("invalid name")
	}
	vmNicInst.Name = *v.Name

	if v.Description != nil {
		vmNicInst.Description = *v.Description
	}
	if v.Mac != nil {
		vmNicInst.Mac, err = vmnic.ParseMac(*v.Mac)
		if err != nil {
			return vmNicID, err
		}
	}
	if v.Netdevtype != nil {
		newNetDevType, err := vmnic.ParseNetDevType(*v.Netdevtype)
		if err != nil {
			return vmNicID, err
		}
		vmNicInst.NetDevType = newNetDevType
	}
	if v.Nettype != nil {
		newNetType, err := vmnic.ParseNetType(*v.Nettype)
		if err != nil {
			return vmNicID, err
		}
		vmNicInst.NetType = newNetType
	}
	if v.Switchid != nil {
		newSwitchID, err := _switch.ParseSwitchID(*v.Switchid, vmNicInst.NetType)
		if err != nil {
			return vmNicID, err
		}
		vmNicInst.SwitchID = newSwitchID
	}
	vmNicParseRateLimit(&vmNicInst, v.Ratelimit, v.Ratein, v.Rateout)
	newVMNicID, err := vmnic.Create(&vmNicInst)
	if err != nil {
		return &cirrina.VmNicId{}, err
	}
	if newVMNicID != "" {
		return &cirrina.VmNicId{Value: newVMNicID}, nil
	} else {
		return &cirrina.VmNicId{}, errors.New("unknown error creating VMNic")
	}
}

func (s *server) GetVMNicsAll(_ *cirrina.VmNicsQuery, stream cirrina.VMInfo_GetVMNicsAllServer) error {
	var nics []*vmnic.VMNic
	var pvmnicID cirrina.VmNicId

	nics = vmnic.GetAll()
	for e := range nics {
		pvmnicID.Value = nics[e].ID
		err := stream.Send(&pvmnicID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *server) GetVMNicInfo(_ context.Context, v *cirrina.VmNicId) (*cirrina.VmNicInfo, error) {
	var pvmnicinfo cirrina.VmNicInfo

	nicUUID, err := uuid.Parse(v.Value)
	if err != nil {
		return &pvmnicinfo, errors.New("id not specified or invalid")
	}
	vmNic, err := vmnic.GetByID(nicUUID.String())
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

	pvmnicinfo.Switchid = &vmNic.SwitchID
	pvmnicinfo.Mac = &vmNic.Mac
	pvmnicinfo.Ratelimit = &vmNic.RateLimit
	pvmnicinfo.Ratein = &vmNic.RateIn
	pvmnicinfo.Rateout = &vmNic.RateOut

	return &pvmnicinfo, nil
}

func (s *server) SetVMNicSwitch(_ context.Context, v *cirrina.SetVmNicSwitchReq) (*cirrina.ReqBool, error) {
	var r cirrina.ReqBool
	r.Success = false

	if v.Vmnicid == nil || v.Vmnicid.Value == "" {
		return &r, errors.New("nic id not specified or invalid")
	}
	nicUUID, err := uuid.Parse(v.Vmnicid.Value)
	if err != nil {
		return &r, errors.New("nic id not specified or invalid")
	}
	vmNic, err := vmnic.GetByID(nicUUID.String())
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

	var switchID string
	if v.Switchid.Value == "" {
		switchID = ""
	} else {
		switchUUID, err := uuid.Parse(v.Switchid.Value)
		if err != nil {
			return &r, errors.New("id not specified or invalid")
		}
		vmSwitch, err := _switch.GetByID(switchUUID.String())
		if err != nil {
			slog.Error("error getting switch info", "switch", v.Switchid.Value, "err", err)

			return &r, errors.New("switch not found")
		}
		if vmSwitch.Name == "" {
			return &r, errors.New("switch not found")
		}
		switchID = vmSwitch.ID
	}

	err = vmNic.SetSwitch(switchID)
	if err != nil {
		return &r, err
	}
	r.Success = true

	return &r, nil
}

func (s *server) RemoveVMNic(_ context.Context, vn *cirrina.VmNicId) (*cirrina.ReqBool, error) {
	var re cirrina.ReqBool
	re.Success = false
	slog.Debug("RemoveVMNic", "vmnic", vn.Value)

	nicUUID, err := uuid.Parse(vn.Value)
	if err != nil {
		return &re, errors.New("id not specified or invalid")
	}
	vmNic, err := vmnic.GetByID(nicUUID.String())
	if err != nil {
		slog.Error("error getting nic", "vm", vn.Value, "err", err)

		return &re, errors.New("not found")
	}
	if vmNic.Name == "" {
		return &re, errors.New("not found")
	}

	allVms := vm.GetAll()
	for _, aVM := range allVms {
		nics, err := aVM.GetNics()
		if err != nil {
			return &re, err
		}
		for _, aNic := range nics {
			if aNic.ID == nicUUID.String() {
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

func (s *server) GetVMNicVM(_ context.Context, i *cirrina.VmNicId) (v *cirrina.VMID, err error) {
	var pvmID cirrina.VMID

	nicUUID, err := uuid.Parse(i.Value)
	if err != nil {
		return &pvmID, errors.New("id not specified or invalid")
	}
	vmNic, err := vmnic.GetByID(nicUUID.String())
	if err != nil {
		slog.Error("error getting nic", "vm", i.Value, "err", err)

		return &pvmID, errors.New("not found")
	}
	if vmNic.Name == "" {
		return &pvmID, errors.New("not found")
	}

	allVMs := vm.GetAll()
	found := false
	for _, thisVM := range allVMs {
		if thisVM.Config.ID == vmNic.ConfigID {
			if found {
				slog.Error("GetVmNicVm nic in use by more than one VM",
					"nicid", nicUUID.String(),
					"vmid", thisVM.ID,
				)

				return nil, errors.New("nic in use by more than one VM")
			}
			found = true
			pvmID.Value = thisVM.ID
		}
	}

	return &pvmID, nil
}

func updateReqIsValid(v *cirrina.VmNicInfoUpdate) (*vmnic.VMNic, bool) {
	var err error
	var vmNicInst *vmnic.VMNic
	if v == nil || v.Vmnicid == nil || v.Vmnicid.Value == "" {
		return &vmnic.VMNic{}, false
	}
	nicUUID, err := uuid.Parse(v.Vmnicid.Value)
	if err != nil {
		return &vmnic.VMNic{}, false
	}

	vmNicInst, err = vmnic.GetByID(nicUUID.String())
	if err != nil {
		return &vmnic.VMNic{}, false
	}

	return vmNicInst, true
}

func (s *server) UpdateVMNic(_ context.Context, v *cirrina.VmNicInfoUpdate) (*cirrina.ReqBool, error) {
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
		newMac, err = vmnic.ParseMac(*v.Mac)
		if err != nil {
			return &re, err
		}
		vmNicInst.Mac = newMac
	}
	if v.Netdevtype != nil {
		newNetDevType, err := vmnic.ParseNetDevType(*v.Netdevtype)
		if err != nil {
			return &re, err
		}
		vmNicInst.NetDevType = newNetDevType
	}
	if v.Nettype != nil {
		newNetType, err := vmnic.ParseNetType(*v.Nettype)
		if err != nil {
			return &re, err
		}
		vmNicInst.NetType = newNetType
	}
	if v.Switchid != nil {
		newSwitchID, err := _switch.ParseSwitchID(*v.Switchid, vmNicInst.NetType)
		if err != nil {
			return &re, err
		}
		vmNicInst.SwitchID = newSwitchID
	}
	vmNicParseRateLimit(vmNicInst, v.Ratelimit, v.Ratein, v.Rateout)

	err = vmNicInst.Save()
	if err != nil {
		return &re, err
	}

	re.Success = true

	return &re, nil
}

func (s *server) CloneVMNic(_ context.Context, cloneReq *cirrina.VmNicCloneReq) (*cirrina.RequestID, error) {
	if cloneReq == nil || cloneReq.Vmnicid == nil || cloneReq.Vmnicid.Value == "" ||
		cloneReq.NewVmNicName == nil || cloneReq.NewVmNicName.String() == "" {
		return &cirrina.RequestID{}, errors.New("request error")
	}

	nicUUID, err := uuid.Parse(cloneReq.Vmnicid.Value)
	if err != nil {
		return &cirrina.RequestID{}, errors.New("request error")
	}

	vmNicInst, err := vmnic.GetByID(nicUUID.String())
	if err != nil {
		slog.Error("error finding clone nic", "vm", cloneReq.Vmnicid.Value, "err", err)

		return &cirrina.RequestID{}, errors.New("not found")
	}
	if vmNicInst.Name == "" {
		return &cirrina.RequestID{}, errors.New("not found")
	}
	pendingReqIds := requests.PendingReqExists(nicUUID.String())
	if len(pendingReqIds) > 0 {
		return &cirrina.RequestID{}, fmt.Errorf("pending request for %v already exists", cloneReq.Vmnicid.Value)
	}
	newReq, err := requests.CreateNicCloneReq(
		nicUUID.String(), cloneReq.NewVmNicName.Value,
	)
	if err != nil {
		return &cirrina.RequestID{}, err
	}

	return &cirrina.RequestID{Value: newReq.ID}, nil
}

// vmNicParseRateLimit helper for dealing with rate limiting
func vmNicParseRateLimit(vmNicInst *vmnic.VMNic, rateLimit *bool, rateIn *uint64, rateOut *uint64) {
	// can only set rate limiting on IF type devs (TAP and VMNET), not netgraph devs
	if vmNicInst.NetDevType == "TAP" || vmNicInst.NetDevType == "VMNET" {
		vmNicParseRateLimitIf(vmNicInst, rateLimit, rateIn, rateOut)
	} else {
		vmNicParseRateLimitNg(vmNicInst)
	}
}

func vmNicParseRateLimitNg(vmNicInst *vmnic.VMNic) {
	vmNicInst.RateLimit = false
	vmNicInst.RateIn = 0
	vmNicInst.RateOut = 0
}

func vmNicParseRateLimitIf(vmNicInst *vmnic.VMNic, rateLimit *bool, rateIn *uint64, rateOut *uint64) {
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
