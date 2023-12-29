package cmd

import (
	"cirrina/cirrinactl/rpc"
	"fmt"
	"github.com/spf13/cobra"
)

var HostNicsCmd = &cobra.Command{
	Use:          "getnics",
	Short:        "Get list of host nics",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := rpc.GetHostNics()
		if err != nil {
			return err
		}
		for _, nic := range res {
			fmt.Printf("nic: %s\n", nic.InterfaceName)
		}
		return nil
	},
}

var HostVersionCmd = &cobra.Command{
	Use:          "version",
	Short:        "Get host daemon version",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := rpc.GetHostVersion()
		if err != nil {
			return err
		}
		fmt.Printf("version: %s\n", res)
		return nil
	},
}

var HostCmd = &cobra.Command{
	Use:   "host",
	Short: "Commands related to VM server host",
}

func init() {
	disableFlagSorting(HostCmd)
	disableFlagSorting(HostVersionCmd)
	disableFlagSorting(HostNicsCmd)

	HostCmd.AddCommand(HostNicsCmd)
	HostCmd.AddCommand(HostVersionCmd)
}
