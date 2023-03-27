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

func (vm *VM) getDiskArg(slot int) ([]string, int) {
	diskType := "nvme"
	diskPath := "/bhyve/disk/" + vm.Name + ".img"
	diskArg := []string{"-s", strconv.Itoa(slot) + "," + diskType + "," + diskPath}
	slot = slot + 1
	return diskArg, slot
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

func (vm *VM) getHostBridgeArg(slot int) ([]string, int) {
	if !vm.Config.HostBridge {
		return []string{}, slot
	}
	hostBridgeArg := []string{"-s", strconv.Itoa(slot) + ",hostbridge"}
	slot = slot + 1
	return hostBridgeArg, slot
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

func (vm *VM) getLPCArg(slot int) ([]string, int) {
	return []string{"-s", "31,lpc"}, slot
}

func (vm *VM) getTabletArg(slot int) ([]string, int) {
	if !vm.Config.Screen || !vm.Config.Tablet {
		return []string{}, slot
	}
	tabletArg := []string{"-s", strconv.Itoa(slot) + ",xhci,tablet"}
	slot = slot + 1
	return tabletArg, slot
}

func (vm *VM) getVideoArg(slot int) ([]string, int) {
	if !vm.Config.Screen {
		return []string{}, slot
	}
	fbufArg := []string{"-s",
		strconv.Itoa(slot) + ",fbuf,w=" + strconv.Itoa(int(vm.Config.ScreenWidth)) +
			",h=" + strconv.Itoa(int(vm.Config.ScreenHeight)) + ",tcp=0.0.0.0:6900",
	}
	slot = slot + 1
	return fbufArg, slot
}

func (vm *VM) getCOMArg() []string {
	return []string{}
}

func (vm *VM) getNetArg(slot int) ([]string, int) {
	if !vm.Config.Net {
		return []string{}, slot
	}
	netArg := []string{"-s", strconv.Itoa(slot) + ",virtio-net,tap0,mac=00:a0:98:33:3c:93"}
	slot = slot + 1
	return netArg, slot
}
