package vm

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kontera-technologies/go-supervisor/v2"
	"github.com/tarm/serial"
	exec "golang.org/x/sys/execabs"
	"gorm.io/gorm"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/epair"
	"cirrina/cirrinad/iso"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vmnic"
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
	VMID             string
	CPU              uint32 `gorm:"default:1;check:cpu>=1"`
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
	WireGuestMem     bool   `gorm:"default:False;check:wire_guest_mem IN (0,1)"`
	DestroyPowerOff  bool   `gorm:"default:True;check:destroy_power_off IN (0,1)"`
	IgnoreUnknownMSR bool   `gorm:"default:True;check:ignore_unknown_msr IN (0,1)"`
	KbdLayout        string `gorm:"default:default"`
	AutoStart        bool   `gorm:"default:False;check:auto_start IN (0,1)"`
	Sound            bool   `gorm:"default:False;check:sound IN(0,1)"`
	SoundIn          string `gorm:"default:/dev/dsp0"`
	SoundOut         string `gorm:"default:/dev/dsp0"`
	Com1             bool   `gorm:"default:True;check:com1 IN(0,1)"`
	Com1Dev          string `gorm:"default:AUTO"`
	Com1Log          bool   `gorm:"default:False;check:com1_log IN(0,1)"`
	Com2             bool   `gorm:"default:False;check:com2 IN(0,1)"`
	Com2Dev          string `gorm:"default:AUTO"`
	Com2Log          bool   `gorm:"default:False;check:com2_log IN(0,1)"`
	Com3             bool   `gorm:"default:False;check:com3 IN(0,1)"`
	Com3Dev          string `gorm:"default:AUTO"`
	Com3Log          bool   `gorm:"default:False;check:com3_log IN(0,1)"`
	Com4             bool   `gorm:"default:False;check:com4 IN(0,1)"`
	Com4Dev          string `gorm:"default:AUTO"`
	Com4Log          bool   `gorm:"default:False;check:com4_log IN(0,1)"`
	ExtraArgs        string
	ISOs             string
	Disks            string
	Com1Speed        uint32       `gorm:"default:115200;check:com1_speed IN(115200,57600,38400,19200,9600,4800,2400,1200,600,300,200,150,134,110,75,50)"` //nolint:lll
	Com2Speed        uint32       `gorm:"default:115200;check:com2_speed IN(115200,57600,38400,19200,9600,4800,2400,1200,600,300,200,150,134,110,75,50)"` //nolint:lll
	Com3Speed        uint32       `gorm:"default:115200;check:com3_speed IN(115200,57600,38400,19200,9600,4800,2400,1200,600,300,200,150,134,110,75,50)"` //nolint:lll
	Com4Speed        uint32       `gorm:"default:115200;check:com4_speed IN(115200,57600,38400,19200,9600,4800,2400,1200,600,300,200,150,134,110,75,50)"` //nolint:lll
	AutoStartDelay   uint32       `gorm:"default:0;check:auto_start_delay>=0"`
	Debug            bool         `gorm:"default:False;check:debug IN(0,1)"`
	DebugWait        bool         `gorm:"default:False;check:debug_wait IN(0,1)"`
	DebugPort        string       `gorm:"default:AUTO"`
	Priority         int32        `gorm:"default:0;check:priority BETWEEN -20 and 20"`
	Protect          sql.NullBool `gorm:"default:True;check:protect IN(0,1)"`
	Pcpu             uint32
	Rbps             uint32
	Wbps             uint32
	Riops            uint32
	Wiops            uint32
}

type VM struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null;default:null"`
	Name        string `gorm:"not null"`
	Description string
	Status      StatusType `gorm:"type:status_type"`
	BhyvePid    uint32     `gorm:"check:bhyve_pid>=0"`
	VNCPort     int32
	DebugPort   int32
	proc        *supervisor.Process
	mu          sync.RWMutex
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
	Mu     sync.RWMutex
	VMList map[string]*VM
}

var vmStartLock sync.Mutex
var List = &ListType{
	VMList: make(map[string]*VM),
}

func Create(name string, description string, cpu uint32, mem uint32) (vm *VM, err error) {
	var vmInst *VM
	if !util.ValidVMName(name) {
		return vmInst, errVMInvalidName
	}
	if _, err := GetByName(name); err == nil {
		return vmInst, errVMDupe
	}
	vmInst = &VM{
		Name:        name,
		Status:      STOPPED,
		Description: description,
		Config: Config{
			CPU: cpu,
			Mem: mem,
		},
	}
	defer List.Mu.Unlock()
	List.Mu.Lock()
	db := GetVMDB()
	slog.Debug("Creating VM", "vm", name)
	res := db.Create(&vmInst)
	InitOneVM(vmInst)

	return vmInst, res.Error
}

func (vm *VM) Delete() (err error) {
	db := GetVMDB()
	db.Model(&VM{}).Preload("Config").Limit(1).Find(&vm, &VM{ID: vm.ID})
	if vm.ID == "" {
		return errVMNotFound
	}
	res := db.Limit(1).Delete(&vm.Config)
	if res.RowsAffected != 1 {
		// don't fail deleting the VM, may have a bad or missing config, still want to be able to delete VM
		slog.Error("failed to delete config for VM", "vmid", vm.ID)
	}
	res = db.Limit(1).Delete(&vm)
	if res.RowsAffected != 1 {
		slog.Error("error deleting VM", "res", res)

		return errVMInternalDB
	}

	return nil
}

func (vm *VM) Start() (err error) {
	defer vmStartLock.Unlock()
	vmStartLock.Lock()
	if vm.Status != STOPPED {
		return errVMNotStopped
	}
	vm.SetStarting()
	events := make(chan supervisor.Event)
	err = vm.lockDisks()
	if err != nil {
		slog.Error("Failed locking disks", "err", err)

		return err
	}

	cmdName, cmdArgs := vm.generateCommandLine()
	vm.log.Info("start", "cmd", cmdName, "args", cmdArgs)
	vm.createUefiVarsFile()
	vm.netStartup()
	err = vm.Save()
	if err != nil {
		slog.Error("Failed saving VM", "err", err)

		return err
	}

	respawnWait := time.Duration(vm.Config.RestartDelay) * time.Second
	// avoid go-supervisor setting this to default (2m) -- 1ns is hard to differentiate from 0ns and I prefer not to
	// change go-supervisor unless I have to
	if respawnWait == 0 {
		respawnWait = 1
	}

	var processDebug bool
	if config.Config.Log.Level == "debug" {
		slog.Debug("vm.Start enabling process debugging", "vm", vm.Name)
		processDebug = true
	}

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
		RespawnWait:             respawnWait,
		SpawnWait:               respawnWait,
		MaxInterruptAttempts:    1,
		MaxTerminateAttempts:    1,
		IdleTimeout:             -1,
		TerminationGraceTimeout: time.Duration(vm.Config.MaxWait) * time.Second,
		Debug:                   processDebug,
	})

	vm.proc = p
	go vmDaemon(events, vm)

	if err := p.Start(); err != nil {
		panic(fmt.Sprintf("failed to start process: %s", err))
	}

	return nil
}

func (vm *VM) Stop() (err error) {
	if vm.Status == STOPPED {
		slog.Error("tried to stop VM already stopped", "vm", vm.Name)

		return errVMAlreadyStopped
	}
	vm.SetStopping()
	if vm.proc == nil {
		vm.SetStopped()

		return nil
	}
	err = vm.proc.Stop()
	if err != nil {
		slog.Error("Failed to stop VM", "vm", vm.Name, "pid", vm.proc.Pid(), "err", err)

		return errVMStopFail
	}

	return nil
}

func (vm *VM) Save() error {
	db := GetVMDB()

	res := db.Model(&vm.Config).
		Updates(map[string]interface{}{
			"cpu":                &vm.Config.CPU,
			"mem":                &vm.Config.Mem,
			"max_wait":           &vm.Config.MaxWait,
			"restart":            &vm.Config.Restart,
			"restart_delay":      &vm.Config.RestartDelay,
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
			"kbd_layout":         &vm.Config.KbdLayout,
			"auto_start":         &vm.Config.AutoStart,
			"sound":              &vm.Config.Sound,
			"sound_in":           &vm.Config.SoundIn,
			"sound_out":          &vm.Config.SoundOut,
			"Com1":               &vm.Config.Com1,
			"com1_dev":           &vm.Config.Com1Dev,
			"Com2":               &vm.Config.Com2,
			"com2_dev":           &vm.Config.Com2Dev,
			"Com3":               &vm.Config.Com3,
			"com3_dev":           &vm.Config.Com3Dev,
			"com4":               &vm.Config.Com4,
			"com4_dev":           &vm.Config.Com4Dev,
			"extra_args":         &vm.Config.ExtraArgs,
			"is_os":              &vm.Config.ISOs,
			"disks":              &vm.Config.Disks,
			"com1_log":           &vm.Config.Com1Log,
			"com2_log":           &vm.Config.Com2Log,
			"com3_log":           &vm.Config.Com3Log,
			"com4_log":           &vm.Config.Com4Log,
			"com1_speed":         &vm.Config.Com1Speed,
			"com2_speed":         &vm.Config.Com2Speed,
			"com3_speed":         &vm.Config.Com3Speed,
			"com4_speed":         &vm.Config.Com4Speed,
			"auto_start_delay":   &vm.Config.AutoStartDelay,
			"debug":              &vm.Config.Debug,
			"debug_wait":         &vm.Config.DebugWait,
			"debug_port":         &vm.Config.DebugPort,
			"priority":           &vm.Config.Priority,
			"protect":            &vm.Config.Protect,
			"pcpu":               &vm.Config.Pcpu,
			"rbps":               &vm.Config.Rbps,
			"wbps":               &vm.Config.Wbps,
			"riops":              &vm.Config.Riops,
			"wiops":              &vm.Config.Wiops,
		},
		)

	if res.Error != nil {
		slog.Error("error updating VM", "res", res)

		return fmt.Errorf("error updating VM: %w", res.Error)
	}

	res = db.Select([]string{
		"name",
		"description",
		"net_dev",
		"vnc_port",
		"debug_port",
		"com1_dev",
		"com2_dev",
		"com3_dev",
		"com4_dev",
	}).Model(&vm).
		Updates(map[string]interface{}{
			"name":        &vm.Name,
			"description": &vm.Description,
			"vnc_port":    &vm.VNCPort,
			"debug_port":  &vm.DebugPort,
			"com1_dev":    &vm.Com1Dev,
			"com2_dev":    &vm.Com2Dev,
			"com3_dev":    &vm.Com3Dev,
			"com4_dev":    &vm.Com4Dev,
		})

	if res.Error != nil {
		slog.Error("error updating VM", "res", res)

		return fmt.Errorf("error updating VM: %w", res.Error)
	}

	return nil
}

func (vm *VM) MaybeForceKillVM() {
	ex, err := util.PathExists("/dev/vmm/" + vm.Name)
	if err != nil {
		return
	}
	if !ex {
		return
	}
	args := []string{"/usr/sbin/bhyvectl", "--destroy"}
	args = append(args, "--vm="+vm.Name)
	cmd := exec.Command(config.Config.Sys.Sudo, args...)
	_ = cmd.Run()
}

func (vm *VM) createUefiVarsFile() {
	uefiVarsFilePath := config.Config.Disk.VM.Path.State + "/" + vm.Name
	uefiVarsFile := uefiVarsFilePath + "/BHYVE_UEFI_VARS.fd"
	uvPathExists, err := util.PathExists(uefiVarsFilePath)
	if err != nil {
		return
	}
	if !uvPathExists {
		err = os.Mkdir(uefiVarsFilePath, 0755)
		if err != nil {
			slog.Error("failed to create uefi vars path", "err", err)

			return
		}
	}
	uvFileExists, err := util.PathExists(uefiVarsFile)
	if err != nil {
		return
	}
	if !uvFileExists {
		_, err = util.CopyFile(config.Config.Rom.Vars.Template, uefiVarsFile)
		if err != nil {
			slog.Error("failed to copy uefiVars template", "err", err)
		}
	}
}

func netStartupIf(vmNic vmnic.VMNic) error {
	// Create interface
	args := []string{"/sbin/ifconfig", vmNic.NetDev, "create", "group", "cirrinad"}
	cmd := exec.Command(config.Config.Sys.Sudo, args...)
	err := cmd.Run()
	if err != nil {
		slog.Error("failed to create tap", "err", err)

		return fmt.Errorf("error running ifconfig command: %w", err)
	}

	if vmNic.SwitchID == "" {
		return nil
	}
	// Add interface to bridge
	thisSwitch, err := _switch.GetByID(vmNic.SwitchID)
	if err != nil {
		slog.Error("bad switch id",
			"nicname", vmNic.Name, "nicid", vmNic.ID, "switchid", vmNic.SwitchID)

		return fmt.Errorf("error getting switch id: %w", err)
	}
	if thisSwitch.Type != "IF" {
		slog.Error("bridge/interface type mismatch",
			"nicname", vmNic.Name,
			"nicid", vmNic.ID,
			"switchid", vmNic.SwitchID,
		)

		return errVMSwitchNICMismatch
	}
	if vmNic.RateLimit {
		thisEpair, err := setupVMNicRateLimit(vmNic)
		if err != nil {
			return err
		}
		err = _switch.BridgeIfAddMember(thisSwitch.Name, thisEpair+"b", true)
		if err != nil {
			slog.Error("failed to add nic to switch",
				"nicname", vmNic.Name,
				"nicid", vmNic.ID,
				"switchid", vmNic.SwitchID,
				"netdev", vmNic.NetDev,
				"err", err,
			)

			return fmt.Errorf("error adding member to bridge: %w", err)
		}
	} else {
		// mac := GetMac(vmNic, vm)
		err := _switch.BridgeIfAddMember(thisSwitch.Name, vmNic.NetDev, false)
		if err != nil {
			slog.Error("failed to add nic to switch",
				"nicname", vmNic.Name,
				"nicid", vmNic.ID,
				"switchid", vmNic.SwitchID,
				"netdev", vmNic.NetDev,
				"err", err,
			)

			return fmt.Errorf("error adding member to bridge: %w", err)
		}
	}

	return nil
}

func setupVMNicRateLimit(vmNic vmnic.VMNic) (string, error) {
	var err error
	thisEpair := epair.GetDummyEpairName()
	slog.Debug("netStartup rate limiting", "thisEpair", thisEpair)
	err = epair.CreateEpair(thisEpair)
	if err != nil {
		slog.Error("error creating epair", err)

		return "", fmt.Errorf("error creating epair: %w", err)
	}
	vmNic.InstEpair = thisEpair
	err = vmNic.Save()
	if err != nil {
		slog.Error("failed to save net dev", "nic", vmNic.ID, "netdev", vmNic.NetDev)

		return "", fmt.Errorf("error saving NIC: %w", err)
	}
	err = epair.SetRateLimit(thisEpair, vmNic.RateIn, vmNic.RateOut)
	if err != nil {
		slog.Error("failed to set epair rate limit", "epair", thisEpair)

		return "", fmt.Errorf("error setting rate limit: %w", err)
	}
	thisInstSwitch := _switch.GetDummyBridgeName()
	var bridgeMembers []string
	bridgeMembers = append(bridgeMembers, thisEpair+"a")
	bridgeMembers = append(bridgeMembers, vmNic.NetDev)
	err = _switch.CreateIfBridgeWithMembers(thisInstSwitch, bridgeMembers)
	if err != nil {
		slog.Error("failed to create switch",
			"nic", vmNic.ID,
			"thisInstSwitch", thisInstSwitch,
			"err", err,
		)

		return "", fmt.Errorf("error creating bridge: %w", err)
	}
	vmNic.InstBridge = thisInstSwitch
	err = vmNic.Save()
	if err != nil {
		slog.Error("failed to save net dev", "nic", vmNic.ID, "netdev", vmNic.NetDev)

		return "", fmt.Errorf("error saving NIC: %w", err)
	}

	return thisEpair, nil
}

func netStartupNg(vmNic vmnic.VMNic) error {
	thisSwitch, err := _switch.GetByID(vmNic.SwitchID)
	if err != nil {
		slog.Error("bad switch id",
			"nicname", vmNic.Name, "nicid", vmNic.ID, "switchid", vmNic.SwitchID)

		return fmt.Errorf("error getting switch ID: %w", err)
	}
	if thisSwitch.Type != "NG" {
		slog.Error("bridge/interface type mismatch",
			"nicname", vmNic.Name,
			"nicid", vmNic.ID,
			"switchid", vmNic.SwitchID,
		)

		return errVMSwitchNICMismatch
	}

	return nil
}

func (vm *VM) netStartup() {
	vmNicsList, err := vm.GetNics()

	if err != nil {
		slog.Error("netStartup failed to get nics", "err", err)

		return
	}

	for _, vmNic := range vmNicsList {
		switch {
		case vmNic.NetDevType == "TAP" || vmNic.NetDevType == "VMNET":
			err := netStartupIf(vmNic)
			if err != nil {
				slog.Error("error bringing up nic", "err", err)

				continue
			}
		case vmNic.NetDevType == "NETGRAPH":
			err := netStartupNg(vmNic)
			if err != nil {
				slog.Error("error bringing up nic", "err", err)

				continue
			}
		default:
			slog.Debug("unknown net type, can't set up")

			continue
		}
	}
}

func (vm *VM) lockDisks() error {
	vmDisks, err := vm.GetDisks()
	if err != nil {
		return err
	}
	for _, vmDisk := range vmDisks {
		vmDisk.Lock()
	}

	return nil
}

func (vm *VM) unlockDisks() error {
	vmDisks, err := vm.GetDisks()
	if err != nil {
		return err
	}
	for _, vmDisk := range vmDisks {
		vmDisk.Unlock()
	}

	return nil
}

func (vm *VM) applyResourceLimits(vmPid string) {
	if vm.proc == nil || vm.proc.Pid() == 0 || vm.BhyvePid == 0 {
		slog.Error("attempted to apply resource limits to vm that may not be running")

		return
	}
	vm.log.Debug("checking resource limits")
	// vm.proc.Pid aka vm.BhyvePid is actually the sudo proc that's the parent of bhyve
	// call pgrep to get the child (bhyve) -- life would be so much easier if we could run bhyve as non-root
	// should fix supervisor to use int32
	if vm.Config.Pcpu > 0 {
		vm.log.Debug("Setting pcpu limit")
		cpuLimitStr := strconv.FormatUint(uint64(vm.Config.Pcpu), 10)
		args := []string{"/usr/bin/rctl", "-a", "process:" + vmPid + ":pcpu:deny=" + cpuLimitStr}
		cmd := exec.Command(config.Config.Sys.Sudo, args...)
		err := cmd.Run()
		if err != nil {
			slog.Error("failed to set resource limit", "err", err)
		}
	}
	if vm.Config.Rbps > 0 {
		vm.log.Debug("Setting rbps limit")
		rbpsLimitStr := strconv.FormatUint(uint64(vm.Config.Rbps), 10)
		args := []string{"/usr/bin/rctl", "-a", "process:" + vmPid + ":readbps:throttle=" + rbpsLimitStr}
		cmd := exec.Command(config.Config.Sys.Sudo, args...)
		err := cmd.Run()
		if err != nil {
			slog.Error("failed to set resource limit", "err", err)
		}
	}
	if vm.Config.Wbps > 0 {
		vm.log.Debug("Setting wbps limit")
		wbpsLimitStr := strconv.FormatUint(uint64(vm.Config.Wbps), 10)
		args := []string{"/usr/bin/rctl", "-a", "process:" + vmPid + ":writebps:throttle=" + wbpsLimitStr}
		cmd := exec.Command(config.Config.Sys.Sudo, args...)
		err := cmd.Run()
		if err != nil {
			slog.Error("failed to set resource limit", "err", err)
		}
	}
	if vm.Config.Riops > 0 {
		vm.log.Debug("Setting riops limit")
		riopsLimitStr := strconv.FormatUint(uint64(vm.Config.Riops), 10)
		args := []string{"/usr/bin/rctl", "-a", "process:" + vmPid + ":readiops:throttle=" + riopsLimitStr}
		cmd := exec.Command(config.Config.Sys.Sudo, args...)
		err := cmd.Run()
		if err != nil {
			slog.Error("failed to set resource limit", "err", err)
		}
	}
	if vm.Config.Wiops > 0 {
		vm.log.Debug("Setting wiops limit")
		wiopsLimitStr := strconv.FormatUint(uint64(vm.Config.Wiops), 10)
		args := []string{"/usr/bin/rctl", "-a", "process:" + vmPid + ":writeiops:throttle=" + wiopsLimitStr}
		cmd := exec.Command(config.Config.Sys.Sudo, args...)
		err := cmd.Run()
		if err != nil {
			slog.Error("failed to set resource limit", "err", err)
		}
	}
}

// NetCleanup clean up all of a VMs nics
func (vm *VM) NetCleanup() {
	vmNicsList, err := vm.GetNics()
	if err != nil {
		slog.Error("failed to get nics", "err", err)

		return
	}
	for _, vmNic := range vmNicsList {
		switch {
		case vmNic.NetDevType == "TAP" || vmNic.NetDevType == "VMNET":
			err = cleanupIfNic(vmNic)
			if err != nil {
				slog.Error("error cleaning up nic", "vmNic", vmNic, "err", err)
			}
		case vmNic.NetDevType == "NETGRAPH":
			// nothing to do for netgraph
		default:
			slog.Error("unknown net type, can't clean up")
		}
		vmNic.NetDev = ""
		vmNic.InstEpair = ""
		vmNic.InstBridge = ""
		err = vmNic.Save()
		if err != nil {
			slog.Error("failed to save net dev", "nic", vmNic.ID, "netdev", vmNic.NetDev)
		}
	}
}

func vmDaemon(events chan supervisor.Event, vm *VM) {
	for {
		select {
		case msg := <-vm.proc.Stdout():
			vm.log.Info("output", "stdout", *msg)
		case msg := <-vm.proc.Stderr():
			vm.log.Info("output", "stderr", *msg)
		case event := <-events:
			switch event.Code {
			case "ProcessStart":
				vm.log.Info("event", "code", event.Code, "message", event.Message)
				vm.SetRunning(vm.proc.Pid())
				vmPid := strconv.FormatInt(int64(findChildPid(uint32(vm.proc.Pid()))), 10)
				slog.Debug("vmDaemon ProcessStart", "bhyvePid", vm.BhyvePid, "sudoPid", vm.proc.Pid(), "realPid", vmPid)
				vm.setupComLoggers()
				vm.applyResourceLimits(vmPid)
			case "ProcessDone":
				vm.log.Info("event", "code", event.Code, "message", event.Message)
			case "ProcessCrashed":
				vm.log.Info("exited, destroying")
				vm.MaybeForceKillVM()
			default:
				vm.log.Info("event", "code", event.Code, "message", event.Message)
			}
		case <-vm.proc.DoneNotifier():
			slog.Debug("vm stopped",
				"vm_name", vm.Name,
			)
			vm.log.Info("stopped")
			vm.NetCleanup()
			vm.killComLoggers()
			vm.SetStopped()
			err := vm.unlockDisks()
			if err != nil {
				slog.Debug("failed unlock disks", "err", err)

				return
			}
			vm.MaybeForceKillVM()
			vm.log.Info("closing loop we are done")

			return
		}
	}
}

func (vm *VM) GetISOs() ([]iso.ISO, error) {
	var isos []iso.ISO
	// TODO remove all these de-normalizations in favor of gorm native "Has Many" relationships
	for _, cv := range strings.Split(vm.Config.ISOs, ",") {
		if cv == "" {
			continue
		}
		aISO, err := iso.GetByID(cv)
		if err == nil {
			isos = append(isos, *aISO)
		} else {
			slog.Error("bad iso", "iso", cv, "vm", vm.ID)
		}
	}

	return isos, nil
}

func (vm *VM) GetNics() ([]vmnic.VMNic, error) {
	nics := vmnic.GetNics(vm.Config.ID)

	return nics, nil
}

func (vm *VM) GetDisks() ([]*disk.Disk, error) {
	var disks []*disk.Disk
	// TODO remove all these de-normalizations in favor of gorm native "Has Many" relationships
	for _, cv := range strings.Split(vm.Config.Disks, ",") {
		if cv == "" {
			continue
		}
		aDisk, err := disk.GetByID(cv)
		if err == nil {
			disks = append(disks, aDisk)
		} else {
			slog.Error("bad disk", "disk", cv, "vm", vm.ID)
		}
	}

	return disks, nil
}

func (vm *VM) DeleteUEFIState() error {
	uefiVarsFilePath := config.Config.Disk.VM.Path.State + "/" + vm.Name
	uefiVarsFile := uefiVarsFilePath + "/BHYVE_UEFI_VARS.fd"
	uvFileExists, err := util.PathExists(uefiVarsFile)
	if err != nil {
		return fmt.Errorf("error checking if UEFI state file exists: %w", err)
	}
	if uvFileExists {
		if err := os.Remove(uefiVarsFile); err != nil {
			return fmt.Errorf("error removing UEFI state file: %w", err)
		}
	}

	return nil
}

func (vm *VM) AttachIsos(isoIDs []string) error {
	defer vm.mu.Unlock()
	vm.mu.Lock()

	if vm.Status != STOPPED {
		return errVMNotStopped
	}

	for _, aIso := range isoIDs {
		slog.Debug("checking iso exists", "iso", aIso)

		isoUUID, err := uuid.Parse(aIso)
		if err != nil {
			return errVMIsoInvalid
		}

		thisIso, err := iso.GetByID(isoUUID.String())
		if err != nil {
			slog.Error("error getting disk", "disk", aIso, "err", err)

			return errVMIsoNotFound
		}
		if thisIso.Name == "" {
			return errVMIsoNotFound
		}
	}

	var isoConfigVal string
	count := 0

	for _, isoID := range isoIDs {
		if count > 0 {
			isoConfigVal += ","
		}
		isoConfigVal += isoID
		count++
	}
	vm.Config.ISOs = isoConfigVal
	err := vm.Save()
	if err != nil {
		slog.Error("error saving VM", "err", err)

		return err
	}

	return nil
}

// SetNics sets the list of nics attached to a VM to the list passed in
func (vm *VM) SetNics(nicIDs []string) error {
	defer vm.mu.Unlock()
	vm.mu.Lock()
	if vm.Status != STOPPED {
		return errVMNotStopped
	}

	// remove all nics from VM
	err := removeAllNicsFromVM(vm)
	if err != nil {
		return err
	}

	// check that these nics can be attached to this VM
	err = validateNics(nicIDs, vm)
	if err != nil {
		return err
	}

	// add the nics
	for _, nicID := range nicIDs {
		vmNic, err := vmnic.GetByID(nicID)
		if err != nil {
			slog.Error("error looking up nic", "err", err)

			return fmt.Errorf("error getting NIC: %w", err)
		}

		vmNic.ConfigID = vm.Config.ID
		err = vmNic.Save()
		if err != nil {
			slog.Error("error saving NIC", "err", err)

			return fmt.Errorf("error saving NIC: %w", err)
		}
	}

	return nil
}

func (vm *VM) AttachDisks(diskids []string) error {
	defer vm.mu.Unlock()
	vm.mu.Lock()
	if vm.Status != STOPPED {
		return errVMNotStopped
	}

	err := validateDisks(diskids, vm)
	if err != nil {
		return err
	}

	// build disk list string to put into DB
	var disksConfigVal string
	count := 0
	for _, diskID := range diskids {
		if count > 0 {
			disksConfigVal += ","
		}
		disksConfigVal += diskID
		count++
	}
	vm.Config.Disks = disksConfigVal
	err = vm.Save()
	if err != nil {
		slog.Error("error saving VM", "err", err)

		return err
	}

	return nil
}

func (vm *VM) killComLoggers() {
	slog.Debug("killing com loggers")
	var err error

	// change to range when moving to Go 1.22
	for comNum := 1; comNum <= 4; comNum++ {
		err = vm.killCom(comNum)
		if err != nil {
			slog.Error("com kill error", "comNum", 1, "err", err)
			// no need to return error here either
		}
	}
}

func (vm *VM) setupComLoggers() {
	var err error

	// change to range when moving to Go 1.22
	for comNum := 1; comNum <= 4; comNum++ {
		err = vm.setupCom(comNum)
		if err != nil {
			slog.Error("com setup error", "comNum", comNum, "err", err)
			// not returning error since we leave the VM running and hope for the best
		}
	}
}

func comLogger(vm *VM, comNum int) {
	var thisCom *serial.Port
	var thisRChan chan byte
	comChan := make(chan byte, 4096)

	switch comNum {
	case 1:
		thisCom = vm.Com1
		if vm.Config.Com1Log {
			vm.Com1rchan = comChan
			thisRChan = vm.Com1rchan
		}
	case 2:
		thisCom = vm.Com2
		if vm.Config.Com2Log {
			vm.Com2rchan = comChan
			thisRChan = vm.Com2rchan
		}
	case 3:
		thisCom = vm.Com3
		if vm.Config.Com3Log {
			vm.Com3rchan = comChan
			thisRChan = vm.Com3rchan
		}
	case 4:
		thisCom = vm.Com4
		if vm.Config.Com4Log {
			vm.Com4rchan = comChan
			thisRChan = vm.Com4rchan
		}
	default:
		slog.Error("comLogger invalid com", "comNum", comNum)

		return
	}

	comLogPath := config.Config.Disk.VM.Path.State + "/" + vm.Name + "/"
	comLogFile := comLogPath + "com" + strconv.Itoa(comNum) + "_out.log"
	err := GetVMLogPath(comLogPath)
	if err != nil {
		slog.Error("setupComLoggers", "err", err)

		return
	}

	vl, err := os.OpenFile(comLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("failed to open VM output log file", "filename", comLogFile, "err", err)
	}
	defer func(vl *os.File) {
		_ = vl.Close()
	}(vl)

	for {
		if comLoggerRead(vm, comNum, thisCom, vl, thisRChan) {
			return
		}
	}
}

func (vm *VM) Running() bool {
	if vm.Status == RUNNING || vm.Status == STOPPING {
		return true
	}

	return false
}

func (vm *VM) GetComWrite(comNum int) bool {
	var thisComChanWriteFlag bool
	switch comNum {
	case 1:
		thisComChanWriteFlag = vm.Com1write
	case 2:
		thisComChanWriteFlag = vm.Com2write
	case 3:
		thisComChanWriteFlag = vm.Com3write
	case 4:
		thisComChanWriteFlag = vm.Com4write
	}

	return thisComChanWriteFlag
}

func comLoggerRead(vm *VM, comNum int, thisCom *serial.Port, vl *os.File, thisRChan chan byte) bool {
	var thisComChanWriteFlag bool
	b := make([]byte, 1)
	b2 := make([]byte, 1)

	if !vm.Running() {
		slog.Debug("comLogger vm not running, exiting2",
			"vm_id", vm.ID,
			"comNum", comNum,
			"vm.Status", vm.Status,
		)

		return true
	}
	if thisCom == nil {
		slog.Error("comLogger", "msg", "unable to read nil port")

		return true
	}

	thisComChanWriteFlag = vm.GetComWrite(comNum)

	nb, err := thisCom.Read(b)
	if nb > 1 {
		slog.Error("comLogger read more than 1 byte", "nb", nb)
	}
	if errors.Is(err, io.EOF) && !vm.Running() {
		slog.Debug("comLogger vm not running, exiting",
			"vm_id", vm.ID,
			"comNum", comNum,
			"vm.Status", vm.Status,
		)

		return true
	}
	if err != nil && !errors.Is(err, io.EOF) {
		slog.Error("comLogger", "error reading", err)

		return true
	}
	if nb != 0 {
		// write to log file
		_, err = vl.Write(b)

		// write to channel used by remote users, if someone is reading from it
		if thisRChan != nil && thisComChanWriteFlag {
			nb2 := copy(b2, b)
			if nb != nb2 {
				slog.Error("comLogger", "msg", "some bytes lost")
			}
			thisRChan <- b2[0]
		}

		if err != nil {
			slog.Error("comLogger", "error writing", err)

			return true
		}
	}

	return false
}

func (vm *VM) killCom(comNum int) error {
	switch comNum {
	case 1:
		if vm.Com1 != nil {
			_ = vm.Com1.Close()
			vm.Com1 = nil
		}
		if vm.Com1rchan != nil {
			close(vm.Com1rchan)
			vm.Com1rchan = nil
		}
	case 2:
		if vm.Com2 != nil {
			_ = vm.Com2.Close()
			vm.Com2 = nil
		}
		if vm.Com2rchan != nil {
			close(vm.Com2rchan)
			vm.Com2rchan = nil
		}
	case 3:
		if vm.Com3 != nil {
			_ = vm.Com3.Close()
			vm.Com3 = nil
		}
		if vm.Com3rchan != nil {
			close(vm.Com3rchan)
			vm.Com3rchan = nil
		}
	case 4:
		if vm.Com4 != nil {
			_ = vm.Com4.Close()
			vm.Com4 = nil
		}
		if vm.Com4rchan != nil {
			close(vm.Com4rchan)
			vm.Com4rchan = nil
		}
	default:
		slog.Error("invalid com port number", "comNum", comNum)

		return errVMComInvalid
	}

	return nil
}

func (vm *VM) setupCom(comNum int) error {
	var comConfig bool
	var comLog bool
	var comDev string
	var comSpeed uint32

	switch comNum {
	case 1:
		comConfig = vm.Config.Com1
		comLog = vm.Config.Com1Log
		comDev = vm.Com1Dev
		comSpeed = vm.Config.Com1Speed
	case 2:
		comConfig = vm.Config.Com2
		comLog = vm.Config.Com2Log
		comDev = vm.Com2Dev
		comSpeed = vm.Config.Com2Speed
	case 3:
		comConfig = vm.Config.Com3
		comLog = vm.Config.Com3Log
		comDev = vm.Com3Dev
		comSpeed = vm.Config.Com3Speed
	case 4:
		comConfig = vm.Config.Com4
		comLog = vm.Config.Com4Log
		comDev = vm.Com4Dev
		comSpeed = vm.Config.Com4Speed
	default:
		slog.Error("invalid com port number", "comNum", comNum)

		return errVMComInvalid
	}
	if !comConfig {
		slog.Debug("vm com not enabled, skipping setup", "comNum", comNum, "comConfig", comConfig)

		return nil
	}
	if comDev == "" {
		slog.Error("com port enabled but com dev not set", "comNum", comNum, "comConfig", comConfig)

		return errVMComDevNotSet
	}
	// if com != nil {
	// 	slog.Error("com port already set, cannot setup com port", "comNum", comNum, "com", com)
	// 	return errors.New("com port already set")
	// }

	// attach serial port object to VM object
	slog.Debug("checking com is readable", "comDev", comDev)
	err := ensureComDevReadable(comDev)
	if err != nil {
		slog.Error("error checking com readable", "comNum", comNum, "err", err)

		return err
	}
	cr, err := startSerialPort(comDev, uint(comSpeed))
	if err != nil {
		slog.Error("error starting com", "comNum", comNum, "err", err)

		return err
	}
	// com = cr

	// actually setup logging if required
	if comLog {
		go comLogger(vm, comNum)
	}

	switch comNum {
	case 1:
		vm.Com1 = cr
	case 2:
		vm.Com2 = cr
	case 3:
		vm.Com3 = cr
	case 4:
		vm.Com4 = cr
	default:
		slog.Error("invalid com port number", "comNum", comNum)

		return errVMComInvalid
	}

	return nil
}

// validateDisks check if disks can be attached to a VM
func validateDisks(diskids []string, vm *VM) error {
	occurred := map[string]bool{}

	for _, aDisk := range diskids {
		slog.Debug("checking disk exists", "disk", aDisk)

		diskUUID, err := uuid.Parse(aDisk)
		if err != nil {
			return errVMDiskInvalid
		}

		thisDisk, err := disk.GetByID(diskUUID.String())
		if err != nil {
			slog.Error("error getting disk", "disk", aDisk, "err", err)

			return fmt.Errorf("error getting disk: %w", err)
		}
		if thisDisk.Name == "" {
			return errVMDiskNotFound
		}

		if !occurred[aDisk] {
			occurred[aDisk] = true
		} else {
			slog.Error("duplicate disk id", "disk", aDisk)

			return errVMDiskDupe
		}

		slog.Debug("checking if disk is attached to another VM", "disk", aDisk)
		diskIsAttached, err := diskAttached(aDisk, vm)
		if err != nil {
			return err
		}
		if diskIsAttached {
			return errVMDiskAttached
		}
	}

	return nil
}

// diskAttached check if disk is attached to another VM besides this one
func diskAttached(aDisk string, vm *VM) (bool, error) {
	allVms := GetAll()
	for _, aVM := range allVms {
		vmDisks, err := aVM.GetDisks()
		if err != nil {
			return true, err
		}
		for _, aVMDisk := range vmDisks {
			if aDisk == aVMDisk.ID && aVM.ID != vm.ID {
				return true, nil
			}
		}
	}

	return false, nil
}

// validateNics check if nics can be attached to a VM
func validateNics(nicIDs []string, vm *VM) error {
	occurred := map[string]bool{}
	for _, aNic := range nicIDs {
		slog.Debug("checking vm nic exists", "vmnic", aNic)

		nicUUID, err := uuid.Parse(aNic)
		if err != nil {
			return errVMNICInvalid
		}

		thisNic, err := vmnic.GetByID(nicUUID.String())
		if err != nil {
			slog.Error("error getting nic", "nic", aNic, "err", err)

			return fmt.Errorf("nic not found: %w", err)
		}
		if thisNic.Name == "" {
			return errVMNICNotFound
		}

		if !occurred[aNic] {
			occurred[aNic] = true
		} else {
			slog.Error("duplicate nic id", "nic", aNic)

			return errVMNicDupe
		}

		slog.Debug("checking if nic is attached to another VM", "nic", aNic)
		err = nicAttached(aNic, vm)
		if err != nil {
			return err
		}
	}

	return nil
}

// nicAttached check if nic is attached to another VM besides this one
func nicAttached(aNic string, vm *VM) error {
	allVms := GetAll()
	for _, aVM := range allVms {
		vmNics, err := aVM.GetNics()
		if err != nil {
			return err
		}
		for _, aVMNic := range vmNics {
			if aNic == aVMNic.ID && aVM.ID != vm.ID {
				slog.Error("nic is already attached to VM", "disk", aNic, "vm", aVM.ID)

				return errVMNicAttached
			}
		}
	}

	return nil
}

// removeAllNicsFromVM does what it says on the tin, mate
func removeAllNicsFromVM(vm *VM) error {
	thisVMNics, err := vm.GetNics()
	if err != nil {
		slog.Error("error looking up vm nics", "err", err)

		return err
	}
	for _, aNic := range thisVMNics {
		aNic.ConfigID = 0
		err := aNic.Save()
		if err != nil {
			slog.Error("error saving NIC", "err", err)

			return fmt.Errorf("error saving NIC: %w", err)
		}
	}

	return nil
}

// cleanup tap/vmnet type nic
func cleanupIfNic(vmNic vmnic.VMNic) error {
	var err error
	if vmNic.NetDev != "" {
		args := []string{"/sbin/ifconfig", vmNic.NetDev, "destroy"}
		cmd := exec.Command(config.Config.Sys.Sudo, args...)
		err = cmd.Run()
		if err != nil {
			slog.Error("failed to destroy network interface", "err", err)
		}
	}
	if vmNic.InstEpair != "" {
		err = epair.DestroyEpair(vmNic.InstEpair)
		if err != nil {
			slog.Error("failed to destroy epair", err)
		}
	}
	if vmNic.InstBridge != "" {
		err = _switch.DestroyIfBridge(vmNic.InstBridge, false)
		if err != nil {
			slog.Error("failed to destroy switch", err)
		}
	}
	// tap/vmnet nics may be connected to an epair which is connected
	// to a netgraph pipe for purposes for rate limiting
	if vmNic.InstEpair != "" {
		err = epair.NgDestroyPipe(vmNic.InstEpair + "a")
		if err != nil {
			slog.Error("failed to ng pipe", err)
		}
		err = epair.NgDestroyPipe(vmNic.InstEpair + "b")
		if err != nil {
			slog.Error("failed to ng pipe", err)
		}
	}

	if err != nil {
		return fmt.Errorf("error cleaning up NIC: %w", err)
	}

	return nil
}
