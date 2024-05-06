package vm

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/tarm/serial"
	"golang.org/x/sys/execabs"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

func GetVMLogPath(logpath string) error {
	var err error

	var logPathExists bool

	logPathExists, err = util.PathExists(logpath)
	if err != nil {
		return fmt.Errorf("error getting VM log path: %w", err)
	}

	if !logPathExists {
		err = os.MkdirAll(logpath, 0o755)
		if err != nil {
			return fmt.Errorf("error getting VM log path: %w", err)
		}
	}

	return nil
}

func InitOneVM(vmInst *VM) {
	vmLogPath := config.Config.Disk.VM.Path.State + "/" + vmInst.Name

	err := GetVMLogPath(vmLogPath)
	if err != nil {
		panic(err)
	}

	vmLogFilePath := vmLogPath + "/log"

	vmLogFile, err := os.OpenFile(vmLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Error("failed to open VM log file", "err", err)
	}

	programLevel := new(slog.LevelVar) // Info by default
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

	List.VMList[vmInst.ID] = vmInst
}

func AutoStartVMs() {
	for _, vmInst := range List.VMList {
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

func GetAll() []*VM {
	var allVMs []*VM
	for _, value := range List.VMList {
		allVMs = append(allVMs, value)
	}

	return allVMs
}

func GetByName(name string) (*VM, error) {
	defer List.Mu.RUnlock()
	List.Mu.RLock()
	for _, t := range List.VMList {
		if t.Name == name {
			return t, nil
		}
	}

	return &VM{}, errVMNotFound
}

func GetByID(id string) (*VM, error) {
	defer List.Mu.RUnlock()
	List.Mu.RLock()

	vmInst, valid := List.VMList[id]
	if valid {
		return vmInst, nil
	}

	return nil, errVMNotFound
}

func LogAllVMStatus() {
	defer List.Mu.Unlock()
	List.Mu.Lock()
	for _, vmInst := range List.VMList {
		if vmInst.Status != RUNNING {
			slog.Info("vm",
				"id", vmInst.ID,
				"name", vmInst.Name,
				"cpus", vmInst.Config.CPU,
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
					"cpus", vmInst.Config.CPU,
					"state", vmInst.Status,
					"pid", nil,
				)
			} else {
				slog.Info("vm",
					"id", vmInst.ID,
					"name", vmInst.Name,
					"cpus", vmInst.Config.CPU,
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
	for _, vmInst := range List.VMList {
		if vmInst.Status != STOPPED {
			count++
		}
	}

	return count
}

func KillVMs() {
	defer List.Mu.RUnlock()
	List.Mu.RLock()
	for _, vmInst := range List.VMList {
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

	// TODO -- this was initially meant to avoid two VMs using the same VNC port
	// but want to allow that as long as they aren't running at the same time -- unfortunately, at this point,
	// where we are starting the VM, it's too late to check and return an error and fail the startup request
	// need to add a check in VM startup request processing to check that and return error
	//
	// right now, the behavior is to simply move to a different port
	for _, vmInst := range List.VMList {
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
	for _, vmInst := range List.VMList {
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
	for _, vmInst := range List.VMList {
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

		return errVMInvalidComDev
	}

	comBaseDev := comDev[:len(comDev)-1]
	comReadDev := comBaseDev + "B"
	slog.Debug("Checking com dev readable", "comDev", comDev, "comReadDev", comReadDev)

	exists, err := util.PathExists(comReadDev)
	if err != nil {
		return fmt.Errorf("error checking vm com dev: %w", err)
	}

	if !exists {
		return errVMComDevNonexistent
	}

	comReadFileInfo, err := os.Stat(comReadDev)
	if err != nil {
		return fmt.Errorf("error checking vm com dev: %w", err)
	}

	if comReadFileInfo.IsDir() {
		return errVMComDevIsDir
	}

	comReadStat, ok := comReadFileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		slog.Error("type failure", "comReadFileInfo", comReadFileInfo, "comReadDev", comReadDev)

		return errVMTypeFailure
	}

	if comReadStat == nil {
		return errVMTypeConversionFailure
	}

	myUID, _, err := util.GetMyUIDGID()
	if err != nil {
		return fmt.Errorf("failure getting my UID/GID: %w", err)
	}

	if comReadStat.Uid == myUID {
		// everything is good, nothing to do
		return nil
	}

	slog.Debug("ensureComDevReadable uid mismatch, fixing", "uid", comReadStat.Uid, "myUID", myUID)

	myUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("error checking vm com dev: %w", err)
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

func findChildPid(findPid uint32) uint32 {
	var childPid uint32

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

	if err = cmd.Start(); err != nil {
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

		var tempPid1 uint64

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
	if !strings.HasSuffix(comDev, "A") {
		return nil, errVMInvalidComDev
	}

	comBaseDev := comDev[:len(comDev)-1]
	comReadDev := comBaseDev + "B"
	slog.Debug("startSerialPort starting serial port on com",
		"comReadDev", comReadDev,
		"comSpeed", comSpeed,
	)

	serialConfig := &serial.Config{
		Name:        comReadDev,
		Baud:        int(comSpeed),
		ReadTimeout: 500 * time.Millisecond,
	}

	comReader, err := serial.OpenPort(serialConfig)
	if err != nil {
		slog.Error("startSerialPort error opening comReadDev", "error", err)

		return nil, fmt.Errorf("error starting com port: %w", err)
	}

	slog.Debug("startSerialLogger", "opened", comReadDev)

	return comReader, nil
}
