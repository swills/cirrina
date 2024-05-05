package vmswitch

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	exec "golang.org/x/sys/execabs"
	"gorm.io/gorm"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vmnic"
)

type Switch struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null;default:null"`
	Name        string `gorm:"uniqueIndex;not null;default:null"`
	Description string
	Type        string `gorm:"default:IF;check:type IN ('IF','NG')"`
	Uplink      string
}

func switchTypeValid(switchType string) bool {
	switch switchType {
	case "IF":
		return true
	case "NG":
		return true
	default:
		return false
	}
}

func switchCheckUplink(switchInst *Switch) error {
	switch switchInst.Type {
	case "IF":
		if switchInst.Uplink != "" {
			alreadyUsed, err := MemberUsedByIfBridge(switchInst.Uplink)
			if err != nil {
				return errSwitchInternalChecking
			}
			if alreadyUsed {
				return errSwitchUplinkInUse
			}
		}
	case "NG":
		if switchInst.Uplink != "" {
			alreadyUsed, err := MemberUsedByNgBridge(switchInst.Uplink)
			if err != nil {
				return errSwitchInternalChecking
			}
			if alreadyUsed {
				return errSwitchUplinkInUse
			}
		}
	default:
		slog.Error("bad switch type", "switchType", switchInst.Type)

		return errSwitchInvalidType
	}

	return nil
}

func validateSwitch(switchInst *Switch) error {
	if !util.ValidSwitchName(switchInst.Name) {
		return errSwitchInvalidName
	}

	if !switchTypeValid(switchInst.Type) {
		return errSwitchInvalidType
	}

	err := switchCheckUplink(switchInst)
	if err != nil {
		return err
	}

	return nil
}

func Create(switchInst *Switch) error {
	err := validateSwitch(switchInst)
	if err != nil {
		slog.Error("error validating switch", "switch", switchInst.Name, "err", err)

		return err
	}

	switchAlreadyExists, err := switchExists(switchInst.Name)
	if err != nil {
		slog.Error("error checking db for switch", "name", switchInst.Name, "err", err)

		return err
	}
	if switchAlreadyExists {
		slog.Error("switch exists", "switch", switchInst.Name)

		return errSwitchExists
	}

	db := getSwitchDB()
	res := db.Create(&switchInst)
	if res.RowsAffected != 1 {
		return fmt.Errorf("incorrect number of rows affected, err: %w", res.Error)
	}
	if res.Error != nil {
		return res.Error
	}

	return nil
}

func GetByID(switchID string) (*Switch, error) {
	var aSwitch *Switch
	db := getSwitchDB()
	res := db.Limit(1).Find(&aSwitch, "id = ?", switchID)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected != 1 {
		return nil, errSwitchNotFound
	}

	return aSwitch, nil
}

func GetByName(name string) (*Switch, error) {
	var aSwitch *Switch
	db := getSwitchDB()
	res := db.Limit(1).Find(&aSwitch, "name = ?", name)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected != 1 {
		return nil, errSwitchNotFound
	}

	return aSwitch, nil
}

func GetAll() []*Switch {
	var result []*Switch
	db := getSwitchDB()
	db.Find(&result)

	return result
}

func (d *Switch) Save() error {
	db := getSwitchDB()

	res := db.Model(&d).
		Updates(map[string]interface{}{
			"name":        &d.Name,
			"description": &d.Description,
			"type":        &d.Type,
			"uplink":      &d.Uplink,
		},
		)

	if res.Error != nil {
		return errSwitchInternalDB
	}

	return nil
}

func Delete(switchID string) error {
	if switchID == "" {
		return errSwitchInvalidID
	}
	switchDB := getSwitchDB()
	dSwitch, err := GetByID(switchID)
	if err != nil {
		return errSwitchNotFound
	}

	err2 := CheckSwitchInUse(switchID)
	if err2 != nil {
		return err2
	}

	res := switchDB.Limit(1).Unscoped().Delete(&dSwitch)
	if res.RowsAffected != 1 {
		slog.Error("error saving switch", "res", res)

		return errSwitchInternalDB
	}

	return nil
}

func CheckSwitchInUse(id string) error {
	vmNics := vmnic.GetAll()
	for _, vmNic := range vmNics {
		if vmNic.SwitchID == id {
			return errSwitchInUse
		}
	}

	return nil
}

func CheckInterfaceExists(interfaceName string) bool {
	netDevs := util.GetHostInterfaces()

	for _, nic := range netDevs {
		if nic == interfaceName {
			return true
		}
	}

	return false
}

func CreateBridges() {
	allBridges := GetAll()

	for num, bridge := range allBridges {
		slog.Debug("creating bridge", "num", num, "bridge", bridge.Name)
		switch bridge.Type {
		case "IF":
			slog.Debug("creating if bridge", "name", bridge.Name)
			err := BuildIfBridge(bridge)
			if err != nil {
				slog.Error("error creating if bridge", "err", err)

				return
			}
		case "NG":
			slog.Debug("creating ng bridge", "name", bridge.Name)
			err := BuildNgBridge(bridge)
			if err != nil {
				slog.Error("error creating ng bridge",
					"name", bridge.Name,
					"err", err,
				)

				return
			}
		default:
			slog.Debug("unknown bridge type", "name", bridge.Name, "type", bridge.Type)
		}
	}
}

func DestroyBridges() {
	allBridges := GetAll()

	exitingIfBridges, err := GetAllIfBridges()
	if err != nil {
		slog.Error("error getting all if bridges")
	}
	exitingNgBridges, err := GetAllNgBridges()
	if err != nil {
		slog.Error("error getting all ng bridges")
	}

	for _, bridge := range allBridges {
		switch bridge.Type {
		case "IF":
			if util.ContainsStr(exitingIfBridges, bridge.Name) {
				slog.Debug("destroying if bridge", "name", bridge.Name)
				err = DestroyIfBridge(bridge.Name, true)
				if err != nil {
					slog.Error("error destroying if bridge", "err", err)
				}
			}
		case "NG":
			if util.ContainsStr(exitingNgBridges, bridge.Name) {
				slog.Debug("destroying ng bridge", "name", bridge.Name)
				err = DestroyNgBridge(bridge.Name)
				if err != nil {
					slog.Error("error destroying if bridge", "err", err)
				}
			}
		default:
			slog.Debug("unknown bridge type", "name", bridge.Name, "type", bridge.Type)
		}
	}
}

func BridgeIfAddMember(bridgeName string, memberName string) error {
	// TODO - check that the member name is a host interface or a VM nic interface

	cmd := exec.Command(config.Config.Sys.Sudo, "/sbin/ifconfig", bridgeName, "addm", memberName)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error running ifconfig command: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("error running ifconfig: %w", err)
	}

	return nil
}

func MemberUsedByNgBridge(member string) (bool, error) {
	allBridges, err := GetAllNgBridges()
	if err != nil {
		slog.Error("error getting all if bridges", "err", err)

		return false, err
	}
	for _, aBridge := range allBridges {
		var allNgBridgeMembers []ngPeer
		var existingMembers []string

		// extra work here since this returns a ngPeer
		allNgBridgeMembers, err = getNgBridgeMembers(aBridge)
		if err != nil {
			slog.Error("error getting ng bridge members", "bridge", aBridge)

			return false, err
		}
		for _, m := range allNgBridgeMembers {
			existingMembers = append(existingMembers, m.PeerName)
		}
		if util.ContainsStr(existingMembers, member) {
			return true, nil
		}
	}

	return false, nil
}

func BuildNgBridge(switchInst *Switch) error {
	var members []string
	// TODO remove all these de-normalizations in favor of gorm native "Has Many" relationships
	memberList := strings.Split(switchInst.Uplink, ",")

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
				"bridge", switchInst.Name, "uplink", member,
			)

			continue
		}
		// it can't be a member of another bridge already
		alreadyUsed, err := MemberUsedByNgBridge(member)
		if err != nil {
			slog.Error("error checking if member already used", "err", err)

			continue
		}
		if alreadyUsed {
			slog.Error("another bridge already contains member, member can not be in two bridges of "+
				"same type, skipping adding", "bridge", switchInst.Name, "member", member,
			)

			continue
		}
		members = append(members, member)
	}

	err := createNgBridgeWithMembers(switchInst.Name, members)

	return err
}

func MemberUsedByIfBridge(member string) (bool, error) {
	allBridges, err := GetAllIfBridges()
	if err != nil {
		slog.Error("error getting all if bridges", "err", err)
	}
	for _, aBridge := range allBridges {
		existingMembers, err := getIfBridgeMembers(aBridge)
		if err != nil {
			slog.Error("error getting if bridge members", "bridge", aBridge)

			return false, err
		}
		if util.ContainsStr(existingMembers, member) {
			return true, nil
		}
	}

	return false, nil
}

func BuildIfBridge(switchInst *Switch) error {
	var members []string
	// TODO remove all these de-normalizations in favor of gorm native "Has Many" relationships
	memberList := strings.Split(switchInst.Uplink, ",")

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
				"bridge", switchInst.Name, "uplink", member,
			)

			continue
		}
		// it can't be a member of another bridge already
		alreadyUsed, err := MemberUsedByIfBridge(member)
		if err != nil {
			slog.Error("error checking if member already used", "err", err)

			continue
		}
		if alreadyUsed {
			slog.Error("another bridge already contains member, member can not be in two bridges of "+
				"same type, skipping adding", "bridge", switchInst.Name, "member", member,
			)

			continue
		}
		members = append(members, member)
	}
	err := CreateIfBridgeWithMembers(switchInst.Name, members)

	return err
}

func ngGetBridgeNextLink(bridge string) (string, error) {
	var nextLink string
	var err error

	bridgePeers, err := getNgBridgeMembers(bridge)
	if err != nil {
		return nextLink, err
	}

	nextLink = ngBridgeNextLink(bridgePeers)

	return nextLink, nil
}

// GetNgDev returns the netDev (stored in DB) and netDevArg (passed to bhyve)
func GetNgDev(switchID string, name string) (string, string, error) {
	var err error

	thisSwitch, err := GetByID(switchID)
	if err != nil {
		slog.Error("switch lookup error", "switchid", switchID)
	}

	bridgePeers, err := getNgBridgeMembers(thisSwitch.Name)
	if err != nil {
		return "", "", err
	}

	nextLink := ngBridgeNextLink(bridgePeers)

	ngNetDev := thisSwitch.Name + "," + nextLink
	netDevArg := "netgraph,path=" + thisSwitch.Name + ":,peerhook=" + nextLink + ",socket=" + name

	return ngNetDev, netDevArg, nil
}

func (d *Switch) UnsetUplink() error {
	switch d.Type {
	case "IF":
		slog.Debug("unsetting IF bridge uplink", "id", d.ID)
		err := bridgeIfDeleteMember(d.Name, d.Uplink)
		if err != nil {
			return err
		}
		d.Uplink = ""
		err = d.Save()
		if err != nil {
			return err
		}

		return nil
	case "NG":
		slog.Debug("unsetting NG bridge uplink", "id", d.ID)
		err := bridgeNgRemoveUplink(d.Name, d.Uplink)
		if err != nil {
			return err
		}
		d.Uplink = ""
		err = d.Save()
		if err != nil {
			return err
		}

		return nil
	default:
		return errSwitchInvalidType
	}
}

func (d *Switch) SetUplink(uplink string) error {
	netDevs := util.GetHostInterfaces()

	if !util.ContainsStr(netDevs, uplink) {
		return errSwitchInvalidUplink
	}

	switch d.Type {
	case "IF":
		return setUplinkIf(uplink, d)
	case "NG":
		return setUplinkNG(uplink, d)
	default:
		return errSwitchInvalidType
	}
}

func setUplinkNG(uplink string, switchInst *Switch) error {
	// it can't be a member of another bridge already
	alreadyUsed, err := MemberUsedByNgBridge(uplink)
	if err != nil {
		slog.Error("error checking if member already used", "err", err)

		return err
	}
	if alreadyUsed {
		slog.Error("another bridge already contains member, member can not be in two bridges of "+
			"same type, skipping adding", "member", uplink,
		)

		return errSwitchUplinkInUse
	}
	slog.Debug("setting NG bridge uplink", "id", switchInst.ID)
	err = BridgeNgAddMember(switchInst.Name, uplink)
	if err != nil {
		return err
	}
	switchInst.Uplink = uplink
	err = switchInst.Save()
	if err != nil {
		return err
	}

	return nil
}

func setUplinkIf(uplink string, switchInst *Switch) error {
	alreadyUsed, err := MemberUsedByIfBridge(uplink)
	if err != nil {
		return err
	}
	if alreadyUsed {
		slog.Error("another bridge already contains member, member can not be in two bridges of "+
			"same type, skipping adding", "member", uplink,
		)

		return errSwitchUplinkInUse
	}

	slog.Debug("setting IF bridge uplink", "id", switchInst.ID)
	err = BridgeIfAddMember(switchInst.Name, uplink)
	if err != nil {
		return err
	}
	switchInst.Uplink = uplink
	err = switchInst.Save()
	if err != nil {
		return err
	}

	return nil
}

func BridgeNgAddMember(bridgeName string, memberName string) error {
	link, err := ngGetBridgeNextLink(bridgeName)
	if err != nil {
		return err
	}
	cmd := exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "connect",
		memberName+":", bridgeName+":", "lower", link)
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl connect error", "err", err)

		return fmt.Errorf("error running ngctl command: %w", err)
	}

	link, err = ngGetBridgeNextLink(bridgeName)
	if err != nil {
		return err
	}
	cmd = exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "connect",
		memberName+":", bridgeName+":", "upper", link)
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl connect error", "err", err)

		return fmt.Errorf("error running ngctl command: %w", err)
	}

	return nil
}

func DestroyIfBridge(name string, cleanup bool) error {
	// TODO allow other bridge names
	if !strings.HasPrefix(name, "bridge") {
		slog.Error("invalid bridge name", "name", name)

		return errSwitchInvalidName
	}
	if cleanup {
		err := bridgeIfDeleteAllMembers(name)
		if err != nil {
			return err
		}
	}
	cmd := exec.Command(config.Config.Sys.Sudo, "/sbin/ifconfig", name, "destroy")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error running ifconfig command: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		slog.Error("failed running ifconfig", "err", err, "out", out)

		return fmt.Errorf("error running ifconfig command: %w", err)
	}

	return nil
}

func DestroyNgBridge(netDev string) error {
	var err error
	if netDev == "" {
		return errSwitchInvalidNetDevEmpty
	}
	cmd := exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "msg",
		netDev+":", "shutdown")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl msg shutdown error", "err", err)

		return fmt.Errorf("error running ngctl command: %w", err)
	}

	return nil
}

func ParseSwitchID(switchID string, netDevType string) (string, error) {
	var res string
	if switchID == "" {
		return switchID, errSwitchInvalidID
	}

	switchUUID, err := uuid.Parse(switchID)
	if err != nil {
		return res, errSwitchInvalidID
	}
	switchInst, err := GetByID(switchUUID.String())
	if err != nil {
		slog.Debug("error getting switch id",
			"id", switchID,
			"err", err,
		)

		return res, errSwitchInvalidID
	}
	if switchInst.Name == "" {
		return res, errSwitchInvalidName
	}
	if netDevType == "TAP" || netDevType == "VMNET" {
		if switchInst.Type != "IF" {
			return res, errSwitchUplinkWrongType
		}
	} else if netDevType == "NETGRAPH" {
		if switchInst.Type != "NG" {
			return res, errSwitchUplinkWrongType
		}
	}
	res = switchUUID.String()

	return res, nil
}

func switchExists(switchName string) (bool, error) {
	var err error
	_, err = GetByName(switchName)
	if err != nil {
		if !errors.Is(err, errSwitchNotFound) {
			return false, err
		}

		return false, nil
	}

	return true, nil
}
