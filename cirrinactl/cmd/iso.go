package cmd

import (
	"cirrina/cirrinactl/rpc"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/table"
	"github.com/spf13/cobra"
	"io"
	"os"
	"sort"
	"strconv"
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
			id     string
			info   rpc.IsoInfo
			size   string
			vmName string
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

var IsoUploadCmd = &cobra.Command{
	Use:          "upload",
	Short:        "Upload an ISO",
	Long:         "Upload an ISO image from local storage",
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
		fmt.Printf("ISO Upload complete\n")
		return nil
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

	addNameOrIdArgs(IsoUploadCmd, &IsoName, &IsoId, "ISO")
	disableFlagSorting(IsoUploadCmd)
	IsoUploadCmd.Flags().StringVarP(&IsoFilePath,
		"path", "p", IsoFilePath, "Path to ISO File to upload",
	)
	err = IsoUploadCmd.MarkFlagRequired("path")
	if err != nil {
		panic(err)
	}

	IsoCmd.AddCommand(IsoListCmd)
	IsoCmd.AddCommand(IsoCreateCmd)
	IsoCmd.AddCommand(IsoRemoveCmd)
	IsoCmd.AddCommand(IsoUploadCmd)
}
