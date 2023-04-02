package vm

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/kontera-technologies/go-supervisor/v2"
	"gorm.io/gorm"
	"log"
	"os"
	"os/exec"
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

var baseVMStatePath = "/usr/home/swills/.local/state/weasel/vms/"
var bootRomPath = "/usr/local/share/uefi-firmware/BHYVE_UEFI.fd"
var uefiVarFileTemplate = "/usr/local/share/uefi-firmware/BHYVE_UEFI_VARS.fd"

type Config struct {
	gorm.Model
	VmId             string
	Cpu              uint32 `gorm:"default:1;check:cpu BETWEEN 1 and 16"`
	Mem              uint32 `gorm:"default:128;check:mem>=128"`
	MaxWait          uint32 `gorm:"default:120;check:max_wait>=0"`
	Restart          bool   `gorm:"default:True;check:restart IN (0,1)"`
	RestartDelay     uint32 `gorm:"default:1;check:restart_delay>=0"`
	Net              bool   `gorm:"default:True;check:screen IN (0,1)"`
	Mac              string `gorm:"default:AUTO"`
	Screen           bool   `gorm:"default:True;check:screen IN (0,1)"`
	ScreenWidth      uint32 `gorm:"default:1920;check:screen_width BETWEEN 640 and 1920"`
	ScreenHeight     uint32 `gorm:"default:1080;check:screen_height BETWEEN 480 and 1200"`
	VNCWait          bool   `gorm:"default:False;check:restart IN(0,1)"`
	VNCPort          string `gorm:"default:AUTO"`
	Tablet           bool   `gorm:"default:True;check:restart IN(0,1)"`
	StoreUEFIVars    bool   `gorm:"default:True;check:restart IN(0,1)"`
	UTCTime          bool   `gorm:"default:True;check:restart IN(0,1)"`
	HostBridge       bool   `gorm:"default:True;check:restart IN(0,1)"`
	ACPI             bool   `gorm:"default:True;check:restart IN(0,1)"`
	UseHLT           bool   `gorm:"default:True;check:restart IN(0,1)"`
	ExitOnPause      bool   `gorm:"default:True;check:restart IN (0,1)"`
	WireGuestMem     bool   `gorm:"default:True;check:restart IN (0,1)"`
	DestroyPowerOff  bool   `gorm:"default:True;check:restart IN (0,1)"`
	IgnoreUnknownMSR bool   `gorm:"default:True;check:restart IN (0,1)"`
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
	Config      Config
	netDev      string
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
	if strings.Contains(vm.Name, "/") {
		return errors.New("illegal character in vm name")
	}
	db := getVmDb()
	log.Printf("Creating VM %v", vm.Name)
	res := db.Create(&vm)
	return res.Error
}

func (vm *VM) Delete() (err error) {
	db := getVmDb()
	db.Model(&VM{}).Preload("Config").Limit(1).Find(&vm, &VM{ID: vm.ID})
	if vm.ID == "" {
		return errors.New("not found")
	}
	res := db.Delete(&vm.Config)
	if res.RowsAffected != 1 {
		return errors.New("failed to delete Config")
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

	cmdName, cmdArgs, err := vm.generateCommandLine()
	if err != nil {
		return err
	}
	log.Printf("cmd: %v, args: %v", cmdName, cmdArgs)
	vm.createUefiVarsFile()
	if vm.Config.Net {
		vm.createTapInt()
		vm.addTapToBridge()
	}
	// TODO -- check return code --
	// EXIT STATUS
	//     Exit status indicates how the VM was terminated:
	//
	//     0       rebooted 				-- definitely restart
	//     1       powered off				-- don't restart?
	//     2       halted					-- ??
	//     3       triple fault				-- ??
	//     4       exited due to an error	-- ??

	p := supervisor.NewProcess(supervisor.ProcessOptions{
		Name:                    cmdName,
		Args:                    cmdArgs,
		Dir:                     "/",
		Id:                      vm.Name,
		EventNotifier:           events,
		OutputParser:            supervisor.MakeBytesParser,
		ErrorParser:             supervisor.MakeBytesParser,
		MaxSpawns:               -1,
		MaxSpawnAttempts:        -1,
		MaxRespawnBackOff:       time.Duration(vm.Config.RestartDelay) * time.Second,
		MaxSpawnBackOff:         time.Duration(vm.Config.RestartDelay) * time.Second,
		MaxInterruptAttempts:    1,
		MaxTerminateAttempts:    1,
		IdleTimeout:             -1,
		TerminationGraceTimeout: time.Duration(vm.Config.MaxWait) * time.Second,
	})

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
	delete(vmProcesses, vm.ID)
	vm.maybeForceKillVM()
	return nil
}

func (vm *VM) Save() error {
	db := getVmDb()

	res := db.Model(&vm.Config).
		Updates(map[string]interface{}{
			"cpu":                &vm.Config.Cpu,
			"mem":                &vm.Config.Mem,
			"max_wait":           &vm.Config.MaxWait,
			"restart":            &vm.Config.Restart,
			"restart_delay":      &vm.Config.RestartDelay,
			"net":                &vm.Config.Net,
			"mac":                &vm.Config.Mac,
			"screen":             &vm.Config.Screen,
			"screen_width":       &vm.Config.ScreenWidth,
			"screen_height":      &vm.Config.ScreenHeight,
			"vnc_wait":           &vm.Config.VNCWait,
			"vnc_port":           &vm.Config.VNCPort,
			"tablet":             &vm.Config.Tablet,
			"store_uefi_vars":    &vm.Config.StoreUEFIVars,
			"utc_time":           &vm.Config.UTCTime,
			"host_bridge":        &vm.Config.HostBridge,
			"acpi":               &vm.Config.ACPI,
			"use_hlt":            &vm.Config.UseHLT,
			"exit_on_pause":      &vm.Config.ExitOnPause,
			"wire_guest_mem":     &vm.Config.WireGuestMem,
			"destroy_power_off":  &vm.Config.DestroyPowerOff,
			"ignore_unknown_msr": &vm.Config.IgnoreUnknownMSR,
		},
		)

	if res.Error != nil {
		return errors.New("error updating VM")
	}

	res = db.Select([]string{"name", "description", "vnc_port"}).Model(&vm).
		Updates(map[string]interface{}{
			"name":        &vm.Name,
			"description": &vm.Description,
			"vnc_port":    &vm.VNCPort,
		})

	if res.Error != nil {
		log.Printf("db update error: %v", res.Error)
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
	db.Model(&VM{}).Preload("Config").Limit(1).Find(&vm, &VM{ID: id})
	if vm.ID == "" {
		return VM{}, errors.New("not found")
	}
	return vm, nil
}

func GetByName(name string) (vm VM, err error) {
	db := getVmDb()
	db.Model(&VM{}).Preload("Config").Limit(1).Find(&vm, &VM{Name: name})
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

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (vm *VM) maybeForceKillVM() {
	ex, err := exists("/dev/vmm/" + vm.Name)
	if err != nil {
		return
	}
	if !ex {
		return
	}
	args := []string{"/usr/sbin/bhyvectl", "--destroy"}
	args = append(args, "--vm="+vm.Name)
	cmd := exec.Command("/usr/local/bin/sudo", args...)
	_ = cmd.Run()
}

func copyFile(in, out string) (int64, error) {
	i, e := os.Open(in)
	if e != nil {
		return 0, e
	}
	defer func(i *os.File) {
		_ = i.Close()
	}(i)
	o, e := os.Create(out)
	if e != nil {
		return 0, e
	}
	defer func(o *os.File) {
		_ = o.Close()
	}(o)
	return o.ReadFrom(i)
}

func (vm *VM) createUefiVarsFile() {
	uefiVarsFilePath := baseVMStatePath + "/" + vm.Name
	uefiVarsFile := uefiVarsFilePath + "/BHYVE_UEFI_VARS.fd"
	uvPathExists, err := exists(uefiVarsFilePath)
	if err != nil {
		return
	}
	if !uvPathExists {
		err = os.Mkdir(uefiVarsFilePath, 0755)
		if err != nil {
			log.Printf("failed to create uefi vars path: %v", err)
			return
		}
	}
	uvFileExists, err := exists(uefiVarsFile)
	if err != nil {
		return
	}
	if !uvFileExists {
		_, err = copyFile(uefiVarFileTemplate, uefiVarsFile)
		if err != nil {
			log.Printf("failed to copy uefiVars template: %v", err)
		}
	}
}

func (vm *VM) createTapInt() {
	log.Printf("creating tap dev %v", vm.netDev)
	args := []string{"/sbin/ifconfig", vm.netDev, "create"}
	cmd := exec.Command("/usr/local/bin/sudo", args...)
	err := cmd.Run()
	if err != nil {
		log.Printf("failed to create tap: %v", err)
	}
}

func (vm *VM) destroyTapInt() {
	log.Printf("destroying tap dev %v", vm.netDev)
	args := []string{"/sbin/ifconfig", vm.netDev, "destroy"}
	cmd := exec.Command("/usr/local/bin/sudo", args...)
	err := cmd.Run()
	if err != nil {
		log.Printf("failed to destroy tap : %v", err)
	}

}

func (vm *VM) addTapToBridge() {
	log.Printf("Adding tap dev %v to bridge", vm.netDev)
	args := []string{"/sbin/ifconfig", "bridge0", "addm", vm.netDev}
	cmd := exec.Command("/usr/local/bin/sudo", args...)
	err := cmd.Run()
	if err != nil {
		log.Printf("failed to add tap to bridge: %v", err)
	}

}

func vmDaemon(p *supervisor.Process, events chan supervisor.Event, vm VM) {
	for {
		select {
		case msg := <-p.Stdout():
			log.Printf("VM %v Received STDOUT message: %s\n", vm.ID, *msg)
		case msg := <-p.Stderr():
			log.Printf("VM %v Received STDERR message: %s\n", vm.ID, *msg)
		case event := <-events:
			switch event.Code {
			case "ProcessStart":
				go log.Printf("VM %v Received event ProcessStart: %s %s\n", vm.ID, event.Code, event.Message)
				go setRunning(vm.ID, p.Pid())
				vmProcesses[vm.ID] = p
			case "ProcessDone":
				exitStatus := parseStopMessage(event.Message)
				log.Printf("stop message: %v", event.Message)
				log.Printf("VM %v stopped, exitStatus: %v", vm.ID, exitStatus)
				setStopped(vm.ID)
				delete(vmProcesses, vm.ID)
				if vm.Config.Net {
					// TODO - handle vmnet and netgraph
					vm.destroyTapInt()
				}
				vm.maybeForceKillVM()
			default:
				log.Printf("VM %v Received event: %s - %s\n", vm.ID, event.Code, event.Message)
			}
		case <-p.DoneNotifier():
			setStopped(vm.ID)
			delete(vmProcesses, vm.ID)
			if vm.Config.Net {
				// TODO - handle vmnet and netgraph
				vm.destroyTapInt()
			}
			vm.maybeForceKillVM()
			log.Printf("VM %v closing loop we are done...", vm.ID)
			return
		}
	}
}

func PrintVMStatus() {
	runningVMs := len(vmProcesses)
	log.Printf("vmstatus: running VMs: %v", runningVMs)
	for vmId := range vmProcesses {
		log.Printf("running vm: %v, pid: %v", vmId, vmProcesses[vmId].Pid())
	}
}

func KillVMs() {
	for vmId, process := range vmProcesses {
		log.Printf("killing vm: %v", vmId)
		go func(newProcess *supervisor.Process) {
			_ = newProcess.Stop()
		}(process)
	}

}

func GetRunningVMs() int {
	return len(vmProcesses)
}
