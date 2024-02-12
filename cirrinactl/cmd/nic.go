package cmd

import (
	"cirrina/cirrinactl/rpc"
	"errors"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var NicName string
var NicDescription string
var NicType = "virtio-net"
var NicDevType = "tap"
var NicMac = "AUTO"
var NicSwitchId = ""
var NicId string
var NicRateLimited bool
var NicRateIn uint64
var NicRateOut uint64
var NicCloneName string
var NicCloneMac string
var NicDescriptionChanged bool

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
				rateLimited: rateLimited,
				rateIn:      rateIn,
				rateOut:     rateOut,
			}
			names = append(names, nicInfo.Name)
		}

		sort.Strings(names)
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		if ShowUUID {
			t.AppendHeader(
				table.Row{"NAME", "UUID", "MAC", "TYPE", "DEV-TYPE", "UPLINK", "VM",
					"RATE-LIMITED", "RATE-IN", "RATE-OUT", "DESCRIPTION"},
			)
		} else {
			t.AppendHeader(
				table.Row{"NAME", "MAC", "TYPE", "DEV-TYPE", "UPLINK", "VM",
					"RATE-LIMITED", "RATE-IN", "RATE-OUT", "DESCRIPTION"},
			)
		}
		t.SetStyle(myTableStyle)
		for _, name := range names {
			if ShowUUID {
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
			} else {
				t.AppendRow(table.Row{
					name,
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
		if NicName == "" {
			return errors.New("empty NIC name")
		}
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
		if NicId == "" {
			NicId, err := rpc.NicNameToId(NicName)
			if err != nil {
				return err
			}
			if NicId == "" {
				return errors.New("NIC not found")
			}
		}
		err := rpc.RmNic(NicId)
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

var NicCloneCmd = &cobra.Command{
	Use:          "clone",
	Short:        "Clone a NIC",
	SilenceUsage: true,
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

		if NicCloneName == "" {
			return errors.New("empty NIC name")
		}

		if CheckReqStat {
			fmt.Print("Cloning NIC (timeout: 10s): ")
		}
		reqId, err := rpc.CloneNic(
			NicId, NicCloneName, NicCloneMac,
		)
		if err != nil {
			return err
		}

		if !CheckReqStat {
			fmt.Printf("Request submitted\n")
			return nil
		}

		timeout := time.Now().Add(time.Second * 10)

		var reqStat rpc.ReqStatus
		for time.Now().Before(timeout) {
			reqStat, err = rpc.ReqStat(reqId)
			if err != nil {
				return err
			}
			if reqStat.Complete {
				break
			}
			fmt.Printf(".")
			time.Sleep(time.Second)
		}
		if reqStat.Success {
			fmt.Printf(" done")
		} else {
			fmt.Printf(" failed")
		}
		fmt.Printf("\n")
		return nil
	},
}

var NicUpdateCmd = &cobra.Command{
	Use:          "update",
	Short:        "update NIC",
	SilenceUsage: true,
	Args: func(cmd *cobra.Command, args []string) error {
		NicDescriptionChanged = cmd.Flags().Changed("description")
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
				return errors.New("nic not found")
			}
		}

		// currently only support changing nic description
		var newDesc *string
		if NicDescriptionChanged {
			newDesc = &NicDescription
		}
		err = rpc.UpdateNic(NicId, newDesc)
		if err != nil {
			return err
		}
		fmt.Printf("Nic updated\n")
		return nil
	},
}

var NicCmd = &cobra.Command{
	Use:   "nic",
	Short: "Create, list, modify, destroy virtual NICs",
}

func init() {
	disableFlagSorting(NicCmd)

	disableFlagSorting(NicListCmd)
	NicListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print speeds in human readable form",
	)
	NicListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)

	disableFlagSorting(NicCreateCmd)
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

	disableFlagSorting(NicRemoveCmd)
	addNameOrIdArgs(NicRemoveCmd, &NicName, &NicId, "NIC")

	disableFlagSorting(NicSetSwitchCmd)
	addNameOrIdArgs(NicSetSwitchCmd, &NicName, &NicId, "NIC")
	NicSetSwitchCmd.Flags().StringVarP(&SwitchName,
		"switch-name", "N", SwitchName, "Name of Switch",
	)
	NicSetSwitchCmd.Flags().StringVarP(&SwitchId, "switch-id", "I", SwitchId, "Id of Switch")
	NicSetSwitchCmd.MarkFlagsOneRequired("switch-name", "switch-id")
	NicSetSwitchCmd.MarkFlagsMutuallyExclusive("switch-name", "switch-id")

	disableFlagSorting(NicCloneCmd)
	addNameOrIdArgs(NicCloneCmd, &NicName, &NicId, "NIC")

	NicCloneCmd.Flags().StringVarP(&NicCloneName,
		"new-name", "N", NicCloneName, "Name of Cloned NIC",
	)
	NicCloneCmd.Flags().StringVarP(&NicCloneMac, "mac", "m", NicCloneMac, "New MAC address of cloned NIC")
	NicCloneCmd.Flags().BoolVarP(&CheckReqStat, "status", "s", CheckReqStat, "Check status")

	disableFlagSorting(NicUpdateCmd)
	addNameOrIdArgs(NicUpdateCmd, &NicName, &NicId, "NIC")
	NicUpdateCmd.Flags().StringVarP(&NicDescription,
		"description", "d", NicDescription, "description of NIC",
	)

	NicCmd.AddCommand(NicListCmd)
	NicCmd.AddCommand(NicCreateCmd)
	NicCmd.AddCommand(NicRemoveCmd)
	NicCmd.AddCommand(NicSetSwitchCmd)
	NicCmd.AddCommand(NicCloneCmd)
	NicCmd.AddCommand(NicUpdateCmd)
}
