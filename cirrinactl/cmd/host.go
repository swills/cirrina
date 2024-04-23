package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"cirrina/cirrinactl/rpc"
)

var HostNicsCmd = &cobra.Command{
	Use:          "getnics",
	Short:        "Get list of host nics",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		res, err := rpc.GetHostNics()
		if err != nil {
			return fmt.Errorf("failed getting host nics: %w", err)
		}
		for _, nic := range res {
			fmt.Printf("nic: %s\n", nic)
		}

		return nil
	},
}

var HostVersionCmd = &cobra.Command{
	Use:          "version",
	Short:        "Get host daemon version",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error
		var res string
		err = hostPing()
		if err != nil {
			return fmt.Errorf("failed getting host version: %w", err)
		}

		res, err = rpc.GetHostVersion()
		if err != nil {
			return fmt.Errorf("failed getting host version: %w", err)
		}
		fmt.Printf("version: %s\n", res)

		return nil
	},
}

var HostCmd = &cobra.Command{
	Use:   "host",
	Short: "Commands related to VM server host",
}

func hostPing() error {
	_, err := rpc.GetHostVersion()
	if err != nil {
		return fmt.Errorf("host not available: %w", err)
	}

	return nil
}
