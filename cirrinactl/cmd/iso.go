package cmd

import (
	"cirrina/cirrinactl/rpc"
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
)

var (
	IsoName        string
	IsoDescription string
	IsoId          string
	IsoFilePath    string
)

var IsoListCmd = &cobra.Command{
	Use:          "list",
	Short:        "List ISOs",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ids, err := rpc.GetIsoIds()
		if err != nil {
			return err
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
	RunE: func(cmd *cobra.Command, args []string) error {
		if IsoName == "" {
			return errors.New("empty ISO name")
		}
		res, err := rpc.AddIso(IsoName, IsoDescription)
		if err != nil {
			return err
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

	if IsoId == "" {
		panic("empty iso id")
	}
	var upload <-chan rpc.UploadStat
	upload, err = rpc.IsoUpload(IsoId, isoChecksum, uint64(isoSize), f2)
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
		return err
	}
	isoSize := fi.Size()

	var f2 *os.File
	f2, err = os.Open(IsoFilePath)
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
		return err
	}
	isoSize := fi.Size()
	var f *os.File
	f, err = os.Open(IsoFilePath)
	if err != nil {
		return err
	}
	hasher := sha512.New()
	fmt.Printf("Calculating iso checksum\n")
	if _, err = io.Copy(hasher, f); err != nil {
		return err
	}
	isoChecksum := hex.EncodeToString(hasher.Sum(nil))
	err = f.Close()
	if err != nil {
		return err
	}
	var f2 *os.File
	f2, err = os.Open(IsoFilePath)
	if err != nil {
		return err
	}
	fmt.Printf("Uploading iso. file-path=%s, id=%s, size=%d, checksum=%s\n",
		IsoFilePath,
		IsoId,
		isoSize,
		isoChecksum,
	)
	fmt.Printf("Streaming: ")
	var upload <-chan rpc.UploadStat
	upload, err = rpc.IsoUpload(IsoId, isoChecksum, uint64(isoSize), f2)
	if err != nil {
		return err
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
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		err = hostPing()
		if err != nil {
			return errors.New("host not available")
		}

		if IsoId == "" {
			var aNotFoundErr *rpc.NotFoundError
			IsoId, err = rpc.IsoNameToId(IsoName)
			if err != nil {
				if errors.As(err, &aNotFoundErr) {
					IsoId, err = rpc.AddIso(IsoName, IsoDescription)
					if err != nil {
						return err
					}
				} else {
					return err
				}
			}
		}

		if CheckReqStat {
			return uploadIsoWithStatus()
		} else {
			return uploadIsoWithoutStatus()
		}
	},
}

var IsoRemoveCmd = &cobra.Command{
	Use:          "remove",
	Short:        "Remove an ISO",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if IsoId == "" {
			IsoId, err = rpc.IsoNameToId(IsoName)
			if err != nil {
				return err
			}
			if IsoId == "" {
				return errors.New("ISO not found")
			}
		}
		err = rpc.RmIso(IsoId)
		if err != nil {
			return err
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
	addNameOrIdArgs(IsoRemoveCmd, &IsoName, &IsoId, "ISO")

	disableFlagSorting(IsoUploadCmd)
	addNameOrIdArgs(IsoUploadCmd, &IsoName, &IsoId, "ISO")
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
