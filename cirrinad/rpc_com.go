package main

import (
	"cirrina/cirrina"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/vm"
	"errors"
	"golang.org/x/exp/slog"
	"io"
	"os"
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
	if vmInst.Config.Com1Log {
		com1Chan := make(chan byte)
		vmInst.Com1rchan = com1Chan
		defer func() { vmInst.Com1rchan = nil }()
	}

	vmInst.Com1lock.Lock()
	defer vmInst.Com1lock.Unlock()

	slog.Debug("Com1Interactive", "vm_id", vmid.Value)
	go func(vmInst *vm.VM, stream cirrina.VMInfo_Com1InteractiveServer) {
		b := make([]byte, 1)
		for {
			if vmInst.Com1 == nil {
				if vmInst.Config.Com1Log {
					<-vmInst.Com1rchan
					vmInst.Com1rchan = nil
				}
				return
			}
			if vmInst.Status != "RUNNING" {
				if vmInst.Config.Com1Log {
					<-vmInst.Com1rchan
					vmInst.Com1rchan = nil
				}
				return
			}

			// get byte from channel if logging, else read port directly
			if vmInst.Config.Com1Log {
				var b2 byte
				b2 = <-vmInst.Com1rchan
				b[0] = b2
				req := cirrina.ComDataResponse{
					ComOutBytes: b,
				}
				err = stream.Send(&req)
				if err != nil {
					// unreachable
					slog.Debug("Com1Interactive logged failure sending to com channel", "err", err)
					<-vmInst.Com1rchan
					vmInst.Com1rchan = nil
					return
				}
			} else {
				nb, err := vmInst.Com1.Read(b)
				if nb > 1 {
					slog.Error("Com1Interactive read more than 1 byte", "nb", nb)
				}
				if err == io.EOF && vmInst.Status != vm.RUNNING {
					slog.Debug("comLogger", "msg", "vm not running, exiting")
					return
				}
				if err != nil && err != io.EOF {
					slog.Error("Com1Interactive error reading com port", "err", err)
					return
				}
				if nb != 0 {
					req := cirrina.ComDataResponse{
						ComOutBytes: b,
					}
					err = stream.Send(&req)
					if err != nil {
						slog.Debug("Com1Interactive un-logged failure sending to com channel", "err", err)
						if vmInst.Config.Com1Log {
							<-vmInst.Com1rchan
							vmInst.Com1rchan = nil
						}
						return
					}
				}
			}
		}
	}(vmInst, stream)

	var vl *os.File
	if vmInst.Config.Com1Log {
		com1LogPath := config.Config.Disk.VM.Path.State + "/" + vmInst.Name + "/"
		com1LogFile := com1LogPath + "com1_in.log"
		err = vm.GetVmLogPath(com1LogPath)
		if err != nil {
			slog.Error("Com1Interactive", "err", err)
			return err
		}
		vl, err = os.OpenFile(com1LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			slog.Error("failed to open VM in log file", "err", err)
		}
		defer func(vl *os.File) {
			_ = vl.Close()
		}(vl)
	}

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
		_, err = vmInst.Com1.Write(inBytes)
		if err != nil {
			return err
		}
		if vmInst.Config.Com1Log {
			_, err = vl.Write(inBytes)
			if err != nil {
				return err
			}
		}
	}
}
