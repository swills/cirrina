package cmd

import (
	"cirrina/cirrinactl/rpc"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"time"
)

var VmCom1Cmd = &cobra.Command{
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

var VmCom2Cmd = &cobra.Command{
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

var VmCom3Cmd = &cobra.Command{
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

var VmCom4Cmd = &cobra.Command{
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
	if VmId == "" {
		VmId, err = rpc.VmNameToId(VmName)
		if err != nil {
			return err
		}
		if VmId == "" {
			return errors.New("VM not found")
		}
	}
	var running bool
	running, err = rpc.VmRunning(VmId)
	if err != nil {
		return err
	}
	if !running {
		return errors.New("vm not running")
	}

	fmt.Print("starting terminal session, press ctrl-\\ to quit\n")
	time.Sleep(1 * time.Second)

	err = rpc.UseCom(VmId, comNum)
	if err != nil {
		return err
	}
	return nil
}

func init() {
	disableFlagSorting(VmCom1Cmd)
	addNameOrIdArgs(VmCom1Cmd, &VmName, &VmId, "VM")

	disableFlagSorting(VmCom2Cmd)
	addNameOrIdArgs(VmCom2Cmd, &VmName, &VmId, "VM")

	disableFlagSorting(VmCom3Cmd)
	addNameOrIdArgs(VmCom3Cmd, &VmName, &VmId, "VM")

	disableFlagSorting(VmCom4Cmd)
	addNameOrIdArgs(VmCom4Cmd, &VmName, &VmId, "VM")
}
