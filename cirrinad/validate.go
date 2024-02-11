package main

import (
	"bufio"
	"bytes"
	"cirrina/cirrinad/config"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"errors"
	"fmt"
	"github.com/hashicorp/go-version"
	"golang.org/x/sys/execabs"
	"golang.org/x/sys/unix"
	"io/fs"
	"log/slog"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func checkSudoCmd(expectedExit int, expectedStdOut string, expectedStdErr string, cmdArgs ...string) (err error) {
	var emptyBytes []byte
	var outBytes bytes.Buffer
	var errBytes bytes.Buffer
	var exitCode int
	var exitErr *execabs.ExitError
	var c []string

	c = append(c, "-S") // ensure no password prompt on tty
	c = append(c, cmdArgs...)

	checkCmd := execabs.Command(config.Config.Sys.Sudo, c...)
	checkCmd.Stdin = bytes.NewBuffer(emptyBytes)
	checkCmd.Stdout = &outBytes
	checkCmd.Stderr = &errBytes
	err = checkCmd.Run()
	if err != nil {
		// ignore exitErr as we check exit code below
		if !errors.As(err, &exitErr) {
			slog.Debug("checkSudoCmd failed starting command", "command", checkCmd.String(), "err", err.Error())
			return err
		}
	}
	exitCode = checkCmd.ProcessState.ExitCode()
	if exitCode != expectedExit {
		slog.Debug("exitCode mismatch running command", "command", cmdArgs, "err", err, "out", outBytes.String(), "err", errBytes.String(), "exitCode", exitCode)
		return errors.New("exitCode mismatch running command")
	}
	if !strings.HasPrefix(outBytes.String(), expectedStdOut) {
		slog.Debug("stdout prefix mismatch running command", "command", cmdArgs, "err", err, "out", outBytes.String(), "err", errBytes.String(), "exitCode", exitCode)
		return errors.New("stdout prefix mismatch running command")
	}
	if !strings.HasPrefix(errBytes.String(), expectedStdErr) {
		slog.Debug("stderr prefix mismatch running command", "command", cmdArgs, "err", err, "out", outBytes.String(), "err", errBytes.String(), "exitCode", exitCode)
		return errors.New("stderr prefix mismatch running command")
	}
	return nil
}

// getTmpFileName returns the name of a tmp file that doesn't exist or maybe an error
func getTmpFileName() (tmpFileName string, err error) {
	tmpDir := os.Getenv("TMPDIR")
	if tmpDir == "" {
		tmpDir = "/tmp"
	}

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	try := 0
	for {
		randomNum := r1.Intn(1000000000)
		tmpFileName = tmpDir + string(os.PathSeparator) + "cirrinad" + strconv.Itoa(randomNum)
		_, err = os.Stat(tmpFileName)
		if err == nil {
			if try++; try < 10000 {
				continue
			}
			// couldn't find a file name that doesn't exist?
			return "", errors.New("couldn't find a tmp file")
		} else if errors.Is(err, fs.ErrNotExist) {
			break
		} else {
			return "", err
		}
	}
	return tmpFileName, nil
}

func kmodLoaded(name string) (loaded bool) {
	slog.Debug("checking module loaded", "module", name)
	cmd := execabs.Command("/sbin/kldstat", "-q", "-n", name)
	err := cmd.Run()
	if err == nil {
		loaded = true
	}
	return loaded
}

func kmodInited(name string) (inited bool) {
	slog.Debug("checking module initialized", "module", name)
	cmd := execabs.Command("/sbin/kldstat", "-q", "-m", name)
	err := cmd.Run()
	if err == nil {
		inited = true
	}
	return inited
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
	var emptyBytes []byte
	var outBytes bytes.Buffer
	var errBytes bytes.Buffer

	checkCmd := execabs.Command("/sbin/sysctl", "-n", "hw.hv_vendor")
	checkCmd.Stdin = bytes.NewBuffer(emptyBytes)
	checkCmd.Stdout = &outBytes
	checkCmd.Stderr = &errBytes
	err := checkCmd.Run()
	if err != nil {
		slog.Error("Failed checking hypervisor", "command", checkCmd.String(), "err", err.Error())
		os.Exit(1)
	}
	hvVendor := strings.TrimSpace(outBytes.String())
	if hvVendor != "" {
		slog.Error("Refusing to run inside virtualized environment", "hvVendor", hvVendor)
		os.Exit(1)
	}
}

func validateJailed() {
	var emptyBytes []byte
	var outBytes bytes.Buffer
	var errBytes bytes.Buffer

	checkCmd := execabs.Command("/sbin/sysctl", "-n", "security.jail.jailed")
	checkCmd.Stdin = bytes.NewBuffer(emptyBytes)
	checkCmd.Stdout = &outBytes
	checkCmd.Stderr = &errBytes
	err := checkCmd.Run()
	if err != nil {
		slog.Error("Failed checking jail status", "command", checkCmd.String(), "err", err.Error())
		os.Exit(1)
	}
	jailed := strings.TrimSpace(outBytes.String())
	slog.Debug("validateJailed", "jailed", jailed)
	if jailed != "0" {
		slog.Error("Refusing to run inside jailed environment")
		os.Exit(1)
	}
}

func validateSudoCommands() {
	var err error

	err = checkSudoCmd(0, "", "", "/sbin/ifconfig")
	if err != nil {
		slog.Error("error running /sbin/ifconfig, check sudo config", "err", err.Error())
		os.Exit(1)
	}

	err = checkSudoCmd(0, "", "", "/sbin/zfs", "-V")
	if err != nil {
		slog.Error("error running /sbin/zfs, check sudo config", "err", err.Error())
		os.Exit(1)
	}

	err = checkSudoCmd(0, "", "", "/usr/bin/nice", "/bin/echo", "-n")
	if err != nil {
		slog.Error("error running /usr/bin/nice, check sudo config", "err", err.Error())
		os.Exit(1)
	}

	err = checkSudoCmd(0, "", "", "/usr/bin/protect", "/bin/echo", "-n")
	if err != nil {
		slog.Error("error running /usr/bin/protect, check sudo config", "err", err.Error())
		os.Exit(1)
	}

	err = checkSudoCmd(0, "", "", "/usr/bin/rctl")
	if err != nil {
		slog.Error("error running /usr/bin/rctl, check sudo config", "err", err.Error())
		os.Exit(1)
	}

	name, err := getTmpFileName()
	if err != nil {
		slog.Error("getTmpFilename failed", "err", err.Error())
		os.Exit(1)
	}
	slog.Debug("Checking tmp file", "name", name)
	err = checkSudoCmd(0, "", "", "/usr/bin/truncate", "-c", "-s", "1", name)
	if err != nil {
		slog.Error("error running /usr/bin/truncate, check sudo config", "err", err.Error())
		os.Exit(1)
	}

	err = checkSudoCmd(0, "", "", "/usr/sbin/bhyve", "-h")
	if err != nil {
		slog.Error("error running /usr/sbin/bhyve, check sudo config", "err", err.Error())
		os.Exit(1)
	}

	err = checkSudoCmd(1, "", "Usage: bhyvectl", "/usr/sbin/bhyvectl")
	if err != nil {
		slog.Error("error running /usr/sbin/bhyvectl, check sudo config", "err", err.Error())
		os.Exit(1)
	}

	err = checkSudoCmd(0, "Available commands:", "", "/usr/sbin/ngctl", "help")
	if err != nil {
		slog.Error("error running /usr/sbin/ngctl, check sudo config", "err", err.Error())
		os.Exit(1)
	}

	err = checkSudoCmd(0, "1 init", "", "/bin/pgrep", "-a", "-l", "-x", "init")
	if err != nil {
		slog.Error("error running /bin/pgrep, check sudo config", "err", err.Error())
		os.Exit(1)
	}
	err = checkSudoCmd(1, "", "usage: chown", "/usr/sbin/chown")
	if err != nil {
		slog.Error("error running /usr/sbin/chown, check sudo config", "err", err.Error())
		os.Exit(1)
	}
}

func validateArch() {
	runtimeArch := runtime.GOARCH
	switch runtimeArch {
	case "amd64":
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
	utsname := unix.Utsname{}
	err := unix.Uname(&utsname)
	if err != nil {
		slog.Error("Unable to validate OS version")
		os.Exit(1)
	}

	var r []byte
	for _, b := range utsname.Release {
		if b == 0 {
			break
		}
		r = append(r, b)
	}

	release := fmt.Sprintf("%s", r)
	re := regexp.MustCompile("-.*")
	ov := re.ReplaceAllString(release, "")
	ovi, err := version.NewVersion(ov)
	if err != nil {
		slog.Error("failed to get OS version", "release", string(utsname.Release[:]))
		os.Exit(1)
	}
	ver132, err := version.NewVersion("13.2")
	if err != nil {
		slog.Error("failed to create a version for 13.2")
		os.Exit(1)
	}
	ver140, err := version.NewVersion("14.0")
	if err != nil {
		slog.Error("failed to create a version for 14.0")
		os.Exit(1)
	}

	slog.Debug("validate OS", "ovi", ovi)
	// Check for valid OS version, see https://www.freebsd.org/security/
	// as of commit, 13.2 and 14.0 are oldest supported versions
	if ovi.LessThan(ver132) && ovi.LessThan(ver140) {
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
	var emptyBytes []byte
	var outBytes bytes.Buffer
	var errBytes bytes.Buffer

	checkCmd := execabs.Command("/sbin/zfs", "list", config.Config.Disk.VM.Path.Zpool)
	checkCmd.Stdin = bytes.NewBuffer(emptyBytes)
	checkCmd.Stdout = &outBytes
	checkCmd.Stderr = &errBytes
	err := checkCmd.Run()
	if err != nil {
		slog.Error("zfs dataset not available, please fix or reconfigure", checkCmd.String(), "err", err.Error())
		os.Exit(1)
	}

	poolParts := strings.Split(config.Config.Disk.VM.Path.Zpool, "/")
	poolName := poolParts[0]

	var rawCapacity string
	cmd := execabs.Command("/sbin/zpool", "list", "-H", poolName)
	defer func(cmd *execabs.Cmd) {
		err := cmd.Wait()
		if err != nil {
			slog.Error("error checking zpool", "err", err)
			os.Exit(1)
		}
	}(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("error checking zpool", "err", err)
		os.Exit(1)
	}
	if err := cmd.Start(); err != nil {
		slog.Error("error checking zpool", "err", err)
		os.Exit(1)
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		text := scanner.Text()
		textFields := strings.Fields(text)
		if len(textFields) != 11 {
			continue
		}
		if !strings.HasSuffix(textFields[7], "%") {
			continue
		}
		rawCapacity = strings.TrimSuffix(textFields[7], "%")
	}
	if err := scanner.Err(); err != nil {
		slog.Error("error checking zpool", "err", err)
		os.Exit(1)
	}
	capacity, err := strconv.Atoi(rawCapacity)
	if err != nil {
		slog.Error("error checking zpool", "err", err)
		os.Exit(1)
	}
	if capacity > 99 {
		slog.Error("zpool at critical usage, refusing to run", "capacity", capacity)
		os.Exit(1)
	} else if capacity > 95 {
		slog.Warn("zpool at very high usage, be careful", "capacity", capacity)
	} else if capacity > 90 {
		slog.Warn("zpool at high usage, be careful", "capacity", capacity)
	} else if capacity > 80 {
		slog.Warn("zpool nearing high usage, consider reducing usage", "capacity", capacity)
	} else {
		slog.Debug("zpool usage OK", "capacity", capacity)
	}
}

func validateNetworkConf() {
	if !util.IsValidIP(config.Config.Network.Grpc.Ip) {
		slog.Error("Invalid listen IP in config, please reconfigure")
		os.Exit(1)
	}

	if !util.IsValidTcpPort(config.Config.Network.Grpc.Port) {
		slog.Error("Invalid listen port in config, please reconfigure")
		os.Exit(1)
	}

	// is MAC parseable?
	macTest := config.Config.Network.Mac.Oui + ":ff:ff:ff"
	_, err := net.ParseMAC(macTest)
	if err != nil {
		slog.Error("Invalid NIC MAC OUI in config, please reconfigure")
		os.Exit(1)
	}

	// is MAC broadcast?
	isBroadcast, err := util.MacIsBroadcast(macTest)
	if err != nil {
		slog.Error("invalid MAC OUI", "OUI", config.Config.Network.Mac.Oui, "err", err)
		os.Exit(1)
	}
	if isBroadcast {
		slog.Error("invalid MAC OUI, may not use potentially broadcast OUI", "oui", config.Config.Network.Mac.Oui)
		os.Exit(1)
	}

	// is MAC multicast?
	isMulticast, err := util.MacIsMulticast(macTest)
	if err != nil {
		slog.Error("invalid MAC OUI ", "OUI", macTest, "err", err)
		os.Exit(1)
	}
	if isMulticast {
		slog.Error("invalid MAC OUI, may not use multicast OUI", "oui", config.Config.Network.Mac.Oui)
		os.Exit(1)
	}
}

func validateVncConfig() {
	if !util.IsValidIP(config.Config.Vnc.Ip) {
		slog.Error("Invalid VNC IP in config, please reconfigure")
		os.Exit(1)
	}

	if !util.IsValidTcpPort(config.Config.Vnc.Port) {
		slog.Error("Invalid VNC port in config, please reconfigure")
		os.Exit(1)
	}
}

func validateDebugConfig() {
	if !util.IsValidIP(config.Config.Debug.Ip) {
		slog.Error("Invalid debug IP in config, please reconfigure")
		os.Exit(1)
	}

	if !util.IsValidTcpPort(config.Config.Debug.Port) {
		slog.Error("Invalid debug port in config, please reconfigure")
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
		slog.Error("rom vars template not installed or path invalid, please install edk2-bhyve (sysutils/edk2) or reconfigure")
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
		logFileStat := logFilePathInfo.Sys().(*syscall.Stat_t)
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

func validateDiskConfig() {
	_, err := util.ParseDiskSize(config.Config.Disk.Default.Size)
	if err != nil {
		slog.Error("default disk size invalid")
		os.Exit(1)
	}

	if config.Config.Disk.VM.Path.Image == "" {
		if err != nil {
			slog.Error("disk.vm.path.image not set, please reconfigure")
			os.Exit(1)
		}
	}

	if config.Config.Disk.VM.Path.Iso == "" {
		if err != nil {
			slog.Error("disk.vm.path.iso not set, please reconfigure")
			os.Exit(1)
		}
	}

	if config.Config.Disk.VM.Path.State == "" {
		if err != nil {
			slog.Error("disk.vm.path.state not set, please reconfigure")
			os.Exit(1)
		}
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

	validateZpoolConf()
}

func validateConfig() {
	// requests init func validates db config via util.ValidateDbConfig()
	validateSudoConfig()
	validateVncConfig()
	validateDebugConfig()
	validateRomConfig()
	validateDiskConfig()
	// validateLogConfig called early in Execute
	validateNetworkConf()
}

func validateSysctls() {
	var emptyBytes []byte
	var outBytes bytes.Buffer
	var errBytes bytes.Buffer

	checkCmd := execabs.Command("/sbin/sysctl", "-n", "security.bsd.see_other_gids")
	checkCmd.Stdin = bytes.NewBuffer(emptyBytes)
	checkCmd.Stdout = &outBytes
	checkCmd.Stderr = &errBytes
	err := checkCmd.Run()
	if err != nil {
		slog.Error("Failed checking sysctl security.bsd.see_other_gids", checkCmd.String(), "err", err.Error())
		os.Exit(1)
	}
	seeOtherGids := strings.TrimSpace(outBytes.String())
	if seeOtherGids != "1" {
		slog.Error("Unable to run with other GIDs not visible", "security.bsd.see_other_gids", seeOtherGids)
		os.Exit(1)
	}

	outBytes.Reset()
	errBytes.Reset()
	checkCmd = execabs.Command("/sbin/sysctl", "-n", "security.bsd.see_other_uids")
	checkCmd.Stdin = bytes.NewBuffer(emptyBytes)
	checkCmd.Stdout = &outBytes
	checkCmd.Stderr = &errBytes
	err = checkCmd.Run()
	if err != nil {
		slog.Error("Failed checking sysctl security.bsd.see_other_uids", "command", checkCmd.String(), "err", err.Error())
		os.Exit(1)
	}
	seeOtherUids := strings.TrimSpace(outBytes.String())
	if seeOtherGids != "1" {
		slog.Error("Unable to run with other UIDs not visible", "security.bsd.see_other_uids", seeOtherUids)
		os.Exit(1)
	}

	outBytes.Reset()
	errBytes.Reset()
	checkCmd = execabs.Command("/sbin/sysctl", "-n", "kern.securelevel")
	checkCmd.Stdin = bytes.NewBuffer(emptyBytes)
	checkCmd.Stdout = &outBytes
	checkCmd.Stderr = &errBytes
	err = checkCmd.Run()
	if err != nil {
		slog.Error("Failed checking sysctl kern.securelevel", "command", checkCmd.String(), "err", err.Error())
		os.Exit(1)
	}
	secureLevelStr := strings.TrimSpace(outBytes.String())
	if err != nil {
		slog.Error("Failed checking sysctl kern.securelevel", "secureLevelStr", secureLevelStr)
		os.Exit(1)
	}
	secureLevel, err := strconv.ParseInt(secureLevelStr, 10, 8)
	if secureLevel > 0 {
		slog.Error("Unable to run with kern.securelevel > 0", "kern.securelevel", secureLevel)
		os.Exit(1)
	}
}

func validateMyId() {
	myUid, myGid, err := util.GetMyUidGid()
	if err != nil {
		slog.Error("failed getting my uid/gid")
		os.Exit(1)
	}
	if myUid == 0 || myGid == 0 {
		slog.Error("refusing to run as root/wheel user/group")
		os.Exit(1)
	}
}

func validateDb() {
	// validate db contents are sane: assume DB is correct, but maybe system state has changed
	// TODO -- validate the backing (file, zvol, volpath) of every disk/iso exists

	var ifUplinks []string
	var ngUplinks []string
	// validate every switch's uplink interface exist, check for duplicates
	allBridges := _switch.GetAll()
	for _, bridge := range allBridges {
		if bridge.Uplink == "" {
			continue
		}
		exists := _switch.CheckInterfaceExists(bridge.Uplink)
		if !exists {
			slog.Warn("bridge uplink does not exist, will be ignored", "bridge", bridge.Name, "uplink", bridge.Uplink)
			continue
		}
		switch bridge.Type {
		case "IF":
			if util.ContainsStr(ifUplinks, bridge.Uplink) {
				slog.Error("uplink used twice", "bridge", bridge.Name, "uplink", bridge.Uplink)
			} else {
				ifUplinks = append(ifUplinks, bridge.Uplink)
			}
		case "NG":
			if util.ContainsStr(ngUplinks, bridge.Uplink) {
				slog.Error("uplink used twice", "bridge", bridge.Name, "uplink", bridge.Uplink)
			} else {
				ngUplinks = append(ngUplinks, bridge.Uplink)
			}
		default:
			slog.Error("unknown switch type checking uplinks", "bridge", bridge.Name, "type", bridge.Type)
		}
	}
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
	validateMyId()
	validateSudoCommands()
	validateSysctls()
	validateConfig()
	validateDb()
}
