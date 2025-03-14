package rpc

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cast"

	"cirrina/cirrina"
)

func AddIso(ctx context.Context, name string, descr string) (string, error) {
	var err error

	isoInfo := &cirrina.ISOInfo{
		Name:        &name,
		Description: &descr,
	}

	var res *cirrina.ISOID

	res, err = serverClient.AddISO(ctx, isoInfo)
	if err != nil {
		return "", fmt.Errorf("unable to add iso: %w", err)
	}

	return res.GetValue(), nil
}

func GetIsoIDs(ctx context.Context) ([]string, error) {
	var err error

	var IsoIDs []string

	var res cirrina.VMInfo_GetISOsClient

	res, err = serverClient.GetISOs(ctx, &cirrina.ISOsQuery{})
	if err != nil {
		return []string{}, fmt.Errorf("unable to get isos: %w", err)
	}

	var isoID *cirrina.ISOID

	for {
		isoID, err = res.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return []string{}, fmt.Errorf("unable to get isos: %w", err)
		}

		IsoIDs = append(IsoIDs, isoID.GetValue())
	}

	return IsoIDs, nil
}

func RmIso(ctx context.Context, id string) error {
	var err error

	var res *cirrina.ReqBool

	res, err = serverClient.RemoveISO(ctx, &cirrina.ISOID{Value: id})
	if err != nil {
		return fmt.Errorf("unable to remove iso: %w", err)
	}

	if !res.GetSuccess() {
		return errReqFailed
	}

	return nil
}

func GetIsoInfo(ctx context.Context, isoID string) (IsoInfo, error) {
	if isoID == "" {
		return IsoInfo{}, errIsoEmptyID
	}

	var err error

	var isoInfo *cirrina.ISOInfo

	isoInfo, err = serverClient.GetISOInfo(ctx, &cirrina.ISOID{Value: isoID})
	if err != nil {
		return IsoInfo{}, fmt.Errorf("unable to get iso info: %w", err)
	}

	return IsoInfo{
		Name:  isoInfo.GetName(),
		Descr: isoInfo.GetDescription(),
		Size:  isoInfo.GetSize(),
	}, nil
}

func IsoNameToID(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", errIsoEmptyName
	}

	isoIDs, err := GetIsoIDs(ctx)
	if err != nil {
		return "", err
	}

	found := false

	var isoID string

	for _, aIsoID := range isoIDs {
		var isoInfo IsoInfo

		isoInfo, err = GetIsoInfo(ctx, aIsoID)
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

func isoUploadFile(isoID string, isoSize uint64, isoChecksum string, isoFile *os.File,
	uploadStatChan chan<- UploadStat) {
	var err error

	var stream cirrina.VMInfo_UploadIsoClient

	defer func(isoFile *os.File) {
		_ = isoFile.Close()
	}(isoFile)

	// prevent timeouts
	backgroundContext := context.Background()

	stream, err = serverClient.UploadIso(backgroundContext)
	if err != nil {
		uploadStatChan <- UploadStat{
			UploadedChunk: false,
			Complete:      false,
			Err:           err,
		}
	}

	err = isoUploadFileSetupRequest(isoID, isoSize, isoChecksum, stream, uploadStatChan)
	if err != nil {
		return
	}

	err = isoUploadFileBytes(isoFile, stream, uploadStatChan)
	if err != nil {
		return
	}

	isoUploadFileComplete(stream, uploadStatChan)
}

func isoUploadFileBytes(isoFile *os.File,
	stream cirrina.VMInfo_UploadIsoClient, uploadStatChan chan<- UploadStat) error {
	var err error

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
				Err:           err,
			}

			return fmt.Errorf("error reading file bytes: %w", err)
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
				Err:           err,
			}

			return fmt.Errorf("error sending file bytes: %w", err)
		}
		uploadStatChan <- UploadStat{
			UploadedChunk: true,
			Complete:      false,
			UploadedBytes: cast.ToUint64(nBytesRead),
			Err:           nil,
		}
	}

	return nil
}

func isoUploadFileComplete(stream cirrina.VMInfo_UploadIsoClient, uploadStatChan chan<- UploadStat) {
	var err error

	var reply *cirrina.ReqBool

	reply, err = stream.CloseAndRecv()
	if err != nil {
		uploadStatChan <- UploadStat{
			UploadedChunk: false,
			Complete:      false,
			Err:           fmt.Errorf("unable to upload iso: %w", err),
		}
	}

	if !reply.GetSuccess() {
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
}

func isoUploadFileSetupRequest(isoID string, isoSize uint64, isoChecksum string,
	stream cirrina.VMInfo_UploadIsoClient, uploadStatChan chan<- UploadStat) error {
	var err error

	setupReq := &cirrina.ISOImageRequest{
		Data: &cirrina.ISOImageRequest_Isouploadinfo{
			Isouploadinfo: &cirrina.ISOUploadInfo{
				Isoid:     &cirrina.ISOID{Value: isoID},
				Size:      isoSize,
				Sha512Sum: isoChecksum,
			},
		},
	}

	err = stream.Send(setupReq)
	if err != nil {
		uploadStatChan <- UploadStat{
			UploadedChunk: false,
			Complete:      false,
			Err:           fmt.Errorf("unable to upload iso: %w", err),
		}

		return fmt.Errorf("unable to upload iso: %w", err)
	}

	return nil
}

func IsoUpload(isoID string, isoChecksum string,
	isoSize uint64, isoFile *os.File,
) (<-chan UploadStat, error) {
	uploadStatChan := make(chan UploadStat, 1)

	if isoID == "" {
		return uploadStatChan, errIsoEmptyID
	}

	// actually send file, sending status to status channel
	go isoUploadFile(isoID, isoSize, isoChecksum, isoFile, uploadStatChan)

	return uploadStatChan, nil
}

func ISOGetVMIDs(ctx context.Context, isoID string) ([]string, error) {
	var err error

	if isoID == "" {
		return []string{}, errIsoEmptyID
	}

	var vmIDs []string

	var res cirrina.VMInfo_GetISOVMsClient

	res, err = serverClient.GetISOVMs(ctx, &cirrina.ISOID{Value: isoID})
	if err != nil {
		return []string{}, fmt.Errorf("unable to get ISOs: %w", err)
	}

	var vmID *cirrina.VMID

	for {
		vmID, err = res.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return []string{}, fmt.Errorf("unable to get isos: %w", err)
		}

		vmIDs = append(vmIDs, vmID.GetValue())
	}

	return vmIDs, nil
}
