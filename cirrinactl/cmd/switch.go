package cmd

import (
	"cirrina/cirrinactl/rpc"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var SwitchName string
var SwitchDescription string
var SwitchDescriptionChanged bool
var SwitchUplinkName string
var SwitchType = "IF"
var SwitchId string

var SwitchCmd = &cobra.Command{
	Use:   "switch",
	Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Short: "Create, list, modify, destroy virtual switches",
}

var SwitchListCmd = &cobra.Command{
	Use:          "list",
	Short:        "list virtual switches",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := rpc.GetSwitches()
		if err != nil {
			return err
		}

		var names []string
		type switchListInfo struct {
			switchId   string
			switchInfo rpc.SwitchInfo
		}

		switchInfos := make(map[string]switchListInfo)
		for _, id := range res {
			res, err := rpc.GetSwitch(id)
			if err != nil {
				return err
			}
			names = append(names, res.Name)
			switchInfos[res.Name] = switchListInfo{
				switchId:   id,
				switchInfo: res,
			}
		}

		sort.Strings(names)
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		if ShowUUID {
			t.AppendHeader(table.Row{"NAME", "UUID", "TYPE", "UPLINK", "DESCRIPTION"})
		} else {
			t.AppendHeader(table.Row{"NAME", "TYPE", "UPLINK", "DESCRIPTION"})
		}
		t.SetStyle(myTableStyle)
		for _, name := range names {
			if ShowUUID {
				t.AppendRow(table.Row{
					name,
					switchInfos[name].switchId,
					switchInfos[name].switchInfo.SwitchType,
					switchInfos[name].switchInfo.Uplink,
					switchInfos[name].switchInfo.Descr,
				})
			} else {
				t.AppendRow(table.Row{
					name,
					switchInfos[name].switchInfo.SwitchType,
					switchInfos[name].switchInfo.Uplink,
					switchInfos[name].switchInfo.Descr,
				})
			}
		}
		t.Render()
		return nil
	},
}

var SwitchCreateCmd = &cobra.Command{
	Use:          "create",
	Short:        "create virtual switch",
	SilenceUsage: true,
	Long: "Create a virtual switch.\n\nSwitches may be one of two types: \n\n" +
		"if_bridge (also called IF)\nnetgraph (also called NG)\n\nSwitches " +
		"of type if_bridge must be named starting with \"bridge\" followed " +
		"by a number, for example \"bridge0\".\nSwitches of type netgraph " +
		"must be named starting with \"bnet\" followed by a number, for example \"bnet0\".",
	RunE: func(cmd *cobra.Command, args []string) error {
		if SwitchName == "" {
			return errors.New("empty switch name")
		}
		res, err := rpc.AddSwitch(SwitchName, &SwitchDescription, &SwitchType, &SwitchUplinkName)
		if err != nil {
			return err
		}
		fmt.Printf("Switch created. id: %s\n", res)
		return nil
	},
}

var SwitchDestroyCmd = &cobra.Command{
	Use:          "destroy",
	Short:        "destroy virtual switch",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if SwitchId == "" {
			SwitchId, err = rpc.SwitchNameToId(SwitchName)
			if err != nil {
				return err
			}
			if SwitchId == "" {
				return errors.New("switch not found")
			}
		}

		err = rpc.RemoveSwitch(SwitchId)
		if err != nil {
			return err
		}
		fmt.Printf("Switch deleted\n")
		return nil
	},
}

var SwitchUplinkCmd = &cobra.Command{
	Use:          "set-uplink",
	Short:        "set switch uplink",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if SwitchId == "" {
			SwitchId, err = rpc.SwitchNameToId(SwitchName)
			if err != nil {
				return err
			}
			if SwitchId == "" {
				return errors.New("switch not found")
			}
		}
		err = rpc.SetSwitchUplink(SwitchId, &SwitchUplinkName)
		if err != nil {
			return err
		}
		fmt.Printf("Switch uplink set\n")
		return nil
	},
}

var SwitchUpdateCmd = &cobra.Command{
	Use:          "update",
	Short:        "update switch",
	SilenceUsage: true,
	Args: func(cmd *cobra.Command, args []string) error {
		SwitchDescriptionChanged = cmd.Flags().Changed("description")
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if SwitchId == "" {
			SwitchId, err = rpc.SwitchNameToId(SwitchName)
			if err != nil {
				return err
			}
			if SwitchId == "" {
				return errors.New("switch not found")
			}
		}

		// currently only support changing switch description
		var newDesc *string
		if SwitchDescriptionChanged {
			newDesc = &SwitchDescription
		}
		err = rpc.UpdateSwitch(SwitchId, newDesc)
		if err != nil {
			return err
		}
		fmt.Printf("Switch updated\n")
		return nil
	},
}

func init() {
	disableFlagSorting(SwitchCmd)

	disableFlagSorting(SwitchListCmd)
	SwitchListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)

	disableFlagSorting(SwitchCreateCmd)
	SwitchCreateCmd.Flags().StringVarP(&SwitchName, "name", "n", SwitchName, "name of switch")
	err := SwitchCreateCmd.MarkFlagRequired("name")
	if err != nil {
		panic(err)
	}
	SwitchCreateCmd.Flags().StringVarP(&SwitchDescription,
		"description", "d", SwitchDescription, "description of switch",
	)
	SwitchCreateCmd.Flags().StringVarP(&SwitchType, "type", "t", SwitchType, "type of switch")
	SwitchCreateCmd.Flags().StringVarP(&SwitchUplinkName,
		"uplink", "u", SwitchName, "uplink name",
	)

	disableFlagSorting(SwitchDestroyCmd)
	addNameOrIdArgs(SwitchDestroyCmd, &SwitchName, &SwitchId, "switch")

	disableFlagSorting(SwitchUplinkCmd)
	addNameOrIdArgs(SwitchUplinkCmd, &SwitchName, &SwitchId, "switch")
	SwitchUplinkCmd.Flags().StringVarP(&SwitchUplinkName,
		"uplink", "u", SwitchName, "uplink name",
	)
	err = SwitchUplinkCmd.MarkFlagRequired("uplink")
	if err != nil {
		panic(err)
	}

	disableFlagSorting(SwitchUpdateCmd)
	addNameOrIdArgs(SwitchUpdateCmd, &SwitchName, &SwitchId, "switch")
	SwitchUpdateCmd.Flags().StringVarP(&SwitchDescription,
		"description", "d", SwitchDescription, "description of switch",
	)

	SwitchCmd.AddCommand(SwitchListCmd)
	SwitchCmd.AddCommand(SwitchCreateCmd)
	SwitchCmd.AddCommand(SwitchDestroyCmd)
	SwitchCmd.AddCommand(SwitchUpdateCmd)
	SwitchCmd.AddCommand(SwitchUplinkCmd)
}
