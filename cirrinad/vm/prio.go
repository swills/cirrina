package vm

import (
	"log/slog"
	"strconv"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

func (v *VM) applyResourceLimits() {
	if v.proc == nil || v.proc.Pid() == 0 || v.BhyvePid == 0 {
		slog.Error("attempted to apply resource limits to v that may not be running")

		return
	}

	v.log.Debug("checking resource limits")

	if v.Config.Pcpu > 0 {
		v.applyResourceLimitCPU()
	}

	if v.Config.Rbps > 0 {
		v.applyResourceLimitReadBPS()
	}

	if v.Config.Wbps > 0 {
		v.applyResourceLimitWriteBPS()
	}

	if v.Config.Riops > 0 {
		v.applyResourceLimitReadIOPS()
	}

	if v.Config.Wiops > 0 {
		v.applyResourceLimitWriteIOPS()
	}
}

func (v *VM) applyResourceLimitWriteIOPS() {
	v.log.Debug("Setting wiops limit")
	wiopsLimitStr := strconv.FormatUint(uint64(v.Config.Wiops), 10)
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{
			"/usr/bin/rctl", "-a", "process:" +
				strconv.FormatUint(uint64(v.BhyvePid), 10) + ":writeiops:throttle=" + wiopsLimitStr,
		},
	)

	if err != nil {
		slog.Error("failed to set resource limit",
			"BhyvePid", v.BhyvePid,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}
}

func (v *VM) applyResourceLimitReadIOPS() {
	v.log.Debug("Setting riops limit")
	riopsLimitStr := strconv.FormatUint(uint64(v.Config.Riops), 10)
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{
			"/usr/bin/rctl", "-a", "process:" +
				strconv.FormatUint(uint64(v.BhyvePid), 10) + ":readiops:throttle=" + riopsLimitStr,
		},
	)

	if err != nil {
		slog.Error("failed to set resource limit",
			"BhyvePid", v.BhyvePid,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}
}

func (v *VM) applyResourceLimitWriteBPS() {
	v.log.Debug("Setting wbps limit")
	wbpsLimitStr := strconv.FormatUint(uint64(v.Config.Wbps), 10)
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{
			"/usr/bin/rctl", "-a", "process:" +
				strconv.FormatUint(uint64(v.BhyvePid), 10) + ":writebps:throttle=" + wbpsLimitStr,
		},
	)

	if err != nil {
		slog.Error("failed to set resource limit",
			"BhyvePid", v.BhyvePid,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}
}

func (v *VM) applyResourceLimitReadBPS() {
	v.log.Debug("Setting rbps limit")
	rbpsLimitStr := strconv.FormatUint(uint64(v.Config.Rbps), 10)
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{
			"/usr/bin/rctl", "-a", "process:" +
				strconv.FormatUint(uint64(v.BhyvePid), 10) + ":readbps:throttle=" + rbpsLimitStr,
		},
	)

	if err != nil {
		slog.Error("failed to set resource limit",
			"BhyvePid", v.BhyvePid,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}
}

func (v *VM) applyResourceLimitCPU() {
	v.log.Debug("Setting pcpu limit")
	cpuLimitStr := strconv.FormatUint(uint64(v.Config.Pcpu), 10)
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{
			"/usr/bin/rctl", "-a", "process:" +
				strconv.FormatUint(uint64(v.BhyvePid), 10) + ":pcpu:deny=" + cpuLimitStr,
		},
	)

	if err != nil {
		slog.Error("failed to set resource limit",
			"BhyvePid", v.BhyvePid,
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}
}
