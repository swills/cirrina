package vm

import (
	"errors"
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

type Config struct {
	gorm.Model
	VmId         string
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
	VMConfig    Config
}

var vmProcesses = make(map[string]*supervisor.Process)

func (vm *VM) BeforeCreate(_ *gorm.DB) (err error) {
	vm.ID = uuid.NewString()
	return nil
}

func Create(vm *VM) error {
	if vm.ID != "" {
		return errors.New("cannot specify VM Id")
	}
	_, err := GetByName(vm.Name)
	if err == nil {
		return errors.New("vm with same name already exists")
	}
	db := getVmDb()
	log.Printf("Creating VM %v", vm.Name)
	res := db.Create(&vm)
	return res.Error
}

func (vm *VM) Delete() (err error) {
	db := getVmDb()
	db.Model(&VM{}).Preload("VMConfig").Limit(1).Find(&vm, &VM{ID: vm.ID})
	if vm.ID == "" {
		return errors.New("not found")
	}
	res := db.Delete(&vm.VMConfig)
	if res.RowsAffected != 1 {
		return errors.New("failed to delete VMConfig")
	}
	res = db.Delete(&vm)
	if res.RowsAffected != 1 {
		return errors.New("failed to delete VM")
	}
	return nil
}

func (vm *VM) Start() (err error) {
	if vm.Status != STOPPED {
		return errors.New("must be stopped first")
	}
	log.Printf("Starting VM %v", vm.Name)
	log.Printf("vm: %v", vm)
	setStarting(vm.ID)
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

	vmProcesses[vm.ID] = p

	go vmDaemon(p, events, *vm)

	if err := p.Start(); err != nil {
		panic(fmt.Sprintf("failed to start process: %s", err))
	}
	return nil
}

func (vm *VM) Stop() (err error) {
	if vm.Status != RUNNING {
		return errors.New("must be running first")
	}
	p := vmProcesses[vm.ID]
	log.Printf("stopping pid %v", p.Pid())
	setStopping(vm.ID)
	err = p.Stop()
	if err != nil {
		log.Printf("Failed to stop %v", p.Pid())
		return errors.New("stop failed")
	}
	setStopped(vm.ID)
	return nil
}

func (vm *VM) Save() error {
	db := getVmDb()
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		return errors.New("error updating VM")
	}
	return nil
}

func (vm *VM) String() string {
	return fmt.Sprintf("name: %s id: %s", vm.Name, vm.ID)
}

func GetAll() []VM {
	var result []VM

	db := getVmDb()
	db.Find(&result)

	return result
}

func GetByID(id string) (vm VM, err error) {
	db := getVmDb()
	db.Model(&VM{}).Preload("VMConfig").Limit(1).Find(&vm, &VM{ID: id})
	if vm.ID == "" {
		return VM{}, errors.New("not found")
	}
	return vm, nil
}

func GetByName(name string) (vm VM, err error) {
	db := getVmDb()
	db.Model(&VM{}).Preload("VMConfig").Limit(1).Find(&vm, &VM{Name: name})
	if vm.ID == "" {
		return VM{}, errors.New("not found")
	}
	return vm, nil
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
				go setRunning(vm.ID, p.Pid())
			case "ProcessDone":
				exitStatus := parseStopMessage(event.Message)
				log.Printf("stop message: %v", event.Message)
				log.Printf("VM %v stopped, exitStatus: %v", vm.ID, exitStatus)
				go setStopped(vm.ID)
			default:
				log.Printf("Received event: %s - %s\n", event.Code, event.Message)
			}
		case <-p.DoneNotifier():
			log.Println("Closing loop we are done...")
			return
		}
	}
}
