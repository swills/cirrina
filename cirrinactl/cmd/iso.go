package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"

	"cirrina/cirrinactl/rpc"
)

var (
	IsoName        string
	IsoDescription string
	IsoID          string
	IsoFilePath    string
)

var IsoListCmd = &cobra.Command{
	Use:          "list",
	Short:        "List ISOs",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		isoIDs, err := rpc.GetIsoIDs(ctx)
		if err != nil {
			return fmt.Errorf("error getting ISO IDs: %w", err)
		}

		var names []string
		type isoListInfo struct {
			id   string
			info rpc.IsoInfo
			size string
		}

		isoInfos := make(map[string]isoListInfo)

		for _, isoID := range isoIDs {
			isoInfo, err := rpc.GetIsoInfo(ctx, isoID)
			if err != nil {
				return fmt.Errorf("error getting iso info: %w", err)
			}
			var isoSize string

			if Humanize {
				isoSize = humanize.IBytes(isoInfo.Size)
			} else {
				isoSize = strconv.FormatUint(isoInfo.Size, 10)
			}

			isoInfos[isoInfo.Name] = isoListInfo{
				id:   isoID,
				size: isoSize,
			}
			names = append(names, isoInfo.Name)
		}

		sort.Strings(names)

		isoTableWriter := table.NewWriter()
		isoTableWriter.SetOutputMirror(os.Stdout)
		if ShowUUID {
			isoTableWriter.AppendHeader(table.Row{"NAME", "UUID", "SIZE", "DESCRIPTION"})
		} else {
			isoTableWriter.AppendHeader(table.Row{"NAME", "SIZE", "DESCRIPTION"})
		}
		isoTableWriter.SetStyle(myTableStyle)
		for _, name := range names {
			if ShowUUID {
				isoTableWriter.AppendRow(table.Row{
					name,
					isoInfos[name].id,
					isoInfos[name].size,
					isoInfos[name].info.Descr,
				})
			} else {
				isoTableWriter.AppendRow(table.Row{
					name,
					isoInfos[name].size,
					isoInfos[name].info.Descr,
				})
			}
		}
		isoTableWriter.Render()

		return nil
	},
}

var IsoCreateCmd = &cobra.Command{
	Use:          "create",
	Short:        "Create an ISO",
	Long:         "Create a name entry for an ISO with no content -- see upload to add content",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		if IsoName == "" {
			return errIsoEmptyName
		}
		res, err := rpc.AddIso(ctx, IsoName, IsoDescription)
		if err != nil {
			return fmt.Errorf("error adding iso: %w", err)
		}
		fmt.Printf("ISO created. id: %s\n", res)

		return nil
	},
}

func trackIsoUpload(isoProgressWriter progress.Writer, isoSize uint64, isoFile *os.File) {
	isoChecksum, err := checksumWithProgress(isoProgressWriter, isoSize, isoFile.Name())
	if err != nil {
		return
	}

	uploadTracker := progress.Tracker{
		Message: "Uploading",
		Total:   cast.ToInt64(isoSize),
		Units:   progress.UnitsBytes,
	}
	isoProgressWriter.AppendTracker(&uploadTracker)
	uploadTracker.Start()

	var upload <-chan rpc.UploadStat

	upload, err = rpc.IsoUpload(IsoID, isoChecksum, cast.ToUint64(isoSize), isoFile)
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
			newTotal := cast.ToUint64(uploadTracker.Value()) + uploadStatEvent.UploadedBytes
			if newTotal > isoSize {
				panic("uploaded more bytes than size of file")
			}
			// prevent uploadTracker being done before the Complete message arrives
			if newTotal == isoSize {
				newTotal--
			}

			uploadTracker.SetValue(cast.ToInt64(newTotal))
		}

		if uploadStatEvent.Complete {
			uploadTracker.MarkAsDone()
		}
	}
}

func uploadIsoWithProgress(isoFilePath string) error {
	var err error

	var isoFileInfo os.FileInfo

	isoFileInfo, err = os.Stat(isoFilePath)
	if err != nil {
		return fmt.Errorf("error stating iso: %w", err)
	}

	isoSize := isoFileInfo.Size()

	var isoFile *os.File

	isoFile, err = os.Open(isoFilePath)
	if err != nil {
		return fmt.Errorf("error opening iso: %w", err)
	}

	isoProgressWriter := progress.NewWriter()
	isoProgressWriter.SetTrackerPosition(progress.PositionRight)
	isoProgressWriter.SetStyle(progress.StyleBlocks)

	isoProgressWriter.Style().Visibility.ETA = true
	isoProgressWriter.Style().Options.ETAPrecision = time.Second
	isoProgressWriter.Style().Options.SpeedPrecision = time.Second
	isoProgressWriter.Style().Options.TimeInProgressPrecision = time.Second
	isoProgressWriter.Style().Options.TimeDonePrecision = time.Second
	isoProgressWriter.Style().Options.TimeOverallPrecision = time.Second
	isoProgressWriter.SetAutoStop(false)
	isoProgressWriter.SetMessageLength(20)

	go isoProgressWriter.Render()
	go trackIsoUpload(isoProgressWriter, cast.ToUint64(isoSize), isoFile)

	// wait for upload to start
	for !isoProgressWriter.IsRenderInProgress() {
		time.Sleep(time.Millisecond * 100)
	}

	// wait for upload to finish
	for isoProgressWriter.IsRenderInProgress() {
		if isoProgressWriter.LengthActive() == 0 {
			isoProgressWriter.Stop()
		}

		time.Sleep(time.Millisecond * 100)
	}

	return nil
}

func uploadIsoWithoutProgress(isoFilePath string) error {
	var err error

	var isoSize int64

	var isoChecksum string

	fmt.Printf("Calculating iso checksum\n")

	isoSize, isoChecksum, err = checksumWithoutProgress(isoFilePath)
	if err != nil {
		return err
	}

	var isoFile *os.File

	isoFile, err = os.Open(isoFilePath)
	if err != nil {
		return fmt.Errorf("error opening iso: %w", err)
	}

	fmt.Printf("Uploading iso. file-path=%s, id=%s, size=%d, checksum=%s\n",
		isoFilePath,
		IsoID,
		isoSize,
		isoChecksum,
	)

	fmt.Printf("Streaming: ")

	var upload <-chan rpc.UploadStat

	upload, err = rpc.IsoUpload(IsoID, isoChecksum, cast.ToUint64(isoSize), isoFile)
	if err != nil {
		return fmt.Errorf("error uploading iso: %w", err)
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
	fmt.Printf("ISO Upload complete\n")

	return nil
}

var IsoUploadCmd = &cobra.Command{
	Use:          "upload",
	Short:        "Upload an ISO",
	Long:         "Upload an ISO image from local storage",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		err = hostPing(ctx)
		if err != nil {
			return errHostNotAvailable
		}

		if IsoID == "" {
			IsoID, err = rpc.IsoNameToID(ctx, IsoName)
			if err != nil {
				if errors.Is(err, rpc.ErrNotFound) {
					IsoID, err = rpc.AddIso(ctx, IsoName, IsoDescription)
					if err != nil {
						return fmt.Errorf("error adding iso: %w", err)
					}
				} else {
					return fmt.Errorf("error getting iso id: %w", err)
				}
			}
		}

		if !CheckReqStat {
			return uploadIsoWithoutProgress(IsoFilePath)
		}

		return uploadIsoWithProgress(IsoFilePath)
	},
}

var IsoDeleteCmd = &cobra.Command{
	Use:          "delete",
	Short:        "Delete an ISO",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpc.ServerTimeout)*time.Second)
		defer cancel()

		if IsoID == "" {
			IsoID, err = rpc.IsoNameToID(ctx, IsoName)
			if err != nil {
				return fmt.Errorf("error getting iso id: %w", err)
			}
			if IsoID == "" {
				return errIsoNotFound
			}
		}
		err = rpc.RmIso(ctx, IsoID)
		if err != nil {
			return fmt.Errorf("error removing iso: %w", err)
		}

		fmt.Printf("ISO deleted\n")

		return nil
	},
}

var IsoCmd = &cobra.Command{
	Use:   "iso",
	Short: "Create, list, modify, delete ISOs",
}
