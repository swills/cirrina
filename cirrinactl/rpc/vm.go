package rpc

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"cirrina/cirrina"
)

func AddVM(name string, descrPtr *string, cpuPtr *uint32, memPtr *uint32) (string, error) {
	var err error

	if name == "" {
		return "", errVMEmptyName
	}

	VMConfig := &cirrina.VMConfig{
		Name: &name,
	}

	if descrPtr != nil {
		VMConfig.Description = descrPtr
	}

	if cpuPtr != nil {
		VMConfig.Cpu = cpuPtr
	}

	if memPtr != nil {
		VMConfig.Mem = memPtr
	}

	var res *cirrina.VMID

	res, err = serverClient.AddVM(defaultServerContext, VMConfig)
	if err != nil {
		return "", fmt.Errorf("unable to add VM: %w", err)
	}

	return res.GetValue(), nil
}

func DeleteVM(vmID string) (string, error) {
	var err error

	if vmID == "" {
		return "", errVMEmptyID
	}

	var reqID *cirrina.RequestID

	reqID, err = serverClient.DeleteVM(defaultServerContext, &cirrina.VMID{Value: vmID})
	if err != nil {
		return "", fmt.Errorf("unable to delete VM: %w", err)
	}

	return reqID.GetValue(), nil
}

func StopVM(vmID string) (string, error) {
	var err error

	if vmID == "" {
		return "", errVMEmptyID
	}

	var reqID *cirrina.RequestID

	reqID, err = serverClient.StopVM(defaultServerContext, &cirrina.VMID{Value: vmID})
	if err != nil {
		return "", fmt.Errorf("unable to stop VM: %w", err)
	}

	return reqID.GetValue(), nil
}

func StartVM(vmID string) (string, error) {
	var err error

	if vmID == "" {
		return "", errVMEmptyID
	}

	var reqID *cirrina.RequestID

	reqID, err = serverClient.StartVM(defaultServerContext, &cirrina.VMID{Value: vmID})
	if err != nil {
		return "", fmt.Errorf("unable to start VM: %w", err)
	}

	return reqID.GetValue(), nil
}

func GetVMName(vmID string) (string, error) {
	var err error

	if vmID == "" {
		return "", errVMEmptyID
	}

	var res *wrapperspb.StringValue

	res, err = serverClient.GetVMName(defaultServerContext, &cirrina.VMID{Value: vmID})
	if err != nil {
		return "", fmt.Errorf("unable to get VM name: %w", err)
	}

	return res.GetValue(), nil
}

func GetVMId(name string) (string, error) {
	var err error

	if name == "" {
		return "", errVMEmptyName
	}

	var res *cirrina.VMID

	res, err = serverClient.GetVMID(defaultServerContext, wrapperspb.String(name))
	if err != nil {
		return "", fmt.Errorf("unable to get VM ID: %w", err)
	}

	return res.GetValue(), nil
}

func GetVMConfig(vmID string) (VMConfig, error) {
	var err error

	if vmID == "" {
		return VMConfig{}, errVMEmptyID
	}

	var res *cirrina.VMConfig

	res, err = serverClient.GetVMConfig(defaultServerContext, &cirrina.VMID{Value: vmID})
	if err != nil {
		return VMConfig{}, fmt.Errorf("unable to get VM config: %w", err)
	}

	var retVMConfig VMConfig
	retVMConfig.ID = res.GetId()

	retVMConfig = parseOptionalVMConfigBasic(res, retVMConfig)
	retVMConfig = parseOptionalVMConfigPriority(res, retVMConfig)
	retVMConfig = parseOptionalVMConfigSerialCom1(res, retVMConfig)
	retVMConfig = parseOptionalVMConfigSerialCom2(res, retVMConfig)
	retVMConfig = parseOptionalVMConfigSerialCom3(res, retVMConfig)
	retVMConfig = parseOptionalVMConfigSerialCom4(res, retVMConfig)
	retVMConfig = parseOptionalVMConfigScreen(res, retVMConfig)
	retVMConfig = parseOptionalVMConfigSound(res, retVMConfig)
	retVMConfig = parseOptionalVMConfigStart(res, retVMConfig)
	retVMConfig = parseOptionalVMConfigAdvanced(res, retVMConfig)

	return retVMConfig, nil
}

func GetVMIds() ([]string, error) {
	var err error

	var res cirrina.VMInfo_GetVMsClient
	res, err = serverClient.GetVMs(defaultServerContext, &cirrina.VMsQuery{})

	var ids []string

	if err != nil {
		return []string{}, fmt.Errorf("unable to get VM IDs: %w", err)
	}

	for {
		var aVMID *cirrina.VMID

		aVMID, err = res.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return []string{}, fmt.Errorf("unable to get aVMID IDs: %w", err)
		}

		ids = append(ids, aVMID.GetValue())
	}

	return ids, nil
}

func GetVMState(vmID string) (string, string, string, error) {
	var err error

	if vmID == "" {
		return "", "", "", errVMEmptyID
	}

	var res *cirrina.VMState

	res, err = serverClient.GetVMState(defaultServerContext, &cirrina.VMID{Value: vmID})
	if err != nil {
		return "", "", "", fmt.Errorf("unable to get VM state: %w", err)
	}

	var vmstate string

	switch res.GetStatus() {
	case cirrina.VmStatus_STATUS_STOPPED:
		vmstate = "stopped"
	case cirrina.VmStatus_STATUS_STARTING:
		vmstate = "starting"
	case cirrina.VmStatus_STATUS_RUNNING:
		vmstate = "running"
	case cirrina.VmStatus_STATUS_STOPPING:
		vmstate = "stopping"
	}

	return vmstate, strconv.FormatInt(int64(res.GetVncPort()), 10), strconv.FormatInt(int64(res.GetDebugPort()), 10), nil
}

func VMRunning(vmID string) (bool, error) {
	r, _, _, err := GetVMState(vmID)
	if err != nil {
		return false, err
	}

	if r == "running" {
		return true, nil
	}

	return false, nil
}

func VMStopped(vmID string) (bool, error) {
	r, _, _, err := GetVMState(vmID)
	if err != nil {
		return false, err
	}

	if r == "stopped" {
		return true, nil
	}

	return false, nil
}

func VMNameToID(name string) (string, error) {
	if name == "" {
		return "", errVMEmptyName
	}

	res, err := GetVMId(name)
	if err != nil {
		return "", err
	}

	if res == "" {
		return "", errVMNotFound
	}

	return res, nil
}

func VMIdToName(vmID string) (string, error) {
	if vmID == "" {
		return "", errVMEmptyID
	}

	res, err := GetVMName(vmID)
	if err != nil {
		return "", err
	}

	if res == "" {
		return "", errVMNotFound
	}

	return res, nil
}

func UpdateVMConfig(myNewConfig *cirrina.VMConfig) error {
	var err error

	_, err = serverClient.UpdateVM(defaultServerContext, myNewConfig)
	if err != nil {
		return fmt.Errorf("unable to update VM: %w", err)
	}

	return nil
}

func VMClearUefiVars(vmID string) (bool, error) {
	var err error

	var res *cirrina.ReqBool

	res, err = serverClient.ClearUEFIState(defaultServerContext, &cirrina.VMID{Value: vmID})
	if err != nil {
		return false, fmt.Errorf("unable to clear UEFI state vars: %w", err)
	}

	return res.GetSuccess(), nil
}

func parseOptionalVMConfigBasic(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	retVMConfig.Name = res.GetName()
	retVMConfig.Description = res.GetDescription()
	retVMConfig.CPU = res.GetCpu()
	retVMConfig.Mem = res.GetMem()

	return retVMConfig
}

func parseOptionalVMConfigPriority(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	retVMConfig.Priority = res.GetPriority()
	retVMConfig.Protect = res.GetProtect()
	retVMConfig.Pcpu = res.GetPcpu()
	retVMConfig.Rbps = res.GetRbps()
	retVMConfig.Wbps = res.GetWbps()
	retVMConfig.Riops = res.GetRiops()
	retVMConfig.Wiops = res.GetWiops()

	return retVMConfig
}

func parseOptionalVMConfigSerialCom1(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	retVMConfig.Com1 = res.GetCom1()
	retVMConfig.Com1Log = res.GetCom1Log()
	retVMConfig.Com1Dev = res.GetCom1Dev()
	retVMConfig.Com1Speed = res.GetCom1Speed()

	return retVMConfig
}

func parseOptionalVMConfigSerialCom2(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	retVMConfig.Com2 = res.GetCom2()
	retVMConfig.Com2Log = res.GetCom2Log()
	retVMConfig.Com2Dev = res.GetCom2Dev()
	retVMConfig.Com2Speed = res.GetCom2Speed()

	return retVMConfig
}

func parseOptionalVMConfigSerialCom3(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	retVMConfig.Com3 = res.GetCom3()
	retVMConfig.Com3Log = res.GetCom3Log()
	retVMConfig.Com3Dev = res.GetCom3Dev()
	retVMConfig.Com3Speed = res.GetCom3Speed()

	return retVMConfig
}

func parseOptionalVMConfigSerialCom4(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	retVMConfig.Com4 = res.GetCom4()
	retVMConfig.Com4Log = res.GetCom4Log()
	retVMConfig.Com4Dev = res.GetCom4Dev()
	retVMConfig.Com4Speed = res.GetCom4Speed()

	return retVMConfig
}

func parseOptionalVMConfigScreen(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	retVMConfig.Screen = res.GetScreen()
	retVMConfig.Vncport = res.GetVncport()
	retVMConfig.ScreenWidth = res.GetScreenWidth()
	retVMConfig.ScreenHeight = res.GetScreenHeight()
	retVMConfig.Vncwait = res.GetVncwait()
	retVMConfig.Tablet = res.GetTablet()
	retVMConfig.Keyboard = res.GetKeyboard()

	return retVMConfig
}

func parseOptionalVMConfigSound(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	retVMConfig.Sound = res.GetSound()
	retVMConfig.SoundIn = res.GetSoundIn()
	retVMConfig.SoundOut = res.GetSoundOut()

	return retVMConfig
}

func parseOptionalVMConfigStart(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	retVMConfig.Autostart = res.GetAutostart()
	retVMConfig.AutostartDelay = res.GetAutostartDelay()
	retVMConfig.Restart = res.GetRestart()
	retVMConfig.RestartDelay = res.GetRestartDelay()

	return retVMConfig
}

func parseOptionalVMConfigAdvanced(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	retVMConfig.MaxWait = res.GetMaxWait()
	retVMConfig.Storeuefi = res.GetStoreuefi()
	retVMConfig.Utc = res.GetUtc()
	retVMConfig.Dpo = res.GetDpo()
	retVMConfig.Wireguestmem = res.GetWireguestmem()
	retVMConfig.Hostbridge = res.GetHostbridge()
	retVMConfig.Acpi = res.GetAcpi()
	retVMConfig.Eop = res.GetEop()
	retVMConfig.Ium = res.GetIum()
	retVMConfig.Hlt = res.GetHlt()
	retVMConfig.Debug = res.GetDebug()
	retVMConfig.DebugWait = res.GetDebugWait()
	retVMConfig.DebugPort = res.GetDebugPort()
	retVMConfig.ExtraArgs = res.GetExtraArgs()

	return retVMConfig
}
