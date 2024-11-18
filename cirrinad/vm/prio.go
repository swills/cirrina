package vm

import (
	"log/slog"
	"strconv"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

func (vm *VM) applyResourceLimits() {
	if vm.proc == nil || vm.proc.Pid() == 0 || vm.BhyvePid == 0 {
		slog.Error("attempted to apply resource limits to vm that may not be running")

		return
	}

	vm.log.Debug("checking resource limits")

	if vm.Config.Pcpu > 0 {
		vm.applyResourceLimitCPU()
	}

	if vm.Config.Rbps > 0 {
		vm.applyResourceLimitReadBPS()
	}

	if vm.Config.Wbps > 0 {
		vm.applyResourceLimitWriteBPS()
	}

	if vm.Config.Riops > 0 {
		vm.applyResourceLimitReadIOPS()
	}

	if vm.Config.Wiops > 0 {
		vm.applyResourceLimitWriteIOPS()
	}
}

func (vm *VM) applyResourceLimitWriteIOPS() {
	vm.log.Debug("Setting wiops limit")
	wiopsLimitStr := strconv.FormatUint(uint64(vm.Config.Wiops), 10)
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{
			"/usr/bin/rctl", "-a", "process:" +
				strconv.FormatUint(uint64(vm.BhyvePid), 10) + ":writeiops:throttle=" + wiopsLimitStr,
		},
	)

	if err != nil {
		slog.Error("failed to set resource limit",
			"BhyvePid", vm.BhyvePid,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}
}

func (vm *VM) applyResourceLimitReadIOPS() {
	vm.log.Debug("Setting riops limit")
	riopsLimitStr := strconv.FormatUint(uint64(vm.Config.Riops), 10)
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{
			"/usr/bin/rctl", "-a", "process:" +
				strconv.FormatUint(uint64(vm.BhyvePid), 10) + ":readiops:throttle=" + riopsLimitStr,
		},
	)

	if err != nil {
		slog.Error("failed to set resource limit",
			"BhyvePid", vm.BhyvePid,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}
}

func (vm *VM) applyResourceLimitWriteBPS() {
	vm.log.Debug("Setting wbps limit")
	wbpsLimitStr := strconv.FormatUint(uint64(vm.Config.Wbps), 10)
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{
			"/usr/bin/rctl", "-a", "process:" +
				strconv.FormatUint(uint64(vm.BhyvePid), 10) + ":writebps:throttle=" + wbpsLimitStr,
		},
	)

	if err != nil {
		slog.Error("failed to set resource limit",
			"BhyvePid", vm.BhyvePid,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}
}

func (vm *VM) applyResourceLimitReadBPS() {
	vm.log.Debug("Setting rbps limit")
	rbpsLimitStr := strconv.FormatUint(uint64(vm.Config.Rbps), 10)
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{
			"/usr/bin/rctl", "-a", "process:" +
				strconv.FormatUint(uint64(vm.BhyvePid), 10) + ":readbps:throttle=" + rbpsLimitStr,
		},
	)

	if err != nil {
		slog.Error("failed to set resource limit",
			"BhyvePid", vm.BhyvePid,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}
}

func (vm *VM) applyResourceLimitCPU() {
	vm.log.Debug("Setting pcpu limit")
	cpuLimitStr := strconv.FormatUint(uint64(vm.Config.Pcpu), 10)
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{
			"/usr/bin/rctl", "-a", "process:" +
				strconv.FormatUint(uint64(vm.BhyvePid), 10) + ":pcpu:deny=" + cpuLimitStr,
		},
	)

	if err != nil {
		slog.Error("failed to set resource limit",
			"BhyvePid", vm.BhyvePid,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}
}
