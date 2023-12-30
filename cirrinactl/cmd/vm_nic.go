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
)

var VmNicsListCmd = &cobra.Command{
	Use:          "list",
	Short:        "Get list of NICs connected to VM",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName)
			if err != nil {
				return err
			}
			if VmId == "" {
				return errors.New("VM not found")
			}
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

		var nicIds []string
		nicIds, err = rpc.GetVmNics(VmId)
		if err != nil {
			return err
		}
		for _, id := range nicIds {
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
				table.Row{"NAME", "UUID", "MAC", "TYPE", "DEV-TYPE", "SWITCH",
					"RATE-LIMITED", "RATE-IN", "RATE-OUT", "DESCRIPTION"},
			)
		} else {
			t.AppendHeader(
				table.Row{"NAME", "MAC", "TYPE", "DEV-TYPE", "SWITCH",
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

var VmNicsAddCmd = &cobra.Command{
	Use:          "add",
	Short:        "Add NIC to VM",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName)
			if err != nil {
				return err
			}
			if VmId == "" {
				return errors.New("VM not found")
			}
		}
		if NicId == "" {
			NicId, err = rpc.NicNameToId(NicName)
			if err != nil {
				return err
			}
			if NicId == "" {
				return errors.New("NIC not found")
			}
		}
		var nicIds []string
		nicIds, err = rpc.GetVmNics(VmId)
		if err != nil {
			return err
		}

		nicIds = append(nicIds, NicId)
		var res bool
		res, err = rpc.VmSetNics(VmId, nicIds)
		if err != nil {
			return err
		}
		if !res {
			return errors.New("failed")
		}
		fmt.Printf("Added\n")
		return nil
	},
}

var VmNicsRmCmd = &cobra.Command{
	Use:          "remove",
	Short:        "Un-attach a NIC from a VM",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName)
			if err != nil {
				return err
			}
			if VmId == "" {
				return errors.New("VM not found")
			}
		}
		if NicId == "" {
			NicId, err = rpc.NicNameToId(NicName)
			if err != nil {
				return err
			}
			if NicId == "" {
				return errors.New("NIC not found")
			}
		}

		var nicIds []string
		nicIds, err = rpc.GetVmNics(VmId)
		if err != nil {
			return err
		}

		var newNicIds []string
		for _, id := range nicIds {
			if id != NicId {
				newNicIds = append(newNicIds, id)
			}
		}

		var res bool
		res, err = rpc.VmSetNics(VmId, newNicIds)
		if err != nil {
			return err
		}
		if !res {
			return errors.New("failed")
		}
		fmt.Printf("Removed\n")
		return nil
	},
}

var VmNicsCmd = &cobra.Command{
	Use:   "nic",
	Short: "NIC related operations on VMs",
	Long:  "List NICs attached to VMs, attach NICs to VMs and un-attach NICs from VMs",
}

func init() {
	disableFlagSorting(VmNicsCmd)

	disableFlagSorting(VmNicsListCmd)
	addNameOrIdArgs(VmNicsListCmd, &VmName, &VmId, "VM")
	VmNicsListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	VmNicsListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)

	disableFlagSorting(VmNicsAddCmd)
	addNameOrIdArgs(VmNicsAddCmd, &VmName, &VmId, "VM")
	VmNicsAddCmd.Flags().StringVarP(&NicName, "nic-name", "N", NicName, "Name of Nic")
	VmNicsAddCmd.Flags().StringVarP(&NicId, "nic-id", "I", NicId, "Id of Nic")
	VmNicsAddCmd.MarkFlagsOneRequired("nic-name", "nic-id")
	VmNicsAddCmd.MarkFlagsMutuallyExclusive("nic-name", "nic-id")

	disableFlagSorting(VmNicsRmCmd)
	addNameOrIdArgs(VmNicsRmCmd, &VmName, &VmId, "VM")
	VmNicsRmCmd.Flags().StringVarP(&NicName, "nic-name", "N", NicName, "Name of Nic")
	VmNicsRmCmd.Flags().StringVarP(&NicId, "nic-id", "I", NicId, "Id of Nic")
	VmNicsRmCmd.MarkFlagsOneRequired("nic-name", "nic-id")
	VmNicsRmCmd.MarkFlagsMutuallyExclusive("nic-name", "nic-id")

	VmNicsCmd.AddCommand(VmNicsListCmd)
	VmNicsCmd.AddCommand(VmNicsAddCmd)
	VmNicsCmd.AddCommand(VmNicsRmCmd)

	VmCmd.AddCommand(VmNicsCmd)
}
