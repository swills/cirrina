package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"cirrina/cirrinactl/rpc"
)

var ReqID string

var ReqStatCmd = &cobra.Command{
	Use:          "reqstat",
	Short:        "Get status of request",
	Long:         "Check if a server request has completed and if it was successful",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		res, err := rpc.ReqStat(ctx, ReqID)
		if err != nil {
			return fmt.Errorf("error checking request status: %w", err)
		}
		fmt.Printf("req status: complete=%v, success=%v\n", res.Complete, res.Success)

		return nil
	},
}
