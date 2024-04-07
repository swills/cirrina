package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"cirrina/cirrinactl/rpc"
)

var VmIsoListCmd = &cobra.Command{
	Use:          "list",
	Short:        "Get list of ISOs connected to VM",
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

		var isoIds []string
		isoIds, err = rpc.GetVmIsos(VmId)
		if err != nil {
			return err
		}

		var names []string
		type isoListInfo struct {
			id   string
			info rpc.IsoInfo
			size string
		}
		isoInfos := make(map[string]isoListInfo)
		for _, id := range isoIds {
			var isoInfo rpc.IsoInfo
			isoInfo, err = rpc.GetIsoInfo(id)
			if err != nil {
				return err
			}

			var isoSize string
			if Humanize {
				isoSize = humanize.IBytes(isoInfo.Size)
			} else {
				isoSize = strconv.FormatUint(isoInfo.Size, 10)
			}

			isoInfos[isoInfo.Name] = isoListInfo{
				id:   id,
				size: isoSize,
			}
			names = append(names, isoInfo.Name)
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		if ShowUUID {
			t.AppendHeader(table.Row{"NAME", "UUID", "SIZE", "DESCRIPTION"})
		} else {
			t.AppendHeader(table.Row{"NAME", "SIZE", "DESCRIPTION"})
		}
		t.SetStyle(myTableStyle)
		for _, name := range names {
			if ShowUUID {
				t.AppendRow(table.Row{
					name,
					isoInfos[name].id,
					isoInfos[name].size,
					isoInfos[name].info.Descr,
				})
			} else {
				t.AppendRow(table.Row{
					name,
					isoInfos[name].size,
					isoInfos[name].info.Descr,
				})
			}
		}
		t.Render()

		return nil
	},
}

var VmIsosAddCmd = &cobra.Command{
	Use:          "add",
	Short:        "Add ISO to VM",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName)
			if err != nil {
				return nil
			}
			if VmId == "" {
				return errors.New("VM not found")
			}
		}
		if IsoId == "" {
			IsoId, err = rpc.IsoNameToId(IsoName)
			if err != nil {
				return err
			}
			if IsoId == "" {
				return errors.New("ISO not found")
			}
		}

		var isoIds []string
		isoIds, err = rpc.GetVmIsos(VmId)
		if err != nil {
			return err
		}

		isoIds = append(isoIds, IsoId)
		var res bool
		res, err = rpc.VmSetIsos(VmId, isoIds)
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

var VmIsosRmCmd = &cobra.Command{
	Use:          "remove",
	Short:        "Un-attach a ISO from a VM",
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
		if IsoId == "" {
			IsoId, err = rpc.IsoNameToId(IsoName)
			if err != nil {
				return err
			}
			if IsoId == "" {
				return errors.New("ISO not found")
			}
		}

		var isoIds []string
		isoIds, err = rpc.GetVmIsos(VmId)
		if err != nil {
			return err
		}

		var newIsoIds []string
		for _, id := range isoIds {
			if id != IsoId {
				newIsoIds = append(newIsoIds, id)
			}
		}

		var res bool
		res, err = rpc.VmSetIsos(VmId, newIsoIds)
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

var VmIsosCmd = &cobra.Command{
	Use:   "iso",
	Short: "ISO related operations on VMs",
	Long:  "List ISOs attached to VMs, attach ISOs to VMs and un-attach ISOs from VMs",
}

func init() {
	disableFlagSorting(VmIsosCmd)

	disableFlagSorting(VmIsoListCmd)
	addNameOrIdArgs(VmIsoListCmd, &VmName, &VmId, "VM")
	VmIsoListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	VmIsoListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)

	disableFlagSorting(VmIsosAddCmd)
	addNameOrIdArgs(VmIsosAddCmd, &VmName, &VmId, "VM")
	VmIsosAddCmd.Flags().StringVarP(&IsoName, "iso-name", "N", IsoName, "Name of Iso")
	VmIsosAddCmd.Flags().StringVarP(&IsoId, "iso-id", "I", IsoId, "Id of Iso")
	VmIsosAddCmd.MarkFlagsOneRequired("iso-name", "iso-id")
	VmIsosAddCmd.MarkFlagsMutuallyExclusive("iso-name", "iso-id")

	disableFlagSorting(VmIsosRmCmd)
	addNameOrIdArgs(VmIsosRmCmd, &VmName, &VmId, "VM")
	VmIsosRmCmd.Flags().StringVarP(&IsoName, "iso-name", "N", IsoName, "Name of Iso")
	VmIsosRmCmd.Flags().StringVarP(&IsoId, "iso-id", "I", IsoId, "Id of Iso")
	VmIsosRmCmd.MarkFlagsOneRequired("iso-name", "iso-id")
	VmIsosRmCmd.MarkFlagsMutuallyExclusive("iso-name", "iso-id")

	VmIsosCmd.AddCommand(VmIsoListCmd)
	VmIsosCmd.AddCommand(VmIsosAddCmd)
	VmIsosCmd.AddCommand(VmIsosRmCmd)

	VmCmd.AddCommand(VmIsosCmd)
}
