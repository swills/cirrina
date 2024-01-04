package cmd

import (
	"cirrina/cirrinactl/rpc"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"io"
	"os"
	"sort"
	"strconv"
	"time"
)

var DiskName string
var DiskDescription string
var DiskDescriptionChanged bool
var DiskType = "nvme"
var DiskTypeChanged bool
var DiskDevType = "FILE"
var DiskSize = "1G"
var DiskId string
var DiskDirect = false
var DiskDirectChanged = false
var DiskCache = true
var DiskCacheChanged = false
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
		for _, diskId := range res {
			diskInfo, err := rpc.GetDiskInfo(diskId)
			if err != nil {
				return err
			}

			var vmId string
			vmId, err = rpc.DiskGetVmId(diskId)
			if err != nil {
				return err
			}
			var vmName string
			if vmId != "" {
				vmName, err = rpc.VmIdToName(vmId)
				if err != nil {
					return err
				}
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
				id:     diskId,
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
		if ShowUUID {
			t.AppendHeader(
				table.Row{"NAME", "UUID", "TYPE", "SIZE", "USAGE", "VM", "DEV-TYPE", "CACHE", "DIRECT", "DESCRIPTION"},
			)
		} else {
			t.AppendHeader(
				table.Row{"NAME", "TYPE", "SIZE", "USAGE", "VM", "DEV-TYPE", "CACHE", "DIRECT", "DESCRIPTION"},
			)
		}
		t.SetStyle(myTableStyle)
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 3, Align: text.AlignRight, AlignHeader: text.AlignRight},
			{Number: 4, Align: text.AlignRight, AlignHeader: text.AlignRight},
		})
		for _, diskName := range names {
			if ShowUUID {
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
			} else {
				t.AppendRow(table.Row{
					diskName,
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
		var err error
		if DiskId == "" {
			DiskId, err = rpc.DiskNameToId(DiskName)
			if err != nil {
				return err
			}
			if DiskId == "" {
				return errors.New("disk not found")
			}
		}
		err = rpc.RmDisk(DiskId)
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
		DiskDirectChanged = cmd.Flags().Changed("direct")
		DiskCacheChanged = cmd.Flags().Changed("cache")
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		var newDescr *string
		var newType *string
		var newDirect *bool
		var newCache *bool

		if DiskId == "" {
			DiskId, err = rpc.DiskNameToId(DiskName)
			if err != nil {
				return err
			}
			if DiskId == "" {
				return errors.New("disk not found")
			}
		}

		if DiskDescriptionChanged {
			newDescr = &DiskDescription
		}

		if DiskTypeChanged {
			newType = &DiskType
		}

		if DiskDirectChanged {
			newDirect = &DiskDirect
		}

		if DiskCacheChanged {
			newCache = &DiskCache
		}

		// TODO size

		err = rpc.UpdateDisk(DiskId, newDescr, newType, newDirect, newCache)
		if err != nil {
			return err
		}
		fmt.Printf("Updated disk\n")
		return nil
	},
}

func trackDiskUpload(pw progress.Writer, diskSize int64, f2 *os.File) {
	var err error

	checksumTracker := progress.Tracker{
		Message: "Calculating checksum",
		Total:   diskSize,
		Units:   progress.UnitsBytes,
	}
	pw.AppendTracker(&checksumTracker)
	checksumTracker.Start()

	var f *os.File
	f, err = os.Open(DiskFilePath)
	if err != nil {
		fmt.Printf("error opening file: %s\n", err)
	}

	hasher := sha512.New()

	var complete bool
	var n int64
	var checksumTotal int64
	for !complete {
		n, err = io.CopyN(hasher, f, 1024*1024)
		checksumTotal = checksumTotal + n
		checksumTracker.SetValue(checksumTotal)
		if err != nil {
			if err == io.EOF {
				complete = true
			} else {
				checksumTracker.MarkAsErrored()
			}
		}
	}

	diskChecksum := hex.EncodeToString(hasher.Sum(nil))
	err = f.Close()
	if err != nil {
		fmt.Printf("error closing file: %s\n", err)
	}
	checksumTracker.MarkAsDone()

	uploadTracker := progress.Tracker{
		Message: "Uploading",
		Total:   diskSize,
		Units:   progress.UnitsBytes,
	}
	pw.AppendTracker(&uploadTracker)
	uploadTracker.Start()

	if DiskId == "" {
		panic("empty disk id")
	}
	var upload <-chan rpc.UploadStat
	upload, err = rpc.DiskUpload(DiskId, diskChecksum, uint64(diskSize), f2)
	if err != nil {
		uploadTracker.MarkAsErrored()
		return
	}
	for !uploadTracker.IsDone() {
		select {
		case uploadStatEvent := <-upload:
			if uploadStatEvent.Err != nil {
				uploadTracker.MarkAsErrored()
			}
			if uploadStatEvent.UploadedChunk {
				newTotal := uploadTracker.Value() + int64(uploadStatEvent.UploadedBytes)
				if newTotal > diskSize {
					panic("uploaded more bytes than size of file")
				}
				// prevent uploadTracker being done before the Complete message arrives
				if newTotal == diskSize {
					newTotal--
				}
				uploadTracker.SetValue(newTotal)
			}
			if uploadStatEvent.Complete {
				uploadTracker.MarkAsDone()
			}
		}
	}
	return
}

func uploadDiskWithStatus() error {
	var err error
	var fi os.FileInfo
	fi, err = os.Stat(DiskFilePath)
	if err != nil {
		return err
	}
	diskSize := fi.Size()

	var f2 *os.File
	f2, err = os.Open(DiskFilePath)
	if err != nil {
		return err
	}

	pw := progress.NewWriter()
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetStyle(progress.StyleBlocks)
	pw.Style().Visibility.ETA = true
	pw.Style().Options.ETAPrecision = time.Second
	pw.Style().Options.SpeedPrecision = time.Second
	pw.Style().Options.TimeInProgressPrecision = time.Second
	pw.Style().Options.TimeDonePrecision = time.Second
	pw.Style().Options.TimeOverallPrecision = time.Second
	pw.SetAutoStop(false)
	pw.SetMessageWidth(20)

	go pw.Render()
	go trackDiskUpload(pw, diskSize, f2)

	// wait for upload to start
	for !pw.IsRenderInProgress() {
		time.Sleep(time.Millisecond * 100)
	}

	// wait for upload to finish
	for pw.IsRenderInProgress() {
		if pw.LengthActive() == 0 {
			pw.Stop()
		}
		time.Sleep(time.Millisecond * 100)
	}
	return nil
}

func uploadDiskWithoutStatus() error {
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
}

var DiskUploadCmd = &cobra.Command{
	Use:          "upload",
	Short:        "Upload a disk image",
	Long:         "Upload a disk image from local storage",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		var diskVmId string
		var diskVmStatus string

		err = hostPing()
		if err != nil {
			return errors.New("host not available")
		}

		if DiskId == "" {
			var aNotFoundErr *rpc.NotFoundError
			DiskId, err = rpc.DiskNameToId(DiskName)
			if err != nil {
				if errors.As(err, &aNotFoundErr) {
					DiskId, err = rpc.AddDisk(DiskName, DiskDescription, DiskSize, DiskType, DiskDevType, DiskCache, DiskDirect)
					if err != nil {
						return err
					}
				} else {
					return err
				}
			}
		}

		diskVmId, err = rpc.DiskGetVmId(DiskId)
		if err != nil {
			return err
		}
		if diskVmId != "" {
			diskVmStatus, _, _, err = rpc.GetVMState(diskVmId)
			if err != nil {
				return err
			}
			if diskVmStatus != "stopped" {
				return errors.New("unable to upload disk used by running VM")
			}
		}

		if CheckReqStat {
			return uploadDiskWithStatus()
		} else {
			return uploadDiskWithoutStatus()
		}
	},
}

var DiskCmd = &cobra.Command{
	Use:   "disk",
	Short: "Create, list, modify, destroy virtual disks",
}

func init() {
	disableFlagSorting(DiskCmd)

	disableFlagSorting(DiskListCmd)
	DiskListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	DiskListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)

	disableFlagSorting(DiskCreateCmd)
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

	disableFlagSorting(DiskRemoveCmd)
	addNameOrIdArgs(DiskRemoveCmd, &DiskName, &DiskId, "disk")

	disableFlagSorting(DiskUpdateCmd)
	addNameOrIdArgs(DiskUpdateCmd, &DiskName, &DiskId, "disk")
	DiskUpdateCmd.Flags().StringVarP(&DiskDescription,
		"description", "d", DiskDescription, "description of disk",
	)
	DiskUpdateCmd.Flags().StringVarP(&DiskType, "type", "t", DiskType, "type of disk - nvme, ahci, or virtioblk")
	DiskUpdateCmd.Flags().BoolVar(&DiskCache,
		"cache", DiskCache, "Enable or disable OS caching for this disk",
	)
	DiskUpdateCmd.Flags().BoolVar(&DiskDirect,
		"direct", DiskDirect, "Enable or disable synchronous writes for this disk",
	)

	disableFlagSorting(DiskUploadCmd)
	addNameOrIdArgs(DiskUploadCmd, &DiskName, &DiskId, "disk")
	DiskUploadCmd.Flags().StringVarP(&DiskFilePath,
		"path", "p", DiskFilePath, "Path to Disk File to upload",
	)
	err = DiskUploadCmd.MarkFlagRequired("path")
	if err != nil {
		panic(err)
	}
	DiskUploadCmd.Flags().BoolVarP(&CheckReqStat, "status", "s", CheckReqStat, "Check status")

	DiskCmd.AddCommand(DiskListCmd)
	DiskCmd.AddCommand(DiskCreateCmd)
	DiskCmd.AddCommand(DiskRemoveCmd)
	DiskCmd.AddCommand(DiskUpdateCmd)
	DiskCmd.AddCommand(DiskUploadCmd)
}
