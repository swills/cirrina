package vm

import (
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/iso"
	"cirrina/cirrinad/util"
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
	if vm.Config.Net {
		vm.netStartup()
	}
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
			"kbd_layout":         &vm.Config.KbdLayout,
			"auto_start":         &vm.Config.AutoStart,
			"net_dev_type":       &vm.Config.NetDevType,
			"net_type":           &vm.Config.NetType,
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
			"net_dev":     &vm.NetDev,
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

func ngCreateBridge(netDev string, bridgePeer string) (err error) {
	if netDev == "" {
		return errors.New("netDev can't be empty")
	}
	if bridgePeer == "" {
		return errors.New("bridgePeer can't be empty")
	}
	cmd := exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "mkpeer",
		bridgePeer+":", "bridge", "lower", "link0")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl mkpeer error", "err", err)
		return err
	}
	cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "name",
		bridgePeer+":lower", netDev)
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl name err", "err", err)
		return err
	}
	useUplink := true
	var upper string
	if useUplink {
		upper = "uplink"
	} else {
		upper = "link"
	}
	cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "connect",
		bridgePeer+":", netDev+":", "upper", upper+"1")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl connect error", "err", err)
		return err
	}
	cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "msg",
		bridgePeer+":", "setpromisc", "1")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl msg error", "err", err)
		return err
	}
	cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "msg",
		bridgePeer+":", "setautosrc", "0")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl msg error", "err", err)
		return err
	}
	return nil
}

func (vm *VM) netStartup() {
	if vm.Config.NetDevType == "TAP" || vm.Config.NetDevType == "VMNET" {
		args := []string{"/sbin/ifconfig", vm.NetDev, "create"}
		cmd := exec.Command(config.Config.Sys.Sudo, args...)
		err := cmd.Run()
		if err != nil {
			slog.Error("failed to create tap", "err", err)
		}
		args = []string{"/sbin/ifconfig", config.Config.Network.Bridge, "addm", vm.NetDev}
		cmd = exec.Command(config.Config.Sys.Sudo, args...)
		err = cmd.Run()
		if err != nil {
			slog.Error("failed to add tap to bridge", "err", err)
		}
	} else if vm.Config.NetDevType == "NETGRAPH" {
		bridgeList, err := ngGetBridges()
		if err != nil {
			slog.Error("error getting bridge list", "err", err)
			return
		}
		if !containsStr(bridgeList, vm.NetDev) {
			err := ngCreateBridge(vm.NetDev, config.Config.Network.Interface)
			if err != nil {
				slog.Error("ngCreateBridge err", "err", err)
			}
		}
	} else {
		slog.Debug("unknown net type, can't set up")
		return
	}
}

func (vm *VM) netCleanup() {
	if vm.NetDev == "" {
		return
	}
	if strings.HasPrefix(vm.NetDev, "tap") || strings.HasPrefix(vm.NetDev, "vmnet") {
		args := []string{"/sbin/ifconfig", vm.NetDev, "destroy"}
		cmd := exec.Command(config.Config.Sys.Sudo, args...)
		err := cmd.Run()
		if err != nil {
			slog.Error("failed to destroy network interface", "err", err)
		}
	} else if strings.HasPrefix(vm.NetDev, "bnet") {
		// TODO - nothing to do for now, later this will need to check and destroy the netgraph bridge
		return
	} else {
		slog.Debug("unknown net type, can't clean up")
		return
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
			List.VmList[vm.ID].NetDev = ""
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

func (vm *VM) GetDisks() ([]disk.Disk, error) {
	var disks []disk.Disk
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

func (vm *VM) AttachDisk(diskids []string) error {
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
				// skip if it's attached to this VM, we replace the full list
				// TODO check that we don't attach the same disk twice
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
