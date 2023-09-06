package cmd

import (
	conn2 "cirrina/cirrinactl/rpc"
	"cirrina/cirrinactl/util"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"log"

	"github.com/spf13/cobra"
)

var NicCmd = &cobra.Command{
	Use:   "nic",
	Short: "Create, list, modify, destroy virtual NICs",
}

var NicName string
var NicDescription string
var NicType = "virtio-net"
var NicDevType = "tap"
var NicMac = "AUTO"
var NicSwitchId = ""
var NicId string
var NicIdChanged bool
var NicRateLimited bool
var NicRateIn uint64
var NicRateOut uint64
var NicHumanize bool

var NicListCmd = &cobra.Command{
	Use:   "list",
	Short: "list NICs",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := conn2.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		util.GetVmNicsAll(c, ctx, NicHumanize)
	},
}

var NicCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "create virtual NIC",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := conn2.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		_, err = util.AddVmNic(&NicName, c, ctx, &NicDescription, &NicType, &NicDevType, &NicMac,
			&NicSwitchId, &NicRateLimited, &NicRateIn, &NicRateOut)
		if err != nil {
			s := status.Convert(err)
			fmt.Printf("error: could not create a new NIC: %s\n", s.Message())
			return
		}
	},
}

var NicRemoveCmd = &cobra.Command{
	Use:   "destroy",
	Short: "remove virtual NIC",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := conn2.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		util.RmVmNic(&NicName, c, ctx)
	},
}

var NicSetSwitchCmd = &cobra.Command{
	Use:   "setswitch",
	Short: "Connect NIC to switch",
	Long:  "Connect a NIC to a switch, or set switch to empty to remove",
	Args: func(cmd *cobra.Command, args []string) error {
		SwitchIdChanged = cmd.Flags().Changed("switch-id")
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := conn2.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()

		if NicId == "" {
			NicId, err = conn2.NicNameToId(&NicName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if NicId == "" {
				log.Fatalf("Nic not found")
			}
		}

		if SwitchId == "" && !SwitchIdChanged && SwitchName != "" {
			SwitchId, err = conn2.SwitchNameToId(&SwitchName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if SwitchId == "" {
				log.Fatalf("Switch not found")
			}
		}
		util.NicSetSwitch(NicId, SwitchId, c, ctx)
	},
}

func init() {
	NicCreateCmd.Flags().StringVarP(&NicName, "name", "n", NicName, "name of NIC")
	err := NicCreateCmd.MarkFlagRequired("name")
	if err != nil {
		log.Fatalf(err.Error())
	}
	NicCreateCmd.Flags().StringVarP(&NicDescription, "description", "d", NicDescription, "description of NIC")
	NicCreateCmd.Flags().StringVarP(&NicType, "type", "t", NicType, "type of NIC")
	NicCreateCmd.Flags().StringVarP(&NicDevType, "devtype", "v", NicDevType, "NIC dev type")
	NicCreateCmd.Flags().StringVarP(&NicMac, "mac", "m", NicMac, "MAC address of NIC")
	NicCreateCmd.Flags().StringVarP(&NicSwitchId, "switch", "s", NicSwitchId, "uplink switch ID of NIC")
	NicCreateCmd.Flags().BoolVar(&NicRateLimited, "rate-limit", NicRateLimited, "Rate limit the NIC")
	NicCreateCmd.Flags().Uint64Var(&NicRateIn, "rate-in", NicRateIn, "Inbound rate limit of NIC")
	NicCreateCmd.Flags().Uint64Var(&NicRateOut, "rate-out", NicRateOut, "Outbound rate limit of NIC")

	NicRemoveCmd.Flags().StringVarP(&NicName, "name", "n", NicName, "name of NIC")
	err = NicRemoveCmd.MarkFlagRequired("name")
	if err != nil {
		log.Fatalf(err.Error())
	}

	NicListCmd.Flags().BoolVarP(&NicHumanize, "human", "H", NicHumanize, "Print speeds in human readable form")

	NicCmd.AddCommand(NicListCmd)
	NicCmd.AddCommand(NicCreateCmd)
	NicCmd.AddCommand(NicRemoveCmd)

	NicSetSwitchCmd.Flags().StringVarP(&NicName, "name", "n", NicName, "Name of Nic")
	NicSetSwitchCmd.Flags().StringVarP(&NicId, "id", "i", NicId, "Id of Nic")
	NicSetSwitchCmd.MarkFlagsOneRequired("name", "id")

	NicSetSwitchCmd.Flags().StringVarP(&SwitchName, "switch-name", "N", SwitchName, "Name of Switch")
	NicSetSwitchCmd.Flags().StringVarP(&SwitchId, "switch-id", "I", SwitchId, "Id of Switch")
	NicSetSwitchCmd.MarkFlagsOneRequired("switch-name", "switch-id")

	NicCmd.AddCommand(NicSetSwitchCmd)
}
