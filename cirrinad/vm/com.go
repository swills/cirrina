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

func (v *VM) killComLoggers() {
	slog.Debug("killing com loggers")

	var err error

	// change to range when moving to Go 1.22
	for comNum := 1; comNum <= 4; comNum++ {
		err = v.killCom(comNum)
		if err != nil {
			// no need to return error here either
			slog.Error("com kill error", "comNum", 1, "err", err)
		}
	}
}

func (v *VM) setupComLoggers() {
	var err error

	// change to range when moving to Go 1.22
	for comNum := 1; comNum <= 4; comNum++ {
		err = v.setupCom(comNum)
		if err != nil {
			// not returning error since we leave the VM running and hope for the best
			slog.Error("com setup error", "comNum", comNum, "err", err)
		}
	}
}

func (v *VM) comLogger(comNum int) {
	var thisCom *serial.Port

	var thisRChan chan byte

	comChan := make(chan byte, 4096)

	switch comNum {
	case 1:
		thisCom = v.Com1

		if v.Config.Com1Log {
			v.Com1rchan = comChan
			thisRChan = v.Com1rchan
		}
	case 2:
		thisCom = v.Com2

		if v.Config.Com2Log {
			v.Com2rchan = comChan
			thisRChan = v.Com2rchan
		}
	case 3:
		thisCom = v.Com3

		if v.Config.Com3Log {
			v.Com3rchan = comChan
			thisRChan = v.Com3rchan
		}
	case 4:
		thisCom = v.Com4

		if v.Config.Com4Log {
			v.Com4rchan = comChan
			thisRChan = v.Com4rchan
		}
	default:
		slog.Error("comLogger invalid com", "comNum", comNum)

		return
	}

	logFile, err := v.comLoggerGetLogFile(comNum)
	if err != nil {
		slog.Error("error getting com log file", "err", err)

		return
	}

	defer func(vl *os.File) {
		_ = vl.Close()
	}(logFile)

	for {
		if v.comLoggerRead(comNum, thisCom, logFile, thisRChan) {
			return
		}
	}
}

func (v *VM) comLoggerGetLogFile(comNum int) (*os.File, error) {
	logFilePath := config.Config.Disk.VM.Path.State + "/" + v.Name + "/"
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

func (v *VM) GetComWrite(comNum int) bool {
	var thisComChanWriteFlag bool

	switch comNum {
	case 1:
		thisComChanWriteFlag = v.Com1write
	case 2:
		thisComChanWriteFlag = v.Com2write
	case 3:
		thisComChanWriteFlag = v.Com3write
	case 4:
		thisComChanWriteFlag = v.Com4write
	}

	return thisComChanWriteFlag
}

func (v *VM) comLoggerRead(comNum int, thisCom *serial.Port, logFile *os.File, thisRChan chan byte) bool {
	var thisComChanWriteFlag bool

	logBuffer := make([]byte, 1)
	streamBuffer := make([]byte, 1)

	if !v.Running() {
		slog.Debug("comLogger vm not running, exiting2",
			"vm_id", v.ID,
			"comNum", comNum,
			"vm.Status", v.Status,
		)

		return true
	}

	if thisCom == nil {
		slog.Error("comLogger", "msg", "unable to read nil port")

		return true
	}

	thisComChanWriteFlag = v.GetComWrite(comNum)

	nBytes, err := thisCom.Read(logBuffer)
	if nBytes > 1 {
		slog.Error("comLogger read more than 1 byte", "nBytes", nBytes)
	}

	if errors.Is(err, io.EOF) && !v.Running() {
		slog.Debug("comLogger vm not running, exiting",
			"vm_id", v.ID,
			"comNum", comNum,
			"vm.Status", v.Status,
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
		if v.Status != STOPPED && thisRChan != nil && thisComChanWriteFlag {
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

func (v *VM) killCom(comNum int) error {
	switch comNum {
	case 1:
		if v.Com1 != nil {
			_ = v.Com1.Close()
			v.Com1 = nil
		}

		if v.Com1rchan != nil {
			close(v.Com1rchan)
			v.Com1rchan = nil
		}
	case 2:
		if v.Com2 != nil {
			_ = v.Com2.Close()
			v.Com2 = nil
		}

		if v.Com2rchan != nil {
			close(v.Com2rchan)
			v.Com2rchan = nil
		}
	case 3:
		if v.Com3 != nil {
			_ = v.Com3.Close()
			v.Com3 = nil
		}

		if v.Com3rchan != nil {
			close(v.Com3rchan)
			v.Com3rchan = nil
		}
	case 4:
		if v.Com4 != nil {
			_ = v.Com4.Close()
			v.Com4 = nil
		}

		if v.Com4rchan != nil {
			close(v.Com4rchan)
			v.Com4rchan = nil
		}
	default:
		slog.Error("invalid com port number", "comNum", comNum)

		return errVMComInvalid
	}

	return nil
}

func (v *VM) setupCom(comNum int) error {
	var comConfig bool

	var comLog bool

	var comDev string

	var comSpeed uint32

	var err error

	comConfig, comLog, comDev, comSpeed, err = v.comSetupGetVars(comNum)
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
		go v.comLogger(comNum)
	}

	switch comNum {
	case 1:
		v.Com1 = serialPort
	case 2:
		v.Com2 = serialPort
	case 3:
		v.Com3 = serialPort
	case 4:
		v.Com4 = serialPort
	default:
		slog.Error("invalid com port number", "comNum", comNum)

		return errVMComInvalid
	}

	return nil
}

func (v *VM) comSetupGetVars(comNum int) (bool, bool, string, uint32, error) {
	var comConfig bool

	var comLog bool

	var comDev string

	var comSpeed uint32

	switch comNum {
	case 1:
		comConfig = v.Config.Com1
		comLog = v.Config.Com1Log
		comDev = v.Com1Dev
		comSpeed = v.Config.Com1Speed
	case 2:
		comConfig = v.Config.Com2
		comLog = v.Config.Com2Log
		comDev = v.Com2Dev
		comSpeed = v.Config.Com2Speed
	case 3:
		comConfig = v.Config.Com3
		comLog = v.Config.Com3Log
		comDev = v.Com3Dev
		comSpeed = v.Config.Com3Speed
	case 4:
		comConfig = v.Config.Com4
		comLog = v.Config.Com4Log
		comDev = v.Com4Dev
		comSpeed = v.Config.Com4Speed
	default:
		slog.Error("invalid com port number", "comNum", comNum)

		return false, false, "", 0, errVMComInvalid
	}

	return comConfig, comLog, comDev, comSpeed, nil
}
