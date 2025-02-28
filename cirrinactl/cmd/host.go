package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"cirrina/cirrinactl/rpc"
)

var HostNicsCmd = &cobra.Command{
	Use:          "getnics",
	Short:        "Get list of host nics",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		res, err := rpc.GetHostNics(ctx)
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

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		err = hostPing(ctx)
		if err != nil {
			return fmt.Errorf("failed getting host version: %w", err)
		}

		res, err = rpc.GetHostVersion(ctx)
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

func hostPing(ctx context.Context) error {
	_, err := rpc.GetHostVersion(ctx)
	if err != nil {
		return fmt.Errorf("host not available: %w", err)
	}

	return nil
}
