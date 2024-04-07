package main

import (
	"bytes"
	"log/slog"
	"net"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/execabs"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/requests"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
)

func cleanupVms() {
	// deal with any leftover running VMs
	vmList := vm.GetAllDb()
	for _, aVm := range vmList {
		vmmPath := "/dev/vmm/" + aVm.Name
		slog.Debug("checking VM", "name", aVm.Name, "path", vmmPath)
		exists, err := util.PathExists(vmmPath)
		if err != nil {
			slog.Error("error checking VM", "err", err)

			continue
		}
		if !exists {
			continue
		}
		slog.Debug("leftover VM exists, checking pid", "name", aVm.Name, "pid", aVm.BhyvePid)
		var pidStat bool
		// check pid
		if aVm.BhyvePid > 0 {
			pidStat, err = util.PidExists(int(aVm.BhyvePid))
			if err != nil {
				slog.Error("error checking VM", "err", err)
			}
		}
		if pidStat {
			slog.Debug("leftover VM exists", "name", aVm.Name, "pid", aVm.BhyvePid, "maxWait", aVm.Config.MaxWait)
			killLeftoverVM(aVm)
		}
		slog.Debug("destroying VM", "name", aVm.Name)
		aVm.MaybeForceKillVM()
	}
}

func killLeftoverVM(aVm *vm.VM) {
	var err error
	var pidStat bool
	var sleptTime time.Duration
	_ = syscall.Kill(int(aVm.BhyvePid), syscall.SIGTERM)
	for {
		pidStat, err = util.PidExists(int(aVm.BhyvePid))
		if err != nil {
			slog.Error("error checking VM", "err", err)

			break
		}
		if !pidStat {
			break
		}
		time.Sleep(10 * time.Millisecond)
		sleptTime += 10 * time.Millisecond
		if sleptTime > (time.Duration(aVm.Config.MaxWait) * time.Second) {
			break
		}
	}
	pidStillExists, err := util.PidExists(int(aVm.BhyvePid))
	if err != nil {
		slog.Error("error checking VM", "err", err)
	} else if pidStillExists {
		slog.Error("VM refused to die")
	}
}

func cleanupNet() {
	// clean up leftover VM nets and mark everything stopped
	vmList := vm.GetAllDb()
	for _, aVm := range vmList {
		slog.Debug("cleaning up VM net(s)", "name", aVm.Name)
		aVm.NetCleanup()
		slog.Debug("marking VM stopped", "name", aVm.Name)
		aVm.SetStopped()
	}

	// destroy all the bridges we know about
	_switch.DestroyBridges()

	// look for network things in cirrinad group and destroy them
	netInterfaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	slog.Debug("cleanupNet", "netInterfaces", netInterfaces)
	for _, inter := range netInterfaces {
		intGroups, err := util.GetIntGroups(inter.Name)
		if err != nil {
			slog.Error("failed to get interface groups", "err", err)
		}
		if !util.ContainsStr(intGroups, "cirrinad") {
			continue
		}
		slog.Debug("leftover interface found, destroying", "name", inter.Name)

		cmd := execabs.Command(config.Config.Sys.Sudo, "/sbin/ifconfig", inter.Name, "destroy")
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Start(); err != nil {
			slog.Error("failed running ifconfig", "err", err, "out", out)
		}
		if err := cmd.Wait(); err != nil {
			slog.Error("failed running ifconfig", "err", err, "out", out)
		}
	}
}

func cleanupDb() {
	rowsCleared := requests.FailAllPending()
	slog.Debug("cleared failed requests", "rowsCleared", rowsCleared)
	allDisks := disk.GetAllDb()
	for _, diskInst := range allDisks {
		if strings.HasSuffix(diskInst.Name, ".img") {
			newName := strings.TrimSuffix(diskInst.Name, ".img")
			slog.Debug("renaming disk", "name", diskInst.Name, "newName", newName)
			diskInst.Name = newName
			err := diskInst.Save()
			if err != nil {
				slog.Error("cleanupDb failed saving new disk name", "err", err)
			}
		}
	}
}

func cleanupSystem() {
	cleanupVms()
	cleanupNet()
	cleanupDb()
}
