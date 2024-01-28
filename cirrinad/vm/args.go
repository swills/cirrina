package vm

import (
	"errors"
	"math"
	"net"
	"os"
	"strconv"
	"strings"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/iso"
	"cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm_nics"
	"github.com/rxwycdh/rxhash"
	"log/slog"
)

type MacHashData struct {
	VmId    string
	VmName  string
	NicId   string
	NicName string
}

func (vm *VM) getKeyboardArg() []string {
	if vm.Config.Screen && vm.Config.KbdLayout != "default" {
		return []string{"-K", vm.Config.KbdLayout}
	}
	return []string{}
}

func (vm *VM) getACPIArg() []string {
	if vm.Config.ACPI {
		return []string{"-A"}
	}
	return []string{}
}

func (vm *VM) getCDArg(slot int) ([]string, int) {
	var cdString []string
	maxSataDevs := 32 - slot - 1
	devCount := 0
	// TODO remove all these de-normalizations in favor of gorm native "Has Many" relationships

	isoList := strings.Split(vm.Config.ISOs, ",")
	for _, isoItem := range isoList {
		if isoItem == "" {
			continue
		}
		thisIso, err := iso.GetById(isoItem)
		if err != nil {
			slog.Error("error getting ISO", "isoItem", isoItem, "err", err)
			return []string{}, slot
		}
		if thisIso.Path == "" {
			slog.Error("empty iso path, correcting", "iso", thisIso.Name, "id", thisIso.ID, "path", thisIso.Path)
			thisIso.Path = config.Config.Disk.VM.Path.Iso + string(os.PathSeparator) + thisIso.Name
		}
		slog.Debug("getCDArg", "name", thisIso.Name, "id", thisIso.ID, "path", thisIso.Path)
		if devCount <= maxSataDevs {
			thisCd := []string{"-s", strconv.Itoa(slot) + ":0,ahci,cd:" + thisIso.Path}
			cdString = append(cdString, thisCd...)
			devCount = devCount + 1
			slot = slot + 1
		}
	}
	return cdString, slot
}

func (vm *VM) getCpuArg() []string {
	var vmCpus uint16
	hostCpus, err := util.GetHostMaxVmCpus()
	if err != nil {
		return []string{}
	}
	if vm.Config.Cpu > uint32(hostCpus) || vm.Config.Cpu > math.MaxUint16 {
		vmCpus = hostCpus
	} else {
		vmCpus = uint16(vm.Config.Cpu)
	}
	return []string{"-c", strconv.Itoa(int(vmCpus))}
}

func (vm *VM) getOneDiskArg(thisDisk *disk.Disk) (hdArg string, err error) {
	diskController := ""
	nocache := ""
	direct := ""

	diskPath, err := thisDisk.GetPath()
	if err != nil {
		slog.Error("error getting disk path", "diskId", thisDisk.ID, "diskName", thisDisk.Name, "diskPath", diskPath, "err", err)
		return "", err
	}
	diskExists, err := thisDisk.VerifyExists()
	if err != nil {
		slog.Error("error checking disk path exists", "diskId", thisDisk.ID, "diskName", thisDisk.Name, "diskPath", diskPath)
		return "", err
	}
	if !diskExists {
		slog.Error("disk path does not exist", "diskId", thisDisk.ID, "diskName", thisDisk.Name, "diskPath", diskPath)
		return "", err
	}
	if thisDisk.Type == "NVME" {
		diskController = "nvme"
	} else if thisDisk.Type == "AHCI-HD" {
		diskController = "ahci-hd"
	} else if thisDisk.Type == "VIRTIO-BLK" {
		diskController = "virtio-blk"
	} else {
		slog.Error("unknown disk type", "type", thisDisk.Type)
		return "", errors.New("unknown disk type")
	}
	if thisDisk.DiskCache.Valid && !thisDisk.DiskCache.Bool {
		nocache = ",nocache"
	}
	if thisDisk.DiskDirect.Valid && thisDisk.DiskDirect.Bool {
		direct = ",direct"
	}

	return diskController + "," + diskPath + nocache + direct, nil
}

func (vm *VM) getDiskArg(slot int) ([]string, int) {
	// TODO don't use one PCI slot per ahci (SATA) disk device, attach multiple disks to each controller
	maxSataDevs := 32 - slot - 1 // FIXME -- this is awful but needed until we attach multiple sata disks to each controller
	sataDevCount := 0

	var diskString []string
	// TODO remove all these de-normalizations in favor of gorm native "Has Many" relationships
	diskIds := strings.Split(vm.Config.Disks, ",")
	for _, diskId := range diskIds {
		if diskId == "" {
			continue
		}
		thisDisk, err := disk.GetById(diskId)
		if err != nil {
			slog.Error("error getting disk, skipping", "diskId", diskId, "err", err)
			continue
		}
		if thisDisk.Type == "AHCI-HD" {
			sataDevCount = sataDevCount + 1
		}
		if sataDevCount > maxSataDevs {
			slog.Error("sata dev count exceeded, skipping disk", "diskId", diskId, "diskName")
			continue
		}

		oneHdString, err := vm.getOneDiskArg(thisDisk)
		if err != nil || oneHdString == "" {
			slog.Error("error adding disk, skipping", "diskId", diskId, "err", err)
			continue
		}
		thisHd := []string{"-s", strconv.Itoa(slot) + "," + oneHdString}
		diskString = append(diskString, thisHd...)
		slot = slot + 1
	}
	return diskString, slot
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
	return strings.Fields(vm.Config.ExtraArgs)
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
	var romArg []string

	if vm.Config.StoreUEFIVars {
		uefiVarsPath := baseVMStatePath + "/" + vm.Name + "/BHYVE_UEFI_VARS.fd"
		romArg = []string{
			"-l",
			"bootrom," + bootRomPath + "," + uefiVarsPath,
		}
	} else {
		romArg = []string{
			"-l",
			"bootrom," + bootRomPath,
		}
	}

	return romArg
}

func (vm *VM) getDebugArg() []string {
	var debugArg []string

	firstDebugPort := config.Config.Debug.Port
	debugListenIP := config.Config.Debug.Ip
	var debugListenPortInt int
	var debugListenPort string
	var debugWaitStr string
	var err error

	if !vm.Config.Debug {
		return []string{}
	}

	if vm.Config.DebugPort == "AUTO" {
		usedDebugPorts := GetUsedDebugPorts()
		debugListenPortInt, err = util.GetFreeTCPPort(int(firstDebugPort), usedDebugPorts)
		if err != nil {
			return []string{}
		}
		debugListenPort = strconv.Itoa(debugListenPortInt)
	} else {
		debugListenPort = vm.Config.DebugPort
		debugListenPortInt, err = strconv.Atoi(debugListenPort)
		if err != nil {
			return []string{}
		}
	}
	vm.SetDebugPort(debugListenPortInt)

	if vm.Config.DebugWait {
		debugWaitStr = "w"
	}

	debugArg = []string{
		"-G",
		debugWaitStr + debugListenIP + ":" + debugListenPort,
	}
	return debugArg
}

func (vm *VM) getSoundArg(slot int) ([]string, int) {
	if !vm.Config.Sound {
		return []string{}, slot
	}
	var soundArg []string
	var soundString string
	inPathExists, err := util.PathExists(vm.Config.SoundIn)
	if err != nil {
		slog.Error("sound input check error", "err", err)
	}
	outPathExists, err := util.PathExists(vm.Config.SoundIn)
	if err != nil {
		slog.Error("sound output check error", "err", err)
	}
	if inPathExists || outPathExists {
		soundString = ",hda"
		if outPathExists {
			soundString = soundString + ",play=" + vm.Config.SoundOut
		} else {
			slog.Debug("sound output path does not exist", "path", vm.Config.SoundOut)
		}
		if inPathExists {
			soundString = soundString + ",rec=" + vm.Config.SoundIn
		} else {
			slog.Debug("sound input path does not exist", "path", vm.Config.SoundIn)
		}
	}
	soundArg = []string{"-s", strconv.Itoa(slot) + soundString}
	slot = slot + 1
	return soundArg, slot
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

	firstVncPort := config.Config.Vnc.Port
	vncListenIP := config.Config.Vnc.Ip
	var vncListenPortInt int
	var vncListenPort string
	var err error

	if vm.Config.VNCPort == "AUTO" {
		usedVncPorts := GetUsedVncPorts()
		vncListenPortInt, err = util.GetFreeTCPPort(int(firstVncPort), usedVncPorts)
		if err != nil {
			return []string{}, slot
		}
		vncListenPort = strconv.Itoa(vncListenPortInt)
	} else {
		vncListenPort = vm.Config.VNCPort
		vncListenPortInt, err = strconv.Atoi(vncListenPort)
		if err != nil {
			return []string{}, slot
		}
	}
	vm.SetVNCPort(vncListenPortInt)

	fbufArg := []string{
		"-s",
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

func (vm *VM) getNetArg(slot int) ([]string, int) {
	var netArgs []string

	originalSlot := slot

	// TODO remove all these de-normalizations in favor of gorm native "Has Many" relationships
	nicList := strings.Split(vm.Config.Nics, ",")
	for _, nicItem := range nicList {
		if nicItem == "" {
			continue
		}
		slog.Debug("adding nic", "nic", nicItem)
		thisNic, err := vm_nics.GetById(nicItem)
		if err != nil {
			slog.Error("error getting Disk", "nicItem", nicItem, "err", err)
			return []string{}, originalSlot
		}
		var netType string
		var netDevArg string

		if thisNic.NetType == "VIRTIONET" {
			netType = "virtio-net"
		} else if thisNic.NetType == "E1000" {
			netType = "e1000"
		} else {
			slog.Debug("unknown net type, cannot configure", "netType", thisNic.NetType)
			return []string{}, originalSlot
		}

		if thisNic.NetDevType == "TAP" {
			thisNic.NetDev = GetTapDev()
			netDevArg = thisNic.NetDev
			err := thisNic.Save()
			if err != nil {
				slog.Error("failed to save net dev", "nic", thisNic.ID, "netdev", thisNic.NetDev)
				return []string{}, slot
			}
			netDevArg = thisNic.NetDev
		} else if thisNic.NetDevType == "VMNET" {
			thisNic.NetDev = GetVmnetDev()
			netDevArg = thisNic.NetDev
			err := thisNic.Save()
			if err != nil {
				slog.Error("failed to save net dev", "nic", thisNic.ID, "netdev", thisNic.NetDev)
				return []string{}, slot
			}
			netDevArg = thisNic.NetDev
		} else if thisNic.NetDevType == "NETGRAPH" {
			ngNetDev, ngPeerHook, err := _switch.GetNgDev(thisNic.SwitchId)
			if err != nil {
				slog.Error("GetNgDev error", "err", err)
				return []string{}, slot
			}
			thisNic.NetDev = ngNetDev + "," + ngPeerHook
			err = thisNic.Save()
			if err != nil {
				slog.Error("failed to save net dev", "nic", thisNic.ID, "netdev", thisNic.NetDev)
				return []string{}, slot
			}
			netDevArg = "netgraph,path=" + ngNetDev + ":,peerhook=" + ngPeerHook + ",socket=" + vm.Name
		} else {
			slog.Debug("unknown net dev type", "netDevType", thisNic.NetDevType)
			return []string{}, slot
		}
		slog.Debug("getNetArg", "netdevarg", netDevArg)
		macAddress := GetMac(thisNic, vm)
		var macString string
		if macAddress != "" {
			macString = ",mac=" + macAddress
		}
		netArg := []string{"-s", strconv.Itoa(slot) + "," + netType + "," + netDevArg + macString}
		slot = slot + 1
		netArgs = append(netArgs, netArg...)
	}

	return netArgs, slot
}

func GetMac(thisNic *vm_nics.VmNic, vm *VM) string {
	var macAddress string
	if thisNic.Mac == "AUTO" {
		// if MAC is AUTO, we still generate our own here rather than letting bhyve generate it, because:
		// 1. Bhyve is still using the NetApp MAC:
		// https://cgit.freebsd.org/src/tree/usr.sbin/bhyve/net_utils.c?id=1d386b48a555f61cb7325543adbbb5c3f3407a66#n115
		// 2. We want to be able to distinguish our VMs from other VMs
		slog.Debug("getNetArg: Generating MAC")
		thisNicHashData := MacHashData{
			vm.ID,
			vm.Name,
			thisNic.ID,
			thisNic.Name,
		}
		h1, err := rxhash.HashStruct(thisNicHashData)
		if err != nil {
			slog.Error("getNetArg error generating mac", "err", err)
			return ""
		}
		slog.Debug("getNetArg", "h1", h1)
		mac := string(h1[0]) + string(h1[1]) + ":" +
			string(h1[2]) + string(h1[3]) + ":" +
			string(h1[4]) + string(h1[5])
		slog.Debug("getNetArg", "mac", mac)
		macAddress = config.Config.Network.Mac.Oui + ":" + mac
	} else {
		macAddress = thisNic.Mac
	}
	return macAddress
}

// TODO move to _switch

func GetTapDev() string {
	freeTapDevFound := false
	var netDevs []string
	tapDev := ""
	tapNum := 0
	interfaces, _ := net.Interfaces()
	for _, inter := range interfaces {
		netDevs = append(netDevs, inter.Name)
	}
	for !freeTapDevFound {
		tapDev = "tap" + strconv.Itoa(tapNum)
		if !util.ContainsStr(netDevs, tapDev) && !IsNetPortUsed(tapDev) {
			freeTapDevFound = true
		} else {
			tapNum = tapNum + 1
		}
	}
	return tapDev
}

// TODO move to _switch

func GetVmnetDev() string {
	freeVmnetDevFound := false
	var netDevs []string
	vmnetDev := ""
	vmnetNum := 0
	interfaces, _ := net.Interfaces()
	for _, inter := range interfaces {
		netDevs = append(netDevs, inter.Name)
	}
	for !freeVmnetDevFound {
		vmnetDev = "vmnet" + strconv.Itoa(vmnetNum)
		if !util.ContainsStr(netDevs, vmnetDev) && !IsNetPortUsed(vmnetDev) {
			freeVmnetDevFound = true
		} else {
			vmnetNum = vmnetNum + 1
		}
	}
	return vmnetDev
}

func getCom(comDev string, vmName string, num int) ([]string, string) {
	nmdm := ""
	var comArg []string
	if comDev == "AUTO" {
		nmdm = "/dev/nmdm-" + vmName + "-com" + strconv.Itoa(num) + "-A"
	} else {
		nmdm = comDev
	}
	slog.Debug("getCom", "nmdm", nmdm)
	comArg = append(comArg, "-l", "com"+strconv.Itoa(num)+","+nmdm)
	return comArg, nmdm
}

func (vm *VM) generateCommandLine() (name string, args []string, err error) {
	name = config.Config.Sys.Sudo
	slot := 0
	var com1Arg []string
	var com2Arg []string
	var com3Arg []string
	var com4Arg []string
	var cdArg []string
	com1Dev := ""
	com2Dev := ""
	com3Dev := ""
	com4Dev := ""
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
	debugArg := vm.getDebugArg()
	hostBridgeArg, slot := vm.getHostBridgeArg(slot)
	fbufArg, slot := vm.getVideoArg(slot)
	tabletArg, slot := vm.getTabletArg(slot)
	netArg, slot := vm.getNetArg(slot)
	diskArg, slot := vm.getDiskArg(slot)
	cdArg, slot = vm.getCDArg(slot)
	soundArg, slot := vm.getSoundArg(slot)
	if vm.Config.Com1 {
		com1Arg, com1Dev = getCom(vm.Config.Com1Dev, vm.Name, 1)
	}
	if vm.Config.Com2 {
		com2Arg, com2Dev = getCom(vm.Config.Com2Dev, vm.Name, 2)
	}
	if vm.Config.Com3 {
		com3Arg, com3Dev = getCom(vm.Config.Com3Dev, vm.Name, 3)
	}
	if vm.Config.Com4 {
		com4Arg, com4Dev = getCom(vm.Config.Com4Dev, vm.Name, 4)
	}

	vm.Com1Dev = com1Dev
	vm.Com2Dev = com2Dev
	vm.Com3Dev = com3Dev
	vm.Com4Dev = com4Dev
	_ = vm.Save()

	lpcArg, slot := vm.getLPCArg(slot)

	kbdArg := vm.getKeyboardArg()

	extraArgs := vm.getExtraArg()

	if vm.Config.Protect.Valid && vm.Config.Protect.Bool {
		args = append(args, "/usr/bin/protect")
	}
	if vm.Config.Priority != 0 {
		args = append(args, "/usr/bin/nice", "-n", strconv.FormatInt(int64(vm.Config.Priority), 10))
	}
	args = append(args, "/usr/sbin/bhyve")
	args = append(args, kbdArg...)
	args = append(args, acpiArg...)
	args = append(args, haltArg...)
	args = append(args, eopArg...)
	args = append(args, wireArg...)
	args = append(args, dpoArg...)
	args = append(args, msrArg...)
	args = append(args, utcArg...)
	args = append(args, romArg...)
	args = append(args, debugArg...)
	args = append(args, cpuArg...)
	args = append(args, memArg...)
	args = append(args, hostBridgeArg...)
	args = append(args, cdArg...)
	args = append(args, fbufArg...)
	args = append(args, tabletArg...)
	args = append(args, netArg...)
	args = append(args, diskArg...)
	args = append(args, soundArg...)
	args = append(args, lpcArg...)
	if len(com1Arg) != 0 {
		slog.Debug("com1Arg", "com1Arg", com1Arg)
		args = append(args, com1Arg...)
	}
	if len(com2Arg) != 0 {
		slog.Debug("com2Arg", "com2Arg", com2Arg)
		args = append(args, com2Arg...)
	}
	if len(com3Arg) != 0 {
		slog.Debug("com3Arg", "com3Arg", com3Arg)
		args = append(args, com3Arg...)
	}
	if len(com4Arg) != 0 {
		slog.Debug("com4Arg", "com4Arg", com4Arg)
		args = append(args, com4Arg...)
	}
	args = append(args, extraArgs...)
	args = append(args, "-U", vm.ID)
	args = append(args, vm.Name)
	return name, args, nil
}
