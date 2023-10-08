package cmd

import (
	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinactl/util"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"log"
)

var DiskName string
var DiskDescription string
var DiskDescriptionChanged bool
var DiskType = "nvme"
var DiskTypeChanged bool
var DiskDevType = "FILE"
var DiskSize = "1G"
var DiskId string
var DiskIdChanged bool
var DiskUseHumanize bool
var DiskCache = true
var DiskDirect = false

var DiskListCmd = &cobra.Command{
	Use:   "list",
	Short: "list disks",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		err = util.GetDisks(c, ctx, DiskUseHumanize)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var DiskCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "create virtual disk",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		_, err = util.AddDisk(&DiskName, c, ctx, &DiskDescription, &DiskSize, &DiskType, &DiskDevType, DiskCache, DiskDirect)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var DiskRemoveCmd = &cobra.Command{
	Use:   "destroy",
	Short: "remove virtual disk",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
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

var DiskUpdateCmd = &cobra.Command{
	Use:   "modify",
	Short: "modify virtual disk",
	Args: func(cmd *cobra.Command, args []string) error {
		DiskDescriptionChanged = cmd.Flags().Changed("description")
		DiskTypeChanged = cmd.Flags().Changed("type")
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		if DiskId == "" {
			DiskId, err = rpc.DiskNameToId(&DiskName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if DiskId == "" {
				log.Fatalf("Disk not found")
			}
		}
		if DiskName == "" {
			DiskName, err = rpc.DiskIdToName(DiskId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		// currently only support changing disk description, and type
		err = util.UpdateDisk(DiskName, c, ctx, DiskDescriptionChanged, DiskDescription, DiskTypeChanged, DiskType)
		if err != nil {
			log.Fatal(err)
		}
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
	DiskCreateCmd.Flags().StringVar(&DiskDevType, "dev-type", DiskDevType, "Dev type of disk - file or zvol")
	DiskCreateCmd.Flags().BoolVar(&DiskCache, "cache", DiskCache, "Enable or disable OS caching for this disk")
	DiskCreateCmd.Flags().BoolVar(&DiskDirect, "direct", DiskDirect, "Enable or disable synchronous writes for this disk")

	DiskRemoveCmd.Flags().StringVarP(&DiskName, "name", "n", DiskName, "name of disk")
	err = DiskRemoveCmd.MarkFlagRequired("name")
	if err != nil {
		log.Fatalf(err.Error())
	}

	DiskListCmd.Flags().BoolVarP(&DiskUseHumanize, "human", "H", DiskUseHumanize, "Print sizes in human readable form")
	DiskUpdateCmd.Flags().StringVarP(&DiskName, "name", "n", DiskName, "name of disk")
	DiskUpdateCmd.Flags().StringVarP(&DiskId, "id", "i", DiskId, "id of disk")
	DiskUpdateCmd.MarkFlagsOneRequired("name", "id")
	DiskUpdateCmd.Flags().StringVarP(&DiskDescription, "description", "d", DiskDescription, "description of disk")
	DiskUpdateCmd.Flags().StringVarP(&DiskType, "type", "t", DiskType, "type of disk")

	DiskCmd.AddCommand(DiskListCmd)
	DiskCmd.AddCommand(DiskCreateCmd)
	DiskCmd.AddCommand(DiskRemoveCmd)
	DiskCmd.AddCommand(DiskUpdateCmd)
}
