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

	"github.com/spf13/cast"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	_switch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vmnic"
)

var getHostMaxVMCpusFunc = util.GetHostMaxVMCpus
var NetInterfacesFunc = net.Interfaces
var GetFreeTCPPortFunc = util.GetFreeTCPPort

func (v *VM) getKeyboardArg() []string {
	if v.Config.Screen && v.Config.KbdLayout != "default" {
		return []string{"-K", v.Config.KbdLayout}
	}

	return []string{}
}

func (v *VM) getACPIArg() []string {
	if v.Config.ACPI {
		return []string{"-A"}
	}

	return []string{}
}

func (v *VM) getCDArg(slot int) ([]string, int) {
	var cdString []string

	// max slot is 31, be sure to leave room for the lpc, which is currently always added
	maxSataDevs := 31 - slot - 1
	devCount := 0

	for _, isoItem := range v.ISOs {
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

func (v *VM) getCPUArg() []string {
	var vmCpus uint16

	hostCpus, err := getHostMaxVMCpusFunc()
	if err != nil {
		return []string{}
	}

	if v.Config.CPU > math.MaxUint16 || !util.NumCpusValid(cast.ToUint16(v.Config.CPU)) {
		vmCpus = hostCpus
	} else {
		vmCpus = cast.ToUint16(v.Config.CPU)
	}

	return []string{"-c", strconv.FormatInt(int64(vmCpus), 10)}
}

func (v *VM) getOneDiskArg(thisDisk *disk.Disk) (string, error) {
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

func (v *VM) getDiskArg(slot int) ([]string, int) {
	// TODO don't use one PCI slot per ahci (SATA) disk device, attach multiple disks to each controller
	// FIXME -- this is awful but needed until we attach multiple sata disks to each controller
	maxSataDevs := 31 - slot - 1
	sataDevCount := 0

	var diskString []string
	// TODO remove all these de-normalizations in favor of gorm native "Has Many" relationships

	for _, diskItem := range v.Disks {
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

		oneHdString, err := v.getOneDiskArg(thisDisk)
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

func (v *VM) getDPOArg() []string {
	if v.Config.DestroyPowerOff {
		return []string{"-D"}
	}

	return []string{}
}

func (v *VM) getEOPArg() []string {
	if v.Config.ExitOnPause {
		return []string{"-P"}
	}

	return []string{}
}

func (v *VM) getExtraArg() []string {
	return strings.Fields(v.Config.ExtraArgs)
}

func (v *VM) getHLTArg() []string {
	if v.Config.UseHLT {
		return []string{"-H"}
	}

	return []string{}
}

func (v *VM) getHostBridgeArg(slot int) ([]string, int) {
	if !v.Config.HostBridge {
		return []string{}, slot
	}

	hostBridgeArg := []string{"-s", strconv.FormatInt(int64(slot), 10) + ",hostbridge"}
	slot++

	return hostBridgeArg, slot
}

func (v *VM) getMemArg() []string {
	return []string{"-m", strconv.FormatInt(int64(v.Config.Mem), 10) + "m"}
}

func (v *VM) getMSRArg() []string {
	if v.Config.IgnoreUnknownMSR {
		return []string{"-w"}
	}

	return []string{}
}

func (v *VM) getROMArg() []string {
	var romArg []string

	if v.Config.StoreUEFIVars {
		uefiVarsPath := filepath.Join(config.Config.Disk.VM.Path.State, filepath.Join(v.Name, "BHYVE_UEFI_VARS.fd"))
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

func (v *VM) getDebugArg() []string {
	var debugArg []string

	firstDebugPort := config.Config.Debug.Port
	debugListenIP := config.Config.Debug.IP

	var debugListenPortUint16 uint16

	var debugListenPort string

	var debugWaitStr string

	var err error

	if !v.Config.Debug {
		return []string{}
	}

	if v.Config.DebugPort == "AUTO" {
		usedDebugPorts := getUsedDebugPorts()

		debugListenPortUint16, err = GetFreeTCPPortFunc(firstDebugPort, usedDebugPorts)
		if err != nil {
			slog.Error("error getting free tcp port", "err", err)

			return []string{}
		}

		debugListenPort = strconv.FormatInt(int64(debugListenPortUint16), 10)
	} else {
		var debugListenPortUint64 uint64

		debugListenPort = v.Config.DebugPort

		debugListenPortUint64, err = strconv.ParseUint(debugListenPort, 10, 16)
		if err != nil {
			slog.Error("error parsing debug listen port", "err", err)

			return []string{}
		}

		debugListenPortUint16 = cast.ToUint16(debugListenPortUint64)
	}

	v.SetDebugPort(debugListenPortUint16)

	if v.Config.DebugWait {
		debugWaitStr = "w"
	}

	debugArg = []string{
		"-G",
		debugWaitStr + debugListenIP + ":" + debugListenPort,
	}

	return debugArg
}

func (v *VM) getSoundArg(slot int) ([]string, int) {
	if !v.Config.Sound {
		return []string{}, slot
	}

	var soundArg []string

	var soundString string

	inPathExists, inErr := PathExistsFunc(v.Config.SoundIn)
	if inErr != nil {
		slog.Error("sound input check error", "err", inErr)
	}

	outPathExists, outErr := PathExistsFunc(v.Config.SoundOut)
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
		soundString = soundString + ",play=" + v.Config.SoundOut
	} else {
		slog.Debug("sound output path does not exist", "path", v.Config.SoundOut)
	}

	if inPathExists && inErr == nil {
		soundString = soundString + ",rec=" + v.Config.SoundIn
	} else {
		slog.Debug("sound input path does not exist", "path", v.Config.SoundIn)
	}

	return []string{"-s", strconv.FormatInt(int64(slot), 10) + soundString}, slot + 1
}

func (v *VM) getUTCArg() []string {
	if v.Config.UTCTime {
		return []string{"-u"}
	}

	return []string{}
}

func (v *VM) getWireArg() []string {
	if v.Config.WireGuestMem {
		return []string{"-S"}
	}

	return []string{}
}

func (v *VM) getLPCArg(slot int) ([]string, int) {
	return []string{"-s", "31,lpc"}, slot
}

func (v *VM) getTabletArg(slot int) ([]string, int) {
	if !v.Config.Screen || !v.Config.Tablet {
		return []string{}, slot
	}

	tabletArg := []string{"-s", strconv.FormatInt(int64(slot), 10) + ",xhci,tablet"}
	slot++

	return tabletArg, slot
}

func (v *VM) getVideoArg(slot int) ([]string, int) {
	if !v.Config.Screen {
		return []string{}, slot
	}

	firstVncPort := config.Config.Vnc.Port
	vncListenIP := config.Config.Vnc.IP

	var vncListenPortUint16 uint16

	var vncListenPort string

	var err error

	if v.Config.VNCPort == "AUTO" {
		usedVncPorts := getUsedVncPorts()

		vncListenPortUint16, err = GetFreeTCPPortFunc(firstVncPort, usedVncPorts)
		if err != nil {
			return []string{}, slot
		}

		vncListenPort = strconv.FormatInt(int64(vncListenPortUint16), 10)
	} else {
		var vncListenPortInt64 int64

		vncListenPort = v.Config.VNCPort

		vncListenPortInt64, err = strconv.ParseInt(vncListenPort, 10, 16)
		if err != nil {
			return []string{}, slot
		}

		vncListenPortUint16 = cast.ToUint16(vncListenPortInt64)
	}

	v.SetVNCPort(vncListenPortUint16)

	fbufArg := []string{
		"-s",
		strconv.FormatInt(int64(slot), 10) +
			",fbuf" +
			",w=" + strconv.FormatInt(int64(v.Config.ScreenWidth), 10) +
			",h=" + strconv.FormatInt(int64(v.Config.ScreenHeight), 10) +
			",tcp=" + vncListenIP + ":" + vncListenPort,
	}
	if v.Config.VNCWait {
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

func (v *VM) getNetArgs(slot int) ([]string, int) {
	var err error

	var netArgs []string

	originalSlot := slot

	nicList, err := vmnic.GetNics(v.Config.ID)
	if err != nil {
		slog.Error("error getting v nics", "err", err)

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

		nicItem.NetDev, netDevArg, err = getNetDevTypeArg(nicItem.NetDevType, nicItem.SwitchID, v.Name)
		if err != nil {
			slog.Error("getNetDevTypeArg error", "err", err)

			return []string{}, slot
		}

		err = nicItem.Save()
		if err != nil {
			slog.Error("failed to save net dev", "nic", nicItem.ID, "netdev", nicItem.NetDev)

			return []string{}, slot
		}

		macAddress := nicItem.GetMAC(v.ID, v.Name)

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

// getTapDev returns the netDev (stored in DB) and netDevArg (passed to bhyve) -- both happen to be the same here
func getTapDev() (string, string) {
	freeTapDevFound := false

	tapDev := ""
	tapNum := 0

	interfaces, _ := NetInterfacesFunc()

	netDevs := make([]string, 0, len(interfaces))

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

	vmnetDev := ""
	vmnetNum := 0

	interfaces, _ := NetInterfacesFunc()

	netDevs := make([]string, 0, len(interfaces))

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

func (v *VM) generateCommandLine() (string, []string) {
	var args []string

	// we always start with sudo, at least until bhyve can run as non-root
	// and no, 'doas' does not work for our needs
	name := config.Config.Sys.Sudo

	slot := 0
	hostBridgeArg, fbufArg, tabletArg, netArg, diskArg, cdArg, soundArg, lpcArg := v.getSlotArgs(slot)

	com1Arg, com2Arg, com3Arg, com4Arg := v.getComArgs()

	args = v.addProtectArgs(args)
	args = v.addPriorityArgs(args)
	args = append(args, "/usr/sbin/bhyve")
	args = append(args, "-U", v.ID)
	args = v.addSomeArgs(args)
	args = addSlotArgs(args, hostBridgeArg, cdArg, fbufArg, tabletArg, netArg, diskArg, soundArg, lpcArg)
	args = addComArgs(args, com1Arg, com2Arg, com3Arg, com4Arg)
	args = append(args, v.getExtraArg()...)
	args = append(args, v.Name)

	_ = v.Save()

	return name, args
}

func (v *VM) addPriorityArgs(args []string) []string {
	if v.Config.Priority != 0 {
		args = append(args, "/usr/bin/nice", "-n", strconv.FormatInt(int64(v.Config.Priority), 10))
	}

	return args
}

func (v *VM) addProtectArgs(args []string) []string {
	if v.Config.Protect.Valid && v.Config.Protect.Bool {
		args = append(args, "/usr/bin/protect", "-i")
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

func (v *VM) addSomeArgs(args []string) []string {
	args = append(args, v.getKeyboardArg()...)
	args = append(args, v.getACPIArg()...)
	args = append(args, v.getHLTArg()...)
	args = append(args, v.getEOPArg()...)
	args = append(args, v.getWireArg()...)
	args = append(args, v.getDPOArg()...)
	args = append(args, v.getMSRArg()...)
	args = append(args, v.getUTCArg()...)
	args = append(args, v.getROMArg()...)
	args = append(args, v.getDebugArg()...)
	args = append(args, v.getCPUArg()...)
	args = append(args, v.getMemArg()...)

	return args
}

func (v *VM) getSlotArgs(slot int) ([]string, []string, []string, []string, []string, []string, []string, []string) {
	hostBridgeArg, slot := v.getHostBridgeArg(slot)
	fbufArg, slot := v.getVideoArg(slot)
	tabletArg, slot := v.getTabletArg(slot)
	netArg, slot := v.getNetArgs(slot)
	diskArg, slot := v.getDiskArg(slot)
	cdArg, slot := v.getCDArg(slot)
	soundArg, slot := v.getSoundArg(slot)
	lpcArg, slot := v.getLPCArg(slot)
	slog.Debug("last slot", "slot", slot)

	return hostBridgeArg, fbufArg, tabletArg, netArg, diskArg, cdArg, soundArg, lpcArg
}

func (v *VM) getComArgs() ([]string, []string, []string, []string) {
	var com1Arg []string

	var com2Arg []string

	var com3Arg []string

	var com4Arg []string

	if v.Config.Com1 {
		com1Arg, v.Com1Dev = getCom(v.Config.Com1Dev, v.Name, 1)
	}

	if v.Config.Com2 {
		com2Arg, v.Com2Dev = getCom(v.Config.Com2Dev, v.Name, 2)
	}

	if v.Config.Com3 {
		com3Arg, v.Com3Dev = getCom(v.Config.Com3Dev, v.Name, 3)
	}

	if v.Config.Com4 {
		com4Arg, v.Com4Dev = getCom(v.Config.Com4Dev, v.Name, 4)
	}

	return com1Arg, com2Arg, com3Arg, com4Arg
}
