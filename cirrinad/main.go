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

	"log"
	"time"
)

var sigIntHandlerRunning = false

func handleSigInfo() {
	var mem runtime.MemStats
	vm.PrintVMStatus()
	runtime.ReadMemStats(&mem)
	log.Printf("mem.Alloc: %v", mem.Alloc)
	log.Printf("mem.TotalAlloc: %v", mem.TotalAlloc)
	log.Printf("mem.HeapAlloc: %v", mem.HeapAlloc)
	log.Printf("mem.NumGC: %v", mem.NumGC)
	log.Printf("mem.Sys: %v", mem.Sys)
	runtime.GC()
	runtime.ReadMemStats(&mem)
	log.Printf("mem.Alloc: %v", mem.Alloc)
	log.Printf("mem.TotalAlloc: %v", mem.TotalAlloc)
	log.Printf("mem.HeapAlloc: %v", mem.HeapAlloc)
	log.Printf("mem.NumGC: %v", mem.NumGC)
	log.Printf("mem.Sys: %v", mem.Sys)
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
		log.Printf("waiting on %v running VM(s)", runningVMs)
		time.Sleep(time.Second)
	}
	log.Printf("Exiting normally")
	os.Exit(0)
}

func handleSigTerm() {
	fmt.Printf("SIGTERM received, exiting\n")
	os.Exit(0)
}

func sigHandler(signal os.Signal) {
	log.Printf("handling signal %v", signal)
	switch signal {
	case syscall.SIGINFO:
		go handleSigInfo()
	case syscall.SIGINT:
		go handleSigInt()
	case syscall.SIGTERM:
		handleSigTerm()
	case syscall.SIGCHLD:
		log.Printf("got SIGCHLD")
	default:
		fmt.Println("Ignoring signal ", signal)
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
		log.Printf("failed to open log file: %v", err)
		return
	}
	logger := slog.New(slog.NewTextHandler(logFile))
	slog.SetDefault(logger)
	log.Print("Starting daemon")
	go vm.AutoStartVMs()
	go rpcServer()
	go processRequests()
	for {
		time.Sleep(1 * time.Second)
	}
}
