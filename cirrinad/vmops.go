package main

import (
	"cirrina/cirrinad/requests"
	"fmt"
	"github.com/kontera-technologies/go-supervisor/v2"
	"log"
	"strconv"
	"strings"
	"time"
)

var vmProcs = make(map[string]*supervisor.Process)

func parseStopMessage(message string) int {
	var exitStatus int
	words := strings.Fields(message)
	if len(words) < 2 {
		return -1
	}
	exitStatusStr := words[2]
	exitStatus, err := strconv.Atoi(exitStatusStr)
	if err != nil {
		fmt.Printf("%T, %v, %v\n", exitStatus, exitStatus, err)
	}
	return exitStatus
}

func startVM(rs *requests.Request) {
	vm := VM{ID: rs.VMID}
	db := getVMDB()
	db.Model(&VM{}).Preload("VMConfig").Limit(1).Find(&vm, &VM{ID: rs.VMID})
	dbSetVMStarting(rs.VMID)
	vm.Start()
	MarkReqSuccessful(rs)
}

func stopVM(rs *requests.Request) {
	log.Printf("stopping VM %v", rs.VMID)
	vm := VM{ID: rs.VMID}
	dbSetVMStopping(vm.ID)
	vm.Stop()
	MarkReqSuccessful(rs)
	dbSetVMStopped(rs.VMID)
}

func vmDaemon(p *supervisor.Process, events chan supervisor.Event, vm VM) {
	for {
		select {
		case msg := <-p.Stdout():
			log.Printf("Received STDOUT message: %s\n", *msg)
		case msg := <-p.Stderr():
			log.Printf("Received STDERR message: %s\n", *msg)
		case event := <-events:
			switch event.Code {
			case "ProcessStart":
				go log.Printf("Received event ProcessStart: %s %s\n", event.Code, event.Message)
				go dbSetVMRunning(vm.ID, p.Pid())
			case "ProcessDone":
				exitStatus := parseStopMessage(event.Message)
				log.Printf("stop message: %v", event.Message)
				log.Printf("VM %v stopped, exitStatus: %v", vm.ID, exitStatus)
				go dbSetVMStopped(vm.ID)
			default:
				log.Printf("Received event: %s - %s\n", event.Code, event.Message)
			}
		case <-p.DoneNotifier():
			log.Println("Closing loop we are done...")
			return
		}
	}
}

func deleteVM(rs *requests.Request) {
	vm := VM{}
	db := getVMDB()
	db.Model(&VM{}).Preload("VMConfig").Limit(1).Find(&vm, &VM{ID: rs.VMID})
	res := db.Delete(&vm.VMConfig)
	if res.RowsAffected != 1 {
		MarkReqFailed(rs)
		return
	}
	res = db.Delete(&vm)
	if res.RowsAffected != 1 {
		MarkReqFailed(rs)
		return
	}
	MarkReqSuccessful(rs)
}

func (vm *VM) Start() {
	log.Printf("Starting %v", vm.Name)
	events := make(chan supervisor.Event)
	p := supervisor.NewProcess(supervisor.ProcessOptions{
		Name:                 "/sbin/ping",
		Args:                 []string{"-c", "9", "localhost"},
		Dir:                  "/",
		Id:                   vm.Name,
		EventNotifier:        events,
		OutputParser:         supervisor.MakeBytesParser,
		ErrorParser:          supervisor.MakeBytesParser,
		MaxSpawns:            -1,
		MaxSpawnAttempts:     -1,
		MaxRespawnBackOff:    time.Duration(vm.VMConfig.RestartDelay) * time.Second,
		MaxSpawnBackOff:      time.Duration(vm.VMConfig.RestartDelay) * time.Second,
		MaxInterruptAttempts: 1,
		MaxTerminateAttempts: 1,
		IdleTimeout:          -1,
	})

	vmProcs[vm.ID] = p

	go vmDaemon(p, events, *vm)

	if err := p.Start(); err != nil {
		panic(fmt.Sprintf("failed to start process: %s", err))
	}
}

func (vm *VM) Stop() {
	p := vmProcs[vm.ID]
	log.Printf("stopping pid %v", p.Pid())
	err := p.Stop()
	if err != nil {
		log.Printf("Failed to stop %v", p.Pid())
	}
}
