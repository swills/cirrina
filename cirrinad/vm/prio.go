package vm

import (
	"log/slog"
	"strconv"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

func (vm *VM) applyResourceLimits(vmPid uint32) {
	if vm.proc == nil || vm.proc.Pid() == 0 || vm.BhyvePid == 0 {
		slog.Error("attempted to apply resource limits to vm that may not be running")

		return
	}

	actualVMPidStr := strconv.FormatUint(uint64(vmPid), 10)

	vm.log.Debug("checking resource limits")

	if vm.Config.Pcpu > 0 {
		applyResourceLimitCPU(actualVMPidStr, vm)
	}

	if vm.Config.Rbps > 0 {
		applyResourceLimitReadBPS(actualVMPidStr, vm)
	}

	if vm.Config.Wbps > 0 {
		applyResourceLimitWriteBPS(actualVMPidStr, vm)
	}

	if vm.Config.Riops > 0 {
		applyResourceLimitReadIOPS(actualVMPidStr, vm)
	}

	if vm.Config.Wiops > 0 {
		applyResourceLimitWriteIOPS(actualVMPidStr, vm)
	}
}

func applyResourceLimitWriteIOPS(vmPid string, vm *VM) {
	vm.log.Debug("Setting wiops limit")
	wiopsLimitStr := strconv.FormatUint(uint64(vm.Config.Wiops), 10)
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/bin/rctl", "-a", "process:" + vmPid + ":writeiops:throttle=" + wiopsLimitStr},
	)

	if err != nil {
		slog.Error("failed to set resource limit",
			"vmPid", vmPid,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}
}

func applyResourceLimitReadIOPS(vmPid string, vm *VM) {
	vm.log.Debug("Setting riops limit")
	riopsLimitStr := strconv.FormatUint(uint64(vm.Config.Riops), 10)
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/bin/rctl", "-a", "process:" + vmPid + ":readiops:throttle=" + riopsLimitStr},
	)

	if err != nil {
		slog.Error("failed to set resource limit",
			"vmPid", vmPid,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}
}

func applyResourceLimitWriteBPS(vmPid string, vm *VM) {
	vm.log.Debug("Setting wbps limit")
	wbpsLimitStr := strconv.FormatUint(uint64(vm.Config.Wbps), 10)
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/bin/rctl", "-a", "process:" + vmPid + ":writebps:throttle=" + wbpsLimitStr},
	)

	if err != nil {
		slog.Error("failed to set resource limit",
			"vmPid", vmPid,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}
}

func applyResourceLimitReadBPS(vmPid string, vm *VM) {
	vm.log.Debug("Setting rbps limit")
	rbpsLimitStr := strconv.FormatUint(uint64(vm.Config.Rbps), 10)
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/bin/rctl", "-a", "process:" + vmPid + ":readbps:throttle=" + rbpsLimitStr},
	)

	if err != nil {
		slog.Error("failed to set resource limit",
			"vmPid", vmPid,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}
}

func applyResourceLimitCPU(vmPid string, vm *VM) {
	vm.log.Debug("Setting pcpu limit")
	cpuLimitStr := strconv.FormatUint(uint64(vm.Config.Pcpu), 10)
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/bin/rctl", "-a", "process:" + vmPid + ":pcpu:deny=" + cpuLimitStr},
	)

	if err != nil {
		slog.Error("failed to set resource limit",
			"vmPid", vmPid,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}
}
