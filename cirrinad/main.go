package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"cirrina/cirrinad/config"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"

	"golang.org/x/exp/slog"
)

var sigIntHandlerRunning = false

func handleSigInfo() {
	var mem runtime.MemStats
	vm.LogAllVmStatus()
	runtime.ReadMemStats(&mem)
	slog.Debug("MemStats",
		"mem.Alloc", mem.Alloc,
		"mem.TotalAlloc", mem.TotalAlloc,
		"mem.HeapAlloc", mem.HeapAlloc,
		"mem.NumGC", mem.NumGC,
		"mem.Sys", mem.Sys,
	)
	runtime.GC()
	runtime.ReadMemStats(&mem)
	slog.Debug("MemStats",
		"mem.Alloc", mem.Alloc,
		"mem.TotalAlloc", mem.TotalAlloc,
		"mem.HeapAlloc", mem.HeapAlloc,
		"mem.NumGC", mem.NumGC,
		"mem.Sys", mem.Sys,
	)
}

func handleSigInt() {
	if sigIntHandlerRunning {
		return
	}
	sigIntHandlerRunning = true
	vm.KillVMs()
	for {
		runningVMs := vm.GetRunningVMs()
		if runningVMs == 0 {
			break
		}
		slog.Info("waiting on running VM(s)", "count", runningVMs)
		time.Sleep(time.Second)
	}
	_switch.DestroyBridges()
	slog.Info("Exiting normally")
	os.Exit(0)
}

func handleSigTerm() {
	fmt.Printf("SIGTERM received, exiting\n")
	os.Exit(0)
}

func sigHandler(signal os.Signal) {
	slog.Debug("got signal", "signal", signal)
	switch signal {
	case syscall.SIGINFO:
		go handleSigInfo()
	case syscall.SIGINT:
		go handleSigInt()
	case syscall.SIGTERM:
		handleSigTerm()
	default:
		slog.Info("Ignoring signal", "signal", signal)
	}
}

func cleanUpVms() {
	vmList := vm.GetAll()
	for _, aVm := range vmList {
		if aVm.Status != vm.STOPPED {
			// check /dev/vmm entry
			vmmPath := "/dev/vmm/" + aVm.Name
			slog.Debug("checking VM", "name", aVm.Name, "path", vmmPath)
			exists, err := util.PathExists(vmmPath)
			if err != nil {
				slog.Error("error checking VM", "err", err)
			}
			slog.Debug("leftover VM exists, checking pid", "name", aVm.Name, "pid", aVm.BhyvePid)
			// check pid
			pidStat, err := util.PidExists(int(aVm.BhyvePid))
			if err != nil {
				slog.Error("error checking VM", "err", err)
			}
			if exists {
				slog.Debug("killing VM")
				if pidStat {
					slog.Debug("leftover pid exists", "name", aVm.Name, "pid", aVm.BhyvePid, "maxWait", aVm.Config.MaxWait)
					var sleptTime time.Duration
					err = syscall.Kill(int(aVm.BhyvePid), syscall.SIGTERM)
					if err != nil {
						return
					}
					for {
						pidStat, err := util.PidExists(int(aVm.BhyvePid))
						if err != nil {
							slog.Error("error checking VM", "err", err)
							return
						}
						if !pidStat {
							break
						}
						time.Sleep(10 * time.Millisecond)
						sleptTime += 10 * time.Millisecond
						if sleptTime > (time.Duration(aVm.Config.MaxWait) * time.Second) {
							break
						}
					}
					pidStillExists, err := util.PidExists(int(aVm.BhyvePid))
					if err != nil {
						slog.Error("error checking VM", "err", err)
						return
					}
					if pidStillExists {
						slog.Error("VM refused to die")
					}
				}
			}
			slog.Debug("destroying VM", "name", aVm.Name)
			aVm.MaybeForceKillVM()
			aVm.NetCleanup()
			aVm.SetStopped()
		}
	}
}

func cleanupNet() {
	// destroy all the bridges we know about
	_switch.DestroyBridges()

	// look for network things in cirrinad group and destroy them
	netInterfaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	slog.Debug("GetHostInterfaces", "netInterfaces", netInterfaces)
	for _, inter := range netInterfaces {
		intGroups, err := util.GetIntGroups(inter.Name)
		if err != nil {
			slog.Error("failed to get interface groups", "err", err)
		}
		if !util.ContainsStr(intGroups, "cirrinad") {
			continue
		}
		slog.Debug("leftover interface found, destroying", "name", inter.Name)

		cmd := exec.Command(config.Config.Sys.Sudo, "/sbin/ifconfig", inter.Name, "destroy")
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Start(); err != nil {
			slog.Error("failed running ifconfig", "err", err, "out", out)
		}
		if err := cmd.Wait(); err != nil {
			slog.Error("failed running ifconfig", "err", err, "out", out)
		}
	}
}

func kmodLoaded(name string) (loaded bool) {
	slog.Debug("checking module loaded", "module", name)
	cmd := exec.Command("/sbin/kldstat", "-q", "-n", name)
	err := cmd.Run()
	if err == nil {
		loaded = true
	}
	return loaded
}

func kmodInited(name string) (inited bool) {
	slog.Debug("checking module initialized", "module", name)
	cmd := exec.Command("/sbin/kldstat", "-q", "-m", name)
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
			slog.Debug("module not loaded", "module", module)
			fmt.Printf("Module %s not loaded, please load before using\n", module)
			os.Exit(1)
		}
		inited := kmodInited(module)
		if !inited {
			slog.Debug("module not initialized", "module", module)
			fmt.Printf("Module %s not initialized, please fix before using\n", module)
			os.Exit(1)
		}
	}
}

func validateVirt() {
	var emptyBytes []byte
	var outBytes bytes.Buffer
	var errBytes bytes.Buffer
	var exitErr *exec.ExitError
	var exitCode int

	checkCmd := exec.Command(config.Config.Sys.Sudo, "-S", "/sbin/sysctl", "hw.hv_vendor")
	checkCmd.Stdin = bytes.NewBuffer(emptyBytes)
	checkCmd.Stdout = &outBytes
	checkCmd.Stderr = &errBytes
	err := checkCmd.Run()
	if err != nil {
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
		if exitCode != 0 || outBytes.String() != "hw.hv_vendor: " {
			slog.Error("Refusing to run inside virtualized environment")
			fmt.Printf("Refusing to run inside virtualized environment\n")
			os.Exit(1)
		}
	}
}

func checkSudoCmd(expectedExit int, expectedOut string, cmdArgs ...string) (err error) {
	var emptyBytes []byte
	var outBytes bytes.Buffer
	var errBytes bytes.Buffer
	var exitErr *exec.ExitError
	var exitCode int
	var c []string

	c = append(c, "-S") // ensure no password prompt on tty
	c = append(c, cmdArgs...)

	checkCmd := exec.Command(config.Config.Sys.Sudo, c...)
	checkCmd.Stdin = bytes.NewBuffer(emptyBytes)
	checkCmd.Stdout = &outBytes
	checkCmd.Stderr = &errBytes
	err = checkCmd.Run()
	if err != nil {
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
		if exitCode != expectedExit || !strings.HasPrefix(errBytes.String(), expectedOut) {
			slog.Error("failed running command", "command", cmdArgs, "err", err, "out", outBytes.String(), "err", errBytes.String(), "exitCode", exitCode)
			return err
		}
	}
	return nil
}

func validateSudo() {
	var err error

	err = checkSudoCmd(0, "", "/sbin/ifconfig")
	if err != nil {
		fmt.Printf("error running /sbin/ifconfig, check sudo config\n")
		os.Exit(1)
	}

	err = checkSudoCmd(0, "", "/sbin/zfs", "-V")
	if err != nil {
		fmt.Printf("error running /sbin/zfs, check sudo config\n")
		os.Exit(1)
	}

	err = checkSudoCmd(0, "", "/usr/bin/nice", "/bin/echo", "-n")
	if err != nil {
		fmt.Printf("error running /usr/bin/nice, check sudo config\n")
		os.Exit(1)
	}

	err = checkSudoCmd(0, "", "/usr/bin/protect", "/bin/echo", "-n")
	if err != nil {
		fmt.Printf("error running /usr/bin/protect, check sudo config\n")
		os.Exit(1)
	}

	err = checkSudoCmd(0, "", "/usr/bin/rctl")
	if err != nil {
		fmt.Printf("error running /usr/bin/rctl, check sudo config\n")
		os.Exit(1)
	}

	tmpFile, ok := os.CreateTemp("", "cirrinad")
	if ok != nil {
		slog.Error("failed creating tmp file")
		fmt.Printf("failed creating tmp file")
		os.Exit(1)
	}
	err = checkSudoCmd(0, "", "/usr/bin/truncate", "-c", "-s", "1", tmpFile.Name())
	if err != nil {
		fmt.Printf("error running /usr/bin/truncate, check sudo config\n")
		os.Exit(1)
	}

	err = checkSudoCmd(0, "", "/usr/sbin/bhyve", "-h")
	if err != nil {
		fmt.Printf("error running /usr/sbin/bhyve, check sudo config\n")
		os.Exit(1)
	}

	err = checkSudoCmd(1, "Usage: bhyvectl", "/usr/sbin/bhyvectl")
	if err != nil {
		fmt.Printf("error running /usr/sbin/bhyvectl, check sudo config\n")
		os.Exit(1)
	}

	err = checkSudoCmd(1, "Usage: bhyvectl", "/usr/sbin/ngctl", "help")
	if err != nil {
		fmt.Printf("error running /usr/sbin/ngctl, check sudo config\n")
		os.Exit(1)
	}
}

func validateArch() {
	runtimeArch := runtime.GOARCH
	switch runtimeArch {
	case "amd64":
	default:
		fmt.Printf("Unsupported Architecture\n")
		os.Exit(1)
	}
}

func validateOS() {
	runtimeOS := runtime.GOOS
	switch runtimeOS {
	case "freebsd":
	default:
		fmt.Printf("Unsupported OS\n")
		os.Exit(1)
	}
}

func validateOSVersion() {
	utsname := unix.Utsname{}
	err := unix.Uname(&utsname)
	if err != nil {
		slog.Error("Failed to get uname", "err", err)
		fmt.Printf("Unable to validate OS version\n")
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
	ovi, err := strconv.ParseFloat(ov, 32)
	if err != nil {
		slog.Error("failed to get OS version", "release", string(utsname.Release[:]))
		fmt.Printf("Error getting OS version\n")
		os.Exit(1)
	}

	slog.Debug("validate OS", "ovi", ovi)
	// Check for valid OS version, see https://www.freebsd.org/security/
	// as of commit, 12.4 and 13.2 are oldest supported versions
	if ovi < 12.4 || (ovi > 13 && ovi < 13.2) {
		slog.Error("Unsupported OS version", "ovi", ovi)
		fmt.Printf("Unsupported OS version: %f\n", ovi)
		os.Exit(1)
	}
}

func validateSystem() {
	slog.Debug("validating system")
	validateArch()
	validateOS()
	validateOSVersion()
	validateKmods()
	validateVirt()
	validateSudo()
	// TODO: further validation
}

func main() {
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGINFO)

	go func() {
		for {
			s := <-signals
			sigHandler(s)
		}
	}()
	logFile, err := os.OpenFile(config.Config.Log.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Error("failed to open log file: %v", err)
		return
	}
	programLevel := new(slog.LevelVar) // Info by default
	logger := slog.New(slog.HandlerOptions{Level: programLevel}.NewTextHandler(logFile))
	slog.SetDefault(logger)
	if config.Config.Log.Level == "info" {
		slog.Info("log level set to info")
		programLevel.Set(slog.LevelInfo)
	} else if config.Config.Log.Level == "debug" {
		slog.Info("log level set to debug")
		programLevel.Set(slog.LevelDebug)
	} else {
		programLevel.Set(slog.LevelInfo)
		slog.Info("log level not set or un-parseable, setting to info")
	}

	slog.Debug("Starting host validation")
	validateSystem()
	slog.Debug("Finished host validation")
	slog.Debug("Clean up starting")
	cleanUpVms()
	cleanupNet()
	slog.Debug("Clean up complete")

	slog.Debug("Creating bridges")
	_switch.CreateBridges()

	slog.Info("Starting Daemon")

	go vm.AutoStartVMs()
	go rpcServer()
	go processRequests()

	for {
		time.Sleep(1 * time.Second)
	}
}
