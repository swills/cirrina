package vm

import (
	"log/slog"
	"strconv"

	"golang.org/x/sys/execabs"

	"cirrina/cirrinad/config"
)

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
		applyResourceLimitCPU(vmPid, vm)
	}

	if vm.Config.Rbps > 0 {
		applyResourceLimitReadBPS(vmPid, vm)
	}

	if vm.Config.Wbps > 0 {
		applyResourceLimitWriteBPS(vmPid, vm)
	}

	if vm.Config.Riops > 0 {
		applyResourceLimitsReadIOPS(vmPid, vm)
	}

	if vm.Config.Wiops > 0 {
		applyResourceLimitWriteIOPS(vmPid, vm)
	}
}

func applyResourceLimitWriteIOPS(vmPid string, vm *VM) {
	vm.log.Debug("Setting wiops limit")
	wiopsLimitStr := strconv.FormatUint(uint64(vm.Config.Wiops), 10)
	args := []string{"/usr/bin/rctl", "-a", "process:" + vmPid + ":writeiops:throttle=" + wiopsLimitStr}
	cmd := execabs.Command(config.Config.Sys.Sudo, args...)

	err := cmd.Run()
	if err != nil {
		slog.Error("failed to set resource limit", "err", err)
	}
}

func applyResourceLimitsReadIOPS(vmPid string, vm *VM) {
	vm.log.Debug("Setting riops limit")
	riopsLimitStr := strconv.FormatUint(uint64(vm.Config.Riops), 10)
	args := []string{"/usr/bin/rctl", "-a", "process:" + vmPid + ":readiops:throttle=" + riopsLimitStr}
	cmd := execabs.Command(config.Config.Sys.Sudo, args...)

	err := cmd.Run()
	if err != nil {
		slog.Error("failed to set resource limit", "err", err)
	}
}

func applyResourceLimitWriteBPS(vmPid string, vm *VM) {
	vm.log.Debug("Setting wbps limit")
	wbpsLimitStr := strconv.FormatUint(uint64(vm.Config.Wbps), 10)
	args := []string{"/usr/bin/rctl", "-a", "process:" + vmPid + ":writebps:throttle=" + wbpsLimitStr}
	cmd := execabs.Command(config.Config.Sys.Sudo, args...)

	err := cmd.Run()
	if err != nil {
		slog.Error("failed to set resource limit", "err", err)
	}
}

func applyResourceLimitReadBPS(vmPid string, vm *VM) {
	vm.log.Debug("Setting rbps limit")
	rbpsLimitStr := strconv.FormatUint(uint64(vm.Config.Rbps), 10)
	args := []string{"/usr/bin/rctl", "-a", "process:" + vmPid + ":readbps:throttle=" + rbpsLimitStr}
	cmd := execabs.Command(config.Config.Sys.Sudo, args...)

	err := cmd.Run()
	if err != nil {
		slog.Error("failed to set resource limit", "err", err)
	}
}

func applyResourceLimitCPU(vmPid string, vm *VM) {
	vm.log.Debug("Setting pcpu limit")
	cpuLimitStr := strconv.FormatUint(uint64(vm.Config.Pcpu), 10)
	args := []string{"/usr/bin/rctl", "-a", "process:" + vmPid + ":pcpu:deny=" + cpuLimitStr}
	cmd := execabs.Command(config.Config.Sys.Sudo, args...)

	err := cmd.Run()
	if err != nil {
		slog.Error("failed to set resource limit", "err", err)
	}
}
