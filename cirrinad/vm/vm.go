package vm

import (
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kontera-technologies/go-supervisor/v2"
	"github.com/tarm/serial"
	"gorm.io/gorm"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/iso"
	"cirrina/cirrinad/util"
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
	ID          string `gorm:"uniqueIndex;not null;default:null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Name        string         `gorm:"not null"`
	Description string
	Status      StatusType `gorm:"type:status_type"`
	BhyvePid    uint32     `gorm:"check:bhyve_pid>=0"`
	VNCPort     int32
	DebugPort   int32
	proc        *supervisor.Process
	mu          sync.RWMutex
	log         slog.Logger
	Config      Config
	ISOs        []iso.ISO `gorm:"-:all"` // -- ignore this, we're doing it ourselves now
	Com1Dev     string    // TODO make a com struct and put these in it?
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

var (
	vmStartLock sync.Mutex
	List        = &ListType{
		VMList: make(map[string]*VM),
	}
)

func Create(vmInst *VM) error {
	vmAlreadyExists, err := vmExists(vmInst.Name)
	if err != nil {
		slog.Error("error checking db for vm", "name", vmInst.Name, "err", err)

		return err
	}

	if vmAlreadyExists {
		slog.Error("VM exists", "VM", vmInst.Name)

		return errVMDupe
	}

	err = validateVM(vmInst)
	if err != nil {
		slog.Error("error validating vm", "VM", vmInst, "err", err)

		return err
	}

	defer List.Mu.Unlock()
	List.Mu.Lock()
	db := GetVMDB()

	slog.Debug("Creating VM", "vm", vmInst.Name)

	res := db.Create(&vmInst)
	if res.RowsAffected != 1 {
		return fmt.Errorf("incorrect number of rows affected, err: %w", res.Error)
	}

	if res.Error != nil {
		return res.Error
	}

	initOneVM(vmInst)

	return nil
}

//nolint:funlen
func (vm *VM) Save() error {
	vmDB := GetVMDB()

	res := vmDB.Model(&vm.Config).
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

	res = vmDB.Select([]string{
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

	// delete all isos from VM
	res = vmDB.Exec("DELETE FROM `vm_isos` WHERE `vm_id` = ?", vm.ID)
	if res.Error != nil {
		slog.Error("error updating VM", "res.Error", res.Error)

		return fmt.Errorf("error updating VM: %w", res.Error)
	}

	// add all new isos to vm
	err := vmDB.Transaction(func(tx *gorm.DB) error {
		for i, vmISO := range vm.ISOs {
			// N.B.: must use tx here, not vmDB
			res = tx.Exec("INSERT INTO `vm_isos` (`vm_id`,`iso_id`, `position`) VALUES (?,?,?)", vm.ID, vmISO.ID, i)
			if res.Error != nil || res.RowsAffected != 1 {
				slog.Error("error adding to vm_isos", "res.Error", res.Error)

				return fmt.Errorf("error updating VM: %w", res.Error)
			}
		}

		return nil
	})
	if err != nil {
		slog.Error("error updating VM", "err", err)

		return fmt.Errorf("error updating VM: %w", err)
	}

	return nil
}

func (vm *VM) Delete() error {
	vmDB := GetVMDB()
	vmDB.Model(&VM{}).Preload("Config").Limit(1).Find(&vm, &VM{ID: vm.ID})

	if vm.ID == "" {
		return errVMNotFound
	}

	res := vmDB.Limit(1).Delete(&vm.Config)
	if res.RowsAffected != 1 {
		// don't fail deleting the VM, may have a bad or missing config, still want to be able to delete VM
		slog.Error("failed to delete config for VM", "vmid", vm.ID)
	}

	res = vmDB.Limit(1).Delete(&vm)
	if res.RowsAffected != 1 {
		slog.Error("error deleting VM", "res", res)

		return errVMInternalDB
	}

	return nil
}

func (vm *VM) Running() bool {
	if vm.Status == RUNNING || vm.Status == STOPPING {
		return true
	}

	return false
}

func (vm *VM) Start() error {
	var err error
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

	vmProc := supervisor.NewProcess(supervisor.ProcessOptions{
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

	vm.proc = vmProc
	go vmDaemon(events, vm)

	err = vmProc.Start()
	if err != nil {
		panic(fmt.Sprintf("failed to start process: %s", err))
	}

	return nil
}

func (vm *VM) Stop() error {
	var err error

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

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		args,
	)
	if string(stdErrBytes) != "" || returnCode != 0 || err != nil {
		slog.Error("error running command",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}
}

func vmDaemon(events chan supervisor.Event, thisVM *VM) {
	for {
		select {
		case msg := <-thisVM.proc.Stdout():
			thisVM.log.Info("output", "stdout", *msg)
		case msg := <-thisVM.proc.Stderr():
			thisVM.log.Info("output", "stderr", *msg)
		case event := <-events:
			switch event.Code {
			case "ProcessStart":
				thisVM.log.Info("event", "code", event.Code, "message", event.Message)
				vmPid := findChildProcName(uint32(thisVM.proc.Pid()), "bhyve")

				if vmPid == 0 {
					slog.Error("failed to find vm PID, continuing anyway")
				}

				thisVM.SetRunning(int(vmPid))
				slog.Debug("vmDaemon ProcessStart", "bhyvePid", thisVM.BhyvePid, "sudoPid", thisVM.proc.Pid(), "realPid", vmPid)
				thisVM.setupComLoggers()
				thisVM.applyResourceLimits(vmPid)
			case "ProcessDone":
				thisVM.log.Info("event", "code", event.Code, "message", event.Message)
			case "ProcessCrashed":
				thisVM.log.Info("exited, destroying")
				thisVM.MaybeForceKillVM()
			default:
				thisVM.log.Info("event", "code", event.Code, "message", event.Message)
			}
		case <-thisVM.proc.DoneNotifier():
			slog.Debug("vm stopped",
				"vm_name", thisVM.Name,
			)
			thisVM.log.Info("stopped")
			thisVM.NetCleanup()
			thisVM.killComLoggers()
			thisVM.SetStopped()

			err := thisVM.unlockDisks()
			if err != nil {
				slog.Debug("failed unlock disks", "err", err)

				return
			}

			thisVM.MaybeForceKillVM()
			thisVM.log.Info("closing loop we are done")

			return
		}
	}
}

func validateVM(vmInst *VM) error {
	if !util.ValidVMName(vmInst.Name) {
		return errVMInvalidName
	}

	return nil
}

func vmExists(vmName string) (bool, error) {
	_, err := GetByName(vmName)
	if err == nil {
		return true, errVMDupe
	}

	return false, nil
}
