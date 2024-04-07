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

var VmDisksListCmd = &cobra.Command{
	Use:          "list",
	Short:        "Get list of disks connected to VM",
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
		type diskListInfo struct {
			info  rpc.DiskInfo
			id    string
			size  string
			usage string
		}

		diskInfos := make(map[string]diskListInfo)
		var diskIds []string
		diskIds, err = rpc.GetVmDisks(VmId)
		if err != nil {
			return err
		}
		for _, id := range diskIds {
			diskInfo, err := rpc.GetDiskInfo(id)
			if err != nil {
				return err
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

var VmDiskAddCmd = &cobra.Command{
	Use:          "add",
	Short:        "Add disk to VM",
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
			DiskId, err = rpc.DiskNameToId(DiskName)
			if err != nil {
				return err
			}
			if DiskId == "" {
				return errors.New("disk not found")
			}
		}

		var diskIds []string
		diskIds, err = rpc.GetVmDisks(VmId)
		if err != nil {
			return err
		}
		diskIds = append(diskIds, DiskId)

		var res bool
		res, err = rpc.VmSetDisks(VmId, diskIds)
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

var VmDiskRmCmd = &cobra.Command{
	Use:          "remove",
	Short:        "Detach a disk from a VM",
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
			DiskId, err = rpc.DiskNameToId(DiskName)
			if err != nil {
				return err
			}
			if DiskId == "" {
				return errors.New("disk not found")
			}
		}
		var diskIds []string
		diskIds, err = rpc.GetVmDisks(VmId)
		if err != nil {
			return err
		}

		var newDiskIds []string
		for _, id := range diskIds {
			if id != DiskId {
				newDiskIds = append(newDiskIds, id)
			}
		}

		var res bool
		res, err = rpc.VmSetDisks(VmId, newDiskIds)
		if err != nil {
			return err
		}
		if !res {
			return errors.New("failed")
		}
		fmt.Printf("Disk removed from VM\n")

		return nil
	},
}

var VmDisksCmd = &cobra.Command{
	Use:   "disk",
	Short: "Disk related operations on VMs",
	Long:  "List disks attached to VMs, attach disks to VMs and un-attach disks from VMs",
}

func init() {
	disableFlagSorting(VmDisksCmd)

	disableFlagSorting(VmDisksListCmd)
	addNameOrIdArgs(VmDisksListCmd, &VmName, &VmId, "VM")
	VmDisksListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	VmDisksListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)

	disableFlagSorting(VmDiskAddCmd)
	addNameOrIdArgs(VmDiskAddCmd, &VmName, &VmId, "VM")
	VmDiskAddCmd.Flags().StringVarP(&DiskName, "disk-name", "N", DiskName, "Name of Disk")
	VmDiskAddCmd.Flags().StringVarP(&DiskId, "disk-id", "I", DiskId, "Id of Disk")
	VmDiskAddCmd.MarkFlagsOneRequired("disk-name", "disk-id")
	VmDiskAddCmd.MarkFlagsMutuallyExclusive("disk-name", "disk-id")

	disableFlagSorting(VmDiskRmCmd)
	addNameOrIdArgs(VmDiskRmCmd, &VmName, &VmId, "VM")
	VmDiskRmCmd.Flags().StringVarP(&DiskName, "disk-name", "N", DiskName, "Name of Disk")
	VmDiskRmCmd.Flags().StringVarP(&DiskId, "disk-id", "I", DiskId, "Id of Disk")
	VmDiskRmCmd.MarkFlagsOneRequired("disk-name", "disk-id")
	VmDiskRmCmd.MarkFlagsMutuallyExclusive("disk-name", "disk-id")

	VmDisksCmd.AddCommand(VmDisksListCmd)
	VmDisksCmd.AddCommand(VmDiskAddCmd)
	VmDisksCmd.AddCommand(VmDiskRmCmd)

	VmCmd.AddCommand(VmDisksCmd)
}
