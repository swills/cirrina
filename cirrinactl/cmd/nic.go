package cmd

import (
	"cirrina/cirrinactl/rpc"
	"errors"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/table"
	"github.com/spf13/cobra"
	"os"
	"sort"
	"strconv"
	"strings"
)

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

var NicListCmd = &cobra.Command{
	Use:          "list",
	Short:        "list virtual NICs",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := rpc.GetVmNicsAll()
		if err != nil {
			return err
		}
		var names []string
		type nicListInfo struct {
			nicId       string
			info        rpc.NicInfo
			vmName      string
			rateLimited string
			rateIn      string
			rateOut     string
		}
		nicInfos := make(map[string]nicListInfo)

		for _, id := range res {
			nicInfo, err := rpc.GetVmNicInfo(id)
			if err != nil {
				return err
			}

			var vmName string
			vmName, err = rpc.NicGetVm(id)
			if err != nil {
				return err
			}
			rateLimited := "unknown"
			var rateIn string
			var rateOut string

			if nicInfo.RateLimited {
				rateLimited = "yes"
				if Humanize {
					rateIn = humanize.Bytes(nicInfo.RateIn)
					rateIn = strings.Replace(rateIn, "B", "b", 1) + "ps"
					rateOut = humanize.Bytes(nicInfo.RateOut)
					rateOut = strings.Replace(rateOut, "B", "b", 1) + "ps"
				} else {
					rateIn = strconv.FormatUint(nicInfo.RateIn, 10)
					rateOut = strconv.FormatUint(nicInfo.RateOut, 10)
				}
			} else {
				rateLimited = "no"
			}
			nicInfos[nicInfo.Name] = nicListInfo{
				nicId:       id,
				info:        nicInfo,
				vmName:      vmName,
				rateLimited: rateLimited,
				rateIn:      rateIn,
				rateOut:     rateOut,
			}
			names = append(names, nicInfo.Name)
		}

		sort.Strings(names)
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(
			table.Row{"NAME", "UUID", "MAC", "TYPE", "DEV-TYPE", "UPLINK", "VM",
				"RATE-LIMITED", "RATE-IN", "RATE-OUT", "DESCRIPTION"},
		)
		t.SetStyle(myTableStyle)
		for _, name := range names {
			t.AppendRow(table.Row{
				name,
				nicInfos[name].nicId,
				nicInfos[name].info.Mac,
				nicInfos[name].info.NetType,
				nicInfos[name].info.NetDevType,
				nicInfos[name].info.Uplink,
				nicInfos[name].vmName,
				nicInfos[name].rateLimited,
				nicInfos[name].rateIn,
				nicInfos[name].rateOut,
				nicInfos[name].info.Descr,
			})
		}
		t.Render()
		return nil
	},
}

var NicCreateCmd = &cobra.Command{
	Use:          "create",
	Short:        "create virtual NIC",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := rpc.AddNic(
			NicName, NicDescription, NicMac, NicType, NicDevType,
			NicRateLimited, NicRateIn, NicRateOut, NicSwitchId,
		)
		if err != nil {
			return err
		}
		fmt.Printf("NIC created. id: %s\n", res)
		return nil
	},
}

var NicRemoveCmd = &cobra.Command{
	Use:          "destroy",
	Short:        "remove virtual nic",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		nicId, err := rpc.NicNameToId(NicName)
		if err != nil {
			return err
		}
		err = rpc.RmNic(nicId)
		if err != nil {
			return err
		}
		fmt.Printf("NIC deleted\n")
		return nil
	},
}

var NicSetSwitchCmd = &cobra.Command{
	Use:          "setswitch",
	Short:        "Connect NIC to switch",
	Long:         "Connect a NIC to a switch, or set switch to empty to remove",
	SilenceUsage: true,
	Args: func(cmd *cobra.Command, args []string) error {
		SwitchIdChanged = cmd.Flags().Changed("switch-id")
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if NicId == "" {
			NicId, err = rpc.NicNameToId(NicName)
			if err != nil {
				return err
			}
			if NicId == "" {
				return errors.New("NIC not found")
			}
		}

		if SwitchId == "" && !SwitchIdChanged && SwitchName != "" {
			SwitchId, err = rpc.SwitchNameToId(SwitchName)
			if err != nil {
				return err
			}
			if SwitchId == "" {
				return errors.New("switch not found")
			}
		}

		err = rpc.SetVmNicSwitch(NicId, SwitchId)
		if err != nil {
			return err
		}
		fmt.Printf("Added NIC to switch\n")
		return nil
	},
}

var NicCmd = &cobra.Command{
	Use:   "nic",
	Short: "Create, list, modify, destroy virtual NICs",
}

func init() {
	NicCreateCmd.Flags().StringVarP(&NicName, "name", "n", NicName, "name of NIC")
	err := NicCreateCmd.MarkFlagRequired("name")
	if err != nil {
		panic(err)
	}
	NicCreateCmd.Flags().StringVarP(&NicDescription,
		"description", "d", NicDescription, "description of NIC",
	)
	NicCreateCmd.Flags().StringVarP(&NicType, "type", "t", NicType, "type of NIC")
	NicCreateCmd.Flags().StringVarP(&NicDevType, "devtype", "v", NicDevType, "NIC dev type")
	NicCreateCmd.Flags().StringVarP(&NicMac, "mac", "m", NicMac, "MAC address of NIC")
	NicCreateCmd.Flags().StringVarP(&NicSwitchId,
		"switch", "s", NicSwitchId, "uplink switch ID of NIC",
	)
	NicCreateCmd.Flags().BoolVar(&NicRateLimited, "rate-limit", NicRateLimited, "Rate limit the NIC")
	NicCreateCmd.Flags().Uint64Var(&NicRateIn, "rate-in", NicRateIn, "Inbound rate limit of NIC")
	NicCreateCmd.Flags().Uint64Var(&NicRateOut, "rate-out", NicRateOut, "Outbound rate limit of NIC")

	NicRemoveCmd.Flags().StringVarP(&NicName, "name", "n", NicName, "name of NIC")
	err = NicRemoveCmd.MarkFlagRequired("name")
	if err != nil {
		panic(err)
	}

	NicListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print speeds in human readable form",
	)

	NicCmd.AddCommand(NicListCmd)
	NicCmd.AddCommand(NicCreateCmd)
	NicCmd.AddCommand(NicRemoveCmd)

	NicSetSwitchCmd.Flags().StringVarP(&NicName, "name", "n", NicName, "Name of Nic")
	NicSetSwitchCmd.Flags().StringVarP(&NicId, "id", "i", NicId, "Id of Nic")
	NicSetSwitchCmd.MarkFlagsOneRequired("name", "id")
	NicSetSwitchCmd.MarkFlagsMutuallyExclusive("name", "id")

	NicSetSwitchCmd.Flags().StringVarP(&SwitchName,
		"switch-name", "N", SwitchName, "Name of Switch",
	)
	NicSetSwitchCmd.Flags().StringVarP(&SwitchId, "switch-id", "I", SwitchId, "Id of Switch")
	NicSetSwitchCmd.MarkFlagsOneRequired("switch-name", "switch-id")
	NicSetSwitchCmd.MarkFlagsMutuallyExclusive("switch-name", "switch-id")

	NicCmd.AddCommand(NicSetSwitchCmd)
}
