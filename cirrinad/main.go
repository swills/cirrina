package main

import (
	"bytes"
	"cirrina/cirrinad/config"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
	"fmt"
	"golang.org/x/exp/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"time"
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

func main() {
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGINFO)

	go func() {
		for {
			s := <-signals
			sigHandler(s)
		}
	}()
	logFile, err := os.OpenFile(config.Config.Log.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("failed to open log file: %v", err)
		return
	}
	var programLevel = new(slog.LevelVar) // Info by default
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
