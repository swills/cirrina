package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/tarm/serial"

	"cirrina/cirrina"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/vm"
)

func (s *server) Com1Interactive(stream cirrina.VMInfo_Com1InteractiveServer) error {
	in, err := stream.Recv()
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error receiving from com stream: %w", err)
	}
	vmid := in.GetVmId()
	vmuuid, err := uuid.Parse(vmid.Value)
	if err != nil {
		return errInvalidID
	}
	vmInst, err := vm.GetByID(vmuuid.String())
	if err != nil {
		return fmt.Errorf("error getting VM ID: %w", err)
	}

	if vmInst.Status != "RUNNING" {
		return errInvalidVMStateStop
	}

	return comInteractive(stream, vmInst, 1)
}

func (s *server) Com2Interactive(stream cirrina.VMInfo_Com2InteractiveServer) error {
	in, err := stream.Recv()
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error receiving from com stream: %w", err)
	}
	vmid := in.GetVmId()
	vmuuid, err := uuid.Parse(vmid.Value)
	if err != nil {
		return errInvalidID
	}
	vmInst, err := vm.GetByID(vmuuid.String())
	if err != nil {
		return fmt.Errorf("error getting VM ID: %w", err)
	}

	if vmInst.Status != "RUNNING" {
		return errInvalidVMStateStop
	}

	return comInteractive(stream, vmInst, 2)
}

func (s *server) Com3Interactive(stream cirrina.VMInfo_Com3InteractiveServer) error {
	in, err := stream.Recv()
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error receiving from com stream: %w", err)
	}
	vmid := in.GetVmId()
	vmuuid, err := uuid.Parse(vmid.Value)
	if err != nil {
		return errInvalidID
	}
	vmInst, err := vm.GetByID(vmuuid.String())
	if err != nil {
		return fmt.Errorf("error getting VM ID: %w", err)
	}

	if vmInst.Status != "RUNNING" {
		return errInvalidVMStateStop
	}

	return comInteractive(stream, vmInst, 3)
}

func (s *server) Com4Interactive(stream cirrina.VMInfo_Com4InteractiveServer) error {
	in, err := stream.Recv()
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error receiving from com stream: %w", err)
	}
	vmid := in.GetVmId()
	vmuuid, err := uuid.Parse(vmid.Value)
	if err != nil {
		return errInvalidID
	}
	vmInst, err := vm.GetByID(vmuuid.String())
	if err != nil {
		return fmt.Errorf("error getting VM ID: %w", err)
	}

	if vmInst.Status != "RUNNING" {
		return errInvalidVMStateStop
	}

	return comInteractive(stream, vmInst, 4)
}

// FIXME -- cheating a bit here
func comInteractive(stream cirrina.VMInfo_Com1InteractiveServer, vmInst *vm.VM, comNum int) error {
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

		return errComInvalid
	}
	err := comInteractiveSetup(thisCom)
	if err != nil {
		return err
	}

	go comInteractiveStreamSend(stream, vmInst, thisCom, thisComLog, thisRChan)

	var vl *os.File
	if thisComLog {
		comLogPath := config.Config.Disk.VM.Path.State + "/" + vmInst.Name + "/"
		comLogFile := comLogPath + "com" + strconv.Itoa(comNum) + "_in.log"
		err := vm.GetVMLogPath(comLogPath)
		if err != nil {
			slog.Error("ComInteractive", "err", err)

			return fmt.Errorf("error getting com log file path: %w", err)
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
		done, err2 := comInteractiveStreamReceive(stream, vmInst, thisCom, thisComLog, vl)
		if done {
			return err2
		}
	}
}

// comInteractiveSetup a few minor com setup things
func comInteractiveSetup(thisCom *serial.Port) error {
	if thisCom == nil {
		slog.Error("tried to start com but serial port is nil")

		return errComDevNotSet
	}
	// discard any existing input/output
	// Flush() doesn't seem to flush everything?
	for {
		b := make([]byte, 1)
		nb, err := thisCom.Read(b)
		if nb == 0 {
			break
		}
		if err != nil {
			slog.Error("error setting up com interactive", "err", err)

			return fmt.Errorf("error reading com: %w", err)
		}
	}

	return nil
}

// comInteractiveStreamReceive user -> com and/or log
func comInteractiveStreamReceive(stream cirrina.VMInfo_Com1InteractiveServer, vmInst *vm.VM,
	thisCom *serial.Port, thisComLog bool, vl *os.File) (bool, error) {
	if !vmInst.Running() {
		return true, nil
	}
	in, err := stream.Recv()
	if errors.Is(err, io.EOF) {
		return true, nil
	}
	if err != nil {
		return true, fmt.Errorf("error receiving from com stream: %w", err)
	}
	inBytes := in.GetComInBytes()
	_, err = thisCom.Write(inBytes)
	if err != nil {
		return true, fmt.Errorf("error writing to com: %w", err)
	}
	if thisComLog {
		_, err = vl.Write(inBytes)
		if err != nil {
			return true, fmt.Errorf("error writing to com log: %w", err)
		}
	}

	return false, nil
}

// comInteractiveStreamSend com -> user and/or log
func comInteractiveStreamSend(stream cirrina.VMInfo_Com1InteractiveServer, vmInst *vm.VM, thisCom *serial.Port,
	thisComLog bool, thisRChan chan byte) {
	b := make([]byte, 1)
	for {
		if thisCom == nil || !vmInst.Running() {
			return
		}

		// get byte from channel if logging, else read port directly
		if thisComLog {
			if comIntStreamSendFromLog(stream, thisRChan, b) {
				return
			}
		} else {
			if comIntStreamSendFromDev(stream, vmInst, thisCom, b) {
				return
			}
		}
	}
}

func comIntStreamSendFromDev(stream cirrina.VMInfo_Com1InteractiveServer, vmInst *vm.VM, thisCom *serial.Port,
	b []byte) bool {
	nb, err := thisCom.Read(b)
	if nb > 1 {
		slog.Error("ComInteractive read more than 1 byte", "nb", nb)
	}
	if errors.Is(err, io.EOF) && !vmInst.Running() {
		slog.Debug("ComInteractive", "msg", "vm not running, exiting")

		return true
	}
	if err != nil && !errors.Is(err, io.EOF) {
		slog.Error("ComInteractive error reading com port", "err", err)

		return true
	}
	if nb != 0 {
		req := cirrina.ComDataResponse{
			ComOutBytes: b,
		}
		err = stream.Send(&req)
		if err != nil {
			// slog.Debug("ComInteractive un-logged failure sending to com channel", "err", err)
			return true
		}
	}

	return false
}

func comIntStreamSendFromLog(stream cirrina.VMInfo_Com1InteractiveServer, thisRChan chan byte, b []byte) bool {
	var b2 = <-thisRChan
	b[0] = b2
	req := cirrina.ComDataResponse{
		ComOutBytes: b,
	}
	err := stream.Send(&req)
	if err != nil {
		// unreachable
		slog.Debug("ComInteractive logged failure sending to com channel", "err", err)

		return true
	}

	return false
}
