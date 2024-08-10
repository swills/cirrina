package main

import (
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/requests"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm"
)

func cleanupVms() error {
	var err error

	// deal with any leftover running VMs
	vmList := vm.GetAll()

	for _, aVM := range vmList {
		vmmPath := "/dev/vmm/" + aVM.Name
		slog.Debug("checking VM", "name", aVM.Name, "path", vmmPath)

		err = aVM.SetStopped()
		if err != nil {
			slog.Error("error stopping VM", "err", err)
		}

		exists, err := util.PathExists(vmmPath)
		if err != nil {
			slog.Error("error checking VM", "err", err)

			return fmt.Errorf("error cleaning VM: %w", err)
		}

		if !exists {
			continue
		}

		slog.Debug("leftover VM exists, checking pid", "name", aVM.Name, "pid", aVM.BhyvePid)

		var pidStat bool
		// check pid
		if aVM.BhyvePid > 0 {
			pidStat, err = util.PidExists(int(aVM.BhyvePid))
			if err != nil {
				slog.Error("error checking VM", "err", err)
			}
		}

		if pidStat {
			slog.Debug("leftover VM process running, stopping",
				"name", aVM.Name,
				"pid", aVM.BhyvePid,
				"maxWait", aVM.Config.MaxWait,
			)
			aVM.SetStopping()
			killLeftoverVM(aVM)
		}

		slog.Debug("destroying VM", "name", aVM.Name)
		aVM.BhyvectlDestroy()

		err = aVM.SetStopped()
		if err != nil {
			slog.Error("error stopping VM", "err", err)
		}
	}

	return nil
}

func killLeftoverVM(aVM *vm.VM) {
	var err error

	var pidStat bool

	var sleptTime time.Duration

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/bin/kill", strconv.FormatUint(uint64(aVM.BhyvePid), 10)},
	)
	if err != nil {
		slog.Error("ifconfig error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return
	}

	for {
		pidStat, err = util.PidExists(int(aVM.BhyvePid))
		if err != nil {
			slog.Error("error checking VM", "err", err)

			break
		}

		if !pidStat {
			break
		}

		time.Sleep(10 * time.Millisecond)

		sleptTime += 10 * time.Millisecond
		if sleptTime > (time.Duration(aVM.Config.MaxWait) * time.Second) {
			break
		}
	}

	pidStillExists, err := util.PidExists(int(aVM.BhyvePid))
	if err != nil {
		slog.Error("error checking VM", "err", err)
	} else if pidStillExists {
		slog.Error("VM refused to die")
	}
}

func cleanupNet() error {
	var err error

	// clean up leftover VM nets and mark everything stopped
	vmList := vm.GetAll()

	for _, aVM := range vmList {
		slog.Debug("cleaning up VM net(s)", "name", aVM.Name)
		aVM.NetCleanup()
		slog.Debug("marking VM stopped", "name", aVM.Name)

		err = aVM.SetStopped()
		if err != nil {
			slog.Error("error stopping VM", "err", err)
		}
	}

	// destroy all the bridges we know about
	err = _switch.DestroyBridges()
	if err != nil {
		panic(err)
	}

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
