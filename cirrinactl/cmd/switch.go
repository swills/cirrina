package cmd

import (
	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinactl/util"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"log"
)

var SwitchName string
var SwitchDescription string
var SwitchDescriptionChanged bool
var SwitchUplinkName string
var SwitchType = "IF"
var SwitchId string
var SwitchIdChanged bool

var SwitchCmd = &cobra.Command{
	Use:   "switch",
	Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Short: "Create, list, modify, destroy virtual switches",
}

var SwitchListCmd = &cobra.Command{
	Use:   "list",
	Short: "list virtual switches",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		util.GetSwitches(c, ctx)
	},
}

var SwitchCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "create virtual switch",
	Long: "Create a virtual switch.\n\nSwitches may be one of two types: \n\n" +
		"if_bridge (also called IF)\nnetgraph (also called NG)\n\nSwitches " +
		"of type if_bridge must be named starting with \"bridge\" followed " +
		"by a number, for example \"bridge0\".\nSwitches of type netgraph " +
		"must be named starting with \"bnet\" followed by a number, for example \"bnet0\".",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		util.AddSwitch(SwitchName, c, ctx, SwitchDescription, SwitchType)
	},
}

var SwitchDestroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "destroy virtual switch",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		if SwitchId == "" {
			SwitchId, err = rpc.SwitchNameToId(&SwitchName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if SwitchId == "" {
				log.Fatalf("Switch not found")
			}
		}
		if SwitchName == "" {
			SwitchName, err = rpc.SwitchIdToName(SwitchId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		util.RmSwitch(SwitchName, c, ctx)
	},
}

var SwitchUplinkCmd = &cobra.Command{
	Use:   "set-uplink",
	Short: "set switch uplink",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		if SwitchId == "" {
			SwitchId, err = rpc.SwitchNameToId(&SwitchName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if SwitchId == "" {
				log.Fatalf("Switch not found")
			}
		}
		if SwitchName == "" {
			SwitchName, err = rpc.SwitchIdToName(SwitchId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		util.SetUplink(SwitchName, c, ctx, SwitchUplinkName)
	},
}

var SwitchUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "update switch",
	Args: func(cmd *cobra.Command, args []string) error {
		SwitchDescriptionChanged = cmd.Flags().Changed("description")
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
		if SwitchId == "" {
			SwitchId, err = rpc.SwitchNameToId(&SwitchName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if SwitchId == "" {
				log.Fatalf("Switch not found")
			}
		}
		if SwitchName == "" {
			SwitchName, err = rpc.SwitchIdToName(SwitchId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		// currently only support changing switch description
		util.UpdateSwitch(SwitchName, c, ctx, SwitchDescriptionChanged, SwitchDescription)
	},
}

func init() {
	SwitchCreateCmd.Flags().StringVarP(&SwitchName, "name", "n", SwitchName, "name of switch")
	err := SwitchCreateCmd.MarkFlagRequired("name")
	if err != nil {
		log.Fatalf(err.Error())
	}
	SwitchCreateCmd.Flags().StringVarP(&SwitchDescription, "description", "d", SwitchDescription, "description of switch")
	SwitchCreateCmd.Flags().StringVarP(&SwitchType, "type", "t", SwitchType, "type of switch")

	SwitchDestroyCmd.Flags().StringVarP(&SwitchName, "name", "n", SwitchName, "name of switch")
	err = SwitchDestroyCmd.MarkFlagRequired("name")
	if err != nil {
		log.Fatalf(err.Error())
	}

	SwitchUplinkCmd.Flags().StringVarP(&SwitchName, "name", "n", SwitchName, "name of switch")
	err = SwitchUplinkCmd.MarkFlagRequired("name")
	if err != nil {
		log.Fatalf(err.Error())
	}
	SwitchUplinkCmd.Flags().StringVarP(&SwitchUplinkName, "uplink", "u", SwitchName, "uplink name")
	err = SwitchUplinkCmd.MarkFlagRequired("uplink")
	if err != nil {
		log.Fatalf(err.Error())
	}

	SwitchUpdateCmd.Flags().StringVarP(&SwitchName, "name", "n", SwitchName, "name of Switch")
	SwitchUpdateCmd.Flags().StringVarP(&SwitchId, "id", "i", SwitchId, "id of Switch")
	SwitchUpdateCmd.MarkFlagsOneRequired("name", "id")
	SwitchUpdateCmd.Flags().StringVarP(&SwitchDescription, "description", "d", SwitchDescription, "description of switch")

	SwitchCmd.AddCommand(SwitchListCmd)
	SwitchCmd.AddCommand(SwitchCreateCmd)
	SwitchCmd.AddCommand(SwitchDestroyCmd)
	SwitchCmd.AddCommand(SwitchUplinkCmd)
	SwitchCmd.AddCommand(SwitchUpdateCmd)
}
