package main

import (
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/vm"
	"fmt"
	"golang.org/x/exp/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"time"
)

var sigIntHandlerRunning = false

func handleSigInfo() {
	var mem runtime.MemStats
	vm.PrintVMStatus()
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
	slog.Info("Starting Daemon")

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
	go vm.AutoStartVMs()
	go rpcServer()
	go processRequests()
	for {
		time.Sleep(1 * time.Second)
	}
}
