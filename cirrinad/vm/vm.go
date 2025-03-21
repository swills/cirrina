package vm

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/kontera-technologies/go-supervisor/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cast"
	"github.com/tarm/serial"
	"gorm.io/gorm"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/iso"
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
	CPU              uint32 `gorm:"default:1;check:cpu>=1"` // should be uint16 but changing requires db migration
	Mem              uint32 `gorm:"default:128;check:mem>=128"`
	MaxWait          uint32 `gorm:"default:120;check:max_wait>=0"`
	Restart          bool   `gorm:"default:True;check:restart IN (0,1)"`
	RestartDelay     uint32 `gorm:"default:1;check:restart_delay>=0"`
	Screen           bool   `gorm:"default:True;check:screen IN (0,1)"`
	ScreenWidth      uint32 `gorm:"default:1920;check:screen_width BETWEEN 640 and 3840"`
	ScreenHeight     uint32 `gorm:"default:1080;check:screen_height BETWEEN 480 and 2160"`
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
	VNCPort     int32      // should be uint16 but changing requires db migration
	DebugPort   int32      // should be uint16 but changing requires db migration
	proc        *supervisor.Process
	mu          sync.RWMutex
	log         slog.Logger
	Config      Config
	ISOs        []*iso.ISO   `gorm:"-:all"` // -- ignore this, we're doing it ourselves
	Disks       []*disk.Disk `gorm:"-:all"` // -- ignore this, we're doing it ourselves
	Com1Dev     string       // TODO make a com struct and put these in it?
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

	runningVMsGauge prometheus.Gauge
	totalVMsGauge   prometheus.Gauge
	cpuVMGauge      prometheus.Gauge
	memVMGauge      prometheus.Gauge
)

func (v *VM) monitor(events chan supervisor.Event) {
	for {
		select {
		case msg := <-v.proc.Stdout():
			v.log.Info("output", "stdout", *msg)
		case msg := <-v.proc.Stderr():
			v.log.Info("output", "stderr", *msg)
		case event := <-events:
			switch event.Code {
			case "ProcessStart":
				v.log.Info("event", "code", event.Code, "message", event.Message)

				vmPid := findChildProcName(cast.ToUint32(v.proc.Pid()), "bhyve")
				if vmPid == 0 {
					slog.Error("failed to find VM PID, stopping VM", "event.Message", event.Message)

					err := v.Stop()
					if err != nil {
						slog.Error("failed to stop VM", "err", err)
					}

					v.Done()

					return
				}

				v.SetRunning(vmPid)
				slog.Debug("vmDaemon ProcessStart",
					"bhyvePid", v.BhyvePid,
					"sudoPid", v.proc.Pid(),
				)
				v.setupComLoggers()
				v.applyResourceLimits()
			case "ProcessDone":
				v.log.Info("event", "code", event.Code, "message", event.Message)
			case "ProcessCrashed":
				v.log.Info("exited, destroying") // will be respawned, leave disks/nics alone
				v.killComLoggers()
				v.BhyvectlDestroy()
			default:
				v.log.Info("event", "code", event.Code, "message", event.Message)
			}
		case <-v.proc.DoneNotifier():
			v.Done()

			return
		}
	}
}

func (v *VM) Done() {
	slog.Debug("VM Stop initVM", "vm_name", v.Name)
	v.log.Debug("VM Stop initVM")

	v.killComLoggers()
	v.BhyvectlDestroy()

	err := v.SetStopped()
	if err != nil {
		// log error but continue
		slog.Error("error stopping VM", "err", err)
	}

	v.unlockDisks()
	v.NetStop()

	slog.Debug("VM Stop finalized", "vm_name", v.Name)
	v.log.Debug("VM Stop finalized")

	if config.Config.Metrics.Enabled {
		runningVMsGauge.Dec()
		cpuVMGauge.Sub(float64(v.Config.CPU))
		memVMGauge.Sub(float64(v.Config.Mem))
	}
}

func Exists(vmName string) bool {
	_, err := GetByName(vmName)

	return err == nil
}

func Create(vmInst *VM) error {
	vmAlreadyExists := Exists(vmInst.Name)

	if vmAlreadyExists {
		slog.Error("VM exists", "VM", vmInst.Name)

		return errVMDupe
	}

	err := vmInst.validate()
	if err != nil {
		slog.Error("error validating vm", "VM", vmInst, "err", err)

		return err
	}

	defer List.Mu.Unlock()
	List.Mu.Lock()
	db := GetVMDB()

	slog.Debug("Creating VM", "vm", vmInst.Name)

	res := db.Create(&vmInst)
	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected != 1 {
		return fmt.Errorf("incorrect number of rows affected, err: %w", res.Error)
	}

	vmInst.initVM()

	return nil
}

//nolint:funlen,cyclop
func (v *VM) Save() error {
	vmDB := GetVMDB()

	if v == nil || v.ID == "" || v.Config.ID == 0 {
		return errVMInternalDB
	}

	if slices.Contains(v.ISOs, nil) || slices.Contains(v.Disks, nil) {
		return errVMInternalDB
	}

	res := vmDB.Model(&v.Config).
		Updates(map[string]interface{}{
			"cpu":                &v.Config.CPU,
			"mem":                &v.Config.Mem,
			"max_wait":           &v.Config.MaxWait,
			"restart":            &v.Config.Restart,
			"restart_delay":      &v.Config.RestartDelay,
			"screen":             &v.Config.Screen,
			"screen_width":       &v.Config.ScreenWidth,
			"screen_height":      &v.Config.ScreenHeight,
			"vnc_wait":           &v.Config.VNCWait,
			"vnc_port":           &v.Config.VNCPort,
			"tablet":             &v.Config.Tablet,
			"store_uefi_vars":    &v.Config.StoreUEFIVars,
			"utc_time":           &v.Config.UTCTime,
			"host_bridge":        &v.Config.HostBridge,
			"acpi":               &v.Config.ACPI,
			"use_hlt":            &v.Config.UseHLT,
			"exit_on_pause":      &v.Config.ExitOnPause,
			"wire_guest_mem":     &v.Config.WireGuestMem,
			"destroy_power_off":  &v.Config.DestroyPowerOff,
			"ignore_unknown_msr": &v.Config.IgnoreUnknownMSR,
			"kbd_layout":         &v.Config.KbdLayout,
			"auto_start":         &v.Config.AutoStart,
			"sound":              &v.Config.Sound,
			"sound_in":           &v.Config.SoundIn,
			"sound_out":          &v.Config.SoundOut,
			"Com1":               &v.Config.Com1,
			"com1_dev":           &v.Config.Com1Dev,
			"Com2":               &v.Config.Com2,
			"com2_dev":           &v.Config.Com2Dev,
			"Com3":               &v.Config.Com3,
			"com3_dev":           &v.Config.Com3Dev,
			"com4":               &v.Config.Com4,
			"com4_dev":           &v.Config.Com4Dev,
			"extra_args":         &v.Config.ExtraArgs,
			"com1_log":           &v.Config.Com1Log,
			"com2_log":           &v.Config.Com2Log,
			"com3_log":           &v.Config.Com3Log,
			"com4_log":           &v.Config.Com4Log,
			"com1_speed":         &v.Config.Com1Speed,
			"com2_speed":         &v.Config.Com2Speed,
			"com3_speed":         &v.Config.Com3Speed,
			"com4_speed":         &v.Config.Com4Speed,
			"auto_start_delay":   &v.Config.AutoStartDelay,
			"debug":              &v.Config.Debug,
			"debug_wait":         &v.Config.DebugWait,
			"debug_port":         &v.Config.DebugPort,
			"priority":           &v.Config.Priority,
			"protect":            &v.Config.Protect,
			"pcpu":               &v.Config.Pcpu,
			"rbps":               &v.Config.Rbps,
			"wbps":               &v.Config.Wbps,
			"riops":              &v.Config.Riops,
			"wiops":              &v.Config.Wiops,
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
	}).Model(&v).
		Updates(map[string]interface{}{
			"name":        &v.Name,
			"description": &v.Description,
			"vnc_port":    &v.VNCPort,
			"debug_port":  &v.DebugPort,
			"com1_dev":    &v.Com1Dev,
			"com2_dev":    &v.Com2Dev,
			"com3_dev":    &v.Com3Dev,
			"com4_dev":    &v.Com4Dev,
		})

	if res.Error != nil {
		slog.Error("error updating VM", "res", res)

		return fmt.Errorf("error updating VM: %w", res.Error)
	}

	// delete all isos from VM
	res = vmDB.Exec("DELETE FROM `vm_isos` WHERE `vm_id` = ?", v.ID)
	if res.Error != nil {
		slog.Error("error updating VM", "res.Error", res.Error)

		return fmt.Errorf("error updating VM: %w", res.Error)
	}

	// add all new isos to vm
	err := vmDB.Transaction(func(txDB *gorm.DB) error {
		position := 0

		for _, vmISO := range v.ISOs {
			// this can only happen if another go-routine modified the VM after we checked above
			if vmISO == nil {
				continue
			}
			// N.B.: must use txDB here, not VMDB
			res = txDB.Exec("INSERT INTO `vm_isos` (`vm_id`,`iso_id`, `position`) VALUES (?,?,?)", v.ID, vmISO.ID, position)
			if res.Error != nil || res.RowsAffected != 1 {
				slog.Error("error adding to vm_isos", "res.Error", res.Error)

				return fmt.Errorf("error updating VM: %w", res.Error)
			}

			position++
		}

		return nil
	})
	if err != nil {
		slog.Error("error updating VM", "err", err)

		return fmt.Errorf("error updating VM: %w", err)
	}

	// delete all disks from VM
	res = vmDB.Exec("DELETE FROM `vm_disks` WHERE `vm_id` = ?", v.ID)
	if res.Error != nil {
		slog.Error("error updating VM", "res.Error", res.Error)

		return fmt.Errorf("error updating VM: %w", res.Error)
	}

	// add all new disks to vm
	err = vmDB.Transaction(func(txDB *gorm.DB) error {
		position := 0

		for _, vmDisk := range v.Disks {
			// this can only happen if another go-routine modified the VM after we checked above
			if vmDisk == nil {
				continue
			}
			// N.B.: must use txDB here, not VMDB
			res = txDB.Exec("INSERT INTO `vm_disks` (`vm_id`,`disk_id`, `position`) VALUES (?,?,?)", v.ID, vmDisk.ID, position)
			if res.Error != nil || res.RowsAffected != 1 {
				slog.Error("error adding to vm_disks", "res.Error", res.Error)

				return fmt.Errorf("error updating VM: %w", res.Error)
			}

			position++
		}

		return nil
	})
	if err != nil {
		slog.Error("error updating VM", "err", err)

		return fmt.Errorf("error updating VM: %w", err)
	}

	return nil
}

func (v *VM) Delete() error {
	vmDB := GetVMDB()

	// detach disks
	err := v.AttachDisks([]string{})
	if err != nil {
		slog.Error("failed detaching disks from VM", "err", err)
	}

	// detach isos
	err = v.AttachIsos([]*iso.ISO{})
	if err != nil {
		slog.Error("failed detaching isos from VM", "err", err)
	}

	// detach nics
	err = v.SetNics([]string{})
	if err != nil {
		slog.Error("failed detaching nics from VM", "err", err)
	}

	res := vmDB.Limit(1).Delete(&v.Config)
	if res.RowsAffected != 1 {
		// don't fail deleting the VM, may have a bad or missing config, still want to be able to delete VM
		slog.Error("failed to delete config for VM", "vmid", v.ID)
	}

	res = vmDB.Limit(1).Delete(&v)
	if res.RowsAffected != 1 {
		slog.Error("error deleting VM", "res", res)

		return errVMInternalDB
	}

	if config.Config.Metrics.Enabled {
		totalVMsGauge.Dec()
	}

	return nil
}

func (v *VM) Running() bool {
	if v.Status == RUNNING || v.Status == STOPPING {
		return true
	}

	return false
}

func (v *VM) Start() error {
	var err error
	defer vmStartLock.Unlock()
	vmStartLock.Lock()

	if v.Status != STOPPED {
		return errVMNotStopped
	}

	v.SetStarting()

	events := make(chan supervisor.Event)

	v.lockDisks()

	cmdName, cmdArgs := v.generateCommandLine()
	v.log.Info("start", "cmd", cmdName, "args", cmdArgs)
	v.createUefiVarsFile()

	err = v.netStart()
	if err != nil {
		slog.Error("Failed VM net startup, cleaning up", "err", err)
		v.NetStop()

		return err
	}

	err = v.Save()
	if err != nil {
		slog.Error("Failed saving VM", "err", err)

		return err
	}

	respawnWait := time.Duration(v.Config.RestartDelay) * time.Second
	// avoid go-supervisor setting this to default (2m) -- 1ns is hard to differentiate from 0ns and I prefer not to
	// change go-supervisor unless I have to
	if respawnWait == 0 {
		respawnWait = 1
	}

	var processDebug bool

	if config.Config.Log.Level == "debug" {
		slog.Debug("vm.Start enabling process debugging", "vm", v.Name)

		processDebug = true
	}

	vmProc := supervisor.NewProcess(supervisor.ProcessOptions{
		Name:                    cmdName,
		Args:                    cmdArgs,
		Dir:                     "/",
		Id:                      v.Name,
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
		TerminationGraceTimeout: time.Duration(v.Config.MaxWait) * time.Second,
		Debug:                   processDebug,
	})

	v.proc = vmProc
	go v.monitor(events)

	err = vmProc.Start()
	if err != nil {
		slog.Error("failed to start process", "err", err)
		panic(fmt.Sprintf("failed to start process: %s", err))
	}

	if config.Config.Metrics.Enabled {
		runningVMsGauge.Inc()
		cpuVMGauge.Add(float64(v.Config.CPU))
		memVMGauge.Add(float64(v.Config.Mem))
	}

	return nil
}

func (v *VM) Stop() error {
	var err error

	if v.Status == STOPPED {
		return nil
	}

	v.SetStopping()

	if v.proc == nil {
		err = v.SetStopped()
		if err != nil {
			slog.Error("error stopping VM", "err", err)

			return fmt.Errorf("error stopping VM: %w", err)
		}

		return nil
	}

	err = v.proc.Stop()
	if err != nil {
		slog.Error("Failed to stop VM", "vm", v.Name, "pid", v.proc.Pid(), "err", err)

		return errVMStopFail
	}

	return nil
}

func (v *VM) BhyvectlDestroy() {
	ex, err := PathExistsFunc("/dev/vmm/" + v.Name)
	if err != nil {
		return
	}

	if !ex {
		return
	}

	args := []string{"/usr/sbin/bhyvectl", "--destroy"}
	args = append(args, "--vm="+v.Name)

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

func (v *VM) validate() error {
	if !util.ValidVMName(v.Name) {
		return errVMInvalidName
	}

	return nil
}

// initVM initializes and adds a VM to the in memory cache of VMs
// note, callers must lock the in memory cache via List.Mu.Lock()
func (v *VM) initVM() {
	vmLogPath := config.Config.Disk.VM.Path.State + "/" + v.Name

	err := GetVMLogPath(vmLogPath)
	if err != nil {
		slog.Error("failed to init vm", "err", err)
		panic(err)
	}

	vmLogFilePath := vmLogPath + "/log"

	vmLogFile, err := OsOpenFileFunc(vmLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Error("failed to open VM log file", "err", err)
	}

	programLevel := new(slog.LevelVar) // Info by default
	vmLogger := slog.New(slog.NewTextHandler(vmLogFile, &slog.HandlerOptions{Level: programLevel}))

	v.log = *vmLogger

	switch strings.ToLower(config.Config.Log.Level) {
	case "debug":
		programLevel.Set(slog.LevelDebug)
	case "info":
		programLevel.Set(slog.LevelInfo)
	case "warn":
		programLevel.Set(slog.LevelWarn)
	case "error":
		programLevel.Set(slog.LevelError)
	default:
		programLevel.Set(slog.LevelInfo)
	}

	List.VMList[v.ID] = v

	if config.Config.Metrics.Enabled {
		totalVMsGauge.Inc()
	}
}

func (v *VM) doAutostart() {
	slog.Debug(
		"AutoStartVMs sleeping for auto start delay",
		"vm", v.Name,
		"auto_start_delay", v.Config.AutoStartDelay,
	)
	time.Sleep(time.Duration(v.Config.AutoStartDelay) * time.Second)

	err := v.Start()
	if err != nil {
		slog.Error("auto start failed", "vm", v.ID, "name", v.Name, "err", err)
	}
}

// CheckAll gets all disks/nics/isos and ensure the VM they are attached in the join table exists
func CheckAll() {
	checkDiskAttachments()
	checkIsoAttachments()
	checkNicAttachments()
}

func checkDiskAttachments() {
	vmDB := GetVMDB()

	allDisks := disk.GetAllDB()
	for _, aDisk := range allDisks {
		vmIDs := aDisk.GetVMIDs()
		for _, vmID := range vmIDs {
			// check the VM exists
			_, err := GetByID(vmID)
			if err != nil {
				slog.Error("disk attached to non-existent VM, removing", "disk.ID", aDisk.ID, "vm.ID", vmID)

				res := vmDB.Exec("DELETE FROM `vm_disks` WHERE `vm_id` = ?", vmID)
				if res.Error != nil {
					slog.Error("error removing bad attachment", "res.Error", res.Error)
				}
			}
		}
	}
}

func checkNicAttachments() {
	allNics := vmnic.GetAll()
	for _, aNic := range allNics {
		vmIDs := aNic.GetVMIDs()
		for _, vmID := range vmIDs {
			// check the VM exists
			_, err := GetByID(vmID)
			if err != nil {
				slog.Error("nic attached to non-existent VM, removing", "nic.ID", aNic.ID, "vm.ID", vmID)

				aNic.ConfigID = 0

				err = aNic.Save()
				if err != nil {
					slog.Error("error saving NIC", "err", err)
				}
			}
		}
	}
}

func checkIsoAttachments() {
	vmDB := GetVMDB()

	allISOs := iso.GetAll()
	for _, aISO := range allISOs {
		vmIDs := aISO.GetVMIDs()
		for _, vmID := range vmIDs {
			// check the VM exists
			_, err := GetByID(vmID)
			if err != nil {
				slog.Error("iso attached to non-existent VM, removing", "iso.ID", aISO.ID, "vm.ID", vmID)

				res := vmDB.Exec("DELETE FROM `vm_isos` WHERE `vm_id` = ?", vmID)
				if res.Error != nil {
					slog.Error("error removing bad attachment", "res.Error", res.Error)
				}
			}
		}
	}
}
