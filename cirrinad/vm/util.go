package vm

import (
	"bufio"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
	"errors"
	"fmt"
	"github.com/tarm/serial"
	"golang.org/x/sys/execabs"
	"log/slog"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func GetVmLogPath(logpath string) error {
	ex, err := util.PathExists(logpath)
	if err != nil {
		return err
	}
	if !ex {
		err := os.MkdirAll(logpath, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func InitOneVm(vmInst *VM) {
	vmLogPath := config.Config.Disk.VM.Path.State + "/" + vmInst.Name
	err := GetVmLogPath(vmLogPath)
	if err != nil {
		panic(err)
	}
	vmLogFilePath := vmLogPath + "/log"
	vmLogFile, err := os.OpenFile(vmLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("failed to open VM log file", "err", err)
	}
	var programLevel = new(slog.LevelVar) // Info by default
	vmLogger := slog.New(slog.NewTextHandler(vmLogFile, &slog.HandlerOptions{Level: programLevel}))

	vmInst.log = *vmLogger

	switch strings.ToLower(config.Config.Log.Level) {
	case "debug":
		programLevel.Set(slog.LevelDebug)
	case "info":
		programLevel.Set(slog.LevelInfo)
	case "warn":
		programLevel.Set(slog.LevelWarn)
	case "error":
		programLevel.Set(slog.LevelError)
	default:
		programLevel.Set(slog.LevelInfo)
	}

	List.VmList[vmInst.ID] = vmInst
}

func AutoStartVMs() {
	for _, vmInst := range List.VmList {
		if vmInst.Config.AutoStart {
			go doAutostart(vmInst)
		}
	}
}

func doAutostart(vmInst *VM) {
	slog.Debug(
		"AutoStartVMs sleeping for auto start delay",
		"vm", vmInst.Name,
		"auto_start_delay", vmInst.Config.AutoStartDelay,
	)
	time.Sleep(time.Duration(vmInst.Config.AutoStartDelay) * time.Second)
	err := vmInst.Start()
	if err != nil {
		slog.Error("auto start failed", "vm", vmInst.ID, "name", vmInst.Name, "err", err)
	}
}

func GetAll() (allVMs []*VM) {
	for _, value := range List.VmList {
		allVMs = append(allVMs, value)
	}
	return allVMs
}

func GetByName(name string) (v *VM, err error) {
	defer List.Mu.RUnlock()
	List.Mu.RLock()
	for _, t := range List.VmList {
		if t.Name == name {
			return t, nil
		}
	}
	return &VM{}, errors.New("not found")
}

func GetById(Id string) (v *VM, err error) {
	defer List.Mu.RUnlock()
	List.Mu.RLock()
	vmInst, valid := List.VmList[Id]
	if valid {
		return vmInst, nil
	}
	return nil, errors.New("not found")
}

func LogAllVmStatus() {
	defer List.Mu.Unlock()
	List.Mu.Lock()
	for _, vmInst := range List.VmList {
		if vmInst.Status != RUNNING {
			slog.Info("vm",
				"id", vmInst.ID,
				"name", vmInst.Name,
				"cpus", vmInst.Config.Cpu,
				"state", vmInst.Status,
				"pid", nil,
			)
		} else {
			if vmInst.proc == nil {
				vmInst.SetStopped()
				vmInst.MaybeForceKillVM()
				slog.Info("vm",
					"id", vmInst.ID,
					"name", vmInst.Name,
					"cpus", vmInst.Config.Cpu,
					"state", vmInst.Status,
					"pid", nil,
				)
			} else {
				slog.Info("vm",
					"id", vmInst.ID,
					"name", vmInst.Name,
					"cpus", vmInst.Config.Cpu,
					"state", vmInst.Status,
					"pid", vmInst.proc.Pid(),
				)
			}
		}
	}
}

func GetRunningVMs() int {
	count := 0
	defer List.Mu.RUnlock()
	List.Mu.RLock()
	for _, vmInst := range List.VmList {
		if vmInst.Status != STOPPED {
			count += 1
		}
	}
	return count
}

func KillVMs() {
	defer List.Mu.RUnlock()
	List.Mu.RLock()
	for _, vmInst := range List.VmList {
		if vmInst.Status != STOPPED {
			go func(aVmInst *VM) {
				err := aVmInst.Stop()
				if err != nil {
					slog.Error("error stopping VM", "err", err)
				}
			}(vmInst)
		}
	}
}

func GetUsedVncPorts() []int {
	var ret []int
	defer List.Mu.RUnlock()
	List.Mu.RLock()
	for _, vmInst := range List.VmList {
		if vmInst.Status != STOPPED {
			ret = append(ret, int(vmInst.VNCPort))
		}
	}
	return ret
}

func GetUsedDebugPorts() []int {
	var ret []int
	defer List.Mu.RUnlock()
	List.Mu.RLock()
	for _, vmInst := range List.VmList {
		if vmInst.Status != STOPPED {
			ret = append(ret, int(vmInst.DebugPort))
		}
	}
	return ret
}

func GetUsedNetPorts() []string {
	var ret []string
	defer List.Mu.RUnlock()
	List.Mu.RLock()
	for _, vmInst := range List.VmList {
		vmNicsList, err := vmInst.GetNics()
		if err != nil {
			slog.Error("GetUsedNetPorts failed to get nics", "err", err)
		}
		for _, vmNic := range vmNicsList {
			ret = append(ret, vmNic.NetDev)
		}
	}
	return ret
}

func IsNetPortUsed(netPort string) bool {
	usedNetPorts := GetUsedNetPorts()
	for _, port := range usedNetPorts {
		if port == netPort {
			return true
		}
	}
	return false
}

func ensureComDevReadable(comDev string) error {
	if !strings.HasSuffix(comDev, "A") {
		slog.Error("error checking com dev readable: invalid com dev", "comDev", comDev)
		return errors.New("invalid com dev")
	}
	comBaseDev := comDev[:len(comDev)-1]
	comReadDev := comBaseDev + "B"
	slog.Debug("Checking com dev readable", "comDev", comDev, "comReadDev", comReadDev)
	exists, err := util.PathExists(comReadDev)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("comDev does not exists)")
	}
	comReadFileInfo, err := os.Stat(comReadDev)
	if err != nil {
		return err
	}
	if comReadFileInfo.IsDir() {
		return errors.New("error checking com dev readable: comReadDev is directory")
	}
	comReadStat := comReadFileInfo.Sys().(*syscall.Stat_t)
	if comReadStat == nil {
		return errors.New("failed converting comReadFileInfo to Stat_t")
	}
	myUid, _, err := util.GetMyUidGid()
	if err != nil {
		return errors.New("failed getting my uid")
	}
	if comReadStat.Uid == myUid {
		// everything is good, nothing to do
		return nil
	}
	slog.Debug("ensureComDevReadable uid mismatch, fixing", "uid", comReadStat.Uid, "myUid", myUid)
	myUser, err := user.Current()
	if err != nil {
		return err
	}
	args := []string{"/usr/sbin/chown", myUser.Username, comReadDev}
	cmd := execabs.Command(config.Config.Sys.Sudo, args...)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to fix ownership of comReadDev %s: %w", comReadDev, err)
	}
	slog.Debug("ensureComDevReadable user mismatch fixed")
	return nil
}

func findChildPid(findPid uint32) (childPid uint32) {
	slog.Debug("FindChildPid finding child proc")
	pidString := strconv.FormatUint(uint64(findPid), 10)
	args := []string{"/bin/pgrep", "-P", pidString}
	cmd := execabs.Command(config.Config.Sys.Sudo, args...)
	defer func(cmd *execabs.Cmd) {
		err := cmd.Wait()
		if err != nil {
			slog.Error("FindChildPid error", "err", err)
		}
	}(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("FindChildPid error", "err", err)
		return 0
	}
	if err := cmd.Start(); err != nil {
		slog.Error("FindChildPid error", "err", err)
		return 0
	}
	found := false
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		text := scanner.Text()
		textFields := strings.Fields(text)
		fl := len(textFields)
		if fl != 1 {
			slog.Debug("FindChildPid pgrep extra fields", "text", text)
		}
		tempPid1 := uint64(0)
		if !found {
			found = true
			tempPid1, err = strconv.ParseUint(textFields[0], 10, 32)
			if err != nil {
				slog.Error("FindChildPid error", "err", err)
				return 0
			}
			tempPid2 := uint32(tempPid1)
			childPid = tempPid2
		} else {
			slog.Debug("FindChildPid found too many child procs")
		}
	}
	if err := scanner.Err(); err != nil {
		slog.Error("FindChildPid error", "err", err)
	}
	slog.Debug("FindChildPid returning childPid", "childPid", childPid)
	return childPid
}

func startSerialPort(comDev string, comSpeed uint) (*serial.Port, error) {
	if strings.HasSuffix(comDev, "A") {
		comBaseDev := comDev[:len(comDev)-1]
		comReadDev := comBaseDev + "B"
		slog.Debug("startSerialPort starting serial port on com",
			"comReadDev", comReadDev,
			"comSpeed", comSpeed,
		)
		c := &serial.Config{
			Name:        comReadDev,
			Baud:        int(comSpeed),
			ReadTimeout: 500 * time.Millisecond,
		}
		comReader, err := serial.OpenPort(c)
		if err != nil {
			slog.Error("startSerialPort error opening comReadDev", "error", err)
			return nil, err
		}
		slog.Debug("startSerialLogger", "opened", comReadDev)
		return comReader, nil
	}
	return nil, errors.New("invalid com dev")
}
