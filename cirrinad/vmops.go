package main

import (
	"fmt"
	"github.com/kontera-technologies/go-supervisor/v2"
	"gorm.io/gorm"
	"log"
	"strconv"
	"strings"
	"time"
)

var vmProcs = make(map[string]*supervisor.Process)

func startVM(rs *Request) {
	vm := VM{ID: rs.VMID}
	db := getVMDB()
	db.Model(&VM{}).Preload("VMConfig").Limit(1).Find(&vm, &VM{ID: rs.VMID})
	dbSetVMStarting(rs.VMID)
	vm.Start()
	dbSetReqComplete(rs.ID)

}

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

func stopVM(rs *Request) {
	log.Printf("stopping VM %v", rs.VMID)
	vm := VM{ID: rs.VMID}
	db := getVMDB()
	vm.Status = STOPPED
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		MarkReqFailed(rs)
		return
	}
	MarkReqSuccessful(rs)
	p := vmProcs[rs.VMID]
	log.Printf("stopping pid %v", p.Pid())
	dbSetVMStopping(rs.VMID)
	err := p.Stop()
	if err != nil {
		log.Printf("Failed to stop %v", p.Pid())
	}
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

func deleteVM(rs *Request) {
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
