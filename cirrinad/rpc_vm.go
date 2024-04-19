package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"cirrina/cirrina"
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
)

func (s *server) UpdateVM(_ context.Context, vmConfig *cirrina.VMConfig) (*cirrina.ReqBool, error) {
	res := cirrina.ReqBool{}
	res.Success = false

	vmUUID, err := uuid.Parse(vmConfig.Id)
	if err != nil {
		return &res, fmt.Errorf("error parsing VM ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("UpdateVM error getting vm", "vm", vmConfig.Id, "err", err)

		return &res, errNotFound
	}
	if vmInst.Name == "" {
		return &res, errNotFound
	}

	err = updateVMBasics(vmConfig, vmInst)
	if err != nil {
		return &res, err
	}

	err = updateVMCom1(vmConfig, vmInst)
	if err != nil {
		return &res, err
	}
	err = updateVMCom2(vmConfig, vmInst)
	if err != nil {
		return &res, err
	}
	err = updateVMCom3(vmConfig, vmInst)
	if err != nil {
		return &res, err
	}
	err = updateVMCom4(vmConfig, vmInst)
	if err != nil {
		return &res, err
	}

	updateVMScreen(vmConfig, vmInst)
	err = updateVMScreenOptions(vmConfig, vmInst)
	if err != nil {
		return &res, err
	}

	err = updateVMSound(vmConfig, vmInst)
	if err != nil {
		return &res, err
	}

	updateVMStart(vmConfig, vmInst)
	updateVMAdvanced1(vmConfig, vmInst)
	updateVMAdvanced2(vmConfig, vmInst)

	err = updateVMDebug(vmConfig, vmInst)
	if err != nil {
		return &res, err
	}

	updateVMPriorityLimits(vmConfig, vmInst)

	err = vmInst.Save()
	if err != nil {
		return &res, fmt.Errorf("error saving VM: %w", err)
	}
	res.Success = true

	return &res, nil
}

func updateVMPriorityLimits(vmConfig *cirrina.VMConfig, vmInst *vm.VM) {
	if vmConfig.Priority != nil {
		vmInst.Config.Priority = *vmConfig.Priority
	}
	if vmConfig.Protect != nil {
		vmInst.Config.Protect = sql.NullBool{Bool: *vmConfig.Protect, Valid: true}
	}
	if vmConfig.Pcpu != nil {
		vmInst.Config.Pcpu = *vmConfig.Pcpu
	}
	if vmConfig.Rbps != nil {
		vmInst.Config.Rbps = *vmConfig.Rbps
	}
	if vmConfig.Wbps != nil {
		vmInst.Config.Wbps = *vmConfig.Wbps
	}
	if vmConfig.Riops != nil {
		vmInst.Config.Riops = *vmConfig.Riops
	}
	if vmConfig.Wiops != nil {
		vmInst.Config.Wiops = *vmConfig.Wiops
	}
}

func updateVMDebug(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	if vmConfig.Debug != nil {
		if *vmConfig.Debug {
			vmInst.Config.Debug = true
		} else {
			vmInst.Config.Debug = false
		}
	}
	if vmConfig.DebugWait != nil {
		if *vmConfig.DebugWait {
			vmInst.Config.DebugWait = true
		} else {
			vmInst.Config.DebugWait = false
		}
	}
	if vmConfig.DebugPort != nil {
		if *vmConfig.DebugPort != "AUTO" {
			port, err := strconv.Atoi(*vmConfig.DebugPort)
			if err != nil || port < 1024 || port > 65535 {
				return errInvalidDebugPort
			}
		}
		vmInst.Config.DebugPort = *vmConfig.DebugPort
	}

	return nil
}

func updateVMAdvanced1(vmConfig *cirrina.VMConfig, vmInst *vm.VM) {
	if vmConfig.Hostbridge != nil {
		if *vmConfig.Hostbridge {
			vmInst.Config.HostBridge = true
		} else {
			vmInst.Config.HostBridge = false
		}
	}
	if vmConfig.Acpi != nil {
		if *vmConfig.Acpi {
			vmInst.Config.ACPI = true
		} else {
			vmInst.Config.ACPI = false
		}
	}
	if vmConfig.Storeuefi != nil {
		if *vmConfig.Storeuefi {
			vmInst.Config.StoreUEFIVars = true
		} else {
			vmInst.Config.StoreUEFIVars = false
		}
	}
	if vmConfig.Utc != nil {
		if *vmConfig.Utc {
			vmInst.Config.UTCTime = true
		} else {
			vmInst.Config.UTCTime = false
		}
	}
	if vmConfig.Wireguestmem != nil {
		if *vmConfig.Wireguestmem {
			vmInst.Config.WireGuestMem = true
		} else {
			vmInst.Config.WireGuestMem = false
		}
	}
}

func updateVMAdvanced2(vmConfig *cirrina.VMConfig, vmInst *vm.VM) {
	if vmConfig.Dpo != nil {
		if *vmConfig.Dpo {
			vmInst.Config.DestroyPowerOff = true
		} else {
			vmInst.Config.DestroyPowerOff = false
		}
	}
	if vmConfig.Eop != nil {
		if *vmConfig.Eop {
			vmInst.Config.ExitOnPause = true
		} else {
			vmInst.Config.ExitOnPause = false
		}
	}
	if vmConfig.Ium != nil {
		if *vmConfig.Ium {
			vmInst.Config.IgnoreUnknownMSR = true
		} else {
			vmInst.Config.IgnoreUnknownMSR = false
		}
	}
	if vmConfig.Hlt != nil {
		if *vmConfig.Hlt {
			vmInst.Config.UseHLT = true
		} else {
			vmInst.Config.UseHLT = false
		}
	}
	// TODO -- potential security issue, should it be removed?
	if vmConfig.ExtraArgs != nil {
		vmInst.Config.ExtraArgs = *vmConfig.ExtraArgs
	}
}

func updateVMStart(vmConfig *cirrina.VMConfig, vmInst *vm.VM) {
	if vmConfig.Autostart != nil {
		if *vmConfig.Autostart {
			vmInst.Config.AutoStart = true
		} else {
			vmInst.Config.AutoStart = false
		}
	}
	if vmConfig.AutostartDelay != nil {
		if *vmConfig.AutostartDelay > 3600 {
			vmInst.Config.AutoStartDelay = 3600
		} else {
			vmInst.Config.AutoStartDelay = *vmConfig.AutostartDelay
		}
	}
	if vmConfig.Restart != nil {
		if *vmConfig.Restart {
			vmInst.Config.Restart = true
		} else {
			vmInst.Config.Restart = false
		}
	}
	if vmConfig.RestartDelay != nil {
		vmInst.Config.RestartDelay = *vmConfig.RestartDelay
	}
	if vmConfig.MaxWait != nil {
		vmInst.Config.MaxWait = *vmConfig.MaxWait
	}
}

func updateVMSound(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	if vmConfig.Sound != nil {
		if *vmConfig.Sound {
			vmInst.Config.Sound = true
		} else {
			vmInst.Config.Sound = false
		}
	}
	if vmConfig.SoundIn != nil {
		if !strings.HasPrefix(*vmConfig.SoundIn, "/dev/dsp") {
			return errInvalidSoundDev
		}
		vmInst.Config.SoundIn = *vmConfig.SoundIn
	}
	if vmConfig.SoundOut != nil {
		if !strings.HasPrefix(*vmConfig.SoundOut, "/dev/dsp") {
			return errInvalidSoundDev
		}
		vmInst.Config.SoundOut = *vmConfig.SoundOut
	}

	return nil
}

func updateVMScreen(vmConfig *cirrina.VMConfig, vmInst *vm.VM) {
	if vmConfig.Screen != nil {
		if *vmConfig.Screen {
			vmInst.Config.Screen = true
		} else {
			vmInst.Config.Screen = false
		}
	}
	if vmConfig.ScreenWidth != nil {
		vmInst.Config.ScreenWidth = *vmConfig.ScreenWidth
	}
	if vmConfig.ScreenHeight != nil {
		vmInst.Config.ScreenHeight = *vmConfig.ScreenHeight
	}
}

func updateVMScreenOptions(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	if vmConfig.Vncport != nil {
		if *vmConfig.Vncport != "AUTO" {
			port, err := strconv.Atoi(*vmConfig.Vncport)
			if err != nil || port < 1024 || port > 65535 {
				return errInvalidVncPort
			}
		}
		vmInst.Config.VNCPort = *vmConfig.Vncport
	}
	if vmConfig.Keyboard != nil {
		layoutNames := GetKbdLayoutNames()
		if !util.ContainsStr(layoutNames, *vmConfig.Keyboard) {
			return errInvalidKeyboardLayout
		}
		vmInst.Config.KbdLayout = *vmConfig.Keyboard
	}
	if vmConfig.Tablet != nil {
		if *vmConfig.Tablet {
			vmInst.Config.Tablet = true
		} else {
			vmInst.Config.Tablet = false
		}
	}
	if vmConfig.Vncwait != nil {
		if *vmConfig.Vncwait {
			vmInst.Config.VNCWait = true
		} else {
			vmInst.Config.VNCWait = false
		}
	}

	return nil
}

func updateVMCom1(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	if vmConfig.Com1 != nil {
		if *vmConfig.Com1 {
			vmInst.Config.Com1 = true
		} else {
			vmInst.Config.Com1 = false
		}
	}
	if vmConfig.Com1Dev != nil {
		if *vmConfig.Com1Dev != "AUTO" {
			if !strings.HasPrefix(*vmConfig.Com1Dev, "/dev/nmdm") {
				return errInvalidComDev
			}
		}
		vmInst.Config.Com1Dev = *vmConfig.Com1Dev
	}
	if vmConfig.Com1Log != nil {
		if *vmConfig.Com1Log {
			vmInst.Config.Com1Log = true
		} else {
			vmInst.Config.Com1Log = false
		}
	}
	if vmConfig.Com1Speed != nil {
		vmInst.Config.Com1Speed = *vmConfig.Com1Speed
	}

	return nil
}

func updateVMCom2(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	if vmConfig.Com2 != nil {
		if *vmConfig.Com2 {
			vmInst.Config.Com2 = true
		} else {
			vmInst.Config.Com2 = false
		}
	}
	if vmConfig.Com2Dev != nil {
		if *vmConfig.Com2Dev != "AUTO" {
			if !strings.HasPrefix(*vmConfig.Com2Dev, "/dev/nmdm") {
				return errInvalidComDev
			}
		}
		vmInst.Config.Com2Dev = *vmConfig.Com2Dev
	}
	if vmConfig.Com2Log != nil {
		if *vmConfig.Com2Log {
			vmInst.Config.Com2Log = true
		} else {
			vmInst.Config.Com2Log = false
		}
	}
	if vmConfig.Com2Speed != nil {
		vmInst.Config.Com2Speed = *vmConfig.Com2Speed
	}

	return nil
}

func updateVMCom3(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	if vmConfig.Com3 != nil {
		if *vmConfig.Com3 {
			vmInst.Config.Com3 = true
		} else {
			vmInst.Config.Com3 = false
		}
	}
	if vmConfig.Com3Dev != nil {
		if *vmConfig.Com3Dev != "AUTO" {
			if !strings.HasPrefix(*vmConfig.Com3Dev, "/dev/nmdm") {
				return errInvalidComDev
			}
		}
		vmInst.Config.Com3Dev = *vmConfig.Com3Dev
	}
	if vmConfig.Com3Log != nil {
		if *vmConfig.Com3Log {
			vmInst.Config.Com3Log = true
		} else {
			vmInst.Config.Com3Log = false
		}
	}
	if vmConfig.Com3Speed != nil {
		vmInst.Config.Com3Speed = *vmConfig.Com3Speed
	}

	return nil
}

func updateVMCom4(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	if vmConfig.Com4 != nil {
		if *vmConfig.Com4 {
			vmInst.Config.Com4 = true
		} else {
			vmInst.Config.Com4 = false
		}
	}
	if vmConfig.Com4Dev != nil {
		if *vmConfig.Com4Dev != "AUTO" {
			if !strings.HasPrefix(*vmConfig.Com4Dev, "/dev/nmdm") {
				return errInvalidComDev
			}
		}
		vmInst.Config.Com4Dev = *vmConfig.Com4Dev
	}
	if vmConfig.Com4Log != nil {
		if *vmConfig.Com4Log {
			vmInst.Config.Com4Log = true
		} else {
			vmInst.Config.Com4Log = false
		}
	}
	if vmConfig.Com4Speed != nil {
		vmInst.Config.Com4Speed = *vmConfig.Com4Speed
	}

	return nil
}

func updateVMBasics(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	if vmConfig.Name != nil {
		if !util.ValidVMName(*vmConfig.Name) {
			return errInvalidName
		}
		if _, err := vm.GetByName(*vmConfig.Name); err == nil {
			return errVMDupe
		}
		vmInst.Name = *vmConfig.Name
	}
	if vmConfig.Description != nil {
		vmInst.Description = *vmConfig.Description
	}
	if vmConfig.Cpu != nil {
		vmInst.Config.CPU = *vmConfig.Cpu
	}
	if vmConfig.Mem != nil {
		vmInst.Config.Mem = *vmConfig.Mem
	}

	return nil
}

func (s *server) GetVMID(_ context.Context, vmID *wrapperspb.StringValue) (*cirrina.VMID, error) {
	var vmName string

	if vmID == nil {
		return &cirrina.VMID{}, errInvalidID
	}

	vmName = vmID.GetValue()
	vmInst, err := vm.GetByName(vmName)
	if err != nil {
		return &cirrina.VMID{}, fmt.Errorf("error getting VM: %w", err)
	}

	return &cirrina.VMID{Value: vmInst.ID}, nil
}

func (s *server) GetVMName(_ context.Context, vmID *cirrina.VMID) (*wrapperspb.StringValue, error) {
	vmUUID, err := uuid.Parse(vmID.Value)
	if err != nil {
		return wrapperspb.String(""), fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("GetVMConfig error getting vm", "vm", vmID.Value, "err", err)

		return wrapperspb.String(""), fmt.Errorf("error getting VM: %w", err)
	}

	return wrapperspb.String(vmInst.Name), nil
}

func (s *server) GetVMConfig(_ context.Context, vmID *cirrina.VMID) (*cirrina.VMConfig, error) {
	var pvm cirrina.VMConfig

	vmUUID, err := uuid.Parse(vmID.Value)
	if err != nil {
		return &pvm, fmt.Errorf("error parsing VM ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("GetVMConfig error getting vm", "vm", vmID.Value, "err", err)

		return &pvm, errNotFound
	}
	if vmInst.Name == "" {
		return &pvm, errNotFound
	}
	pvm.Id = vmID.Value
	pvm.Name = &vmInst.Name
	pvm.Description = &vmInst.Description
	pvm.Cpu = &vmInst.Config.CPU
	pvm.Mem = &vmInst.Config.Mem
	pvm.MaxWait = &vmInst.Config.MaxWait
	pvm.Restart = &vmInst.Config.Restart
	pvm.RestartDelay = &vmInst.Config.RestartDelay
	pvm.Screen = &vmInst.Config.Screen
	pvm.ScreenWidth = &vmInst.Config.ScreenWidth
	pvm.ScreenHeight = &vmInst.Config.ScreenHeight
	pvm.Vncwait = &vmInst.Config.VNCWait
	pvm.Vncport = &vmInst.Config.VNCPort
	pvm.Wireguestmem = &vmInst.Config.WireGuestMem
	pvm.Tablet = &vmInst.Config.Tablet
	pvm.Storeuefi = &vmInst.Config.StoreUEFIVars
	pvm.Utc = &vmInst.Config.UTCTime
	pvm.Hostbridge = &vmInst.Config.HostBridge
	pvm.Acpi = &vmInst.Config.ACPI
	pvm.Hlt = &vmInst.Config.UseHLT
	pvm.Eop = &vmInst.Config.ExitOnPause
	pvm.Dpo = &vmInst.Config.DestroyPowerOff
	pvm.Ium = &vmInst.Config.IgnoreUnknownMSR
	pvm.Keyboard = &vmInst.Config.KbdLayout
	pvm.Autostart = &vmInst.Config.AutoStart
	pvm.Sound = &vmInst.Config.Sound
	pvm.SoundIn = &vmInst.Config.SoundIn
	pvm.SoundOut = &vmInst.Config.SoundOut
	pvm.Com1 = &vmInst.Config.Com1
	pvm.Com1Dev = &vmInst.Config.Com1Dev
	pvm.Com2 = &vmInst.Config.Com2
	pvm.Com2Dev = &vmInst.Config.Com2Dev
	pvm.Com3 = &vmInst.Config.Com3
	pvm.Com3Dev = &vmInst.Config.Com3Dev
	pvm.Com4 = &vmInst.Config.Com4
	pvm.Com4Dev = &vmInst.Config.Com4Dev
	pvm.Com1Log = &vmInst.Config.Com1Log
	pvm.Com2Log = &vmInst.Config.Com2Log
	pvm.Com3Log = &vmInst.Config.Com3Log
	pvm.Com4Log = &vmInst.Config.Com4Log
	pvm.Com1Speed = &vmInst.Config.Com1Speed
	pvm.Com2Speed = &vmInst.Config.Com2Speed
	pvm.Com3Speed = &vmInst.Config.Com3Speed
	pvm.Com4Speed = &vmInst.Config.Com4Speed
	pvm.AutostartDelay = &vmInst.Config.AutoStartDelay
	pvm.ExtraArgs = &vmInst.Config.ExtraArgs
	pvm.Debug = &vmInst.Config.Debug
	pvm.DebugWait = &vmInst.Config.DebugWait
	pvm.DebugPort = &vmInst.Config.DebugPort
	pvm.Priority = &vmInst.Config.Priority
	pvm.Protect = &vmInst.Config.Protect.Bool
	pvm.Pcpu = &vmInst.Config.Pcpu
	pvm.Rbps = &vmInst.Config.Rbps
	pvm.Wbps = &vmInst.Config.Wbps
	pvm.Riops = &vmInst.Config.Riops
	pvm.Wiops = &vmInst.Config.Wiops

	return &pvm, nil
}

func (s *server) GetVMs(_ *cirrina.VMsQuery, stream cirrina.VMInfo_GetVMsServer) error {
	var allVMs []*vm.VM
	var pvmID cirrina.VMID

	allVMs = vm.GetAll()
	for e := range allVMs {
		pvmID.Value = allVMs[e].ID
		err := stream.Send(&pvmID)
		if err != nil {
			return fmt.Errorf("error sending to stream: %w", err)
		}
	}

	return nil
}

func (s *server) GetVMState(_ context.Context, vmID *cirrina.VMID) (*cirrina.VMState, error) {
	pvm := cirrina.VMState{}
	vmUUID, err := uuid.Parse(vmID.Value)
	if err != nil {
		return &pvm, fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("GetVMState error getting vm", "vm", vmID.Value, "err", err)

		return &pvm, errNotFound
	}
	if vmInst.Name == "" {
		return &pvm, errNotFound
	}

	switch vmInst.Status {
	case vm.STOPPED:
		pvm.Status = cirrina.VmStatus_STATUS_STOPPED
	case vm.STARTING:
		pvm.Status = cirrina.VmStatus_STATUS_STARTING
		pvm.VncPort = vmInst.VNCPort
		pvm.DebugPort = vmInst.DebugPort
	case vm.RUNNING:
		pvm.Status = cirrina.VmStatus_STATUS_RUNNING
		pvm.VncPort = vmInst.VNCPort
		pvm.DebugPort = vmInst.DebugPort
	case vm.STOPPING:
		pvm.Status = cirrina.VmStatus_STATUS_STOPPING
		pvm.VncPort = vmInst.VNCPort
		pvm.DebugPort = vmInst.DebugPort
	default:
		return &pvm, errInvalidVMState
	}

	return &pvm, nil
}

func (s *server) AddVM(_ context.Context, vmConfig *cirrina.VMConfig) (*cirrina.VMID, error) {
	defaultVMDescription := ""
	var defaultVMCpuCount uint32 = 1
	var defaultVMMemCount uint32 = 128

	if vmConfig.Name == nil {
		return &cirrina.VMID{}, errInvalidName
	}
	if !util.ValidVMName(*vmConfig.Name) {
		return &cirrina.VMID{}, errInvalidName
	}

	if vmConfig.Description == nil {
		vmConfig.Description = &defaultVMDescription
	}
	if vmConfig.Cpu == nil || *vmConfig.Cpu < 1 || *vmConfig.Cpu > 16 {
		vmConfig.Cpu = &defaultVMCpuCount
	}
	if vmConfig.Mem == nil || *vmConfig.Mem < 128 {
		vmConfig.Mem = &defaultVMMemCount
	}
	vmInst, err := vm.Create(*vmConfig.Name, *vmConfig.Description, *vmConfig.Cpu, *vmConfig.Mem)
	if err != nil {
		return &cirrina.VMID{}, fmt.Errorf("error creating VM: %w", err)
	}
	slog.Debug("Created VM", "vm", vmInst.ID)

	return &cirrina.VMID{Value: vmInst.ID}, nil
}

func (s *server) DeleteVM(_ context.Context, vmID *cirrina.VMID) (*cirrina.RequestID, error) {
	vmUUID, err := uuid.Parse(vmID.Value)
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error parsing VM ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("DeleteVM error getting vm", "vm", vmID.Value, "err", err)

		return &cirrina.RequestID{}, fmt.Errorf("error getting VM: %w", err)
	}
	if vmInst.Name == "" {
		return &cirrina.RequestID{}, errNotFound
	}
	pendingReqIDs := requests.PendingReqExists(vmID.Value)
	if len(pendingReqIDs) > 0 {
		return &cirrina.RequestID{}, errReqExists
	}
	if vmInst.Status != vm.STOPPED {
		return &cirrina.RequestID{}, errInvalidVMStateDelete
	}
	newReq, err := requests.CreateVMReq(requests.VMDELETE, vmID.Value)
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error creating request: %w", err)
	}

	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) SetVMISOs(_ context.Context, setISOReq *cirrina.SetISOReq) (*cirrina.ReqBool, error) {
	res := cirrina.ReqBool{}
	res.Success = false

	vmUUID, err := uuid.Parse(setISOReq.Id)
	if err != nil {
		return &res, fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("SetVmISOs error getting vm", "vm", setISOReq.Id, "err", err)

		return &res, errNotFound
	}
	if vmInst.Name == "" {
		return &res, errNotFound
	}

	err = vmInst.AttachIsos(setISOReq.Isoid)
	if err != nil {
		return &res, fmt.Errorf("error attaching ISO: %w", err)
	}
	res.Success = true

	return &res, nil
}

func (s *server) SetVMNics(_ context.Context, setNicReq *cirrina.SetNicReq) (*cirrina.ReqBool, error) {
	var res cirrina.ReqBool
	res.Success = false
	slog.Debug("SetVmNics", "vm", setNicReq.Vmid, "vmnic", setNicReq.Vmnicid)

	vmUUID, err := uuid.Parse(setNicReq.Vmid)
	if err != nil {
		return &res, fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("SetVmNics error getting vm", "vm", setNicReq.Vmid, "err", err)

		return &res, fmt.Errorf("error getting VM ID: %w", err)
	}
	if vmInst.Name == "" {
		return &res, errNotFound
	}

	err = vmInst.SetNics(setNicReq.Vmnicid)
	if err != nil {
		return &res, fmt.Errorf("error setting NICs: %w", err)
	}
	res.Success = true

	return &res, nil
}

func (s *server) SetVMDisks(_ context.Context, setDiskReq *cirrina.SetDiskReq) (*cirrina.ReqBool, error) {
	res := cirrina.ReqBool{}
	res.Success = false
	slog.Debug("SetVmDisks", "vm", setDiskReq.Id, "disk", setDiskReq.Diskid)

	vmUUID, err := uuid.Parse(setDiskReq.Id)
	if err != nil {
		return &res, fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("SetVmDisks error getting vm", "vm", setDiskReq.Id, "err", err)

		return &res, errNotFound
	}
	if vmInst.Name == "" {
		return &res, errNotFound
	}
	err = vmInst.AttachDisks(setDiskReq.Diskid)
	if err != nil {
		return &res, fmt.Errorf("error attaching disk: %w", err)
	}
	res.Success = true

	return &res, nil
}

func (s *server) GetVMISOs(vmID *cirrina.VMID, stream cirrina.VMInfo_GetVMISOsServer) error {
	vmUUID, err := uuid.Parse(vmID.Value)
	if err != nil {
		return fmt.Errorf("error parsing ID: %w", err)
	}

	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("GetVmISOs error getting vm", "vm", vmID.Value, "err", err)

		return fmt.Errorf("error getting VM: %w", err)
	}
	if vmInst.Name == "" {
		return errNotFound
	}

	isos, err := vmInst.GetISOs()
	if err != nil {
		return fmt.Errorf("error getting ISOs: %w", err)
	}
	var isoID cirrina.ISOID

	for _, e := range isos {
		isoID.Value = e.ID
		err := stream.Send(&isoID)
		if err != nil {
			return fmt.Errorf("error sending to stream: %w", err)
		}
	}

	return nil
}

func (s *server) GetVMDisks(vmID *cirrina.VMID, stream cirrina.VMInfo_GetVMDisksServer) error {
	vmUUID, err := uuid.Parse(vmID.Value)
	if err != nil {
		return fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("GetVmDisks error getting vm", "vm", vmID.Value, "err", err)

		return fmt.Errorf("error getting VM: %w", err)
	}
	if vmInst.Name == "" {
		return errNotFound
	}

	disks, err := vmInst.GetDisks()
	if err != nil {
		return fmt.Errorf("error getting disks: %w", err)
	}
	var diskID cirrina.DiskId

	for _, e := range disks {
		diskID.Value = e.ID
		err := stream.Send(&diskID)
		if err != nil {
			return fmt.Errorf("error sending to stream: %w", err)
		}
	}

	return nil
}

func (s *server) StartVM(_ context.Context, vmID *cirrina.VMID) (*cirrina.RequestID, error) {
	vmUUID, err := uuid.Parse(vmID.Value)
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("StartVM error getting vm", "vm", vmID.Value, "err", err)

		return &cirrina.RequestID{}, errNotFound
	}
	if vmInst.Name == "" {
		return &cirrina.RequestID{}, errNotFound
	}
	pendingReqIDs := requests.PendingReqExists(vmID.Value)
	if len(pendingReqIDs) > 0 {
		return &cirrina.RequestID{}, errReqExists
	}
	if vmInst.Status != vm.STOPPED {
		return &cirrina.RequestID{}, errInvalidVMStateStart
	}
	newReq, err := requests.CreateVMReq(requests.VMSTART, vmUUID.String())
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error creating request: %w", err)
	}

	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) StopVM(_ context.Context, vmID *cirrina.VMID) (*cirrina.RequestID, error) {
	vmUUID, err := uuid.Parse(vmID.Value)
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("StopVM error getting vm", "vm", vmID.Value, "err", err)

		return &cirrina.RequestID{}, fmt.Errorf("error getting VM: %w", err)
	}
	if vmInst.Name == "" {
		return &cirrina.RequestID{}, errNotFound
	}
	pendingReqIDs := requests.PendingReqExists(vmID.Value)
	if len(pendingReqIDs) > 0 {
		return &cirrina.RequestID{}, errReqExists
	}
	if vmInst.Status != vm.RUNNING {
		return &cirrina.RequestID{}, errInvalidVMStateStop
	}
	newReq, err := requests.CreateVMReq(requests.VMSTOP, vmID.Value)
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error creating request: %w", err)
	}

	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) GetVMNics(vmID *cirrina.VMID, stream cirrina.VMInfo_GetVMNicsServer) error {
	var pvmnicID cirrina.VmNicId
	vmUUID, err := uuid.Parse(vmID.Value)
	if err != nil {
		return fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("GetVMNics error getting vm", "vm", vmID.Value, "err", err)

		return fmt.Errorf("error getting VM: %w", err)
	}
	if vmInst.Name == "" {
		return errNotFound
	}
	vmNics, err := vmInst.GetNics()
	if err != nil {
		return fmt.Errorf("error getting NICs: %w", err)
	}

	for e := range vmNics {
		pvmnicID.Value = vmNics[e].ID
		err := stream.Send(&pvmnicID)
		if err != nil {
			return fmt.Errorf("error sending to stream: %w", err)
		}
	}

	return nil
}
