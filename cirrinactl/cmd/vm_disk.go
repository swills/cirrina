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

var VMDisksListCmd = &cobra.Command{
	Use:          "list",
	Short:        "Get list of disks connected to VM",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if VMID == "" {
			VMID, err = rpc.VMNameToID(VMName)
			if err != nil {
				return fmt.Errorf("failed getting VM ID: %w", err)
			}
			if VMID == "" {
				return errors.New("VM not found")
			}
		}

		var names []string
		type diskListInfo struct {
			info  rpc.DiskInfo
			id    string
			size  string
			usage string
		}

		diskInfos := make(map[string]diskListInfo)
		var diskIds []string
		diskIds, err = rpc.GetVMDisks(VMID)
		if err != nil {
			return fmt.Errorf("failed getting disks: %w", err)
		}
		for _, id := range diskIds {
			diskInfo, err := rpc.GetDiskInfo(id)
			if err != nil {
				return fmt.Errorf("failed getting disk info: %w", err)
			}
			var diskSize string
			var diskUsage string
			if Humanize {
				diskSize = humanize.IBytes(diskInfo.Size)
				diskUsage = humanize.IBytes(diskInfo.Usage)
			} else {
				diskSize = strconv.FormatUint(diskInfo.Size, 10)
				diskUsage = strconv.FormatUint(diskInfo.Usage, 10)
			}
			diskInfos[diskInfo.Name] = diskListInfo{
				id:    id,
				info:  diskInfo,
				size:  diskSize,
				usage: diskUsage,
			}
			names = append(names, diskInfo.Name)

		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		if ShowUUID {
			t.AppendHeader(
				table.Row{"NAME", "UUID", "TYPE", "SIZE", "USAGE", "DEV-TYPE", "CACHE", "DIRECT", "DESCRIPTION"},
			)
		} else {
			t.AppendHeader(
				table.Row{"NAME", "TYPE", "SIZE", "USAGE", "DEV-TYPE", "CACHE", "DIRECT", "DESCRIPTION"},
			)
		}

		t.SetStyle(myTableStyle)
		for _, diskName := range names {
			if ShowUUID {
				t.AppendRow(table.Row{
					diskName,
					diskInfos[diskName].id,
					diskInfos[diskName].info.DiskType,
					diskInfos[diskName].size,
					diskInfos[diskName].usage,
					diskInfos[diskName].info.DiskDevType,
					diskInfos[diskName].info.Cache,
					diskInfos[diskName].info.Direct,
					diskInfos[diskName].info.Descr,
				})
			} else {
				t.AppendRow(table.Row{
					diskName,
					diskInfos[diskName].info.DiskType,
					diskInfos[diskName].size,
					diskInfos[diskName].usage,
					diskInfos[diskName].info.DiskDevType,
					diskInfos[diskName].info.Cache,
					diskInfos[diskName].info.Direct,
					diskInfos[diskName].info.Descr,
				})
			}
		}
		t.Render()

		return nil
	},
}

var VMDiskAddCmd = &cobra.Command{
	Use:          "add",
	Short:        "Add disk to VM",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		if VMID == "" {
			VMID, err = rpc.VMNameToID(VMName)
			if err != nil {
				return fmt.Errorf("failed getting VM ID: %w", err)
			}
			if VMID == "" {
				return errors.New("VM not found")
			}
		}
		if DiskID == "" {
			DiskID, err = rpc.DiskNameToID(DiskName)
			if err != nil {
				return fmt.Errorf("failed getting disk ID: %w", err)
			}
			if DiskID == "" {
				return errors.New("disk not found")
			}
		}

		var diskIds []string
		diskIds, err = rpc.GetVMDisks(VMID)
		if err != nil {
			if err != nil {
				return fmt.Errorf("failed getting disks: %w", err)
			}
		}
		diskIds = append(diskIds, DiskID)

		var res bool
		res, err = rpc.VMSetDisks(VMID, diskIds)
		if err != nil {
			return fmt.Errorf("failed setting disks: %w", err)
		}
		if !res {
			return errors.New("failed")
		}
		fmt.Printf("Added\n")

		return nil
	},
}

var VMDiskRmCmd = &cobra.Command{
	Use:          "remove",
	Short:        "Detach a disk from a VM",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		if VMID == "" {
			VMID, err = rpc.VMNameToID(VMName)
			if err != nil {
				return fmt.Errorf("failed getting VM ID: %w", err)
			}
			if VMID == "" {
				return errors.New("VM not found")
			}
		}
		if DiskID == "" {
			DiskID, err = rpc.DiskNameToID(DiskName)
			if err != nil {
				return fmt.Errorf("failed getting disk ID: %w", err)
			}
			if DiskID == "" {
				return errors.New("disk not found")
			}
		}
		var diskIds []string
		diskIds, err = rpc.GetVMDisks(VMID)
		if err != nil {
			return fmt.Errorf("failed getting VM disks: %w", err)
		}

		var newDiskIds []string
		for _, id := range diskIds {
			if id != DiskID {
				newDiskIds = append(newDiskIds, id)
			}
		}

		var res bool
		res, err = rpc.VMSetDisks(VMID, newDiskIds)
		if err != nil {
			return fmt.Errorf("failed setting VM disks: %w", err)
		}
		if !res {
			return errors.New("failed")
		}
		fmt.Printf("Disk removed from VM\n")

		return nil
	},
}

var VMDisksCmd = &cobra.Command{
	Use:   "disk",
	Short: "Disk related operations on VMs",
	Long:  "List disks attached to VMs, attach disks to VMs and un-attach disks from VMs",
}

func init() {
	disableFlagSorting(VMDisksCmd)

	disableFlagSorting(VMDisksListCmd)
	addNameOrIDArgs(VMDisksListCmd, &VMName, &VMID, "VM")
	VMDisksListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	VMDisksListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)

	disableFlagSorting(VMDiskAddCmd)
	addNameOrIDArgs(VMDiskAddCmd, &VMName, &VMID, "VM")
	VMDiskAddCmd.Flags().StringVarP(&DiskName, "disk-name", "N", DiskName, "Name of Disk")
	VMDiskAddCmd.Flags().StringVarP(&DiskID, "disk-id", "I", DiskID, "ID of Disk")
	VMDiskAddCmd.MarkFlagsOneRequired("disk-name", "disk-id")
	VMDiskAddCmd.MarkFlagsMutuallyExclusive("disk-name", "disk-id")

	disableFlagSorting(VMDiskRmCmd)
	addNameOrIDArgs(VMDiskRmCmd, &VMName, &VMID, "VM")
	VMDiskRmCmd.Flags().StringVarP(&DiskName, "disk-name", "N", DiskName, "Name of Disk")
	VMDiskRmCmd.Flags().StringVarP(&DiskID, "disk-id", "I", DiskID, "ID of Disk")
	VMDiskRmCmd.MarkFlagsOneRequired("disk-name", "disk-id")
	VMDiskRmCmd.MarkFlagsMutuallyExclusive("disk-name", "disk-id")

	VMDisksCmd.AddCommand(VMDisksListCmd)
	VMDisksCmd.AddCommand(VMDiskAddCmd)
	VMDisksCmd.AddCommand(VMDiskRmCmd)

	VMCmd.AddCommand(VMDisksCmd)
}
