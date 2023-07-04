package main

import (
	"bufio"
	"cirrina/cirrina"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
)

func addISO(namePtr *string, c cirrina.VMInfoClient, ctx context.Context, descrPtr *string) {
	if *namePtr == "" {
		log.Fatalf("Name not specified")
		return
	}
	res, err := c.AddISO(ctx, &cirrina.ISOInfo{
		Name:        namePtr,
		Description: descrPtr,
	})
	if err != nil {
		log.Fatalf("could not create ISO: %v", err)
		return
	}
	fmt.Printf("Created ISO %v\n", res.Value)
}

func uploadIso(c cirrina.VMInfoClient, ctx context.Context, idPtr *string, filePathPtr *string) {
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
