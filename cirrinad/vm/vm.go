package vm

import (
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/iso"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm_nics"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/kontera-technologies/go-supervisor/v2"
	"golang.org/x/exp/slog"
	"gorm.io/gorm"
	"os"
	"os/exec"
	"strings"
	"time"
)

func (vm *VM) BeforeCreate(_ *gorm.DB) (err error) {
	vm.ID = uuid.NewString()
	return nil
}

func Create(name string, description string, cpu uint32, mem uint32) (vm *VM, err error) {
	var vmInst *VM
	if strings.Contains(name, "/") {
		return vmInst, errors.New("illegal character in vm name")
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
	defer func() {
		vm.mu.Unlock()
	}()
	vm.mu.Lock()
	vm.setStarting()
	List.VmList[vm.ID].Status = STARTING
	events := make(chan supervisor.Event)

	cmdName, cmdArgs, err := vm.generateCommandLine()
	vm.log.Info("start", "cmd", cmdName, "args", cmdArgs)
	vm.createUefiVarsFile()
	vm.netStartup()
	err = vm.Save()
	if err != nil {
		return err
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
		RespawnWait:             time.Duration(vm.Config.RestartDelay) * time.Second,
		SpawnWait:               time.Duration(vm.Config.RestartDelay) * time.Second,
		MaxInterruptAttempts:    1,
		MaxTerminateAttempts:    1,
		IdleTimeout:             -1,
		TerminationGraceTimeout: time.Duration(vm.Config.MaxWait) * time.Second,
	})
	List.VmList[vm.ID].proc = p
	go vmDaemon(events, vm)

	if err := p.Start(); err != nil {
		panic(fmt.Sprintf("failed to start process: %s", err))
	}
	return nil
}

func (vm *VM) Stop() (err error) {
	if vm.Status != RUNNING {
		slog.Error("tried to stop VM that is not running", "vm", vm.Name)
		return errors.New("must be running first")
	}
	defer vm.mu.Unlock()
	vm.mu.Lock()
	setStopping(vm.ID)
	if vm.proc == nil {
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
			"kbd_layout":         &vm.Config.KbdLayout,
			"auto_start":         &vm.Config.AutoStart,
			"sound":              &vm.Config.Sound,
			"sound_in":           &vm.Config.SoundIn,
			"sound_out":          &vm.Config.SoundOut,
			"com1":               &vm.Config.Com1,
			"com1_dev":           &vm.Config.Com1Dev,
			"com2":               &vm.Config.Com2,
			"com2_dev":           &vm.Config.Com2Dev,
			"com3":               &vm.Config.Com3,
			"com3_dev":           &vm.Config.Com3Dev,
			"com4":               &vm.Config.Com4,
			"com4_dev":           &vm.Config.Com4Dev,
			"extra_args":         &vm.Config.ExtraArgs,
			"is_os":              &vm.Config.ISOs,
			"disks":              &vm.Config.Disks,
			"nics":               &vm.Config.Nics,
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
	}).Model(&vm).
		Updates(map[string]interface{}{
			"name":        &vm.Name,
			"description": &vm.Description,
			"vnc_port":    &vm.VNCPort,
		})

	if res.Error != nil {
		slog.Error("db update error", "err", res.Error)
		return errors.New("error updating VM")
	}
	return nil
}

func (vm *VM) String() string {
	return fmt.Sprintf("name: %s id: %s", vm.Name, vm.ID)
}

func (vm *VM) maybeForceKillVM() {
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
		_, err = copyFile(uefiVarFileTemplate, uefiVarsFile)
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
			args := []string{"/sbin/ifconfig", vmNic.NetDev, "create"}
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
					err := _switch.BridgeIfAddMember(thisSwitch.Name, vmNic.NetDev)
					if err != nil {
						slog.Error("failed to add nic to switch",
							"nicname", vmNic.Name,
							"nicid", vmNic.ID,
							"switchid", vmNic.SwitchId,
							"err", err,
						)
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

func (vm *VM) netCleanup() {

	vmNicsList, err := vm.GetNics()

	if err != nil {
		slog.Error("netStartup failed to get nics", "err", err)
		return
	}

	for _, vmNic := range vmNicsList {
		if vmNic.NetDevType == "TAP" || vmNic.NetDevType == "VMNET" {
			args := []string{"/sbin/ifconfig", vmNic.NetDev, "destroy"}
			cmd := exec.Command(config.Config.Sys.Sudo, args...)
			err := cmd.Run()
			if err != nil {
				slog.Error("failed to destroy network interface", "err", err)
			}

		} else if vmNic.NetDevType == "NETGRAPH" {
			// nothing to do for netgraph
		} else {
			slog.Error("unknown net type, can't clean up")
		}
		vmNic.NetDev = ""
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
				go setRunning(vm.ID, vm.proc.Pid())
				vm.mu.Lock()
				List.VmList[vm.ID].Status = RUNNING
				vm.mu.Unlock()
			case "ProcessDone":
				vm.log.Info("event", "code", event.Code, "message", event.Message)
			case "ProcessCrashed":
				vm.log.Info("exited, destroying")
				vm.maybeForceKillVM()
			default:
				vm.log.Info("event", "code", event.Code, "message", event.Message)
			}
		case <-vm.proc.DoneNotifier():
			vm.log.Info("stopped")
			vm.netCleanup()
			setStopped(vm.ID)
			vm.mu.Lock()
			List.VmList[vm.ID].Status = STOPPED
			List.VmList[vm.ID].VNCPort = 0
			List.VmList[vm.ID].BhyvePid = 0
			vm.mu.Unlock()
			vm.maybeForceKillVM()
			vm.log.Info("closing loop we are done")
			return
		}
	}
}

func (vm *VM) GetISOs() ([]iso.ISO, error) {
	var isos []iso.ISO
	slog.Debug("GetISOs", "vm", vm.ID, "ISOs", vm.Config.ISOs)
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

func (vm *VM) AttachNics(nicIds []string) error {
	ac := 0
	defer List.Mu.Unlock()
	List.Mu.Lock()
	if vm.Status != STOPPED {
		return errors.New("VM must be stopped before adding disk(s)")
	}
	occurred := map[string]bool{}
	var result []string

	for _, aNic := range nicIds {
		slog.Debug("checking vm nic exists", "vmnic", aNic)

		if occurred[aNic] != true {
			occurred[aNic] = true
			result = append(result, aNic)
		} else {
			slog.Error("duplicate nic id", "nic", aNic)
			return errors.New("nic may only be added once")
		}

		_, err := vm_nics.GetById(aNic)
		if err != nil {
			return err
		}

		slog.Debug("checking if nic is attached to another VM", "nic", aNic)
		allVms := GetAll()
		for _, aVm := range allVms {
			vmNics, err := aVm.GetNics()
			if err != nil {
				return err
			}
			if aVm.ID == vm.ID {
				if ac > 0 {
					slog.Error("nic attached twice", "nic", aNic, "vm", aVm.ID)
					return errors.New("nic attached twice")
				}
				ac += 1
				continue
			}
			for _, aVmNic := range vmNics {
				if aNic == aVmNic.ID {
					slog.Error("nic is already attached to VM", "disk", aNic, "vm", aVm.ID)
					return errors.New("nic already attached")
				}
			}
		}
	}

	var nicsConfigVal string
	count := 0
	for _, diskId := range nicIds {
		if count > 0 {
			nicsConfigVal += ","
		}
		nicsConfigVal += diskId
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
	ac := 0
	defer List.Mu.Unlock()
	List.Mu.Lock()
	if vm.Status != STOPPED {
		return errors.New("VM must be stopped before adding disk(s)")
	}

	occurred := map[string]bool{}
	var result []string

	for _, aDisk := range diskids {
		slog.Debug("checking disk exists", "disk", aDisk)

		if occurred[aDisk] != true {
			occurred[aDisk] = true
			result = append(result, aDisk)
		} else {
			slog.Error("duplicate disk id", "disk", aDisk)
			return errors.New("disk may only be added once")
		}

		_, err := disk.GetById(aDisk)
		if err != nil {
			return err
		}

		slog.Debug("checking if disk is attached to another VM", "disk", aDisk)
		allVms := GetAll()
		for _, aVm := range allVms {
			vmDisks, err := aVm.GetDisks()
			if err != nil {
				return err
			}
			if aVm.ID == vm.ID {
				if ac > 0 {
					slog.Error("disk attached twice", "disk", aDisk, "vm", aVm.ID)
					return errors.New("disk attached twice")

				}
				ac += 1
				continue
			}
			for _, aVmDisk := range vmDisks {
				if aDisk == aVmDisk.ID {
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
