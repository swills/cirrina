package vmswitch

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

func getIfBridgeMembers(name string) ([]string, error) {
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

	bridgeMembersStrings := strings.Split(string(stdOutBytes), "\n")

	members := make([]string, 0, len(bridgeMembersStrings))

	for _, line := range bridgeMembersStrings {
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
		return ErrSwitchInvalidName
	}

	// TODO allow other bridge names by creating with a dummy name and then renaming
	if !strings.HasPrefix(name, "bridge") {
		slog.Error("invalid bridge name", "name", name)

		return errSwitchInvalidBridgeNameIF
	}

	allIfBridges, err := GetAllIfSwitches()
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
		err := switchIfDeleteMember(name, member)
		if err != nil {
			return err
		}
	}

	return nil
}

func switchIfDeleteMember(bridgeName string, memberName string) error {
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

func GetAllIfSwitches() ([]string, error) {
	var err error

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

	bridgesStrs := strings.Split(string(stdOutBytes), "\n")

	bridges := make([]string, 0, len(bridgesStrs))

	for _, line := range bridgesStrs {
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

func CreateIfBridgeWithMembers(bridgeName string, bridgeMembers []string) error {
	if bridgeName == "" {
		return errSwitchInvalidBridgeNameIF
	}

	err := createIfBridge(bridgeName)
	if err != nil {
		return err
	}

	err = bridgeIfDeleteAllMembers(bridgeName)
	if err != nil {
		return err
	}

	for _, member := range bridgeMembers {
		err = SwitchIfAddMember(bridgeName, member)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetDummyBridgeName() string {
	// highest if_bridge num
	bridgeNum := 32767

	bridgeList, err := GetAllIfSwitches()
	if err != nil {
		return ""
	}

	for bridgeNum > 0 {
		bridgeName := "bridge" + strconv.FormatInt(int64(bridgeNum), 10)
		if util.ContainsStr(bridgeList, bridgeName) {
			bridgeNum--
		} else {
			return bridgeName
		}
	}

	return ""
}

func memberUsedByIfSwitch(member string) (bool, error) {
	allBridges, err := GetAllIfSwitches()
	if err != nil {
		slog.Error("error getting all if bridges", "err", err)

		return true, err
	}

	for _, aBridge := range allBridges {
		existingMembers, err := getIfBridgeMembers(aBridge)
		if err != nil {
			slog.Error("error getting if bridge members", "bridge", aBridge)

			return true, err
		}

		if util.ContainsStr(existingMembers, member) {
			return true, nil
		}
	}

	return false, nil
}

func (s *Switch) buildIfSwitch() error {
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
		exists := CheckInterfaceExists(member)
		if !exists {
			slog.Error("attempt to add non-existent member to bridge, ignoring",
				"bridge", s.Name, "uplink", member,
			)

			continue
		}
		// it can't be a member of another bridge already
		alreadyUsed, err := memberUsedByIfSwitch(member)
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

	err := CreateIfBridgeWithMembers(s.Name, members)

	return err
}

func (s *Switch) setUplinkIf(uplink string) error {
	alreadyUsed, err := memberUsedByIfSwitch(uplink)
	if err != nil {
		return err
	}

	if alreadyUsed {
		slog.Error("another bridge already contains member, member can not be in two bridges of "+
			"same type, skipping adding", "member", uplink,
		)

		return errSwitchUplinkInUse
	}

	slog.Debug("setting IF bridge uplink", "id", s.ID)

	err = SwitchIfAddMember(s.Name, uplink)
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

func (s *Switch) validateIfSwitch() error {
	// it can't be a member of another bridge of same type already
	if s.Uplink != "" {
		alreadyUsed, err := memberUsedByIfSwitch(s.Uplink)
		if err != nil {
			slog.Error("error checking if member already used", "err", err)

			return fmt.Errorf("error checking if switch uplink in use by another bridge: %w", err)
		}

		if alreadyUsed {
			return errSwitchUplinkInUse
		}
	}

	return nil
}

func SwitchIfAddMember(bridgeName string, memberName string) error {
	// TODO - check that the member name is a host interface or a VM nic interface
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/ifconfig", bridgeName, "addm", memberName},
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

func DestroyIfSwitch(name string, cleanup bool) error {
	// TODO allow other bridge names
	if !strings.HasPrefix(name, "bridge") {
		slog.Error("invalid bridge name", "name", name)

		return ErrSwitchInvalidName
	}

	if cleanup {
		err := bridgeIfDeleteAllMembers(name)
		if err != nil {
			return err
		}
	}

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/ifconfig", name, "destroy"},
	)
	if err != nil {
		slog.Error("ifconfig destroy error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ifconfig destroy error: %w", err)
	}

	return nil
}
