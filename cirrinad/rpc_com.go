package main

import (
	"cirrina/cirrina"
	"cirrina/cirrinad/vm"
	"errors"
	"golang.org/x/exp/slog"
	"io"
)

func (s *server) Com1Interactive(stream cirrina.VMInfo_Com1InteractiveServer) error {

	in, err := stream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	vmid := in.GetVmId()
	vmInst, err := vm.GetById(vmid.Value)
	if err != nil {
		return err
	}

	if vmInst.Status != "RUNNING" {
		return errors.New("vm not running")
	}

	if vmInst.Com1 == nil {
		return errors.New("com not available")
	}

	vmInst.Com1lock.Lock()
	defer vmInst.Com1lock.Unlock()

	slog.Debug("Com1Interactive", "vm_id", vmid.Value)
	go func(vmInst *vm.VM, stream cirrina.VMInfo_Com1InteractiveServer) {
		for {
			if vmInst.Com1 == nil {
				return
			}
			if vmInst.Status != "RUNNING" {
				return
			}
			b := make([]byte, 1)
			_, err := vmInst.Com1.Read(b)
			if err == io.EOF {
				return
			}
			if err != nil {
				return
			}
			req := cirrina.ComDataResponse{
				ComOutBytes: b,
			}

			err = stream.Send(&req)
			if err != nil {
				return
			}
		}
	}(vmInst, stream)

	//slog.Debug("starting loop")
	for {

		if vmInst.Status != "RUNNING" {
			return nil
		}
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		inBytes := in.GetComInBytes()
		//slog.Debug("Com1Interactive", "in", inBytes)
		_, err = vmInst.Com1.Write(inBytes)
		if err != nil {
			return err
		}

	}
}
