package main

import (
	"os"
	"os/signal"
	"runtime"
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
