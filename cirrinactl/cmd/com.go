package cmd

import (
	"cirrina/cirrinactl/rpc"
	"fmt"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"log"
	"time"
)

var VmCom1Cmd = &cobra.Command{
	Use:   "useCom1",
	Short: "Connect to VMs Com1",
	Run: func(cmd *cobra.Command, args []string) {
		startCom(1)
	},
}

var VmCom2Cmd = &cobra.Command{
	Use:   "useCom2",
	Short: "Connect to VMs Com2",
	Run: func(cmd *cobra.Command, args []string) {
		startCom(2)
	},
}

var VmCom3Cmd = &cobra.Command{
	Use:   "useCom3",
	Short: "Connect to VMs Com3",
	Run: func(cmd *cobra.Command, args []string) {
		startCom(3)
	},
}
var VmCom4Cmd = &cobra.Command{
	Use:   "useCom4",
	Short: "Connect to VMs Com4",
	Run: func(cmd *cobra.Command, args []string) {
		startCom(4)
	},
}

func startCom(comNum int) {
	conn, c, ctx, cancel, err := rpc.SetupConn()
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()

	if VmId == "" {
		VmId, err = rpc.VmNameToId(VmName, c, ctx)
		if err != nil {
			log.Fatalf(err.Error())
		}
		if VmId == "" {
			log.Fatalf("VM not found")
		}
	}
	running, err := rpc.VmRunning(&VmId, c, ctx)
	if err != nil {
		log.Fatalf(err.Error())
	}
	if !running {
		log.Fatalf("vm not running\n")
	}

	_ = conn.Close()

	conn, c, err = rpc.SetupConnNoTimeoutNoContext()
	if err != nil {
		log.Fatal(err)
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	fmt.Print("starting terminal session, press ctrl-\\ to quit\n")
	time.Sleep(1 * time.Second)

	err = rpc.UseCom(c, &VmId, comNum)
	if err != nil {
		log.Fatalf("failed to get stream: %v", err)
	}
}

func init() {
	VmCom1Cmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmCom1Cmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmCom1Cmd.MarkFlagsOneRequired("name", "id")
	VmCom2Cmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmCom2Cmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmCom2Cmd.MarkFlagsOneRequired("name", "id")
	VmCom3Cmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmCom3Cmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmCom3Cmd.MarkFlagsOneRequired("name", "id")
	VmCom4Cmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmCom4Cmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmCom4Cmd.MarkFlagsOneRequired("name", "id")
}
