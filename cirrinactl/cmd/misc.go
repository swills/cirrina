package cmd

import (
	"cirrina/cirrinactl/rpc"
	"fmt"
	"github.com/spf13/cobra"
)

var ReqId string

var ReqStatCmd = &cobra.Command{
	Use:          "reqstat",
	Short:        "Get status of request",
	Long:         "Check if a server request has completed and if it was successful",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := rpc.ReqStat(ReqId)
		if err != nil {
			return err
		}
		fmt.Printf("req status: complete=%v, success=%v\n", res.Complete, res.Success)
		return nil
	},
}

func init() {
	ReqStatCmd.Flags().StringVarP(&ReqId, "id", "i", ReqId, "Id of request")
	err := ReqStatCmd.MarkFlagRequired("id")
	if err != nil {
		panic(err)
	}
}
