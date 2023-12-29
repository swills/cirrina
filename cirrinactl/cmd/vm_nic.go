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

var VmNicsGetCmd = &cobra.Command{
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
		t.AppendHeader(
			table.Row{"NAME", "UUID", "MAC", "TYPE", "DEV-TYPE", "SWITCH",
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

	VmNicsGetCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmNicsGetCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmNicsGetCmd.MarkFlagsOneRequired("name", "id")
	VmNicsGetCmd.MarkFlagsMutuallyExclusive("name", "id")
	VmNicsGetCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	disableFlagSorting(VmNicsGetCmd)

	VmNicsAddCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmNicsAddCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmNicsAddCmd.MarkFlagsOneRequired("name", "id")
	VmNicsAddCmd.MarkFlagsMutuallyExclusive("name", "id")
	disableFlagSorting(VmNicsAddCmd)

	VmNicsAddCmd.Flags().StringVarP(&NicName, "nic-name", "N", NicName, "Name of Nic")
	VmNicsAddCmd.Flags().StringVarP(&NicId, "nic-id", "I", NicId, "Id of Nic")
	VmNicsAddCmd.MarkFlagsOneRequired("nic-name", "nic-id")
	VmNicsAddCmd.MarkFlagsMutuallyExclusive("nic-name", "nic-id")

	VmNicsRmCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmNicsRmCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmNicsRmCmd.MarkFlagsOneRequired("name", "id")
	VmNicsRmCmd.MarkFlagsOneRequired("name", "id")
	disableFlagSorting(VmNicsRmCmd)

	VmNicsRmCmd.Flags().StringVarP(&NicName, "nic-name", "N", NicName, "Name of Nic")
	VmNicsRmCmd.Flags().StringVarP(&NicId, "nic-id", "I", NicId, "Id of Nic")
	VmNicsRmCmd.MarkFlagsOneRequired("nic-name", "nic-id")
	VmNicsRmCmd.MarkFlagsMutuallyExclusive("nic-name", "nic-id")

	VmNicsCmd.AddCommand(VmNicsGetCmd)
	VmNicsCmd.AddCommand(VmNicsAddCmd)
	VmNicsCmd.AddCommand(VmNicsRmCmd)

	VmCmd.AddCommand(VmNicsCmd)
}
