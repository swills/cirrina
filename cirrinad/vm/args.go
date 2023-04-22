package vm

import (
	"bufio"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/iso"
	"cirrina/cirrinad/util"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/exp/slog"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

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
	return []string{"-c", strconv.Itoa(int(vm.Config.Cpu))}
}

func (vm *VM) getDiskArg(slot int) ([]string, int) {
	originalSlot := slot
	// TODO deal with multiple controller types
	// TODO don't use one PCI slot per disk device, attach multiple disks to each controller

	//diskType := "nvme"
	//diskPath := config.Config.Disk.VM.Path.Image + "/" + vm.Name + ".img"
	//diskArg := []string{"-s", strconv.Itoa(slot) + "," + diskType + "," + diskPath}
	//slot = slot + 1
	//return diskArg, slot
	//

	var diskString []string
	maxSataDevs := 32 - slot - 1
	devCount := 0

	diskList := strings.Split(vm.Config.Disks, ",")
	for _, diskItem := range diskList {
		if diskItem == "" {
			continue
		}
		slog.Debug("adding disk", "disk", diskItem)
		thisDisk, err := disk.GetById(diskItem)
		if err != nil {
			slog.Error("error getting Disk", "diskItem", diskItem, "err", err)
			return []string{}, originalSlot
		}
		thisDiskExists, err := util.PathExists(thisDisk.Path)
		if err != nil {
			slog.Error("error getting disk path", "path", thisDisk.Path, "err", err)
			return []string{}, originalSlot
		}
		if !thisDiskExists {
			slog.Debug("disk path does not exist", "path", thisDisk.Path, "err", err)
			return []string{}, originalSlot
		}
		if thisDisk.Type == "NVME" {
			thisHd := []string{"-s", strconv.Itoa(slot) + ",nvme," + thisDisk.Path}
			diskString = append(diskString, thisHd...)
			devCount = devCount + 1
			slot = slot + 1
		} else if thisDisk.Type == "AHCI-HD" {
			if devCount <= maxSataDevs {
				thisHd := []string{"-s", strconv.Itoa(slot) + ",ahci-hd," + thisDisk.Path}
				diskString = append(diskString, thisHd...)
				devCount = devCount + 1
				slot = slot + 1
			}
		} else if thisDisk.Type == "VIRTIO-BLK" {
			thisHd := []string{"-s", strconv.Itoa(slot) + ",virtio-blk," + thisDisk.Path}
			diskString = append(diskString, thisHd...)
			devCount = devCount + 1
			slot = slot + 1
		} else {
			slog.Error("unknown disk type", "type", thisDisk.Type)
			return []string{}, originalSlot
		}
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

func getFreePort(firstVncPort int) (port int, err error) {
	cmd := exec.Command("netstat", "-an", "--libxo", "json")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	var result map[string]interface{}
	if err := json.NewDecoder(stdout).Decode(&result); err != nil {
		return 0, err
	}
	if err := cmd.Wait(); err != nil {
		return 0, err
	}
	statistics, valid := result["statistics"].(map[string]interface{})
	if !valid {
		return 0, nil
	}
	sockets, valid := statistics["socket"].([]interface{})
	if !valid {
		return 0, errors.New("failed parsing netstat output - 1")
	}
	localListenPorts := make(map[int]struct{})
	for _, value := range sockets {
		socket, valid := value.(map[string]interface{})
		if !valid {
			continue
		}
		if socket["protocol"] == "tcp4" || socket["protocol"] == "tcp46" || socket["protocol"] == "tcp6" {
			state, valid := socket["tcp-state"].(string)
			if !valid {
				continue
			}
			realState := strings.TrimSpace(state)
			if realState == "LISTEN" {
				local, valid := socket["local"].(map[string]interface{})
				if !valid {
					continue
				}
				port, valid := local["port"].(interface{})
				if !valid {
					continue
				}
				p, valid := port.(string)
				if !valid {
					continue
				}
				portInt, err := strconv.Atoi(p)
				if err != nil {
					return 0, err
				}
				if _, exists := localListenPorts[portInt]; !exists {
					localListenPorts[portInt] = struct{}{}
				}
			}
		}
	}
	var uniqueLocalListenPorts []int
	for l := range localListenPorts {
		uniqueLocalListenPorts = append(uniqueLocalListenPorts, l)
	}
	sort.Slice(uniqueLocalListenPorts, func(i, j int) bool {
		return uniqueLocalListenPorts[i] < uniqueLocalListenPorts[j]
	})

	vncPort := firstVncPort
	for ; vncPort <= 65535; vncPort++ {
		if !containsInt(uniqueLocalListenPorts, vncPort) && !IsVncPortUsed(int32(vncPort)) {
			break
		}
	}
	return vncPort, nil
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
		vncListenPortInt, err = getFreePort(int(firstVncPort))
		if err != nil {
			return []string{}, slot
		}
		vncListenPort = strconv.Itoa(vncListenPortInt)
		vm.setVNCPort(vncListenPortInt)
	} else {
		vncListenPort = vm.Config.VNCPort
		vncListenPortInt, err = strconv.Atoi(vncListenPort)
		if err != nil {
			return []string{}, slot
		}
		vm.setVNCPort(vncListenPortInt)
	}

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

func containsStr(elems []string, v string) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

func containsInt(elems []int, v int) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

type ngNode struct {
	nodeName  string
	nodeType  string
	nodeId    string
	nodeHooks int
}

func ngGetNodes() (ngNodes []ngNode, err error) {
	cmd := exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "list")
	defer func(cmd *exec.Cmd) {
		err := cmd.Wait()
		if err != nil {
			slog.Error("ngctl error", "err", err)
		}
	}(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		text := scanner.Text()
		textFields := strings.Fields(text)
		if len(textFields) != 9 {
			continue
		}
		if !strings.HasPrefix(textFields[0], "Name:") {
			continue
		}
		aNodeName := textFields[1]
		if !strings.HasPrefix(textFields[2], "Type:") {
			continue
		}
		aNodeType := textFields[3]
		if !strings.HasPrefix(textFields[4], "ID:") {
			continue
		}
		aNodeId := textFields[5]
		if !strings.HasPrefix(textFields[7], "hooks:") {
			continue
		}
		aNodeHooks, _ := strconv.Atoi(textFields[8])
		ngNodes = append(ngNodes, ngNode{
			nodeName:  aNodeName,
			nodeType:  aNodeType,
			nodeId:    aNodeId,
			nodeHooks: aNodeHooks,
		})
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}

	return ngNodes, nil
}

func ngGetBridges() (bridges []string, err error) {
	netgraphNodes, err := ngGetNodes()
	if err != nil {
		return nil, err
	}
	// loop and check for type = bridge, add to list and return list
	for _, node := range netgraphNodes {
		if node.nodeType == "bridge" {
			bridges = append(bridges, node.nodeName)
		}
	}
	return bridges, nil
}

func ngAllocateBridge(bridgeNames []string) (bridgeName string) {
	bnetNum := 0
	bridgeFound := false
	for !bridgeFound {
		bridgeName = "bnet" + strconv.Itoa(bnetNum)
		if containsStr(bridgeNames, bridgeName) {
			bnetNum += 1
		} else {
			bridgeFound = true
		}
	}
	return bridgeName
}

type ngBridge struct {
	localHook string
	peerName  string
	peerType  string
	peerId    string
	peerHook  string
}

func ngShowBridge(bridge string) (peers []ngBridge, err error) {
	cmd := exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "show",
		bridge+":")
	defer func(cmd *exec.Cmd) {
		err := cmd.Wait()
		if err != nil {
			slog.Error("ngctl show error", "err", err)
		}
	}(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(stdout)
	lineNo := 0
	for scanner.Scan() {
		text := scanner.Text()
		lineNo += 1
		if lineNo < 4 {
			continue
		}
		textFields := strings.Fields(text)
		if len(textFields) != 5 {
			continue
		}
		aBridge := ngBridge{
			localHook: textFields[0],
			peerName:  textFields[1],
			peerType:  textFields[2],
			peerId:    textFields[3],
			peerHook:  textFields[4],
		}
		peers = append(peers, aBridge)
	}

	return peers, nil
}

func ngBridgeUplink(peers []ngBridge) (link string) {
	var upperLink string
	var lowerLink string

	for _, peer := range peers {
		if peer.peerHook == "upper" {
			upperLink = peer.peerName
		}
		if peer.peerHook == "lower" {
			lowerLink = peer.peerName
		}
	}
	if upperLink != "" && upperLink == lowerLink {
		return upperLink
	}
	return ""
}

func ngBridgeNextPeer(peers []ngBridge) (link string) {
	found := false
	linkNum := 0
	linkName := ""
	var hooks []string

	for _, peer := range peers {
		hooks = append(hooks, peer.localHook)
	}

	for !found {
		linkName = "link" + strconv.Itoa(linkNum)
		if containsStr(hooks, linkName) {
			linkNum += 1
		} else {
			found = true
		}
	}
	return linkName
}

func ngGetDev(link string) (bridge string, peer string, err error) {
	defaultPeerLink := "link2"
	var bridgeNet string
	var nextLink string

	bridgeList, err := ngGetBridges()
	if err != nil {
		return bridge, peer, err
	}

	if link != "" {
		for _, bridge := range bridgeList {
			bridgePeers, err := ngShowBridge(bridge)
			if err != nil {
				return "", "", err
			}
			peerLink := ngBridgeUplink(bridgePeers)
			if peerLink == link {
				bridgeNet = bridge
				nextLink = ngBridgeNextPeer(bridgePeers)
			}
		}
		if bridgeNet == "" {
			bridgeNet = ngAllocateBridge(bridgeList)
			// TODO - ?
			nextLink = defaultPeerLink
		}
	} else {
		// TODO - pick peer automatically?
		return bridge, peer, err
	}
	return bridgeNet, nextLink, nil

}

func (vm *VM) getNetArg(slot int) ([]string, int) {
	if !vm.Config.Net {
		return []string{}, slot
	}
	var netType string
	if vm.Config.NetType == "VIRTIONET" {
		netType = "virtio-net"
	} else if vm.Config.NetType == "E1000" {
		netType = "e1000"
	} else {
		slog.Debug("unknown net type, cannot configure", "netType", vm.Config.NetType)
		return []string{}, slot
	}
	var netDev string
	var netDevArg string
	if vm.Config.NetDevType == "TAP" {
		netDev = getTapDev()
		netDevArg = netDev
	} else if vm.Config.NetDevType == "VMNET" {
		netDev = getVmnetDev()
		netDevArg = netDev
	} else if vm.Config.NetDevType == "NETGRAPH" {
		ngNetDev, ngPeerHook, err := ngGetDev(config.Config.Network.Interface)
		if err != nil {
			slog.Error("ngGetDev error", "err", err)
			return []string{}, slot
		}
		netDev = ngNetDev
		netDevArg = "netgraph,path=" + ngNetDev + ":,peerhook=" + ngPeerHook + ",socket=" + vm.Name
	} else {
		slog.Debug("unknown net dev type", "netDevType", vm.Config.NetDevType)
		return []string{}, slot
	}
	macAddress := vm.Config.Mac
	macString := ""
	if macAddress != "AUTO" {
		macString = ",mac=" + macAddress
	}
	netArg := []string{"-s", strconv.Itoa(slot) + "," + netType + "," + netDevArg + macString}
	slot = slot + 1
	vm.NetDev = netDev
	_ = vm.Save()
	return netArg, slot
}

func getTapDev() string {
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
		if !containsStr(tapDevs, tapDev) && !IsNetPortUsed(tapDev) {
			freeTapDevFound = true
		} else {
			tapNum = tapNum + 1
		}
	}
	return tapDev
}

func getVmnetDev() string {
	freeVmnetDevFound := false
	var vmnetDevs []string
	vmnetDev := ""
	vmnetNum := 0
	interfaces, _ := net.Interfaces()
	for _, inter := range interfaces {
		if strings.Contains(inter.Name, "vmnet") {
			vmnetDevs = append(vmnetDevs, inter.Name)
		}
	}
	for !freeVmnetDevFound {
		vmnetDev = "vmnet" + strconv.Itoa(vmnetNum)
		if !containsStr(vmnetDevs, vmnetDev) && !IsNetPortUsed(vmnetDev) {
			freeVmnetDevFound = true
		} else {
			vmnetNum = vmnetNum + 1
		}
	}
	return vmnetDev
}

func getNmdmNum(offset int) (nmdm string, err error) {
	// dear god this is so ugly please kill it
	var nmdmDevs []string
	var nmdmDev string
	devList, err := os.ReadDir("/dev/")
	if err != nil {
		return "", err
	}
	for _, dev := range devList {
		devName := dev.Name()
		if strings.HasPrefix(devName, "nmdm") {
			a := strings.TrimLeft(devName, "nmdm")
			b := strings.TrimRight(a, "A")
			c := strings.TrimRight(b, "B")
			if !containsStr(nmdmDevs, c) {
				nmdmDevs = append(nmdmDevs, c)
			}
		}
	}
	sort.Strings(nmdmDevs)
	slog.Debug("getNmdmNum", "nmdmDevs", nmdmDevs)
	if len(nmdmDevs) == 0 {
		nmdmDev = "/dev/nmdm" + strconv.Itoa(offset) + "A"
	} else {
		d := nmdmDevs[len(nmdmDevs)-1]
		e, err := strconv.Atoi(d)
		if err != nil {
			return "", err
		}
		f := e + 1 + offset
		nmdmDev = "/dev/nmdm" + strconv.Itoa(f) + "A"
	}
	return nmdmDev, nil
}

func getCom(comDev string, nmdmOffset int, num int) (int, []string, string) {
	var err error
	nmdm := ""
	var comArg []string
	if comDev == "AUTO" {
		nmdm, err = getNmdmNum(nmdmOffset)
		if err != nil {
			return nmdmOffset + 1, comArg, ""
		}
		nmdmOffset = nmdmOffset + 1
	} else {
		nmdm = comDev
	}
	comArg = append(comArg, "-l", "com"+strconv.Itoa(num)+","+nmdm)
	return nmdmOffset, comArg, nmdm
}

func (vm *VM) generateCommandLine() (name string, args []string, err error) {
	name = config.Config.Sys.Sudo
	slot := 0
	nmdmOffset := 0
	var com1Arg []string
	var com2Arg []string
	var com3Arg []string
	var com4Arg []string
	var cdArg []string
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
	cdArg, slot = vm.getCDArg(slot)
	soundArg, slot := vm.getSoundArg(slot)
	if vm.Config.Com1 {
		nmdmOffset, com1Arg, _ = getCom(vm.Config.Com1Dev, nmdmOffset, 1)
	}
	if vm.Config.Com2 {
		nmdmOffset, com2Arg, _ = getCom(vm.Config.Com1Dev, nmdmOffset, 2)
	}
	if vm.Config.Com3 {
		nmdmOffset, com3Arg, _ = getCom(vm.Config.Com1Dev, nmdmOffset, 3)
	}
	if vm.Config.Com4 {
		nmdmOffset, com4Arg, _ = getCom(vm.Config.Com1Dev, nmdmOffset, 4)
	}
	lpcArg, slot := vm.getLPCArg(slot)

	kbdArg := vm.getKeyboardArg()

	extraArgs := vm.getExtraArg()

	args = append(args, "/usr/bin/protect")
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
	args = append(args, vm.Name)
	return name, args, nil
}
