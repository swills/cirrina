package main

import (
	"bytes"
	"cirrina/cirrina"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/iso"
	"cirrina/cirrinad/vm"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
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
	slog.Debug("GetISOInfo", "iso", i.Value)
	if i.Value == "" {
		return &ic, nil
	}
	isoInst, err := iso.GetById(i.Value)
	if err != nil {
		slog.Debug("error getting iso", "iso", i.Value, "err", err)
		return &ic, err
	}
	ic.Name = &isoInst.Name
	ic.Description = &isoInst.Description
	return &ic, nil
}

func (s *server) AddISO(_ context.Context, i *cirrina.ISOInfo) (*cirrina.ISOID, error) {
	isoInst, err := iso.Create(*i.Name, *i.Description)
	if err != nil {
		return &cirrina.ISOID{}, err
	}
	return &cirrina.ISOID{Value: isoInst.ID}, nil
}

func (s *server) UploadIso(stream cirrina.VMInfo_UploadIsoServer) error {
	var re cirrina.ReqBool
	re.Success = false

	imageData := bytes.Buffer{}
	var imageSize uint64
	imageSize = 0

	req, err := stream.Recv()
	if err != nil {
		slog.Error("UploadIso", "msg", "cannot receive image info")
	}
	isoUploadReq := req.GetIsouploadinfo()
	isoId := isoUploadReq.Isoid
	isoInst, err := iso.GetById(isoId.Value)
	if err != nil {
		slog.Debug("UploadIso", "err", err)
		return err
	}

	slog.Debug("UploadIso",
		"iso_id", isoId.Value,
		"iso_name", isoInst.Name,
		"size", isoUploadReq.Size, "checksum", isoUploadReq.Sha512Sum,
	)

	for {
		//slog.Debug("UploadIso", "msg", "waiting to receive more data")
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
		//slog.Debug("UploadIso", "msg", "got data")

		//data := req.GetData()
		//slog.Debug("uploadIos", "data", data)

		_, err = imageData.Write(chunk)
		if err != nil {
			slog.Error("UploadIso", "err", err)
			return errors.New("failed writing image data")
		}

	}

	if imageSize != isoUploadReq.Size {
		slog.Debug("UploadIso", "image upload size incorrect")
		err = stream.SendAndClose(&re)
		if err != nil {
			slog.Error("UploadIso cannot send response", "err", err)
		}
		return nil
	} else {
		isoInst.Size = imageSize
		slog.Debug("UploadIso", "msg", "image size correct")
	}

	hasher := sha512.New()

	hasher.Write(imageData.Bytes())

	isoChecksum := hex.EncodeToString(hasher.Sum(nil))

	if isoChecksum != isoUploadReq.Sha512Sum {
		slog.Debug("UploadIso", "image upload checksum incorrect")
		err = stream.SendAndClose(&re)
		if err != nil {
			slog.Error("UploadIso cannot send response", "err", err)
		}
		return nil
	} else {
		slog.Debug("UploadIso", "msg", "image checksum correct")
		isoInst.Checksum = isoChecksum
	}

	if isoInst.Name == "" {
		slog.Error("Name is empty")
	}

	if isoInst.Path == "" {
		isoInst.Path = config.Config.Disk.VM.Path.Iso + "/" + isoInst.Name
	}

	slog.Debug("UploadIso", "msg", "Saving contents")
	err = os.WriteFile(isoInst.Path, imageData.Bytes(), 0644)
	if err != nil {
		slog.Debug("UploadIso", "msg", "Failed writing iso", "err", err)
		err = stream.SendAndClose(&re)
		if err != nil {
			slog.Error("UploadIso cannot send response", "err", err)
		}
		return nil
	}

	err = isoInst.Save()
	if err != nil {
		slog.Debug("UploadIso", "msg", "Failed saving to db")
		err = stream.SendAndClose(&re)
		if err != nil {
			slog.Error("UploadIso cannot send response", "err", err)
		}
		return nil
	}

	// we're done!
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

	slog.Debug("GetISOInfo", "iso", i.Value)
	if i.Value == "" {
		return &re, errors.New("ISO id must be specified")
	}

	_, err := iso.GetById(i.Value)
	if err != nil {
		slog.Debug("error getting iso", "iso", i.Value, "err", err)
		return &re, err
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
			if vmISO.ID == i.Value {
				slog.Error("RemoveISO",
					"msg", "tried to remove ISO in use by VM",
					"isoid", i.Value,
					"vm", thisVm.ID,
					"vmname", thisVm.Name,
				)
				errorText := fmt.Sprintf("ISO in use by VM %v (%v)", thisVm.ID, thisVm.Name)
				return &re, errors.New(errorText)
			}
		}
	}

	res := iso.Delete(i.Value)
	if res != nil {
		slog.Error("error deleting iso", "res", res)
		errorText := fmt.Sprintf("error deleting iso: %v", err)
		return &re, errors.New(errorText)
	}

	// TODO dare we actually delete data from disk?

	re.Success = true
	return &re, nil
}
