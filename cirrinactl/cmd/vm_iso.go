package cmd

import (
	"cirrina/cirrinactl/rpc"
	"errors"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/table"
	"github.com/spf13/cobra"
	"os"
	"strconv"
)

var VmIsosGetCmd = &cobra.Command{
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
			id     string
			info   rpc.IsoInfo
			size   string
			vmName string
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
		t.AppendHeader(table.Row{"NAME", "UUID", "SIZE", "DESCRIPTION"})
		t.SetStyle(myTableStyle)
		for _, name := range names {
			t.AppendRow(table.Row{
				name,
				isoInfos[name].id,
				isoInfos[name].size,
				isoInfos[name].info.Descr,
			})
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
		if DiskId == "" {
			DiskId, err = rpc.IsoNameToId(IsoName)
			if err != nil {
				return err
			}
			if DiskId == "" {
				return errors.New("ISO not found")
			}
		}

		var isoIds []string
		isoIds, err = rpc.GetVmIsos(VmId)
		if err != nil {
			return err
		}

		isoIds = append(isoIds, DiskId)
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
		if DiskId == "" {
			DiskId, err = rpc.IsoNameToId(IsoName)
			if err != nil {
				return err
			}
			if DiskId == "" {
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
			if id != DiskId {
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
	VmIsosCmd.Flags().SortFlags = false
	VmIsosCmd.PersistentFlags().SortFlags = false
	VmIsosCmd.InheritedFlags().SortFlags = false

	VmIsosGetCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmIsosGetCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmIsosGetCmd.MarkFlagsOneRequired("name", "id")
	VmIsosGetCmd.MarkFlagsMutuallyExclusive("name", "id")
	VmIsosGetCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	VmIsosGetCmd.Flags().SortFlags = false
	VmIsosGetCmd.PersistentFlags().SortFlags = false
	VmIsosGetCmd.InheritedFlags().SortFlags = false

	VmIsosAddCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmIsosAddCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmIsosAddCmd.MarkFlagsOneRequired("name", "id")
	VmIsosAddCmd.MarkFlagsMutuallyExclusive("name", "id")
	VmIsosAddCmd.Flags().SortFlags = false
	VmIsosAddCmd.PersistentFlags().SortFlags = false
	VmIsosAddCmd.InheritedFlags().SortFlags = false

	VmIsosAddCmd.Flags().StringVarP(&IsoName, "iso-name", "N", IsoName, "Name of Iso")
	VmIsosAddCmd.Flags().StringVarP(&DiskId, "iso-id", "I", DiskId, "Id of Iso")
	VmIsosAddCmd.MarkFlagsOneRequired("iso-name", "iso-id")
	VmIsosAddCmd.MarkFlagsMutuallyExclusive("iso-name", "iso-id")

	VmIsosRmCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmIsosRmCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmIsosRmCmd.MarkFlagsOneRequired("name", "id")
	VmIsosRmCmd.MarkFlagsMutuallyExclusive("name", "id")
	VmIsosRmCmd.Flags().SortFlags = false
	VmIsosRmCmd.PersistentFlags().SortFlags = false
	VmIsosRmCmd.InheritedFlags().SortFlags = false

	VmIsosRmCmd.Flags().StringVarP(&IsoName, "iso-name", "N", IsoName, "Name of Iso")
	VmIsosRmCmd.Flags().StringVarP(&DiskId, "iso-id", "I", DiskId, "Id of Iso")
	VmIsosRmCmd.MarkFlagsOneRequired("iso-name", "iso-id")
	VmIsosRmCmd.MarkFlagsMutuallyExclusive("iso-name", "iso-id")

	VmIsosCmd.AddCommand(VmIsosGetCmd)
	VmIsosCmd.AddCommand(VmIsosAddCmd)
	VmIsosCmd.AddCommand(VmIsosRmCmd)

	VmCmd.AddCommand(VmIsosCmd)
}
