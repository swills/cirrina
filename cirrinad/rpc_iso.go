package main

import (
	"bufio"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/google/uuid"

	"cirrina/cirrina"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/iso"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
)

func (s *server) GetISOs(_ *cirrina.ISOsQuery, stream cirrina.VMInfo_GetISOsServer) error {
	var isos []*iso.ISO
	var ISOId cirrina.ISOID
	isos = iso.GetAll()
	for e := range isos {
		ISOId.Value = isos[e].ID
		err := stream.Send(&ISOId)
		if err != nil {
			return fmt.Errorf("error sending to stream: %w", err)
		}
	}

	return nil
}

func (s *server) GetISOInfo(_ context.Context, isoID *cirrina.ISOID) (*cirrina.ISOInfo, error) {
	var isoInfo cirrina.ISOInfo
	isoUUID, err := uuid.Parse(isoID.GetValue())
	if err != nil {
		return &isoInfo, errInvalidID
	}
	isoInst, err := iso.GetByID(isoUUID.String())
	if err != nil {
		slog.Error("error getting iso", "id", isoUUID.String(), "err", err)

		return &isoInfo, errNotFound
	}
	if isoInst.Name == "" {
		slog.Debug("iso not found")

		return &isoInfo, errNotFound
	}
	isoInfo.Name = &isoInst.Name
	isoInfo.Description = &isoInst.Description
	isoInfo.Size = &isoInst.Size

	return &isoInfo, nil
}

func (s *server) AddISO(_ context.Context, isoInfo *cirrina.ISOInfo) (*cirrina.ISOID, error) {
	defaultDescription := ""
	if isoInfo.Name == nil || !util.ValidIsoName(isoInfo.GetName()) {
		return &cirrina.ISOID{}, errInvalidName
	}
	if isoInfo.Description == nil {
		isoInfo.Description = &defaultDescription
	}
	isoInst, err := iso.Create(isoInfo.GetName(), isoInfo.GetDescription())
	if err != nil {
		return &cirrina.ISOID{}, fmt.Errorf("error creating iso: %w", err)
	}

	return &cirrina.ISOID{Value: isoInst.ID}, nil
}

func (s *server) UploadIso(stream cirrina.VMInfo_UploadIsoServer) error {
	var res cirrina.ReqBool
	res.Success = false

	req, err := stream.Recv()
	if err != nil {
		slog.Error("cannot receive image info")
	}
	isoUploadReq := req.GetIsouploadinfo()
	if isoUploadReq == nil || isoUploadReq.GetIsoid() == nil {
		slog.Error("nil isoUploadReq or iso id")

		return errIsoUploadNil
	}
	isoUUID, err := uuid.Parse(isoUploadReq.GetIsoid().GetValue())
	if err != nil {
		slog.Error("iso id not specified or invalid on upload")

		return errInvalidID
	}
	isoInst, err := iso.GetByID(isoUUID.String())
	if err != nil {
		slog.Error("error getting iso", "id", isoUUID.String(), "err", err)

		return errNotFound
	}
	if isoInst.Name == "" {
		slog.Debug("iso not found")

		return errNotFound
	}
	if isoInst.Path == "" {
		isoInst.Path = config.Config.Disk.VM.Path.Iso + string(os.PathSeparator) + isoInst.Name
	}
	isoInst.Size = isoUploadReq.GetSize()

	err = receiveIsoFile(stream, isoUploadReq, isoInst)
	if err != nil {
		slog.Error("error during upload", "err", err)
		err2 := stream.SendAndClose(&res)
		if err2 != nil {
			slog.Error("failed sending error response, ignoring", "err", err, "err2", err2)
		}

		return err
	}

	// save to db
	err = isoInst.Save()
	if err != nil {
		slog.Debug("UploadIso", "msg", "Failed saving to db")
		err = stream.SendAndClose(&res)
		if err != nil {
			slog.Error("UploadIso cannot send response", "err", err)
		}

		return nil
	}
	// we're done, return success to client
	res.Success = true
	err = stream.SendAndClose(&res)
	if err != nil {
		slog.Error("cannot send and close", "err", err)
	}

	return nil
}

func receiveIsoFile(stream cirrina.VMInfo_UploadIsoServer, isoUploadReq *cirrina.ISOUploadInfo,
	isoInst *iso.ISO,
) error {
	isoFile, err := os.Create(isoInst.Path)
	if err != nil {
		slog.Error("Failed to open iso file", "err", err.Error())

		return fmt.Errorf("error creating iso file: %w", err)
	}
	isoFileBuffer := bufio.NewWriter(isoFile)
	var imageSize uint64

	hasher := sha512.New()

	for {
		var req *cirrina.ISOImageRequest
		req, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			slog.Debug("UploadIso", "msg", "no more data")

			break
		}
		if err != nil {
			slog.Error("UploadIso failed receiving", "err", err)

			return fmt.Errorf("error receiving from stream: %w", err)
		}

		chunk := req.GetImage()
		size := len(chunk)

		imageSize += uint64(size)
		_, err = isoFileBuffer.Write(chunk)
		if err != nil {
			slog.Error("UploadIso failed writing", "err", err)

			return fmt.Errorf("error writing iso file: %w", err)
		}
		hasher.Write(chunk)
	}
	imageChecksum := hex.EncodeToString(hasher.Sum(nil))

	// flush buffer
	err = isoFileBuffer.Flush()
	if err != nil {
		slog.Error("error flushing iso file", "err", err)

		return fmt.Errorf("error flushing iso file: %w", err)
	}

	// verify size
	if imageSize != isoUploadReq.GetSize() {
		slog.Error("iso upload size incorrect",
			"imageSize", imageSize,
			"isoUPloadReq.Size", isoUploadReq.GetSize(),
		)

		return errIsoUploadSize
	}
	isoInst.Size = imageSize
	slog.Debug("UploadIso image size correct")

	// verify checksum
	if imageChecksum != isoUploadReq.GetSha512Sum() {
		slog.Error("iso upload checksum incorrect",
			"imageChecksum", imageChecksum,
			"isoUploadReq.Sha512Sum", isoUploadReq.GetSha512Sum(),
		)

		return errIsoUploadChecksum
	}
	isoInst.Checksum = imageChecksum

	// finish saving file
	err = isoFile.Close()
	if err != nil {
		slog.Error("error closing iso file", "err", err)

		return fmt.Errorf("error closing iso file: %w", err)
	}

	return nil
}

func (s *server) RemoveISO(_ context.Context, isoID *cirrina.ISOID) (*cirrina.ReqBool, error) {
	var err error
	var isoUUID uuid.UUID
	var isoInst *iso.ISO

	res := cirrina.ReqBool{}
	res.Success = false

	isoUUID, err = uuid.Parse(isoID.GetValue())
	if err != nil {
		return &res, errInvalidID
	}

	isoInst, err = iso.GetByID(isoUUID.String())
	if err != nil {
		slog.Error("error getting iso", "id", isoUUID.String(), "err", err)

		return &res, errNotFound
	}

	if isoInst.Name == "" {
		slog.Debug("iso not found")

		return &res, errNotFound
	}

	// check that iso is not in use by a VM
	allVMs := vm.GetAll()
	for _, thisVM := range allVMs {
		slog.Debug("vm checks", "vm", thisVM)
		var thisVMISOs []iso.ISO
		thisVMISOs, err = thisVM.GetISOs()
		if err != nil {
			return &res, fmt.Errorf("error getting VM ISOs: %w", err)
		}
		for _, vmISO := range thisVMISOs {
			if vmISO.ID == isoUUID.String() {
				slog.Error("RemoveISO",
					"msg", "tried to remove ISO in use by VM",
					"isoid", isoUUID.String(),
					"vm", thisVM.ID,
					"vmname", thisVM.Name,
				)

				return &res, errIsoInUse
			}
		}
	}

	err = iso.Delete(isoUUID.String())
	if err != nil {
		slog.Error("error deleting iso", "err", err)

		return &res, errISOInternalDB
	}

	// TODO dare we actually delete data from disk?

	res.Success = true

	return &res, nil
}
