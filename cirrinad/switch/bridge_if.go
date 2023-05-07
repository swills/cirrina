package _switch

import (
	"bufio"
	"cirrina/cirrinad/config"
	"errors"
	"fmt"
	"golang.org/x/exp/slog"
	"os/exec"
	"strings"
)

func GetAllIfBridges() (bridges []string, err error) {
	var r []string
	args := []string{"-g", "bridge"}
	cmd := exec.Command("/sbin/ifconfig", args...)
	defer func(cmd *exec.Cmd) {
		err := cmd.Wait()
		if err != nil {
			slog.Error("ifconfig error", "err", err)
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
		if len(textFields) != 1 {
			continue
		}
		aBridgeName := textFields[0]
		r = append(r, aBridgeName)
	}
	if err := scanner.Err(); err != nil {
		fmt.Println(err)
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
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
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
		fmt.Println(err)
	}
	return members, nil
}

func CreateIfBridge(name string) error {
	// TODO allow other bridge names by creating with a dummy name and then renaming
	if !strings.HasPrefix(name, "bridge") {
		slog.Error("invalid bridge name", "name", name)
		return errors.New("invalid bridge name")
	}

	cmd := exec.Command(config.Config.Sys.Sudo, "/sbin/ifconfig", name, "create", "up")
	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		exiterr, ok := err.(*exec.ExitError)
		if !ok {
			slog.Error("failed running ifconfig", "exec", exiterr, "err", err)
			return err
		}
	}
	return nil
}

func DeleteIfBridge(name string) error {
	// TODO allow other bridge names
	if !strings.HasPrefix(name, "bridge") {
		slog.Error("invalid bridge name", "name", name)
		return errors.New("invalid bridge name")
	}
	err := BridgeIfDeleteAllMembers(name)
	if err != nil {
		return err
	}
	cmd := exec.Command(config.Config.Sys.Sudo, "/sbin/ifconfig", name, "destroy")
	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		exiterr, ok := err.(*exec.ExitError)
		if !ok {
			slog.Error("failed running ifconfig", "exec", exiterr, "err", err)
			return err
		}
	}
	return nil

}

func BridgeIfDeleteAllMembers(name string) error {
	bridgeMembers, err := GetIfBridgeMembers(name)
	slog.Debug("deleting all if bridge members", "bridge", name, "members", bridgeMembers)
	if err != nil {
		return err
	}
	for _, member := range bridgeMembers {
		err := BridgeIfDeleteMember(name, member)
		if err != nil {
			return err
		}
	}
	return nil
}

func BridgeIfAddMember(bridgeName string, memberName string) error {
	cmd := exec.Command(config.Config.Sys.Sudo, "/sbin/ifconfig", bridgeName, "addm", memberName)
	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		exiterr, ok := err.(*exec.ExitError)
		if !ok {
			slog.Error("failed running ifconfig", "exec", exiterr, "err", err)
			return err
		}
	}
	return nil
}

func BridgeIfDeleteMember(bridgeName string, memberName string) error {
	cmd := exec.Command(config.Config.Sys.Sudo, "/sbin/ifconfig", bridgeName, "deletem", memberName)
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		exiterr, ok := err.(*exec.ExitError)
		if !ok {
			slog.Error("failed running ifconfig", "exec", exiterr, "err", err)
			return err
		}
	}
	return nil
}

func CreateIfBridgeWithMembers(bridgeName string, bridgeMembers []string) error {
	err := CreateIfBridge(bridgeName)
	if err != nil {
		return err
	}
	err = BridgeIfDeleteAllMembers(bridgeName)
	if err != nil {
		return err
	}
	for _, member := range bridgeMembers {
		err = BridgeIfAddMember(bridgeName, member)
		if err != nil {
			return err
		}
	}
	return nil
}
