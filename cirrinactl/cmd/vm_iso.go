package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"cirrina/cirrinactl/rpc"
)

var VMIsoListCmd = &cobra.Command{
	Use:          "list",
	Short:        "Get list of ISOs connected to VM",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
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

		var isoIDs []string
		isoIDs, err = rpc.GetVMIsos(VMID)
		if err != nil {
			return fmt.Errorf("failed getting VM ISOs: %w", err)
		}

		var names []string
		type isoListInfo struct {
			id   string
			info rpc.IsoInfo
			size string
		}
		isoInfos := make(map[string]isoListInfo)
		for _, id := range isoIDs {
			var isoInfo rpc.IsoInfo
			isoInfo, err = rpc.GetIsoInfo(id)
			if err != nil {
				return fmt.Errorf("failed setting iso info: %w", err)
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

var VMIsosAddCmd = &cobra.Command{
	Use:          "add",
	Short:        "Add ISO to VM",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error

		if VMID == "" {
			VMID, err = rpc.VMNameToID(VMName)
			if err != nil {
				return nil
			}
			if VMID == "" {
				return errVMNotFound
			}
		}
		if IsoID == "" {
			IsoID, err = rpc.IsoNameToID(IsoName)
			if err != nil {
				return fmt.Errorf("failed setting ISO ID: %w", err)
			}
			if IsoID == "" {
				return errIsoNotFound
			}
		}

		var isoIDs []string
		isoIDs, err = rpc.GetVMIsos(VMID)
		if err != nil {
			return fmt.Errorf("failed setting VM ISOs: %w", err)
		}

		isoIDs = append(isoIDs, IsoID)
		var res bool
		res, err = rpc.VMSetIsos(VMID, isoIDs)
		if err != nil {
			return fmt.Errorf("failed setting VM ISOs: %w", err)
		}
		if !res {
			return errReqFailed
		}
		fmt.Printf("Added\n")

		return nil
	},
}

var VMIsosRmCmd = &cobra.Command{
	Use:          "remove",
	Short:        "Un-attach a ISO from a VM",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error

		if VMID == "" {
			VMID, err = rpc.VMNameToID(VMName)
			if err != nil {
				return fmt.Errorf("failed setting VM ID: %w", err)
			}
			if VMID == "" {
				return errVMNotFound
			}
		}
		if IsoID == "" {
			IsoID, err = rpc.IsoNameToID(IsoName)
			if err != nil {
				return fmt.Errorf("failed getting ISO ID: %w", err)
			}
			if IsoID == "" {
				return errIsoNotFound
			}
		}

		var isoIDs []string
		isoIDs, err = rpc.GetVMIsos(VMID)
		if err != nil {
			return fmt.Errorf("failed getting VM ISOs: %w", err)
		}

		var newIsoIDs []string
		for _, id := range isoIDs {
			if id != IsoID {
				newIsoIDs = append(newIsoIDs, id)
			}
		}

		var res bool
		res, err = rpc.VMSetIsos(VMID, newIsoIDs)
		if err != nil {
			return fmt.Errorf("failed setting VM ISOs: %w", err)
		}
		if !res {
			return errReqFailed
		}
		fmt.Printf("Removed\n")

		return nil
	},
}

var VMIsosCmd = &cobra.Command{
	Use:   "iso",
	Short: "ISO related operations on VMs",
	Long:  "List ISOs attached to VMs, attach ISOs to VMs and un-attach ISOs from VMs",
}

func init() {
	disableFlagSorting(VMIsosCmd)

	disableFlagSorting(VMIsoListCmd)
	addNameOrIDArgs(VMIsoListCmd, &VMName, &VMID, "VM")
	VMIsoListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	VMIsoListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)

	disableFlagSorting(VMIsosAddCmd)
	addNameOrIDArgs(VMIsosAddCmd, &VMName, &VMID, "VM")
	VMIsosAddCmd.Flags().StringVarP(&IsoName, "iso-name", "N", IsoName, "Name of Iso")
	VMIsosAddCmd.Flags().StringVarP(&IsoID, "iso-id", "I", IsoID, "ID of Iso")
	VMIsosAddCmd.MarkFlagsOneRequired("iso-name", "iso-id")
	VMIsosAddCmd.MarkFlagsMutuallyExclusive("iso-name", "iso-id")

	disableFlagSorting(VMIsosRmCmd)
	addNameOrIDArgs(VMIsosRmCmd, &VMName, &VMID, "VM")
	VMIsosRmCmd.Flags().StringVarP(&IsoName, "iso-name", "N", IsoName, "Name of Iso")
	VMIsosRmCmd.Flags().StringVarP(&IsoID, "iso-id", "I", IsoID, "ID of Iso")
	VMIsosRmCmd.MarkFlagsOneRequired("iso-name", "iso-id")
	VMIsosRmCmd.MarkFlagsMutuallyExclusive("iso-name", "iso-id")

	VMIsosCmd.AddCommand(VMIsoListCmd)
	VMIsosCmd.AddCommand(VMIsosAddCmd)
	VMIsosCmd.AddCommand(VMIsosRmCmd)

	VMCmd.AddCommand(VMIsosCmd)
}
