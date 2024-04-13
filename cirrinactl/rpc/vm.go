package rpc

import (
	"errors"
	"io"
	"strconv"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"cirrina/cirrina"
)

func AddVM(name string, descrPtr *string, cpuPtr *uint32, memPtr *uint32) (string, error) {
	var err error

	if name == "" {
		return "", errors.New("name not specified")
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
		return "", errors.New(status.Convert(err).Message())
	}

	return res.Value, nil
}

func DeleteVM(id string) (string, error) {
	var err error

	if id == "" {
		return "", errors.New("id not specified")
	}
	var reqID *cirrina.RequestID
	reqID, err = serverClient.DeleteVM(defaultServerContext, &cirrina.VMID{Value: id})
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}

	return reqID.Value, nil
}

func StopVM(id string) (string, error) {
	var err error

	if id == "" {
		return "", errors.New("id not specified")
	}
	var reqID *cirrina.RequestID
	reqID, err = serverClient.StopVM(defaultServerContext, &cirrina.VMID{Value: id})
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}

	return reqID.Value, nil
}

func StartVM(id string) (string, error) {
	var err error

	if id == "" {
		return "", errors.New("id not specified")
	}
	var reqID *cirrina.RequestID
	reqID, err = serverClient.StartVM(defaultServerContext, &cirrina.VMID{Value: id})
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}

	return reqID.Value, nil
}

func GetVMName(id string) (string, error) {
	var err error

	if id == "" {
		return "", errors.New("id not specified")
	}
	var res *wrapperspb.StringValue
	res, err = serverClient.GetVMName(defaultServerContext, &cirrina.VMID{Value: id})
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}

	return res.GetValue(), nil
}

func GetVMId(name string) (string, error) {
	var err error

	if name == "" {
		return "", errors.New("name not specified")
	}
	var res *cirrina.VMID
	res, err = serverClient.GetVMID(defaultServerContext, wrapperspb.String(name))
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}

	return res.Value, nil
}

func GetVMConfig(id string) (VMConfig, error) {
	var err error

	if id == "" {
		return VMConfig{}, errors.New("id not specified")
	}
	var res *cirrina.VMConfig
	res, err = serverClient.GetVMConfig(defaultServerContext, &cirrina.VMID{Value: id})
	if err != nil {
		return VMConfig{}, errors.New(status.Convert(err).Message())
	}
	var rv VMConfig
	rv.ID = res.Id

	rv = parseOptionalVMConfigBasic(res, rv)
	rv = parseOptionalVMConfigPriority(res, rv)
	rv = parseOptionalVMConfigSerialCom1(res, rv)
	rv = parseOptionalVMConfigSerialCom2(res, rv)
	rv = parseOptionalVMConfigSerialCom3(res, rv)
	rv = parseOptionalVMConfigSerialCom4(res, rv)
	rv = parseOptionalVMConfigScreen(res, rv)
	rv = parseOptionalVMConfigSound(res, rv)
	rv = parseOptionalVMConfigStart(res, rv)
	rv = parseOptionalVMConfigAdvanced(res, rv)

	return rv, nil
}

func GetVMIds() ([]string, error) {
	var err error

	var res cirrina.VMInfo_GetVMsClient
	res, err = serverClient.GetVMs(defaultServerContext, &cirrina.VMsQuery{})
	var ids []string
	if err != nil {
		return []string{}, errors.New(status.Convert(err).Message())
	}
	for {
		var VM *cirrina.VMID
		VM, err = res.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return []string{}, errors.New(status.Convert(err).Message())
		}
		ids = append(ids, VM.Value)
	}

	return ids, nil
}

func GetVMState(id string) (string, string, string, error) {
	var err error

	if id == "" {
		return "", "", "", errors.New("id not specified")
	}
	var res *cirrina.VMState
	res, err = serverClient.GetVMState(defaultServerContext, &cirrina.VMID{Value: id})
	if err != nil {
		return "", "", "", errors.New(status.Convert(err).Message())
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

func VMRunning(id string) (bool, error) {
	r, _, _, err := GetVMState(id)
	if err != nil {
		return false, err
	}
	if r == "running" {
		return true, nil
	}

	return false, nil
}

func VMStopped(id string) (bool, error) {
	r, _, _, err := GetVMState(id)
	if err != nil {
		return false, err
	}
	if r == "stopped" {
		return true, nil
	}

	return false, nil
}

func VMNameToID(name string) (string, error) {
	res, err := GetVMId(name)
	if err != nil {
		return "", err
	}
	if res == "" {
		return "", errors.New("VM not found")
	}

	return res, nil
}

func VMIdToName(id string) (string, error) {
	res, err := GetVMName(id)
	if err != nil {
		return "", err
	}
	if res == "" {
		return "", errors.New("VM not found")
	}

	return res, nil
}

func UpdateVMConfig(myNewConfig *cirrina.VMConfig) error {
	var err error

	_, err = serverClient.UpdateVM(defaultServerContext, myNewConfig)
	if err != nil {
		return errors.New(status.Convert(err).Message())
	}

	return nil
}

func VMClearUefiVars(id string) (bool, error) {
	var err error
	var res *cirrina.ReqBool
	res, err = serverClient.ClearUEFIState(defaultServerContext, &cirrina.VMID{Value: id})
	if err != nil {
		return false, errors.New(status.Convert(err).Message())
	}

	return res.Success, nil
}

func parseOptionalVMConfigBasic(res *cirrina.VMConfig, rv VMConfig) VMConfig {
	if res.Name != nil {
		rv.Name = *res.Name
	}
	if res.Description != nil {
		rv.Description = *res.Description
	}
	if res.Cpu != nil {
		rv.CPU = *res.Cpu
	}
	if res.Mem != nil {
		rv.Mem = *res.Mem
	}

	return rv
}

func parseOptionalVMConfigPriority(res *cirrina.VMConfig, rv VMConfig) VMConfig {
	if res.Priority != nil {
		rv.Priority = *res.Priority
	}
	if res.Protect != nil {
		rv.Protect = *res.Protect
	}
	if res.Pcpu != nil {
		rv.Pcpu = *res.Pcpu
	}
	if res.Rbps != nil {
		rv.Rbps = *res.Rbps
	}
	if res.Wbps != nil {
		rv.Wbps = *res.Wbps
	}
	if res.Riops != nil {
		rv.Riops = *res.Riops
	}
	if res.Wiops != nil {
		rv.Wiops = *res.Wiops
	}

	return rv
}

func parseOptionalVMConfigSerialCom1(res *cirrina.VMConfig, rv VMConfig) VMConfig {
	if res.Com1 != nil {
		rv.Com1 = *res.Com1
	}
	if res.Com1Log != nil {
		rv.Com1Log = *res.Com1Log
	}
	if res.Com1Dev != nil {
		rv.Com1Dev = *res.Com1Dev
	}
	if res.Com1Speed != nil {
		rv.Com1Speed = *res.Com1Speed
	}

	return rv
}

func parseOptionalVMConfigSerialCom2(res *cirrina.VMConfig, rv VMConfig) VMConfig {
	if res.Com2 != nil {
		rv.Com2 = *res.Com2
	}
	if res.Com2Log != nil {
		rv.Com2Log = *res.Com2Log
	}
	if res.Com2Dev != nil {
		rv.Com2Dev = *res.Com2Dev
	}
	if res.Com2Speed != nil {
		rv.Com2Speed = *res.Com2Speed
	}

	return rv
}

func parseOptionalVMConfigSerialCom3(res *cirrina.VMConfig, rv VMConfig) VMConfig {
	if res.Com3 != nil {
		rv.Com3 = *res.Com3
	}
	if res.Com3Log != nil {
		rv.Com3Log = *res.Com3Log
	}
	if res.Com3Dev != nil {
		rv.Com3Dev = *res.Com3Dev
	}
	if res.Com3Speed != nil {
		rv.Com3Speed = *res.Com3Speed
	}

	return rv
}

func parseOptionalVMConfigSerialCom4(res *cirrina.VMConfig, rv VMConfig) VMConfig {
	if res.Com4 != nil {
		rv.Com4 = *res.Com4
	}
	if res.Com4Log != nil {
		rv.Com4Log = *res.Com4Log
	}
	if res.Com4Dev != nil {
		rv.Com4Dev = *res.Com4Dev
	}
	if res.Com4Speed != nil {
		rv.Com4Speed = *res.Com4Speed
	}

	return rv
}

func parseOptionalVMConfigScreen(res *cirrina.VMConfig, rv VMConfig) VMConfig {
	if res.Screen != nil {
		rv.Screen = *res.Screen
	}
	if res.Vncport != nil {
		rv.Vncport = *res.Vncport
	}
	if res.ScreenWidth != nil {
		rv.ScreenWidth = *res.ScreenWidth
	}
	if res.ScreenHeight != nil {
		rv.ScreenHeight = *res.ScreenHeight
	}
	if res.Vncwait != nil {
		rv.Vncwait = *res.Vncwait
	}
	if res.Tablet != nil {
		rv.Tablet = *res.Tablet
	}
	if res.Keyboard != nil {
		rv.Keyboard = *res.Keyboard
	}

	return rv
}

func parseOptionalVMConfigSound(res *cirrina.VMConfig, rv VMConfig) VMConfig {
	if res.Sound != nil {
		rv.Sound = *res.Sound
	}
	if res.SoundIn != nil {
		rv.SoundIn = *res.SoundIn
	}
	if res.SoundOut != nil {
		rv.SoundOut = *res.SoundOut
	}

	return rv
}

func parseOptionalVMConfigStart(res *cirrina.VMConfig, rv VMConfig) VMConfig {
	if res.Autostart != nil {
		rv.Autostart = *res.Autostart
	}
	if res.AutostartDelay != nil {
		rv.AutostartDelay = *res.AutostartDelay
	}
	if res.Restart != nil {
		rv.Restart = *res.Restart
	}
	if res.RestartDelay != nil {
		rv.RestartDelay = *res.RestartDelay
	}

	return rv
}

func parseOptionalVMConfigAdvanced(res *cirrina.VMConfig, rv VMConfig) VMConfig {
	if res.MaxWait != nil {
		rv.MaxWait = *res.MaxWait
	}
	if res.Storeuefi != nil {
		rv.Storeuefi = *res.Storeuefi
	}
	if res.Utc != nil {
		rv.Utc = *res.Utc
	}
	if res.Dpo != nil {
		rv.Dpo = *res.Dpo
	}
	if res.Wireguestmem != nil {
		rv.Wireguestmem = *res.Wireguestmem
	}
	if res.Hostbridge != nil {
		rv.Hostbridge = *res.Hostbridge
	}
	if res.Acpi != nil {
		rv.Acpi = *res.Acpi
	}
	if res.Eop != nil {
		rv.Eop = *res.Eop
	}
	if res.Ium != nil {
		rv.Ium = *res.Ium
	}
	if res.Hlt != nil {
		rv.Hlt = *res.Hlt
	}
	if res.Debug != nil {
		rv.Debug = *res.Debug
	}
	if res.DebugWait != nil {
		rv.DebugWait = *res.DebugWait
	}
	if res.DebugPort != nil {
		rv.DebugPort = *res.DebugPort
	}
	if res.ExtraArgs != nil {
		rv.ExtraArgs = *res.ExtraArgs
	}

	return rv
}
