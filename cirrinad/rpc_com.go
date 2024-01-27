package main

import (
	"cirrina/cirrina"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/tarm/serial"
	"io"
	"log/slog"
	"os"
	"strconv"
)

func (s *server) Com1Interactive(stream cirrina.VMInfo_Com1InteractiveServer) error {
	util.Trace()
	in, err := stream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	vmid := in.GetVmId()
	vmuuid, err := uuid.Parse(vmid.Value)
	if err != nil {
		errorMessage := fmt.Sprintf("invalid vm id %s", vmid)
		return errors.New(errorMessage)
	}
	vmInst, err := vm.GetById(vmuuid.String())
	if err != nil {
		return err
	}

	if vmInst.Status != "RUNNING" {
		return errors.New("vm not running")
	}

	return comInteractive(stream, vmInst, 1)
}

func (s *server) Com2Interactive(stream cirrina.VMInfo_Com2InteractiveServer) error {
	util.Trace()
	in, err := stream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	vmid := in.GetVmId()
	vmuuid, err := uuid.Parse(vmid.Value)
	if err != nil {
		errorMessage := fmt.Sprintf("invalid vm id %s", vmid)
		return errors.New(errorMessage)
	}
	vmInst, err := vm.GetById(vmuuid.String())
	if err != nil {
		return err
	}

	if vmInst.Status != "RUNNING" {
		return errors.New("vm not running")
	}

	return comInteractive(stream, vmInst, 2)
}

func (s *server) Com3Interactive(stream cirrina.VMInfo_Com3InteractiveServer) error {
	util.Trace()
	in, err := stream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	vmid := in.GetVmId()
	vmuuid, err := uuid.Parse(vmid.Value)
	if err != nil {
		errorMessage := fmt.Sprintf("invalid vm id %s", vmid)
		return errors.New(errorMessage)
	}
	vmInst, err := vm.GetById(vmuuid.String())
	if err != nil {
		return err
	}

	if vmInst.Status != "RUNNING" {
		return errors.New("vm not running")
	}

	return comInteractive(stream, vmInst, 3)
}

func (s *server) Com4Interactive(stream cirrina.VMInfo_Com4InteractiveServer) error {
	util.Trace()
	in, err := stream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	vmid := in.GetVmId()
	vmuuid, err := uuid.Parse(vmid.Value)
	if err != nil {
		errorMessage := fmt.Sprintf("invalid vm id %s", vmid)
		return errors.New(errorMessage)
	}
	vmInst, err := vm.GetById(vmuuid.String())
	if err != nil {
		return err
	}

	if vmInst.Status != "RUNNING" {
		return errors.New("vm not running")
	}

	return comInteractive(stream, vmInst, 4)
}

// FIXME -- cheating a bit here
func comInteractive(stream cirrina.VMInfo_Com1InteractiveServer, vmInst *vm.VM, comNum int) error {
	util.Trace()

	var thisCom *serial.Port
	var thisComLog bool
	var thisRChan chan byte

	slog.Debug("comInteractive starting", "comNum", comNum)

	switch comNum {
	case 1:
		vmInst.Com1lock.Lock()
		defer vmInst.Com1lock.Unlock()
		thisCom = vmInst.Com1
		thisComLog = vmInst.Config.Com1Log
		if vmInst.Config.Com1Log {
			thisRChan = vmInst.Com1rchan
		}
		vmInst.Com1write = true
		defer func() {
			vmInst.Com1write = false
		}()
	case 2:
		vmInst.Com2lock.Lock()
		defer vmInst.Com2lock.Unlock()
		thisCom = vmInst.Com2
		thisComLog = vmInst.Config.Com2Log
		if vmInst.Config.Com2Log {
			thisRChan = vmInst.Com2rchan
		}
		vmInst.Com2write = true
		defer func() {
			vmInst.Com2write = false
		}()
	case 3:
		vmInst.Com3lock.Lock()
		defer vmInst.Com3lock.Unlock()
		thisCom = vmInst.Com3
		thisComLog = vmInst.Config.Com3Log
		if vmInst.Config.Com3Log {
			thisRChan = vmInst.Com3rchan
		}
		vmInst.Com3write = true
		defer func() {
			vmInst.Com3write = false
		}()
	case 4:
		vmInst.Com4lock.Lock()
		defer vmInst.Com4lock.Unlock()
		thisCom = vmInst.Com4
		thisComLog = vmInst.Config.Com4Log
		if vmInst.Config.Com4Log {
			thisRChan = vmInst.Com4rchan
		}
		vmInst.Com4write = true
		defer func() {
			vmInst.Com4write = false
		}()
	default:
		slog.Error("comLogger invalid com", "comNum", comNum)
		return errors.New("invalid comNum")
	}

	// discard any existing input/output
	// Flush() doesn't seem to flush everything?
	for {
		b := make([]byte, 1)
		if thisCom == nil {
			return nil
		}
		nb, err := thisCom.Read(b)
		if nb == 0 || err != nil {
			break
		}
	}

	if thisCom == nil {
		return errors.New("com not available")
	}

	slog.Debug("ComInteractive", "vm_id", vmInst.ID, "comNum", comNum)
	// FIXME -- cheating a bit here
	go func(vmInst *vm.VM, stream cirrina.VMInfo_Com1InteractiveServer) {
		b := make([]byte, 1)
		for {
			if thisCom == nil || vmInst.Status != "RUNNING" {
				return
			}

			// get byte from channel if logging, else read port directly
			if thisComLog {
				var b2 byte
				b2 = <-thisRChan
				b[0] = b2
				req := cirrina.ComDataResponse{
					ComOutBytes: b,
				}
				err := stream.Send(&req)
				if err != nil {
					// unreachable
					slog.Debug("ComInteractive logged failure sending to com channel", "err", err)
					return
				}
			} else {
				nb, err := thisCom.Read(b)
				if nb > 1 {
					slog.Error("ComInteractive read more than 1 byte", "nb", nb)
				}
				if err == io.EOF && vmInst.Status != vm.RUNNING {
					slog.Debug("ComInteractive", "msg", "vm not running, exiting")
					return
				}
				if err != nil && err != io.EOF {
					slog.Error("ComInteractive error reading com port", "err", err)
					return
				}
				if nb != 0 {
					req := cirrina.ComDataResponse{
						ComOutBytes: b,
					}
					err = stream.Send(&req)
					if err != nil {
						//slog.Debug("ComInteractive un-logged failure sending to com channel", "err", err)
						return
					}
				}
			}
		}
	}(vmInst, stream)

	var vl *os.File
	if thisComLog {
		comLogPath := config.Config.Disk.VM.Path.State + "/" + vmInst.Name + "/"
		comLogFile := comLogPath + "com" + strconv.Itoa(comNum) + "_in.log"
		err := vm.GetVmLogPath(comLogPath)
		if err != nil {
			slog.Error("ComInteractive", "err", err)
			return err
		}
		vl, err = os.OpenFile(comLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			slog.Error("failed to open VM input log file", "filename", comLogFile, "err", err)
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
		_, err = thisCom.Write(inBytes)
		if err != nil {
			return err
		}
		if thisComLog {
			_, err = vl.Write(inBytes)
			if err != nil {
				return err
			}
		}
	}
}
