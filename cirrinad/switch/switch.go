package _switch

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
	"cirrina/cirrinad/vm_nics"
)

func GetById(id string) (s *Switch, err error) {
	db := getSwitchDb()
	db.Limit(1).Find(&s, "id = ?", id)
	if s.Name == "" {
		return s, errors.New("not found")
	}

	return s, nil
}

func GetByName(name string) (s *Switch, err error) {
	db := getSwitchDb()
	db.Limit(1).Find(&s, "name = ?", name)
	if s.ID == "" {
		return s, errors.New("not found")
	}

	return s, nil
}

func GetAll() []*Switch {
	var result []*Switch
	db := getSwitchDb()
	db.Find(&result)

	return result
}

func Create(name string, description string, switchType string, uplink string) (_switch *Switch, err error) {
	var switchInst *Switch
	if !util.ValidSwitchName(name) {
		return switchInst, errors.New("invalid name")
	}
	_, err = GetByName(name)
	if err == nil {
		slog.Error("switch exists", "switch", name)

		return switchInst, errors.New("switch exists")
	}

	if switchType != "IF" && switchType != "NG" {
		slog.Error("bad switch type", "switchType", switchType)

		return switchInst, errors.New("bad switch type")
	}

	switch switchType {
	case "IF":
		if uplink != "" {
			alreadyUsed, err := MemberUsedByIfBridge(uplink)
			if err != nil {
				return switchInst, errors.New("error checking if switch uplink in use by another bridge")
			}
			if alreadyUsed {
				return switchInst, errors.New("uplink already used")
			}
		}
	case "NG":
		if uplink != "" {
			alreadyUsed, err := MemberUsedByNgBridge(uplink)
			if err != nil {
				return switchInst, errors.New("error checking if switch uplink in use by another bridge")
			}
			if alreadyUsed {
				return switchInst, errors.New("uplink already used")
			}
		}
	default:
		slog.Error("bad switch type", "switchType", switchType)

		return switchInst, errors.New("bad switch type")
	}

	switchInst = &Switch{
		Name:        name,
		Description: description,
		Type:        switchType,
		Uplink:      uplink,
	}
	db := getSwitchDb()
	res := db.Create(&switchInst)

	return switchInst, res.Error
}

func Delete(id string) (err error) {
	if id == "" {
		return errors.New("unable to delete, switch id empty")
	}
	db := getSwitchDb()
	dSwitch, err := GetById(id)
	if err != nil {
		errorText := fmt.Sprintf("switch %v not found", id)

		return errors.New(errorText)
	}

	err2 := CheckSwitchInUse(id)
	if err2 != nil {
		return err2
	}

	res := db.Limit(1).Unscoped().Delete(&dSwitch)
	if res.RowsAffected != 1 {
		errText := fmt.Sprintf("switch delete error, rows affected %v", res.RowsAffected)

		return errors.New(errText)
	}

	return nil
}

func CheckSwitchInUse(id string) error {
	vmNics := vm_nics.GetAll()
	for _, vmNic := range vmNics {
		if vmNic.SwitchId == id {
			return errors.New("switch in use")
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
				err := DestroyIfBridge(bridge.Name, true)
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

func BridgeIfAddMember(bridgeName string, memberName string, learn bool) error {
	// TODO
	// netDevs := util.GetHostInterfaces()
	//
	// if !util.ContainsStr(netDevs, memberName) {
	// 	return errors.New("invalid switch member name")
	// }

	cmd := exec.Command(config.Config.Sys.Sudo, "/sbin/ifconfig", bridgeName, "addm", memberName)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		slog.Error("failed running ifconfig", "err", err, "out", out)
		errtxt := fmt.Sprintf("ifconfig failed: err: %v, out: %v", err, out)

		return errors.New(errtxt)
	}

	slog.Debug("learn info", "learn", learn)
	// code I was testing for disabling "learning" on bridges, ie, being like vmware to a degree -- that is, not
	// allow VMs to be "promiscuous" and snoop on each others traffic
	// decided not to use right now. may come back to it later

	// if !learn {
	// slog.Debug("BridgeIfAddMember", "learn", learn, "mac", mac)
	// cmd = exec.Command(config.Config.Sys.Sudo, "/sbin/ifconfig", bridgeName, "-learn", memberName)
	// cmd.Stdout = &out
	// if err := cmd.Start(); err != nil {
	// 	return err
	// }
	// if err := cmd.Wait(); err != nil {
	// 	slog.Error("failed running ifconfig", "err", err, "out", out)
	// 	errtxt := fmt.Sprintf("ifconfig failed: err: %v, out: %v", err, out)
	// 	return errors.New(errtxt)
	// }
	//
	// cmd = exec.Command(config.Config.Sys.Sudo, "/sbin/ifconfig", bridgeName, "-discover", memberName)
	// cmd.Stdout = &out
	// if err := cmd.Start(); err != nil {
	// 	return err
	// }
	// if err := cmd.Wait(); err != nil {
	// 	slog.Error("failed running ifconfig", "err", err, "out", out)
	// 	errtxt := fmt.Sprintf("ifconfig failed: err: %v, out: %v", err, out)
	// 	return errors.New(errtxt)
	// }
	// if mac != "" {
	// 	// https://cgit.freebsd.org/src/tree/sbin/ifconfig/ifbridge.c?id=eba230afba4932f02a1ca44efc797cf7499a5cb0#n405
	// 	// patched this to 0
	// 	cmd = exec.Command(config.Config.Sys.Sudo, "/usr/obj/usr/src/amd64.amd64/sbin/ifconfig/ifconfig", bridgeName, "static", memberName, mac)
	// 	cmd.Stdout = &out
	// 	if err := cmd.Start(); err != nil {
	// 		return err
	// 	}
	// 	if err := cmd.Wait(); err != nil {
	// 		slog.Error("failed running ifconfig", "err", err, "out", out)
	// 		errtxt := fmt.Sprintf("ifconfig failed: err: %v, out: %v", err, out)
	// 		return errors.New(errtxt)
	// 	}
	// }
	// }

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
		allNgBridgeMembers, err = GetNgBridgeMembers(aBridge)
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
		existingMembers, err := GetIfBridgeMembers(aBridge)
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

func ngGetBridgeNextLink(bridge string) (nextLink string, err error) {
	bridgePeers, err := GetNgBridgeMembers(bridge)
	if err != nil {
		return nextLink, err
	}

	nextLink = ngBridgeNextLink(bridgePeers)

	return nextLink, nil
}

func GetNgDev(switchId string) (bridge string, peer string, err error) {
	thisSwitch, err := GetById(switchId)
	if err != nil {
		slog.Error("switch lookup error", "switchid", switchId)
	}

	bridgePeers, err := GetNgBridgeMembers(thisSwitch.Name)
	if err != nil {
		return "", "", err
	}

	nextLink := ngBridgeNextLink(bridgePeers)

	return thisSwitch.Name, nextLink, nil
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
		return errors.New("unknown switch type")
	}
}

func (d *Switch) SetUplink(uplink string) error {
	netDevs := util.GetHostInterfaces()

	if !util.ContainsStr(netDevs, uplink) {
		return errors.New("invalid switch uplink name")
	}

	switch d.Type {
	case "IF":
		alreadyUsed, err := MemberUsedByIfBridge(uplink)
		if err != nil {
			return err
		}
		if alreadyUsed {
			slog.Error("another bridge already contains member, member can not be in two bridges of "+
				"same type, skipping adding", "member", uplink,
			)

			return errors.New("uplink already used")
		}

		slog.Debug("setting IF bridge uplink", "id", d.ID)
		err = BridgeIfAddMember(d.Name, uplink, true)
		if err != nil {
			return err
		}
		d.Uplink = uplink
		err = d.Save()
		if err != nil {
			return err
		}

		return nil
	case "NG":
		// it can't be a member of another bridge already
		alreadyUsed, err := MemberUsedByNgBridge(uplink)
		if err != nil {
			slog.Error("error checking if member already used", "err", err)
			if err != nil {
				return err
			}
		}
		if alreadyUsed {
			slog.Error("another bridge already contains member, member can not be in two bridges of "+
				"same type, skipping adding", "member", uplink,
			)

			return errors.New("uplink already used")
		}
		slog.Debug("setting NG bridge uplink", "id", d.ID)
		err = BridgeNgAddMember(d.Name, uplink)
		if err != nil {
			return err
		}
		d.Uplink = uplink
		err = d.Save()
		if err != nil {
			return err
		}

		return nil
	default:
		return errors.New("unknown switch type")
	}
}

func (d *Switch) Save() error {
	db := getSwitchDb()

	res := db.Model(&d).
		Updates(map[string]interface{}{
			"name":        &d.Name,
			"description": &d.Description,
			"type":        &d.Type,
			"uplink":      &d.Uplink,
		},
		)

	if res.Error != nil {
		return errors.New("error updating switch")
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

		return err
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

		return err
	}

	return nil
}

func DestroyIfBridge(name string, cleanup bool) error {
	// TODO allow other bridge names
	if !strings.HasPrefix(name, "bridge") {
		slog.Error("invalid bridge name", "name", name)

		return errors.New("invalid bridge name")
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
		return err
	}
	if err := cmd.Wait(); err != nil {
		slog.Error("failed running ifconfig", "err", err, "out", out)

		return err
	}

	return nil
}

func DestroyNgBridge(netDev string) (err error) {
	if netDev == "" {
		return errors.New("netDev can't be empty")
	}
	cmd := exec.Command(config.Config.Sys.Sudo, "/usr/sbin/ngctl", "msg",
		netDev+":", "shutdown")
	err = cmd.Run()
	if err != nil {
		slog.Error("ngctl msg shutdown error", "err", err)

		return err
	}

	return nil
}

type Switch struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null;default:null"`
	Name        string `gorm:"uniqueIndex;not null;default:null"`
	Description string
	Type        string `gorm:"default:IF;check:type IN ('IF','NG')"`
	Uplink      string
}

func ParseSwitchId(switchId string, netDevType string) (res string, err error) {
	if switchId == "" {
		return switchId, err
	}

	switchUuid, err := uuid.Parse(switchId)
	if err != nil {
		return res, errors.New("switch id invalid")
	}
	switchInst, err := GetById(switchUuid.String())
	if err != nil {
		slog.Debug("error getting switch id",
			"id", switchId,
			"err", err,
		)

		return res, errors.New("switch id invalid")
	}
	if switchInst.Name == "" {
		return res, errors.New("switch id invalid")
	}
	if netDevType == "TAP" || netDevType == "VMNET" {
		if switchInst.Type != "IF" {
			return res, errors.New("uplink switch has wrong type")
		}
	} else if netDevType == "NETGRAPH" {
		if switchInst.Type != "NG" {
			return res, errors.New("uplink switch has wrong type")
		}
	}
	res = switchUuid.String()

	return res, nil
}
