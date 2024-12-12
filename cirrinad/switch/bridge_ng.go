package vmswitch

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

type NgNode struct {
	NodeName  string
	NodeType  string
	NodeID    string
	NodeHooks int
}

type ngPeer struct {
	LocalHook string
	PeerName  string
	PeerType  string
	PeerID    string
	PeerHook  string
}

func ngGetNodes() ([]NgNode, error) {
	var err error

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/ngctl", "list"},
	)
	if err != nil {
		slog.Error("ngctl error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return nil, fmt.Errorf("ngctl error: %w", err)
	}

	nodesStrs := strings.Split(string(stdOutBytes), "\n")

	ngNodes := make([]NgNode, 0, len(nodesStrs))

	for _, line := range nodesStrs {
		if len(line) == 0 {
			continue
		}

		textFields := strings.Fields(line)
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

		aNodeID := textFields[5]

		if !strings.HasPrefix(textFields[7], "hooks:") {
			continue
		}

		aNodeHooks, _ := strconv.ParseInt(textFields[8], 10, 64)
		ngNodes = append(ngNodes, NgNode{
			NodeName:  aNodeName,
			NodeType:  aNodeType,
			NodeID:    aNodeID,
			NodeHooks: int(aNodeHooks),
		})
	}

	return ngNodes, nil
}

func getNgBridgeMembers(bridge string) ([]ngPeer, error) {
	var err error

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/ngctl", "show", bridge + ":"},
	)
	if err != nil {
		slog.Error("ngctl error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return nil, fmt.Errorf("ngctl error: %w", err)
	}

	lineNo := 0

	bridgeMembers := strings.Split(string(stdOutBytes), "\n")
	peers := make([]ngPeer, 0, len(bridgeMembers))

	for _, line := range bridgeMembers {
		if len(line) == 0 {
			continue
		}

		lineNo++

		if lineNo < 4 {
			continue
		}

		textFields := strings.Fields(line)
		if len(textFields) != 5 {
			continue
		}

		aPeer := ngPeer{
			LocalHook: textFields[0],
			PeerName:  textFields[1],
			PeerType:  textFields[2],
			PeerID:    textFields[3],
			PeerHook:  textFields[4],
		}
		peers = append(peers, aPeer)
	}

	return peers, nil
}

func ngBridgeNextLink(peers []ngPeer) string {
	found := false
	linkNum := 0
	linkName := ""

	hooks := make([]string, 0, len(peers))

	for _, peer := range peers {
		hooks = append(hooks, peer.LocalHook)
	}

	for !found {
		linkName = "link" + strconv.FormatInt(int64(linkNum), 10)
		if util.ContainsStr(hooks, linkName) {
			linkNum++
		} else {
			found = true
		}
	}

	return linkName
}

func createNgBridge(name string) error {
	var err error

	if name == "" {
		return ErrSwitchInvalidName
	}

	if !strings.HasPrefix(name, "bnet") {
		slog.Error("invalid bridge name", "name", name)

		return ErrSwitchInvalidName
	}

	allIfBridges, err := getAllNgSwitches()
	if err != nil {
		slog.Debug("failed to get all if bridges", "err", err)

		return err
	}

	if util.ContainsStr(allIfBridges, name) {
		slog.Debug("bridge already exists", "bridge", name)

		return ErrSwitchExists
	}

	// actually create the ng bridge
	err = actualNgBridgeCreate(name)
	if err != nil {
		return err
	}

	return nil
}

func actualNgBridgeCreate(netDev string) error {
	// create a dummy if_bridge to connect the ng_bridge to
	dummyIfBridgeName := getDummyBridgeName()
	if dummyIfBridgeName == "" {
		return errSwitchFailDummy
	}

	err := createIfSwitch(dummyIfBridgeName)
	if err != nil {
		slog.Error("dummy if_bridge creation error", "err", err)

		return err
	}

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/ngctl", "mkpeer", dummyIfBridgeName + ":", "bridge", "lower", "link0"},
	)
	if err != nil {
		slog.Error("ngctl error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ngctl error: %w", err)
	}

	stdOutBytes, stdErrBytes, returnCode, err = util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/ngctl", "name", dummyIfBridgeName + ":lower", netDev},
	)
	if err != nil {
		slog.Error("ngctl error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ngctl error: %w", err)
	}

	upper := "uplink"

	stdOutBytes, stdErrBytes, returnCode, err = util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/ngctl", "connect", dummyIfBridgeName + ":", netDev + ":", "upper", upper + "1"},
	)
	if err != nil {
		slog.Error("ngctl error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ngctl error: %w", err)
	}

	stdOutBytes, stdErrBytes, returnCode, err = util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/ngctl", "msg", netDev + ":", "setpersistent"},
	)
	if err != nil {
		slog.Error("ngctl error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ngctl error: %w", err)
	}

	// and delete our dummy if_bridge
	err = destroyIfSwitch(dummyIfBridgeName, false)
	if err != nil {
		slog.Error("dummy if_bridge deletion error", "err", err)

		return err
	}

	return nil
}

func createNgBridgeWithMembers(bridgeName string, bridgeMembers []string) error {
	err := createNgBridge(bridgeName)
	if err != nil {
		slog.Error("createNgBridgeWithMembers error creating bridge",
			"name", bridgeName,
			"err", err,
		)

		return err
	}

	err = bridgeNgDeleteAllPeers(bridgeName)
	if err != nil {
		slog.Error("createNgBridgeWithMembers error deleting bridge peers",
			"name", bridgeName,
			"err", err,
		)

		return err
	}

	for _, member := range bridgeMembers {
		exists := util.CheckInterfaceExists(member)
		if !exists {
			slog.Error("attempt to add non-existent member to bridge, ignoring",
				"bridge", bridgeName, "uplink", member,
			)

			continue
		}

		err = switchNgAddMember(bridgeName, member)
		if err != nil {
			slog.Error("createNgBridgeWithMembers error adding bridge member",
				"name", bridgeName,
				"member", member,
				"err", err,
			)

			continue
		}
	}

	return nil
}

func bridgeNgDeleteAllPeers(name string) error {
	bridgePeers, err := getNgBridgeMembers(name)
	slog.Debug("deleting all ng bridge members", "bridge", name, "members", bridgePeers)

	if err != nil {
		return err
	}

	for _, member := range bridgePeers {
		err := bridgeNgDeletePeer(name, member.PeerName)
		if err != nil {
			return err
		}
	}

	return nil
}

func bridgeNgDeletePeer(bridgeName string, hook string) error {
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/ngctl", "rmhook", bridgeName + ":", hook},
	)
	if err != nil {
		slog.Error("ngctl error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ngctl error: %w", err)
	}

	return nil
}

func switchNgRemoveUplink(bridgeName string, peerName string) error {
	var thisPeer ngPeer

	bridgePeers, err := getNgBridgeMembers(bridgeName)
	if err != nil {
		return err
	}

	for _, peer := range bridgePeers {
		slog.Debug("switchNgRemoveUplink", "peer", peer)

		if peer.PeerName == peerName {
			thisPeer = peer

			err = bridgeNgDeletePeer(bridgeName, thisPeer.LocalHook)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getAllNgSwitches() ([]string, error) {
	var bridges []string

	netgraphNodes, err := ngGetNodes()
	if err != nil {
		return nil, err
	}
	// loop and check for type = bridge, add to list and return list
	for _, node := range netgraphNodes {
		if node.NodeType == "bridge" {
			bridges = append(bridges, node.NodeName)
		}
	}

	return bridges, nil
}

func memberUsedByNgSwitch(member string) (bool, error) {
	allBridges, err := getAllNgSwitches()
	if err != nil {
		slog.Error("error getting all if bridges", "err", err)

		return false, err
	}

	for _, aBridge := range allBridges {
		var allNgBridgeMembers []ngPeer

		var existingMembers []string

		// extra work here since this returns a ngPeer
		allNgBridgeMembers, err = getNgBridgeMembers(aBridge)
		if err != nil {
			slog.Error("error getting ng bridge members", "bridge", aBridge)

			return false, err
		}

		for _, m := range allNgBridgeMembers {
			existingMembers = append(existingMembers, m.PeerName)
		}

		if util.ContainsStr(existingMembers, member) {
			return true, nil
		}
	}

	return false, nil
}

func ngGetBridgeNextLink(bridge string) (string, error) {
	var nextLink string

	var err error

	bridgePeers, err := getNgBridgeMembers(bridge)
	if err != nil {
		return nextLink, err
	}

	nextLink = ngBridgeNextLink(bridgePeers)

	return nextLink, nil
}

func (s *Switch) setUplinkNG(uplink string) error {
	netDevs := util.GetHostInterfaces()

	if !util.ContainsStr(netDevs, uplink) {
		return errSwitchInvalidUplink
	}

	// it can't be a member of another bridge already
	alreadyUsed, err := memberUsedByNgSwitch(uplink)
	if err != nil {
		slog.Error("error checking if member already used", "err", err)

		return err
	}

	if alreadyUsed {
		slog.Error("another bridge already contains member, member can not be in two bridges of "+
			"same type, skipping adding", "member", uplink,
		)

		return errSwitchUplinkInUse
	}

	slog.Debug("setting NG bridge uplink", "id", s.ID)

	err = switchNgAddMember(s.Name, uplink)
	if err != nil {
		return err
	}

	s.Uplink = uplink

	err = s.Save()
	if err != nil {
		return err
	}

	return nil
}

func (s *Switch) validateNgSwitch() error {
	// it can't be a member of another bridge of same type already
	if s.Uplink != "" {
		alreadyUsed, err := memberUsedByNgSwitch(s.Uplink)
		if err != nil {
			slog.Error("error checking if member already used", "err", err)

			return fmt.Errorf("error checking if member already used: %w", err)
		}

		if alreadyUsed {
			return errSwitchUplinkInUse
		}
	}

	return nil
}

func switchNgAddMember(bridgeName string, memberName string) error {
	link, err := ngGetBridgeNextLink(bridgeName)
	if err != nil {
		return err
	}

	memberNameNg := strings.Replace(memberName, ".", "_", 1)

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/ngctl", "connect", memberNameNg + ":", bridgeName + ":", "lower", link},
	)
	if err != nil {
		slog.Error("ngctl connect error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ngctl connect error: %w", err)
	}

	link, err = ngGetBridgeNextLink(bridgeName)
	if err != nil {
		return err
	}

	stdOutBytes, stdErrBytes, returnCode, err = util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/ngctl", "connect", memberNameNg + ":", bridgeName + ":", "upper", link},
	)
	if err != nil {
		slog.Error("ngctl connect error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ngctl connect error: %w", err)
	}

	return nil
}

func destroyNgSwitch(netDev string) error {
	var err error

	if netDev == "" {
		return ErrSwitchInvalidName
	}

	if !strings.HasPrefix(netDev, "bnet") {
		slog.Error("invalid switch name", "name", netDev)

		return ErrSwitchInvalidName
	}

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/ngctl", "msg", netDev + ":", "shutdown"},
	)
	if err != nil {
		slog.Error("ngctl msg shutdown error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ngctl msg shutdown error: %w", err)
	}

	return nil
}

func (s *Switch) buildNgSwitch() error {
	// TODO remove all these de-normalizations in favor of gorm native "Has Many" relationships
	memberList := strings.Split(s.Uplink, ",")

	members := make([]string, 0, len(memberList))

	// sanity checking of bridge members
	for _, member := range memberList {
		// it can't be empty
		if member == "" {
			continue
		}
		// it has to exist
		exists := util.CheckInterfaceExists(member)
		if !exists {
			slog.Error("attempt to add non-existent member to bridge, ignoring",
				"bridge", s.Name, "uplink", member,
			)

			continue
		}
		// it can't be a member of another bridge already
		alreadyUsed, err := memberUsedByNgSwitch(member)
		if err != nil {
			slog.Error("error checking if member already used", "err", err)

			continue
		}

		if alreadyUsed {
			slog.Error("another bridge already contains member, member can not be in two bridges of "+
				"same type, skipping adding", "bridge", s.Name, "member", member,
			)

			continue
		}

		members = append(members, member)
	}

	err := createNgBridgeWithMembers(s.Name, members)

	return err
}

// GetNgDev returns the netDev (stored in DB) and netDevArg (passed to bhyve)
func GetNgDev(switchID string, name string) (string, string, error) {
	var err error

	thisSwitch, err := GetByID(switchID)
	if err != nil {
		slog.Error("switch lookup error", "switchid", switchID)

		return "", "", err
	}

	bridgePeers, err := getNgBridgeMembers(thisSwitch.Name)
	if err != nil {
		return "", "", err
	}

	nextLink := ngBridgeNextLink(bridgePeers)

	ngNetDev := thisSwitch.Name + "," + nextLink
	netDevArg := "netgraph,path=" + thisSwitch.Name + ":,peerhook=" + nextLink + ",socket=" + name

	return ngNetDev, netDevArg, nil
}
