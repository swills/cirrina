package components

import (
	"fmt"

	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinaweb/util"
)

type VM struct {
	ID          string
	Name        string
	NameOrID    string
	CPUs        uint32
	Memory      uint32
	Description string
	Running     bool
	VNCPort     uint64
	Disks       []Disk
	ISOs        []ISO
	NICs        []NIC
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
