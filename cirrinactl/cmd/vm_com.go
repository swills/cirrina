package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"cirrina/cirrinactl/rpc"
)

var VMCom1Cmd = &cobra.Command{
	Use:          "useCom1",
	Short:        "Connect to VMs Com1",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := startCom(1)
		if err != nil {
			return err
		}

		return nil
	},
}

var VMCom2Cmd = &cobra.Command{
	Use:          "useCom2",
	Short:        "Connect to VMs Com2",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := startCom(2)
		if err != nil {
			return err
		}

		return nil
	},
}

var VMCom3Cmd = &cobra.Command{
	Use:          "useCom3",
	Short:        "Connect to VMs Com3",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := startCom(3)
		if err != nil {
			return err
		}

		return nil
	},
}

var VMCom4Cmd = &cobra.Command{
	Use:          "useCom4",
	Short:        "Connect to VMs Com4",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := startCom(4)
		if err != nil {
			return err
		}

		return nil
	},
}

func startCom(comNum int) error {
	var err error
	if VMID == "" {
		VMID, err = rpc.VMNameToID(VMName)
		if err != nil {
			return err
		}
		if VMID == "" {
			return errors.New("VM not found")
		}
	}
	var running bool
	running, err = rpc.VMRunning(VMID)
	if err != nil {
		return err
	}
	if !running {
		if VMName != "" {
			return fmt.Errorf("vm %s not running", VMName)
		} else {
			return fmt.Errorf("vm %s not running", VMID)
		}
	}

	fmt.Print("starting terminal session, press ctrl-\\ to quit\n")
	time.Sleep(1 * time.Second)

	err = rpc.UseCom(VMID, comNum)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	disableFlagSorting(VMCom1Cmd)
	addNameOrIDArgs(VMCom1Cmd, &VMName, &VMID, "VM")

	disableFlagSorting(VMCom2Cmd)
	addNameOrIDArgs(VMCom2Cmd, &VMName, &VMID, "VM")

	disableFlagSorting(VMCom3Cmd)
	addNameOrIDArgs(VMCom3Cmd, &VMName, &VMID, "VM")

	disableFlagSorting(VMCom4Cmd)
	addNameOrIDArgs(VMCom4Cmd, &VMName, &VMID, "VM")
}
