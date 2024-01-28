package main

import (
	"cirrina/cirrinad/util"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"cirrina/cirrinad/config"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/vm"

	"log/slog"
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

func destroyPidFile() {

}

// write pid file, make sure it doesn't exist already, exit if it does
func writePidFile() {
	pidFilePath, err := filepath.Abs(config.Config.Sys.PidFilePath)
	if err != nil {
		slog.Error("failed to get absolute path to log")
		os.Exit(1)
	}
	_, err = os.Stat(pidFilePath)
	if err == nil {
		slog.Warn("pid file exists, checking pid")
		existingPidFileContent, err := os.ReadFile(pidFilePath)
		if err != nil {
			slog.Error("pid file exists and unable to read it, please fix")
			os.Exit(1)
		}
		existingPid, err := strconv.Atoi(string(existingPidFileContent))
		if err != nil {
			slog.Error("failed getting existing pid")
			os.Exit(1)
		}
		procExists, err := util.PidExists(existingPid)
		if err != nil {
			slog.Error("failed checking existing pid")
			os.Exit(1)
		}
		if procExists {
			slog.Error("duplicate processes not allowed, please kill existing pid", "existingPid", existingPid)
			os.Exit(1)
		} else {
			slog.Warn("left over pid file detected, but process seems not to exist, deleting pid file")
			err := os.Remove(pidFilePath)
			if err != nil {
				slog.Error("failed removing leftover pid file, please fix")
				os.Exit(1)
			}
		}
	}
	myPid := os.Getpid()

	var pidMode os.FileMode
	pidMode = 0x755
	err = os.WriteFile(pidFilePath, []byte(strconv.Itoa(myPid)), pidMode)
	if err != nil {
		slog.Error("failed writing pid file", "err", err)
		os.Exit(1)
		return
	}
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
	destroyPidFile()
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
	logger := slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: programLevel}))
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

	slog.Debug("Checking for existing proc")
	validatePidFilePathConfig()
	slog.Debug("Writing pid file")
	writePidFile()

	slog.Debug("Starting host validation")
	validateSystem()
	slog.Debug("Finished host validation")
	slog.Debug("Clean up starting")
	cleanupSystem()
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
