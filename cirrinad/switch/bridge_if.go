package vmswitch

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

func GetAllIfBridges() ([]string, error) {
	var err error

	var bridges []string

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd("/sbin/ifconfig", []string{"-g", "bridge"})
	if err != nil {
		slog.Error("ifconfig error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return nil, fmt.Errorf("ifconfig error: %w", err)
	}

	for _, line := range strings.Split(string(stdOutBytes), "\n") {
		if len(line) == 0 {
			continue
		}

		textFields := strings.Fields(line)
		if len(textFields) != 1 {
			continue
		}

		aBridgeName := textFields[0]
		bridges = append(bridges, aBridgeName)
	}

	return bridges, nil
}

func getIfBridgeMembers(name string) ([]string, error) {
	var members []string

	var err error

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd("/sbin/ifconfig", []string{name})
	if err != nil {
		slog.Error("ifconfig error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return nil, fmt.Errorf("ifconfig error: %w", err)
	}

	for _, line := range strings.Split(string(stdOutBytes), "\n") {
		if len(line) == 0 {
			continue
		}

		textFields := strings.Fields(line)
		if len(textFields) != 3 {
			continue
		}

		if textFields[0] != "member:" {
			continue
		}

		aBridgeMember := textFields[1]
		members = append(members, aBridgeMember)
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
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/ifconfig", name, "create", "group", "cirrinad", "up"},
	)
	if err != nil {
		slog.Error("ifconfig error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ifconfig error: %w", err)
	}

	return nil
}

func bridgeIfDeleteAllMembers(name string) error {
	bridgeMembers, err := getIfBridgeMembers(name)
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
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/ifconfig", bridgeName, "deletem", memberName},
	)
	if err != nil {
		slog.Error("ifconfig error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ifconfig error: %w", err)
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
		err = BridgeIfAddMember(bridgeName, member)
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
