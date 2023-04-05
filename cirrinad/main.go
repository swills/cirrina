package main

import (
	"cirrina/cirrinad/vm"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"log"
	"time"
)

const (
	port = ":50051"
)

var sigIntHandlerRunning = false

func handleSigInfo() {
	vm.PrintVMStatus()
}

func handleSigInt() {
	if sigIntHandlerRunning {
		return
	}
	sigIntHandlerRunning = true
	log.Printf("stopping all VMs\n")
	vm.KillVMs()
	for {
		runningVMs := vm.GetRunningVMs()
		if runningVMs == 0 {
			break
		}
		log.Printf("waiting on %v running VM(s)", runningVMs)
		time.Sleep(time.Second)
	}
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

	log.Print("Starting daemon")
	go rpcServer()
	go processRequests()
	for {
		time.Sleep(1 * time.Second)
	}
}
