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

func GetAllIfBridges() (bridges []string, err error) {
	var r []string
	cmd := exec.Command("/sbin/ifconfig", "-g", "bridge")
	defer func(cmd *exec.Cmd) {
		err := cmd.Wait()
		if err != nil {
			slog.Error("ifconfig error", "err", err)
		}
	}(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed running ifconfig command: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed running ifconfig command: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		text := scanner.Text()
		textFields := strings.Fields(text)
		if len(textFields) != 1 {
			continue
		}
		aBridgeName := textFields[0]
		r = append(r, aBridgeName)
	}
	if err := scanner.Err(); err != nil {
		slog.Error("error scanning ifconfig output", "err", err)

		return []string{}, fmt.Errorf("failed parsing ifconfig output: %w", err)
	}

	return r, nil
}

func GetIfBridgeMembers(name string) (members []string, err error) {
	args := []string{name}
	cmd := exec.Command("/sbin/ifconfig", args...)
	defer func(cmd *exec.Cmd) {
		err := cmd.Wait()
		if err != nil {
			slog.Error("ifconfig error", "err", err)
		}
	}(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed running ifconfig command: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed running ifconfig command: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		text := scanner.Text()
		textFields := strings.Fields(text)
		if len(textFields) != 3 {
			continue
		}
		if textFields[0] != "member:" {
			continue
		}
		aBridgeMember := textFields[1]
		members = append(members, aBridgeMember)
	}
	if err := scanner.Err(); err != nil {
		slog.Error("error scanning ifconfig output", "err", err)

		return []string{}, fmt.Errorf("failed parsing ifconfig output: %w", err)
	}

	return members, nil
}

func createIfBridge(name string) error {
	if name == "" {
		return errSwitchInvalidName
	}

	// TODO allow other bridge names by creating with a dummy name and then renaming
	if !strings.HasPrefix(name, "bridge") {
		slog.Error("invalid bridge name", "name", name)

		return errSwitchInvalidBridgeNameIF
	}
	allIfBridges, err := GetAllIfBridges()
	if err != nil {
		slog.Debug("failed to get all if bridges", "err", err)

		return err
	}
	if util.ContainsStr(allIfBridges, name) {
		slog.Debug("bridge already exists", "bridge", name)

		return errSwitchInvalidBridgeDupe
	}

	err = actualIfBridgeCreate(name)
	if err != nil {
		return err
	}

	return nil
}

func actualIfBridgeCreate(name string) error {
	cmd := exec.Command(config.Config.Sys.Sudo, "/sbin/ifconfig", name, "create", "group", "cirrinad", "up")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed running ifconfig: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		slog.Error("failed running ifconfig", "err", err, "out", out)

		return fmt.Errorf("failed running ifconfig: %w", err)
	}

	return nil
}

func bridgeIfDeleteAllMembers(name string) error {
	bridgeMembers, err := GetIfBridgeMembers(name)
	slog.Debug("deleting all if bridge members", "bridge", name, "members", bridgeMembers)
	if err != nil {
		return err
	}
	for _, member := range bridgeMembers {
		err := bridgeIfDeleteMember(name, member)
		if err != nil {
			return err
		}
	}

	return nil
}

func bridgeIfDeleteMember(bridgeName string, memberName string) error {
	cmd := exec.Command(config.Config.Sys.Sudo, "/sbin/ifconfig", bridgeName, "deletem", memberName)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed running ifconfig: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		slog.Error("failed running ifconfig", "err", err, "out", out)

		return fmt.Errorf("failed running ifconfig: %w", err)
	}

	return nil
}

func CreateIfBridgeWithMembers(bridgeName string, bridgeMembers []string) error {
	err := createIfBridge(bridgeName)
	if err != nil {
		return err
	}
	err = bridgeIfDeleteAllMembers(bridgeName)
	if err != nil {
		return err
	}
	for _, member := range bridgeMembers {
		// we always learn on the uplink
		err = BridgeIfAddMember(bridgeName, member, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetDummyBridgeName() string {
	// highest if_bridge num
	bridgeNum := 32767

	bridgeList, err := GetAllIfBridges()
	if err != nil {
		return ""
	}

	for bridgeNum > 0 {
		bridgeName := "bridge" + strconv.Itoa(bridgeNum)
		if util.ContainsStr(bridgeList, bridgeName) {
			bridgeNum--
		} else {
			return bridgeName
		}
	}

	return ""
}
