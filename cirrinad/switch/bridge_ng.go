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

func ngGetBridges() (bridges []string, err error) {
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

func ngBridgeUplink(peers []ngPeer) (link string) {
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

func ngBridgeNextPeer(peers []ngPeer) (link string) {
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

func ngCreateBridge(netDev string, bridgePeer string) (err error) {
	var dummy bool
	dummy_if_bridge_name := "bridge32767"
	if netDev == "" {
		return errors.New("netDev can't be empty")
	}
	if bridgePeer == "" {
		dummy = true
		// create a dummy if_bridge to connect the ng_bridge to

		// TODO this needs to check if it exists already and pick a name that doesn't exist
		//   for now, just go with the highest possible name, making this function not thread safe
		err = createIfBridge(dummy_if_bridge_name)
		if err != nil {
			slog.Error("dummy if_bridge creation error", "err", err)
			return err
		}
		bridgePeer = dummy_if_bridge_name
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
	if !dummy {
		cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "msg",
			bridgePeer+":", "setpromisc", "1")
		err = cmd.Run()
		if err != nil {
			slog.Error("ngctl msg setpromisc error", "err", err)
			return err
		}
		cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "msg",
			bridgePeer+":", "setautosrc", "0")
		err = cmd.Run()
		if err != nil {
			slog.Error("ngctl msg setautosrc error", "err", err)
			return err
		}
	}
	cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "msg",
		netDev+":", "setpersistent")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl msg setpersistent error", "err", err)
		return err
	}
	if dummy {
		err = deleteIfBridge(bridgePeer, false)
		if err != nil {
			slog.Error("dummy if_bridge deletion error", "err", err)
			return err
		}
	}

	return nil
}

func ngDestroyBridge(netDev string) (err error) {
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
