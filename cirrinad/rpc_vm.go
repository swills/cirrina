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

func (s *server) UpdateVM(_ context.Context, rc *cirrina.VMConfig) (*cirrina.ReqBool, error) {
	re := cirrina.ReqBool{}
	re.Success = false

	vmUUID, err := uuid.Parse(rc.Id)
	if err != nil {
		return &re, fmt.Errorf("error parsing VM ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("UpdateVM error getting vm", "vm", rc.Id, "err", err)

		return &re, errNotFound
	}
	if vmInst.Name == "" {
		return &re, errNotFound
	}

	err = updateVMBasics(rc, vmInst)
	if err != nil {
		return &re, err
	}

	err = updateVMCom1(rc, vmInst)
	if err != nil {
		return &re, err
	}
	err = updateVMCom2(rc, vmInst)
	if err != nil {
		return &re, err
	}
	err = updateVMCom3(rc, vmInst)
	if err != nil {
		return &re, err
	}
	err = updateVMCom4(rc, vmInst)
	if err != nil {
		return &re, err
	}

	updateVMScreen(rc, vmInst)
	err = updateVMScreenOptions(rc, vmInst)
	if err != nil {
		return &re, err
	}

	err = updateVMSound(rc, vmInst)
	if err != nil {
		return &re, err
	}

	updateVMStart(rc, vmInst)
	updateVMAdvanced1(rc, vmInst)
	updateVMAdvanced2(rc, vmInst)

	err = updateVMDebug(rc, vmInst)
	if err != nil {
		return &re, err
	}

	updateVMPriorityLimits(rc, vmInst)

	err = vmInst.Save()
	if err != nil {
		return &re, fmt.Errorf("error saving VM: %w", err)
	}
	re.Success = true

	return &re, nil
}

func updateVMPriorityLimits(rc *cirrina.VMConfig, vmInst *vm.VM) {
	if rc.Priority != nil {
		vmInst.Config.Priority = *rc.Priority
	}
	if rc.Protect != nil {
		vmInst.Config.Protect = sql.NullBool{Bool: *rc.Protect, Valid: true}
	}
	if rc.Pcpu != nil {
		vmInst.Config.Pcpu = *rc.Pcpu
	}
	if rc.Rbps != nil {
		vmInst.Config.Rbps = *rc.Rbps
	}
	if rc.Wbps != nil {
		vmInst.Config.Wbps = *rc.Wbps
	}
	if rc.Riops != nil {
		vmInst.Config.Riops = *rc.Riops
	}
	if rc.Wiops != nil {
		vmInst.Config.Wiops = *rc.Wiops
	}
}

func updateVMDebug(rc *cirrina.VMConfig, vmInst *vm.VM) error {
	if rc.Debug != nil {
		if *rc.Debug {
			vmInst.Config.Debug = true
		} else {
			vmInst.Config.Debug = false
		}
	}
	if rc.DebugWait != nil {
		if *rc.DebugWait {
			vmInst.Config.DebugWait = true
		} else {
			vmInst.Config.DebugWait = false
		}
	}
	if rc.DebugPort != nil {
		if *rc.DebugPort != "AUTO" {
			port, err := strconv.Atoi(*rc.DebugPort)
			if err != nil || port < 1024 || port > 65535 {
				return errInvalidDebugPort
			}
		}
		vmInst.Config.DebugPort = *rc.DebugPort
	}

	return nil
}

func updateVMAdvanced1(rc *cirrina.VMConfig, vmInst *vm.VM) {
	if rc.Hostbridge != nil {
		if *rc.Hostbridge {
			vmInst.Config.HostBridge = true
		} else {
			vmInst.Config.HostBridge = false
		}
	}
	if rc.Acpi != nil {
		if *rc.Acpi {
			vmInst.Config.ACPI = true
		} else {
			vmInst.Config.ACPI = false
		}
	}
	if rc.Storeuefi != nil {
		if *rc.Storeuefi {
			vmInst.Config.StoreUEFIVars = true
		} else {
			vmInst.Config.StoreUEFIVars = false
		}
	}
	if rc.Utc != nil {
		if *rc.Utc {
			vmInst.Config.UTCTime = true
		} else {
			vmInst.Config.UTCTime = false
		}
	}
	if rc.Wireguestmem != nil {
		if *rc.Wireguestmem {
			vmInst.Config.WireGuestMem = true
		} else {
			vmInst.Config.WireGuestMem = false
		}
	}
}

func updateVMAdvanced2(rc *cirrina.VMConfig, vmInst *vm.VM) {
	if rc.Dpo != nil {
		if *rc.Dpo {
			vmInst.Config.DestroyPowerOff = true
		} else {
			vmInst.Config.DestroyPowerOff = false
		}
	}
	if rc.Eop != nil {
		if *rc.Eop {
			vmInst.Config.ExitOnPause = true
		} else {
			vmInst.Config.ExitOnPause = false
		}
	}
	if rc.Ium != nil {
		if *rc.Ium {
			vmInst.Config.IgnoreUnknownMSR = true
		} else {
			vmInst.Config.IgnoreUnknownMSR = false
		}
	}
	if rc.Hlt != nil {
		if *rc.Hlt {
			vmInst.Config.UseHLT = true
		} else {
			vmInst.Config.UseHLT = false
		}
	}
	// TODO -- potential security issue, should it be removed?
	if rc.ExtraArgs != nil {
		vmInst.Config.ExtraArgs = *rc.ExtraArgs
	}
}

func updateVMStart(rc *cirrina.VMConfig, vmInst *vm.VM) {
	if rc.Autostart != nil {
		if *rc.Autostart {
			vmInst.Config.AutoStart = true
		} else {
			vmInst.Config.AutoStart = false
		}
	}
	if rc.AutostartDelay != nil {
		if *rc.AutostartDelay > 3600 {
			vmInst.Config.AutoStartDelay = 3600
		} else {
			vmInst.Config.AutoStartDelay = *rc.AutostartDelay
		}
	}
	if rc.Restart != nil {
		if *rc.Restart {
			vmInst.Config.Restart = true
		} else {
			vmInst.Config.Restart = false
		}
	}
	if rc.RestartDelay != nil {
		vmInst.Config.RestartDelay = *rc.RestartDelay
	}
	if rc.MaxWait != nil {
		vmInst.Config.MaxWait = *rc.MaxWait
	}
}

func updateVMSound(rc *cirrina.VMConfig, vmInst *vm.VM) error {
	if rc.Sound != nil {
		if *rc.Sound {
			vmInst.Config.Sound = true
		} else {
			vmInst.Config.Sound = false
		}
	}
	if rc.SoundIn != nil {
		if !strings.HasPrefix(*rc.SoundIn, "/dev/dsp") {
			return errInvalidSoundDev
		}
		vmInst.Config.SoundIn = *rc.SoundIn
	}
	if rc.SoundOut != nil {
		if !strings.HasPrefix(*rc.SoundOut, "/dev/dsp") {
			return errInvalidSoundDev
		}
		vmInst.Config.SoundOut = *rc.SoundOut
	}

	return nil
}

func updateVMScreen(rc *cirrina.VMConfig, vmInst *vm.VM) {
	if rc.Screen != nil {
		if *rc.Screen {
			vmInst.Config.Screen = true
		} else {
			vmInst.Config.Screen = false
		}
	}
	if rc.ScreenWidth != nil {
		vmInst.Config.ScreenWidth = *rc.ScreenWidth
	}
	if rc.ScreenHeight != nil {
		vmInst.Config.ScreenHeight = *rc.ScreenHeight
	}
}

func updateVMScreenOptions(rc *cirrina.VMConfig, vmInst *vm.VM) error {
	if rc.Vncport != nil {
		if *rc.Vncport != "AUTO" {
			port, err := strconv.Atoi(*rc.Vncport)
			if err != nil || port < 1024 || port > 65535 {
				return errInvalidVncPort
			}
		}
		vmInst.Config.VNCPort = *rc.Vncport
	}
	if rc.Keyboard != nil {
		layoutNames := GetKbdLayoutNames()
		if !util.ContainsStr(layoutNames, *rc.Keyboard) {
			return errInvalidKeyboardLayout
		}
		vmInst.Config.KbdLayout = *rc.Keyboard
	}
	if rc.Tablet != nil {
		if *rc.Tablet {
			vmInst.Config.Tablet = true
		} else {
			vmInst.Config.Tablet = false
		}
	}
	if rc.Vncwait != nil {
		if *rc.Vncwait {
			vmInst.Config.VNCWait = true
		} else {
			vmInst.Config.VNCWait = false
		}
	}

	return nil
}

func updateVMCom1(rc *cirrina.VMConfig, vmInst *vm.VM) error {
	if rc.Com1 != nil {
		if *rc.Com1 {
			vmInst.Config.Com1 = true
		} else {
			vmInst.Config.Com1 = false
		}
	}
	if rc.Com1Dev != nil {
		if *rc.Com1Dev != "AUTO" {
			if !strings.HasPrefix(*rc.Com1Dev, "/dev/nmdm") {
				return errInvalidComDev
			}
		}
		vmInst.Config.Com1Dev = *rc.Com1Dev
	}
	if rc.Com1Log != nil {
		if *rc.Com1Log {
			vmInst.Config.Com1Log = true
		} else {
			vmInst.Config.Com1Log = false
		}
	}
	if rc.Com1Speed != nil {
		vmInst.Config.Com1Speed = *rc.Com1Speed
	}

	return nil
}

func updateVMCom2(rc *cirrina.VMConfig, vmInst *vm.VM) error {
	if rc.Com2 != nil {
		if *rc.Com2 {
			vmInst.Config.Com2 = true
		} else {
			vmInst.Config.Com2 = false
		}
	}
	if rc.Com2Dev != nil {
		if *rc.Com2Dev != "AUTO" {
			if !strings.HasPrefix(*rc.Com2Dev, "/dev/nmdm") {
				return errInvalidComDev
			}
		}
		vmInst.Config.Com2Dev = *rc.Com2Dev
	}
	if rc.Com2Log != nil {
		if *rc.Com2Log {
			vmInst.Config.Com2Log = true
		} else {
			vmInst.Config.Com2Log = false
		}
	}
	if rc.Com2Speed != nil {
		vmInst.Config.Com2Speed = *rc.Com2Speed
	}

	return nil
}

func updateVMCom3(rc *cirrina.VMConfig, vmInst *vm.VM) error {
	if rc.Com3 != nil {
		if *rc.Com3 {
			vmInst.Config.Com3 = true
		} else {
			vmInst.Config.Com3 = false
		}
	}
	if rc.Com3Dev != nil {
		if *rc.Com3Dev != "AUTO" {
			if !strings.HasPrefix(*rc.Com3Dev, "/dev/nmdm") {
				return errInvalidComDev
			}
		}
		vmInst.Config.Com3Dev = *rc.Com3Dev
	}
	if rc.Com3Log != nil {
		if *rc.Com3Log {
			vmInst.Config.Com3Log = true
		} else {
			vmInst.Config.Com3Log = false
		}
	}
	if rc.Com3Speed != nil {
		vmInst.Config.Com3Speed = *rc.Com3Speed
	}

	return nil
}

func updateVMCom4(rc *cirrina.VMConfig, vmInst *vm.VM) error {
	if rc.Com4 != nil {
		if *rc.Com4 {
			vmInst.Config.Com4 = true
		} else {
			vmInst.Config.Com4 = false
		}
	}
	if rc.Com4Dev != nil {
		if *rc.Com4Dev != "AUTO" {
			if !strings.HasPrefix(*rc.Com4Dev, "/dev/nmdm") {
				return errInvalidComDev
			}
		}
		vmInst.Config.Com4Dev = *rc.Com4Dev
	}
	if rc.Com4Log != nil {
		if *rc.Com4Log {
			vmInst.Config.Com4Log = true
		} else {
			vmInst.Config.Com4Log = false
		}
	}
	if rc.Com4Speed != nil {
		vmInst.Config.Com4Speed = *rc.Com4Speed
	}

	return nil
}

func updateVMBasics(rc *cirrina.VMConfig, vmInst *vm.VM) error {
	if rc.Name != nil {
		if !util.ValidVMName(*rc.Name) {
			return errInvalidName
		}
		if _, err := vm.GetByName(*rc.Name); err == nil {
			return errVMDupe
		}
		vmInst.Name = *rc.Name
	}
	if rc.Description != nil {
		vmInst.Description = *rc.Description
	}
	if rc.Cpu != nil {
		vmInst.Config.CPU = *rc.Cpu
	}
	if rc.Mem != nil {
		vmInst.Config.Mem = *rc.Mem
	}

	return nil
}

func (s *server) GetVMID(_ context.Context, v *wrapperspb.StringValue) (*cirrina.VMID, error) {
	var vmName string

	if v == nil {
		return &cirrina.VMID{}, errInvalidID
	}

	vmName = v.GetValue()
	vmInst, err := vm.GetByName(vmName)
	if err != nil {
		return &cirrina.VMID{}, fmt.Errorf("error getting VM: %w", err)
	}

	return &cirrina.VMID{Value: vmInst.ID}, nil
}

func (s *server) GetVMName(_ context.Context, v *cirrina.VMID) (*wrapperspb.StringValue, error) {
	vmUUID, err := uuid.Parse(v.Value)
	if err != nil {
		return wrapperspb.String(""), fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("GetVMConfig error getting vm", "vm", v.Value, "err", err)

		return wrapperspb.String(""), fmt.Errorf("error getting VM: %w", err)
	}

	return wrapperspb.String(vmInst.Name), nil
}

func (s *server) GetVMConfig(_ context.Context, v *cirrina.VMID) (*cirrina.VMConfig, error) {
	var pvm cirrina.VMConfig

	vmUUID, err := uuid.Parse(v.Value)
	if err != nil {
		return &pvm, fmt.Errorf("error parsing VM ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("GetVMConfig error getting vm", "vm", v.Value, "err", err)

		return &pvm, errNotFound
	}
	if vmInst.Name == "" {
		return &pvm, errNotFound
	}
	pvm.Id = v.Value
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

func (s *server) GetVMState(_ context.Context, p *cirrina.VMID) (*cirrina.VMState, error) {
	pvm := cirrina.VMState{}
	vmUUID, err := uuid.Parse(p.Value)
	if err != nil {
		return &pvm, fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("GetVMState error getting vm", "vm", p.Value, "err", err)

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

func (s *server) AddVM(_ context.Context, v *cirrina.VMConfig) (*cirrina.VMID, error) {
	defaultVMDescription := ""
	var defaultVMCpuCount uint32 = 1
	var defaultVMMemCount uint32 = 128

	if v.Name == nil {
		return &cirrina.VMID{}, errInvalidName
	}
	if !util.ValidVMName(*v.Name) {
		return &cirrina.VMID{}, errInvalidName
	}

	if v.Description == nil {
		v.Description = &defaultVMDescription
	}
	if v.Cpu == nil || *v.Cpu < 1 || *v.Cpu > 16 {
		v.Cpu = &defaultVMCpuCount
	}
	if v.Mem == nil || *v.Mem < 128 {
		v.Mem = &defaultVMMemCount
	}
	vmInst, err := vm.Create(*v.Name, *v.Description, *v.Cpu, *v.Mem)
	if err != nil {
		return &cirrina.VMID{}, fmt.Errorf("error creating VM: %w", err)
	}
	slog.Debug("Created VM", "vm", vmInst.ID)

	return &cirrina.VMID{Value: vmInst.ID}, nil
}

func (s *server) DeleteVM(_ context.Context, v *cirrina.VMID) (*cirrina.RequestID, error) {
	vmUUID, err := uuid.Parse(v.Value)
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error parsing VM ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("DeleteVM error getting vm", "vm", v.Value, "err", err)

		return &cirrina.RequestID{}, fmt.Errorf("error getting VM: %w", err)
	}
	if vmInst.Name == "" {
		return &cirrina.RequestID{}, errNotFound
	}
	pendingReqIds := requests.PendingReqExists(v.Value)
	if len(pendingReqIds) > 0 {
		return &cirrina.RequestID{}, errReqExists
	}
	if vmInst.Status != vm.STOPPED {
		return &cirrina.RequestID{}, errInvalidVMStateDelete
	}
	newReq, err := requests.CreateVMReq(requests.VMDELETE, v.Value)
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error creating request: %w", err)
	}

	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) SetVMISOs(_ context.Context, sr *cirrina.SetISOReq) (*cirrina.ReqBool, error) {
	re := cirrina.ReqBool{}
	re.Success = false

	vmUUID, err := uuid.Parse(sr.Id)
	if err != nil {
		return &re, fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("SetVmISOs error getting vm", "vm", sr.Id, "err", err)

		return &re, errNotFound
	}
	if vmInst.Name == "" {
		return &re, errNotFound
	}

	err = vmInst.AttachIsos(sr.Isoid)
	if err != nil {
		return &re, fmt.Errorf("error attaching ISO: %w", err)
	}
	re.Success = true

	return &re, nil
}

func (s *server) SetVMNics(_ context.Context, sn *cirrina.SetNicReq) (*cirrina.ReqBool, error) {
	var re cirrina.ReqBool
	re.Success = false
	slog.Debug("SetVmNics", "vm", sn.Vmid, "vmnic", sn.Vmnicid)

	vmUUID, err := uuid.Parse(sn.Vmid)
	if err != nil {
		return &re, fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("SetVmNics error getting vm", "vm", sn.Vmid, "err", err)

		return &re, fmt.Errorf("error getting VM ID: %w", err)
	}
	if vmInst.Name == "" {
		return &re, errNotFound
	}

	err = vmInst.SetNics(sn.Vmnicid)
	if err != nil {
		return &re, fmt.Errorf("error setting NICs: %w", err)
	}
	re.Success = true

	return &re, nil
}

func (s *server) SetVMDisks(_ context.Context, sr *cirrina.SetDiskReq) (*cirrina.ReqBool, error) {
	re := cirrina.ReqBool{}
	re.Success = false
	slog.Debug("SetVmDisks", "vm", sr.Id, "disk", sr.Diskid)

	vmUUID, err := uuid.Parse(sr.Id)
	if err != nil {
		return &re, fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("SetVmDisks error getting vm", "vm", sr.Id, "err", err)

		return &re, errNotFound
	}
	if vmInst.Name == "" {
		return &re, errNotFound
	}
	err = vmInst.AttachDisks(sr.Diskid)
	if err != nil {
		return &re, fmt.Errorf("error attaching disk: %w", err)
	}
	re.Success = true

	return &re, nil
}

func (s *server) GetVMISOs(v *cirrina.VMID, stream cirrina.VMInfo_GetVMISOsServer) error {
	vmUUID, err := uuid.Parse(v.Value)
	if err != nil {
		return fmt.Errorf("error parsing ID: %w", err)
	}

	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("GetVmISOs error getting vm", "vm", v.Value, "err", err)

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

func (s *server) GetVMDisks(v *cirrina.VMID, stream cirrina.VMInfo_GetVMDisksServer) error {
	vmUUID, err := uuid.Parse(v.Value)
	if err != nil {
		return fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("GetVmDisks error getting vm", "vm", v.Value, "err", err)

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

func (s *server) StartVM(_ context.Context, v *cirrina.VMID) (*cirrina.RequestID, error) {
	vmUUID, err := uuid.Parse(v.Value)
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("StartVM error getting vm", "vm", v.Value, "err", err)

		return &cirrina.RequestID{}, errNotFound
	}
	if vmInst.Name == "" {
		return &cirrina.RequestID{}, errNotFound
	}
	pendingReqIds := requests.PendingReqExists(v.Value)
	if len(pendingReqIds) > 0 {
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

func (s *server) StopVM(_ context.Context, v *cirrina.VMID) (*cirrina.RequestID, error) {
	vmUUID, err := uuid.Parse(v.Value)
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("StopVM error getting vm", "vm", v.Value, "err", err)

		return &cirrina.RequestID{}, fmt.Errorf("error getting VM: %w", err)
	}
	if vmInst.Name == "" {
		return &cirrina.RequestID{}, errNotFound
	}
	pendingReqIds := requests.PendingReqExists(v.Value)
	if len(pendingReqIds) > 0 {
		return &cirrina.RequestID{}, errReqExists
	}
	if vmInst.Status != vm.RUNNING {
		return &cirrina.RequestID{}, errInvalidVMStateStop
	}
	newReq, err := requests.CreateVMReq(requests.VMSTOP, v.Value)
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error creating request: %w", err)
	}

	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) GetVMNics(v *cirrina.VMID, stream cirrina.VMInfo_GetVMNicsServer) error {
	var pvmnicID cirrina.VmNicId
	vmUUID, err := uuid.Parse(v.Value)
	if err != nil {
		return fmt.Errorf("error parsing ID: %w", err)
	}
	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("GetVMNics error getting vm", "vm", v.Value, "err", err)

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
