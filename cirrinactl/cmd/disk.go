package cmd

import (
	conn2 "cirrina/cirrinactl/rpc"
	"cirrina/cirrinactl/util"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"log"
)

var DiskName string
var DiskDescription string
var DiskType = "nvme"
var DiskSize = "1G"
var DiskId string
var DiskIdChanged bool

var DiskListCmd = &cobra.Command{
	Use:   "list",
	Short: "list disks",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := conn2.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		err = util.GetDisks(c, ctx)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var DiskCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "create virtual disk",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := conn2.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		_, err = util.AddDisk(&DiskName, c, ctx, &DiskDescription, &DiskSize, &DiskType)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var DiskRemoveCmd = &cobra.Command{
	Use:   "destroy",
	Short: "remove virtual disk",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := conn2.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		util.RmDisk(DiskName, c, ctx)
	},
}

var DiskCmd = &cobra.Command{
	Use:   "disk",
	Short: "Create, list, modify, destroy virtual disks",
}

func init() {
	DiskCreateCmd.Flags().StringVarP(&DiskName, "name", "n", DiskName, "name of disk")
	err := DiskCreateCmd.MarkFlagRequired("name")
	if err != nil {
		log.Fatalf(err.Error())
	}
	DiskCreateCmd.Flags().StringVarP(&DiskSize, "size", "s", DiskName, "size of disk")
	err = DiskCreateCmd.MarkFlagRequired("size")
	if err != nil {
		log.Fatalf(err.Error())
	}
	DiskCreateCmd.Flags().StringVarP(&DiskDescription, "description", "d", DiskDescription, "description of disk")
	DiskCreateCmd.Flags().StringVarP(&DiskType, "type", "t", DiskType, "type of disk")

	DiskRemoveCmd.Flags().StringVarP(&DiskName, "name", "n", DiskName, "name of disk")
	err = DiskRemoveCmd.MarkFlagRequired("name")
	if err != nil {
		log.Fatalf(err.Error())
	}

	DiskCmd.AddCommand(DiskListCmd)
	DiskCmd.AddCommand(DiskCreateCmd)
	DiskCmd.AddCommand(DiskRemoveCmd)
}
