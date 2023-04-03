package vm

import (
	"net"
	"strconv"
	"strings"
)

func (vm *VM) getKeyboardArg() []string {
	// TODO -- see old weasel code...
	return []string{}
}

func (vm *VM) getACPIArg() []string {
	if vm.Config.ACPI {
		return []string{"-A"}
	}
	return []string{}
}

func (vm *VM) getCDArg() []string {
	// TODO -- see old weasel code... really should use a list...
	return []string{}
}

func (vm *VM) getCpuArg() []string {
	return []string{"-c", strconv.Itoa(int(vm.Config.Cpu))}
}

func (vm *VM) getDiskArg(slot int) ([]string, int) {
	// TODO check that disk is enabled
	// TODO check that disk file exists
	// TODO use disk list, etc.
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
	// TODO just get this from DB
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
	uefiVarsPath := baseVMStatePath + "/" + vm.Name + "/BHYVE_UEFI_VARS.fd"
	// TODO check that storing uefi vars is enabled,
	//   if so, include vars file path and check it exists, copy it if not
	//   if not, just include rom path
	// TODO check that uefiVarsPath exists, if not, copy from template file
	return []string{
		"-l",
		"bootrom," + bootRomPath + "," + uefiVarsPath,
	}
}

func (vm *VM) getSoundArg() []string {
	// TODO -- see old weasel code...
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
	// TODO see old weasel code, mostly just need to rename to function to find free nmdm dev
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

	vncListenIP := "0.0.0.0"
	var vncListenPort string

	if vm.Config.VNCPort == "AUTO" {
		// this is a terrible way to select a port, but oh well
		vncListenPort = strconv.Itoa(6900 + len(vmProcesses))
	} else {
		vncListenPort = vm.Config.VNCPort
	}
	vncListenPortTmp, _ := strconv.Atoi(vncListenPort)
	vm.setVNCPort(vncListenPortTmp)

	fbufArg := []string{"-s",
		strconv.Itoa(slot) +
			",fbuf" +
			",w=" + strconv.Itoa(int(vm.Config.ScreenWidth)) +
			",h=" + strconv.Itoa(int(vm.Config.ScreenHeight)) +
			",tcp=" + vncListenIP + ":" + vncListenPort,
	}
	if vm.Config.VNCWait {
		fbufArg[1] = fbufArg[1] + ",wait"
	}
	slot = slot + 1
	return fbufArg, slot
}

func (vm *VM) getCOMArg() []string {
	// TODO -- see old weasel code...
	return []string{}
}

func contains(elems []string, v string) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

func (vm *VM) getNetArg(slot int) ([]string, int) {
	// TODO -- see old weasel code... -- mostly just handing other net types
	if !vm.Config.Net {
		return []string{}, slot
	}
	netType := "virtio-net"
	freeTapDevFound := false
	var tapDevs []string
	tapDev := ""
	tapNum := 0
	interfaces, _ := net.Interfaces()
	for _, inter := range interfaces {
		if strings.Contains(inter.Name, "tap") {
			tapDevs = append(tapDevs, inter.Name)
		}
	}
	for !freeTapDevFound {
		tapDev = "tap" + strconv.Itoa(tapNum)
		if !contains(tapDevs, tapDev) {
			freeTapDevFound = true
		} else {
			tapNum = tapNum + 1
		}
	}
	macAddress := vm.Config.Mac
	macString := ""
	if macAddress != "AUTO" {
		macString = ",mac=" + macAddress
	}
	netArg := []string{"-s", strconv.Itoa(slot) + "," + netType + "," + tapDev + macString}
	slot = slot + 1
	vm.netDev = tapDev
	return netArg, slot
}

func (vm *VM) generateCommandLine() (name string, args []string, err error) {
	name = "/usr/local/bin/sudo"
	slot := 0
	cpuArg := vm.getCpuArg()
	memArg := vm.getMemArg()
	acpiArg := vm.getACPIArg()
	haltArg := vm.getHLTArg()
	eopArg := vm.getEOPArg()
	wireArg := vm.getWireArg()
	dpoArg := vm.getDPOArg()
	msrArg := vm.getMSRArg()
	utcArg := vm.getUTCArg()
	romArg := vm.getROMArg()
	hostBridgeArg, slot := vm.getHostBridgeArg(slot)
	fbufArg, slot := vm.getVideoArg(slot)
	tabletArg, slot := vm.getTabletArg(slot)
	netArg, slot := vm.getNetArg(slot)
	diskArg, slot := vm.getDiskArg(slot)
	lpcArg, slot := vm.getLPCArg(slot)

	// TODO - add keyboard arg
	// TODO - add cd arg
	// TODO - add sound arg
	// TODO - add com args
	// TODO - add extra args

	args = append(args, "/usr/bin/protect")
	args = append(args, "/usr/sbin/bhyve")
	args = append(args, acpiArg...)
	args = append(args, haltArg...)
	args = append(args, eopArg...)
	args = append(args, wireArg...)
	args = append(args, dpoArg...)
	args = append(args, msrArg...)
	args = append(args, utcArg...)
	args = append(args, romArg...)
	args = append(args, cpuArg...)
	args = append(args, memArg...)
	args = append(args, hostBridgeArg...)
	args = append(args, fbufArg...)
	args = append(args, tabletArg...)
	args = append(args, netArg...)
	args = append(args, diskArg...)
	args = append(args, lpcArg...)
	args = append(args, vm.Name)
	return name, args, nil
}
