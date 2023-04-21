package main

import (
	"cirrina/cirrina"
	"cirrina/cirrinad/iso"
	"context"
	"golang.org/x/exp/slog"
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
	//if _, err := iso.GetByName(*isoInfo.Name); err == nil {
	//	return &cirrina.ISOID{}, errors.New(fmt.Sprintf("%v already exists", v.Name))
	//
	//}
	//defer vm.List.Mu.Unlock()
	//vm.List.Mu.Lock()
	isoInst, err := iso.Create(*i.Name, *i.Description, *i.Path)
	if err != nil {
		return &cirrina.ISOID{}, err
	}
	//iso.List.VmList[vmInst.ID] = vmInst
	return &cirrina.ISOID{Value: isoInst.ID}, nil
}
