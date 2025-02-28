package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	epb "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"

	"cirrina/cirrinactl/rpc"
)

var (
	SwitchName               string
	SwitchDescription        string
	SwitchDescriptionChanged bool
	SwitchUplinkName         string
	SwitchType               = "IF"
	SwitchID                 string
)

var SwitchCmd = &cobra.Command{
	Use:   "switch",
	Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Short: "Create, list, modify, delete virtual switches",
}

var SwitchListCmd = &cobra.Command{
	Use:          "list",
	Short:        "list virtual switches",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		switchIDs, err := rpc.GetSwitches(ctx)
		if err != nil {
			return fmt.Errorf("error getting switches: %w", err)
		}

		var names []string
		type switchListInfo struct {
			switchID   string
			switchInfo rpc.SwitchInfo
		}

		switchInfos := make(map[string]switchListInfo)
		for _, switchID := range switchIDs {
			res, err := rpc.GetSwitch(ctx, switchID)
			if err != nil {
				return fmt.Errorf("error getting switch: %w", err)
			}
			names = append(names, res.Name)
			switchInfos[res.Name] = switchListInfo{
				switchID:   switchID,
				switchInfo: res,
			}
		}

		sort.Strings(names)
		switchTableWriter := table.NewWriter()
		switchTableWriter.SetOutputMirror(os.Stdout)
		if ShowUUID {
			switchTableWriter.AppendHeader(table.Row{"NAME", "UUID", "TYPE", "UPLINK", "DESCRIPTION"})
		} else {
			switchTableWriter.AppendHeader(table.Row{"NAME", "TYPE", "UPLINK", "DESCRIPTION"})
		}
		switchTableWriter.SetStyle(myTableStyle)
		for _, name := range names {
			if ShowUUID {
				switchTableWriter.AppendRow(table.Row{
					name,
					switchInfos[name].switchID,
					switchInfos[name].switchInfo.SwitchType,
					switchInfos[name].switchInfo.Uplink,
					switchInfos[name].switchInfo.Descr,
				})
			} else {
				switchTableWriter.AppendRow(table.Row{
					name,
					switchInfos[name].switchInfo.SwitchType,
					switchInfos[name].switchInfo.Uplink,
					switchInfos[name].switchInfo.Descr,
				})
			}
		}
		switchTableWriter.Render()

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
	RunE: func(_ *cobra.Command, _ []string) error {
		if SwitchName == "" {
			return errSwitchEmptyName
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		res, err := rpc.AddSwitch(ctx, SwitchName, &SwitchDescription, &SwitchType, &SwitchUplinkName)
		if err != nil {
			return fmt.Errorf("error adding switch: %w", err)
		}
		fmt.Printf("Switch created. id: %s\n", res)

		return nil
	},
}

var SwitchDeleteCmd = &cobra.Command{
	Use:          "delete",
	Short:        "delete virtual switch",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		if SwitchID == "" {
			SwitchID, err = rpc.SwitchNameToID(ctx, SwitchName)
			if err != nil {
				return fmt.Errorf("error getting switch ID: %w", err)
			}
			if SwitchID == "" {
				return errSwitchNotFound
			}
		}

		err = rpc.DeleteSwitch(ctx, SwitchID)
		if err != nil {
			s := status.Convert(err)
			for _, d := range s.Details() {
				switch info := d.(type) {
				case *epb.PreconditionFailure:
					var gotDesc bool
					for _, v := range info.GetViolations() {
						gotDesc = true
						fmt.Printf("%s\n", v.GetDescription())
					}

					if !gotDesc {
						fmt.Printf("error: %s", info)
					}

					return ErrServerError
				default:
					fmt.Printf("Unexpected type: %s", info)

					return ErrServerError
				}
			}
		}
		fmt.Printf("Switch deleted\n")

		return nil
	},
}

var SwitchUplinkCmd = &cobra.Command{
	Use:          "set-uplink",
	Short:        "set switch uplink",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		if SwitchID == "" {
			SwitchID, err = rpc.SwitchNameToID(ctx, SwitchName)
			if err != nil {
				return fmt.Errorf("error getting switch id: %w", err)
			}
			if SwitchID == "" {
				return errSwitchNotFound
			}
		}
		err = rpc.SetSwitchUplink(ctx, SwitchID, &SwitchUplinkName)
		if err != nil {
			return fmt.Errorf("error setting switch uplink: %w", err)
		}
		fmt.Printf("Switch uplink set\n")

		return nil
	},
}

var SwitchUpdateCmd = &cobra.Command{
	Use:          "update",
	Short:        "update switch",
	SilenceUsage: true,
	Args: func(cmd *cobra.Command, _ []string) error {
		SwitchDescriptionChanged = cmd.Flags().Changed("description")

		return nil
	},
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		if SwitchID == "" {
			SwitchID, err = rpc.SwitchNameToID(ctx, SwitchName)
			if err != nil {
				return fmt.Errorf("error getting switch id: %w", err)
			}
			if SwitchID == "" {
				return errSwitchNotFound
			}
		}

		// currently only support changing switch description
		var newDesc *string
		if SwitchDescriptionChanged {
			newDesc = &SwitchDescription
		}
		err = rpc.UpdateSwitch(ctx, SwitchID, newDesc)
		if err != nil {
			return fmt.Errorf("error updating switch: %w", err)
		}
		fmt.Printf("Switch updated\n")

		return nil
	},
}
