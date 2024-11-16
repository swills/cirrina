package vm

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"

	"github.com/tarm/serial"

	"cirrina/cirrinad/config"
)

func (vm *VM) killComLoggers() {
	slog.Debug("killing com loggers")

	var err error

	// change to range when moving to Go 1.22
	for comNum := 1; comNum <= 4; comNum++ {
		err = vm.killCom(comNum)
		if err != nil {
			// no need to return error here either
			slog.Error("com kill error", "comNum", 1, "err", err)
		}
	}
}

func (vm *VM) setupComLoggers() {
	var err error

	// change to range when moving to Go 1.22
	for comNum := 1; comNum <= 4; comNum++ {
		err = vm.setupCom(comNum)
		if err != nil {
			// not returning error since we leave the VM running and hope for the best
			slog.Error("com setup error", "comNum", comNum, "err", err)
		}
	}
}

func comLogger(thisVM *VM, comNum int) {
	var thisCom *serial.Port

	var thisRChan chan byte

	comChan := make(chan byte, 4096)

	switch comNum {
	case 1:
		thisCom = thisVM.Com1

		if thisVM.Config.Com1Log {
			thisVM.Com1rchan = comChan
			thisRChan = thisVM.Com1rchan
		}
	case 2:
		thisCom = thisVM.Com2

		if thisVM.Config.Com2Log {
			thisVM.Com2rchan = comChan
			thisRChan = thisVM.Com2rchan
		}
	case 3:
		thisCom = thisVM.Com3

		if thisVM.Config.Com3Log {
			thisVM.Com3rchan = comChan
			thisRChan = thisVM.Com3rchan
		}
	case 4:
		thisCom = thisVM.Com4

		if thisVM.Config.Com4Log {
			thisVM.Com4rchan = comChan
			thisRChan = thisVM.Com4rchan
		}
	default:
		slog.Error("comLogger invalid com", "comNum", comNum)

		return
	}

	logFile, err := comLoggerGetLogFile(thisVM, comNum)
	if err != nil {
		slog.Error("error getting com log file", "err", err)

		return
	}

	defer func(vl *os.File) {
		_ = vl.Close()
	}(logFile)

	for {
		if comLoggerRead(thisVM, comNum, thisCom, logFile, thisRChan) {
			return
		}
	}
}

func comLoggerGetLogFile(thisVM *VM, comNum int) (*os.File, error) {
	logFilePath := config.Config.Disk.VM.Path.State + "/" + thisVM.Name + "/"
	logFileName := logFilePath + "com" + strconv.FormatInt(int64(comNum), 10) + "_out.log"

	err := GetVMLogPath(logFilePath)
	if err != nil {
		slog.Error("setupComLoggers", "err", err)

		return nil, fmt.Errorf("error getting com log file name: %w", err)
	}

	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Error("failed to open VM output log file", "filename", logFileName, "err", err)

		return nil, fmt.Errorf("error opening com log file name: %w", err)
	}

	return logFile, nil
}

func (vm *VM) GetComWrite(comNum int) bool {
	var thisComChanWriteFlag bool

	switch comNum {
	case 1:
		thisComChanWriteFlag = vm.Com1write
	case 2:
		thisComChanWriteFlag = vm.Com2write
	case 3:
		thisComChanWriteFlag = vm.Com3write
	case 4:
		thisComChanWriteFlag = vm.Com4write
	}

	return thisComChanWriteFlag
}

func comLoggerRead(thisVM *VM, comNum int, thisCom *serial.Port, logFile *os.File, thisRChan chan byte) bool {
	var thisComChanWriteFlag bool

	logBuffer := make([]byte, 1)
	streamBuffer := make([]byte, 1)

	if !thisVM.Running() {
		slog.Debug("comLogger vm not running, exiting2",
			"vm_id", thisVM.ID,
			"comNum", comNum,
			"vm.Status", thisVM.Status,
		)

		return true
	}

	if thisCom == nil {
		slog.Error("comLogger", "msg", "unable to read nil port")

		return true
	}

	thisComChanWriteFlag = thisVM.GetComWrite(comNum)

	nBytes, err := thisCom.Read(logBuffer)
	if nBytes > 1 {
		slog.Error("comLogger read more than 1 byte", "nBytes", nBytes)
	}

	if errors.Is(err, io.EOF) && !thisVM.Running() {
		slog.Debug("comLogger vm not running, exiting",
			"vm_id", thisVM.ID,
			"comNum", comNum,
			"vm.Status", thisVM.Status,
		)

		return true
	}

	if err != nil && !errors.Is(err, io.EOF) {
		slog.Error("comLogger", "error reading", err)

		return true
	}

	if nBytes != 0 {
		// write to log file
		_, err = logFile.Write(logBuffer)

		// write to channel used by remote users, if someone is reading from it
		if thisVM.Status != STOPPED && thisRChan != nil && thisComChanWriteFlag {
			nb2 := copy(streamBuffer, logBuffer)
			if nBytes != nb2 {
				slog.Error("comLogger", "msg", "some bytes lost")
			}
			thisRChan <- streamBuffer[0]
		}

		if err != nil {
			slog.Error("comLogger", "error writing", err)

			return true
		}
	}

	return false
}

func (vm *VM) killCom(comNum int) error {
	switch comNum {
	case 1:
		if vm.Com1 != nil {
			_ = vm.Com1.Close()
			vm.Com1 = nil
		}

		if vm.Com1rchan != nil {
			close(vm.Com1rchan)
			vm.Com1rchan = nil
		}
	case 2:
		if vm.Com2 != nil {
			_ = vm.Com2.Close()
			vm.Com2 = nil
		}

		if vm.Com2rchan != nil {
			close(vm.Com2rchan)
			vm.Com2rchan = nil
		}
	case 3:
		if vm.Com3 != nil {
			_ = vm.Com3.Close()
			vm.Com3 = nil
		}

		if vm.Com3rchan != nil {
			close(vm.Com3rchan)
			vm.Com3rchan = nil
		}
	case 4:
		if vm.Com4 != nil {
			_ = vm.Com4.Close()
			vm.Com4 = nil
		}

		if vm.Com4rchan != nil {
			close(vm.Com4rchan)
			vm.Com4rchan = nil
		}
	default:
		slog.Error("invalid com port number", "comNum", comNum)

		return errVMComInvalid
	}

	return nil
}

func (vm *VM) setupCom(comNum int) error {
	var comConfig bool

	var comLog bool

	var comDev string

	var comSpeed uint32

	var err error

	comConfig, comLog, comDev, comSpeed, err = comSetupGetVars(comNum, vm)
	if err != nil {
		return err
	}

	if !comConfig {
		slog.Debug("vm com not enabled, skipping setup", "comNum", comNum, "comConfig", comConfig)

		return nil
	}

	if comDev == "" {
		slog.Error("com port enabled but com dev not set", "comNum", comNum, "comConfig", comConfig)

		return errVMComDevNotSet
	}

	// attach serial port object to VM object
	slog.Debug("checking com is readable", "comDev", comDev)

	err = ensureComDevReadable(comDev)
	if err != nil {
		slog.Error("error checking com readable", "comNum", comNum, "err", err)

		return err
	}

	serialPort, err := startSerialPort(comDev, uint(comSpeed))
	if err != nil {
		slog.Error("error starting com", "comNum", comNum, "err", err)

		return err
	}

	// actually setup logging if required
	if comLog {
		go comLogger(vm, comNum)
	}

	switch comNum {
	case 1:
		vm.Com1 = serialPort
	case 2:
		vm.Com2 = serialPort
	case 3:
		vm.Com3 = serialPort
	case 4:
		vm.Com4 = serialPort
	default:
		slog.Error("invalid com port number", "comNum", comNum)

		return errVMComInvalid
	}

	return nil
}

func comSetupGetVars(comNum int, aVM *VM) (bool, bool, string, uint32, error) {
	var comConfig bool

	var comLog bool

	var comDev string

	var comSpeed uint32

	switch comNum {
	case 1:
		comConfig = aVM.Config.Com1
		comLog = aVM.Config.Com1Log
		comDev = aVM.Com1Dev
		comSpeed = aVM.Config.Com1Speed
	case 2:
		comConfig = aVM.Config.Com2
		comLog = aVM.Config.Com2Log
		comDev = aVM.Com2Dev
		comSpeed = aVM.Config.Com2Speed
	case 3:
		comConfig = aVM.Config.Com3
		comLog = aVM.Config.Com3Log
		comDev = aVM.Com3Dev
		comSpeed = aVM.Config.Com3Speed
	case 4:
		comConfig = aVM.Config.Com4
		comLog = aVM.Config.Com4Log
		comDev = aVM.Com4Dev
		comSpeed = aVM.Config.Com4Speed
	default:
		slog.Error("invalid com port number", "comNum", comNum)

		return false, false, "", 0, errVMComInvalid
	}

	return comConfig, comLog, comDev, comSpeed, nil
}
