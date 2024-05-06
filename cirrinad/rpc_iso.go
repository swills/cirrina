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

	path := config.Config.Disk.VM.Path.Iso + "/" + isoInfo.GetName()
	isoInst := &iso.ISO{
		Name:        isoInfo.GetName(),
		Description: isoInfo.GetDescription(),
		Path:        path,
	}

	err := iso.Create(isoInst)
	if err != nil {
		return nil, fmt.Errorf("error creating iso: %w", err)
	}

	return &cirrina.ISOID{Value: isoInst.ID}, nil
}

func (s *server) UploadIso(stream cirrina.VMInfo_UploadIsoServer) error {
	var res cirrina.ReqBool
	res.Success = false

	isoUploadReq, isoInst, err := validateIsoUploadRequest(stream)
	if err != nil {
		return err
	}

	if isoInst.Path == "" {
		isoInst.Path = config.Config.Disk.VM.Path.Iso + string(os.PathSeparator) + isoInst.Name
	}

	isoInst.Size = isoUploadReq.GetSize()

	var isoFile *os.File

	isoFile, err = os.Create(isoInst.Path)
	if err != nil {
		slog.Error("Failed to open iso file", "err", err.Error())

		return fmt.Errorf("error creating iso file: %w", err)
	}

	err = receiveIsoFile(stream, isoUploadReq, isoInst, isoFile)
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

func validateIsoUploadRequest(stream cirrina.VMInfo_UploadIsoServer) (*cirrina.ISOUploadInfo, *iso.ISO, error) {
	var err error

	var req *cirrina.ISOImageRequest

	req, err = stream.Recv()
	if err != nil {
		slog.Error("cannot receive image info")

		return nil, nil, errIsoUploadNil
	}

	isoUploadReq := req.GetIsouploadinfo()
	if isoUploadReq == nil || isoUploadReq.GetIsoid() == nil {
		slog.Error("nil isoUploadReq or iso id")

		return nil, nil, errIsoUploadNil
	}

	isoUUID, err := uuid.Parse(isoUploadReq.GetIsoid().GetValue())
	if err != nil {
		slog.Error("iso id not specified or invalid on upload")

		return nil, nil, errInvalidID
	}

	isoInst, err := iso.GetByID(isoUUID.String())
	if err != nil {
		slog.Error("error getting iso", "id", isoUUID.String(), "err", err)

		return nil, nil, errNotFound
	}

	if isoInst.Name == "" {
		slog.Debug("iso not found")

		return nil, nil, errNotFound
	}

	return isoUploadReq, isoInst, nil
}

func receiveIsoFile(stream cirrina.VMInfo_UploadIsoServer, isoUploadReq *cirrina.ISOUploadInfo,
	isoInst *iso.ISO, isoFile *os.File,
) error {
	var err error

	var imageSize uint64

	isoFileBuffer := bufio.NewWriter(isoFile)

	hasher := sha512.New()

	for {
		var req *cirrina.ISOImageRequest

		req, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return fmt.Errorf("error receiving from stream: %w", err)
		}

		chunk := req.GetImage()
		imageSize += uint64(len(chunk))

		_, err = isoFileBuffer.Write(chunk)
		if err != nil {
			return fmt.Errorf("error writing iso file: %w", err)
		}

		hasher.Write(chunk)
	}

	imageChecksum := hex.EncodeToString(hasher.Sum(nil))

	// flush buffer
	err = isoFileBuffer.Flush()
	if err != nil {
		return fmt.Errorf("error flushing iso file: %w", err)
	}

	// verify size
	if imageSize != isoUploadReq.GetSize() {
		return errIsoUploadSize
	}

	isoInst.Size = imageSize

	// verify checksum
	if imageChecksum != isoUploadReq.GetSha512Sum() {
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
