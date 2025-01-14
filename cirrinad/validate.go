package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/hashicorp/go-version"
	"golang.org/x/sys/unix"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/iso"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
	"cirrina/cirrinad/vmnic"
)

func checkSudoCmd(expectedExit int, expectedStdOut string, expectedStdErr string, cmdArgs ...string) error {
	var runCmdStrArgs []string

	// "-S" to ensure no password prompt on tty
	runCmdStrArgs = append(runCmdStrArgs, "-S")
	runCmdStrArgs = append(runCmdStrArgs, cmdArgs...)

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(config.Config.Sys.Sudo, runCmdStrArgs)
	// we don't check err here because we get err if returnCode != 0 which is sometimes what we expect, so instead
	// we check returnCode

	if returnCode != expectedExit {
		slog.Debug("exitCode mismatch running command",
			"command", cmdArgs, "out", stdOutBytes, "err", stdErrBytes, "returnCode", returnCode, "err", err)

		return errExitCodeMismatch
	}

	if !strings.HasPrefix(string(stdOutBytes), expectedStdOut) {
		slog.Debug("stdout prefix mismatch running command",
			"command", cmdArgs, "out", stdOutBytes, "err", stdErrBytes, "returnCode", returnCode, "err", err)

		return errSTDOUTMismatch
	}

	if !strings.HasPrefix(string(stdErrBytes), expectedStdErr) {
		slog.Debug("stderr prefix mismatch running command",
			"command", cmdArgs, "out", stdOutBytes, "err", stdErrBytes, "returnCode", returnCode, "err", err)

		return errSTDERRMismatch
	}

	return nil
}

// getTmpFileName returns the name of a tmp file that doesn't exist or maybe an error
func getTmpFileName() (string, error) {
	var tmpFileName string

	var err error

	tmpDir := os.Getenv("TMPDIR")
	if tmpDir == "" {
		tmpDir = "/tmp"
	}

	try := 0
nameLoop:
	for {
		var randomNumBig *big.Int
		randomNumBig, err = rand.Int(rand.Reader, big.NewInt(1000000000))
		randomNum := int(randomNumBig.Int64())
		if err != nil {
			return "", fmt.Errorf("couldn't find a tmp file %w", err)
		}
		tmpFileName = tmpDir + string(os.PathSeparator) + "cirrinad" + strconv.FormatInt(int64(randomNum), 10)
		_, err = os.Stat(tmpFileName)
		switch {
		case err == nil:
			try++
			if try < 10000 {
				continue
			}
			// couldn't find a file name that doesn't exist?
			return "", errUnableToMakeTmpFile
		case errors.Is(err, fs.ErrNotExist):
			break nameLoop
		default:
			return "", fmt.Errorf("couldn't find a tmp file %w", err)
		}
	}

	return tmpFileName, nil
}

func kmodLoaded(name string) bool {
	stdOutBytes, stdErrBytes, rc, err := util.RunCmd("/sbin/kldstat", []string{"-q", "-n", name})
	slog.Debug("kldstat -q -n", "name", name, "stdOutBytes", stdOutBytes, "stdErrBytes", stdErrBytes, "rc", rc, "err", err)

	return rc == 0 && err == nil
}

func kmodInited(name string) bool {
	stdOutBytes, stdErrBytes, rc, err := util.RunCmd("/sbin/kldstat", []string{"-q", "-m", name})
	slog.Debug("kldstat -q -m", "name", name, "stdOutBytes", stdOutBytes, "stdErrBytes", stdErrBytes, "rc", rc, "err", err)

	return rc == 0 && err == nil
}

func validateKmods() {
	slog.Debug("validating kernel modules")

	moduleList := []string{"vmm", "nmdm", "if_bridge", "if_epair", "ng_bridge", "ng_ether", "ng_pipe"}

	for _, module := range moduleList {
		loaded := kmodLoaded(module)
		if !loaded {
			slog.Error("module not loaded, please load all kernel modules", "module", module)
			os.Exit(1)
		}

		inited := kmodInited(module)
		if !inited {
			slog.Error("module failed to initialize, please fix", "module", module)
			os.Exit(1)
		}
	}
}

func validateVirt() {
	stdOutBytes, stdErrBytes, rc, err := util.RunCmd("/sbin/sysctl", []string{"-n", "hw.hv_vendor"})
	if string(stdErrBytes) != "" || rc != 0 || err != nil {
		slog.Error("error running command", "stdOutBytes", stdOutBytes, "stdErrBytes", stdErrBytes, "rc", rc, "err", err)
		os.Exit(1)
	}

	hvVendor := strings.TrimSpace(string(stdOutBytes))
	if hvVendor != "" {
		slog.Error("Refusing to run inside virtualized environment", "hvVendor", hvVendor)
		os.Exit(1)
	}
}

func validateJailed() {
	stdOutBytes, stdErrBytes, rc, err := util.RunCmd(
		"/sbin/sysctl", []string{"-n", "security.jail.jailed"})
	if string(stdErrBytes) != "" || rc != 0 || err != nil {
		slog.Error("error running command", "stdOutBytes", stdOutBytes, "stdErrBytes", stdErrBytes, "rc", rc, "err", err)
		os.Exit(1)
	}

	jailed := strings.TrimSpace(string(stdOutBytes))
	slog.Debug("validateJailed", "jailed", jailed)

	if jailed != "0" {
		slog.Error("Refusing to run inside jailed environment")
		os.Exit(1)
	}
}

type cmdCheckData struct {
	args           []string
	expectedExit   int
	expectedStdOut string
	expectedStdErr string
}

func validateSudoCommands() {
	var err error

	allCmdChecks := getSudoCommandsList()

	for _, cmdCheck := range allCmdChecks {
		err = checkSudoCmd(cmdCheck.expectedExit, cmdCheck.expectedStdOut, cmdCheck.expectedStdErr, cmdCheck.args...)
		if err != nil {
			slog.Error("error running cmd, check sudo config",
				"cmd", cmdCheck.args,
				"err", err.Error(),
			)
			os.Exit(1)
		}
	}
}

func getSudoCommandsList() []cmdCheckData {
	var err error

	var name string

	name, err = getTmpFileName()
	if err != nil {
		slog.Error("getTmpFilename failed", "err", err.Error())
		os.Exit(1)
	}

	allCmdChecks := []cmdCheckData{
		{
			args:         []string{"/sbin/ifconfig"},
			expectedExit: 0, expectedStdOut: "", expectedStdErr: "",
		},
		{
			args:         []string{"/sbin/zfs", "version"},
			expectedExit: 0, expectedStdOut: "", expectedStdErr: "",
		},
		{
			args:         []string{"/usr/bin/nice", "/bin/echo", "-n"},
			expectedExit: 0, expectedStdOut: "", expectedStdErr: "",
		},
		{
			args:         []string{"/usr/bin/protect", "/bin/echo", "-n"},
			expectedExit: 0, expectedStdOut: "", expectedStdErr: "",
		},
		{
			args:         []string{"/usr/bin/rctl"},
			expectedExit: 0, expectedStdOut: "", expectedStdErr: "",
		},
		{
			args:         []string{"/usr/sbin/bhyve", "-h"},
			expectedExit: 0, expectedStdOut: "", expectedStdErr: "",
		},
		{
			args:         []string{"/usr/sbin/bhyvectl"},
			expectedExit: 1, expectedStdOut: "", expectedStdErr: "Usage: bhyvectl",
		},
		{
			args:         []string{"/usr/sbin/ngctl", "help"},
			expectedExit: 0, expectedStdOut: "Available commands:", expectedStdErr: "",
		},
		{
			args:         []string{"/bin/pgrep", "-a", "-l", "-x", "init"},
			expectedExit: 0, expectedStdOut: "1 init", expectedStdErr: "",
		},
		{
			args:         []string{"/usr/sbin/chown"},
			expectedExit: 1, expectedStdOut: "", expectedStdErr: "usage: chown",
		},
		{
			args:         []string{"/usr/bin/truncate", "-c", "-s", "1", name},
			expectedExit: 0, expectedStdOut: "", expectedStdErr: "",
		},
	}

	return allCmdChecks
}

func validateArch() {
	runtimeArch := runtime.GOARCH
	switch runtimeArch {
	case "amd64":
		// Do nothing
	case "arm64":
		// Do nothing
	default:
		slog.Error("Unsupported Architecture")
		os.Exit(1)
	}
}

func validateOS() {
	runtimeOS := runtime.GOOS
	switch runtimeOS {
	case "freebsd":
		// Do nothing
	default:
		slog.Error("Unsupported OS")
		os.Exit(1)
	}
}

func validateOSVersion() {
	ovi, err := util.GetOsVersion()
	if err != nil {
		slog.Error("failed to get os version", "err", err)
		os.Exit(1)
	}

	ver133, err := version.NewVersion("13.3")
	if err != nil {
		slog.Error("failed to create a version for 13.3")
		os.Exit(1)
	}

	ver141, err := version.NewVersion("14.1")
	if err != nil {
		slog.Error("failed to create a version for 14.1")
		os.Exit(1)
	}

	slog.Debug("validate OS", "ovi", ovi)
	// Check for valid OS version, see https://www.freebsd.org/security/
	// as of commit, 13.3 and 14.1 are oldest supported versions
	if ovi.LessThan(ver133) && ovi.LessThan(ver141) {
		slog.Error("Unsupported OS version", "version", ovi)
		os.Exit(1)
	}
}

func validateSudoConfig() {
	sudoPath, err := filepath.Abs(config.Config.Sys.Sudo)
	if err != nil {
		slog.Error("failed to get absolute path to sudo")
		os.Exit(1)
	}

	sudoFileInfo, err := os.Stat(sudoPath)
	if err != nil {
		slog.Error("failed to stat sudo")
		os.Exit(1)
	}

	sudoIsDir := sudoFileInfo.IsDir()
	if sudoIsDir {
		slog.Error("sudo is a directory?")
		os.Exit(1)
	}

	sudoMode := sudoFileInfo.Mode()
	if !util.ModeIsExecOther(sudoMode) {
		slog.Error("sudo permissions not exec other", "sudoMode", sudoMode)
		os.Exit(1)
	}

	if !util.ModeIsSuid(sudoMode) {
		slog.Error("sudo permissions not suid", "sudoMode", sudoMode)
		os.Exit(1)
	}
}

func validateZpoolConf() {
	// it's valid for the zpool not to be configured if you want to only use file storage
	if config.Config.Disk.VM.Path.Zpool == "" {
		return
	}

	stdOutBytes, stdErrBytes, rc, err := util.RunCmd("/sbin/zfs", []string{"list", config.Config.Disk.VM.Path.Zpool})
	if string(stdErrBytes) != "" || rc != 0 || err != nil {
		slog.Error("zfs dataset not available, please fix or reconfigure",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"rc", rc,
			"err", err,
		)
		os.Exit(1)
	}

	poolParts := strings.Split(config.Config.Disk.VM.Path.Zpool, "/")
	poolName := poolParts[0]

	checkZpoolCapacity(poolName)
}

func checkZpoolCapacity(poolName string) {
	var rawCapacity string

	stdOutBytes, stdErrBytes, rc, err := util.RunCmd("/sbin/zpool", []string{"list", "-H", poolName})
	if string(stdErrBytes) != "" || rc != 0 || err != nil {
		slog.Error("error checking zpool",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"rc", rc,
			"err", err,
		)
		os.Exit(1)
	}

	for _, line := range strings.Split(string(stdOutBytes), "\n") {
		textFields := strings.Fields(line)
		if len(textFields) != 11 {
			continue
		}

		if !strings.HasSuffix(textFields[7], "%") {
			continue
		}

		rawCapacity = strings.TrimSuffix(textFields[7], "%")
	}

	capacity, err := strconv.ParseInt(rawCapacity, 10, 64)
	if err != nil {
		slog.Error("error checking zpool", "err", err)
		os.Exit(1)
	}

	switch {
	case capacity > 99:
		slog.Error("zpool at critical usage, refusing to run", "capacity", capacity)
		os.Exit(1)
	case capacity > 95:
		slog.Warn("zpool at very high usage, be careful", "capacity", capacity)
	case capacity > 90:
		slog.Warn("zpool at high usage, be careful", "capacity", capacity)
	case capacity > 80:
		slog.Warn("zpool nearing high usage, consider reducing usage", "capacity", capacity)
	default:
		slog.Debug("zpool usage OK", "capacity", capacity)
	}
}

func validateNetworkConf() {
	if !util.IsValidIP(config.Config.Network.Grpc.IP) {
		slog.Error("Invalid listen IP in config, please reconfigure")
		os.Exit(1)
	}

	if config.Config.Network.Grpc.Timeout <= 0 {
		slog.Error("Invalid gRPC timeout, must be greater than 0")
		os.Exit(1)
	}

	// is MAC parseable?
	macTest := config.Config.Network.Mac.Oui + ":ff:ff:ff"

	_, err := vmnic.ParseMac(macTest)
	if err != nil {
		slog.Error("Invalid NIC MAC OUI in config, please reconfigure")
		os.Exit(1)
	}
}

func validateVncConfig() {
	if !util.IsValidIP(config.Config.Vnc.IP) {
		slog.Error("Invalid VNC IP in config, please reconfigure")
		os.Exit(1)
	}
}

func validateDebugConfig() {
	if !util.IsValidIP(config.Config.Debug.IP) {
		slog.Error("Invalid debug IP in config, please reconfigure")
		os.Exit(1)
	}
}

func validateRomConfig() {
	romPath, err := filepath.Abs(config.Config.Rom.Path)
	if err != nil {
		slog.Error("failed to get absolute path to rom file")
		os.Exit(1)
	}

	romFileInfo, err := os.Stat(romPath)
	if err != nil {
		slog.Error("rom not installed or path invalid, please install edk2-bhyve (sysutils/edk2) or reconfigure")
		os.Exit(1)
	}

	romIsDir := romFileInfo.IsDir()
	if romIsDir {
		slog.Error("rom config points to directory, please reconfigure")
		os.Exit(1)
	}

	varTemplatePath, err := filepath.Abs(config.Config.Rom.Vars.Template)
	if err != nil {
		slog.Error("failed to get absolute path to rom vars template file")
		os.Exit(1)
	}

	varTemplateFileInfo, err := os.Stat(varTemplatePath)
	if err != nil {
		slog.Error("rom vars template not installed or path invalid, " +
			"please install edk2-bhyve (sysutils/edk2) or reconfigure")
		os.Exit(1)
	}

	varTemplateFileIsDir := varTemplateFileInfo.IsDir()
	if varTemplateFileIsDir {
		slog.Error("rom vars template config points to directory, please reconfigure")
		os.Exit(1)
	}
}

func validateLogConfig() {
	var checkLogPathDirPerms bool

	logFilePath, err := filepath.Abs(config.Config.Log.Path)
	if err != nil {
		slog.Error("failed to get absolute path to log")
		os.Exit(1)
	}

	logFilePathInfo, err := os.Stat(logFilePath)
	if err != nil {
		// if the file doesn't exist, that's OK, we can create it if we have permission
		checkLogPathDirPerms = true
	}

	if !checkLogPathDirPerms {
		if logFilePathInfo.IsDir() {
			slog.Error("log path is a directory, please reconfigure to point to a file", "logFilePath", logFilePath)
			os.Exit(1)
		}

		logFileStat, ok := logFilePathInfo.Sys().(*syscall.Stat_t)
		if !ok {
			slog.Error("type failure", "logFilePathInfo", logFilePathInfo, "logFileStat", logFileStat)
			os.Exit(1)
		}

		if logFileStat == nil {
			slog.Error("failed getting log file sys info")
			os.Exit(1)
		}

		return
	}

	logDir := filepath.Dir(config.Config.Log.Path)
	if unix.Access(logDir, unix.W_OK) != nil {
		errM := fmt.Sprintf("log dir %s not writable", logDir)
		slog.Error(errM)
		os.Exit(1)
	}
}

func validatePidFilePathConfig() {
	pidFilePath, err := filepath.Abs(config.Config.Sys.PidFilePath)
	if err != nil {
		slog.Error("failed to get absolute path to pid file")
		os.Exit(1)
	}

	pidDir := filepath.Dir(config.Config.Sys.PidFilePath)
	if pidFilePath == pidDir {
		slog.Error("pid file path is a directory, please reconfigure to point to a file", "pidFilePath", pidFilePath)
		os.Exit(1)
	}

	if unix.Access(pidDir, unix.W_OK) != nil {
		errM := fmt.Sprintf("pid dir %s not writable", pidDir)
		slog.Error(errM)
		os.Exit(1)
	}
}

func validateDefaultDiskSize() {
	_, err := util.ParseDiskSize(config.Config.Disk.Default.Size)
	if err != nil {
		slog.Error("default disk size invalid")
		os.Exit(1)
	}
}

func validateStatePath() {
	if config.Config.Disk.VM.Path.State == "" {
		slog.Error("disk.vm.path.state not set, please reconfigure")
		os.Exit(1)
	}

	diskStatePath, err := filepath.Abs(config.Config.Disk.VM.Path.State)
	if err != nil {
		slog.Error("failed parsing disk vm path state, please reconfigure")
		os.Exit(1)
	}

	if unix.Access(diskStatePath, unix.W_OK) != nil {
		errM := fmt.Sprintf("disk state dir %s not writable", diskStatePath)
		slog.Error(errM)
		os.Exit(1)
	}
}

func validateIsoPath() {
	if config.Config.Disk.VM.Path.Iso == "" {
		slog.Error("disk.vm.path.iso not set, please reconfigure")
		os.Exit(1)
	}

	diskIsoPath, err := filepath.Abs(config.Config.Disk.VM.Path.Iso)
	if err != nil {
		slog.Error("failed parsing disk vm path iso, please reconfigure")
		os.Exit(1)
	}

	if unix.Access(diskIsoPath, unix.W_OK) != nil {
		errM := fmt.Sprintf("iso dir %s not writable", diskIsoPath)
		slog.Error(errM)
		os.Exit(1)
	}
}

func validateDiskFilePath() {
	if config.Config.Disk.VM.Path.Image == "" {
		slog.Error("disk.vm.path.image not set, please reconfigure")
		os.Exit(1)
	}

	diskImagePath, err := filepath.Abs(config.Config.Disk.VM.Path.Image)
	if err != nil {
		slog.Error("failed parsing disk vm path image, please reconfigure")
		os.Exit(1)
	}

	if unix.Access(diskImagePath, unix.W_OK) != nil {
		errM := fmt.Sprintf("disk image dir %s not writable", diskImagePath)
		slog.Error(errM)
		os.Exit(1)
	}
}

func validateConfig() {
	// main.doDBMigrations func validates db config via util.ValidateDBConfig()
	validateSudoConfig()
	validateVncConfig()
	validateDebugConfig()
	validateRomConfig()
	validateDefaultDiskSize()
	validateDiskFilePath()
	validateIsoPath()
	validateStatePath()
	validateZpoolConf()
	// validateLogConfig called early in main.rootCmd.RunE
	validateNetworkConf()
}

func validateSysctlSeeOtherGIDs() {
	stdOutBytes, stdErrBytes, rc, err := util.RunCmd("/sbin/sysctl", []string{"-n", "security.bsd.see_other_gids"})
	if string(stdErrBytes) != "" || rc != 0 || err != nil {
		slog.Error("Failed checking sysctl security.bsd.see_other_gids",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"rc", rc,
			"err", err,
		)
		os.Exit(1)
	}

	seeOtherGids := strings.TrimSpace(string(stdOutBytes))
	if seeOtherGids != "1" {
		slog.Error("Unable to run with other GIDs not visible", "security.bsd.see_other_gids", seeOtherGids)
		os.Exit(1)
	}
}

func validateSysctlSeeOtherUIDs() {
	stdOutBytes, stdErrBytes, rc, err := util.RunCmd("/sbin/sysctl", []string{"-n", "security.bsd.see_other_uids"})
	if string(stdErrBytes) != "" || rc != 0 || err != nil {
		slog.Error("Failed checking sysctl security.bsd.see_other_uids",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"rc", rc,
			"err", err,
		)
		os.Exit(1)
	}

	seeOtherUids := strings.TrimSpace(string(stdOutBytes))
	if seeOtherUids != "1" {
		slog.Error("Unable to run with other UIDs not visible", "security.bsd.see_other_uids", seeOtherUids)
		os.Exit(1)
	}
}

func validateSysctlSecureLevel() {
	stdOutBytes, stdErrBytes, rc, err := util.RunCmd("/sbin/sysctl", []string{"-n", "kern.securelevel"})
	if string(stdErrBytes) != "" || rc != 0 || err != nil {
		slog.Error("Failed checking sysctl kern.securelevel",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"rc", rc,
			"err", err,
		)
		os.Exit(1)
	}

	secureLevelStr := strings.TrimSpace(string(stdOutBytes))

	secureLevel, err := strconv.ParseInt(secureLevelStr, 10, 8)
	if err != nil {
		slog.Error("failed parsing secure level", "secureLevelStr", secureLevelStr)
	}

	if secureLevel > 0 {
		slog.Error("Unable to run with kern.securelevel > 0", "kern.securelevel", secureLevel)
		os.Exit(1)
	}
}

func validateMyID() {
	myUID, myGID, err := util.GetMyUIDGID()
	if err != nil {
		slog.Error("failed getting my uid/gid")
		os.Exit(1)
	}

	if myUID == 0 || myGID == 0 {
		slog.Error("refusing to run as root/wheel user/group")
		os.Exit(1)
	}
}

// validateDB validate db contents are sane: assume DB is correct, but maybe system state has changed
// called after migrations
func validateDB() {
	disk.CheckAll()
	iso.CheckAll()
	_switch.CheckAll()
	vmnic.CheckAll()
	vm.CheckAll()
}

// TODO check that users home dir is /nonexistent and that their login shell is /sbin/nologin
func validateSystem() {
	slog.Debug("validating system")
	validateArch()
	validateOS()
	validateOSVersion()
	validateKmods()
	validateVirt()
	validateJailed()
	validateMyID()
	validateSudoCommands()
	validateSysctlSeeOtherGIDs()
	validateSysctlSeeOtherUIDs()
	validateSysctlSecureLevel()
	validateConfig()
}
