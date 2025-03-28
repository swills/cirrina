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

var (
	NicName               string
	NicDescription        string
	NicType               = "virtio-net"
	NicTypeChanged        bool
	NicDevType            = "tap"
	NicDevTypeChanged     bool
	NicMac                = "AUTO"
	NicMacChanged         bool
	NicSwitchID           string
	NicSwitchIDChanged    bool
	NicSwitchName         string
	NicSwitchNameChanged  bool
	NicID                 string
	NicRateLimited        bool
	NicRateLimitedChanged bool
	NicRateIn             uint64
	NicRateInChanged      bool
	NicRateOut            uint64
	NicRateOutChanged     bool
	NicCloneName          string
	NicDescriptionChanged bool
)

var NicListCmd = &cobra.Command{
	Use:          "list",
	Short:        "list virtual NICs",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		nicIDs, err := rpc.GetVMNicsAll(ctx)
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

		for _, nicID := range nicIDs {
			nicInfo, err := rpc.GetVMNicInfo(ctx, nicID)
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
				nicID:       nicID,
				info:        nicInfo,
				rateLimited: rateLimited,
				vmName:      nicInfo.VMName,
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
					"NAME", "UUID", "MAC", "TYPE", "DEV-TYPE", "UPLINK", "VM",
					"RATE-LIMITED", "RATE-IN", "RATE-OUT", "DESCRIPTION",
				},
			)
		} else {
			nicTableWriter.AppendHeader(
				table.Row{
					"NAME", "MAC", "TYPE", "DEV-TYPE", "UPLINK", "VM",
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
					nicInfos[name].vmName,
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
					nicInfos[name].vmName,
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

var NicCreateCmd = &cobra.Command{
	Use:          "create",
	Short:        "create virtual NIC",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		if NicName == "" {
			return errNicEmptyName
		}
		if NicSwitchID == "" {
			if NicSwitchName != "" {
				NicSwitchID, err = rpc.SwitchNameToID(ctx, NicSwitchName)
				if err != nil {
					return fmt.Errorf("error getting switch id: %w", err)
				}
			}
		}

		res, err := rpc.AddNic(ctx,
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

var NicDeleteCmd = &cobra.Command{
	Use:          "delete",
	Short:        "delete virtual nic",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		if NicID == "" {
			NicID, err = rpc.NicNameToID(ctx, NicName)
			if err != nil {
				return fmt.Errorf("error getting nic id: %w", err)
			}
			if NicID == "" {
				return errNicNotFound
			}
		}
		err = rpc.RmNic(ctx, NicID)
		if err != nil {
			return fmt.Errorf("error removing nic: %w", err)
		}
		fmt.Printf("NIC deleted\n")

		return nil
	},
}

var NicSetSwitchCmd = &cobra.Command{
	Use:          "connect",
	Short:        "Connect NIC to switch",
	Long:         "Connect a NIC to a switch, or set switch to empty to disconnect",
	SilenceUsage: true,
	Args: func(cmd *cobra.Command, _ []string) error {
		NicSwitchIDChanged = cmd.Flags().Changed("switch-id")
		NicSwitchNameChanged = cmd.Flags().Changed("switch-name")

		return nil
	},
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		if NicID == "" {
			NicID, err = rpc.NicNameToID(ctx, NicName)
			if err != nil {
				return fmt.Errorf("error getting nic id: %w", err)
			}
			if NicID == "" {
				return errNicNotFound
			}
		}

		if NicSwitchID == "" && !NicSwitchIDChanged && NicSwitchName != "" {
			NicSwitchID, err = rpc.SwitchNameToID(ctx, NicSwitchName)
			if err != nil {
				return fmt.Errorf("error getting switch id: %w", err)
			}
			if NicSwitchID == "" {
				return errSwitchNotFound
			}
		}

		err = rpc.SetVMNicSwitch(ctx, NicID, NicSwitchID)
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
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		if NicID == "" {
			NicID, err = rpc.NicNameToID(ctx, NicName)
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
		reqID, err := rpc.CloneNic(ctx, NicID, NicCloneName)
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
			reqStat, err = rpc.ReqStat(ctx, reqID)
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
	Args: func(cmd *cobra.Command, _ []string) error {
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
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		if NicID == "" {
			NicID, err = rpc.NicNameToID(ctx, NicName)
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
			var NewNicSwitchID string
			NewNicSwitchID, err = rpc.SwitchNameToID(ctx, NicSwitchName)
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
		err = rpc.UpdateNic(ctx, NicID, newDesc, newMac, newNicType, newNicDevType,
			newRateLimit, newRateIn, newRateOut, newSwitchID)
		if err != nil {
			return fmt.Errorf("error updating nic: %w", err)
		}
		fmt.Printf("Nic updated\n")

		return nil
	},
}

var NicCmd = &cobra.Command{
	Use:   "nic",
	Short: "Create, list, modify, delete virtual NICs",
}
