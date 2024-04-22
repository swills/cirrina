package vmswitch

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	exec "golang.org/x/sys/execabs"

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
	var ngNodes []NgNode
	var err error
	cmd := exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "list")
	defer func(cmd *exec.Cmd) {
		err = cmd.Wait()
		if err != nil {
			slog.Error("ngctl error", "err", err)
		}
	}(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed running ngctl: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed running ngctl: %w", err)
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
		aNodeID := textFields[5]
		if !strings.HasPrefix(textFields[7], "hooks:") {
			continue
		}
		aNodeHooks, _ := strconv.Atoi(textFields[8])
		ngNodes = append(ngNodes, NgNode{
			NodeName:  aNodeName,
			NodeType:  aNodeType,
			NodeID:    aNodeID,
			NodeHooks: aNodeHooks,
		})
	}

	if err := scanner.Err(); err != nil {
		slog.Error("error scanning ngctl output", "err", err)

		return []NgNode{}, fmt.Errorf("error parsing ngctl output: %w", err)
	}

	return ngNodes, nil
}

func GetAllNgBridges() ([]string, error) {
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

func getNgBridgeMembers(bridge string) ([]ngPeer, error) {
	var err error
	var peers []ngPeer
	cmd := exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "show",
		bridge+":")
	defer func(cmd *exec.Cmd) {
		err = cmd.Wait()
		if err != nil {
			slog.Error("ngctl show error", "err", err)
		}
	}(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("error running ngctl command: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("error running ngctl command: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	lineNo := 0
	for scanner.Scan() {
		text := scanner.Text()
		lineNo++
		if lineNo < 4 {
			continue
		}
		textFields := strings.Fields(text)
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
	var hooks []string

	for _, peer := range peers {
		hooks = append(hooks, peer.LocalHook)
	}

	for !found {
		linkName = "link" + strconv.Itoa(linkNum)
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
		return errSwitchInvalidName
	}

	if !strings.HasPrefix(name, "bnet") {
		slog.Error("invalid bridge name", "name", name)

		return errSwitchInvalidBridgeNameNG
	}

	allIfBridges, err := GetAllNgBridges()
	if err != nil {
		slog.Debug("failed to get all if bridges", "err", err)

		return err
	}
	if util.ContainsStr(allIfBridges, name) {
		slog.Debug("bridge already exists", "bridge", name)

		return errSwitchInvalidBridgeDupe
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
	dummyIfBridgeName := GetDummyBridgeName()
	if dummyIfBridgeName == "" {
		return errSwitchFailDummy
	}
	err := createIfBridge(dummyIfBridgeName)
	if err != nil {
		slog.Error("dummy if_bridge creation error", "err", err)

		return err
	}

	cmd := exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "mkpeer",
		dummyIfBridgeName+":", "bridge", "lower", "link0")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl mkpeer error", "err", err)

		return fmt.Errorf("error running ngctl command: %w", err)
	}
	cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "name",
		dummyIfBridgeName+":lower", netDev)
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl name err", "err", err)

		return fmt.Errorf("error running ngctl command: %w", err)
	}
	upper := "uplink"
	cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "connect",
		dummyIfBridgeName+":", netDev+":", "upper", upper+"1")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl connect error", "err", err)

		return fmt.Errorf("failed running ngctl command: %w", err)
	}
	cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "msg",
		netDev+":", "setpersistent")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl msg setpersistent error", "err", err)

		return fmt.Errorf("failed running ngctl command: %w", err)
	}

	// and delete our dummy if_bridge
	err = DestroyIfBridge(dummyIfBridgeName, false)
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
		exists := CheckInterfaceExists(member)
		if !exists {
			slog.Error("attempt to add non-existent member to bridge, ignoring",
				"bridge", bridgeName, "uplink", member,
			)

			continue
		}
		err = BridgeNgAddMember(bridgeName, member)
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
	var out bytes.Buffer
	cmd := exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "rmhook", bridgeName+":", hook)
	cmd.Stdout = &out
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error running ngctl: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		slog.Error("failed running ngctl", "err", err, "out", out)

		return fmt.Errorf("error running ngctl: %w", err)
	}

	return nil
}

func bridgeNgRemoveUplink(bridgeName string, peerName string) error {
	var thisPeer ngPeer
	bridgePeers, err := getNgBridgeMembers(bridgeName)
	if err != nil {
		return err
	}

	for _, peer := range bridgePeers {
		slog.Debug("bridgeNgRemoveUplink", "peer", peer)
		if peer.PeerName == peerName {
			thisPeer = peer
			err = bridgeNgDeletePeer(bridgeName, thisPeer.LocalHook)
			if err != nil {
				return err
			}
		}
	}
	// if thisPeer.PeerName == "" {
	// 	return errors.New("uplink not found")
	// }

	return nil
}
