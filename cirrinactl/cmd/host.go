package cmd

import (
	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinactl/util"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"log"
)

var HostNicsCmd = &cobra.Command{
	Use:   "getnics",
	Short: "Get list of host nics",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		util.GetNics(c, ctx)
	},
}

var HostVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Get host daemon version",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		util.GetHostVersion(c, ctx)
	},
}

var HostCmd = &cobra.Command{
	Use:   "host",
	Short: "Commands related to VM server host",
}

func init() {
	HostCmd.AddCommand(HostNicsCmd)
	HostCmd.AddCommand(HostVersionCmd)
}
