package rpc

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"cirrina/cirrina"
)

func AddIso(name string, descr string) (string, error) {
	var err error

	j := &cirrina.ISOInfo{
		Name:        &name,
		Description: &descr,
	}
	var res *cirrina.ISOID
	res, err = serverClient.AddISO(defaultServerContext, j)
	if err != nil {
		return "", fmt.Errorf("unable to add iso: %w", err)
	}

	return res.Value, nil
}

func GetIsoIDs() ([]string, error) {
	var err error
	var VMIDs []string
	var res cirrina.VMInfo_GetISOsClient
	res, err = serverClient.GetISOs(defaultServerContext, &cirrina.ISOsQuery{})
	if err != nil {
		return []string{}, fmt.Errorf("unable to get isos: %w", err)
	}
	for {
		VMID, err := res.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return []string{}, fmt.Errorf("unable to get isos: %w", err)
		}
		VMIDs = append(VMIDs, VMID.Value)
	}

	return VMIDs, nil
}

func RmIso(id string) error {
	var err error
	var res *cirrina.ReqBool
	res, err = serverClient.RemoveISO(defaultServerContext, &cirrina.ISOID{Value: id})
	if err != nil {
		return fmt.Errorf("unable to remove iso: %w", err)
	}
	if !res.Success {
		return errReqFailed
	}

	return nil
}

func GetIsoInfo(isoID string) (IsoInfo, error) {
	if isoID == "" {
		return IsoInfo{}, errIsoEmptyID
	}

	var err error

	var isoInfo *cirrina.ISOInfo
	isoInfo, err = serverClient.GetISOInfo(defaultServerContext, &cirrina.ISOID{Value: isoID})
	if err != nil {
		return IsoInfo{}, fmt.Errorf("unable to get iso info: %w", err)
	}

	return IsoInfo{
		Name:  *isoInfo.Name,
		Descr: *isoInfo.Description,
		Size:  *isoInfo.Size,
	}, nil
}

func IsoNameToID(name string) (string, error) {
	if name == "" {
		return "", errIsoEmptyName
	}
	isoIDs, err := GetIsoIDs()
	if err != nil {
		return "", err
	}

	found := false
	var isoID string
	for _, aIsoID := range isoIDs {
		var isoInfo IsoInfo
		isoInfo, err = GetIsoInfo(aIsoID)
		if err != nil {
			return "", err
		}
		if isoInfo.Name == name {
			if found {
				return "", errIsoDuplicate
			}
			found = true
			isoID = aIsoID
		}
	}
	if !found {
		return "", ErrNotFound
	}

	return isoID, nil
}

// func IsoIdToName(s string) (string, error) {
// 	var err error
// 	var res *cirrina.ISOInfo
// 	res, err = serverClient.GetISOInfo(defaultServerContext, &cirrina.ISOID{Value: s})
// 	if err != nil {
// 		return "", errors.New(status.Convert(err).Message())
// 	}
// 	return *res.Name, nil
// }

func IsoUpload(isoID string, isoChecksum string,
	isoSize uint64, isoFile *os.File,
) (<-chan UploadStat, error) {
	uploadStatChan := make(chan UploadStat, 1)

	if isoID == "" {
		return uploadStatChan, errIsoEmptyID
	}

	// actually send file, sending status to status channel
	go func(isoFile *os.File, uploadStatChan chan<- UploadStat) {
		defer func(isoFile *os.File) {
			_ = isoFile.Close()
		}(isoFile)
		var err error

		// prevent timeouts
		defaultServerContext = context.Background()

		thisIsoID := cirrina.ISOID{Value: isoID}

		setupReq := &cirrina.ISOImageRequest{
			Data: &cirrina.ISOImageRequest_Isouploadinfo{
				Isouploadinfo: &cirrina.ISOUploadInfo{
					Isoid:     &thisIsoID,
					Size:      isoSize,
					Sha512Sum: isoChecksum,
				},
			},
		}

		var stream cirrina.VMInfo_UploadIsoClient
		stream, err = serverClient.UploadIso(defaultServerContext)
		if err != nil {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           fmt.Errorf("unable to upload iso: %w", err),
			}
		}

		err = stream.Send(setupReq)
		if err != nil {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           fmt.Errorf("unable to upload iso: %w", err),
			}
		}

		reader := bufio.NewReader(isoFile)
		buffer := make([]byte, 1024*1024)

		var complete bool
		var nBytesRead int
		for !complete {
			nBytesRead, err = reader.Read(buffer)
			if errors.Is(err, io.EOF) {
				complete = true
			}
			if err != nil && !errors.Is(err, io.EOF) {
				uploadStatChan <- UploadStat{
					UploadedChunk: false,
					Complete:      false,
					Err:           fmt.Errorf("unable to upload iso: %w", err),
				}
			}
			dataReq := &cirrina.ISOImageRequest{
				Data: &cirrina.ISOImageRequest_Image{
					Image: buffer[:nBytesRead],
				},
			}
			err = stream.Send(dataReq)
			if err != nil {
				uploadStatChan <- UploadStat{
					UploadedChunk: false,
					Complete:      false,
					Err:           fmt.Errorf("unable to upload iso: %w", err),
				}
			}
			uploadStatChan <- UploadStat{
				UploadedChunk: true,
				Complete:      false,
				UploadedBytes: nBytesRead,
				Err:           nil,
			}
		}

		var reply *cirrina.ReqBool
		reply, err = stream.CloseAndRecv()
		if err != nil {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           fmt.Errorf("unable to upload iso: %w", err),
			}
		}
		if !reply.Success {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           errReqFailed,
			}
		}

		// finished!
		uploadStatChan <- UploadStat{
			UploadedChunk: false,
			Complete:      true,
			Err:           nil,
		}
	}(isoFile, uploadStatChan)

	return uploadStatChan, nil
}
