package components

import (
	"fmt"

	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinaweb/util"
)

type RuntimeSettings struct {
	AutoStart        bool
	AutoRestart      bool
	AutoStartDelay   uint32
	AutoRestartDelay uint32
	ShutdownTimeout  uint32
}

type AdvancedSettings struct {
	StoreUEFI              bool
	Wire                   bool
	ExitOnPause            bool
	ClockUTC               bool
	HostBridge             bool
	IgnoreUnimplementedMSR bool
	DestroyOnPowerOff      bool
	GenerateACPITables     bool
	UseHLT                 bool
	StartDebugServer       bool
	WaitDebugConn          bool
	DebugPort              string
	ExtraArgs              string
}

type VM struct {
	ID               string
	Name             string
	NameOrID         string
	CPUs             uint32
	Memory           uint32
	Description      string
	Running          bool
	VNCPort          uint64
	Disks            []Disk
	ISOs             []ISO
	NICs             []NIC
	COM1             COM
	COM2             COM
	COM3             COM
	COM4             COM
	Display          Display
	Audio            Audio
	RuntimeSettings  RuntimeSettings
	AdvancedSettings AdvancedSettings
}

func (v VM) Start() error {
	err := util.InitRPCConn()
	if err != nil {
		return fmt.Errorf("error starting VM, failed to get connection: %w", err)
	}

	_, err = rpc.StartVM(v.ID)
	if err != nil {
		return fmt.Errorf("error starting VM: %w", err)
	}

	return nil
}

func (v VM) Stop() error {
	err := util.InitRPCConn()
	if err != nil {
		return fmt.Errorf("error starting VM, failed to get connection: %w", err)
	}

	_, err = rpc.StopVM(v.ID)
	if err != nil {
		return fmt.Errorf("error stopping VM: %w", err)
	}

	return nil
}

func (v VM) ClearUEFIVars() error {
	err := util.InitRPCConn()
	if err != nil {
		return fmt.Errorf("error clearing UEFI vars, failed to get connection: %w", err)
	}

	_, err = rpc.VMClearUefiVars(v.ID)
	if err != nil {
		return fmt.Errorf("error clearing UEFI vars: %w", err)
	}

	return nil
}
