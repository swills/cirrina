package _switch

import (
	"bufio"
	"bytes"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
	"errors"
	"fmt"
	exec "golang.org/x/sys/execabs"
	"log/slog"
	"strconv"
	"strings"
)

type NgNode struct {
	NodeName  string
	NodeType  string
	NodeId    string
	NodeHooks int
}

type ngPeer struct {
	LocalHook string
	PeerName  string
	PeerType  string
	PeerId    string
	PeerHook  string
}

func ngGetNodes() (ngNodes []NgNode, err error) {
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
		ngNodes = append(ngNodes, NgNode{
			NodeName:  aNodeName,
			NodeType:  aNodeType,
			NodeId:    aNodeId,
			NodeHooks: aNodeHooks,
		})
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}

	return ngNodes, nil
}

func getAllNgBridges() (bridges []string, err error) {
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

func ngGetBridgePeers(bridge string) (peers []ngPeer, err error) {
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
		aPeer := ngPeer{
			LocalHook: textFields[0],
			PeerName:  textFields[1],
			PeerType:  textFields[2],
			PeerId:    textFields[3],
			PeerHook:  textFields[4],
		}
		peers = append(peers, aPeer)
	}

	return peers, nil
}

func ngBridgeNextLink(peers []ngPeer) (link string) {
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
			linkNum += 1
		} else {
			found = true
		}
	}
	return linkName
}

func createNgBridge(name string) (err error) {
	if name == "" {
		return errors.New("name can't be empty")
	}

	if !strings.HasPrefix(name, "bnet") {
		slog.Error("invalid bridge name", "name", name)
		return errors.New("invalid bridge name, bridge name must start with \"bnet\"")
	}

	allIfBridges, err := getAllNgBridges()
	if err != nil {
		slog.Debug("failed to get all if bridges", "err", err)
		return err
	}
	if util.ContainsStr(allIfBridges, name) {
		slog.Debug("bridge already exists", "bridge", name)
		return errors.New("duplicate bridge")
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
		return errors.New("failed to create ng bridge: could not get dummy bridge name")
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
		return err
	}
	cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "name",
		dummyIfBridgeName+":lower", netDev)
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl name err", "err", err)
		return err
	}
	//useUplink := true
	//var upper string
	upper := "uplink"
	//if useUplink {
	//	upper = "uplink"
	//} else {
	//	upper = "link"
	//}
	cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "connect",
		dummyIfBridgeName+":", netDev+":", "upper", upper+"1")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl connect error", "err", err)
		return err
	}
	//cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "msg",
	//	dummyIfBridgeName+":", "setpromisc", "1")
	//err = cmd.Run()
	//if err != nil {
	//	slog.Error("ngctl msg setpromisc error", "err", err)
	//	return err
	//}
	//cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "msg",
	//	dummyIfBridgeName+":", "setautosrc", "0")
	//err = cmd.Run()
	//if err != nil {
	//	slog.Error("ngctl msg setautosrc error", "err", err)
	//	return err
	//}
	cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "msg",
		netDev+":", "setpersistent")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl msg setpersistent error", "err", err)
		return err
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
		err = BridgeNgAddMember(bridgeName, member)
		if err != nil {
			slog.Error("createNgBridgeWithMembers error adding bridge member",
				"name", bridgeName,
				"member", member,
				"err", err,
			)
			return err
		}
	}
	return nil
}

func bridgeNgDeleteAllPeers(name string) error {
	bridgePeers, err := ngGetBridgePeers(name)
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
		return err
	}
	if err := cmd.Wait(); err != nil {
		slog.Error("failed running ngctl", "err", err, "out", out)
		return err
	}
	return nil
}

func bridgeNgRemoveUplink(bridgeName string, peerName string) error {
	var thisPeer ngPeer
	bridgePeers, err := ngGetBridgePeers(bridgeName)
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
	//if thisPeer.PeerName == "" {
	//	return errors.New("uplink not found")
	//}

	return nil
}
