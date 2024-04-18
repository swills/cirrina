package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"cirrina/cirrinactl/rpc"
)

var VMNicsListCmd = &cobra.Command{
	Use:          "list",
	Short:        "Get list of NICs connected to VM",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if VMID == "" {
			VMID, err = rpc.VMNameToID(VMName)
			if err != nil {
				return fmt.Errorf("failed getting VM ID: %w", err)
			}
			if VMID == "" {
				return errVMNotFound
			}
		}

		var names []string
		type nicListInfo struct {
			nicID       string
			info        rpc.NicInfo
			rateLimited string
			rateIn      string
			rateOut     string
		}
		nicInfos := make(map[string]nicListInfo)

		var nicIds []string
		nicIds, err = rpc.GetVMNics(VMID)
		if err != nil {
			return fmt.Errorf("failed getting VM NICs: %w", err)
		}
		for _, id := range nicIds {
			nicInfo, err := rpc.GetVMNicInfo(id)
			if err != nil {
				return fmt.Errorf("failed getting NIC info: %w", err)
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
					nicInfos[name].nicID,
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

var VMNicsAddCmd = &cobra.Command{
	Use:          "add",
	Short:        "Add NIC to VM",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		if VMID == "" {
			VMID, err = rpc.VMNameToID(VMName)
			if err != nil {
				return fmt.Errorf("failed getting VM ID: %w", err)
			}
			if VMID == "" {
				return errVMNotFound
			}
		}
		if NicID == "" {
			NicID, err = rpc.NicNameToID(NicName)
			if err != nil {
				return fmt.Errorf("failed getting NIC ID: %w", err)
			}
			if NicID == "" {
				return errNicNotFound
			}
		}
		var nicIds []string
		nicIds, err = rpc.GetVMNics(VMID)
		if err != nil {
			return fmt.Errorf("failed getting VM NICs: %w", err)
		}

		nicIds = append(nicIds, NicID)
		var res bool
		res, err = rpc.VMSetNics(VMID, nicIds)
		if err != nil {
			return fmt.Errorf("failed setting VM NICs: %w", err)
		}
		if !res {
			return errReqFailed
		}
		fmt.Printf("Added\n")

		return nil
	},
}

var VMNicsRmCmd = &cobra.Command{
	Use:          "remove",
	Short:        "Un-attach a NIC from a VM",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		if VMID == "" {
			VMID, err = rpc.VMNameToID(VMName)
			if err != nil {
				return fmt.Errorf("failed getting VM ID: %w", err)
			}
			if VMID == "" {
				return errVMNotFound
			}
		}
		if NicID == "" {
			NicID, err = rpc.NicNameToID(NicName)
			if err != nil {
				return fmt.Errorf("failed getting NIC ID: %w", err)
			}
			if NicID == "" {
				return errNicNotFound
			}
		}

		var nicIds []string
		nicIds, err = rpc.GetVMNics(VMID)
		if err != nil {
			return fmt.Errorf("failed getting VM NICs: %w", err)
		}

		var newNicIds []string
		for _, id := range nicIds {
			if id != NicID {
				newNicIds = append(newNicIds, id)
			}
		}

		var res bool
		res, err = rpc.VMSetNics(VMID, newNicIds)
		if err != nil {
			return fmt.Errorf("failed setting VM NICs: %w", err)
		}
		if !res {
			return errReqFailed
		}
		fmt.Printf("Removed\n")

		return nil
	},
}

var VMNicsCmd = &cobra.Command{
	Use:   "nic",
	Short: "NIC related operations on VMs",
	Long:  "List NICs attached to VMs, attach NICs to VMs and un-attach NICs from VMs",
}

func init() {
	disableFlagSorting(VMNicsCmd)

	disableFlagSorting(VMNicsListCmd)
	addNameOrIDArgs(VMNicsListCmd, &VMName, &VMID, "VM")
	VMNicsListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	VMNicsListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)

	disableFlagSorting(VMNicsAddCmd)
	addNameOrIDArgs(VMNicsAddCmd, &VMName, &VMID, "VM")
	VMNicsAddCmd.Flags().StringVarP(&NicName, "nic-name", "N", NicName, "Name of Nic")
	VMNicsAddCmd.Flags().StringVarP(&NicID, "nic-id", "I", NicID, "ID of Nic")
	VMNicsAddCmd.MarkFlagsOneRequired("nic-name", "nic-id")
	VMNicsAddCmd.MarkFlagsMutuallyExclusive("nic-name", "nic-id")

	disableFlagSorting(VMNicsRmCmd)
	addNameOrIDArgs(VMNicsRmCmd, &VMName, &VMID, "VM")
	VMNicsRmCmd.Flags().StringVarP(&NicName, "nic-name", "N", NicName, "Name of Nic")
	VMNicsRmCmd.Flags().StringVarP(&NicID, "nic-id", "I", NicID, "ID of Nic")
	VMNicsRmCmd.MarkFlagsOneRequired("nic-name", "nic-id")
	VMNicsRmCmd.MarkFlagsMutuallyExclusive("nic-name", "nic-id")

	VMNicsCmd.AddCommand(VMNicsListCmd)
	VMNicsCmd.AddCommand(VMNicsAddCmd)
	VMNicsCmd.AddCommand(VMNicsRmCmd)

	VMCmd.AddCommand(VMNicsCmd)
}
