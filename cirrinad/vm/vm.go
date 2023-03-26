package vm

import (
	"cirrina/cirrina"
	"fmt"
	"github.com/google/uuid"
	"github.com/kontera-technologies/go-supervisor/v2"
	"gorm.io/gorm"
	"log"
	"strconv"
	"strings"
	"time"
)

type StatusType string

const (
	STOPPED  StatusType = "STOPPED"
	STARTING StatusType = "STARTING"
	RUNNING  StatusType = "RUNNING"
	STOPPING StatusType = "STOPPING"
)

type VMConfig struct {
	gorm.Model
	VMID         string
	Cpu          uint32 `gorm:"default:1;check:cpu BETWEEN 1 and 16"`
	Mem          uint32 `gorm:"default:128;check:mem>=128"`
	MaxWait      uint32 `gorm:"default:120;check:max_wait>=0"`
	Restart      bool   `gorm:"default:True;check:restart IN (0,1)"`
	RestartDelay uint32 `gorm:"default:1;check:restart_delay>=0"`
	Screen       bool   `gorm:"default:True;check:screen IN (0,1)"`
	ScreenWidth  uint32 `gorm:"default:1920;check:screen_width BETWEEN 640 and 1920"`
	ScreenHeight uint32 `gorm:"default:1080;check:screen_height BETWEEN 480 and 1200"`
}

type VM struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null"`
	Name        string `gorm:"not null"`
	Description string
	Status      StatusType `gorm:"type:status_type"`
	BhyvePid    uint32     `gorm:"check:bhyve_pid>=0"`
	NetDev      string
	VNCPort     int32
	VMConfig    VMConfig
}

func (vm *VM) BeforeCreate(_ *gorm.DB) (err error) {
	vm.ID = uuid.NewString()
	return nil
}

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
				go DbSetVMRunning(vm.ID, p.Pid())
			case "ProcessDone":
				exitStatus := parseStopMessage(event.Message)
				log.Printf("stop message: %v", event.Message)
				log.Printf("VM %v stopped, exitStatus: %v", vm.ID, exitStatus)
				go DbSetVMStopped(vm.ID)
			default:
				log.Printf("Received event: %s - %s\n", event.Code, event.Message)
			}
		case <-p.DoneNotifier():
			log.Println("Closing loop we are done...")
			return
		}
	}
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

func VMExists(v *cirrina.VMID) bool {
	vm := VM{}
	db := GetVMDB()
	db.Model(&VM{}).Limit(1).Find(&vm, &VM{ID: v.Value})
	if vm.ID == "" {
		return false
	}
	return true
}
