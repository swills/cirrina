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

	return res.Value, nil
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

	return reqID.Value, nil
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

	return reqID.Value, nil
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

	return reqID.Value, nil
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

	return res.Value, nil
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
	retVMConfig.ID = res.Id

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
		ids = append(ids, aVMID.Value)
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
	switch res.Status {
	case cirrina.VmStatus_STATUS_STOPPED:
		vmstate = "stopped"
	case cirrina.VmStatus_STATUS_STARTING:
		vmstate = "starting"
	case cirrina.VmStatus_STATUS_RUNNING:
		vmstate = "running"
	case cirrina.VmStatus_STATUS_STOPPING:
		vmstate = "stopping"
	}

	return vmstate, strconv.FormatInt(int64(res.VncPort), 10), strconv.FormatInt(int64(res.DebugPort), 10), nil
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

	return res.Success, nil
}

func parseOptionalVMConfigBasic(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	if res.Name != nil {
		retVMConfig.Name = *res.Name
	}
	if res.Description != nil {
		retVMConfig.Description = *res.Description
	}
	if res.Cpu != nil {
		retVMConfig.CPU = *res.Cpu
	}
	if res.Mem != nil {
		retVMConfig.Mem = *res.Mem
	}

	return retVMConfig
}

func parseOptionalVMConfigPriority(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	if res.Priority != nil {
		retVMConfig.Priority = *res.Priority
	}
	if res.Protect != nil {
		retVMConfig.Protect = *res.Protect
	}
	if res.Pcpu != nil {
		retVMConfig.Pcpu = *res.Pcpu
	}
	if res.Rbps != nil {
		retVMConfig.Rbps = *res.Rbps
	}
	if res.Wbps != nil {
		retVMConfig.Wbps = *res.Wbps
	}
	if res.Riops != nil {
		retVMConfig.Riops = *res.Riops
	}
	if res.Wiops != nil {
		retVMConfig.Wiops = *res.Wiops
	}

	return retVMConfig
}

func parseOptionalVMConfigSerialCom1(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	if res.Com1 != nil {
		retVMConfig.Com1 = *res.Com1
	}
	if res.Com1Log != nil {
		retVMConfig.Com1Log = *res.Com1Log
	}
	if res.Com1Dev != nil {
		retVMConfig.Com1Dev = *res.Com1Dev
	}
	if res.Com1Speed != nil {
		retVMConfig.Com1Speed = *res.Com1Speed
	}

	return retVMConfig
}

func parseOptionalVMConfigSerialCom2(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	if res.Com2 != nil {
		retVMConfig.Com2 = *res.Com2
	}
	if res.Com2Log != nil {
		retVMConfig.Com2Log = *res.Com2Log
	}
	if res.Com2Dev != nil {
		retVMConfig.Com2Dev = *res.Com2Dev
	}
	if res.Com2Speed != nil {
		retVMConfig.Com2Speed = *res.Com2Speed
	}

	return retVMConfig
}

func parseOptionalVMConfigSerialCom3(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	if res.Com3 != nil {
		retVMConfig.Com3 = *res.Com3
	}
	if res.Com3Log != nil {
		retVMConfig.Com3Log = *res.Com3Log
	}
	if res.Com3Dev != nil {
		retVMConfig.Com3Dev = *res.Com3Dev
	}
	if res.Com3Speed != nil {
		retVMConfig.Com3Speed = *res.Com3Speed
	}

	return retVMConfig
}

func parseOptionalVMConfigSerialCom4(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	if res.Com4 != nil {
		retVMConfig.Com4 = *res.Com4
	}
	if res.Com4Log != nil {
		retVMConfig.Com4Log = *res.Com4Log
	}
	if res.Com4Dev != nil {
		retVMConfig.Com4Dev = *res.Com4Dev
	}
	if res.Com4Speed != nil {
		retVMConfig.Com4Speed = *res.Com4Speed
	}

	return retVMConfig
}

func parseOptionalVMConfigScreen(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	if res.Screen != nil {
		retVMConfig.Screen = *res.Screen
	}
	if res.Vncport != nil {
		retVMConfig.Vncport = *res.Vncport
	}
	if res.ScreenWidth != nil {
		retVMConfig.ScreenWidth = *res.ScreenWidth
	}
	if res.ScreenHeight != nil {
		retVMConfig.ScreenHeight = *res.ScreenHeight
	}
	if res.Vncwait != nil {
		retVMConfig.Vncwait = *res.Vncwait
	}
	if res.Tablet != nil {
		retVMConfig.Tablet = *res.Tablet
	}
	if res.Keyboard != nil {
		retVMConfig.Keyboard = *res.Keyboard
	}

	return retVMConfig
}

func parseOptionalVMConfigSound(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	if res.Sound != nil {
		retVMConfig.Sound = *res.Sound
	}
	if res.SoundIn != nil {
		retVMConfig.SoundIn = *res.SoundIn
	}
	if res.SoundOut != nil {
		retVMConfig.SoundOut = *res.SoundOut
	}

	return retVMConfig
}

func parseOptionalVMConfigStart(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	if res.Autostart != nil {
		retVMConfig.Autostart = *res.Autostart
	}
	if res.AutostartDelay != nil {
		retVMConfig.AutostartDelay = *res.AutostartDelay
	}
	if res.Restart != nil {
		retVMConfig.Restart = *res.Restart
	}
	if res.RestartDelay != nil {
		retVMConfig.RestartDelay = *res.RestartDelay
	}

	return retVMConfig
}

func parseOptionalVMConfigAdvanced(res *cirrina.VMConfig, retVMConfig VMConfig) VMConfig {
	if res.MaxWait != nil {
		retVMConfig.MaxWait = *res.MaxWait
	}
	if res.Storeuefi != nil {
		retVMConfig.Storeuefi = *res.Storeuefi
	}
	if res.Utc != nil {
		retVMConfig.Utc = *res.Utc
	}
	if res.Dpo != nil {
		retVMConfig.Dpo = *res.Dpo
	}
	if res.Wireguestmem != nil {
		retVMConfig.Wireguestmem = *res.Wireguestmem
	}
	if res.Hostbridge != nil {
		retVMConfig.Hostbridge = *res.Hostbridge
	}
	if res.Acpi != nil {
		retVMConfig.Acpi = *res.Acpi
	}
	if res.Eop != nil {
		retVMConfig.Eop = *res.Eop
	}
	if res.Ium != nil {
		retVMConfig.Ium = *res.Ium
	}
	if res.Hlt != nil {
		retVMConfig.Hlt = *res.Hlt
	}
	if res.Debug != nil {
		retVMConfig.Debug = *res.Debug
	}
	if res.DebugWait != nil {
		retVMConfig.DebugWait = *res.DebugWait
	}
	if res.DebugPort != nil {
		retVMConfig.DebugPort = *res.DebugPort
	}
	if res.ExtraArgs != nil {
		retVMConfig.ExtraArgs = *res.ExtraArgs
	}

	return retVMConfig
}
