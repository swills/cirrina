package main

import (
	"bufio"
	"cirrina/cirrina"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/iso"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/exp/slog"
	"io"
	"os"
)

func (s *server) GetISOs(_ *cirrina.ISOsQuery, stream cirrina.VMInfo_GetISOsServer) error {
	var isos []*iso.ISO
	var ISOId cirrina.ISOID
	isos = iso.GetAll()
	for e := range isos {
		ISOId.Value = isos[e].ID
		err := stream.Send(&ISOId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *server) GetISOInfo(_ context.Context, i *cirrina.ISOID) (*cirrina.ISOInfo, error) {
	var ic cirrina.ISOInfo
	isoUuid, err := uuid.Parse(i.Value)
	if err != nil {
		return &ic, errors.New("id not specified or invalid")
	}
	isoInst, err := iso.GetById(isoUuid.String())
	if err != nil {
		slog.Error("error getting iso", "id", isoUuid.String(), "err", err)
		return &ic, errors.New("not found")
	}
	if isoInst.Name == "" {
		slog.Debug("iso not found")
		return &ic, errors.New("not found")
	}
	ic.Name = &isoInst.Name
	ic.Description = &isoInst.Description
	ic.Size = &isoInst.Size
	return &ic, nil
}

func (s *server) AddISO(_ context.Context, i *cirrina.ISOInfo) (*cirrina.ISOID, error) {
	defaultDescription := ""
	if i.Name == nil || !util.ValidIsoName(*i.Name) {
		return &cirrina.ISOID{}, errors.New("invalid name")
	}
	if i.Description == nil {
		i.Description = &defaultDescription
	}
	isoInst, err := iso.Create(*i.Name, *i.Description)
	if err != nil {
		return &cirrina.ISOID{}, err
	}
	return &cirrina.ISOID{Value: isoInst.ID}, nil
}

func (s *server) UploadIso(stream cirrina.VMInfo_UploadIsoServer) error {
	var re cirrina.ReqBool
	re.Success = false
	var imageSize uint64
	imageSize = 0

	req, err := stream.Recv()
	if err != nil {
		slog.Error("UploadIso", "msg", "cannot receive image info")
	}
	isoUploadReq := req.GetISOUploadInfo()
	isoId := isoUploadReq.Isoid

	isoUuid, err := uuid.Parse(isoId.Value)
	if err != nil {
		return errors.New("id not specified or invalid")
	}
	isoInst, err := iso.GetById(isoUuid.String())
	if err != nil {
		slog.Error("error getting iso", "id", isoUuid.String(), "err", err)
		return errors.New("not found")
	}
	if isoInst.Name == "" {
		slog.Debug("iso not found")
		return errors.New("not found")
	}

	slog.Debug("UploadIso",
		"iso_id", isoId.Value,
		"iso_name", isoInst.Name,
		"size", isoUploadReq.Size, "checksum", isoUploadReq.Sha512Sum,
	)

	if isoInst.Path == "" {
		isoInst.Path = config.Config.Disk.VM.Path.Iso + string(os.PathSeparator) + isoInst.Name
	}

	err = isoInst.Save()
	if err != nil {
		slog.Error("UploadIso", "msg", "Failed saving to db")
		err = stream.SendAndClose(&re)
		if err != nil {
			slog.Error("UploadIso cannot send response", "err", err)
		}
		return nil
	}

	isoFile, err := os.Create(isoInst.Path)
	if err != nil {
		slog.Error("Failed to open iso file", "err", err.Error())
		return err
	}
	isoFileBuffer := bufio.NewWriter(isoFile)

	hasher := sha512.New()

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			slog.Debug("UploadIso", "msg", "no more data")
			break
		}
		if err != nil {
			slog.Error("UploadIso", "err", err)
			return errors.New("failed reading image date")
		}

		chunk := req.GetImage()
		size := len(chunk)
		slog.Debug("UploadIso got data", "size", size)

		imageSize += uint64(size)
		_, err = isoFileBuffer.Write(chunk)
		if err != nil {
			slog.Error("UploadIso", "err", err)
			return errors.New("failed writing image data")
		}
		hasher.Write(chunk)
	}
	// flush buffer
	isoFileBuffer.Flush()

	// verify size
	if imageSize != isoUploadReq.Size {
		slog.Error("UploadIso", "image upload size incorrect")
		err = stream.SendAndClose(&re)
		if err != nil {
			slog.Error("UploadIso cannot send response", "err", err)
		}
		return nil
	} else {
		isoInst.Size = imageSize
		slog.Debug("UploadIso", "msg", "image size correct")
	}

	// verify checksum
	isoInst.Checksum = hex.EncodeToString(hasher.Sum(nil))
	if isoInst.Checksum != isoUploadReq.Sha512Sum {
		slog.Debug("UploadIso", "image upload checksum incorrect")
		err = stream.SendAndClose(&re)
		if err != nil {
			slog.Error("UploadIso cannot send response", "err", err)
		}
		return nil
	}

	// finish saving file
	err = isoFile.Close()
	if err != nil {
		slog.Debug("UploadIso", "msg", "Failed writing iso", "err", err)
		err = stream.SendAndClose(&re)
		if err != nil {
			slog.Error("UploadIso cannot send response", "err", err)
		}
		return nil
	}

	// save to db
	err = isoInst.Save()
	if err != nil {
		slog.Debug("UploadIso", "msg", "Failed saving to db")
		err = stream.SendAndClose(&re)
		if err != nil {
			slog.Error("UploadIso cannot send response", "err", err)
		}
		return nil
	}

	// we're done, return success to client
	re.Success = true
	err = stream.SendAndClose(&re)
	if err != nil {
		slog.Error("UploadIso cannot send response", "err", err)
	}
	slog.Debug("UploadIso complete")
	return nil
}

func (s *server) RemoveISO(_ context.Context, i *cirrina.ISOID) (*cirrina.ReqBool, error) {
	re := cirrina.ReqBool{}
	re.Success = false

	isoUuid, err := uuid.Parse(i.Value)
	if err != nil {
		return &re, errors.New("id not specified or invalid")
	}
	isoInst, err := iso.GetById(isoUuid.String())
	if err != nil {
		slog.Error("error getting iso", "id", isoUuid.String(), "err", err)
		return &re, errors.New("not found")
	}
	if isoInst.Name == "" {
		slog.Debug("iso not found")
		return &re, errors.New("not found")
	}

	// check that iso is not in use by a VM
	allVMs := vm.GetAll()
	for _, thisVm := range allVMs {
		slog.Debug("vm checks", "vm", thisVm)
		thisVmISOs, err := thisVm.GetISOs()
		if err != nil {
			return &re, err
		}
		for _, vmISO := range thisVmISOs {
			if vmISO.ID == isoUuid.String() {
				slog.Error("RemoveISO",
					"msg", "tried to remove ISO in use by VM",
					"isoid", isoUuid.String(),
					"vm", thisVm.ID,
					"vmname", thisVm.Name,
				)
				errorText := fmt.Sprintf("ISO in use by VM %v (%v)", thisVm.ID, thisVm.Name)
				return &re, errors.New(errorText)
			}
		}
	}

	res := iso.Delete(isoUuid.String())
	if res != nil {
		slog.Error("error deleting iso", "res", res)
		errorText := fmt.Sprintf("error deleting iso: %v", err)
		return &re, errors.New(errorText)
	}

	// TODO dare we actually delete data from disk?

	re.Success = true
	return &re, nil
}
