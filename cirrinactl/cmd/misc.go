package cmd

import (
	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinactl/util"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"log"
)

var ReqId string

var ReqStatCmd = &cobra.Command{
	Use:   "reqstat",
	Short: "Get status of request",
	Long:  "Check if a server request has completed and if it was successful or not",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		util.ReqStat(&ReqId, c, ctx)
	},
}

func init() {
	ReqStatCmd.Flags().StringVarP(&ReqId, "id", "i", ReqId, "Id of request")
}
