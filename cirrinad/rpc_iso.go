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

	isoFile, err = osCreateFunc(isoInst.Path)
	if err != nil {
		slog.Error("Failed to open iso file", "err", err.Error())

		return fmt.Errorf("error creating iso file: %w", err)
	}

	err = receiveIsoFile(stream, isoUploadReq, isoInst, isoFile)
	if err != nil {
		slog.Error("error during upload", "err", err)

		err = stream.SendAndClose(&cirrina.ReqBool{Success: false})
		if err != nil {
			return fmt.Errorf("error during iso upload: %w", err)
		}

		return nil
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

	err = stream.SendAndClose(&cirrina.ReqBool{Success: true})
	if err != nil {
		return fmt.Errorf("error returning status: %w", err)
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

		if imageSize > isoUploadReq.GetSize() {
			return errIsoUploadSize
		}

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

	res := cirrina.ReqBool{}
	res.Success = false

	isoUUID, err = uuid.Parse(isoID.GetValue())
	if err != nil {
		return &res, errInvalidID
	}

	dIso, err := iso.GetByID(isoUUID.String())
	if err != nil {
		return &res, errIsoNotFound
	}

	// check that iso is not in use by a VM
	allVMs := vm.GetAll()
	for _, thisVM := range allVMs {
		slog.Debug("vm checks", "vm", thisVM)

		for _, vmISO := range thisVM.ISOs {
			if vmISO == nil {
				continue
			}

			if vmISO.ID == dIso.ID {
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

	err = dIso.Delete()
	if err != nil {
		slog.Error("error deleting iso", "err", err)

		return &res, errISOInternalDB
	}

	res.Success = true

	return &res, nil
}

func (s *server) GetISOVMs(isoID *cirrina.ISOID, stream cirrina.VMInfo_GetISOVMsServer) error {
	isoUUID, err := uuid.Parse(isoID.GetValue())
	if err != nil {
		return fmt.Errorf("error parsing ID: %w", err)
	}

	var isoInst *iso.ISO

	isoInst, err = iso.GetByID(isoUUID.String())
	if err != nil {
		slog.Error("GetISOVMs error getting ISO", "iso", isoUUID.String(), "err", err)

		return fmt.Errorf("error getting ISO: %w", err)
	}

	if isoInst.Name == "" {
		slog.Debug("iso not found")

		return errNotFound
	}

	vmIDs := isoInst.GetVMIDs()
	for _, vmID := range vmIDs {
		_, err = vm.GetByID(vmID)
		if err != nil {
			slog.Error("iso attached to non-existent VM, ignoring", "iso.ID", isoInst.ID, "err", err)

			continue
		}

		err = stream.Send(&cirrina.VMID{Value: vmID})
		if err != nil {
			slog.Error("error sending GetISOVMs response", "err", err)

			return fmt.Errorf("error sending GetISOVMs response: %w", err)
		}
	}

	return nil
}
