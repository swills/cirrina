package main

import (
	"bytes"
	"cirrina/cirrinad/requests"
	exec "golang.org/x/sys/execabs"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"cirrina/cirrinad/config"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"

	"golang.org/x/exp/slog"
)

var mainVersion = "unknown"

var shutdownHandlerRunning = false
var shutdownWaitGroup = sync.WaitGroup{}

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

func shutdownHandler() {
	if shutdownHandlerRunning {
		return
	}
	shutdownHandlerRunning = true
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
	shutdownWaitGroup.Done()
}

func sigHandler(signal os.Signal) {
	slog.Debug("got signal", "signal", signal)
	switch signal {
	case syscall.SIGINFO:
		handleSigInfo()
	case syscall.SIGINT:
		shutdownHandler()
	case syscall.SIGTERM:
		shutdownHandler()
	default:
		slog.Info("Ignoring signal", "signal", signal)
	}
}

func cleanUpVms() {
	vmList := vm.GetAll()
	// deal with any leftover running VMs
	for _, aVm := range vmList {
		vmmPath := "/dev/vmm/" + aVm.Name
		slog.Debug("checking VM", "name", aVm.Name, "path", vmmPath)
		exists, err := util.PathExists(vmmPath)
		if err != nil {
			slog.Error("error checking VM", "err", err)
			continue
		}
		if !exists {
			continue
		}
		slog.Debug("leftover VM exists, checking pid", "name", aVm.Name, "pid", aVm.BhyvePid)
		var pidStat bool
		// check pid
		if aVm.BhyvePid > 0 {
			pidStat, err = util.PidExists(int(aVm.BhyvePid))
			if err != nil {
				slog.Error("error checking VM", "err", err)
			}
		}
		if pidStat {
			slog.Debug("leftover VM exists", "name", aVm.Name, "pid", aVm.BhyvePid, "maxWait", aVm.Config.MaxWait)
			var sleptTime time.Duration
			_ = syscall.Kill(int(aVm.BhyvePid), syscall.SIGTERM)
			for {
				pidStat, err = util.PidExists(int(aVm.BhyvePid))
				if err != nil {
					slog.Error("error checking VM", "err", err)
					break
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
			} else {
				if pidStillExists {
					slog.Error("VM refused to die")
				}
			}
		}
		slog.Debug("destroying VM", "name", aVm.Name)
		aVm.MaybeForceKillVM()
	}

	// clean up leftover nets and mark everything stopped
	for _, aVm := range vmList {
		slog.Debug("cleaning up VM net(s)", "name", aVm.Name)
		aVm.NetCleanup()
		slog.Debug("marking VM stopped", "name", aVm.Name)
		aVm.SetStopped()
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

func cleanupDb() {
	rowsCleared := requests.FailAllPending()
	slog.Debug("cleared failed requests", "rowsCleared", rowsCleared)
}

func main() {
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGINFO)
	signal.Notify(signals, os.Interrupt, syscall.SIGINT)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		for {
			s := <-signals
			sigHandler(s)
		}
	}()

	validateLogConfig()

	logFile, err := os.OpenFile(config.Config.Log.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Error("failed to open log file", err)
		return
	}
	programLevel := new(slog.LevelVar) // Info by default
	logger := slog.New(slog.HandlerOptions{Level: programLevel}.NewTextHandler(logFile))
	slog.SetDefault(logger)
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

	slog.Debug("Starting host validation")
	validateSystem()
	slog.Debug("Finished host validation")
	slog.Debug("Clean up starting")
	cleanUpVms()
	cleanupNet()
	cleanupDb()
	slog.Debug("Clean up complete")

	slog.Debug("Creating bridges")
	_switch.CreateBridges()

	slog.Info("Starting Daemon")

	go vm.AutoStartVMs()
	go rpcServer()
	go processRequests()

	shutdownWaitGroup.Add(1)
	shutdownWaitGroup.Wait()
}
