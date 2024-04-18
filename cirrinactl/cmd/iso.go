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
		ids, err := rpc.GetIsoIDs()
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

		for _, id := range ids {
			isoInfo, err := rpc.GetIsoInfo(id)
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
				id:   id,
				size: isoSize,
			}
			names = append(names, isoInfo.Name)
		}

		sort.Strings(names)

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

var IsoCreateCmd = &cobra.Command{
	Use:          "create",
	Short:        "Create an ISO",
	Long:         "Create a name entry for an ISO with no content -- see upload to add content",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		if IsoName == "" {
			return errIsoEmptyName
		}
		res, err := rpc.AddIso(IsoName, IsoDescription)
		if err != nil {
			return fmt.Errorf("error adding iso: %w", err)
		}
		fmt.Printf("ISO created. id: %s\n", res)

		return nil
	},
}

func trackIsoUpload(pw progress.Writer, isoSize int64, f2 *os.File) {
	var err error

	checksumTracker := progress.Tracker{
		Message: "Calculating checksum",
		Total:   isoSize,
		Units:   progress.UnitsBytes,
	}
	pw.AppendTracker(&checksumTracker)
	checksumTracker.Start()

	var f *os.File
	f, err = os.Open(IsoFilePath)
	if err != nil {
		fmt.Printf("error opening file: %s\n", err)
	}

	hasher := sha512.New()

	var complete bool
	var n int64
	var checksumTotal int64
	for !complete {
		n, err = io.CopyN(hasher, f, 1024*1024)
		checksumTotal += n
		checksumTracker.SetValue(checksumTotal)
		if err != nil {
			if errors.Is(err, io.EOF) {
				complete = true
			} else {
				checksumTracker.MarkAsErrored()
			}
		}
	}

	isoChecksum := hex.EncodeToString(hasher.Sum(nil))
	err = f.Close()
	if err != nil {
		fmt.Printf("error closing file: %s\n", err)
	}
	checksumTracker.MarkAsDone()

	uploadTracker := progress.Tracker{
		Message: "Uploading",
		Total:   isoSize,
		Units:   progress.UnitsBytes,
	}
	pw.AppendTracker(&uploadTracker)
	uploadTracker.Start()

	if IsoID == "" {
		panic("empty iso id")
	}
	var upload <-chan rpc.UploadStat
	upload, err = rpc.IsoUpload(IsoID, isoChecksum, uint64(isoSize), f2)
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
			if newTotal > isoSize {
				panic("uploaded more bytes than size of file")
			}
			// prevent uploadTracker being done before the Complete message arrives
			if newTotal == isoSize {
				newTotal--
			}
			uploadTracker.SetValue(newTotal)
		}
		if uploadStatEvent.Complete {
			uploadTracker.MarkAsDone()
		}
	}
}

func uploadIsoWithStatus() error {
	var err error
	var fi os.FileInfo
	fi, err = os.Stat(IsoFilePath)
	if err != nil {
		return fmt.Errorf("error stating iso: %w", err)
	}
	isoSize := fi.Size()

	var f2 *os.File
	f2, err = os.Open(IsoFilePath)
	if err != nil {
		return fmt.Errorf("error opening iso: %w", err)
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
	pw.SetMessageLength(20)

	go pw.Render()
	go trackIsoUpload(pw, isoSize, f2)

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

func uploadIsoWithoutStatus() error {
	var err error
	var fi os.FileInfo
	fi, err = os.Stat(IsoFilePath)
	if err != nil {
		return fmt.Errorf("error stating iso: %w", err)
	}
	isoSize := fi.Size()
	var f *os.File
	f, err = os.Open(IsoFilePath)
	if err != nil {
		return fmt.Errorf("error opening iso: %w", err)
	}
	hasher := sha512.New()
	fmt.Printf("Calculating iso checksum\n")
	if _, err = io.Copy(hasher, f); err != nil {
		return fmt.Errorf("error copying iso data: %w", err)
	}
	isoChecksum := hex.EncodeToString(hasher.Sum(nil))
	err = f.Close()
	if err != nil {
		return fmt.Errorf("error closing iso: %w", err)
	}
	var f2 *os.File
	f2, err = os.Open(IsoFilePath)
	if err != nil {
		return fmt.Errorf("error opening iso: %w", err)
	}
	fmt.Printf("Uploading iso. file-path=%s, id=%s, size=%d, checksum=%s\n",
		IsoFilePath,
		IsoID,
		isoSize,
		isoChecksum,
	)
	fmt.Printf("Streaming: ")
	var upload <-chan rpc.UploadStat
	upload, err = rpc.IsoUpload(IsoID, isoChecksum, uint64(isoSize), f2)
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
		err = hostPing()
		if err != nil {
			return errHostNotAvailable
		}

		if IsoID == "" {
			IsoID, err = rpc.IsoNameToID(IsoName)
			if err != nil {
				if errors.Is(err, errIsoNotFound) {
					IsoID, err = rpc.AddIso(IsoName, IsoDescription)
					if err != nil {
						return fmt.Errorf("error adding iso: %w", err)
					}
				} else {
					return fmt.Errorf("error getting iso id: %w", err)
				}
			}
		}

		if !CheckReqStat {
			return uploadIsoWithoutStatus()
		}

		return uploadIsoWithStatus()
	},
}

var IsoRemoveCmd = &cobra.Command{
	Use:          "remove",
	Short:        "Remove an ISO",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error
		if IsoID == "" {
			IsoID, err = rpc.IsoNameToID(IsoName)
			if err != nil {
				return fmt.Errorf("error getting iso id: %w", err)
			}
			if IsoID == "" {
				return errIsoNotFound
			}
		}
		err = rpc.RmIso(IsoID)
		if err != nil {
			return fmt.Errorf("error removing iso: %w", err)
		}

		fmt.Printf("ISO deleted\n")

		return nil
	},
}

var IsoCmd = &cobra.Command{
	Use:   "iso",
	Short: "Create, list, modify, destroy ISOs",
}

func init() {
	disableFlagSorting(IsoCmd)

	disableFlagSorting(IsoListCmd)
	IsoListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	IsoListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)

	disableFlagSorting(IsoCreateCmd)
	IsoCreateCmd.Flags().StringVarP(&IsoName,
		"name", "n", IsoName, "name of ISO",
	)
	err := IsoCreateCmd.MarkFlagRequired("name")
	if err != nil {
		panic(err)
	}
	IsoCreateCmd.Flags().StringVarP(&IsoDescription,
		"description", "d", IsoDescription, "description of ISO",
	)

	disableFlagSorting(IsoRemoveCmd)
	addNameOrIDArgs(IsoRemoveCmd, &IsoName, &IsoID, "ISO")

	disableFlagSorting(IsoUploadCmd)
	addNameOrIDArgs(IsoUploadCmd, &IsoName, &IsoID, "ISO")
	IsoUploadCmd.Flags().StringVarP(&IsoFilePath,
		"path", "p", IsoFilePath, "Path to ISO File to upload",
	)
	err = IsoUploadCmd.MarkFlagRequired("path")
	if err != nil {
		panic(err)
	}
	IsoUploadCmd.Flags().BoolVarP(&CheckReqStat, "status", "s", CheckReqStat, "Check status")

	IsoCmd.AddCommand(IsoListCmd)
	IsoCmd.AddCommand(IsoCreateCmd)
	IsoCmd.AddCommand(IsoRemoveCmd)
	IsoCmd.AddCommand(IsoUploadCmd)
}
