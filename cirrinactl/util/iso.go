package util

import (
	"bufio"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"sort"

	"cirrina/cirrina"
	"cirrina/cirrinactl/rpc"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"google.golang.org/grpc/status"
)

func AddISO(namePtr *string, c cirrina.VMInfoClient, ctx context.Context, descrPtr *string) {
	if *namePtr == "" {
		log.Fatalf("Name not specified")
		return
	}

	j := &cirrina.ISOInfo{
		Name: namePtr,
	}

	if descrPtr != nil {
		j.Description = descrPtr
	}

	res, err := rpc.AddIso(j, c, ctx)
	if err != nil {
		log.Fatalf("could not create ISO: %v", err)
		return
	}
	fmt.Printf("Created ISO %v\n", res)
}

func UploadIso(c cirrina.VMInfoClient, ctx context.Context, idPtr *string, filePathPtr *string) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	if *filePathPtr == "" {
		log.Fatalf("File path not specified")
		return
	}

	thisisoid := cirrina.ISOID{Value: *idPtr}

	fi, err := os.Stat(*filePathPtr)
	if err != nil {
		log.Printf("error: %v", err)
	}
	isoSize := fi.Size()

	f, err := os.Open(*filePathPtr)
	if err != nil {
		log.Printf("error: %v", err)
	}

	hasher := sha512.New()

	if _, err := io.Copy(hasher, f); err != nil {
		log.Fatalf("Failed reading iso: %v", err)
		return
	}

	isoChecksum := hex.EncodeToString(hasher.Sum(nil))

	err = f.Close()
	if err != nil {
		log.Fatalf("failed closing: %v", err)
		return
	}

	log.Printf("Uploading %v for ISO %v, size %v checksum %v", *filePathPtr, *idPtr, isoSize, isoChecksum)

	stream, err := c.UploadIso(ctx)
	if err != nil {
		log.Fatalf("failed to get stream: %v", err)
		return
	}

	req := &cirrina.ISOImageRequest{
		Data: &cirrina.ISOImageRequest_Isouploadinfo{
			Isouploadinfo: &cirrina.ISOUploadInfo{
				Isoid:     &thisisoid,
				Size:      uint64(isoSize),
				Sha512Sum: isoChecksum,
			},
		},
	}

	err = stream.Send(req)
	if err != nil {
		fmt.Printf("Upload iso failed: %v\n", err)
	} else {
		fmt.Printf("sent iso info %v\n", *idPtr)
	}

	f, err = os.Open(*filePathPtr)
	if err != nil {
		log.Printf("error: %v", err)
	}

	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	// actually send file
	reader := bufio.NewReader(f)
	buffer := make([]byte, 1024*1024)

	log.Printf("Streaming: ")
	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("cannot read chunk to buffer: ", err)
		}

		req := &cirrina.ISOImageRequest{
			Data: &cirrina.ISOImageRequest_Image{
				Image: buffer[:n],
			},
		}

		err = stream.Send(req)
		if err != nil {
			fmt.Printf("\nsending req failed: %v\n", err)
		} else {
			fmt.Printf(".")
		}
	}
	fmt.Printf("\n")

	reply, err := stream.CloseAndRecv()
	if err != nil {
		fmt.Printf("cannot receive response: %v\n", err)
	}
	fmt.Printf("ISO Upload complete: %v\n", reply)
}

func RmIso(name string, c cirrina.VMInfoClient, ctx context.Context) {
	IsoId, err := rpc.IsoNameToId(&name, c, ctx)
	if err != nil {
		s := status.Convert(err)
		fmt.Printf("error: could not delete iso: %s\n", s.Message())
		return
	}
	_, err = rpc.RmIso(&IsoId, c, ctx)
	if err != nil {
		s := status.Convert(err)
		fmt.Printf("error: could not delete iso: %s\n", s.Message())
		return
	}
}

func ListIsos(c cirrina.VMInfoClient, ctx context.Context, useHumanize bool) {
	ids, err := rpc.GetIsoIds(c, ctx)
	if err != nil {
		fmt.Printf("failed to get iso IDs: %s\n", err.Error())
		return
	}

	var names []string
	type ThisIsoInfo struct {
		id    string
		descr string
		size  uint64
	}

	isoInfos := make(map[string]ThisIsoInfo)

	for _, id := range ids {
		res, err := rpc.GetIsoInfo(&id, c, ctx)
		if err != nil {
			log.Fatalf("could not get VM: %v", err)
			return
		}

		aIsoInfo := ThisIsoInfo{
			id:    id,
			descr: *res.Description,
			size:  *res.Size,
		}
		isoInfos[*res.Name] = aIsoInfo
		names = append(names, *res.Name)

	}

	sort.Strings(names)

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"NAME", "UUID", "SIZE", "DESCRIPTION"})
	t.SetStyle(table.Style{
		Name: "myNewStyle",
		Box: table.BoxStyle{
			MiddleHorizontal: "-", // bug in go-pretty causes panic if this is empty
			PaddingRight:     "  ",
		},
		Format: table.FormatOptions{
			Footer: text.FormatUpper,
			Header: text.FormatUpper,
			Row:    text.FormatDefault,
		},
		Options: table.Options{
			DrawBorder:      false,
			SeparateColumns: false,
			SeparateFooter:  false,
			SeparateHeader:  false,
			SeparateRows:    false,
		},
	})
	for _, name := range names {
		if useHumanize {
			t.AppendRow(table.Row{
				name,
				isoInfos[name].id,
				humanize.IBytes(isoInfos[name].size),
				isoInfos[name].descr,
			})
		} else {
			t.AppendRow(table.Row{
				name,
				isoInfos[name].id,
				isoInfos[name].size,
				isoInfos[name].descr,
			})
		}
	}
	t.Render()
}
