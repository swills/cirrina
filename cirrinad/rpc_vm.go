package main

import (
	"cirrina/cirrina"
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/vm"
	"context"
	"errors"
	"fmt"
	"golang.org/x/exp/slog"
)

func (s *server) UpdateVM(_ context.Context, rc *cirrina.VMConfig) (*cirrina.ReqBool, error) {
	re := cirrina.ReqBool{}
	re.Success = false
	vmInst, err := vm.GetById(rc.Id)
	if err != nil {
		slog.Debug("error getting vm", "vm", rc.Id, "err", err)
		return &cirrina.ReqBool{}, err
	}
	reflect := rc.ProtoReflect()
	if isOptionPassed(reflect, "name") {
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
		if *rc.Storeuefi == true {
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
		vmInst.Config.VNCPort = *rc.Vncport
	}
	if isOptionPassed(reflect, "keyboard") {
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
		vmInst.Config.SoundIn = *rc.SoundIn
	}
	if isOptionPassed(reflect, "sound_out") {
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
		vmInst.Config.Com4Dev = *rc.Com4Dev
	}
	if isOptionPassed(reflect, "extra_args") {
		vmInst.Config.ExtraArgs = *rc.ExtraArgs
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
	vmInst, err := vm.GetById(v.Value)
	if err != nil {
		slog.Debug("error getting vm", "vm", v.Value, "err", err)
		return &pvm, err
	}
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
	pvm.Vncport = &vmInst.Config.VNCPort
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
	if vmInst.Config.ExtraArgs != "" {
		pvm.ExtraArgs = &vmInst.Config.ExtraArgs
	}
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
	vmInst, err := vm.GetById(p.Value)
	pvm := cirrina.VMState{}
	if err != nil {
		slog.Debug("error getting vm", "vm", p.Value, "err", err)
		return &pvm, err
	}
	switch vmInst.Status {
	case vm.STOPPED:
		pvm.Status = cirrina.VmStatus_STATUS_STOPPED
	case vm.STARTING:
		pvm.Status = cirrina.VmStatus_STATUS_STARTING
		pvm.VncPort = vmInst.VNCPort
	case vm.RUNNING:
		pvm.Status = cirrina.VmStatus_STATUS_RUNNING
		pvm.VncPort = vmInst.VNCPort
	case vm.STOPPING:
		pvm.Status = cirrina.VmStatus_STATUS_STOPPING
		pvm.VncPort = vmInst.VNCPort
	default:
		return &pvm, errors.New("unknown VM state")
	}
	return &pvm, nil
}

func (s *server) AddVM(_ context.Context, v *cirrina.VMConfig) (*cirrina.VMID, error) {
	if _, err := vm.GetByName(*v.Name); err == nil {
		return &cirrina.VMID{}, errors.New(fmt.Sprintf("%v already exists", v.Name))

	}
	defer vm.List.Mu.Unlock()
	vm.List.Mu.Lock()
	vmInst, err := vm.Create(*v.Name, *v.Description, *v.Cpu, *v.Mem)
	vm.InitOneVm(vmInst)
	slog.Debug("Created VM", "vm", vmInst.ID)
	if err != nil {
		return &cirrina.VMID{}, err
	}
	vm.List.VmList[vmInst.ID] = vmInst
	return &cirrina.VMID{Value: vmInst.ID}, nil
}

func (s *server) DeleteVM(_ context.Context, v *cirrina.VMID) (*cirrina.RequestID, error) {
	vmInst, err := vm.GetById(v.Value)
	if err != nil {
		return &cirrina.RequestID{}, err
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
	var isosConfigVal string
	count := 0
	re := cirrina.ReqBool{}
	re.Success = false
	for _, isoid := range sr.Isoid {
		if count > 0 {
			isosConfigVal += ","
		}
		// TODO check that ISO exists
		isosConfigVal += isoid
		count += 1
	}
	vmInst, err := vm.GetById(sr.Id)
	if err != nil {
		slog.Error("error getting vm", "vm", sr.Id, "err", err)
		return &re, err
	}
	vmInst.Config.ISOs = isosConfigVal
	err = vmInst.Save()
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
	vmInst, err := vm.GetById(sn.Vmid)
	if err != nil {
		slog.Error("error getting vm", "vm", sn.Vmid, "err", err)
		return &re, err
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
	vmInst, err := vm.GetById(sr.Id)
	if err != nil {
		slog.Error("error getting vm", "vm", sr.Id, "err", err)
		return &re, err
	}
	err = vmInst.AttachDisks(sr.Diskid)
	if err != nil {
		return &re, err
	}
	re.Success = true
	return &re, nil
}

func (s *server) GetVmISOs(v *cirrina.VMID, stream cirrina.VMInfo_GetVmISOsServer) error {
	vmInst, err := vm.GetById(v.Value)
	slog.Debug("GetVmISOs", "vm", v.Value, "isos", vmInst.Config.ISOs)
	if err != nil {
		slog.Error("error getting vm", "vm", v.Value, "err", err)
		return err
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
	slog.Debug("GetVMDisks called")
	vmInst, err := vm.GetById(v.Value)
	slog.Debug("GetVMDisks", "vm", v.Value, "disks", vmInst.Config.Disks)
	if err != nil {
		slog.Error("error getting vm", "vm", v.Value, "err", err)
		return err
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
	_, err := vm.GetById(v.Value)
	if err != nil {
		return &cirrina.RequestID{}, err
	}
	if requests.PendingReqExists(v.Value) {
		return &cirrina.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	vmInst, err := vm.GetById(v.Value)
	if err != nil {
		slog.Debug("error getting vm", "vm", v.Value, "err", err)
		return &cirrina.RequestID{}, err
	}
	if vmInst.Status != vm.STOPPED {
		return &cirrina.RequestID{}, errors.New("vm must be stopped before starting")
	}
	newReq, err := requests.Create(requests.START, v.Value)
	if err != nil {
		return &cirrina.RequestID{}, err
	}
	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) StopVM(_ context.Context, v *cirrina.VMID) (*cirrina.RequestID, error) {
	_, err := vm.GetById(v.Value)
	if err != nil {
		return &cirrina.RequestID{}, err
	}
	if requests.PendingReqExists(v.Value) {
		return &cirrina.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	vmInst, err := vm.GetById(v.Value)
	if err != nil {
		slog.Error("error getting vm", "vm", v.Value, "err", err)
		return &cirrina.RequestID{}, err
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

	vmInst, err := vm.GetById(v.Value)
	if err != nil {
		return err
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
