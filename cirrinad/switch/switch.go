package _switch

import (
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
	"errors"
	"fmt"
	"golang.org/x/exp/slog"
	"os/exec"
	"strings"
)

func GetById(id string) (s *Switch, err error) {
	db := getSwitchDb()
	db.Limit(1).Find(&s, "id = ?", id)
	return s, nil
}

func GetByName(name string) (s *Switch, err error) {
	db := getSwitchDb()
	db.Limit(1).Find(&s, "name = ?", name)
	return s, nil
}

func GetAll() []*Switch {
	var result []*Switch
	db := getSwitchDb()
	db.Find(&result)
	return result
}

func Create(name string, description string, switchType string) (_switch *Switch, err error) {
	var switchInst *Switch
	if strings.Contains(name, "/") {
		return switchInst, errors.New("illegal character in switch name")
	}
	existingSwitch, err := GetByName(name)
	if err != nil {
		slog.Error("error checking db for switch", "name", name, "err", err)
		return switchInst, err
	}
	if existingSwitch.Name != "" {
		slog.Error("switch exists", "switch", name)
		return switchInst, errors.New("switch exists")
	}

	if switchType != "IF" && switchType != "NG" {
		slog.Error("bad switch type", "switchType", switchType)
		return switchInst, errors.New("bad switch type")
	}

	switchInst = &Switch{
		Name:        name,
		Description: description,
		Type:        switchType,
	}
	db := getSwitchDb()
	res := db.Create(&switchInst)
	return switchInst, res.Error
}

func Delete(id string) (err error) {
	// TODO check that switch is not in use
	if id == "" {
		return errors.New("unable to delete, switch id empty")
	}
	db := getSwitchDb()
	dSwitch, err := GetById(id)
	if err != nil {
		errorText := fmt.Sprintf("switch %v not found", id)
		return errors.New(errorText)
	}
	res := db.Limit(1).Delete(&dSwitch)
	if res.RowsAffected != 1 {
		errText := fmt.Sprintf("switch delete error, rows affected %v", res.RowsAffected)
		return errors.New(errText)
	}
	return nil
}

func CreateBridges() {
	allBridges := GetAll()
	allIfBridges, err := getAllIfBridges()
	if err != nil {
		slog.Debug("failed to get all if bridges", "err", err)
		return
	}

	for num, bridge := range allBridges {
		slog.Debug("creating bridge", "num", num, "bridge", bridge.Name)
		if bridge.Type == "IF" {
			slog.Debug("creating if bridge", "name", bridge.Name)
			if util.ContainsStr(allIfBridges, bridge.Name) {
				slog.Debug("bridge already exists, skipping", "bridge", bridge.Name)
			} else {
				err := BuildIfBridge(bridge)
				if err != nil {
					slog.Error("error creating if bridge", "err", err)
					return
				}
			}
		} else if bridge.Type == "NG" {
			slog.Debug("creating ng bridge", "name", bridge.Name)
			bridgeList, err := ngGetBridges()
			if err != nil {
				slog.Error("error getting bridge list", "err", err)
				return
			}
			if !util.ContainsStr(bridgeList, bridge.Name) {
				err := ngCreateBridge(bridge.Name, bridge.Uplink)
				if err != nil {
					slog.Error("ngCreateBridge err", "err", err)
				}
			}
		} else {
			slog.Debug("unknown bridge type", "name", bridge.Name, "type", bridge.Type)
		}
	}
}

func DestroyBridges() {
	allBridges := GetAll()
	for num, bridge := range allBridges {
		slog.Debug("destroying bridge", "num", num, "bridge", bridge.Name)
		if bridge.Type == "IF" {
			slog.Debug("destroying if bridge", "name", bridge.Name)
			err := deleteIfBridge(bridge.Name, true)
			if err != nil {
				slog.Debug("error destroying if bridge", "err", err)
			}
		} else if bridge.Type == "NG" {
			slog.Debug("destroying ng bridge", "name", bridge.Name)
			err := ngDestroyBridge(bridge.Name)
			if err != nil {
				slog.Debug("error destroying if bridge", "err", err)
			}
		} else {
			slog.Debug("unknown bridge type", "name", bridge.Name, "type", bridge.Type)
		}
	}
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

func BuildIfBridge(switchInst *Switch) error {
	var members []string
	// TODO remove all these de-normalizations in favor of gorm native "Has Many" relationships
	memberList := strings.Split(switchInst.Uplink, ",")
	for _, member := range memberList {
		if member == "" {
			continue
		}
		members = append(members, member)
	}

	err := createIfBridgeWithMembers(switchInst.Name, members)
	return err
}

func GetNgDev(switchId string) (bridge string, peer string, err error) {
	thisSwitch, err := GetById(switchId)
	if err != nil {
		slog.Error("switch lookup error", "switchid", switchId)
	}

	bridgePeers, err := ngGetBridgePeers(thisSwitch.Name)
	if err != nil {
		return "", "", err
	}

	nextLink := ngBridgeNextPeer(bridgePeers)
	return thisSwitch.Name, nextLink, nil
}
