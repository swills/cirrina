package cmd

import (
	"fmt"

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
		res, err := rpc.ReqStat(ReqID)
		if err != nil {
			return fmt.Errorf("error checking request status: %w", err)
		}
		fmt.Printf("req status: complete=%v, success=%v\n", res.Complete, res.Success)

		return nil
	},
}

func init() {
	disableFlagSorting(ReqStatCmd)
	ReqStatCmd.Flags().StringVarP(&ReqID, "id", "i", ReqID, "ID of request")
	err := ReqStatCmd.MarkFlagRequired("id")
	if err != nil {
		panic(err)
	}
}
