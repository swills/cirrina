package cmd

import (
	conn2 "cirrina/cirrinactl/rpc"
	"cirrina/cirrinactl/util"
	"github.com/spf13/cobra"
	"strconv"
)

var TuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start terminal UI",
	Run: func(cmd *cobra.Command, args []string) {
		serverAddr := conn2.ServerName + ":" + strconv.FormatInt(int64(conn2.ServerPort), 10)
		util.StartTui(serverAddr)
	},
}
