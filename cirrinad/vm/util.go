package vm

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/tarm/serial"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vmnic"
)

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

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/chown", myUser.Username, comReadDev},
	)
	if err != nil {
		slog.Error("failed to fix ownership of comReadDev",
			"comReadDev", comReadDev,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("failed to fix ownership of: comReadDev: %s err: %w", comReadDev, err)
	}

	slog.Debug("ensureComDevReadable user mismatch fixed")

	return nil
}

func findChildPid(findPid uint32) uint32 {
	var childPid uint32

	slog.Debug("FindChildPid finding child proc")

	pidString := strconv.FormatUint(uint64(findPid), 10)

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/bin/pgrep", "-P", pidString},
	)
	if err != nil {
		slog.Error("FindChildPid error",
			"pidString", pidString,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}

	found := false

	for _, line := range strings.Split(string(stdOutBytes), "\n") {
		textFields := strings.Fields(line)

		fl := len(textFields)
		if fl <= 0 {
			continue
		}

		if fl > 1 {
			slog.Debug("FindChildPid pgrep extra fields", "line", line)
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

	slog.Debug("FindChildPid returning childPid", "childPid", childPid)

	return childPid
}

func findChildProcName(startPid uint32, procName string) uint32 {
	// dig around to get the child (bhyve) pid -- life would be so much easier if we could run bhyve as non-root
	// should fix supervisor to use int32
	slog.Debug("looking for process with name starting with pid", "procName", procName, "startPid", startPid)

	maxDepth := 4

	count := 0

	childPid := startPid

	foundProcName := findProcName(childPid)
	// might be "/usr/sbin/bhyve", might be "bhyve:"
	for !strings.Contains(foundProcName, procName) && count <= maxDepth {
		count++
		childPid = findChildPid(childPid)

		if childPid == 0 {
			return 0
		}

		foundProcName = findProcName(childPid)
	}
	slog.Debug("findChildProcName got process name", "foundProcName", foundProcName)

	return childPid
}

func findProcName(pid uint32) string {
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		"/bin/ps",
		[]string{"--libxo", "json", "-p", strconv.FormatInt(int64(pid), 10)},
	)
	if err != nil {
		slog.Error("failed to search for pid",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return ""
	}

	procName, err := parsePsJSONOutput(stdOutBytes)
	if err != nil {
		return ""
	}

	return procName
}

func getUsedVncPorts() []int {
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
		if vmInst.Status == STOPPED {
			continue
		}

		ret = append(ret, int(vmInst.VNCPort))
	}

	return ret
}

func getUsedDebugPorts() []int {
	var ret []int
	defer List.Mu.RUnlock()
	List.Mu.RLock()
	for _, vmInst := range List.VMList {
		if vmInst.Status == STOPPED {
			continue
		}

		ret = append(ret, int(vmInst.DebugPort))
	}

	return ret
}

func getUsedNetPorts() []string {
	var ret []string
	defer List.Mu.RUnlock()
	List.Mu.RLock()

	for _, vmInst := range List.VMList {
		if vmInst.Status == STOPPED {
			continue
		}

		vmNicsList, err := vmnic.GetNics(vmInst.Config.ID)
		if err != nil {
			slog.Error("getUsedNetPorts failed to get nics", "err", err)

			continue
		}

		for _, vmNic := range vmNicsList {
			ret = append(ret, vmNic.NetDev)
		}
	}

	return ret
}

// initOneVM initializes and adds a VM to the in memory cache of VMs
// note, callers must lock the in memory cache via List.Mu.Lock()
func initOneVM(vmInst *VM) {
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

func isNetPortUsed(netPort string) bool {
	usedNetPorts := getUsedNetPorts()
	for _, port := range usedNetPorts {
		if port == netPort {
			return true
		}
	}

	return false
}

func parsePsJSONOutput(psJSONOutput []byte) (string, error) {
	var result map[string]interface{}

	err := json.Unmarshal(psJSONOutput, &result)
	if err != nil {
		return "", fmt.Errorf("failed parsing netstat json output: %w", err)
	}

	procInfo, valid := result["process-information"].(map[string]interface{})
	if !valid {
		return "", errFailedParsing
	}

	processes, valid := procInfo["process"].([]interface{})

	if !valid {
		return "", errFailedParsing
	}

	if len(processes) != 1 {
		return "", errFailedParsing
	}

	thisProcInfo := processes[0]
	procCommand, valid := thisProcInfo.(map[string]interface{})

	if !valid {
		return "", errFailedParsing
	}

	procNameFullI, valid := procCommand["command"]
	if !valid {
		return "", errFailedParsing
	}

	procNameFull, valid := procNameFullI.(string)

	if !valid {
		return "", errFailedParsing
	}

	procArgList := strings.Split(procNameFull, " ")

	procName := procArgList[0]

	if len(procName) < 1 {
		return "", errFailedParsing
	}

	return procName, nil
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

func AutoStartVMs() {
	for _, vmInst := range List.VMList {
		if vmInst.Config.AutoStart {
			go doAutostart(vmInst)
		}
	}
}

func GetAll() []*VM {
	var allVMs []*VM
	for _, value := range List.VMList {
		allVMs = append(allVMs, value)
	}

	return allVMs
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
