package cmd

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	"cirrina/cirrinactl/rpc"
)

var (
	DiskName               string
	DiskDescription        string
	DiskDescriptionChanged bool
	DiskType               = "nvme"
	DiskTypeChanged        bool
	DiskDevType            = "FILE"
	DiskSize               = "1G"
	DiskID                 string
	DiskDirect             = false
	DiskDirectChanged      = false
	DiskCache              = true
	DiskCacheChanged       = false
	DiskFilePath           string
)

var DiskListCmd = &cobra.Command{
	Use:          "list",
	Short:        "list disks",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		res, err := rpc.GetDisks()
		if err != nil {
			return fmt.Errorf("failed getting disk list: %w", err)
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
		for _, diskID := range res {
			diskInfo, err := rpc.GetDiskInfo(diskID)
			if err != nil {
				return fmt.Errorf("failed getting disk info for disk %s: %w", diskID, err)
			}

			var vmID string
			vmID, err = rpc.DiskGetVMID(diskID)
			if err != nil {
				return fmt.Errorf("failed getting vm info for disk %s: %w", diskID, err)
			}
			var vmName string
			if vmID != "" {
				vmName, err = rpc.VMIdToName(vmID)
				if err != nil {
					vmName = ""
				}
			}

			var aDiskListInfo diskListInfo

			aDiskListInfo.id = diskID
			aDiskListInfo.vmName = vmName
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

		sort.Strings(names)
		diskTableWriter := table.NewWriter()
		diskTableWriter.SetOutputMirror(os.Stdout)
		diskTableWriter.SetStyle(myTableStyle)
		if ShowUUID {
			if ShowDiskSizeUsage {
				diskTableWriter.AppendHeader(
					table.Row{"NAME", "UUID", "TYPE", "SIZE", "USAGE", "VM", "DEV-TYPE", "CACHE", "DIRECT", "DESCRIPTION"},
				)
				diskTableWriter.SetColumnConfigs([]table.ColumnConfig{
					{Number: 4, Align: text.AlignRight, AlignHeader: text.AlignRight},
					{Number: 5, Align: text.AlignRight, AlignHeader: text.AlignRight},
				})
			} else {
				diskTableWriter.AppendHeader(
					table.Row{"NAME", "UUID", "TYPE", "VM", "DEV-TYPE", "CACHE", "DIRECT", "DESCRIPTION"},
				)
			}
		} else {
			if ShowDiskSizeUsage {
				diskTableWriter.AppendHeader(
					table.Row{"NAME", "TYPE", "SIZE", "USAGE", "VM", "DEV-TYPE", "CACHE", "DIRECT", "DESCRIPTION"},
				)
				diskTableWriter.SetColumnConfigs([]table.ColumnConfig{
					{Number: 3, Align: text.AlignRight, AlignHeader: text.AlignRight},
					{Number: 4, Align: text.AlignRight, AlignHeader: text.AlignRight},
				})
			} else {
				diskTableWriter.AppendHeader(
					table.Row{"NAME", "TYPE", "VM", "DEV-TYPE", "CACHE", "DIRECT", "DESCRIPTION"},
				)
			}
		}

		for _, diskName := range names {
			if ShowUUID {
				if ShowDiskSizeUsage {
					diskTableWriter.AppendRow(table.Row{
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
					diskTableWriter.AppendRow(table.Row{
						diskName,
						diskInfos[diskName].id,
						diskInfos[diskName].info.DiskType,
						diskInfos[diskName].vmName,
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
						diskInfos[diskName].vmName,
						diskInfos[diskName].info.DiskDevType,
						diskInfos[diskName].info.Cache,
						diskInfos[diskName].info.Direct,
						diskInfos[diskName].info.Descr,
					})
				} else {
					diskTableWriter.AppendRow(table.Row{
						diskName,
						diskInfos[diskName].info.DiskType,
						diskInfos[diskName].vmName,
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

var DiskCreateCmd = &cobra.Command{
	Use:          "create",
	Short:        "create virtual disk",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		if DiskName == "" {
			return errDiskEmptyName
		}
		res, err := rpc.AddDisk(DiskName, DiskDescription, DiskSize, DiskType, DiskDevType, DiskCache, DiskDirect)
		if err != nil {
			return fmt.Errorf("failed adding disk %s: %w", DiskName, err)
		}
		fmt.Printf("Disk created. ID: %s\n", res)

		return nil
	},
}

var DiskDeleteCmd = &cobra.Command{
	Use:          "delete",
	Short:        "delete virtual disk",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error
		if DiskID == "" {
			DiskID, err = rpc.DiskNameToID(DiskName)
			if err != nil {
				return fmt.Errorf("failed getting disk ID: %w", err)
			}
			if DiskID == "" {
				return errDiskNotFound
			}
		}
		err = rpc.RmDisk(DiskID)
		if err != nil {
			return fmt.Errorf("failed removing disk: %w", err)
		}
		fmt.Printf("Disk deleted\n")

		return nil
	},
}

var DiskUpdateCmd = &cobra.Command{
	Use:          "modify",
	Short:        "modify virtual disk",
	SilenceUsage: true,
	Args: func(cmd *cobra.Command, _ []string) error {
		DiskDescriptionChanged = cmd.Flags().Changed("description")
		DiskTypeChanged = cmd.Flags().Changed("type")
		DiskDirectChanged = cmd.Flags().Changed("direct")
		DiskCacheChanged = cmd.Flags().Changed("cache")

		return nil
	},
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error
		var newDescr *string
		var newType *string
		var newDirect *bool
		var newCache *bool

		if DiskID == "" {
			DiskID, err = rpc.DiskNameToID(DiskName)
			if err != nil {
				return fmt.Errorf("failed getting disk ID: %w", err)
			}
			if DiskID == "" {
				return errDiskNotFound
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

		err = rpc.UpdateDisk(DiskID, newDescr, newType, newDirect, newCache)
		if err != nil {
			return fmt.Errorf("failed updating disk: %w", err)
		}
		fmt.Printf("Updated disk\n")

		return nil
	},
}

var DiskWipeCmd = &cobra.Command{
	Use:          "wipe",
	Short:        "wipe virtual disk, permanently destroys all data on the virtual disk",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error
		if DiskID == "" {
			DiskID, err = rpc.DiskNameToID(DiskName)
			if err != nil {
				return fmt.Errorf("failed getting disk ID: %w", err)
			}
			if DiskID == "" {
				return errDiskNotFound
			}
		}

		var reqID string
		var reqStat rpc.ReqStatus

		reqID, err = rpc.WipeDisk(DiskID)
		if err != nil {
			return fmt.Errorf("failed wiping disk: %w", err)
		}

		if !CheckReqStat {
			fmt.Print("Disk Wiped\n")

			return nil
		}

		fmt.Printf("Wiping Disk: ")
		for {
			reqStat, err = rpc.ReqStat(reqID)
			if err != nil {
				return fmt.Errorf("failed checking request status: %w", err)
			}
			if reqStat.Success {
				fmt.Printf(" done")
			}
			if reqStat.Complete {
				break
			}
			fmt.Printf(".")
			time.Sleep(time.Second)
			rpc.ResetConnTimeout()
		}
		fmt.Printf("\n")

		return nil
	},
}

func trackDiskUpload(diskProgressWriter progress.Writer, diskSize int64, diskFile *os.File) {
	var err error

	diskChecksum, err := checksumWithProgress(diskProgressWriter, diskSize)
	if err != nil {
		return
	}

	uploadTracker := progress.Tracker{
		Message: "Uploading",
		Total:   diskSize,
		Units:   progress.UnitsBytes,
	}
	diskProgressWriter.AppendTracker(&uploadTracker)
	uploadTracker.Start()

	var upload <-chan rpc.UploadStat

	upload, err = rpc.DiskUpload(DiskID, diskChecksum, uint64(diskSize), diskFile)
	if err != nil {
		uploadTracker.MarkAsErrored()

		return
	}

	for !uploadTracker.IsDone() {
		uploadStatEvent := <-upload
		if uploadStatEvent.Err != nil {
			uploadTracker.MarkAsErrored()
		}

		if uploadStatEvent.UploadedChunk {
			newTotal := uploadTracker.Value() + int64(uploadStatEvent.UploadedBytes)
			if newTotal > diskSize {
				fmt.Printf("uploaded more bytes than size of file")
				uploadTracker.MarkAsErrored()

				return
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

func checksumWithProgress(diskProgressWriter progress.Writer, diskSize int64) (string, error) {
	var err error

	checksumTracker := progress.Tracker{
		Message: "Calculating checksum",
		Total:   diskSize,
		Units:   progress.UnitsBytes,
	}
	diskProgressWriter.AppendTracker(&checksumTracker)
	checksumTracker.Start()

	var diskHasherFile *os.File

	diskHasherFile, err = os.Open(DiskFilePath)
	if err != nil {
		return "", fmt.Errorf("error opening file: %w", err)
	}

	hasher := sha512.New()

	var complete bool

	var nBytes int64

	var checksumTotal int64

	for !complete {
		nBytes, err = io.CopyN(hasher, diskHasherFile, 1024*1024)
		checksumTotal += nBytes
		checksumTracker.SetValue(checksumTotal)

		if err != nil {
			if errors.Is(err, io.EOF) {
				complete = true
			} else {
				checksumTracker.MarkAsErrored()

				return "", fmt.Errorf("error hashing file: %w", err)
			}
		}
	}

	diskChecksum := hex.EncodeToString(hasher.Sum(nil))

	err = diskHasherFile.Close()
	if err != nil {
		return "", fmt.Errorf("error closing file: %w", err)
	}

	checksumTracker.MarkAsDone()

	return diskChecksum, nil
}

func uploadDiskWithStatus() error {
	var err error

	var diskFileInfo os.FileInfo

	diskFileInfo, err = os.Stat(DiskFilePath)
	if err != nil {
		return fmt.Errorf("failed stating disk: %w", err)
	}

	diskSize := diskFileInfo.Size()

	var diskFile *os.File

	diskFile, err = os.Open(DiskFilePath)
	if err != nil {
		return fmt.Errorf("failed opening disk: %w", err)
	}

	diskUploadProgressWriter := progress.NewWriter()
	diskUploadProgressWriter.SetTrackerPosition(progress.PositionRight)
	diskUploadProgressWriter.SetStyle(progress.StyleBlocks)

	diskUploadProgressWriter.Style().Visibility.ETA = true
	diskUploadProgressWriter.Style().Options.ETAPrecision = time.Second
	diskUploadProgressWriter.Style().Options.SpeedPrecision = time.Second
	diskUploadProgressWriter.Style().Options.TimeInProgressPrecision = time.Second
	diskUploadProgressWriter.Style().Options.TimeDonePrecision = time.Second
	diskUploadProgressWriter.Style().Options.TimeOverallPrecision = time.Second
	diskUploadProgressWriter.SetAutoStop(false)
	diskUploadProgressWriter.SetMessageLength(20)

	go diskUploadProgressWriter.Render()
	go trackDiskUpload(diskUploadProgressWriter, diskSize, diskFile)

	// wait for upload to start
	for !diskUploadProgressWriter.IsRenderInProgress() {
		time.Sleep(time.Millisecond * 100)
	}

	// wait for upload to finish
	for diskUploadProgressWriter.IsRenderInProgress() {
		if diskUploadProgressWriter.LengthActive() == 0 {
			diskUploadProgressWriter.Stop()
		}

		time.Sleep(time.Millisecond * 100)
	}

	return nil
}

func checksumWithoutProgress() (int64, string, error) {
	var err error

	var diskFileInfo os.FileInfo

	diskFileInfo, err = os.Stat(DiskFilePath)
	if err != nil {
		return 0, "", fmt.Errorf("failed stating disk: %w", err)
	}

	diskSize := diskFileInfo.Size()

	var diskHasherFile *os.File

	diskHasherFile, err = os.Open(DiskFilePath)
	if err != nil {
		return 0, "", fmt.Errorf("failed opening disk: %w", err)
	}

	hasher := sha512.New()

	fmt.Printf("Calculating disk checksum\n")

	if _, err = io.Copy(hasher, diskHasherFile); err != nil {
		return 0, "", fmt.Errorf("failed copying data from disk: %w", err)
	}

	diskChecksum := hex.EncodeToString(hasher.Sum(nil))

	err = diskHasherFile.Close()
	if err != nil {
		return 0, "", fmt.Errorf("failed closing disk: %w", err)
	}

	return diskSize, diskChecksum, nil
}

func uploadDiskWithoutStatus() error {
	var err error

	diskSize, diskChecksum, err := checksumWithoutProgress()
	if err != nil {
		return err
	}

	var diskFile *os.File

	diskFile, err = os.Open(DiskFilePath)
	if err != nil {
		return fmt.Errorf("failed opening disk: %w", err)
	}

	fmt.Printf("Uploading disk. file-path=%s, id=%s, size=%d, checksum=%s\n",
		DiskFilePath,
		DiskID,
		diskSize,
		diskChecksum,
	)

	fmt.Printf("Streaming: ")

	var upload <-chan rpc.UploadStat

	upload, err = rpc.DiskUpload(DiskID, diskChecksum, uint64(diskSize), diskFile)
	if err != nil {
		return fmt.Errorf("failed uploading disk: %w", err)
	}
UploadLoop:
	for {
		uploadStatEvent := <-upload
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
	fmt.Printf("\n")
	fmt.Printf("Disk Upload complete\n")

	return nil
}

var DiskUploadCmd = &cobra.Command{
	Use:          "upload",
	Short:        "Upload a disk image",
	Long:         "Upload a disk image from local storage",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error
		var diskVMID string
		var diskVMStatus string

		err = hostPing()
		if err != nil {
			return fmt.Errorf("failed uploading disk: %w", err)
		}

		if DiskID == "" {
			DiskID, err = rpc.DiskNameToID(DiskName)
			if err != nil {
				if errors.Is(err, rpc.ErrNotFound) {
					DiskID, err = rpc.AddDisk(DiskName, DiskDescription, DiskSize, DiskType, DiskDevType, DiskCache, DiskDirect)
					if err != nil {
						return fmt.Errorf("failed creating disk: %w", err)
					}
				} else {
					return fmt.Errorf("failed getting disk ID: %w", err)
				}
			}
		}

		diskVMID, err = rpc.DiskGetVMID(DiskID)
		if err != nil {
			return fmt.Errorf("failed checking disk status: %w", err)
		}
		if diskVMID != "" {
			diskVMStatus, _, _, err = rpc.GetVMState(diskVMID)
			if err != nil {
				return fmt.Errorf("failed checking status of VM which uses disk: %w", err)
			}
			if diskVMStatus != "stopped" {
				return errDiskInUse
			}
		}

		if !CheckReqStat {
			return uploadDiskWithoutStatus()
		}

		return uploadDiskWithStatus()
	},
}

var DiskCmd = &cobra.Command{
	Use:   "disk",
	Short: "Create, list, modify, delete virtual disks",
}
