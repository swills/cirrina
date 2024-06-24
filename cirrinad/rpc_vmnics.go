package main

import (
	"context"
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

func (s *server) AddVMNic(_ context.Context, vmNicInfo *cirrina.VmNicInfo) (*cirrina.VmNicId, error) {
	var vmNicInst vmnic.VMNic

	var vmNicID *cirrina.VmNicId

	var err error

	if vmNicInfo.Name == nil || !util.ValidNicName(vmNicInfo.GetName()) {
		return vmNicID, errInvalidName
	}

	vmNicInst.Name = vmNicInfo.GetName()

	if vmNicInfo.Description != nil {
		vmNicInst.Description = vmNicInfo.GetDescription()
	}

	if vmNicInfo.Mac != nil {
		vmNicInst.Mac, err = vmnic.ParseMac(vmNicInfo.GetMac())
		if err != nil {
			return vmNicID, fmt.Errorf("error parsing MAC: %w", err)
		}
	}

	if vmNicInfo.Netdevtype != nil {
		var newNetDevType string

		newNetDevType, err = vmnic.ParseNetDevType(vmNicInfo.GetNetdevtype())
		if err != nil {
			return vmNicID, fmt.Errorf("error parsing net dev type: %w", err)
		}

		vmNicInst.NetDevType = newNetDevType
	}

	if vmNicInfo.Nettype != nil {
		var newNetType string

		newNetType, err = vmnic.ParseNetType(vmNicInfo.GetNettype())
		if err != nil {
			return vmNicID, fmt.Errorf("error parsing net type: %w", err)
		}

		vmNicInst.NetType = newNetType
	}

	if vmNicInfo.Switchid != nil && vmNicInfo.GetSwitchid() != "" {
		var newSwitchID string

		newSwitchID, err = _switch.ParseSwitchID(vmNicInfo.GetSwitchid(), vmNicInst.NetDevType)
		if err != nil {
			return vmNicID, fmt.Errorf("error parsing switch id: %w", err)
		}

		vmNicInst.SwitchID = newSwitchID
	}

	vmNicParseRateLimit(&vmNicInst, vmNicInfo.GetRatelimit(), vmNicInfo.GetRatein(), vmNicInfo.GetRateout())

	err = vmnic.Create(&vmNicInst)
	if err != nil {
		return &cirrina.VmNicId{}, fmt.Errorf("error creating VM: %w", err)
	}

	return &cirrina.VmNicId{Value: vmNicInst.ID}, nil
}

func (s *server) GetVMNicsAll(_ *cirrina.VmNicsQuery, stream cirrina.VMInfo_GetVMNicsAllServer) error {
	var nics []*vmnic.VMNic

	var pvmnicID cirrina.VmNicId

	nics = vmnic.GetAll()
	for e := range nics {
		pvmnicID.Value = nics[e].ID

		err := stream.Send(&pvmnicID)
		if err != nil {
			return fmt.Errorf("error sending to stream: %w", err)
		}
	}

	return nil
}

func (s *server) GetVMNicInfo(_ context.Context, vmNicID *cirrina.VmNicId) (*cirrina.VmNicInfo, error) {
	var pvmnicinfo cirrina.VmNicInfo

	nicUUID, err := uuid.Parse(vmNicID.GetValue())
	if err != nil {
		return &pvmnicinfo, errInvalidID
	}

	vmNic, err := vmnic.GetByID(nicUUID.String())
	if err != nil {
		slog.Error("error getting nic", "vm", vmNicID.GetValue(), "err", err)

		return &pvmnicinfo, fmt.Errorf("error getting nic: %w", err)
	}

	if vmNic.Name == "" {
		return &pvmnicinfo, errNotFound
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

func (s *server) SetVMNicSwitch(_ context.Context,
	setVMNicSwitchReq *cirrina.SetVmNicSwitchReq,
) (*cirrina.ReqBool, error) {
	var res cirrina.ReqBool
	res.Success = false

	var err error

	var nicUUID uuid.UUID

	if setVMNicSwitchReq.GetVmnicid() == nil || setVMNicSwitchReq.GetVmnicid().GetValue() == "" {
		return &res, errInvalidNicID
	}

	nicUUID, err = uuid.Parse(setVMNicSwitchReq.GetVmnicid().GetValue())
	if err != nil {
		return &res, fmt.Errorf("error parsing NIC ID: %w", err)
	}

	var vmNic *vmnic.VMNic

	vmNic, err = vmnic.GetByID(nicUUID.String())
	if err != nil {
		slog.Error("error getting nic", "vm", setVMNicSwitchReq.GetVmnicid().GetValue(), "err", err)

		return &res, errNotFound
	}

	if vmNic.Name == "" {
		return &res, errNotFound
	}

	if setVMNicSwitchReq.GetSwitchid() == nil {
		return &res, errSwitchNotFound
	}

	var switchID string
	if setVMNicSwitchReq.GetSwitchid().GetValue() == "" {
		switchID = ""
	} else {
		var switchUUID uuid.UUID

		switchUUID, err = uuid.Parse(setVMNicSwitchReq.GetSwitchid().GetValue())
		if err != nil {
			return &res, fmt.Errorf("error parsing switch ID: %w", err)
		}

		var vmSwitch *_switch.Switch

		vmSwitch, err = _switch.GetByID(switchUUID.String())
		if err != nil {
			slog.Error("error getting switch info", "switch", setVMNicSwitchReq.GetSwitchid().GetValue(), "err", err)

			return &res, fmt.Errorf("error getting switch: %w", err)
		}

		if vmSwitch.Name == "" {
			return &res, errSwitchNotFound
		}

		switchID = vmSwitch.ID
	}

	err = vmNic.SetSwitch(switchID)
	if err != nil {
		return &res, fmt.Errorf("error setting switch: %w", err)
	}

	res.Success = true

	return &res, nil
}

func (s *server) RemoveVMNic(_ context.Context, vmNicID *cirrina.VmNicId) (*cirrina.ReqBool, error) {
	var res cirrina.ReqBool
	res.Success = false

	slog.Debug("RemoveVMNic", "vmnic", vmNicID.GetValue())

	var err error

	var nicUUID uuid.UUID

	nicUUID, err = uuid.Parse(vmNicID.GetValue())
	if err != nil {
		return &res, fmt.Errorf("error parsing NIC ID: %w", err)
	}

	var vmNic *vmnic.VMNic

	vmNic, err = vmnic.GetByID(nicUUID.String())
	if err != nil {
		slog.Error("error getting nic", "vm", vmNicID.GetValue(), "err", err)

		return &res, fmt.Errorf("error getting NIC: %w", err)
	}

	if vmNic.Name == "" {
		return &res, errNotFound
	}

	allVms := vm.GetAll()
	for _, aVM := range allVms {
		var nics []vmnic.VMNic

		nics, err = aVM.GetNics()
		if err != nil {
			return &res, fmt.Errorf("error getting NICs: %w", err)
		}

		for _, aNic := range nics {
			if aNic.ID == nicUUID.String() {
				return &res, errNicInUse
			}
		}
	}

	err = vmNic.Delete()
	if err != nil {
		return &res, fmt.Errorf("error deleting NIC: %w", err)
	}

	res.Success = true

	return &res, nil
}

func (s *server) GetVMNicVM(_ context.Context, vmNicID *cirrina.VmNicId) (*cirrina.VMID, error) {
	var pvmID cirrina.VMID

	var err error

	nicUUID, err := uuid.Parse(vmNicID.GetValue())
	if err != nil {
		return &pvmID, fmt.Errorf("error parsing id: %w", err)
	}

	vmNic, err := vmnic.GetByID(nicUUID.String())
	if err != nil {
		slog.Error("error getting nic", "nic", nicUUID.String(), "err", err)

		return &pvmID, fmt.Errorf("error looking up NIC: %w", err)
	}

	if vmNic.Name == "" {
		return &pvmID, errNotFound
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

				return nil, errNicInUseByMultipleVMs
			}

			found = true
			pvmID.Value = thisVM.ID
		}
	}

	return &pvmID, nil
}

func updateReqIsValid(vmNicInfoUpdate *cirrina.VmNicInfoUpdate) (*vmnic.VMNic, bool) {
	var err error

	var vmNicInst *vmnic.VMNic

	if vmNicInfoUpdate == nil || vmNicInfoUpdate.GetVmnicid() == nil || vmNicInfoUpdate.GetVmnicid().GetValue() == "" {
		return nil, false
	}

	nicUUID, err := uuid.Parse(vmNicInfoUpdate.GetVmnicid().GetValue())
	if err != nil {
		return nil, false
	}

	vmNicInst, err = vmnic.GetByID(nicUUID.String())
	if err != nil {
		return nil, false
	}

	return vmNicInst, true
}

func (s *server) UpdateVMNic(_ context.Context, vmNicInfoUpdate *cirrina.VmNicInfoUpdate) (*cirrina.ReqBool, error) {
	var res cirrina.ReqBool

	var err error

	vmNicInst, isValid := updateReqIsValid(vmNicInfoUpdate)

	if !isValid {
		return nil, errInvalidRequest
	}

	err = vmNicParseUpdate(vmNicInfoUpdate, vmNicInst)
	if err != nil {
		return nil, fmt.Errorf("error saving NIC: %w", err)
	}

	var newRateLimit bool

	var newRateIn uint64

	var newRateOut uint64

	if vmNicInfoUpdate.Ratelimit == nil {
		newRateLimit = vmNicInst.RateLimit
	} else {
		newRateLimit = vmNicInfoUpdate.GetRatelimit()
	}

	if vmNicInfoUpdate.Ratein == nil {
		newRateIn = vmNicInst.RateIn
	} else {
		newRateIn = vmNicInfoUpdate.GetRatein()
	}

	if vmNicInfoUpdate.Rateout == nil {
		newRateOut = vmNicInst.RateOut
	} else {
		newRateOut = vmNicInfoUpdate.GetRateout()
	}

	vmNicParseRateLimit(vmNicInst, newRateLimit, newRateIn, newRateOut)

	err = vmNicInst.Save()
	if err != nil {
		return &res, fmt.Errorf("error saving NIC: %w", err)
	}

	res.Success = true

	return &res, nil
}

func vmNicParseUpdate(vmNicInfoUpdate *cirrina.VmNicInfoUpdate, vmNicInst *vmnic.VMNic) error {
	var err error

	if vmNicInfoUpdate.Name != nil {
		if !util.ValidNicName(vmNicInfoUpdate.GetName()) {
			return errInvalidName
		}

		vmNicInst.Name = vmNicInfoUpdate.GetName()
	}

	if vmNicInfoUpdate.Description != nil {
		vmNicInst.Description = vmNicInfoUpdate.GetDescription()
	}

	if vmNicInfoUpdate.Mac != nil {
		var newMac string

		newMac, err = vmnic.ParseMac(vmNicInfoUpdate.GetMac())
		if err != nil {
			return fmt.Errorf("error parsing MAC: %w", err)
		}

		vmNicInst.Mac = newMac
	}

	if vmNicInfoUpdate.Netdevtype != nil {
		var newNetDevType string

		newNetDevType, err = vmnic.ParseNetDevType(vmNicInfoUpdate.GetNetdevtype())
		if err != nil {
			return fmt.Errorf("error parsing net dev type: %w", err)
		}

		vmNicInst.NetDevType = newNetDevType
	}

	if vmNicInfoUpdate.Nettype != nil {
		var newNetType string

		newNetType, err = vmnic.ParseNetType(vmNicInfoUpdate.GetNettype())
		if err != nil {
			return fmt.Errorf("error parsing net type: %w", err)
		}

		vmNicInst.NetType = newNetType
	}

	if vmNicInfoUpdate.Switchid != nil {
		var newSwitchID string

		newSwitchID, err = _switch.ParseSwitchID(vmNicInfoUpdate.GetSwitchid(), vmNicInst.NetDevType)
		if err != nil {
			return fmt.Errorf("error parsing switch ID: %w", err)
		}

		vmNicInst.SwitchID = newSwitchID
	}

	return nil
}

func (s *server) CloneVMNic(_ context.Context, cloneReq *cirrina.VmNicCloneReq) (*cirrina.RequestID, error) {
	if cloneReq == nil || cloneReq.GetVmnicid() == nil || cloneReq.GetVmnicid().GetValue() == "" ||
		cloneReq.GetNewVmNicName() == nil || cloneReq.GetNewVmNicName().String() == "" {
		return nil, errInvalidRequest
	}

	nicUUID, err := uuid.Parse(cloneReq.GetVmnicid().GetValue())
	if err != nil {
		return nil, errInvalidRequest
	}

	vmNicInst, err := vmnic.GetByID(nicUUID.String())
	if err != nil {
		slog.Error("error finding clone nic", "vm", cloneReq.GetVmnicid().GetValue(), "err", err)

		return nil, fmt.Errorf("error finding clone nic: %w", err)
	}

	if vmNicInst.Name == "" {
		return &cirrina.RequestID{}, errNotFound
	}

	pendingReqIDs := requests.PendingReqExists(nicUUID.String())
	if len(pendingReqIDs) > 0 {
		return &cirrina.RequestID{}, errPendingReqExists
	}

	newReq, err := requests.CreateNicCloneReq(
		nicUUID.String(), cloneReq.GetNewVmNicName().GetValue(),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	return &cirrina.RequestID{Value: newReq.ID}, nil
}

// vmNicParseRateLimit helper for dealing with rate limiting
func vmNicParseRateLimit(vmNicInst *vmnic.VMNic, rateLimit bool, rateIn uint64, rateOut uint64) {
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

func vmNicParseRateLimitIf(vmNicInst *vmnic.VMNic, rateLimit bool, rateIn uint64, rateOut uint64) {
	vmNicInst.RateLimit = rateLimit
	if vmNicInst.RateLimit {
		vmNicInst.RateIn = rateIn
		vmNicInst.RateOut = rateOut
	} else { // rate limit disabled, force to zero
		vmNicInst.RateIn = 0
		vmNicInst.RateOut = 0
	}
}
