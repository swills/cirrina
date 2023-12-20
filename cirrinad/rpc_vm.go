package main

import (
	"cirrina/cirrina"
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"log/slog"
	"strconv"
	"strings"
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

	reflect := rc.ProtoReflect()

	if isOptionPassed(reflect, "name") {
		if !util.ValidVmName(*rc.Name) {
			return &re, errors.New("invalid name")
		}
		if _, err := vm.GetByName(*rc.Name); err == nil {
			return &re, errors.New(fmt.Sprintf("%v already exists", *rc.Name))
		}
		vmInst.Name = *rc.Name
	}
	if isOptionPassed(reflect, "description") {
		vmInst.Description = *rc.Description
	}
	if isOptionPassed(reflect, "cpu") {
		vmInst.Config.Cpu = *rc.Cpu
	}
	if isOptionPassed(reflect, "mem") {
		vmInst.Config.Mem = *rc.Mem
	}
	if isOptionPassed(reflect, "priority") {
		vmInst.Config.Priority = *rc.Priority
	}
	if isOptionPassed(reflect, "protect") {
		vmInst.Config.Protect = sql.NullBool{Bool: *rc.Protect, Valid: true}
	}
	if isOptionPassed(reflect, "vncwait") {
		if *rc.Vncwait == true {
			vmInst.Config.VNCWait = true
		} else {
			vmInst.Config.VNCWait = false
		}
	}
	if isOptionPassed(reflect, "acpi") {
		if *rc.Acpi == true {
			vmInst.Config.ACPI = true
		} else {
			vmInst.Config.ACPI = false
		}
	}
	if isOptionPassed(reflect, "utc") {
		if *rc.Utc == true {
			vmInst.Config.UTCTime = true
		} else {
			vmInst.Config.UTCTime = false
		}
	}
	if isOptionPassed(reflect, "max_wait") {
		vmInst.Config.MaxWait = *rc.MaxWait
	}
	if isOptionPassed(reflect, "tablet") {
		if *rc.Tablet == true {
			vmInst.Config.Tablet = true
		} else {
			vmInst.Config.Tablet = false
		}
	}
	if isOptionPassed(reflect, "storeuefi") {
		if *rc.Storeuefi == true {
			vmInst.Config.StoreUEFIVars = true
		} else {
			vmInst.Config.StoreUEFIVars = false
		}
	}
	if isOptionPassed(reflect, "wireguestmem") {
		if *rc.Wireguestmem == true {
			vmInst.Config.WireGuestMem = true
		} else {
			vmInst.Config.WireGuestMem = false
		}
	}
	if isOptionPassed(reflect, "restart") {
		if *rc.Restart == true {
			vmInst.Config.Restart = true
		} else {
			vmInst.Config.Restart = false
		}
	}
	if isOptionPassed(reflect, "screen") {
		if *rc.Screen == true {
			vmInst.Config.Screen = true
		} else {
			vmInst.Config.Screen = false
		}
	}
	if isOptionPassed(reflect, "hlt") {
		if *rc.Hlt == true {
			vmInst.Config.UseHLT = true
		} else {
			vmInst.Config.UseHLT = false
		}
	}
	if isOptionPassed(reflect, "eop") {
		if *rc.Eop == true {
			vmInst.Config.ExitOnPause = true
		} else {
			vmInst.Config.ExitOnPause = false
		}
	}
	if isOptionPassed(reflect, "dpo") {
		if *rc.Dpo == true {
			vmInst.Config.DestroyPowerOff = true
		} else {
			vmInst.Config.DestroyPowerOff = false
		}
	}
	if isOptionPassed(reflect, "ium") {
		if *rc.Ium == true {
			vmInst.Config.IgnoreUnknownMSR = true
		} else {
			vmInst.Config.IgnoreUnknownMSR = false
		}
	}
	if isOptionPassed(reflect, "hostbridge") {
		if *rc.Hostbridge == true {
			vmInst.Config.HostBridge = true
		} else {
			vmInst.Config.HostBridge = false
		}
	}
	if isOptionPassed(reflect, "restart_delay") {
		vmInst.Config.RestartDelay = *rc.RestartDelay
	}
	if isOptionPassed(reflect, "screen_width") {
		vmInst.Config.ScreenWidth = *rc.ScreenWidth
	}
	if isOptionPassed(reflect, "screen_height") {
		vmInst.Config.ScreenHeight = *rc.ScreenHeight
	}
	if isOptionPassed(reflect, "vncport") {
		if *rc.Vncport != "AUTO" {
			port, err := strconv.Atoi(*rc.Vncport)
			if err != nil {
				return &re, errors.New("invalid vnc port")
			}
			if port < 1024 || port > 65535 {
				return &re, errors.New("invalid vnc port")
			}
		}
		vmInst.Config.VNCPort = *rc.Vncport
	}
	if isOptionPassed(reflect, "keyboard") {
		layoutNames := GetKbdLayoutNames()
		if !util.ContainsStr(layoutNames, *rc.Keyboard) {
			return &re, errors.New("invalid keyboard layout")
		}
		vmInst.Config.KbdLayout = *rc.Keyboard
	}
	if isOptionPassed(reflect, "autostart") {
		if *rc.Autostart == true {
			vmInst.Config.AutoStart = true
		} else {
			vmInst.Config.AutoStart = false
		}
	}
	if isOptionPassed(reflect, "sound") {
		if *rc.Sound {
			vmInst.Config.Sound = true
		} else {
			vmInst.Config.Sound = false
		}
	}
	if isOptionPassed(reflect, "sound_in") {
		if !strings.HasPrefix(*rc.SoundIn, "/dev/dsp") {
			return &re, errors.New("invalid sound dev")
		}
		vmInst.Config.SoundIn = *rc.SoundIn
	}
	if isOptionPassed(reflect, "sound_out") {
		if !strings.HasPrefix(*rc.SoundOut, "/dev/dsp") {
			return &re, errors.New("invalid sound dev")
		}
		vmInst.Config.SoundOut = *rc.SoundOut
	}
	if isOptionPassed(reflect, "com1") {
		if *rc.Com1 {
			vmInst.Config.Com1 = true
		} else {
			vmInst.Config.Com1 = false
		}
	}
	if isOptionPassed(reflect, "com1dev") {
		if *rc.Com1Dev != "AUTO" {
			if !strings.HasPrefix(*rc.Com1Dev, "/dev/nmdm") {
				return &re, errors.New("invalid com dev")
			}
		}
		vmInst.Config.Com1Dev = *rc.Com1Dev
	}
	if isOptionPassed(reflect, "com2") {
		if *rc.Com2 {
			vmInst.Config.Com2 = true
		} else {
			vmInst.Config.Com2 = false
		}
	}
	if isOptionPassed(reflect, "com2dev") {
		if *rc.Com2Dev != "AUTO" {
			if !strings.HasPrefix(*rc.Com2Dev, "/dev/nmdm") {
				return &re, errors.New("invalid com dev")
			}
		}
		vmInst.Config.Com2Dev = *rc.Com2Dev
	}
	if isOptionPassed(reflect, "com3") {
		if *rc.Com3 {
			vmInst.Config.Com3 = true
		} else {
			vmInst.Config.Com3 = false
		}
	}
	if isOptionPassed(reflect, "com3dev") {
		if *rc.Com3Dev != "AUTO" {
			if !strings.HasPrefix(*rc.Com3Dev, "/dev/nmdm") {
				return &re, errors.New("invalid com dev")
			}
		}
		vmInst.Config.Com3Dev = *rc.Com3Dev
	}
	if isOptionPassed(reflect, "com4") {
		if *rc.Com4 {
			vmInst.Config.Com4 = true
		} else {
			vmInst.Config.Com4 = false
		}
	}
	if isOptionPassed(reflect, "com4dev") {
		if *rc.Com4Dev != "AUTO" {
			if !strings.HasPrefix(*rc.Com4Dev, "/dev/nmdm") {
				return &re, errors.New("invalid com dev")
			}
		}
		vmInst.Config.Com4Dev = *rc.Com4Dev
	}
	// TODO -- potential security issue, should it be removed?
	if isOptionPassed(reflect, "extra_args") {
		vmInst.Config.ExtraArgs = *rc.ExtraArgs
	}
	if isOptionPassed(reflect, "com1log") {
		if *rc.Com1Log {
			vmInst.Config.Com1Log = true
		} else {
			vmInst.Config.Com1Log = false
		}
	}
	if isOptionPassed(reflect, "com2log") {
		if *rc.Com2Log {
			vmInst.Config.Com2Log = true
		} else {
			vmInst.Config.Com2Log = false
		}
	}
	if isOptionPassed(reflect, "com3log") {
		if *rc.Com3Log {
			vmInst.Config.Com3Log = true
		} else {
			vmInst.Config.Com3Log = false
		}
	}
	if isOptionPassed(reflect, "com4log") {
		if *rc.Com4Log {
			vmInst.Config.Com4Log = true
		} else {
			vmInst.Config.Com4Log = false
		}
	}
	if isOptionPassed(reflect, "com1speed") {
		vmInst.Config.Com1Speed = *rc.Com1Speed
	}
	if isOptionPassed(reflect, "com2speed") {
		vmInst.Config.Com2Speed = *rc.Com2Speed
	}
	if isOptionPassed(reflect, "com3speed") {
		vmInst.Config.Com3Speed = *rc.Com3Speed
	}
	if isOptionPassed(reflect, "com4speed") {
		vmInst.Config.Com4Speed = *rc.Com4Speed
	}
	if isOptionPassed(reflect, "autostart_delay") {
		if *rc.AutostartDelay > 3600 {
			vmInst.Config.AutoStartDelay = 3600
		} else {
			vmInst.Config.AutoStartDelay = *rc.AutostartDelay
		}
	}
	if isOptionPassed(reflect, "debug") {
		if *rc.Debug {
			vmInst.Config.Debug = true
		} else {
			vmInst.Config.Debug = false
		}
	}
	if isOptionPassed(reflect, "debug_wait") {
		if *rc.DebugWait {
			vmInst.Config.DebugWait = true
		} else {
			vmInst.Config.DebugWait = false
		}
	}
	if isOptionPassed(reflect, "debug_port") {
		if *rc.DebugPort != "AUTO" {
			port, err := strconv.Atoi(*rc.DebugPort)
			if err != nil {
				return &re, errors.New("invalid debug port")
			}
			if port < 1024 || port > 65535 {
				return &re, errors.New("invalid debug port")
			}
		}
		vmInst.Config.DebugPort = *rc.DebugPort
	}
	if isOptionPassed(reflect, "pcpu") {
		vmInst.Config.Pcpu = *rc.Pcpu
	}
	if isOptionPassed(reflect, "rbps") {
		vmInst.Config.Rbps = *rc.Rbps
	}
	if isOptionPassed(reflect, "wbps") {
		vmInst.Config.Wbps = *rc.Wbps
	}
	if isOptionPassed(reflect, "riops") {
		vmInst.Config.Riops = *rc.Riops
	}
	if isOptionPassed(reflect, "wiops") {
		vmInst.Config.Wiops = *rc.Wiops
	}
	err = vmInst.Save()
	if err != nil {
		return &re, err
	}
	re.Success = true
	return &re, nil
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
	var vms []*vm.VM
	var pvmId cirrina.VMID

	vms = vm.GetAll()
	for e := range vms {
		pvmId.Value = vms[e].ID
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
	vm.InitOneVm(vmInst)
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
	if requests.PendingReqExists(v.Value) {
		return &cirrina.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	if vmInst.Status != vm.STOPPED {
		return &cirrina.RequestID{}, errors.New("vm must be stopped before deleting")
	}
	newReq, err := requests.Create(requests.DELETE, v.Value)
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

	err = vmInst.AttachNics(sn.Vmnicid)
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
	if requests.PendingReqExists(vmUuid.String()) {
		return &cirrina.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	if vmInst.Status != vm.STOPPED {
		return &cirrina.RequestID{}, errors.New("vm must be stopped before starting")
	}
	newReq, err := requests.Create(requests.START, vmUuid.String())
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
	if requests.PendingReqExists(vmUuid.String()) {
		return &cirrina.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	if vmInst.Status != vm.RUNNING {
		return &cirrina.RequestID{}, errors.New("vm must be running before stopping")
	}
	newReq, err := requests.Create(requests.STOP, v.Value)
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
