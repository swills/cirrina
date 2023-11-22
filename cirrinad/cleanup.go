package main

import (
	"bytes"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
	"log/slog"
	"golang.org/x/sys/execabs"
	"net"
	"syscall"
	"time"
)

func cleanUpVms() {
	vmList := vm.GetAll()
	// deal with any leftover running VMs
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
			} else {
				if pidStillExists {
					slog.Error("VM refused to die")
				}
			}
		}
		slog.Debug("destroying VM", "name", aVm.Name)
		aVm.MaybeForceKillVM()
	}

	// clean up leftover nets and mark everything stopped
	for _, aVm := range vmList {
		slog.Debug("cleaning up VM net(s)", "name", aVm.Name)
		aVm.NetCleanup()
		slog.Debug("marking VM stopped", "name", aVm.Name)
		aVm.SetStopped()
	}
}

func cleanupNet() {
	// destroy all the bridges we know about
	_switch.DestroyBridges()

	// look for network things in cirrinad group and destroy them
	netInterfaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	slog.Debug("GetHostInterfaces", "netInterfaces", netInterfaces)
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
}

func cleanupSystem() {
	cleanUpVms()
	cleanupNet()
	cleanupDb()
}
