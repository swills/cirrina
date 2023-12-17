package rpc

import (
	"bufio"
	"cirrina/cirrina"
	"context"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"io"
	"os"
	"time"
)

func AddIso(name string, descr string) (string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	j := &cirrina.ISOInfo{
		Name:        &name,
		Description: &descr,
	}
	var res *cirrina.ISOID
	res, err = c.AddISO(ctx, j)
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}

	return res.Value, nil
}

func GetIsoIds() ([]string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return []string{}, err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()
	var ids []string
	var res cirrina.VMInfo_GetISOsClient
	res, err = c.GetISOs(ctx, &cirrina.ISOsQuery{})
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
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var res *cirrina.ReqBool
	res, err = c.RemoveISO(ctx, &cirrina.ISOID{Value: id})
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

	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return IsoInfo{}, err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var isoInfo *cirrina.ISOInfo
	isoInfo, err = c.GetISOInfo(ctx, &cirrina.ISOID{Value: id})
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
		return "", errors.New("iso not found")
	}
	return isoId, nil
}

func IsoIdToName(s string) (string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()
	var res *cirrina.ISOInfo
	res, err = c.GetISOInfo(ctx, &cirrina.ISOID{Value: s})
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	return *res.Name, nil
}

func IsoUpload(isoId string, isoChecksum string,
	isoSize uint64, isoFile *os.File) (<-chan UploadStat, error) {
	uploadStatChan := make(chan UploadStat, 1)

	// actually send file, sending status to status channel
	go func(isoFile *os.File, uploadStatChan chan<- UploadStat) {
		defer func(isoFile *os.File) {
			_ = isoFile.Close()
		}(isoFile)

		conn, c, err := SetupConnNoTimeoutNoContext()
		if err != nil {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           err,
			}
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)

		timeout := 1 * time.Hour
		longCtx, longCancel := context.WithTimeout(context.Background(), timeout)
		defer longCancel()

		thisIsoId := cirrina.ISOID{Value: isoId}

		req := &cirrina.ISOImageRequest{
			Data: &cirrina.ISOImageRequest_Isouploadinfo{
				Isouploadinfo: &cirrina.ISOUploadInfo{
					Isoid:     &thisIsoId,
					Size:      isoSize,
					Sha512Sum: isoChecksum,
				},
			},
		}

		var stream cirrina.VMInfo_UploadIsoClient
		stream, err = c.UploadIso(longCtx)
		if err != nil {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           errors.New(status.Convert(err).Message()),
			}
		}

		err = stream.Send(req)
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
			req := &cirrina.ISOImageRequest{
				Data: &cirrina.ISOImageRequest_Image{
					Image: buffer[:n],
				},
			}
			err = stream.Send(req)
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
