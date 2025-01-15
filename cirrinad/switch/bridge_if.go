package vmswitch

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/epair"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vmnic"
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

func createIfSwitch(name string) error {
	if name == "" {
		return ErrSwitchInvalidName
	}

	// TODO allow other bridge names by creating with a dummy name and then renaming
	if !strings.HasPrefix(name, "bridge") {
		slog.Error("invalid bridge name", "name", name)

		return errSwitchInvalidBridgeNameIF
	}

	allIfSwitches, err := getAllIfSwitches()
	if err != nil {
		slog.Debug("failed to get all if bridges", "err", err)

		return err
	}

	if util.ContainsStr(allIfSwitches, name) {
		slog.Debug("bridge already exists", "bridge", name)

		return ErrSwitchExists
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

func switchIfDeleteAllMembers(name string) error {
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

func switchIfDeleteMember(name string, memberName string) error {
	hostInterfaces := util.GetAllHostInterfaces()

	if !util.ContainsStr(hostInterfaces, name) {
		return nil
	}

	bridgeMembers, err := getIfBridgeMembers(name)
	if err != nil {
		return fmt.Errorf("error deleting if switch member: %w", err)
	}

	if !util.ContainsStr(bridgeMembers, memberName) {
		return nil
	}

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/ifconfig", name, "deletem", memberName},
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

func getAllIfSwitches() ([]string, error) {
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

func createIfSwitchWithMembers(name string, bridgeMembers []string) error {
	if name == "" {
		return errSwitchInvalidBridgeNameIF
	}

	err := createIfSwitch(name)
	if err != nil {
		return err
	}

	for _, member := range bridgeMembers {
		err = switchIfAddMember(name, member)
		if err != nil {
			return err
		}
	}

	return nil
}

func getDummyBridgeName() string {
	// highest if_bridge num
	bridgeNum := 32767

	bridgeList, err := getAllIfSwitches()
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
	allBridges, err := getAllIfSwitches()
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
		exists := util.CheckInterfaceExists(member)
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

	err := createIfSwitchWithMembers(s.Name, members)

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

		return ErrSwitchUplinkInUse
	}

	slog.Debug("setting IF bridge uplink", "id", s.ID)

	err = switchIfAddMember(s.Name, uplink)
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
			return ErrSwitchUplinkInUse
		}
	}

	return nil
}

func switchIfAddMember(name string, memberName string) error {
	hostInterfaces := util.GetAllHostInterfaces()

	if !util.ContainsStr(hostInterfaces, name) {
		return ErrSwitchDoesNotExist
	}

	if !util.ContainsStr(hostInterfaces, memberName) {
		return ErrSwitchInterfaceDoesNotExist
	}

	bridgeMembers, err := getIfBridgeMembers(name)
	if err != nil {
		return fmt.Errorf("error getting if switch member: %w", err)
	}

	// already a member
	if util.ContainsStr(bridgeMembers, memberName) {
		return nil
	}

	// TODO - check that the member name is a host interface or a VM nic interface
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/ifconfig", name, "addm", memberName},
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

func destroyIfSwitch(name string, cleanup bool) error {
	// TODO allow other bridge names
	if !strings.HasPrefix(name, "bridge") {
		slog.Error("invalid switch name", "name", name)

		return ErrSwitchInvalidName
	}

	hostInterfaces := util.GetAllHostInterfaces()

	if !util.ContainsStr(hostInterfaces, name) {
		return nil
	}

	if cleanup {
		err := switchIfDeleteAllMembers(name)
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

func setupVMNicRateLimit(vmNic *vmnic.VMNic) (string, error) {
	var err error

	thisEpair := epair.GetDummyEpairName()
	slog.Debug("netStartup rate limiting", "thisEpair", thisEpair)

	err = epair.CreateEpair(thisEpair)
	if err != nil {
		slog.Error("error creating epair", "err", err)

		return "", fmt.Errorf("error creating epair: %w", err)
	}

	vmNic.InstEpair = thisEpair
	err = vmNic.Save()

	if err != nil {
		slog.Error("failed to save net dev", "nic", vmNic.ID, "netdev", vmNic.NetDev)

		return "", fmt.Errorf("error saving NIC: %w", err)
	}

	err = epair.SetRateLimit(thisEpair, vmNic.RateIn, vmNic.RateOut)
	if err != nil {
		slog.Error("failed to set epair rate limit", "epair", thisEpair)

		return "", fmt.Errorf("error setting rate limit: %w", err)
	}

	thisInstSwitch := getDummyBridgeName()

	var bridgeMembers []string
	bridgeMembers = append(bridgeMembers, thisEpair+"a")
	bridgeMembers = append(bridgeMembers, vmNic.NetDev)

	err = createIfSwitchWithMembers(thisInstSwitch, bridgeMembers)
	if err != nil {
		slog.Error("failed to create switch",
			"nic", vmNic.ID,
			"thisInstSwitch", thisInstSwitch,
			"err", err,
		)

		return "", fmt.Errorf("error creating bridge: %w", err)
	}

	vmNic.InstBridge = thisInstSwitch
	err = vmNic.Save()

	if err != nil {
		slog.Error("failed to save net dev", "nic", vmNic.ID, "netdev", vmNic.NetDev)

		return "", fmt.Errorf("error saving NIC: %w", err)
	}

	return thisEpair, nil
}

func unsetVMNicRateLimit(vmNic *vmnic.VMNic) error {
	var changed bool

	if vmNic.InstBridge == "" && vmNic.InstEpair == "" {
		return nil
	}

	if vmNic.InstBridge != "" {
		err := destroyIfSwitch(vmNic.InstBridge, false)
		if err != nil {
			slog.Error("failed to destroy switch", "err", err)
		}

		vmNic.InstBridge = ""
		changed = true
	}

	// tap/vmnet nics may be connected to an epair which is connected
	// to a netgraph pipe for purposes for rate limiting
	if vmNic.InstEpair != "" {
		err := epair.NgShutdownPipe(vmNic.InstEpair + "a")
		if err != nil {
			slog.Error("failed to destroy ng pipe", "err", err)
		}

		err = epair.NgShutdownPipe(vmNic.InstEpair + "b")
		if err != nil {
			slog.Error("failed to destroy ng pipe", "err", err)
		}

		err = epair.DestroyEpair(vmNic.InstEpair)
		if err != nil {
			slog.Error("failed to destroy epair", "err", err)
		}

		vmNic.InstEpair = ""
		changed = true
	}

	if changed {
		err := vmNic.Save()
		if err != nil {
			return fmt.Errorf("error unsetting nic rate limit: %w", err)
		}
	}

	return nil
}

func (s *Switch) connectIfNic(vmNic *vmnic.VMNic) error {
	var err error

	var thisMemberName string

	if vmNic.RateLimit {
		var thisEpair string

		thisEpair, err = setupVMNicRateLimit(vmNic)
		if err != nil {
			return fmt.Errorf("failed setting up nic: %w", err)
		}

		thisMemberName = thisEpair + "b"
	} else {
		thisMemberName = vmNic.NetDev
	}

	err = switchIfAddMember(s.Name, thisMemberName)
	if err != nil {
		slog.Error("failed to add nic to switch",
			"nicname", vmNic.Name,
			"nicid", vmNic.ID,
			"switchid", vmNic.SwitchID,
			"netdev", vmNic.NetDev,
			"err", err,
		)

		return fmt.Errorf("error adding member to switch: %w", err)
	}

	return nil
}

func (s *Switch) disconnectIfNic(vmNic *vmnic.VMNic) error {
	var err error

	var thisMemberName string

	if vmNic.RateLimit {
		// nothing to do
		if vmNic.InstEpair == "" {
			return nil
		}

		thisMemberName = vmNic.InstEpair + "b"
	} else {
		// nothing to do
		if vmNic.NetDev == "" {
			return nil
		}

		thisMemberName = vmNic.NetDev
	}

	err = switchIfDeleteMember(s.Name, thisMemberName)
	if err != nil {
		return fmt.Errorf("error removing member from switch: %w", err)
	}

	err = unsetVMNicRateLimit(vmNic)
	if err != nil {
		slog.Error("failed to unset nic rate limit", "nic", vmNic.ID, "netdev", vmNic.NetDev)

		return fmt.Errorf("error removing member from switch: %w", err)
	}

	return nil
}
