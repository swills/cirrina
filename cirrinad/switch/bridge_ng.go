package _switch

import (
	"bufio"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
	"errors"
	"fmt"
	"golang.org/x/exp/slog"
	"os/exec"
	"strconv"
	"strings"
)

type NgNode struct {
	NodeName  string
	NodeType  string
	NodeId    string
	NodeHooks int
}

type NgBridge struct {
	LocalHook string
	PeerName  string
	PeerType  string
	PeerId    string
	PeerHook  string
}

func NgGetNodes() (ngNodes []NgNode, err error) {
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

func NgGetBridges() (bridges []string, err error) {
	netgraphNodes, err := NgGetNodes()
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

func NgAllocateBridge(bridgeNames []string) (bridgeName string) {
	bnetNum := 0
	bridgeFound := false
	for !bridgeFound {
		bridgeName = "bnet" + strconv.Itoa(bnetNum)
		if util.ContainsStr(bridgeNames, bridgeName) {
			bnetNum += 1
		} else {
			bridgeFound = true
		}
	}
	return bridgeName
}

func NgShowBridge(bridge string) (peers []NgBridge, err error) {
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
		aBridge := NgBridge{
			LocalHook: textFields[0],
			PeerName:  textFields[1],
			PeerType:  textFields[2],
			PeerId:    textFields[3],
			PeerHook:  textFields[4],
		}
		peers = append(peers, aBridge)
	}

	return peers, nil
}

func NgBridgeUplink(peers []NgBridge) (link string) {
	var upperLink string
	var lowerLink string

	for _, peer := range peers {
		if peer.PeerHook == "upper" {
			upperLink = peer.PeerName
		}
		if peer.PeerHook == "lower" {
			lowerLink = peer.PeerName
		}
	}
	if upperLink != "" && upperLink == lowerLink {
		return upperLink
	}
	return ""
}

func NgBridgeNextPeer(peers []NgBridge) (link string) {
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

func NgGetDev(link string) (bridge string, peer string, err error) {
	defaultPeerLink := "link2"
	var bridgeNet string
	var nextLink string

	bridgeList, err := NgGetBridges()
	if err != nil {
		return bridge, peer, err
	}

	if link != "" {
		for _, bridge := range bridgeList {
			bridgePeers, err := NgShowBridge(bridge)
			if err != nil {
				return "", "", err
			}
			peerLink := NgBridgeUplink(bridgePeers)
			if peerLink == link {
				bridgeNet = bridge
				nextLink = NgBridgeNextPeer(bridgePeers)
			}
		}
		if bridgeNet == "" {
			bridgeNet = NgAllocateBridge(bridgeList)
			// TODO - ?
			nextLink = defaultPeerLink
		}
	} else {
		// TODO - pick peer automatically?
		return bridge, peer, err
	}
	return bridgeNet, nextLink, nil

}

func NgCreateBridge(netDev string, bridgePeer string) (err error) {
	if netDev == "" {
		return errors.New("netDev can't be empty")
	}
	if bridgePeer == "" {
		return errors.New("bridgePeer can't be empty")
	}
	cmd := exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "mkpeer",
		bridgePeer+":", "bridge", "lower", "link0")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl mkpeer error", "err", err)
		return err
	}
	cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "name",
		bridgePeer+":lower", netDev)
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl name err", "err", err)
		return err
	}
	useUplink := true
	var upper string
	if useUplink {
		upper = "uplink"
	} else {
		upper = "link"
	}
	cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "connect",
		bridgePeer+":", netDev+":", "upper", upper+"1")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl connect error", "err", err)
		return err
	}
	cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "msg",
		bridgePeer+":", "setpromisc", "1")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl msg error", "err", err)
		return err
	}
	cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "msg",
		bridgePeer+":", "setautosrc", "0")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl msg error", "err", err)
		return err
	}
	return nil
}

func NgDestroyBridge(netDev string) (err error) {
	if netDev == "" {
		return errors.New("netDev can't be empty")
	}
	cmd := exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "msg",
		netDev+":", "shutdown")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl msg error", "err", err)
		return err
	}
	return nil
}
