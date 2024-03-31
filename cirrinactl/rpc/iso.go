package rpc

import (
	"bufio"
	"cirrina/cirrina"
	"context"
	"errors"
	"io"
	"os"

	"google.golang.org/grpc/status"
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
		return "", errors.New(status.Convert(err).Message())
	}

	return res.Value, nil
}

func GetIsoIds() ([]string, error) {
	var err error
	var ids []string
	var res cirrina.VMInfo_GetISOsClient
	res, err = serverClient.GetISOs(defaultServerContext, &cirrina.ISOsQuery{})
	if err != nil {
		return []string{}, errors.New(status.Convert(err).Message())
	}
	for {
		VM, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []string{}, err
		}
		ids = append(ids, VM.Value)
	}
	return ids, nil
}

func RmIso(id string) error {
	var err error
	var res *cirrina.ReqBool
	res, err = serverClient.RemoveISO(defaultServerContext, &cirrina.ISOID{Value: id})
	if err != nil {
		return errors.New(status.Convert(err).Message())
	}
	if !res.Success {
		return errors.New("iso delete failure")
	}
	return nil
}

func GetIsoInfo(id string) (IsoInfo, error) {
	if id == "" {
		return IsoInfo{}, errors.New("id not specified")
	}

	var err error

	var isoInfo *cirrina.ISOInfo
	isoInfo, err = serverClient.GetISOInfo(defaultServerContext, &cirrina.ISOID{Value: id})
	if err != nil {
		return IsoInfo{}, errors.New(status.Convert(err).Message())
	}
	return IsoInfo{
		Name:  *isoInfo.Name,
		Descr: *isoInfo.Description,
		Size:  *isoInfo.Size,
	}, nil
}

func IsoNameToId(name string) (string, error) {
	if name == "" {
		return "", errors.New("iso name not specified")
	}
	isoIds, err := GetIsoIds()
	if err != nil {
		return "", err
	}

	found := false
	var isoId string
	for _, aIsoId := range isoIds {
		var isoInfo IsoInfo
		isoInfo, err = GetIsoInfo(aIsoId)
		if err != nil {
			return "", err
		}
		if err != nil {
			return "", err
		}
		if isoInfo.Name == name {
			if found {
				return "", errors.New("duplicate iso found")
			}
			found = true
			isoId = aIsoId
		}
	}
	if !found {
		return "", &NotFoundError{}
	}
	return isoId, nil
}

//func IsoIdToName(s string) (string, error) {
//	var err error
//	var res *cirrina.ISOInfo
//	res, err = serverClient.GetISOInfo(defaultServerContext, &cirrina.ISOID{Value: s})
//	if err != nil {
//		return "", errors.New(status.Convert(err).Message())
//	}
//	return *res.Name, nil
//}

func IsoUpload(isoId string, isoChecksum string,
	isoSize uint64, isoFile *os.File) (<-chan UploadStat, error) {
	uploadStatChan := make(chan UploadStat, 1)

	if isoId == "" {
		return uploadStatChan, errors.New("empty iso id")
	}

	// actually send file, sending status to status channel
	go func(isoFile *os.File, uploadStatChan chan<- UploadStat) {
		defer func(isoFile *os.File) {
			_ = isoFile.Close()
		}(isoFile)
		var err error

		// prevent timeouts
		defaultServerContext = context.Background()

		thisIsoId := cirrina.ISOID{Value: isoId}

		setupReq := &cirrina.ISOImageRequest{
			Data: &cirrina.ISOImageRequest_Isouploadinfo{
				Isouploadinfo: &cirrina.ISOUploadInfo{
					Isoid:     &thisIsoId,
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
				Err:           errors.New(status.Convert(err).Message()),
			}
		}

		err = stream.Send(setupReq)
		if err != nil {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           errors.New(status.Convert(err).Message()),
			}
		}

		reader := bufio.NewReader(isoFile)
		buffer := make([]byte, 1024*1024)

		var complete bool
		var n int
		for !complete {
			n, err = reader.Read(buffer)
			if err == io.EOF {
				complete = true
			}
			if err != nil && err != io.EOF {
				uploadStatChan <- UploadStat{
					UploadedChunk: false,
					Complete:      false,
					Err:           err,
				}
			}
			dataReq := &cirrina.ISOImageRequest{
				Data: &cirrina.ISOImageRequest_Image{
					Image: buffer[:n],
				},
			}
			err = stream.Send(dataReq)
			if err != nil {
				uploadStatChan <- UploadStat{
					UploadedChunk: false,
					Complete:      false,
					Err:           errors.New(status.Convert(err).Message()),
				}
			}
			uploadStatChan <- UploadStat{
				UploadedChunk: true,
				Complete:      false,
				UploadedBytes: n,
				Err:           nil,
			}
		}

		var reply *cirrina.ReqBool
		reply, err = stream.CloseAndRecv()
		if err != nil {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           errors.New(status.Convert(err).Message()),
			}
		}
		if !reply.Success {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           errors.New("failed"),
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
