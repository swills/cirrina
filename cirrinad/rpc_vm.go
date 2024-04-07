package main

import (
	"context"
	"database/sql"
	"errors"
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

	vmUuid, err := uuid.Parse(rc.Id)
	if err != nil {
		return &re, errors.New("id not specified or invalid")
	}
	vmInst, err := vm.GetById(vmUuid.String())
	if err != nil {
		slog.Error("UpdateVM error getting vm", "vm", rc.Id, "err", err)
		return &re, errors.New("not found")
	}
	if vmInst.Name == "" {
		return &re, errors.New("not found")
	}

	err = updateVmBasics(rc, vmInst)
	if err != nil {
		return &re, err
	}

	err = updateVmCom1(rc, vmInst)
	if err != nil {
		return &re, err
	}
	err = updateVmCom2(rc, vmInst)
	if err != nil {
		return &re, err
	}
	err = updateVmCom3(rc, vmInst)
	if err != nil {
		return &re, err
	}
	err = updateVmCom4(rc, vmInst)
	if err != nil {
		return &re, err
	}

	updateVmScreen(rc, vmInst)
	err = updateVmScreenOptions(rc, vmInst)
	if err != nil {
		return &re, err
	}

	err = updateVmSound(rc, vmInst)
	if err != nil {
		return &re, err
	}

	updateVmStart(rc, vmInst)
	updateVmAdvanced1(rc, vmInst)
	updateVmAdvanced2(rc, vmInst)

	err = updateVmDebug(rc, vmInst)
	if err != nil {
		return &re, err
	}

	updateVmPriorityLimits(rc, vmInst)

	err = vmInst.Save()
	if err != nil {
		return &re, err
	}
	re.Success = true
	return &re, nil
}

func updateVmPriorityLimits(rc *cirrina.VMConfig, vmInst *vm.VM) {
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

func updateVmDebug(rc *cirrina.VMConfig, vmInst *vm.VM) error {
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
			if err != nil {
				return errors.New("invalid debug port")
			}
			if port < 1024 || port > 65535 {
				return errors.New("invalid debug port")
			}
		}
		vmInst.Config.DebugPort = *rc.DebugPort
	}
	return nil
}

func updateVmAdvanced1(rc *cirrina.VMConfig, vmInst *vm.VM) {
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

func updateVmAdvanced2(rc *cirrina.VMConfig, vmInst *vm.VM) {
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

func updateVmStart(rc *cirrina.VMConfig, vmInst *vm.VM) {
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

func updateVmSound(rc *cirrina.VMConfig, vmInst *vm.VM) error {
	if rc.Sound != nil {
		if *rc.Sound {
			vmInst.Config.Sound = true
		} else {
			vmInst.Config.Sound = false
		}
	}
	if rc.SoundIn != nil {
		if !strings.HasPrefix(*rc.SoundIn, "/dev/dsp") {
			return errors.New("invalid sound dev")
		}
		vmInst.Config.SoundIn = *rc.SoundIn
	}
	if rc.SoundOut != nil {
		if !strings.HasPrefix(*rc.SoundOut, "/dev/dsp") {
			return errors.New("invalid sound dev")
		}
		vmInst.Config.SoundOut = *rc.SoundOut
	}
	return nil
}

func updateVmScreen(rc *cirrina.VMConfig, vmInst *vm.VM) {

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

func updateVmScreenOptions(rc *cirrina.VMConfig, vmInst *vm.VM) error {
	if rc.Vncport != nil {
		if *rc.Vncport != "AUTO" {
			port, err := strconv.Atoi(*rc.Vncport)
			if err != nil {
				return errors.New("invalid vnc port")
			}
			if port < 1024 || port > 65535 {
				return errors.New("invalid vnc port")
			}
		}
		vmInst.Config.VNCPort = *rc.Vncport
	}
	if rc.Keyboard != nil {
		layoutNames := GetKbdLayoutNames()
		if !util.ContainsStr(layoutNames, *rc.Keyboard) {
			return errors.New("invalid keyboard layout")
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

func updateVmCom1(rc *cirrina.VMConfig, vmInst *vm.VM) error {
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
				return errors.New("invalid com dev")
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

func updateVmCom2(rc *cirrina.VMConfig, vmInst *vm.VM) error {
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
				return errors.New("invalid com dev")
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

func updateVmCom3(rc *cirrina.VMConfig, vmInst *vm.VM) error {
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
				return errors.New("invalid com dev")
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

func updateVmCom4(rc *cirrina.VMConfig, vmInst *vm.VM) error {
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
				return errors.New("invalid com dev")
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

func updateVmBasics(rc *cirrina.VMConfig, vmInst *vm.VM) error {
	if rc.Name != nil {
		if !util.ValidVmName(*rc.Name) {
			return errors.New("invalid name")
		}
		if _, err := vm.GetByName(*rc.Name); err == nil {
			return fmt.Errorf("%v already exists", *rc.Name)
		}
		vmInst.Name = *rc.Name
	}
	if rc.Description != nil {
		vmInst.Description = *rc.Description
	}
	if rc.Cpu != nil {
		vmInst.Config.Cpu = *rc.Cpu
	}
	if rc.Mem != nil {
		vmInst.Config.Mem = *rc.Mem
	}
	return nil
}

func (s *server) GetVmId(_ context.Context, v *wrapperspb.StringValue) (*cirrina.VMID, error) {
	var vmName string

	if v == nil {
		return &cirrina.VMID{}, errors.New("name not specified or invalid")
	}

	vmName = v.GetValue()
	vmInst, err := vm.GetByName(vmName)
	if err != nil {
		return &cirrina.VMID{}, errors.New("VM not found")
	}

	return &cirrina.VMID{Value: vmInst.ID}, nil
}

func (s *server) GetVmName(_ context.Context, v *cirrina.VMID) (*wrapperspb.StringValue, error) {
	vmUuid, err := uuid.Parse(v.Value)
	if err != nil {
		return wrapperspb.String(""), errors.New("id not specified or invalid")
	}
	vmInst, err := vm.GetById(vmUuid.String())
	if err != nil {
		slog.Error("GetVMConfig error getting vm", "vm", v.Value, "err", err)
		return wrapperspb.String(""), errors.New("VM not found")
	}
	return wrapperspb.String(vmInst.Name), nil
}

func (s *server) GetVMConfig(_ context.Context, v *cirrina.VMID) (*cirrina.VMConfig, error) {
	var pvm cirrina.VMConfig

	vmUuid, err := uuid.Parse(v.Value)
	if err != nil {
		return &pvm, errors.New("id not specified or invalid")
	}
	vmInst, err := vm.GetById(vmUuid.String())
	if err != nil {
		slog.Error("GetVMConfig error getting vm", "vm", v.Value, "err", err)
		return &pvm, errors.New("not found")
	}
	if vmInst.Name == "" {
		return &pvm, errors.New("not found")
	}
	pvm.Id = v.Value
	pvm.Name = &vmInst.Name
	pvm.Description = &vmInst.Description
	pvm.Cpu = &vmInst.Config.Cpu
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
	var pvmId cirrina.VMID

	allVMs = vm.GetAll()
	for e := range allVMs {
		pvmId.Value = allVMs[e].ID
		err := stream.Send(&pvmId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *server) GetVMState(_ context.Context, p *cirrina.VMID) (*cirrina.VMState, error) {
	pvm := cirrina.VMState{}
	vmUuid, err := uuid.Parse(p.Value)
	if err != nil {
		return &pvm, errors.New("id not specified or invalid")
	}
	vmInst, err := vm.GetById(vmUuid.String())
	if err != nil {
		slog.Error("GetVMState error getting vm", "vm", p.Value, "err", err)
		return &pvm, errors.New("not found")
	}
	if vmInst.Name == "" {
		return &pvm, errors.New("not found")
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
		return &pvm, errors.New("unknown VM state")
	}
	return &pvm, nil
}

func (s *server) AddVM(_ context.Context, v *cirrina.VMConfig) (*cirrina.VMID, error) {
	defaultVmDescription := ""
	var defaultVmCpuCount uint32 = 1
	var defaultVmMemCount uint32 = 128

	if v.Name == nil {
		return &cirrina.VMID{}, errors.New("name not specified")
	}
	if !util.ValidVmName(*v.Name) {
		return &cirrina.VMID{}, errors.New("invalid name")
	}

	if v.Description == nil {
		v.Description = &defaultVmDescription
	}
	if v.Cpu == nil || *v.Cpu < 1 || *v.Cpu > 16 {
		v.Cpu = &defaultVmCpuCount
	}
	if v.Mem == nil || *v.Mem < 128 {
		v.Mem = &defaultVmMemCount
	}
	vmInst, err := vm.Create(*v.Name, *v.Description, *v.Cpu, *v.Mem)
	if err != nil {
		return &cirrina.VMID{}, err
	}
	slog.Debug("Created VM", "vm", vmInst.ID)
	if err != nil {
		return &cirrina.VMID{}, err
	}
	return &cirrina.VMID{Value: vmInst.ID}, nil
}

func (s *server) DeleteVM(_ context.Context, v *cirrina.VMID) (*cirrina.RequestID, error) {
	vmUuid, err := uuid.Parse(v.Value)
	if err != nil {
		return &cirrina.RequestID{}, errors.New("id not specified or invalid")
	}
	vmInst, err := vm.GetById(vmUuid.String())
	if err != nil {
		slog.Error("DeleteVM error getting vm", "vm", v.Value, "err", err)
		return &cirrina.RequestID{}, errors.New("not found")
	}
	if vmInst.Name == "" {
		return &cirrina.RequestID{}, errors.New("not found")
	}
	pendingReqIds := requests.PendingReqExists(v.Value)
	if len(pendingReqIds) > 0 {
		return &cirrina.RequestID{}, fmt.Errorf("pending request for %v already exists", v.Value)
	}
	if vmInst.Status != vm.STOPPED {
		return &cirrina.RequestID{}, errors.New("vm must be stopped before deleting")
	}
	newReq, err := requests.CreateVmReq(requests.VMDELETE, v.Value)
	if err != nil {
		return &cirrina.RequestID{}, err
	}
	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) SetVmISOs(_ context.Context, sr *cirrina.SetISOReq) (*cirrina.ReqBool, error) {
	re := cirrina.ReqBool{}
	re.Success = false

	vmUuid, err := uuid.Parse(sr.Id)
	if err != nil {
		return &re, errors.New("id not specified or invalid")
	}
	vmInst, err := vm.GetById(vmUuid.String())
	if err != nil {
		slog.Error("SetVmISOs error getting vm", "vm", sr.Id, "err", err)
		return &re, errors.New("not found")
	}
	if vmInst.Name == "" {
		return &re, errors.New("not found")
	}

	err = vmInst.AttachIsos(sr.Isoid)
	if err != nil {
		return &re, err
	}
	re.Success = true
	return &re, nil
}

func (s *server) SetVmNics(_ context.Context, sn *cirrina.SetNicReq) (*cirrina.ReqBool, error) {
	var re cirrina.ReqBool
	re.Success = false
	slog.Debug("SetVmNics", "vm", sn.Vmid, "vmnic", sn.Vmnicid)

	vmUuid, err := uuid.Parse(sn.Vmid)
	if err != nil {
		return &re, errors.New("id not specified or invalid")
	}
	vmInst, err := vm.GetById(vmUuid.String())
	if err != nil {
		slog.Error("SetVmNics error getting vm", "vm", sn.Vmid, "err", err)
		return &re, errors.New("not found")
	}
	if vmInst.Name == "" {
		return &re, errors.New("not found")
	}

	err = vmInst.SetNics(sn.Vmnicid)
	if err != nil {
		return &re, err
	}
	re.Success = true
	return &re, nil
}

func (s *server) SetVmDisks(_ context.Context, sr *cirrina.SetDiskReq) (*cirrina.ReqBool, error) {
	re := cirrina.ReqBool{}
	re.Success = false
	slog.Debug("SetVmDisks", "vm", sr.Id, "disk", sr.Diskid)

	vmUuid, err := uuid.Parse(sr.Id)
	if err != nil {
		return &re, errors.New("id not specified or invalid")
	}
	vmInst, err := vm.GetById(vmUuid.String())
	if err != nil {
		slog.Error("SetVmDisks error getting vm", "vm", sr.Id, "err", err)
		return &re, errors.New("not found")
	}
	if vmInst.Name == "" {
		return &re, errors.New("not found")
	}
	err = vmInst.AttachDisks(sr.Diskid)
	if err != nil {
		return &re, err
	}
	re.Success = true
	return &re, nil
}

func (s *server) GetVmISOs(v *cirrina.VMID, stream cirrina.VMInfo_GetVmISOsServer) error {
	vmUuid, err := uuid.Parse(v.Value)
	if err != nil {
		return errors.New("id not specified or invalid")
	}

	vmInst, err := vm.GetById(vmUuid.String())
	if err != nil {
		slog.Error("GetVmISOs error getting vm", "vm", v.Value, "err", err)
		return errors.New("not found")
	}
	if vmInst.Name == "" {
		return errors.New("not found")
	}

	isos, err := vmInst.GetISOs()
	if err != nil {
		return err
	}
	var isoId cirrina.ISOID

	for _, e := range isos {
		isoId.Value = e.ID
		err := stream.Send(&isoId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *server) GetVmDisks(v *cirrina.VMID, stream cirrina.VMInfo_GetVmDisksServer) error {
	vmUuid, err := uuid.Parse(v.Value)
	if err != nil {
		return errors.New("id not specified or invalid")
	}
	vmInst, err := vm.GetById(vmUuid.String())
	if err != nil {
		slog.Error("GetVmDisks error getting vm", "vm", v.Value, "err", err)
		return errors.New("not found")
	}
	if vmInst.Name == "" {
		return errors.New("not found")
	}

	disks, err := vmInst.GetDisks()
	if err != nil {
		return err
	}
	var diskId cirrina.DiskId

	for _, e := range disks {
		diskId.Value = e.ID
		err := stream.Send(&diskId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *server) StartVM(_ context.Context, v *cirrina.VMID) (*cirrina.RequestID, error) {
	vmUuid, err := uuid.Parse(v.Value)
	if err != nil {
		return &cirrina.RequestID{}, errors.New("id not specified or invalid")
	}
	vmInst, err := vm.GetById(vmUuid.String())
	if err != nil {
		slog.Error("StartVM error getting vm", "vm", v.Value, "err", err)
		return &cirrina.RequestID{}, errors.New("not found")
	}
	if vmInst.Name == "" {
		return &cirrina.RequestID{}, errors.New("not found")
	}
	pendingReqIds := requests.PendingReqExists(v.Value)
	if len(pendingReqIds) > 0 {
		return &cirrina.RequestID{}, fmt.Errorf("pending request for %v already exists", v.Value)
	}
	if vmInst.Status != vm.STOPPED {
		return &cirrina.RequestID{}, errors.New("vm must be stopped before starting")
	}
	newReq, err := requests.CreateVmReq(requests.VMSTART, vmUuid.String())
	if err != nil {
		return &cirrina.RequestID{}, err
	}
	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) StopVM(_ context.Context, v *cirrina.VMID) (*cirrina.RequestID, error) {
	vmUuid, err := uuid.Parse(v.Value)
	if err != nil {
		return &cirrina.RequestID{}, errors.New("id not specified or invalid")
	}
	vmInst, err := vm.GetById(vmUuid.String())
	if err != nil {
		slog.Error("StopVM error getting vm", "vm", v.Value, "err", err)
		return &cirrina.RequestID{}, errors.New("not found")
	}
	if vmInst.Name == "" {
		return &cirrina.RequestID{}, errors.New("not found")
	}
	pendingReqIds := requests.PendingReqExists(v.Value)
	if len(pendingReqIds) > 0 {
		return &cirrina.RequestID{}, fmt.Errorf("pending request for %v already exists", v.Value)
	}
	if vmInst.Status != vm.RUNNING {
		return &cirrina.RequestID{}, errors.New("vm must be running before stopping")
	}
	newReq, err := requests.CreateVmReq(requests.VMSTOP, v.Value)
	if err != nil {
		return &cirrina.RequestID{}, err
	}
	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) GetVmNics(v *cirrina.VMID, stream cirrina.VMInfo_GetVmNicsServer) error {
	var pvmnicId cirrina.VmNicId
	vmUuid, err := uuid.Parse(v.Value)
	if err != nil {
		return errors.New("id not specified or invalid")
	}
	vmInst, err := vm.GetById(vmUuid.String())
	if err != nil {
		slog.Error("GetVmNics error getting vm", "vm", v.Value, "err", err)
		return errors.New("not found")
	}
	if vmInst.Name == "" {
		return errors.New("not found")
	}
	vmNics, err := vmInst.GetNics()
	if err != nil {
		return err
	}

	for e := range vmNics {
		pvmnicId.Value = vmNics[e].ID
		err := stream.Send(&pvmnicId)
		if err != nil {
			return err
		}
	}

	return nil
}
