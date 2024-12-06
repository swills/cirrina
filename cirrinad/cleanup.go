package main

import (
	"log/slog"
	"net"
	"strings"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/requests"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
)

// cleanupVms checks for leftover VMs and ensures they are killed and marked as stopped in the DB
// this is meant to handle two cases:
// * When the cirrinad process dies and leaves leftover VMs processes
// * When the host was not properly shut down and there are leftover pid files and the DB status is wrong
// In the first case, we have to kill VMs, remove pid files and update the DB
// In the second case, we only have to remove pid files and update the DB
func cleanupVms() error {
	var err error

	vmList := vm.GetAll()

	for _, aVM := range vmList {
		var pidStat bool
		if aVM.BhyvePid > 0 {
			pidStat, err = util.PidExists(int(aVM.BhyvePid))
			if err != nil {
				slog.Error("error checking VM", "err", err)
			} else if pidStat {
				aVM.SetStopping()
				aVM.Kill()
			}
		}

		aVM.BhyvectlDestroy()

		err = aVM.SetStopped()
		if err != nil {
			slog.Error("error stopping VM", "err", err)
		}
	}

	return nil
}

func cleanupNet() error {
	var err error

	// clean up leftover VM nets and mark everything stopped
	vmList := vm.GetAll()

	for _, aVM := range vmList {
		slog.Debug("cleaning up VM net(s)", "name", aVM.Name)
		aVM.NetStop()

		slog.Debug("marking VM stopped", "name", aVM.Name)

		err = aVM.SetStopped()
		if err != nil {
			slog.Error("error stopping VM", "err", err)
		}
	}

	// destroy all the bridges we know about
	err = _switch.DestroySwitches()
	if err != nil {
		slog.Error("error destroying switches", "err", err)
		panic(err)
	}

	// look for network things in cirrinad group and destroy them
	netInterfaces, err := net.Interfaces()
	if err != nil {
		slog.Error("error getting interfaces", "err", err)
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

		stdOutBytes, stdErrBytes, rc, err := util.RunCmd(
			config.Config.Sys.Sudo, []string{"/sbin/ifconfig", inter.Name, "destroy"})
		if string(stdErrBytes) != "" || rc != 0 || err != nil {
			slog.Error("error running command", "stdOutBytes", stdOutBytes, "stdErrBytes", stdErrBytes, "rc", rc, "err", err)
		}
	}

	return nil
}

func cleanupDB() {
	rowsCleared := requests.FailAllPending()
	slog.Debug("cleared failed requests", "rowsCleared", rowsCleared)

	allDisks := disk.GetAllDB()
	for _, diskInst := range allDisks {
		if strings.HasSuffix(diskInst.Name, ".img") {
			newName := strings.TrimSuffix(diskInst.Name, ".img")
			slog.Debug("renaming disk", "name", diskInst.Name, "newName", newName)
			diskInst.Name = newName

			err := diskInst.Save()
			if err != nil {
				slog.Error("cleanupDB failed saving new disk name", "err", err)
			}
		}
	}
}

func cleanupSystem() error {
	err := cleanupVms()
	if err != nil {
		return err
	}

	err = cleanupNet()
	if err != nil {
		return err
	}

	cleanupDB()

	return nil
}
