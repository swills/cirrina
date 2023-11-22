package vm

import (
	"bufio"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/epair"
	"cirrina/cirrinad/iso"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm_nics"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/kontera-technologies/go-supervisor/v2"
	"github.com/tarm/serial"
	exec "golang.org/x/sys/execabs"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func Create(name string, description string, cpu uint32, mem uint32) (vm *VM, err error) {
	var vmInst *VM
	if !util.ValidVmName(name) {
		return vmInst, errors.New("invalid name")
	}
	if _, err := GetByName(name); err == nil {
		return vmInst, errors.New(fmt.Sprintf("%v already exists", name))
	}
	vmInst = &VM{
		Name:        name,
		Status:      STOPPED,
		Description: description,
		Config: Config{
			Cpu: cpu,
			Mem: mem,
		},
	}
	db := getVmDb()
	slog.Debug("Creating VM", "vm", name)
	res := db.Create(&vmInst)
	return vmInst, res.Error
}

func (vm *VM) Delete() (err error) {
	db := getVmDb()
	db.Model(&VM{}).Preload("Config").Limit(1).Find(&vm, &VM{ID: vm.ID})
	if vm.ID == "" {
		return errors.New("not found")
	}
	res := db.Limit(1).Delete(&vm.Config)
	if res.RowsAffected != 1 {
		// don't fail deleting the VM, may have a bad or missing config, still want to be able to delete VM
		slog.Error("failed to delete config for VM", "vmid", vm.ID)
	}
	res = db.Limit(1).Delete(&vm)
	if res.RowsAffected != 1 {
		return errors.New("failed to delete VM")
	}
	return nil
}

var vmStartLock sync.Mutex

func (vm *VM) Start() (err error) {
	defer vmStartLock.Unlock()
	vmStartLock.Lock()
	if vm.Status != STOPPED {
		return errors.New("must be stopped first")
	}
	vm.SetStarting()
	events := make(chan supervisor.Event)

	cmdName, cmdArgs, err := vm.generateCommandLine()
	vm.log.Info("start", "cmd", cmdName, "args", cmdArgs)
	vm.createUefiVarsFile()
	vm.netStartup()
	err = vm.Save()
	if err != nil {
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
		return errors.New("VM already stopped")
	}
	vm.SetStopping()
	if vm.proc == nil {
		vm.SetStopped()
		return nil
	}
	err = vm.proc.Stop()
	if err != nil {
		slog.Error("Failed to stop VM", "vm", vm.Name, "pid", vm.proc.Pid())
		return errors.New("stop failed")
	}
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
			"nics":               &vm.Config.Nics,
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
		return errors.New("error updating VM")
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
		slog.Error("db update error", "err", res.Error)
		return errors.New("error updating VM")
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
	uefiVarsFilePath := baseVMStatePath + "/" + vm.Name
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
		_, err = util.CopyFile(uefiVarFileTemplate, uefiVarsFile)
		if err != nil {
			slog.Error("failed to copy uefiVars template", "err", err)
		}
	}
}

func (vm *VM) netStartup() {
	vmNicsList, err := vm.GetNics()

	if err != nil {
		slog.Error("netStartup failed to get nics", "err", err)
		return
	}

	for _, vmNic := range vmNicsList {
		if vmNic.NetDevType == "TAP" || vmNic.NetDevType == "VMNET" {
			// Create interface
			args := []string{"/sbin/ifconfig", vmNic.NetDev, "create", "group", "cirrinad"}
			cmd := exec.Command(config.Config.Sys.Sudo, args...)
			err := cmd.Run()
			if err != nil {
				slog.Error("failed to create tap", "err", err)
			}
			// Add interface to bridge
			if vmNic.SwitchId != "" {
				thisSwitch, err := _switch.GetById(vmNic.SwitchId)
				if err != nil {
					slog.Error("bad switch id",
						"nicname", vmNic.Name, "nicid", vmNic.ID, "switchid", vmNic.SwitchId)
				}
				if thisSwitch.Type == "IF" {
					if vmNic.RateLimit {
						thisEpair := epair.GetDummyEpairName()
						slog.Debug("netStartup rate limiting", "thisEpair", thisEpair)
						err = epair.CreateEpair(thisEpair)
						if err != nil {
							slog.Error("error creating epair", err)
						}
						vmNic.InstEpair = thisEpair
						err = vmNic.Save()
						if err != nil {
							slog.Error("failed to save net dev", "nic", vmNic.ID, "netdev", vmNic.NetDev)
						}
						err = epair.SetRateLimit(thisEpair, vmNic.RateIn, vmNic.RateOut)
						if err != nil {
							slog.Error("failed to set epair rate limit", "epair", thisEpair)
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
						}
						vmNic.InstBridge = thisInstSwitch
						err = vmNic.Save()
						if err != nil {
							slog.Error("failed to save net dev", "nic", vmNic.ID, "netdev", vmNic.NetDev)
						}
						err := _switch.BridgeIfAddMember(thisSwitch.Name, thisEpair+"b", true, "")
						if err != nil {
							slog.Error("failed to add nic to switch",
								"nicname", vmNic.Name,
								"nicid", vmNic.ID,
								"switchid", vmNic.SwitchId,
								"netdev", vmNic.NetDev,
								"err", err,
							)
						}

					} else {
						mac := GetMac(&vmNic, vm)
						err := _switch.BridgeIfAddMember(thisSwitch.Name, vmNic.NetDev, false, mac)
						if err != nil {
							slog.Error("failed to add nic to switch",
								"nicname", vmNic.Name,
								"nicid", vmNic.ID,
								"switchid", vmNic.SwitchId,
								"netdev", vmNic.NetDev,
								"err", err,
							)
						}
					}
				} else {
					slog.Error("bridge/interface type mismatch",
						"nicname", vmNic.Name,
						"nicid", vmNic.ID,
						"switchid", vmNic.SwitchId,
					)
				}
			}
		} else if vmNic.NetDevType == "NETGRAPH" {
			thisSwitch, err := _switch.GetById(vmNic.SwitchId)
			if err != nil {
				slog.Error("bad switch id",
					"nicname", vmNic.Name, "nicid", vmNic.ID, "switchid", vmNic.SwitchId)
			}
			if thisSwitch.Type != "NG" {
				slog.Error("bridge/interface type mismatch",
					"nicname", vmNic.Name,
					"nicid", vmNic.ID,
					"switchid", vmNic.SwitchId,
				)
			}
		} else {
			slog.Debug("unknown net type, can't set up")
			return
		}
	}
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

func (vm *VM) NetCleanup() {
	vmNicsList, err := vm.GetNics()
	if err != nil {
		slog.Error("netStartup failed to get nics", "err", err)
		return
	}
	for _, vmNic := range vmNicsList {
		if vmNic.NetDevType == "TAP" || vmNic.NetDevType == "VMNET" {
			if vmNic.NetDev != "" {
				args := []string{"/sbin/ifconfig", vmNic.NetDev, "destroy"}
				cmd := exec.Command(config.Config.Sys.Sudo, args...)
				err := cmd.Run()
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
		} else if vmNic.NetDevType == "NETGRAPH" {
			// nothing to do for netgraph
		} else {
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

func findChildPid(findPid uint32) (childPid uint32) {
	slog.Debug("FindChildPid finding child proc")
	pidString := strconv.FormatUint(uint64(findPid), 10)
	args := []string{"/bin/pgrep", "-P", pidString}
	cmd := exec.Command(config.Config.Sys.Sudo, args...)
	defer func(cmd *exec.Cmd) {
		err := cmd.Wait()
		if err != nil {
			slog.Error("FindChildPid error", "err", err)
		}
	}(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("FindChildPid error", "err", err)
		return 0
	}
	if err := cmd.Start(); err != nil {
		slog.Error("FindChildPid error", "err", err)
		return 0
	}
	found := false
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		text := scanner.Text()
		textFields := strings.Fields(text)
		fl := len(textFields)
		if fl != 1 {
			slog.Debug("FindChildPid pgrep extra fields", "text", text)
		}
		tempPid1 := uint64(0)
		if !found {
			found = true
			tempPid1, err = strconv.ParseUint(textFields[0], 10, 32)
			if err != nil {
				slog.Error("FindChildPid error", "err", err)
				return 0
			}
			tempPid2 := uint32(tempPid1)
			childPid = tempPid2
		} else {
			slog.Debug("FindChildPid found too many child procs")
		}
	}
	if err := scanner.Err(); err != nil {
		slog.Error("FindChildPid error", "err", err)
	}
	slog.Debug("FindChildPid returning childPid", "childPid", childPid)
	return childPid
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
			vm.MaybeForceKillVM()
			vm.log.Info("closing loop we are done")
			return
		}
	}
}

func (vm *VM) GetISOs() ([]iso.ISO, error) {
	defer vm.mu.RUnlock()
	vm.mu.RLock()
	var isos []iso.ISO
	// TODO remove all these de-normalizations in favor of gorm native "Has Many" relationships
	for _, cv := range strings.Split(vm.Config.ISOs, ",") {
		if cv == "" {
			continue
		}
		aISO, err := iso.GetById(cv)
		if err == nil {
			isos = append(isos, *aISO)
		} else {
			slog.Error("bad iso", "iso", cv, "vm", vm.ID)
		}
	}
	return isos, nil
}

func (vm *VM) GetNics() ([]vm_nics.VmNic, error) {
	defer vm.mu.RUnlock()
	vm.mu.RLock()
	var nics []vm_nics.VmNic
	// TODO remove all these de-normalizations in favor of gorm native "Has Many" relationships
	for _, cv := range strings.Split(vm.Config.Nics, ",") {
		if cv == "" {
			continue
		}
		aNic, err := vm_nics.GetById(cv)
		if err == nil {
			nics = append(nics, *aNic)
		} else {
			slog.Error("bad nic", "nic", cv, "vm", vm.ID)
		}
	}
	return nics, nil
}

func (vm *VM) GetDisks() ([]disk.Disk, error) {
	defer vm.mu.RUnlock()
	vm.mu.RLock()
	var disks []disk.Disk
	// TODO remove all these de-normalizations in favor of gorm native "Has Many" relationships
	for _, cv := range strings.Split(vm.Config.Disks, ",") {
		if cv == "" {
			continue
		}
		aDisk, err := disk.GetById(cv)
		if err == nil {
			disks = append(disks, *aDisk)
		} else {
			slog.Error("bad disk", "disk", cv, "vm", vm.ID)
		}
	}
	return disks, nil
}

func (vm *VM) DeleteUEFIState() error {
	uefiVarsFilePath := baseVMStatePath + "/" + vm.Name
	uefiVarsFile := uefiVarsFilePath + "/BHYVE_UEFI_VARS.fd"
	uvFileExists, err := util.PathExists(uefiVarsFile)
	if err != nil {
		return nil
	}
	if uvFileExists {
		if err := os.Remove(uefiVarsFile); err != nil {
			return err
		}
	}
	return nil
}

func (vm *VM) AttachIsos(isoIds []string) error {
	defer vm.mu.Unlock()
	vm.mu.Lock()

	if vm.Status != STOPPED {
		return errors.New("VM must be stopped before adding isos(s)")
	}

	for _, aIso := range isoIds {
		slog.Debug("checking iso exists", "iso", aIso)

		isoUuid, err := uuid.Parse(aIso)
		if err != nil {
			return errors.New("iso id not specified or invalid")
		}

		thisIso, err := iso.GetById(isoUuid.String())
		if err != nil {
			slog.Error("error getting disk", "disk", aIso, "err", err)
			return errors.New("iso not found")
		}
		if thisIso.Name == "" {
			return errors.New("iso not found")
		}
	}

	var isoConfigVal string
	count := 0

	for _, isoId := range isoIds {
		if count > 0 {
			isoConfigVal += ","
		}
		isoConfigVal += isoId
		count += 1
	}
	vm.Config.ISOs = isoConfigVal
	err := vm.Save()
	if err != nil {
		slog.Error("error saving VM", "err", err)
		return err
	}
	return nil
}

func (vm *VM) AttachNics(nicIds []string) error {
	defer vm.mu.Unlock()
	vm.mu.Lock()
	if vm.Status != STOPPED {
		return errors.New("VM must be stopped before adding disk(s)")
	}
	occurred := map[string]bool{}
	var result []string

	for _, aNic := range nicIds {
		slog.Debug("checking vm nic exists", "vmnic", aNic)

		nicUuid, err := uuid.Parse(aNic)
		if err != nil {
			return errors.New("nic id not specified or invalid")
		}

		thisNic, err := vm_nics.GetById(nicUuid.String())
		if err != nil {
			slog.Error("error getting nic", "nic", aNic, "err", err)
			return errors.New("nic not found")
		}
		if thisNic.Name == "" {
			return errors.New("nic not found")
		}

		if occurred[aNic] != true {
			occurred[aNic] = true
			result = append(result, aNic)
		} else {
			slog.Error("duplicate nic id", "nic", aNic)
			return errors.New("nic may only be added once")
		}

		slog.Debug("checking if nic is attached to another VM", "nic", aNic)
		allVms := GetAll()
		for _, aVm := range allVms {
			vmNics, err := aVm.GetNics()
			if err != nil {
				return err
			}
			for _, aVmNic := range vmNics {
				if aNic == aVmNic.ID && aVm.ID != vm.ID {
					slog.Error("nic is already attached to VM", "disk", aNic, "vm", aVm.ID)
					return errors.New("nic already attached")
				}
			}
		}
	}

	var nicsConfigVal string
	count := 0
	for _, nicId := range nicIds {
		if count > 0 {
			nicsConfigVal += ","
		}
		nicsConfigVal += nicId
		count += 1
	}
	vm.Config.Nics = nicsConfigVal
	err := vm.Save()
	if err != nil {
		slog.Error("error saving VM", "err", err)
		return err
	}
	return nil
}

func (vm *VM) AttachDisks(diskids []string) error {
	defer vm.mu.Unlock()
	vm.mu.Lock()
	if vm.Status != STOPPED {
		return errors.New("VM must be stopped before adding disk(s)")
	}

	occurred := map[string]bool{}
	var result []string

	for _, aDisk := range diskids {
		slog.Debug("checking disk exists", "disk", aDisk)

		diskUuid, err := uuid.Parse(aDisk)
		if err != nil {
			return errors.New("disk id not specified or invalid")
		}

		thisDisk, err := disk.GetById(diskUuid.String())
		if err != nil {
			return err
		}
		if err != nil {
			slog.Error("error getting disk", "disk", aDisk, "err", err)
			return errors.New("disk not found")
		}
		if thisDisk.Name == "" {
			return errors.New("disk not found")
		}

		if occurred[aDisk] != true {
			occurred[aDisk] = true
			result = append(result, aDisk)
		} else {
			slog.Error("duplicate disk id", "disk", aDisk)
			return errors.New("disk may only be added once")
		}

		slog.Debug("checking if disk is attached to another VM", "disk", aDisk)
		allVms := GetAll()
		for _, aVm := range allVms {
			vmDisks, err := aVm.GetDisks()
			if err != nil {
				return err
			}
			for _, aVmDisk := range vmDisks {
				if aDisk == aVmDisk.ID && aVm.ID != vm.ID {
					slog.Error("disk is already attached to VM", "disk", aDisk, "vm", aVm.ID)
					return errors.New("disk already attached")
				}
			}
		}
	}

	var disksConfigVal string
	count := 0
	for _, diskId := range diskids {
		if count > 0 {
			disksConfigVal += ","
		}
		disksConfigVal += diskId
		count += 1
	}
	vm.Config.Disks = disksConfigVal
	err := vm.Save()
	if err != nil {
		slog.Error("error saving VM", "err", err)
		return err
	}
	return nil
}

func (vm *VM) killComLoggers() {
	if vm.Com1 != nil {
		_ = vm.Com1.Close()
		vm.Com1 = nil
	}
	if vm.Com1rchan != nil {
		vm.Com1rchan = nil
	}
	if vm.Com2 != nil {
		_ = vm.Com2.Close()
		vm.Com2 = nil
	}
	if vm.Com2rchan != nil {
		vm.Com2rchan = nil
	}
	if vm.Com3 != nil {
		_ = vm.Com3.Close()
		vm.Com3 = nil
	}
	if vm.Com3rchan != nil {
		vm.Com3rchan = nil
	}
	if vm.Com4 != nil {
		_ = vm.Com4.Close()
		vm.Com4 = nil
	}
	if vm.Com4rchan != nil {
		vm.Com4rchan = nil
	}
}

func (vm *VM) setupComLoggers() {
	if vm.Com1Dev != "" && vm.Com1 == nil {
		cr, err := startSerialPort(vm.Com1Dev, uint(vm.Config.Com1Speed))
		if err != nil {
			slog.Error("setupComLoggers", "err", err)
			return
		}
		vm.Com1 = cr

		if vm.Config.Com1Log {
			go comLogger(vm, 1)
		}
	}
	if vm.Com2Dev != "" && vm.Com2 == nil {
		cr, err := startSerialPort(vm.Com2Dev, uint(vm.Config.Com2Speed))
		if err != nil {
			slog.Error("setupComLoggers", "err", err)
			return
		}
		vm.Com2 = cr

		if vm.Config.Com2Log {
			go comLogger(vm, 2)
		}
	}
	if vm.Com3Dev != "" && vm.Com3 == nil {
		cr, err := startSerialPort(vm.Com3Dev, uint(vm.Config.Com3Speed))
		if err != nil {
			slog.Error("setupComLoggers", "err", err)
			return
		}
		vm.Com3 = cr

		if vm.Config.Com3Log {
			go comLogger(vm, 3)
		}
	}
	if vm.Com4Dev != "" && vm.Com4 == nil {
		cr, err := startSerialPort(vm.Com4Dev, uint(vm.Config.Com4Speed))
		if err != nil {
			slog.Error("setupComLoggers", "err", err)
			return
		}
		vm.Com4 = cr

		if vm.Config.Com4Log {
			go comLogger(vm, 4)
		}
	}

	return
}

func comLogger(vm *VM, comNum int) {
	var thisCom *serial.Port
	var thisRChan chan byte
	var thisComChanWriteFlag bool
	slog.Debug("comLogger starting", "comNum", comNum)

	switch comNum {
	case 1:
		thisCom = vm.Com1
		if vm.Config.Com1Log {
			com1Chan := make(chan byte)
			vm.Com1rchan = com1Chan
			thisRChan = vm.Com1rchan
		}
	case 2:
		thisCom = vm.Com2
		if vm.Config.Com2Log {
			com2Chan := make(chan byte)
			vm.Com2rchan = com2Chan
			thisRChan = vm.Com2rchan
		}
	case 3:
		thisCom = vm.Com3
		if vm.Config.Com3Log {
			com3Chan := make(chan byte)
			vm.Com3rchan = com3Chan
			thisRChan = vm.Com3rchan
		}
	case 4:
		thisCom = vm.Com4
		if vm.Config.Com4Log {
			com4Chan := make(chan byte)
			vm.Com4rchan = com4Chan
			thisRChan = vm.Com4rchan
		}
	default:
		slog.Error("comLogger invalid com", "comNum", comNum)
		return
	}

	comLogPath := config.Config.Disk.VM.Path.State + "/" + vm.Name + "/"
	comLogFile := comLogPath + "com" + strconv.Itoa(comNum) + "_out.log"
	err := GetVmLogPath(comLogPath)
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

	n := 0
	for {
		if vm.Status != RUNNING {
			slog.Debug("comLogger", "msg", "vm not running, exiting2")
			return
		}
		if thisCom == nil {
			slog.Error("comLogger", "msg", "unable to read nil port")
			return
		}

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

		b := make([]byte, 1)
		b2 := make([]byte, 1)
		nb, err := thisCom.Read(b)
		if nb > 1 {
			slog.Error("comLogger read more than 1 byte", "nb", nb)
		}
		if err == io.EOF && vm.Status != RUNNING {
			slog.Debug("comLogger vm not running, exiting", "vm_id", vm.ID, "comNum", comNum)
			return
		}
		if err != nil && err != io.EOF {
			slog.Error("comLogger", "error reading", err)
			return
		}
		if nb != 0 {
			nb2 := copy(b2, b)
			if nb != nb2 {
				slog.Error("comLogger", "msg", "some bytes lost")
			}

			// write to log file
			_, err = vl.Write(b)

			// write to channel used by remote users, if someone is reading from it
			if thisRChan != nil && thisComChanWriteFlag {
				thisRChan <- b2[0]
			}

			n = n + nb
			if err != nil {
				slog.Error("comLogger", "error writing", err)
				return
			}
		}
	}
}

func startSerialPort(comDev string, comSpeed uint) (*serial.Port, error) {
	if strings.HasSuffix(comDev, "A") {
		comBaseDev := comDev[:len(comDev)-1]
		comReadDev := comBaseDev + "B"
		slog.Debug("startSerialPort starting serial port on com",
			"comReadDev", comReadDev,
			"comSpeed", comSpeed,
		)
		c := &serial.Config{
			Name:        comReadDev,
			Baud:        int(comSpeed),
			ReadTimeout: 500 * time.Millisecond,
		}
		comReader, err := serial.OpenPort(c)
		if err != nil {
			slog.Error("startSerialPort error opening comReadDev", "error", err)
		}
		slog.Debug("startSerialLogger", "opened", comReadDev)
		return comReader, nil
	}
	return nil, errors.New("invalid com dev")
}
