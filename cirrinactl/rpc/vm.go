package rpc

import (
	"cirrina/cirrina"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"io"
	"strconv"
)

func AddVM(name string, descrPtr *string, cpuPtr *uint32, memPtr *uint32) (string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	if name == "" {
		return "", errors.New("name not specified")
	}

	VmConfig := &cirrina.VMConfig{
		Name: &name,
	}

	if descrPtr != nil {
		VmConfig.Description = descrPtr
	}

	if cpuPtr != nil {
		VmConfig.Cpu = cpuPtr
	}

	if memPtr != nil {
		VmConfig.Mem = memPtr
	}

	var res *cirrina.VMID
	res, err = c.AddVM(ctx, VmConfig)
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	return res.Value, nil
}

func DeleteVM(id string) (string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	if id == "" {
		return "", errors.New("id not specified")
	}
	var reqId *cirrina.RequestID
	reqId, err = c.DeleteVM(ctx, &cirrina.VMID{Value: id})
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	return reqId.Value, nil
}

func StopVM(id string) (string, error) {

	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	if id == "" {
		return "", errors.New("id not specified")
	}
	var reqId *cirrina.RequestID
	reqId, err = c.StopVM(ctx, &cirrina.VMID{Value: id})
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	return reqId.Value, nil
}

func StartVM(id string) (string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	if id == "" {
		return "", errors.New("id not specified")
	}
	var reqId *cirrina.RequestID
	reqId, err = c.StartVM(ctx, &cirrina.VMID{Value: id})
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	return reqId.Value, nil
}

func GetVmName(id string) (string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	if id == "" {
		return "", errors.New("id not specified")
	}
	var res *wrapperspb.StringValue
	res, err = c.GetVmName(ctx, &cirrina.VMID{Value: id})
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	return res.GetValue(), nil
}

func GetVmId(name string) (string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	if name == "" {
		return "", errors.New("name not specified")
	}
	var res *cirrina.VMID
	res, err = c.GetVmId(ctx, wrapperspb.String(name))
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	return res.Value, nil
}

func GetVMConfig(id string) (VmConfig, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return VmConfig{}, err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	if id == "" {
		return VmConfig{}, errors.New("id not specified")
	}
	var res *cirrina.VMConfig
	res, err = c.GetVMConfig(ctx, &cirrina.VMID{Value: id})
	if err != nil {
		return VmConfig{}, errors.New(status.Convert(err).Message())
	}
	var rv VmConfig
	rv.Id = res.Id
	if res.Name != nil {
		rv.Name = res.Name
	}
	if res.Description != nil {
		rv.Description = res.Description
	}
	if res.Cpu != nil {
		rv.Cpu = res.Cpu
	}
	if res.Mem != nil {
		rv.Mem = res.Mem
	}
	if res.Priority != nil {
		rv.Priority = res.Priority
	}
	if res.Protect != nil {
		rv.Protect = res.Protect
	}
	if res.Pcpu != nil {
		rv.Pcpu = res.Pcpu
	}

	if res.Rbps != nil {
		rv.Rbps = res.Rbps
	}
	if res.Wbps != nil {
		rv.Wbps = res.Wbps
	}
	if res.Riops != nil {
		rv.Riops = res.Riops
	}
	if res.Wiops != nil {
		rv.Wiops = res.Wiops
	}
	if res.Com1 != nil {
		rv.Com1 = res.Com1
	}
	if res.Com1Log != nil {
		rv.Com1Log = res.Com1Log
	}
	if res.Com1Dev != nil {
		rv.Com1Dev = res.Com1Dev
	}
	if res.Com1Speed != nil {
		rv.Com1Speed = res.Com1Speed
	}

	if res.Com2 != nil {
		rv.Com2 = res.Com2
	}
	if res.Com2Log != nil {
		rv.Com2Log = res.Com2Log
	}
	if res.Com2Dev != nil {
		rv.Com2Dev = res.Com2Dev
	}
	if res.Com2Speed != nil {
		rv.Com2Speed = res.Com2Speed
	}

	if res.Com3 != nil {
		rv.Com3 = res.Com3
	}
	if res.Com3Log != nil {
		rv.Com3Log = res.Com3Log
	}
	if res.Com3Dev != nil {
		rv.Com3Dev = res.Com3Dev
	}
	if res.Com3Speed != nil {
		rv.Com3Speed = res.Com3Speed
	}

	if res.Com4 != nil {
		rv.Com4 = res.Com4
	}
	if res.Com4Log != nil {
		rv.Com4Log = res.Com4Log
	}
	if res.Com4Dev != nil {
		rv.Com4Dev = res.Com4Dev
	}
	if res.Com4Speed != nil {
		rv.Com4Speed = res.Com4Speed
	}

	if res.Screen != nil {
		rv.Screen = res.Screen
	}
	if res.Vncport != nil {
		rv.Vncport = res.Vncport
	}
	if res.ScreenWidth != nil {
		rv.ScreenWidth = res.ScreenWidth
	}
	if res.ScreenHeight != nil {
		rv.ScreenHeight = res.ScreenHeight
	}
	if res.Vncwait != nil {
		rv.Vncwait = res.Vncwait
	}
	if res.Tablet != nil {
		rv.Tablet = res.Tablet
	}
	if res.Keyboard != nil {
		rv.Keyboard = res.Keyboard
	}

	if res.Sound != nil {
		rv.Sound = res.Sound
	}
	if res.SoundIn != nil {
		rv.SoundIn = res.SoundIn
	}
	if res.SoundOut != nil {
		rv.SoundOut = res.SoundOut
	}

	if res.Autostart != nil {
		rv.Autostart = res.Autostart
	}
	if res.AutostartDelay != nil {
		rv.AutostartDelay = res.AutostartDelay
	}
	if res.Restart != nil {
		rv.Restart = res.Restart
	}
	if res.RestartDelay != nil {
		rv.RestartDelay = res.RestartDelay
	}
	if res.MaxWait != nil {
		rv.MaxWait = res.MaxWait
	}

	if res.Storeuefi != nil {
		rv.Storeuefi = res.Storeuefi
	}
	if res.Utc != nil {
		rv.Utc = res.Utc
	}
	if res.Dpo != nil {
		rv.Dpo = res.Dpo
	}
	if res.Wireguestmem != nil {
		rv.Wireguestmem = res.Wireguestmem
	}
	if res.Hostbridge != nil {
		rv.Hostbridge = res.Hostbridge
	}
	if res.Acpi != nil {
		rv.Acpi = res.Acpi
	}
	if res.Eop != nil {
		rv.Eop = res.Eop
	}
	if res.Ium != nil {
		rv.Ium = res.Ium
	}
	if res.Hlt != nil {
		rv.Hlt = res.Hlt
	}
	if res.Debug != nil {
		rv.Debug = res.Debug
	}
	if res.DebugWait != nil {
		rv.DebugWait = res.DebugWait
	}
	if res.DebugPort != nil {
		rv.DebugPort = res.DebugPort
	}

	if res.ExtraArgs != nil {
		rv.ExtraArgs = res.ExtraArgs
	}

	return rv, nil
}

func GetVmIds() ([]string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return []string{}, err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var res cirrina.VMInfo_GetVMsClient
	res, err = c.GetVMs(ctx, &cirrina.VMsQuery{})
	var ids []string
	if err != nil {
		return []string{}, errors.New(status.Convert(err).Message())
	}
	for {
		var VM *cirrina.VMID
		VM, err = res.Recv()
		if err == io.EOF {
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

	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", "", "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	if id == "" {
		return "", "", "", errors.New("id not specified")
	}
	var res *cirrina.VMState
	res, err = c.GetVMState(ctx, &cirrina.VMID{Value: id})
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

func VmRunning(id string) (bool, error) {
	r, _, _, err := GetVMState(id)
	if err != nil {
		return false, err
	}
	if r == "running" {
		return true, nil
	}
	return false, nil
}

func VmStopped(id string) (bool, error) {
	r, _, _, err := GetVMState(id)
	if err != nil {
		return false, err
	}
	if r == "stopped" {
		return true, nil
	}
	return false, nil

}

func VmNameToId(name string) (string, error) {
	res, err := GetVmId(name)
	if err != nil {
		return "", err
	}
	if res == "" {
		return "", errors.New("VM not found")
	}
	return res, nil
}

func VmIdToName(id string) (string, error) {
	res, err := GetVmName(id)
	if err != nil {
		return "", err
	}
	if res == "" {
		return "", errors.New("VM not found")
	}
	return res, nil
}

func UpdateVMConfig(id string, newConfig VmConfig) error {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var myNewConfig cirrina.VMConfig
	myNewConfig.Id = id

	if newConfig.Description != nil {
		myNewConfig.Description = newConfig.Description
	}

	if newConfig.Cpu != nil {
		myNewConfig.Cpu = newConfig.Cpu
	}

	if newConfig.Mem != nil {
		myNewConfig.Mem = newConfig.Mem
	}

	if newConfig.Priority != nil {
		myNewConfig.Priority = newConfig.Priority
	}

	if newConfig.Protect != nil {
		myNewConfig.Protect = newConfig.Protect
	}

	if newConfig.Pcpu != nil {
		myNewConfig.Pcpu = newConfig.Pcpu
	}

	if newConfig.Rbps != nil {
		myNewConfig.Rbps = newConfig.Rbps
	}

	if newConfig.Wbps != nil {
		myNewConfig.Wbps = newConfig.Wbps
	}

	if newConfig.Riops != nil {
		myNewConfig.Riops = newConfig.Riops
	}

	if newConfig.Wiops != nil {
		myNewConfig.Wiops = newConfig.Wiops
	}

	if newConfig.Autostart != nil {
		myNewConfig.Autostart = newConfig.Autostart
	}

	if newConfig.AutostartDelay != nil {
		myNewConfig.AutostartDelay = newConfig.AutostartDelay
	}

	if newConfig.Restart != nil {
		myNewConfig.Restart = newConfig.Restart
	}

	if newConfig.RestartDelay != nil {
		myNewConfig.RestartDelay = newConfig.RestartDelay
	}

	if newConfig.MaxWait != nil {
		myNewConfig.MaxWait = newConfig.MaxWait
	}

	if newConfig.Screen != nil {
		myNewConfig.Screen = newConfig.Screen
	}

	if newConfig.ScreenWidth != nil {
		myNewConfig.ScreenWidth = newConfig.ScreenWidth
	}

	if newConfig.ScreenHeight != nil {
		myNewConfig.ScreenHeight = newConfig.ScreenHeight
	}

	if newConfig.Vncport != nil {
		myNewConfig.Vncport = newConfig.Vncport
	}

	if newConfig.Vncwait != nil {
		myNewConfig.Vncwait = newConfig.Vncwait
	}

	if newConfig.Tablet != nil {
		myNewConfig.Tablet = newConfig.Tablet
	}

	if newConfig.Keyboard != nil {
		myNewConfig.Keyboard = newConfig.Keyboard
	}

	if newConfig.Sound != nil {
		myNewConfig.Sound = newConfig.Sound
	}

	if newConfig.SoundIn != nil {
		myNewConfig.SoundIn = newConfig.SoundIn
	}

	if newConfig.SoundOut != nil {
		myNewConfig.SoundOut = newConfig.SoundOut
	}

	if newConfig.Com1 != nil {
		myNewConfig.Com1 = newConfig.Com1
	}

	if newConfig.Com1Log != nil {
		myNewConfig.Com1Log = newConfig.Com1Log
	}

	if newConfig.Com1Dev != nil {
		myNewConfig.Com1Dev = newConfig.Com1Dev
	}

	if newConfig.Com1Speed != nil {
		myNewConfig.Com1Speed = newConfig.Com1Speed
	}

	if newConfig.Com2 != nil {
		myNewConfig.Com2 = newConfig.Com2
	}

	if newConfig.Com2Log != nil {
		myNewConfig.Com2Log = newConfig.Com2Log
	}

	if newConfig.Com2Dev != nil {
		myNewConfig.Com2Dev = newConfig.Com2Dev
	}

	if newConfig.Com2Speed != nil {
		myNewConfig.Com2Speed = newConfig.Com2Speed
	}

	if newConfig.Com3 != nil {
		myNewConfig.Com3 = newConfig.Com3
	}

	if newConfig.Com3Log != nil {
		myNewConfig.Com3Log = newConfig.Com3Log
	}

	if newConfig.Com3Dev != nil {
		myNewConfig.Com3Dev = newConfig.Com3Dev
	}

	if newConfig.Com3Speed != nil {
		myNewConfig.Com3Speed = newConfig.Com3Speed
	}

	if newConfig.Com4 != nil {
		myNewConfig.Com4 = newConfig.Com4
	}

	if newConfig.Com4Log != nil {
		myNewConfig.Com4Log = newConfig.Com4Log
	}

	if newConfig.Com4Dev != nil {
		myNewConfig.Com4Dev = newConfig.Com4Dev
	}

	if newConfig.Com4Speed != nil {
		myNewConfig.Com4Speed = newConfig.Com4Speed
	}

	if newConfig.Wireguestmem != nil {
		myNewConfig.Wireguestmem = newConfig.Wireguestmem
	}

	if newConfig.Storeuefi != nil {
		myNewConfig.Storeuefi = newConfig.Storeuefi
	}

	if newConfig.Utc != nil {
		myNewConfig.Utc = newConfig.Utc
	}

	if newConfig.Hostbridge != nil {
		myNewConfig.Hostbridge = newConfig.Hostbridge
	}

	if newConfig.Acpi != nil {
		myNewConfig.Acpi = newConfig.Acpi
	}

	if newConfig.Hlt != nil {
		myNewConfig.Hlt = newConfig.Hlt
	}

	if newConfig.Eop != nil {
		myNewConfig.Eop = newConfig.Eop
	}

	if newConfig.Dpo != nil {
		myNewConfig.Dpo = newConfig.Dpo
	}

	if newConfig.Ium != nil {
		myNewConfig.Ium = newConfig.Ium
	}

	if newConfig.Debug != nil {
		myNewConfig.Debug = newConfig.Debug
	}

	if newConfig.DebugWait != nil {
		myNewConfig.DebugWait = newConfig.DebugWait
	}

	if newConfig.DebugPort != nil {
		myNewConfig.DebugPort = newConfig.DebugPort
	}

	if newConfig.ExtraArgs != nil {
		myNewConfig.ExtraArgs = newConfig.ExtraArgs
	}

	_, err = c.UpdateVM(ctx, &myNewConfig)
	if err != nil {
		return errors.New(status.Convert(err).Message())
	}
	return nil
}

func VmClearUefiVars(id string) (bool, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return false, err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var res *cirrina.ReqBool
	res, err = c.ClearUEFIState(ctx, &cirrina.VMID{Value: id})
	if err != nil {
		return false, errors.New(status.Convert(err).Message())
	}
	return res.Success, nil
}
