package vm

import (
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
	"errors"
	"github.com/kontera-technologies/go-supervisor/v2"
	"github.com/tarm/serial"
	"golang.org/x/exp/slog"
	"gorm.io/gorm"
	"os"
	"sync"
	"time"
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
	Sound            bool   `gorm:"default:False;check:vnc_wait IN(0,1)"`
	SoundIn          string `gorm:"default:/dev/dsp0"`
	SoundOut         string `gorm:"default:/dev/dsp0"`
	Com1             bool   `gorm:"default:True;check:vnc_wait IN(0,1)"`
	Com1Dev          string `gorm:"default:AUTO"`
	Com1Log          bool   `gorm:"default:False;check:vnc_wait IN(0,1)"`
	Com2             bool   `gorm:"default:False;check:vnc_wait IN(0,1)"`
	Com2Dev          string `gorm:"default:AUTO"`
	Com2Log          bool   `gorm:"default:False;check:vnc_wait IN(0,1)"`
	Com3             bool   `gorm:"default:False;check:vnc_wait IN(0,1)"`
	Com3Dev          string `gorm:"default:AUTO"`
	Com3Log          bool   `gorm:"default:False;check:vnc_wait IN(0,1)"`
	Com4             bool   `gorm:"default:False;check:vnc_wait IN(0,1)"`
	Com4Dev          string `gorm:"default:AUTO"`
	Com4Log          bool   `gorm:"default:False;check:vnc_wait IN(0,1)"`
	ExtraArgs        string
	ISOs             string
	Disks            string
	Nics             string
	Com1Speed        uint32 `gorm:"default:115200;check:com1_speed IN(115200,57600,38400,19200,9600,4800,2400,1200,600,300,200,150,134,110,75,50)"`
	Com2Speed        uint32 `gorm:"default:115200;check:com1_speed IN(115200,57600,38400,19200,9600,4800,2400,1200,600,300,200,150,134,110,75,50)"`
	Com3Speed        uint32 `gorm:"default:115200;check:com1_speed IN(115200,57600,38400,19200,9600,4800,2400,1200,600,300,200,150,134,110,75,50)"`
	Com4Speed        uint32 `gorm:"default:115200;check:com1_speed IN(115200,57600,38400,19200,9600,4800,2400,1200,600,300,200,150,134,110,75,50)"`
	AutoStartDelay   uint32 `gorm:"default:0;check:auto_start_delay>=0"`
}

type VM struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null"`
	Name        string `gorm:"not null"`
	Description string
	Status      StatusType `gorm:"type:status_type"`
	BhyvePid    uint32     `gorm:"check:bhyve_pid>=0"`
	VNCPort     int32
	proc        *supervisor.Process
	mu          sync.Mutex
	log         slog.Logger
	Config      Config
	Com1Dev     string // TODO make a com struct and put these in it?
	Com2Dev     string
	Com3Dev     string
	Com4Dev     string
	Com1        *serial.Port `gorm:"-:all"`
	Com2        *serial.Port `gorm:"-:all"`
	Com3        *serial.Port `gorm:"-:all"`
	Com4        *serial.Port `gorm:"-:all"`
	Com1lock    sync.Mutex   `gorm:"-:all"`
	Com2lock    sync.Mutex   `gorm:"-:all"`
	Com3lock    sync.Mutex   `gorm:"-:all"`
	Com4lock    sync.Mutex   `gorm:"-:all"`
	Com1rchan   chan byte    `gorm:"-:all"`
	Com1write   bool         `gorm:"-:all"`
	Com2rchan   chan byte    `gorm:"-:all"`
	Com2write   bool         `gorm:"-:all"`
	Com3rchan   chan byte    `gorm:"-:all"`
	Com3write   bool         `gorm:"-:all"`
	Com4rchan   chan byte    `gorm:"-:all"`
	Com4write   bool         `gorm:"-:all"`
}

type ListType struct {
	Mu     sync.Mutex
	VmList map[string]*VM
}

var List = ListType{
	VmList: map[string]*VM{},
}

func GetVmLogPath(logpath string) error {
	ex, err := util.PathExists(logpath)
	if err != nil {
		return err
	}
	if !ex {
		err := os.MkdirAll(logpath, 0755)
		if err != nil {
			return err
		}
	}
	return nil
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
		InitOneVm(vmInst)
	}
	List.Mu.Unlock()
}

func InitOneVm(vmInst *VM) {
	vmLogPath := config.Config.Disk.VM.Path.State + "/" + vmInst.Name
	err := GetVmLogPath(vmLogPath)
	if err != nil {
		panic(err)
	}
	vmLogFilePath := vmLogPath + "/log"
	vmLogFile, err := os.OpenFile(vmLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("failed to open VM log file", "err", err)
	}
	var programLevel = new(slog.LevelVar) // Info by default
	vmLogger := slog.New(slog.HandlerOptions{Level: programLevel}.NewTextHandler(vmLogFile))

	vmInst.log = *vmLogger

	if config.Config.Log.Level == "info" {
		vmInst.log.Info("log level set to info")
		programLevel.Set(slog.LevelInfo)
	} else if config.Config.Log.Level == "debug" {
		vmInst.log.Info("log level set to debug")
		programLevel.Set(slog.LevelDebug)
	} else {
		programLevel.Set(slog.LevelInfo)
		vmInst.log.Info("log level not set or un-parseable, setting to info")
	}

	List.VmList[vmInst.ID] = vmInst
	vmInst.log.Debug("vm init", "id", vmInst.ID, "isos", vmInst.Config.ISOs, "disks", vmInst.Config.Disks)
}

func AutoStartVMs() {
	for _, vmInst := range List.VmList {
		if vmInst.Config.AutoStart {
			go doAutostart(vmInst)
		}
	}
}

func doAutostart(vmInst *VM) {
	func(aVmInst *VM) {
		slog.Debug(
			"AutoStartVMs sleeping for auto start delay",
			"vm", aVmInst.Name,
			"auto_start_delay", aVmInst.Config.AutoStartDelay,
		)
		time.Sleep(time.Duration(aVmInst.Config.AutoStartDelay) * time.Second)
		err := aVmInst.Start()
		if err != nil {
			slog.Error("auto start failed", "vm", vmInst.ID, "name", vmInst.Name, "err", err)
		}
	}(vmInst)
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
			slog.Info("vm",
				"id", vmInst.ID,
				"name", vmInst.Name,
				"cpus", vmInst.Config.Cpu,
				"state", vmInst.Status,
				"pid", nil,
			)
		} else {
			if vmInst.proc == nil {
				setStopped(vmInst.ID)
				List.Mu.Lock()
				List.VmList[vmInst.ID].Status = STOPPED
				List.Mu.Unlock()
				vmInst.maybeForceKillVM()
				slog.Info("vm",
					"id", vmInst.ID,
					"name", vmInst.Name,
					"cpus", vmInst.Config.Cpu,
					"state", vmInst.Status,
					"pid", nil,
				)
			} else {
				slog.Info("vm",
					"id", vmInst.ID,
					"name", vmInst.Name,
					"cpus", vmInst.Config.Cpu,
					"state", vmInst.Status,
					"pid", vmInst.proc.Pid(),
				)
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
					slog.Error("error stopping VM", "err", err)
				}
			}(vmInst)
		}
	}
}

func GetUsedVncPorts() []int {
	var ret []int
	defer List.Mu.Unlock()
	List.Mu.Lock()
	for _, vmInst := range List.VmList {
		if vmInst.Status != STOPPED {
			ret = append(ret, int(vmInst.VNCPort))
		}
	}
	return ret
}

func GetUsedNetPorts() []string {
	var ret []string
	defer List.Mu.Unlock()
	List.Mu.Lock()
	for _, vmInst := range List.VmList {
		vmNicsList, err := vmInst.GetNics()
		if err != nil {
			slog.Error("GetUsedNetPorts failed to get nics", "err", err)
		}
		for _, vmNic := range vmNicsList {
			ret = append(ret, vmNic.NetDev)
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
