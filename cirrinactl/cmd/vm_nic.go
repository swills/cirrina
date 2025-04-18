package cmd

import (
	"context"
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

var VMNicsListCmd = &cobra.Command{
	Use:          "list",
	Short:        "Get list of NICs connected to VM",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		if VMID == "" {
			VMID, err = rpc.VMNameToID(ctx, VMName)
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

		var nicIDs []string
		nicIDs, err = rpc.GetVMNics(ctx, VMID)
		if err != nil {
			return fmt.Errorf("failed getting VM NICs: %w", err)
		}
		for _, nicID := range nicIDs {
			nicInfo, err := rpc.GetVMNicInfo(ctx, nicID)
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
				nicID:       nicID,
				info:        nicInfo,
				rateLimited: rateLimited,
				rateIn:      rateIn,
				rateOut:     rateOut,
			}
			names = append(names, nicInfo.Name)
		}

		sort.Strings(names)
		nicTableWriter := table.NewWriter()
		nicTableWriter.SetOutputMirror(os.Stdout)
		if ShowUUID {
			nicTableWriter.AppendHeader(
				table.Row{
					"NAME", "UUID", "MAC", "TYPE", "DEV-TYPE", "SWITCH",
					"RATE-LIMITED", "RATE-IN", "RATE-OUT", "DESCRIPTION",
				},
			)
		} else {
			nicTableWriter.AppendHeader(
				table.Row{
					"NAME", "MAC", "TYPE", "DEV-TYPE", "SWITCH",
					"RATE-LIMITED", "RATE-IN", "RATE-OUT", "DESCRIPTION",
				},
			)
		}
		nicTableWriter.SetStyle(myTableStyle)
		for _, name := range names {
			if ShowUUID {
				nicTableWriter.AppendRow(table.Row{
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
				nicTableWriter.AppendRow(table.Row{
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
		nicTableWriter.Render()

		return nil
	},
}

var VMNicsAddCmd = &cobra.Command{
	Use:          "add",
	Short:        "Add NIC to VM",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		if VMID == "" {
			VMID, err = rpc.VMNameToID(ctx, VMName)
			if err != nil {
				return fmt.Errorf("failed getting VM ID: %w", err)
			}
			if VMID == "" {
				return errVMNotFound
			}
		}
		if NicID == "" {
			NicID, err = rpc.NicNameToID(ctx, NicName)
			if err != nil {
				return fmt.Errorf("failed getting NIC ID: %w", err)
			}
			if NicID == "" {
				return errNicNotFound
			}
		}
		var nicIDs []string
		nicIDs, err = rpc.GetVMNics(ctx, VMID)
		if err != nil {
			return fmt.Errorf("failed getting VM NICs: %w", err)
		}

		nicIDs = append(nicIDs, NicID)
		var res bool
		res, err = rpc.VMSetNics(ctx, VMID, nicIDs)
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

var VMNicsDisconnectCmd = &cobra.Command{
	Use:          "disconnect",
	Short:        "Disconnect a NIC from a VM",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		if VMID == "" {
			VMID, err = rpc.VMNameToID(ctx, VMName)
			if err != nil {
				return fmt.Errorf("failed getting VM ID: %w", err)
			}
			if VMID == "" {
				return errVMNotFound
			}
		}
		if NicID == "" {
			NicID, err = rpc.NicNameToID(ctx, NicName)
			if err != nil {
				return fmt.Errorf("failed getting NIC ID: %w", err)
			}
			if NicID == "" {
				return errNicNotFound
			}
		}

		var nicIDs []string
		nicIDs, err = rpc.GetVMNics(ctx, VMID)
		if err != nil {
			return fmt.Errorf("failed getting VM NICs: %w", err)
		}

		var newNicIDs []string
		for _, id := range nicIDs {
			if id != NicID {
				newNicIDs = append(newNicIDs, id)
			}
		}

		var res bool
		res, err = rpc.VMSetNics(ctx, VMID, newNicIDs)
		if err != nil {
			return fmt.Errorf("failed setting VM NICs: %w", err)
		}
		if !res {
			return errReqFailed
		}
		fmt.Printf("Deleted\n")

		return nil
	},
}

var VMNicsCmd = &cobra.Command{
	Use:   "nic",
	Short: "NIC related operations on VMs",
	Long:  "List NICs attached to VMs, attach NICs to VMs and un-attach NICs from VMs",
}
