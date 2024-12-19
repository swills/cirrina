package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	"cirrina/cirrinactl/rpc"
)

var VMDisksListCmd = &cobra.Command{
	Use:          "list",
	Short:        "Get list of disks connected to VM",
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

		var names []string
		type diskListInfo struct {
			info  rpc.DiskInfo
			id    string
			size  string
			usage string
		}

		diskInfos := make(map[string]diskListInfo)
		var diskIDs []string
		diskIDs, err = rpc.GetVMDisks(VMID)
		if err != nil {
			return fmt.Errorf("failed getting disks: %w", err)
		}
		for _, diskID := range diskIDs {
			diskInfo, err := rpc.GetDiskInfo(diskID)
			if err != nil {
				return fmt.Errorf("failed getting disk info: %w", err)
			}

			var aDiskListInfo diskListInfo

			aDiskListInfo.id = diskID
			aDiskListInfo.info = diskInfo

			if ShowDiskSizeUsage {
				var diskSize string

				var diskUsage string

				diskSizeUsage, err := rpc.GetDiskSizeUsage(diskID)
				if err != nil {
					return fmt.Errorf("failed getting disk info: %w", err)
				}

				if Humanize {
					diskSize = humanize.IBytes(diskSizeUsage.Size)
					diskUsage = humanize.IBytes(diskSizeUsage.Usage)
				} else {
					diskSize = strconv.FormatUint(diskSizeUsage.Size, 10)
					diskUsage = strconv.FormatUint(diskSizeUsage.Usage, 10)
				}

				aDiskListInfo.size = diskSize
				aDiskListInfo.usage = diskUsage
			}

			diskInfos[diskInfo.Name] = aDiskListInfo

			names = append(names, diskInfo.Name)
		}

		diskTableWriter := table.NewWriter()
		diskTableWriter.SetOutputMirror(os.Stdout)
		if ShowUUID {
			if ShowDiskSizeUsage {
				diskTableWriter.AppendHeader(
					table.Row{"NAME", "UUID", "TYPE", "SIZE", "USAGE", "DEV-TYPE", "CACHE", "DIRECT", "DESCRIPTION"},
				)

				diskTableWriter.SetColumnConfigs([]table.ColumnConfig{
					{Number: 4, Align: text.AlignRight, AlignHeader: text.AlignRight},
					{Number: 5, Align: text.AlignRight, AlignHeader: text.AlignRight},
				})
			} else {
				diskTableWriter.AppendHeader(
					table.Row{"NAME", "UUID", "TYPE", "DEV-TYPE", "CACHE", "DIRECT", "DESCRIPTION"},
				)
			}
		} else {
			if ShowDiskSizeUsage {
				diskTableWriter.AppendHeader(
					table.Row{"NAME", "TYPE", "SIZE", "USAGE", "DEV-TYPE", "CACHE", "DIRECT", "DESCRIPTION"},
				)

				diskTableWriter.SetColumnConfigs([]table.ColumnConfig{
					{Number: 3, Align: text.AlignRight, AlignHeader: text.AlignRight},
					{Number: 4, Align: text.AlignRight, AlignHeader: text.AlignRight},
				})
			} else {
				diskTableWriter.AppendHeader(
					table.Row{"NAME", "TYPE", "DEV-TYPE", "CACHE", "DIRECT", "DESCRIPTION"},
				)
			}
		}

		diskTableWriter.SetStyle(myTableStyle)
		for _, diskName := range names {
			if ShowUUID {
				if ShowDiskSizeUsage {
					diskTableWriter.AppendRow(table.Row{
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
					diskTableWriter.AppendRow(table.Row{
						diskName,
						diskInfos[diskName].id,
						diskInfos[diskName].info.DiskType,
						diskInfos[diskName].info.DiskDevType,
						diskInfos[diskName].info.Cache,
						diskInfos[diskName].info.Direct,
						diskInfos[diskName].info.Descr,
					})
				}
			} else {
				if ShowDiskSizeUsage {
					diskTableWriter.AppendRow(table.Row{
						diskName,
						diskInfos[diskName].info.DiskType,
						diskInfos[diskName].size,
						diskInfos[diskName].usage,
						diskInfos[diskName].info.DiskDevType,
						diskInfos[diskName].info.Cache,
						diskInfos[diskName].info.Direct,
						diskInfos[diskName].info.Descr,
					})
				} else {
					diskTableWriter.AppendRow(table.Row{
						diskName,
						diskInfos[diskName].info.DiskType,
						diskInfos[diskName].info.DiskDevType,
						diskInfos[diskName].info.Cache,
						diskInfos[diskName].info.Direct,
						diskInfos[diskName].info.Descr,
					})
				}
			}
		}
		diskTableWriter.Render()

		return nil
	},
}

var VMDiskAddCmd = &cobra.Command{
	Use:          "add",
	Short:        "Add disk to VM",
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
		if DiskID == "" {
			DiskID, err = rpc.DiskNameToID(DiskName)
			if err != nil {
				return fmt.Errorf("failed getting disk ID: %w", err)
			}
			if DiskID == "" {
				return errDiskNotFound
			}
		}

		var diskIDs []string
		diskIDs, err = rpc.GetVMDisks(VMID)
		if err != nil {
			if err != nil {
				return fmt.Errorf("failed getting disks: %w", err)
			}
		}
		diskIDs = append(diskIDs, DiskID)

		var res bool
		res, err = rpc.VMSetDisks(VMID, diskIDs)
		if err != nil {
			return fmt.Errorf("failed setting disks: %w", err)
		}
		if !res {
			return errReqFailed
		}
		fmt.Printf("Added\n")

		return nil
	},
}

var VMDiskDisconnectCmd = &cobra.Command{
	Use:          "disconnect",
	Short:        "Disconnect a disk from a VM",
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
		if DiskID == "" {
			DiskID, err = rpc.DiskNameToID(DiskName)
			if err != nil {
				return fmt.Errorf("failed getting disk ID: %w", err)
			}
			if DiskID == "" {
				return errDiskNotFound
			}
		}
		var diskIDs []string
		diskIDs, err = rpc.GetVMDisks(VMID)
		if err != nil {
			return fmt.Errorf("failed getting VM disks: %w", err)
		}

		var newDiskIDs []string
		for _, id := range diskIDs {
			if id != DiskID {
				newDiskIDs = append(newDiskIDs, id)
			}
		}

		var res bool
		res, err = rpc.VMSetDisks(VMID, newDiskIDs)
		if err != nil {
			return fmt.Errorf("failed setting VM disks: %w", err)
		}
		if !res {
			return errReqFailed
		}
		fmt.Printf("Disk disconnected from VM\n")

		return nil
	},
}

var VMDisksCmd = &cobra.Command{
	Use:   "disk",
	Short: "Disk related operations on VMs",
	Long:  "List disks attached to VMs, attach disks to VMs and un-attach disks from VMs",
}
