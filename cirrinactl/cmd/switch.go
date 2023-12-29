package cmd

import (
	"cirrina/cirrinactl/rpc"
	"errors"
	"fmt"
	"github.com/jedib0t/go-pretty/table"
	"github.com/spf13/cobra"
	"os"
	"sort"
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
		t.AppendHeader(table.Row{"NAME", "UUID", "TYPE", "UPLINK", "DESCRIPTION"})
		t.SetStyle(myTableStyle)
		for _, name := range names {
			t.AppendRow(table.Row{
				name,
				switchInfos[name].switchId,
				switchInfos[name].switchInfo.SwitchType,
				switchInfos[name].switchInfo.Uplink,
				switchInfos[name].switchInfo.Descr,
			})
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
	disableFlagSorting(SwitchCreateCmd)

	SwitchDestroyCmd.Flags().StringVarP(&SwitchName, "name", "n", SwitchName, "name of switch")
	SwitchDestroyCmd.Flags().StringVarP(&SwitchId, "id", "i", SwitchId, "id of Switch")
	SwitchDestroyCmd.MarkFlagsOneRequired("name", "id")
	SwitchDestroyCmd.MarkFlagsMutuallyExclusive("name", "id")
	disableFlagSorting(SwitchDestroyCmd)

	SwitchUplinkCmd.Flags().StringVarP(&SwitchName, "name", "n", SwitchName, "name of switch")
	SwitchUplinkCmd.Flags().StringVarP(&SwitchId, "id", "i", SwitchId, "id of Switch")
	SwitchUplinkCmd.MarkFlagsOneRequired("name", "id")
	SwitchUplinkCmd.MarkFlagsMutuallyExclusive("name", "id")
	SwitchUplinkCmd.Flags().StringVarP(&SwitchUplinkName,
		"uplink", "u", SwitchName, "uplink name",
	)
	err = SwitchUplinkCmd.MarkFlagRequired("uplink")
	if err != nil {
		panic(err)
	}
	disableFlagSorting(SwitchUplinkCmd)

	SwitchUpdateCmd.Flags().StringVarP(&SwitchName, "name", "n", SwitchName, "name of Switch")
	SwitchUpdateCmd.Flags().StringVarP(&SwitchId, "id", "i", SwitchId, "id of Switch")
	SwitchUpdateCmd.MarkFlagsOneRequired("name", "id")
	SwitchUpdateCmd.MarkFlagsMutuallyExclusive("name", "id")
	disableFlagSorting(SwitchUpdateCmd)

	SwitchUpdateCmd.Flags().StringVarP(&SwitchDescription,
		"description", "d", SwitchDescription, "description of switch",
	)

	SwitchCmd.AddCommand(SwitchListCmd)
	SwitchCmd.AddCommand(SwitchCreateCmd)
	SwitchCmd.AddCommand(SwitchDestroyCmd)
	SwitchCmd.AddCommand(SwitchUplinkCmd)
	SwitchCmd.AddCommand(SwitchUpdateCmd)
}
