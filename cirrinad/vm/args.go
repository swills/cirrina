package vm

import (
	"fmt"
	"log/slog"
	"math"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rxwycdh/rxhash"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vmnic"
)

var getHostMaxVMCpusFunc = util.GetHostMaxVMCpus
var NetInterfacesFunc = net.Interfaces
var GetFreeTCPPortFunc = util.GetFreeTCPPort

type MacHashData struct {
	VMID    string
	VMName  string
	NicID   string
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

	// max slot is 31, be sure to leave room for the lpc, which is currently always added
	maxSataDevs := 31 - slot - 1
	devCount := 0

	for _, isoItem := range vm.ISOs {
		if isoItem == nil {
			continue
		}

		if isoItem.Path == "" {
			slog.Error("empty iso path, correcting", "iso", isoItem.Name, "id", isoItem.ID, "path", isoItem.Path)
			isoItem.Path = config.Config.Disk.VM.Path.Iso + string(os.PathSeparator) + isoItem.Name
		}

		slog.Debug("getCDArg", "name", isoItem.Name, "id", isoItem.ID, "path", isoItem.Path)

		if devCount <= maxSataDevs {
			thisCd := []string{"-s", strconv.FormatInt(int64(slot), 10) + ":0,ahci,cd:" + isoItem.Path}
			cdString = append(cdString, thisCd...)
			devCount++
			slot++
		} else {
			slog.Error("unable to add iso due to lack of slots", "slot", slot, "isoName", isoItem.Name)
		}
	}

	return cdString, slot
}

func (vm *VM) getCPUArg() []string {
	var vmCpus uint16

	hostCpus, err := getHostMaxVMCpusFunc()
	if err != nil {
		return []string{}
	}

	if vm.Config.CPU > math.MaxUint16 || !util.NumCpusValid(uint16(vm.Config.CPU)) {
		vmCpus = hostCpus
	} else {
		vmCpus = uint16(vm.Config.CPU)
	}

	return []string{"-c", strconv.FormatInt(int64(vmCpus), 10)}
}

func (vm *VM) getOneDiskArg(thisDisk *disk.Disk) (string, error) {
	var err error

	var diskController string

	nocache := ""
	direct := ""

	diskPath := thisDisk.GetPath()
	diskExists, err := thisDisk.VerifyExists()

	if err != nil {
		slog.Error("error checking disk path exists", "diskId", thisDisk.ID, "diskName", thisDisk.Name, "diskPath", diskPath)

		return "", fmt.Errorf("error checking disk path exists: %w", err)
	}

	if !diskExists {
		slog.Error("disk path does not exist", "diskId", thisDisk.ID, "diskName", thisDisk.Name, "diskPath", diskPath)

		return "", fmt.Errorf("disk path does not exist: %w", err)
	}

	switch thisDisk.Type {
	case "NVME":
		diskController = "nvme"
	case "AHCI-HD":
		diskController = "ahci-hd"
	case "VIRTIO-BLK":
		diskController = "virtio-blk"
	default:
		slog.Error("unknown disk type", "type", thisDisk.Type)

		return "", errVMUnknownDiskType
	}

	if thisDisk.DevType == "ZVOL" {
		diskPath = filepath.Join("/dev/zvol/", diskPath)
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
	// FIXME -- this is awful but needed until we attach multiple sata disks to each controller
	maxSataDevs := 31 - slot - 1
	sataDevCount := 0

	var diskString []string
	// TODO remove all these de-normalizations in favor of gorm native "Has Many" relationships

	for _, diskItem := range vm.Disks {
		if diskItem == nil {
			continue
		}

		if diskItem.ID == "" {
			continue
		}

		thisDisk, err := disk.GetByID(diskItem.ID)
		if err != nil {
			slog.Error("error getting disk, skipping", "diskID", diskItem.ID, "err", err)

			continue
		}

		if thisDisk.Type == "AHCI-HD" {
			sataDevCount++
		}

		if sataDevCount > maxSataDevs {
			slog.Error("sata dev count exceeded, skipping disk", "diskID", diskItem.ID)

			continue
		}

		oneHdString, err := vm.getOneDiskArg(thisDisk)
		if err != nil || oneHdString == "" {
			slog.Error("error adding disk, skipping", "diskID", diskItem.ID, "err", err)

			continue
		}

		thisHd := []string{"-s", strconv.FormatInt(int64(slot), 10) + "," + oneHdString}
		diskString = append(diskString, thisHd...)
		slot++
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

	hostBridgeArg := []string{"-s", strconv.FormatInt(int64(slot), 10) + ",hostbridge"}
	slot++

	return hostBridgeArg, slot
}

func (vm *VM) getMemArg() []string {
	return []string{"-m", strconv.FormatInt(int64(vm.Config.Mem), 10) + "m"}
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
		uefiVarsPath := filepath.Join(config.Config.Disk.VM.Path.State, filepath.Join(vm.Name, "BHYVE_UEFI_VARS.fd"))
		romArg = []string{
			"-l",
			"bootrom," + config.Config.Rom.Path + "," + uefiVarsPath,
		}
	} else {
		romArg = []string{
			"-l",
			"bootrom," + config.Config.Rom.Path,
		}
	}

	return romArg
}

func (vm *VM) getDebugArg() []string {
	var debugArg []string

	firstDebugPort := config.Config.Debug.Port
	debugListenIP := config.Config.Debug.IP

	var debugListenPortInt int

	var debugListenPort string

	var debugWaitStr string

	var err error

	if !vm.Config.Debug {
		return []string{}
	}

	if vm.Config.DebugPort == "AUTO" {
		usedDebugPorts := getUsedDebugPorts()

		debugListenPortInt, err = GetFreeTCPPortFunc(int(firstDebugPort), usedDebugPorts)
		if err != nil {
			slog.Error("error getting free tcp port", "err", err)

			return []string{}
		}

		debugListenPort = strconv.FormatInt(int64(debugListenPortInt), 10)
	} else {
		var debugListenPortInt64 int64

		debugListenPort = vm.Config.DebugPort

		debugListenPortInt64, err = strconv.ParseInt(debugListenPort, 10, 64)
		if err != nil {
			slog.Error("error parsing debug listen port", "err", err)

			return []string{}
		}

		debugListenPortInt = int(debugListenPortInt64)
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

	inPathExists, inErr := PathExistsFunc(vm.Config.SoundIn)
	if inErr != nil {
		slog.Error("sound input check error", "err", inErr)
	}

	outPathExists, outErr := PathExistsFunc(vm.Config.SoundOut)
	if outErr != nil {
		slog.Error("sound output check error", "err", outErr)
	}

	if !inPathExists && !outPathExists {
		return soundArg, slot
	}

	if inErr != nil && outErr != nil {
		return soundArg, slot
	}

	soundString = ",hda"

	if outPathExists && outErr == nil {
		soundString = soundString + ",play=" + vm.Config.SoundOut
	} else {
		slog.Debug("sound output path does not exist", "path", vm.Config.SoundOut)
	}

	if inPathExists && inErr == nil {
		soundString = soundString + ",rec=" + vm.Config.SoundIn
	} else {
		slog.Debug("sound input path does not exist", "path", vm.Config.SoundIn)
	}

	return []string{"-s", strconv.FormatInt(int64(slot), 10) + soundString}, slot + 1
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

	tabletArg := []string{"-s", strconv.FormatInt(int64(slot), 10) + ",xhci,tablet"}
	slot++

	return tabletArg, slot
}

func (vm *VM) getVideoArg(slot int) ([]string, int) {
	if !vm.Config.Screen {
		return []string{}, slot
	}

	firstVncPort := config.Config.Vnc.Port
	vncListenIP := config.Config.Vnc.IP

	var vncListenPortInt int

	var vncListenPort string

	var err error

	if vm.Config.VNCPort == "AUTO" {
		usedVncPorts := getUsedVncPorts()

		vncListenPortInt, err = GetFreeTCPPortFunc(int(firstVncPort), usedVncPorts)
		if err != nil {
			return []string{}, slot
		}

		vncListenPort = strconv.FormatInt(int64(vncListenPortInt), 10)
	} else {
		var vncListenPortInt64 int64

		vncListenPort = vm.Config.VNCPort

		vncListenPortInt64, err = strconv.ParseInt(vncListenPort, 10, 64)
		if err != nil {
			return []string{}, slot
		}

		vncListenPortInt = int(vncListenPortInt64)
	}

	vm.SetVNCPort(vncListenPortInt)

	fbufArg := []string{
		"-s",
		strconv.FormatInt(int64(slot), 10) +
			",fbuf" +
			",w=" + strconv.FormatInt(int64(vm.Config.ScreenWidth), 10) +
			",h=" + strconv.FormatInt(int64(vm.Config.ScreenHeight), 10) +
			",tcp=" + vncListenIP + ":" + vncListenPort,
	}
	if vm.Config.VNCWait {
		fbufArg[1] += ",wait"
	}

	slot++

	return fbufArg, slot
}

func getNetTypeArg(netType string) (string, error) {
	switch netType {
	case "VIRTIONET":
		return "virtio-net", nil
	case "E1000":
		return "e1000", nil
	default:
		slog.Debug("unknown net type, cannot configure", "netType", netType)

		return "", errVMUnknownNetType
	}
}

func getNetDevTypeArg(netDevType string, switchID string, vmName string) (string, string, error) {
	var err error

	var netDev string

	var netDevArg string

	switch netDevType {
	case "TAP":
		netDev, netDevArg = getTapDev()

		return netDev, netDevArg, nil
	case "VMNET":
		netDev, netDevArg = getVmnetDev()

		return netDev, netDevArg, nil
	case "NETGRAPH":
		netDev, netDevArg, err = _switch.GetNgDev(switchID, vmName)
		if err != nil {
			slog.Error("GetNgDev error", "err", err)

			return "", "", fmt.Errorf("error getting net dev arg: %w", err)
		}

		return netDev, netDevArg, nil
	default:
		slog.Debug("unknown net dev type", "netDevType", netDevType)

		return "", "", errVMUnknownNetDevType
	}
}

func (vm *VM) getNetArgs(slot int) ([]string, int) {
	var err error

	var netArgs []string

	originalSlot := slot

	nicList, err := vmnic.GetNics(vm.Config.ID)
	if err != nil {
		slog.Error("error getting vm nics", "err", err)

		return []string{}, originalSlot
	}

	for _, nicItem := range nicList {
		slog.Debug("adding nic", "nic", nicItem)

		var netType string

		netType, err = getNetTypeArg(nicItem.NetType)
		if err != nil {
			slog.Error("unknown net type, cannot configure", "netType", nicItem.NetType)

			return []string{}, originalSlot
		}

		var netDevArg string

		nicItem.NetDev, netDevArg, err = getNetDevTypeArg(nicItem.NetDevType, nicItem.SwitchID, vm.Name)
		if err != nil {
			slog.Error("getNetDevTypeArg error", "err", err)

			return []string{}, slot
		}

		err = nicItem.Save()
		if err != nil {
			slog.Error("failed to save net dev", "nic", nicItem.ID, "netdev", nicItem.NetDev)

			return []string{}, slot
		}

		macAddress := getMac(nicItem, vm)

		var macString string

		if macAddress != "" {
			macString = ",mac=" + macAddress
		}

		netArg := []string{"-s", strconv.FormatInt(int64(slot), 10) + "," + netType + "," + netDevArg + macString}
		slot++

		netArgs = append(netArgs, netArg...)
	}

	return netArgs, slot
}

func getMac(thisNic vmnic.VMNic, thisVM *VM) string {
	var macAddress string

	if thisNic.Mac == "AUTO" {
		// if MAC is AUTO, we still generate our own here rather than letting bhyve generate it, because:
		// 1. Bhyve is still using the NetApp MAC:
		// https://cgit.freebsd.org/src/tree/usr.sbin/bhyve/net_utils.c?id=1d386b48a555f61cb7325543adbbb5c3f3407a66#n115
		// 2. We want to be able to distinguish our VMs from other VMs
		slog.Debug("getNetArgs: Generating MAC")

		thisNicHashData := MacHashData{
			VMID:    thisVM.ID,
			VMName:  thisVM.Name,
			NicID:   thisNic.ID,
			NicName: thisNic.Name,
		}

		nicHash, err := rxhash.HashStruct(thisNicHashData)
		if err != nil {
			slog.Error("getNetArgs error generating mac", "err", err)

			return ""
		}

		slog.Debug("getNetArgs", "nicHash", nicHash)
		mac := string(nicHash[0]) + string(nicHash[1]) + ":" +
			string(nicHash[2]) + string(nicHash[3]) + ":" +
			string(nicHash[4]) + string(nicHash[5])
		slog.Debug("getNetArgs", "mac", mac)
		macAddress = config.Config.Network.Mac.Oui + ":" + mac
	} else {
		macAddress = thisNic.Mac
	}

	return macAddress
}

// getTapDev returns the netDev (stored in DB) and netDevArg (passed to bhyve) -- both happen to be the same here
func getTapDev() (string, string) {
	freeTapDevFound := false

	var netDevs []string

	tapDev := ""
	tapNum := 0

	interfaces, _ := NetInterfacesFunc()
	for _, inter := range interfaces {
		netDevs = append(netDevs, inter.Name)
	}

	for !freeTapDevFound {
		tapDev = "tap" + strconv.FormatInt(int64(tapNum), 10)
		if !util.ContainsStr(netDevs, tapDev) && !isNetPortUsed(tapDev) {
			freeTapDevFound = true
		} else {
			tapNum++
		}
	}

	return tapDev, tapDev
}

// getVmnetDev returns the netDev (stored in DB) and netDevArg (passed to bhyve) -- both happen to be the same here
func getVmnetDev() (string, string) {
	freeVmnetDevFound := false

	var netDevs []string

	vmnetDev := ""
	vmnetNum := 0

	interfaces, _ := NetInterfacesFunc()
	for _, inter := range interfaces {
		netDevs = append(netDevs, inter.Name)
	}

	for !freeVmnetDevFound {
		vmnetDev = "vmnet" + strconv.FormatInt(int64(vmnetNum), 10)
		if !util.ContainsStr(netDevs, vmnetDev) && !isNetPortUsed(vmnetDev) {
			freeVmnetDevFound = true
		} else {
			vmnetNum++
		}
	}

	return vmnetDev, vmnetDev
}

func getCom(comDev string, vmName string, num int) ([]string, string) {
	var nmdm string

	var comArg []string

	if comDev == "AUTO" {
		nmdm = "/dev/nmdm-" + vmName + "-com" + strconv.FormatInt(int64(num), 10) + "-A"
	} else {
		nmdm = comDev
	}

	slog.Debug("getCom", "nmdm", nmdm)
	comArg = append(comArg, "-l", "com"+strconv.FormatInt(int64(num), 10)+","+nmdm)

	return comArg, nmdm
}

func (vm *VM) generateCommandLine() (string, []string) {
	var args []string

	// we always start with sudo, at least until bhyve can run as non-root
	// and no, 'doas' does not work for our needs
	name := config.Config.Sys.Sudo

	slot := 0
	hostBridgeArg, fbufArg, tabletArg, netArg, diskArg, cdArg, soundArg, lpcArg := getSlotArgs(slot, vm)

	com1Arg, com2Arg, com3Arg, com4Arg := getComArgs(vm)

	args = addProtectArgs(vm, args)
	args = addPriorityArgs(vm, args)
	args = append(args, "/usr/sbin/bhyve")
	args = append(args, "-U", vm.ID)
	args = addSomeArgs(args, vm)
	args = addSlotArgs(args, hostBridgeArg, cdArg, fbufArg, tabletArg, netArg, diskArg, soundArg, lpcArg)
	args = addComArgs(args, com1Arg, com2Arg, com3Arg, com4Arg)
	args = append(args, vm.getExtraArg()...)
	args = append(args, vm.Name)

	_ = vm.Save()

	return name, args
}

func addPriorityArgs(vm *VM, args []string) []string {
	if vm.Config.Priority != 0 {
		args = append(args, "/usr/bin/nice", "-n", strconv.FormatInt(int64(vm.Config.Priority), 10))
	}

	return args
}

func addProtectArgs(vm *VM, args []string) []string {
	if vm.Config.Protect.Valid && vm.Config.Protect.Bool {
		args = append(args, "/usr/bin/protect")
	}

	return args
}

func addComArgs(args []string, com1Arg []string, com2Arg []string, com3Arg []string, com4Arg []string) []string {
	if len(com1Arg) != 0 {
		args = append(args, com1Arg...)
	}

	if len(com2Arg) != 0 {
		args = append(args, com2Arg...)
	}

	if len(com3Arg) != 0 {
		args = append(args, com3Arg...)
	}

	if len(com4Arg) != 0 {
		args = append(args, com4Arg...)
	}

	return args
}

func addSlotArgs(args []string, hostBridgeArg []string, cdArg []string, fbufArg []string, tabletArg []string,
	netArg []string, diskArg []string, soundArg []string, lpcArg []string) []string {
	args = append(args, hostBridgeArg...)
	args = append(args, cdArg...)
	args = append(args, fbufArg...)
	args = append(args, tabletArg...)
	args = append(args, netArg...)
	args = append(args, diskArg...)
	args = append(args, soundArg...)
	args = append(args, lpcArg...)

	return args
}

func addSomeArgs(args []string, aVM *VM) []string {
	args = append(args, aVM.getKeyboardArg()...)
	args = append(args, aVM.getACPIArg()...)
	args = append(args, aVM.getHLTArg()...)
	args = append(args, aVM.getEOPArg()...)
	args = append(args, aVM.getWireArg()...)
	args = append(args, aVM.getDPOArg()...)
	args = append(args, aVM.getMSRArg()...)
	args = append(args, aVM.getUTCArg()...)
	args = append(args, aVM.getROMArg()...)
	args = append(args, aVM.getDebugArg()...)
	args = append(args, aVM.getCPUArg()...)
	args = append(args, aVM.getMemArg()...)

	return args
}

func getSlotArgs(slot int, aVM *VM) ([]string, []string, []string, []string, []string, []string, []string, []string) {
	hostBridgeArg, slot := aVM.getHostBridgeArg(slot)
	fbufArg, slot := aVM.getVideoArg(slot)
	tabletArg, slot := aVM.getTabletArg(slot)
	netArg, slot := aVM.getNetArgs(slot)
	diskArg, slot := aVM.getDiskArg(slot)
	cdArg, slot := aVM.getCDArg(slot)
	soundArg, slot := aVM.getSoundArg(slot)
	lpcArg, slot := aVM.getLPCArg(slot)
	slog.Debug("last slot", "slot", slot)

	return hostBridgeArg, fbufArg, tabletArg, netArg, diskArg, cdArg, soundArg, lpcArg
}

func getComArgs(aVM *VM) ([]string, []string, []string, []string) {
	var com1Arg []string

	var com2Arg []string

	var com3Arg []string

	var com4Arg []string

	if aVM.Config.Com1 {
		com1Arg, aVM.Com1Dev = getCom(aVM.Config.Com1Dev, aVM.Name, 1)
	}

	if aVM.Config.Com2 {
		com2Arg, aVM.Com2Dev = getCom(aVM.Config.Com2Dev, aVM.Name, 2)
	}

	if aVM.Config.Com3 {
		com3Arg, aVM.Com3Dev = getCom(aVM.Config.Com3Dev, aVM.Name, 3)
	}

	if aVM.Config.Com4 {
		com4Arg, aVM.Com4Dev = getCom(aVM.Config.Com4Dev, aVM.Name, 4)
	}

	return com1Arg, com2Arg, com3Arg, com4Arg
}
