package vm

import (
	"cirrina/cirrinad/iso"
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
	log.Printf("Creating VM %v", name)
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
	log.Printf("cmd: %v, args: %v", cmdName, cmdArgs)
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
		log.Printf("tried to stop VM %v that is not running", vm.Name)
		return errors.New("must be running first")
	}
	defer vm.mu.Unlock()
	vm.mu.Lock()
	setStopping(vm.ID)
	err = vm.proc.Stop()
	if err != nil {
		log.Printf("Failed to stop %v", vm.proc.Pid())
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
		log.Printf("db update error: %v", res.Error)
		return errors.New("error updating VM")
	}
	return nil
}

func (vm *VM) String() string {
	return fmt.Sprintf("name: %s id: %s", vm.Name, vm.ID)
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
	// TODO make baseVMStatePath a config item
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

func ngCreateBridge(netDev string, bridgePeer string) (err error) {
	if netDev == "" {
		return errors.New("netDev can't be empty")
	}
	if bridgePeer == "" {
		return errors.New("bridgePeer can't be empty")
	}
	cmd := exec.Command("/usr/local/bin/sudo", "/usr/sbin/ngctl", "mkpeer",
		bridgePeer+":", "bridge", "lower", "link0")
	err = cmd.Run()
	if err != nil {
		log.Printf("ngctl mkpeer err: %v", err)
		return err
	}
	cmd = exec.Command("/usr/local/bin/sudo", "/usr/sbin/ngctl", "name",
		bridgePeer+":lower", netDev)
	err = cmd.Run()
	if err != nil {
		log.Printf("ngctl name err: %v", err)
		return err
	}
	useUplink := true
	var upper string
	if useUplink {
		upper = "uplink"
	} else {
		upper = "link"
	}
	cmd = exec.Command("/usr/local/bin/sudo", "/usr/sbin/ngctl", "connect",
		bridgePeer+":", netDev+":", "upper", upper+"1")
	err = cmd.Run()
	if err != nil {
		log.Printf("ngctl connect err: %v", err)
		return err
	}
	cmd = exec.Command("/usr/local/bin/sudo", "/usr/sbin/ngctl", "msg",
		bridgePeer+":", "setpromisc", "1")
	err = cmd.Run()
	if err != nil {
		log.Printf("ngctl msg err: %v", err)
		return err
	}
	cmd = exec.Command("/usr/local/bin/sudo", "/usr/sbin/ngctl", "msg",
		bridgePeer+":", "setautosrc", "0")
	err = cmd.Run()
	if err != nil {
		log.Printf("ngctl msg err: %v", err)
		return err
	}
	return nil
}

func (vm *VM) netStartup() {
	if vm.Config.NetDevType == "TAP" || vm.Config.NetDevType == "VMNET" {
		args := []string{"/sbin/ifconfig", vm.NetDev, "create"}
		cmd := exec.Command("/usr/local/bin/sudo", args...)
		err := cmd.Run()
		if err != nil {
			log.Printf("failed to create tap: %v", err)
		}
		args = []string{"/sbin/ifconfig", "bridge0", "addm", vm.NetDev}
		cmd = exec.Command("/usr/local/bin/sudo", args...)
		err = cmd.Run()
		if err != nil {
			log.Printf("failed to add tap to bridge: %v", err)
		}
	} else if vm.Config.NetDevType == "NETGRAPH" {
		bridgeList, err := ngGetBridges()
		if err != nil {
			log.Printf("error getting bridge list: %v", err)
			return
		}
		if !containsStr(bridgeList, vm.NetDev) {
			err := ngCreateBridge(vm.NetDev, "em0")
			if err != nil {
				log.Printf("ngCreateBridge err: %v", err)
			}
		}
	} else {
		log.Printf("unknown net type, can't set up")
		return
	}
}

func (vm *VM) netCleanup() {
	if vm.NetDev == "" {
		return
	}
	if strings.HasPrefix(vm.NetDev, "tap") || strings.HasPrefix(vm.NetDev, "vmnet") {
		args := []string{"/sbin/ifconfig", vm.NetDev, "destroy"}
		cmd := exec.Command("/usr/local/bin/sudo", args...)
		err := cmd.Run()
		if err != nil {
			log.Printf("failed to destroy network interface : %v", err)
		}
	} else if strings.HasPrefix(vm.NetDev, "bnet") {
		// TODO - nothing to do for now, later this will need to check and destroy the netgraph bridge
		return
	} else {
		log.Printf("unknown net type, can't clean up")
		return
	}
}

func vmDaemon(events chan supervisor.Event, vm *VM) {
	for {
		select {
		case msg := <-vm.proc.Stdout():
			log.Printf("VM %v Received STDOUT message: %s\n", vm.ID, *msg)
		case msg := <-vm.proc.Stderr():
			log.Printf("VM %v Received STDERR message: %s\n", vm.ID, *msg)
		case event := <-events:
			switch event.Code {
			case "ProcessStart":
				log.Printf("VM %v Received event: %s - %s\n", vm.ID, event.Code, event.Message)
				go setRunning(vm.ID, vm.proc.Pid())
				vm.mu.Lock()
				List.VmList[vm.ID].Status = RUNNING
				vm.mu.Unlock()
			case "ProcessDone":
				exitStatus := parseStopMessage(event.Message)
				log.Printf("stop message: %v", event.Message)
				log.Printf("VM %v stopped, exitStatus: %v", vm.ID, exitStatus)
				//     0       rebooted
				//     1       powered off
				//     2       halted
				//     3       triple fault
				//     4       exited due to an error
				if exitStatus == 0 {
					// set state to restarting?
					log.Printf("VM %v %v rebooted, allowing restart", vm.ID, vm.Name)
				}
			case "ProcessCrashed":
				log.Printf("VM %v %v crashed: %v, destroying", vm.ID, vm.Name, event.Message)
				vm.maybeForceKillVM()
				if vm.Config.Restart {
					log.Printf("VM %v %v restart enabled, allowing restart", vm.ID, vm.Name)
				} else {
					log.Printf("VM %v %v disabled, cleaning up", vm.ID, vm.Name)
					_ = vm.proc.Stop()
					vm.netCleanup()
					setStopped(vm.ID)
					vm.mu.Lock()
					List.VmList[vm.ID].Status = STOPPED
					vm.mu.Unlock()
				}
			default:
				log.Printf("VM %v Received event: %s - %s\n", vm.ID, event.Code, event.Message)
			}
		case <-vm.proc.DoneNotifier():
			log.Printf("VM %v %v done", vm.ID, vm.Name)
			vm.netCleanup()
			setStopped(vm.ID)
			vm.mu.Lock()
			List.VmList[vm.ID].Status = STOPPED
			List.VmList[vm.ID].NetDev = ""
			List.VmList[vm.ID].VNCPort = 0
			List.VmList[vm.ID].BhyvePid = 0
			vm.mu.Unlock()
			vm.maybeForceKillVM()
			log.Printf("VM %v closing loop we are done...", vm.ID)
			return
		}
	}
}

func (vm *VM) GetISOs() ([]iso.ISO, error) {
	var isos []iso.ISO
	for _, cv := range strings.Split(vm.Config.ISOs, ",") {
		if cv == "" {
			continue
		}
		aISO, err := iso.GetById(cv)
		if err == nil {
			isos = append(isos, *aISO)
		} else {
			log.Printf("bad iso %v for vm %v", cv, vm.ID)
		}
	}
	return isos, nil
}
