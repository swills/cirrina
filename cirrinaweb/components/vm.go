package components

import (
	"fmt"

	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinaweb/util"
)

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
