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
	"github.com/hashicorp/go-version"
	"github.com/spf13/cast"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"cirrina/cirrina"
	"cirrina/cirrinad/iso"
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
	"cirrina/cirrinad/vmnic"
)

var (
	defaultVMCpuCount     uint32 = 1
	defaultVMMemCount     uint32 = 128
	GetKbdLayoutNamesFunc        = GetKbdLayoutNames
)

func (s *server) UpdateVM(_ context.Context, vmConfig *cirrina.VMConfig) (*cirrina.ReqBool, error) {
	res := cirrina.ReqBool{}
	res.Success = false

	vmUUID, err := uuid.Parse(vmConfig.GetId())
	if err != nil {
		return &res, fmt.Errorf("error parsing VM ID: %w", err)
	}

	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("UpdateVM error getting vm", "vm", vmConfig.GetId(), "err", err)

		return &res, errNotFound
	}

	if vmInst.Name == "" {
		return &res, errNotFound
	}

	err = updateVMAll(vmConfig, vmInst)
	if err != nil {
		return &res, err
	}

	err = vmInst.Save()
	if err != nil {
		return &res, fmt.Errorf("error saving VM: %w", err)
	}

	res.Success = true

	return &res, nil
}

func updateVMAll(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	var err error

	err = updateVMBasics(vmConfig, vmInst)
	if err != nil {
		return err
	}

	err = updateVMCom1(vmConfig, vmInst)
	if err != nil {
		return err
	}

	err = updateVMCom2(vmConfig, vmInst)
	if err != nil {
		return err
	}

	err = updateVMCom3(vmConfig, vmInst)
	if err != nil {
		return err
	}

	err = updateVMCom4(vmConfig, vmInst)
	if err != nil {
		return err
	}

	err = updateVMScreen(vmConfig, vmInst)
	if err != nil {
		return err
	}

	err = updateVMScreenOptions(vmConfig, vmInst)
	if err != nil {
		return err
	}

	err = updateVMSound(vmConfig, vmInst)
	if err != nil {
		return err
	}

	updateVMStart(vmConfig, vmInst)
	updateVMAdvanced1(vmConfig, vmInst)
	updateVMAdvanced2(vmConfig, vmInst)

	err = updateVMDebug(vmConfig, vmInst)
	if err != nil {
		return err
	}

	updateVMPriorityLimits(vmConfig, vmInst)

	return nil
}

func validateScreenWidth(vmConfig *cirrina.VMConfig) error {
	var osVer *version.Version

	var maxWidth uint32

	var minWidth uint32 = 640

	var err error

	osVer, err = util.GetOsVersion()
	if err != nil {
		slog.Error("failed to get os version", "err", err)

		return fmt.Errorf("error validating config: %w", err)
	}

	ver150, err := version.NewVersion("15.0")
	if err != nil {
		slog.Error("failed to create a version for 15.0")

		return fmt.Errorf("error validating config: %w", err)
	}

	if osVer.LessThan(ver150) {
		maxWidth = 1920
	} else {
		maxWidth = 3840
	}

	if vmConfig.GetScreenWidth() < minWidth || vmConfig.GetScreenWidth() > maxWidth {
		return errInvalidScreenWidth
	}

	return nil
}

func validateScreenHeight(vmConfig *cirrina.VMConfig) error {
	var osVer *version.Version

	var maxHeight uint32

	var minHeight uint32 = 480

	var err error

	osVer, err = util.GetOsVersion()
	if err != nil {
		slog.Error("failed to get os version", "err", err)

		return fmt.Errorf("error validating config: %w", err)
	}

	ver150, err := version.NewVersion("15.0")
	if err != nil {
		slog.Error("failed to create a version for 15.0")

		return fmt.Errorf("error validating config: %w", err)
	}

	if osVer.LessThan(ver150) {
		maxHeight = 1200
	} else {
		maxHeight = 2160
	}

	if vmConfig.GetScreenHeight() < minHeight || vmConfig.GetScreenHeight() > maxHeight {
		return errInvalidScreenHeight
	}

	return nil
}

func updateVMPriorityLimits(vmConfig *cirrina.VMConfig, vmInst *vm.VM) {
	if vmConfig.Priority != nil {
		vmInst.Config.Priority = vmConfig.GetPriority()
	}

	if vmConfig.Protect != nil {
		vmInst.Config.Protect = sql.NullBool{Bool: vmConfig.GetProtect(), Valid: true}
	}

	if vmConfig.Pcpu != nil {
		vmInst.Config.Pcpu = vmConfig.GetPcpu()
	}

	if vmConfig.Rbps != nil {
		vmInst.Config.Rbps = vmConfig.GetRbps()
	}

	if vmConfig.Wbps != nil {
		vmInst.Config.Wbps = vmConfig.GetWbps()
	}

	if vmConfig.Riops != nil {
		vmInst.Config.Riops = vmConfig.GetRiops()
	}

	if vmConfig.Wiops != nil {
		vmInst.Config.Wiops = vmConfig.GetWiops()
	}
}

func updateVMDebug(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	if vmConfig.Debug != nil {
		vmInst.Config.Debug = vmConfig.GetDebug()
	}

	if vmConfig.DebugWait != nil {
		vmInst.Config.DebugWait = vmConfig.GetDebugWait()
	}

	if vmConfig.DebugPort != nil {
		if vmConfig.GetDebugPort() != "AUTO" {
			port, err := strconv.Atoi(vmConfig.GetDebugPort())
			if err != nil || port < 1024 || port > 65535 {
				return errInvalidDebugPort
			}
		}

		vmInst.Config.DebugPort = vmConfig.GetDebugPort()
	}

	return nil
}

func updateVMAdvanced1(vmConfig *cirrina.VMConfig, vmInst *vm.VM) {
	if vmConfig.Hostbridge != nil {
		vmInst.Config.HostBridge = vmConfig.GetHostbridge()
	}

	if vmConfig.Acpi != nil {
		vmInst.Config.ACPI = vmConfig.GetAcpi()
	}

	if vmConfig.Storeuefi != nil {
		vmInst.Config.StoreUEFIVars = vmConfig.GetStoreuefi()
	}

	if vmConfig.Utc != nil {
		vmInst.Config.UTCTime = vmConfig.GetUtc()
	}

	if vmConfig.Wireguestmem != nil {
		vmInst.Config.WireGuestMem = vmConfig.GetWireguestmem()
	}
}

func updateVMAdvanced2(vmConfig *cirrina.VMConfig, vmInst *vm.VM) {
	if vmConfig.Dpo != nil {
		vmInst.Config.DestroyPowerOff = vmConfig.GetDpo()
	}

	if vmConfig.Eop != nil {
		vmInst.Config.ExitOnPause = vmConfig.GetEop()
	}

	if vmConfig.Ium != nil {
		vmInst.Config.IgnoreUnknownMSR = vmConfig.GetIum()
	}

	if vmConfig.Hlt != nil {
		vmInst.Config.UseHLT = vmConfig.GetHlt()
	}

	// TODO -- potential security issue, should it be removed?
	if vmConfig.ExtraArgs != nil {
		vmInst.Config.ExtraArgs = vmConfig.GetExtraArgs()
	}
}

func updateVMStart(vmConfig *cirrina.VMConfig, vmInst *vm.VM) {
	if vmConfig.Autostart != nil {
		vmInst.Config.AutoStart = vmConfig.GetAutostart()
	}

	if vmConfig.AutostartDelay != nil {
		if vmConfig.GetAutostartDelay() > 3600 {
			vmInst.Config.AutoStartDelay = 3600
		} else {
			vmInst.Config.AutoStartDelay = vmConfig.GetAutostartDelay()
		}
	}

	if vmConfig.Restart != nil {
		vmInst.Config.Restart = vmConfig.GetRestart()
	}

	if vmConfig.RestartDelay != nil {
		vmInst.Config.RestartDelay = vmConfig.GetRestartDelay()
	}

	if vmConfig.MaxWait != nil {
		vmInst.Config.MaxWait = vmConfig.GetMaxWait()
	}
}

func updateVMSound(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	if vmConfig.Sound != nil {
		vmInst.Config.Sound = vmConfig.GetSound()
	}

	if vmConfig.SoundIn != nil {
		if !strings.HasPrefix(vmConfig.GetSoundIn(), "/dev/dsp") {
			return errInvalidSoundDev
		}

		vmInst.Config.SoundIn = vmConfig.GetSoundIn()
	}

	if vmConfig.SoundOut != nil {
		if !strings.HasPrefix(vmConfig.GetSoundOut(), "/dev/dsp") {
			return errInvalidSoundDev
		}

		vmInst.Config.SoundOut = vmConfig.GetSoundOut()
	}

	return nil
}

func updateVMScreen(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	var err error

	if vmConfig.Screen != nil {
		vmInst.Config.Screen = vmConfig.GetScreen()
	}

	if vmConfig.ScreenWidth != nil {
		err = validateScreenWidth(vmConfig)
		if err != nil {
			return err
		}

		vmInst.Config.ScreenWidth = vmConfig.GetScreenWidth()
	}

	if vmConfig.ScreenHeight != nil {
		err = validateScreenHeight(vmConfig)
		if err != nil {
			return err
		}

		vmInst.Config.ScreenHeight = vmConfig.GetScreenHeight()
	}

	return nil
}

func updateVMScreenOptions(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	if vmConfig.Vncport != nil {
		if vmConfig.GetVncport() != "AUTO" {
			port, err := strconv.ParseInt(vmConfig.GetVncport(), 10, 64)
			if err != nil || port < 1024 || port > 65535 {
				return errInvalidVncPort
			}
		}

		vmInst.Config.VNCPort = vmConfig.GetVncport()
	}

	if vmConfig.Keyboard != nil {
		layoutNames := GetKbdLayoutNamesFunc()
		if !util.ContainsStr(layoutNames, vmConfig.GetKeyboard()) {
			return errInvalidKeyboardLayout
		}

		vmInst.Config.KbdLayout = vmConfig.GetKeyboard()
	}

	if vmConfig.Tablet != nil {
		vmInst.Config.Tablet = vmConfig.GetTablet()
	}

	if vmConfig.Vncwait != nil {
		vmInst.Config.VNCWait = vmConfig.GetVncwait()
	}

	return nil
}

func updateVMCom1(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	if vmConfig.Com1 != nil {
		vmInst.Config.Com1 = vmConfig.GetCom1()
	}

	if vmConfig.Com1Dev != nil {
		if vmConfig.GetCom1Dev() != "AUTO" {
			if !strings.HasPrefix(vmConfig.GetCom1Dev(), "/dev/nmdm") {
				return errInvalidComDev
			}
		}

		vmInst.Config.Com1Dev = vmConfig.GetCom1Dev()
	}

	if vmConfig.Com1Log != nil {
		vmInst.Config.Com1Log = vmConfig.GetCom1Log()
	}

	if vmConfig.Com1Speed != nil {
		vmInst.Config.Com1Speed = vmConfig.GetCom1Speed()
	}

	return nil
}

func updateVMCom2(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	if vmConfig.Com2 != nil {
		vmInst.Config.Com2 = vmConfig.GetCom2()
	}

	if vmConfig.Com2Dev != nil {
		if vmConfig.GetCom2Dev() != "AUTO" {
			if !strings.HasPrefix(vmConfig.GetCom2Dev(), "/dev/nmdm") {
				return errInvalidComDev
			}
		}

		vmInst.Config.Com2Dev = vmConfig.GetCom2Dev()
	}

	if vmConfig.Com2Log != nil {
		vmInst.Config.Com2Log = vmConfig.GetCom2Log()
	}

	if vmConfig.Com2Speed != nil {
		vmInst.Config.Com2Speed = vmConfig.GetCom2Speed()
	}

	return nil
}

func updateVMCom3(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	if vmConfig.Com3 != nil {
		vmInst.Config.Com3 = vmConfig.GetCom3()
	}

	if vmConfig.Com3Dev != nil {
		if vmConfig.GetCom3Dev() != "AUTO" {
			if !strings.HasPrefix(vmConfig.GetCom3Dev(), "/dev/nmdm") {
				return errInvalidComDev
			}
		}

		vmInst.Config.Com3Dev = vmConfig.GetCom3Dev()
	}

	if vmConfig.Com3Log != nil {
		vmInst.Config.Com3Log = vmConfig.GetCom3Log()
	}

	if vmConfig.Com3Speed != nil {
		vmInst.Config.Com3Speed = vmConfig.GetCom3Speed()
	}

	return nil
}

func updateVMCom4(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	if vmConfig.Com4 != nil {
		vmInst.Config.Com4 = vmConfig.GetCom4()
	}

	if vmConfig.Com4Dev != nil {
		if vmConfig.GetCom4Dev() != "AUTO" {
			if !strings.HasPrefix(vmConfig.GetCom4Dev(), "/dev/nmdm") {
				return errInvalidComDev
			}
		}

		vmInst.Config.Com4Dev = vmConfig.GetCom4Dev()
	}

	if vmConfig.Com4Log != nil {
		vmInst.Config.Com4Log = vmConfig.GetCom4Log()
	}

	if vmConfig.Com4Speed != nil {
		vmInst.Config.Com4Speed = vmConfig.GetCom4Speed()
	}

	return nil
}

func updateVMBasics(vmConfig *cirrina.VMConfig, vmInst *vm.VM) error {
	if vmConfig.Name != nil {
		if !util.ValidVMName(vmConfig.GetName()) {
			return errInvalidName
		}

		exists := vm.Exists(vmConfig.GetName())
		if exists {
			return errVMDupe
		}

		vmInst.Name = vmConfig.GetName()
	}

	if vmConfig.Description != nil {
		vmInst.Description = vmConfig.GetDescription()
	}

	if vmConfig.Cpu != nil && util.NumCpusValid(cast.ToUint16(vmConfig.GetCpu())) {
		vmInst.Config.CPU = vmConfig.GetCpu()
	}

	if vmConfig.Mem != nil {
		vmInst.Config.Mem = vmConfig.GetMem()
	}

	return nil
}

func (s *server) GetVMID(_ context.Context, vmNameReq *wrapperspb.StringValue) (*cirrina.VMID, error) {
	vmName := vmNameReq.GetValue()

	if vmName == "" {
		return &cirrina.VMID{}, errInvalidName
	}

	vmInst, err := vm.GetByName(vmName)
	if err != nil {
		if errors.Is(err, vm.ErrVMNotFound) {
			return &cirrina.VMID{}, fmt.Errorf("error getting VM: %w", status.Error(codes.NotFound, err.Error()))
		}

		return &cirrina.VMID{}, fmt.Errorf("error getting VM: %w", err)
	}

	return &cirrina.VMID{Value: vmInst.ID}, nil
}

func (s *server) GetVMName(_ context.Context, vmID *cirrina.VMID) (*wrapperspb.StringValue, error) {
	vmUUID, err := uuid.Parse(vmID.GetValue())
	if err != nil {
		return wrapperspb.String(""), fmt.Errorf("error parsing ID: %w", err)
	}

	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		if errors.Is(err, errNotFound) {
			return wrapperspb.String(""), status.Error(codes.NotFound, err.Error())
		}

		slog.Error("GetVMName error getting VM", "vm", vmID.GetValue(), "err", err)

		return wrapperspb.String(""), fmt.Errorf("error getting VM: %w", err)
	}

	return wrapperspb.String(vmInst.Name), nil
}

func (s *server) GetVMConfig(_ context.Context, vmID *cirrina.VMID) (*cirrina.VMConfig, error) {
	var pvm cirrina.VMConfig

	vmUUID, err := uuid.Parse(vmID.GetValue())
	if err != nil {
		return &pvm, fmt.Errorf("error parsing VM ID: %w", err)
	}

	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("GetVMConfig error getting vm", "vm", vmID.GetValue(), "err", err)

		return &pvm, errNotFound
	}

	if vmInst.Name == "" {
		return &pvm, errNotFound
	}

	pvm.Id = vmID.GetValue()
	pvm.Name = &vmInst.Name
	pvm.Description = &vmInst.Description
	pvm.Cpu = &vmInst.Config.CPU
	pvm.Mem = &vmInst.Config.Mem

	getVMConfigCom1(&pvm, vmInst)
	getVMConfigCom2(&pvm, vmInst)
	getVMConfigCom3(&pvm, vmInst)
	getVMConfigCom4(&pvm, vmInst)
	getVMConfigScreen(&pvm, vmInst)
	getVMConfigScreenOptions(&pvm, vmInst)
	getVMCOnfigSound(&pvm, vmInst)
	getVMConfigStart(&pvm, vmInst)
	getVMConfigAdvanced1(&pvm, vmInst)
	getVMConfigAdvanced2(&pvm, vmInst)
	getVMConfigDebug(&pvm, vmInst)
	getVMConfigPriorityLimits(&pvm, vmInst)

	return &pvm, nil
}

func getVMConfigCom1(pvm *cirrina.VMConfig, vmInst *vm.VM) {
	pvm.Com1 = &vmInst.Config.Com1
	pvm.Com1Dev = &vmInst.Config.Com1Dev
	pvm.Com1Log = &vmInst.Config.Com1Log
	pvm.Com1Speed = &vmInst.Config.Com1Speed
}

func getVMConfigCom2(pvm *cirrina.VMConfig, vmInst *vm.VM) {
	pvm.Com2 = &vmInst.Config.Com2
	pvm.Com2Dev = &vmInst.Config.Com2Dev
	pvm.Com2Log = &vmInst.Config.Com2Log
	pvm.Com2Speed = &vmInst.Config.Com2Speed
}

func getVMConfigCom3(pvm *cirrina.VMConfig, vmInst *vm.VM) {
	pvm.Com3 = &vmInst.Config.Com3
	pvm.Com3Dev = &vmInst.Config.Com3Dev
	pvm.Com3Log = &vmInst.Config.Com3Log
	pvm.Com3Speed = &vmInst.Config.Com3Speed
}

func getVMConfigCom4(pvm *cirrina.VMConfig, vmInst *vm.VM) {
	pvm.Com4 = &vmInst.Config.Com4
	pvm.Com4Dev = &vmInst.Config.Com4Dev
	pvm.Com4Log = &vmInst.Config.Com4Log
	pvm.Com4Speed = &vmInst.Config.Com4Speed
}

func getVMConfigScreen(pvm *cirrina.VMConfig, vmInst *vm.VM) {
	pvm.Screen = &vmInst.Config.Screen
	pvm.ScreenWidth = &vmInst.Config.ScreenWidth
	pvm.ScreenHeight = &vmInst.Config.ScreenHeight
}

func getVMConfigScreenOptions(pvm *cirrina.VMConfig, vmInst *vm.VM) {
	pvm.Vncport = &vmInst.Config.VNCPort
	pvm.Keyboard = &vmInst.Config.KbdLayout
	pvm.Tablet = &vmInst.Config.Tablet
	pvm.Vncwait = &vmInst.Config.VNCWait
}

func getVMCOnfigSound(pvm *cirrina.VMConfig, vmInst *vm.VM) {
	pvm.Sound = &vmInst.Config.Sound
	pvm.SoundIn = &vmInst.Config.SoundIn
	pvm.SoundOut = &vmInst.Config.SoundOut
}

func getVMConfigStart(pvm *cirrina.VMConfig, vmInst *vm.VM) {
	pvm.Autostart = &vmInst.Config.AutoStart
	pvm.AutostartDelay = &vmInst.Config.AutoStartDelay
	pvm.Restart = &vmInst.Config.Restart
	pvm.RestartDelay = &vmInst.Config.RestartDelay
	pvm.MaxWait = &vmInst.Config.MaxWait
}

func getVMConfigAdvanced1(pvm *cirrina.VMConfig, vmInst *vm.VM) {
	pvm.Hostbridge = &vmInst.Config.HostBridge
	pvm.Acpi = &vmInst.Config.ACPI
	pvm.Storeuefi = &vmInst.Config.StoreUEFIVars
	pvm.Utc = &vmInst.Config.UTCTime
	pvm.Wireguestmem = &vmInst.Config.WireGuestMem
}

func getVMConfigAdvanced2(pvm *cirrina.VMConfig, vmInst *vm.VM) {
	pvm.Hlt = &vmInst.Config.UseHLT
	pvm.Eop = &vmInst.Config.ExitOnPause
	pvm.Dpo = &vmInst.Config.DestroyPowerOff
	pvm.Ium = &vmInst.Config.IgnoreUnknownMSR
	pvm.ExtraArgs = &vmInst.Config.ExtraArgs
}

func getVMConfigDebug(pvm *cirrina.VMConfig, vmInst *vm.VM) {
	pvm.Debug = &vmInst.Config.Debug
	pvm.DebugWait = &vmInst.Config.DebugWait
	pvm.DebugPort = &vmInst.Config.DebugPort
}

func getVMConfigPriorityLimits(pvm *cirrina.VMConfig, vmInst *vm.VM) {
	pvm.Priority = &vmInst.Config.Priority
	pvm.Protect = &vmInst.Config.Protect.Bool
	pvm.Pcpu = &vmInst.Config.Pcpu
	pvm.Rbps = &vmInst.Config.Rbps
	pvm.Wbps = &vmInst.Config.Wbps
	pvm.Riops = &vmInst.Config.Riops
	pvm.Wiops = &vmInst.Config.Wiops
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
	vmUUID, err := uuid.Parse(vmID.GetValue())

	if err != nil {
		return &pvm, fmt.Errorf("error parsing ID: %w", err)
	}

	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("GetVMState error getting vm", "vm", vmID.GetValue(), "err", err)

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

	if vmConfig.Name == nil {
		return &cirrina.VMID{}, errInvalidName
	}

	if !util.ValidVMName(vmConfig.GetName()) {
		return &cirrina.VMID{}, errInvalidName
	}

	if vmConfig.Description == nil {
		vmConfig.Description = &defaultVMDescription
	}

	if vmConfig.Cpu == nil || !util.NumCpusValid(cast.ToUint16(vmConfig.GetCpu())) {
		vmConfig.Cpu = &defaultVMCpuCount
	}

	if vmConfig.Mem == nil || vmConfig.GetMem() < defaultVMMemCount {
		vmConfig.Mem = &defaultVMMemCount
	}

	vmInst := &vm.VM{
		Name:        vmConfig.GetName(),
		Status:      vm.STOPPED,
		Description: vmConfig.GetDescription(),
		Config: vm.Config{
			CPU: vmConfig.GetCpu(),
			Mem: vmConfig.GetMem(),
		},
	}

	err := vm.Create(vmInst)
	if err != nil {
		return &cirrina.VMID{}, fmt.Errorf("error creating VM: %w", err)
	}

	slog.Debug("Created VM", "vm", vmInst.ID)

	return &cirrina.VMID{Value: vmInst.ID}, nil
}

func (s *server) DeleteVM(_ context.Context, vmID *cirrina.VMID) (*cirrina.RequestID, error) {
	vmUUID, err := uuid.Parse(vmID.GetValue())
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error parsing VM ID: %w", err)
	}

	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("DeleteVM error getting vm", "vm", vmID.GetValue(), "err", err)

		return &cirrina.RequestID{}, fmt.Errorf("error getting VM: %w", err)
	}

	if vmInst.Name == "" {
		return &cirrina.RequestID{}, errNotFound
	}

	pendingReqIDs := requests.PendingReqExists(vmID.GetValue())
	if len(pendingReqIDs) > 0 {
		return &cirrina.RequestID{}, errReqExists
	}

	if vmInst.Status != vm.STOPPED {
		return &cirrina.RequestID{}, errInvalidVMStateDelete
	}

	newReq, err := requests.CreateVMReq(requests.VMDELETE, vmID.GetValue())
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error creating request: %w", err)
	}

	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) SetVMISOs(_ context.Context, setISOReq *cirrina.SetISOReq) (*cirrina.ReqBool, error) {
	res := cirrina.ReqBool{}
	res.Success = false

	vmUUID, err := uuid.Parse(setISOReq.GetId())
	if err != nil {
		return &res, fmt.Errorf("error parsing ID: %w", err)
	}

	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("SetVmISOs error getting vm", "vm", setISOReq.GetId(), "err", err)

		return &res, errNotFound
	}

	if vmInst.Name == "" {
		return &res, errNotFound
	}

	isoIDStrs := setISOReq.GetIsoid()

	isoIDs := make([]*iso.ISO, 0, len(isoIDStrs))

	for _, i := range isoIDStrs {
		var thisIso *iso.ISO

		thisIso, err = iso.GetByID(i)
		if err != nil {
			slog.Error("error getting iso", "iso", thisIso, "err", err)

			return &res, errIsoNotFound
		}

		isoIDs = append(isoIDs, thisIso)
	}

	err = vmInst.AttachIsos(isoIDs)
	if err != nil {
		return &res, fmt.Errorf("error attaching ISO: %w", err)
	}

	res.Success = true

	return &res, nil
}

func (s *server) SetVMNics(_ context.Context, setNicReq *cirrina.SetNicReq) (*cirrina.ReqBool, error) {
	var res cirrina.ReqBool
	res.Success = false

	slog.Debug("SetVmNics", "vm", setNicReq.GetVmid(), "vmnic", setNicReq.GetVmnicid())

	vmUUID, err := uuid.Parse(setNicReq.GetVmid())
	if err != nil {
		return &res, fmt.Errorf("error parsing ID: %w", err)
	}

	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("SetVmNics error getting vm", "vm", setNicReq.GetVmid(), "err", err)

		return &res, fmt.Errorf("error getting VM ID: %w", err)
	}

	if vmInst.Name == "" {
		return &res, errNotFound
	}

	err = vmInst.SetNics(setNicReq.GetVmnicid())
	if err != nil {
		return &res, fmt.Errorf("error setting NICs: %w", err)
	}

	res.Success = true

	return &res, nil
}

func (s *server) SetVMDisks(_ context.Context, setDiskReq *cirrina.SetDiskReq) (*cirrina.ReqBool, error) {
	res := cirrina.ReqBool{}
	res.Success = false

	slog.Debug("SetVmDisks", "vm", setDiskReq.GetId(), "disk", setDiskReq.GetDiskid())

	vmUUID, err := uuid.Parse(setDiskReq.GetId())
	if err != nil {
		return &res, fmt.Errorf("error parsing ID: %w", err)
	}

	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("SetVmDisks error getting vm", "vm", setDiskReq.GetId(), "err", err)

		return &res, errNotFound
	}

	if vmInst.Name == "" {
		return &res, errNotFound
	}

	err = vmInst.AttachDisks(setDiskReq.GetDiskid())
	if err != nil {
		return &res, fmt.Errorf("error attaching disk: %w", err)
	}

	res.Success = true

	return &res, nil
}

func (s *server) GetVMISOs(vmID *cirrina.VMID, stream cirrina.VMInfo_GetVMISOsServer) error {
	vmUUID, err := uuid.Parse(vmID.GetValue())
	if err != nil {
		return fmt.Errorf("error parsing ID: %w", err)
	}

	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("GetVmISOs error getting vm", "vm", vmID.GetValue(), "err", err)

		return fmt.Errorf("error getting VM: %w", err)
	}

	if vmInst.Name == "" {
		return errNotFound
	}

	var isoID cirrina.ISOID

	for _, e := range vmInst.ISOs {
		if e == nil {
			continue
		}

		isoID.Value = e.ID

		err := stream.Send(&isoID)
		if err != nil {
			return fmt.Errorf("error sending to stream: %w", err)
		}
	}

	return nil
}

func (s *server) GetVMDisks(vmID *cirrina.VMID, stream cirrina.VMInfo_GetVMDisksServer) error {
	vmUUID, err := uuid.Parse(vmID.GetValue())
	if err != nil {
		return fmt.Errorf("error parsing ID: %w", err)
	}

	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("GetVmDisks error getting vm", "vm", vmID.GetValue(), "err", err)

		return fmt.Errorf("error getting VM: %w", err)
	}

	if vmInst.Name == "" {
		return errNotFound
	}

	var diskID cirrina.DiskId

	for _, e := range vmInst.Disks {
		if e == nil {
			continue
		}

		diskID.Value = e.ID

		err := stream.Send(&diskID)
		if err != nil {
			return fmt.Errorf("error sending to stream: %w", err)
		}
	}

	return nil
}

func (s *server) StartVM(_ context.Context, vmID *cirrina.VMID) (*cirrina.RequestID, error) {
	vmUUID, err := uuid.Parse(vmID.GetValue())
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error parsing ID: %w", err)
	}

	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("StartVM error getting vm", "vm", vmID.GetValue(), "err", err)

		return &cirrina.RequestID{}, errNotFound
	}

	if vmInst.Name == "" {
		return &cirrina.RequestID{}, errNotFound
	}

	pendingReqIDs := requests.PendingReqExists(vmID.GetValue())
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
	vmUUID, err := uuid.Parse(vmID.GetValue())
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error parsing ID: %w", err)
	}

	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("StopVM error getting vm", "vm", vmID.GetValue(), "err", err)

		return &cirrina.RequestID{}, fmt.Errorf("error getting VM: %w", err)
	}

	if vmInst.Name == "" {
		return &cirrina.RequestID{}, errNotFound
	}

	pendingReqIDs := requests.PendingReqExists(vmID.GetValue())
	if len(pendingReqIDs) > 0 {
		return &cirrina.RequestID{}, errReqExists
	}

	if vmInst.Status != vm.RUNNING {
		return &cirrina.RequestID{}, errInvalidVMStateStop
	}

	newReq, err := requests.CreateVMReq(requests.VMSTOP, vmID.GetValue())
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error creating request: %w", err)
	}

	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) GetVMNics(vmID *cirrina.VMID, stream cirrina.VMInfo_GetVMNicsServer) error {
	var pvmnicID cirrina.VmNicId

	vmUUID, err := uuid.Parse(vmID.GetValue())
	if err != nil {
		return fmt.Errorf("error parsing ID: %w", err)
	}

	vmInst, err := vm.GetByID(vmUUID.String())
	if err != nil {
		slog.Error("GetVMNics error getting vm", "vm", vmID.GetValue(), "err", err)

		return fmt.Errorf("error getting VM: %w", err)
	}

	if vmInst.Name == "" {
		return errNotFound
	}

	vmNics, err := vmnic.GetNics(vmInst.Config.ID)
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
