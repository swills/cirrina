package vm

import (
	"cirrina/cirrinad/config"
	"errors"
	"github.com/kontera-technologies/go-supervisor/v2"
	"gorm.io/gorm"
	"log"
	"sync"
)

type StatusType string

const (
	STOPPED  StatusType = "STOPPED"
	STARTING StatusType = "STARTING"
	RUNNING  StatusType = "RUNNING"
	STOPPING StatusType = "STOPPING"
)

var baseVMStatePath = config.Config.Disk.VM.Path.State
var bootRomPath = config.Config.Rom.Path
var uefiVarFileTemplate = config.Config.Rom.Vars.Template

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
	VNCWait          bool   `gorm:"default:False;check:vnc_wait IN(0,1)"`
	VNCPort          string `gorm:"default:AUTO"`
	Tablet           bool   `gorm:"default:True;check:tablet IN(0,1)"`
	StoreUEFIVars    bool   `gorm:"default:True;check:store_uefi_vars IN(0,1)"`
	UTCTime          bool   `gorm:"default:True;check:utc_time IN(0,1)"`
	HostBridge       bool   `gorm:"default:True;check:host_bridge IN(0,1)"`
	ACPI             bool   `gorm:"default:True;check:acpi IN(0,1)"`
	UseHLT           bool   `gorm:"default:True;check:use_hlt IN(0,1)"`
	ExitOnPause      bool   `gorm:"default:True;check:exit_on_pause IN (0,1)"`
	WireGuestMem     bool   `gorm:"default:True;check:wire_guest_mem IN (0,1)"`
	DestroyPowerOff  bool   `gorm:"default:True;check:destroy_power_off IN (0,1)"`
	IgnoreUnknownMSR bool   `gorm:"default:True;check:ignore_unknown_msr IN (0,1)"`
	KbdLayout        string `gorm:"default:default"`
	AutoStart        bool   `gorm:"default:False;check:auto_start IN (0,1)"`
	NetType          string `gorm:"default:VIRTIONET;check:net_type IN (\"VIRTIONET\",\"E1000\")"`
	NetDevType       string `gorm:"default:TAP;check:net_dev_type IN (\"TAP\",\"VMNET\",\"NETGRAPH\")"`
	Sound            bool   `gorm:"default:False;check:vnc_wait IN(0,1)"`
	SoundIn          string `gorm:"default:/dev/dsp0"`
	SoundOut         string `gorm:"default:/dev/dsp0"`
	Com1             bool   `gorm:"default:True;check:vnc_wait IN(0,1)"`
	Com1Dev          string `gorm:"default:AUTO"`
	Com2             bool   `gorm:"default:False;check:vnc_wait IN(0,1)"`
	Com2Dev          string `gorm:"default:AUTO"`
	Com3             bool   `gorm:"default:False;check:vnc_wait IN(0,1)"`
	Com3Dev          string `gorm:"default:AUTO"`
	Com4             bool   `gorm:"default:False;check:vnc_wait IN(0,1)"`
	Com4Dev          string `gorm:"default:AUTO"`
	ExtraArgs        string
	ISOs             string
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
	proc        *supervisor.Process
	mu          sync.Mutex
}

type ListType struct {
	Mu     sync.Mutex
	VmList map[string]*VM
}

var List = ListType{
	VmList: map[string]*VM{},
}

func init() {

	db := getVmDb()
	err := db.AutoMigrate(&VM{})
	if err != nil {
		panic("failed to auto-migrate VMs")
	}
	err = db.AutoMigrate(&Config{})
	if err != nil {
		panic("failed to auto-migrate Configs")
	}
	List.Mu.Lock()
	for _, vmInst := range GetAll() {
		List.VmList[vmInst.ID] = vmInst
	}
	List.Mu.Unlock()
}

func AutoStartVMs() {
	for _, vmInst := range List.VmList {
		if vmInst.Config.AutoStart {
			go func(aVmInst *VM) {
				err := aVmInst.Start()
				if err != nil {
					log.Printf("auto start of %v %v failed: %v", vmInst.ID, vmInst.Name, err)
				}
			}(vmInst)
		}
	}
}
func GetAll() []*VM {
	var result []*VM

	db := getVmDb()
	db.Preload("Config").Find(&result)

	return result
}

func GetByName(name string) (v *VM, err error) {
	defer List.Mu.Unlock()
	List.Mu.Lock()
	for _, t := range List.VmList {
		if t.Name == name {
			return t, nil
		}
	}
	return &VM{}, errors.New("not found")
}

func GetById(Id string) (v *VM, err error) {
	defer List.Mu.Unlock()
	List.Mu.Lock()
	vmInst, valid := List.VmList[Id]
	if valid {
		return vmInst, nil
	} else {
		return vmInst, errors.New("not found")
	}
}

func PrintVMStatus() {
	defer List.Mu.Unlock()
	List.Mu.Lock()
	for _, vmInst := range List.VmList {
		if vmInst.Status != RUNNING {
			log.Printf("vm: id: %v name: %v cpus: %v state: %v pid: %v", vmInst.ID, vmInst.Name, vmInst.Config.Cpu, vmInst.Status, nil)
		} else {
			if vmInst.proc == nil {
				setStopped(vmInst.ID)
				List.Mu.Lock()
				List.VmList[vmInst.ID].Status = STOPPED
				List.Mu.Unlock()
				vmInst.maybeForceKillVM()
				log.Printf("vm: id: %v name: %v cpus: %v state: %v pid: %v", vmInst.ID, vmInst.Name, vmInst.Config.Cpu, vmInst.Status, nil)
			} else {
				log.Printf("vm: id: %v name: %v cpus: %v state: %v pid: %v", vmInst.ID, vmInst.Name, vmInst.Config.Cpu, vmInst.Status, vmInst.proc.Pid())
			}
		}
	}
}

func GetRunningVMs() int {
	count := 0
	for _, vmInst := range List.VmList {
		if vmInst.Status == RUNNING {
			count += 1
		}
	}
	return count
}

func KillVMs() {
	for _, vmInst := range List.VmList {
		if vmInst.Status == RUNNING {
			go func(aVmInst *VM) {
				err := aVmInst.Stop()
				if err != nil {
					log.Printf("error stopping VM: %v", err)
				}
			}(vmInst)
		}
	}
}

func GetUsedVncPorts() []int32 {
	var ret []int32
	defer List.Mu.Unlock()
	List.Mu.Lock()
	for _, vmInst := range List.VmList {
		if vmInst.Status != STOPPED {
			ret = append(ret, vmInst.VNCPort)
		}
	}
	return ret
}

func IsVncPortUsed(vncPort int32) bool {
	usedVncPorts := GetUsedVncPorts()
	for _, port := range usedVncPorts {
		if port == vncPort {
			return true
		}
	}
	return false
}

func GetUsedNetPorts() []string {
	var ret []string
	defer List.Mu.Unlock()
	List.Mu.Lock()
	for _, vmInst := range List.VmList {
		if vmInst.Status != STOPPED {
			ret = append(ret, vmInst.NetDev)
		}
	}
	return ret
}

func IsNetPortUsed(netPort string) bool {
	usedNetPorts := GetUsedNetPorts()
	for _, port := range usedNetPorts {
		if port == netPort {
			return true
		}
	}
	return false
}
