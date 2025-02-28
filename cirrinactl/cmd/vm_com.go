package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"cirrina/cirrinactl/rpc"
)

var VMCom1Cmd = &cobra.Command{
	Use:          "useCom1",
	Short:        "Connect to VMs Com1",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		err := startCom(1)
		if err != nil {
			return fmt.Errorf("failed starting com: %w", err)
		}

		return nil
	},
}

var VMCom2Cmd = &cobra.Command{
	Use:          "useCom2",
	Short:        "Connect to VMs Com2",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		err := startCom(2)
		if err != nil {
			return fmt.Errorf("failed starting com: %w", err)
		}

		return nil
	},
}

var VMCom3Cmd = &cobra.Command{
	Use:          "useCom3",
	Short:        "Connect to VMs Com3",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		err := startCom(3)
		if err != nil {
			return fmt.Errorf("failed starting com: %w", err)
		}

		return nil
	},
}

var VMCom4Cmd = &cobra.Command{
	Use:          "useCom4",
	Short:        "Connect to VMs Com4",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		err := startCom(4)
		if err != nil {
			return fmt.Errorf("failed starting com: %w", err)
		}

		return nil
	},
}

func startCom(comNum int) error {
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
	defer cancel()

	if VMID == "" {
		VMID, err = rpc.VMNameToID(ctx, VMName)
		if err != nil {
			return fmt.Errorf("failed getting VM ID: %w", err)
		}

		if VMID == "" {
			return errVMNotFound
		}
	}

	var running bool

	running, err = rpc.VMRunning(ctx, VMID)
	if err != nil {
		return fmt.Errorf("failed checking VM status: %w", err)
	}

	if !running {
		return errVMNotRunning
	}

	fmt.Print("starting terminal session, press ctrl-\\ to quit\n")
	time.Sleep(25 * time.Millisecond)

	err = rpc.UseCom(VMID, comNum)
	if err != nil {
		return fmt.Errorf("failed starting com: %w", err)
	}

	return nil
}
