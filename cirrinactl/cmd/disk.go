package cmd

import (
	"cirrina/cirrinactl/rpc"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"github.com/spf13/cobra"
	"io"
	"os"
	"sort"
	"strconv"
)

var DiskName string
var DiskDescription string
var DiskDescriptionChanged bool
var DiskType = "nvme"
var DiskTypeChanged bool
var DiskDevType = "FILE"
var DiskSize = "1G"
var DiskId string
var DiskCache = true
var DiskDirect = false
var DiskFilePath string

var DiskListCmd = &cobra.Command{
	Use:          "list",
	Short:        "list disks",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := rpc.GetDisks()
		if err != nil {
			return err
		}

		var names []string
		type diskListInfo struct {
			info   rpc.DiskInfo
			id     string
			vmName string
			size   string
			usage  string
		}

		diskInfos := make(map[string]diskListInfo)
		for _, id := range res {
			diskInfo, err := rpc.GetDiskInfo(id)
			if err != nil {
				return err
			}

			var vmName string
			vmName, err = rpc.DiskGetVm(id)
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
				id:     id,
				vmName: vmName,
				info:   diskInfo,
				size:   diskSize,
				usage:  diskUsage,
			}
			names = append(names, diskInfo.Name)
		}

		sort.Strings(names)
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(
			table.Row{"NAME", "UUID", "TYPE", "SIZE", "USAGE", "VM", "DEV-TYPE", "CACHE", "DIRECT", "DESCRIPTION"},
		)
		t.SetStyle(myTableStyle)
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 4, Align: text.AlignRight, AlignHeader: text.AlignRight},
			{Number: 5, Align: text.AlignRight, AlignHeader: text.AlignRight},
		})
		for _, diskName := range names {
			t.AppendRow(table.Row{
				diskName,
				diskInfos[diskName].id,
				diskInfos[diskName].info.DiskType,
				diskInfos[diskName].size,
				diskInfos[diskName].usage,
				diskInfos[diskName].vmName,
				diskInfos[diskName].info.DiskDevType,
				diskInfos[diskName].info.Cache,
				diskInfos[diskName].info.Direct,
				diskInfos[diskName].info.Descr,
			})
		}
		t.Render()
		return nil
	},
}

var DiskCreateCmd = &cobra.Command{
	Use:          "create",
	Short:        "create virtual disk",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if DiskName == "" {
			return errors.New("empty disk name")
		}
		res, err := rpc.AddDisk(DiskName, DiskDescription, DiskSize, DiskType, DiskDevType, DiskCache, DiskDirect)
		if err != nil {
			return err
		}
		fmt.Printf("Disk created. id: %s\n", res)
		return nil
	},
}

var DiskRemoveCmd = &cobra.Command{
	Use:          "destroy",
	Short:        "remove virtual disk",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		diskId, err := rpc.DiskNameToId(DiskName)
		if err != nil {
			return err
		}
		err = rpc.RmDisk(diskId)
		if err != nil {
			return err
		}
		fmt.Printf("Disk deleted\n")
		return nil
	},
}

var DiskUpdateCmd = &cobra.Command{
	Use:          "modify",
	Short:        "modify virtual disk",
	SilenceUsage: true,
	Args: func(cmd *cobra.Command, args []string) error {
		DiskDescriptionChanged = cmd.Flags().Changed("description")
		DiskTypeChanged = cmd.Flags().Changed("type")
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		var newDescr *string
		var newType *string

		if DiskId == "" {
			DiskId, err := rpc.DiskNameToId(DiskName)
			if err != nil {
				return err
			}
			if DiskId == "" {
				return errors.New("disk not found")
			}
		}
		if DiskName == "" {
			DiskName, err = rpc.DiskIdToName(DiskId)
			if err != nil {
				return err
			}
		}

		// currently only support changing disk description, and type
		if DiskDescriptionChanged {
			newDescr = &DiskDescription
		}
		if DiskTypeChanged {
			newType = &DiskType
		}

		err = rpc.UpdateDisk(DiskId, newDescr, newType)
		if err != nil {
			return err
		}
		fmt.Printf("Updated disk\n")
		return nil
	},
}

var DiskUploadCmd = &cobra.Command{
	Use:          "upload",
	Short:        "Upload a disk image",
	Long:         "Upload a disk image from local storage",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		var fi os.FileInfo
		fi, err = os.Stat(DiskFilePath)
		if err != nil {
			return err
		}
		diskSize := fi.Size()

		var f *os.File
		f, err = os.Open(DiskFilePath)
		if err != nil {
			return err
		}

		hasher := sha512.New()

		fmt.Printf("Calculating disk checksum\n")
		if _, err = io.Copy(hasher, f); err != nil {
			return err
		}

		diskChecksum := hex.EncodeToString(hasher.Sum(nil))
		err = f.Close()
		if err != nil {
			return err
		}
		var f2 *os.File
		f2, err = os.Open(DiskFilePath)
		if err != nil {
			return err
		}

		fmt.Printf("Uploading disk. file-path=%s, id=%s, size=%d, checksum=%s\n",
			DiskFilePath,
			DiskId,
			diskSize,
			diskChecksum,
		)

		fmt.Printf("Streaming: ")
		var upload <-chan rpc.UploadStat
		upload, err = rpc.DiskUpload(DiskId, diskChecksum, uint64(diskSize), f2)
		if err != nil {
			return err
		}
	UploadLoop:
		for {
			select {
			case uploadStatEvent := <-upload:
				if uploadStatEvent.Err != nil {
					return uploadStatEvent.Err
				}
				if uploadStatEvent.UploadedChunk {
					fmt.Printf(".")
				}
				if uploadStatEvent.Complete {
					break UploadLoop
				}
			}
		}
		fmt.Printf("\n")
		fmt.Printf("Disk Upload complete\n")
		return nil
	},
}

var DiskCmd = &cobra.Command{
	Use:   "disk",
	Short: "Create, list, modify, destroy virtual disks",
}

func init() {
	disableFlagSorting(DiskCmd)

	DiskCreateCmd.Flags().StringVarP(&DiskName, "name", "n", DiskName, "name of disk")
	err := DiskCreateCmd.MarkFlagRequired("name")
	if err != nil {
		panic(err)
	}
	DiskCreateCmd.Flags().StringVarP(&DiskSize, "size", "s", DiskName, "size of disk")
	err = DiskCreateCmd.MarkFlagRequired("size")
	if err != nil {
		panic(err)
	}
	DiskCreateCmd.Flags().StringVarP(&DiskDescription,
		"description", "d", DiskDescription, "description of disk",
	)
	DiskCreateCmd.Flags().StringVarP(&DiskType, "type", "t", DiskType, "type of disk - nvme, ahci, or virtioblk")
	DiskCreateCmd.Flags().StringVar(&DiskDevType,
		"dev-type", DiskDevType, "Dev type of disk - file or zvol",
	)
	DiskCreateCmd.Flags().BoolVar(&DiskCache,
		"cache", DiskCache, "Enable or disable OS caching for this disk",
	)
	DiskCreateCmd.Flags().BoolVar(&DiskDirect,
		"direct", DiskDirect, "Enable or disable synchronous writes for this disk",
	)
	disableFlagSorting(DiskCreateCmd)

	DiskRemoveCmd.Flags().StringVarP(&DiskName, "name", "n", DiskName, "name of disk")
	DiskRemoveCmd.Flags().StringVarP(&DiskId, "id", "i", DiskId, "id of disk")
	DiskRemoveCmd.MarkFlagsOneRequired("name", "id")
	DiskRemoveCmd.MarkFlagsMutuallyExclusive("name", "id")
	disableFlagSorting(DiskRemoveCmd)

	DiskListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	disableFlagSorting(DiskListCmd)

	DiskUpdateCmd.Flags().StringVarP(&DiskName, "name", "n", DiskName, "name of disk")
	DiskUpdateCmd.Flags().StringVarP(&DiskId, "id", "i", DiskId, "id of disk")
	DiskUpdateCmd.MarkFlagsOneRequired("name", "id")
	DiskUpdateCmd.MarkFlagsMutuallyExclusive("name", "id")

	DiskUpdateCmd.Flags().StringVarP(&DiskDescription,
		"description", "d", DiskDescription, "description of disk",
	)
	DiskUpdateCmd.Flags().StringVarP(&DiskType, "type", "t", DiskType, "type of disk - nvme, ahci, or virtioblk")
	disableFlagSorting(DiskUpdateCmd)

	DiskUploadCmd.Flags().StringVarP(&DiskId, "id", "i", DiskId, "Id of Disk to upload")
	DiskUploadCmd.Flags().StringVarP(&DiskFilePath,
		"path", "p", DiskFilePath, "Path to Disk File to upload",
	)
	err = DiskUploadCmd.MarkFlagRequired("id")
	if err != nil {
		panic(err)
	}
	err = DiskUploadCmd.MarkFlagRequired("path")
	if err != nil {
		panic(err)
	}
	disableFlagSorting(DiskUploadCmd)

	DiskCmd.AddCommand(DiskListCmd)
	DiskCmd.AddCommand(DiskCreateCmd)
	DiskCmd.AddCommand(DiskRemoveCmd)
	DiskCmd.AddCommand(DiskUpdateCmd)
	DiskCmd.AddCommand(DiskUploadCmd)
}
