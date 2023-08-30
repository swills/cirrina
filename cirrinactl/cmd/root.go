package cmd

import (
	conn2 "cirrina/cirrinactl/rpc"
	"github.com/spf13/cobra"
	"os"
)

var VmName string
var VmId string

var rootCmd = &cobra.Command{
	Use: "cirrinactl",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&conn2.ServerName, "server", "s", "localhost", "server")
	rootCmd.PersistentFlags().Uint16VarP(&conn2.ServerPort, "port", "p", uint16(50051), "port")
	rootCmd.PersistentFlags().Uint64VarP(&conn2.ServerTimeout, "timeout", "t", uint64(1), "timeout in seconds")

	// some VM commands are duplicated at the root
	rootCmd.AddCommand(VmCreateCmd)
	rootCmd.AddCommand(VmListCmd)
	rootCmd.AddCommand(VmStartCmd)
	rootCmd.AddCommand(VmStopCmd)
	rootCmd.AddCommand(VmDestroyCmd)
	rootCmd.AddCommand(VmConfigCmd)
	rootCmd.AddCommand(VmGetCmd)
	rootCmd.AddCommand(VmCom1Cmd)
	rootCmd.AddCommand(VmCom2Cmd)
	rootCmd.AddCommand(VmCom3Cmd)
	rootCmd.AddCommand(VmCom4Cmd)

	rootCmd.AddCommand(VmCmd)
	rootCmd.AddCommand(DiskCmd)
	rootCmd.AddCommand(NicCmd)
	rootCmd.AddCommand(SwitchCmd)
	rootCmd.AddCommand(IsoCmd)
	rootCmd.AddCommand(TuiCmd)
	rootCmd.AddCommand(ReqStatCmd)
	rootCmd.AddCommand(HostCmd)
}
