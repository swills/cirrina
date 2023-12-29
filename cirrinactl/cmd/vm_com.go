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
	VmCom1Cmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmCom1Cmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmCom1Cmd.MarkFlagsOneRequired("name", "id")
	VmCom1Cmd.MarkFlagsMutuallyExclusive("name", "id")
	VmCom1Cmd.Flags().SortFlags = false
	VmCom1Cmd.PersistentFlags().SortFlags = false
	VmCom1Cmd.InheritedFlags().SortFlags = false

	VmCom2Cmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmCom2Cmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmCom2Cmd.MarkFlagsOneRequired("name", "id")
	VmCom2Cmd.MarkFlagsMutuallyExclusive("name", "id")
	VmCom2Cmd.Flags().SortFlags = false
	VmCom2Cmd.PersistentFlags().SortFlags = false
	VmCom2Cmd.InheritedFlags().SortFlags = false

	VmCom3Cmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmCom3Cmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmCom3Cmd.MarkFlagsOneRequired("name", "id")
	VmCom3Cmd.MarkFlagsMutuallyExclusive("name", "id")
	VmCom3Cmd.Flags().SortFlags = false
	VmCom3Cmd.PersistentFlags().SortFlags = false
	VmCom3Cmd.InheritedFlags().SortFlags = false

	VmCom4Cmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmCom4Cmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmCom4Cmd.MarkFlagsOneRequired("name", "id")
	VmCom4Cmd.MarkFlagsMutuallyExclusive("name", "id")
	VmCom4Cmd.Flags().SortFlags = false
	VmCom4Cmd.PersistentFlags().SortFlags = false
	VmCom4Cmd.InheritedFlags().SortFlags = false
}
