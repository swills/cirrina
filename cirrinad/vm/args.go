package vm

import "strconv"

func (vm *VM) getKeyboardArg() []string {
	return []string{}
}

func (vm *VM) getACPIArg() []string {
	if vm.Config.ACPITables {
		return []string{"-A"}
	}
	return []string{}
}

func (vm *VM) getCDArg() []string {
	return []string{}
}

func (vm *VM) getCpuArg() []string {
	return []string{"-c", strconv.Itoa(int(vm.Config.Cpu))}
}

func (vm *VM) getDiskArg() []string {
	diskType := "nvme"
	diskPath := "/bhyve/disk/" + vm.Name + ".img"
	return []string{"-s", "4," + diskType + "," + diskPath}
}

func (vm *VM) getDPOArg() []string {
	if vm.Config.DestroyPowerOff {
		return []string{"-D"}
	}
	return []string{}
}

func (vm *VM) getEOPArg() []string {
	if vm.Config.ExitOnPause {
		return []string{"-P"}
	}
	return []string{}
}

func (vm *VM) getExtraArg() []string {
	return []string{}
}

func (vm *VM) getHLTArg() []string {
	if vm.Config.UseHLT {
		return []string{"-H"}
	}
	return []string{}
}

func (vm *VM) getHostBridgeArg() []string {
	return []string{"-s", "0,hostbridge"}
}

func (vm *VM) getMemArg() []string {
	return []string{"-m", strconv.Itoa(int(vm.Config.Mem)) + "m"}
}

func (vm *VM) getMSRArg() []string {
	if vm.Config.IgnoreUnknownMSR {
		return []string{"-w"}
	}
	return []string{}
}

func (vm *VM) getROMArg() []string {
	bootRomPath := "/usr/local/share/uefi-firmware/BHYVE_UEFI.fd"
	uefiVarsPath := "/usr/home/swills/.local/state/weasel/vms/" + vm.Name + "/BHYVE_UEFI_VARS.fd"

	return []string{
		"-l",
		"bootrom," + bootRomPath + "," + uefiVarsPath,
	}
}

func (vm *VM) getSoundArg() []string {
	return []string{}
}

func (vm *VM) getUTCArg() []string {
	if vm.Config.UTCTime {
		return []string{"-u"}
	}
	return []string{}
}

func (vm *VM) getWireArg() []string {
	if vm.Config.WireGuestMem {
		return []string{"-S"}
	}
	return []string{}
}

func (vm *VM) getNMDMArg() []string {
	return []string{}
}

func (vm *VM) getLPCArg() []string {
	return []string{"-s", "31,lpc"}
}

func (vm *VM) getTabletArg() []string {
	return []string{"-s", "2,xhci,tablet"}
}

func (vm *VM) getVideoArg() []string {
	return []string{"-s", "1,fbuf,w=1920,h=1080,tcp=0.0.0.0:6900"}
}

func (vm *VM) getCOMArg() []string {
	return []string{}
}

func (vm *VM) getNetArg() []string {
	return []string{"-s", "3,virtio-net,tap0,mac=00:a0:98:33:3c:93"}
}
