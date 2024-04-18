package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"cirrina/cirrinactl/rpc"
)

var NicName string
var NicDescription string
var NicType = "virtio-net"
var NicTypeChanged bool
var NicDevType = "tap"
var NicDevTypeChanged bool
var NicMac = "AUTO"
var NicMacChanged bool
var NicSwitchID string
var NicSwitchIDChanged bool
var NicSwitchName string
var NicSwitchNameChanged bool
var NicID string
var NicRateLimited bool
var NicRateLimitedChanged bool
var NicRateIn uint64
var NicRateInChanged bool
var NicRateOut uint64
var NicRateOutChanged bool
var NicCloneName string
var NicDescriptionChanged bool

var NicListCmd = &cobra.Command{
	Use:          "list",
	Short:        "list virtual NICs",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := rpc.GetVMNicsAll()
		if err != nil {
			return fmt.Errorf("error getting all vm nics: %w", err)
		}
		var names []string
		type nicListInfo struct {
			nicID       string
			info        rpc.NicInfo
			vmName      string
			rateLimited string
			rateIn      string
			rateOut     string
		}
		nicInfos := make(map[string]nicListInfo)

		for _, id := range res {
			nicInfo, err := rpc.GetVMNicInfo(id)
			if err != nil {
				return fmt.Errorf("error getting nic info: %w", err)
			}

			var rateLimited string
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
				nicID:       id,
				info:        nicInfo,
				rateLimited: rateLimited,
				vmName:      nicInfo.VMName,
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
					nicInfos[name].nicID,
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
		var err error
		if NicName == "" {
			return errNicEmptyName
		}
		if NicSwitchID == "" {
			if NicSwitchName != "" {
				NicSwitchID, err = rpc.SwitchNameToID(NicSwitchName)
				if err != nil {
					return fmt.Errorf("error getting switch id: %w", err)
				}
				if NicSwitchID == "" {
					return errSwitchNotFound
				}
			}
		}

		res, err := rpc.AddNic(
			NicName, NicDescription, NicMac, NicType, NicDevType,
			NicRateLimited, NicRateIn, NicRateOut, NicSwitchID,
		)
		if err != nil {
			return fmt.Errorf("error adding nic: %w", err)
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
		var err error
		if NicID == "" {
			NicID, err = rpc.NicNameToID(NicName)
			if err != nil {
				return fmt.Errorf("error getting nic id: %w", err)
			}
			if NicID == "" {
				return errNicNotFound
			}
		}
		err = rpc.RmNic(NicID)
		if err != nil {
			return fmt.Errorf("error removing nic: %w", err)
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
		NicSwitchIDChanged = cmd.Flags().Changed("switch-id")

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if NicID == "" {
			NicID, err = rpc.NicNameToID(NicName)
			if err != nil {
				return fmt.Errorf("error getting nic id: %w", err)
			}
			if NicID == "" {
				return errNicNotFound
			}
		}

		if NicSwitchID == "" && !NicSwitchIDChanged && SwitchName != "" {
			NicSwitchID, err = rpc.SwitchNameToID(NicSwitchName)
			if err != nil {
				return fmt.Errorf("error getting switch id: %w", err)
			}
			if NicSwitchID == "" {
				return errSwitchNotFound
			}
		}

		err = rpc.SetVMNicSwitch(NicID, NicSwitchID)
		if err != nil {
			return fmt.Errorf("error setting nic uplink: %w", err)
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
		if NicID == "" {
			NicID, err = rpc.NicNameToID(NicName)
			if err != nil {
				return fmt.Errorf("error getting nic ID: %w", err)
			}
			if NicID == "" {
				return errNicNotFound
			}
		}

		if NicCloneName == "" {
			return errNicEmptyName
		}

		if CheckReqStat {
			fmt.Print("Cloning NIC (timeout: 10s): ")
		}
		reqID, err := rpc.CloneNic(NicID, NicCloneName)
		if err != nil {
			return fmt.Errorf("error cloning nic: %w", err)
		}

		if !CheckReqStat {
			fmt.Printf("Request submitted\n")

			return nil
		}

		timeout := time.Now().Add(time.Second * 10)

		var reqStat rpc.ReqStatus
		for time.Now().Before(timeout) {
			reqStat, err = rpc.ReqStat(reqID)
			if err != nil {
				return fmt.Errorf("error checking request status: %w", err)
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
		NicMacChanged = cmd.Flags().Changed("mac")
		NicDevTypeChanged = cmd.Flags().Changed("devtype")
		NicTypeChanged = cmd.Flags().Changed("type")
		NicRateLimitedChanged = cmd.Flags().Changed("rate-limit")
		NicRateInChanged = cmd.Flags().Changed("rate-in")
		NicRateOutChanged = cmd.Flags().Changed("rate-out")
		NicSwitchIDChanged = cmd.Flags().Changed("switch-id")
		NicSwitchNameChanged = cmd.Flags().Changed("switch-name")

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if NicID == "" {
			NicID, err = rpc.NicNameToID(NicName)
			if err != nil {
				return fmt.Errorf("error getting nic id: %w", err)
			}
			if NicID == "" {
				return errNicNotFound
			}
		}

		var newDesc *string
		var newMac *string
		var newNicType *string
		var newNicDevType *string
		var newRateLimit *bool
		var newRateIn *uint64
		var newRateOut *uint64
		var newSwitchID *string

		if NicDescriptionChanged {
			newDesc = &NicDescription
		}
		if NicMacChanged {
			newMac = &NicMac
		}
		if NicTypeChanged {
			newNicType = &NicType
		}
		if NicDevTypeChanged {
			newNicDevType = &NicDevType
		}
		if NicRateLimitedChanged {
			newRateLimit = &NicRateLimited
		}
		if NicRateInChanged {
			newRateIn = &NicRateIn
		}
		if NicRateOutChanged {
			newRateOut = &NicRateOut
		}
		if NicSwitchNameChanged {
			NewNicSwitchID, err := rpc.SwitchNameToID(NicSwitchName)
			if err != nil {
				return fmt.Errorf("error getting switch id: %w", err)
			}
			if NewNicSwitchID == "" {
				return errSwitchNotFound
			}
			newSwitchID = &NewNicSwitchID
		}
		if NicSwitchIDChanged {
			newSwitchID = &NicSwitchID
		}
		err = rpc.UpdateNic(NicID, newDesc, newMac, newNicType, newNicDevType, newRateLimit, newRateIn, newRateOut, newSwitchID)
		if err != nil {
			return fmt.Errorf("error updating nic: %w", err)
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
	NicCreateCmd.Flags().StringVar(&NicSwitchID,
		"switch-id", NicSwitchID, "NIC uplink switch ID",
	)
	NicCreateCmd.Flags().StringVar(&NicSwitchName,
		"switch-name", NicSwitchName, "NIC uplink switch name",
	)
	NicCreateCmd.Flags().BoolVar(&NicRateLimited, "rate-limit", NicRateLimited, "Rate limit the NIC")
	NicCreateCmd.Flags().Uint64Var(&NicRateIn, "rate-in", NicRateIn, "Inbound rate limit of NIC")
	NicCreateCmd.Flags().Uint64Var(&NicRateOut, "rate-out", NicRateOut, "Outbound rate limit of NIC")

	disableFlagSorting(NicRemoveCmd)
	addNameOrIDArgs(NicRemoveCmd, &NicName, &NicID, "NIC")

	disableFlagSorting(NicSetSwitchCmd)
	addNameOrIDArgs(NicSetSwitchCmd, &NicName, &NicID, "NIC")
	NicSetSwitchCmd.Flags().StringVarP(&NicSwitchName,
		"switch-name", "N", SwitchName, "Switch Name",
	)
	NicSetSwitchCmd.Flags().StringVarP(&NicSwitchID, "switch-id", "I", SwitchID, "ID of Switch")
	NicSetSwitchCmd.MarkFlagsOneRequired("switch-name", "switch-id")
	NicSetSwitchCmd.MarkFlagsMutuallyExclusive("switch-name", "switch-id")

	disableFlagSorting(NicCloneCmd)
	addNameOrIDArgs(NicCloneCmd, &NicName, &NicID, "NIC")

	NicCloneCmd.Flags().StringVar(&NicCloneName,
		"new-name", NicCloneName, "Name of Cloned NIC",
	)
	NicCloneCmd.Flags().BoolVarP(&CheckReqStat, "status", "s", CheckReqStat, "Check status")

	disableFlagSorting(NicUpdateCmd)
	addNameOrIDArgs(NicUpdateCmd, &NicName, &NicID, "NIC")
	NicUpdateCmd.Flags().StringVarP(&NicDescription,
		"description", "d", NicDescription, "description of NIC",
	)
	NicUpdateCmd.Flags().StringVarP(&NicType, "type", "t", NicType, "type of NIC")
	NicUpdateCmd.Flags().StringVarP(&NicDevType, "devtype", "v", NicDevType, "NIC dev type")
	NicUpdateCmd.Flags().StringVarP(&NicMac, "mac", "m", NicMac, "MAC address of NIC")
	NicUpdateCmd.Flags().StringVarP(&NicSwitchID,
		"switch-id", "I", NicSwitchID, "NIC uplink switch ID",
	)
	NicUpdateCmd.Flags().StringVarP(&NicSwitchName,
		"switch-name", "N", NicSwitchName, "NIC uplink switch name",
	)
	NicUpdateCmd.Flags().BoolVar(&NicRateLimited, "rate-limit", NicRateLimited, "Rate limit the NIC")
	NicUpdateCmd.Flags().Uint64Var(&NicRateIn, "rate-in", NicRateIn, "Inbound rate limit of NIC")
	NicUpdateCmd.Flags().Uint64Var(&NicRateOut, "rate-out", NicRateOut, "Outbound rate limit of NIC")

	NicCmd.AddCommand(NicListCmd)
	NicCmd.AddCommand(NicCreateCmd)
	NicCmd.AddCommand(NicRemoveCmd)
	NicCmd.AddCommand(NicSetSwitchCmd)
	NicCmd.AddCommand(NicCloneCmd)
	NicCmd.AddCommand(NicUpdateCmd)
}
