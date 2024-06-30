package vmswitch

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
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

var vmnicGetAllFunc = vmnic.GetAll

func switchTypeValid(switchInst *Switch) bool {
	switch switchInst.Type {
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
			alreadyUsed, err := memberUsedByIfBridge(switchInst.Uplink)
			if err != nil {
				return errSwitchInternalChecking
			}

			if alreadyUsed {
				return errSwitchUplinkInUse
			}
		}
	case "NG":
		if switchInst.Uplink != "" {
			alreadyUsed, err := memberUsedByNgBridge(switchInst.Uplink)
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
	// switchNameValid also checks type, no need to check here
	if !switchNameValid(switchInst) {
		return errSwitchInvalidName
	}

	err := switchCheckUplink(switchInst)
	if err != nil {
		return err
	}

	// default case unreachable
	switch switchInst.Type {
	case "IF":
		return validateIfSwitch(switchInst)
	case "NG":
		return validateNgSwitch(switchInst)
	default:
		return errSwitchInvalidType
	}
}

func memberUsedByNgBridge(member string) (bool, error) {
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

func buildNgBridge(switchInst *Switch) error {
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
		alreadyUsed, err := memberUsedByNgBridge(member)
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

func memberUsedByIfBridge(member string) (bool, error) {
	allBridges, err := GetAllIfBridges()
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

func buildIfBridge(switchInst *Switch) error {
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
		alreadyUsed, err := memberUsedByIfBridge(member)
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

func setUplinkNG(uplink string, switchInst *Switch) error {
	// it can't be a member of another bridge already
	alreadyUsed, err := memberUsedByNgBridge(uplink)
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
	alreadyUsed, err := memberUsedByIfBridge(uplink)
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

func bringUpNewSwitch(switchInst *Switch) error {
	if switchInst == nil || switchInst.ID == "" {
		return errSwitchInvalidID
	}

	switch switchInst.Type {
	case "IF":
		slog.Debug("creating if bridge", "name", switchInst.Name)

		err := buildIfBridge(switchInst)
		if err != nil {
			slog.Error("error creating if bridge", "err", err)
			// already created in db, so ignore system state and proceed on...
			return err
		}
	case "NG":
		slog.Debug("creating ng bridge", "name", switchInst.Name)

		err := buildNgBridge(switchInst)
		if err != nil {
			slog.Error("error creating ng bridge", "err", err)
			// already created in db, so ignore system state and proceed on...
			return nil
		}
	default:
		slog.Error("unknown switch type bringing up new switch")

		return errSwitchInvalidType
	}

	return nil
}

func validateIfSwitch(switchInst *Switch) error {
	// it can't be a member of another bridge of same type already
	if switchInst.Uplink != "" {
		alreadyUsed, err := memberUsedByIfBridge(switchInst.Uplink)
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

func validateNgSwitch(switchInst *Switch) error {
	// it can't be a member of another bridge of same type already
	if switchInst.Uplink != "" {
		alreadyUsed, err := memberUsedByNgBridge(switchInst.Uplink)
		if err != nil {
			slog.Error("error checking if member already used", "err", err)

			return fmt.Errorf("error checking if member already used: %w", err)
		}

		if alreadyUsed {
			return errSwitchUplinkInUse
		}
	}

	return nil
}

func switchNameValid(switchInst *Switch) bool {
	if switchInst.Name == "" {
		return false
	}

	// values must be kept sorted
	myRT := &unicode.RangeTable{
		R16: []unicode.Range16{
			{0x002d, 0x002d, 1}, // -
			{0x0030, 0x0039, 1}, // numbers
			{0x0041, 0x005a, 1}, // upper case letters
			{0x005f, 0x005f, 1}, // _
			{0x0061, 0x007a, 1}, // lower case letters
		},
		LatinOffset: 0,
	}

	validChars := util.CheckInRange(switchInst.Name, myRT)
	if !validChars {
		return false
	}

	switch switchInst.Type {
	case "IF":
		if !strings.HasPrefix(switchInst.Name, "bridge") {
			slog.Error("invalid name", "name", switchInst.Name)

			return false
		}

		bridgeNumStr := strings.TrimPrefix(switchInst.Name, "bridge")

		bridgeNum, err := strconv.Atoi(bridgeNumStr)
		if err != nil {
			slog.Error("invalid bridge name", "name", switchInst.Name)

			return false
		}

		bridgeNumFormattedString := strconv.FormatInt(int64(bridgeNum), 10)
		// Check for silly things like "0123"
		if bridgeNumStr != bridgeNumFormattedString {
			slog.Error("invalid name", "name", switchInst.Name)

			return false
		}
	case "NG":
		if !strings.HasPrefix(switchInst.Name, "bnet") {
			slog.Error("invalid bridge name", "name", switchInst.Name)

			return false
		}

		bridgeNumStr := strings.TrimPrefix(switchInst.Name, "bnet")

		bridgeNum, err := strconv.Atoi(bridgeNumStr)
		if err != nil {
			slog.Error("invalid bridge name", "name", switchInst.Name)

			return false
		}

		bridgeNumFormattedString := strconv.FormatInt(int64(bridgeNum), 10)
		// Check for silly things like "0123"
		if bridgeNumStr != bridgeNumFormattedString {
			slog.Error("invalid name", "name", switchInst.Name)

			return false
		}
	default:
		return false
	}

	return true
}

func BridgeIfAddMember(bridgeName string, memberName string) error {
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

func BridgeNgAddMember(bridgeName string, memberName string) error {
	link, err := ngGetBridgeNextLink(bridgeName)
	if err != nil {
		return err
	}

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/ngctl", "connect", memberName + ":", bridgeName + ":", "lower", link},
	)
	if err != nil {
		slog.Error("ngctl connect error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ngctl connect error: %w", err)
	}

	link, err = ngGetBridgeNextLink(bridgeName)
	if err != nil {
		return err
	}

	stdOutBytes, stdErrBytes, returnCode, err = util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/ngctl", "connect", memberName + ":", bridgeName + ":", "upper", link},
	)
	if err != nil {
		slog.Error("ngctl connect error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ngctl connect error: %w", err)
	}

	return nil
}

func Create(switchInst *Switch) error {
	switchAlreadyExists, err := switchExists(switchInst.Name)
	if err != nil {
		slog.Error("error checking db for switch", "name", switchInst.Name, "err", err)

		return err
	}

	if switchAlreadyExists {
		slog.Error("switch exists", "switch", switchInst.Name)

		return errSwitchExists
	}

	err = validateSwitch(switchInst)
	if err != nil {
		slog.Error("error validating switch", "switch", switchInst.Name, "err", err)

		return err
	}

	db := getSwitchDB()

	res := db.Create(&switchInst)

	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected != 1 {
		return fmt.Errorf("incorrect number of rows affected, err: %w", res.Error)
	}

	return nil
}

func CheckSwitchInUse(id string) error {
	vmNics := vmnicGetAllFunc()
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

func CreateBridges() error {
	allBridges := GetAll()

	for num, bridge := range allBridges {
		slog.Debug("creating bridge", "num", num, "bridge", bridge.Name)

		switch bridge.Type {
		case "IF":
			slog.Debug("creating if bridge", "name", bridge.Name)

			err := buildIfBridge(bridge)
			if err != nil {
				slog.Error("error creating if bridge", "err", err)

				return fmt.Errorf("error creating if bridge: %w", err)
			}
		case "NG":
			slog.Debug("creating ng bridge", "name", bridge.Name)

			err := buildNgBridge(bridge)
			if err != nil {
				slog.Error("error creating ng bridge",
					"name", bridge.Name,
					"err", err,
				)

				return fmt.Errorf("error creating ng bridge: %w", err)
			}
		default:
			slog.Debug("unknown bridge type", "name", bridge.Name, "type", bridge.Type)

			return errSwitchInvalidType
		}
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

func DestroyBridges() error {
	allBridges := GetAll()

	exitingIfBridges, err := GetAllIfBridges()
	if err != nil {
		slog.Error("error getting all if bridges")

		return fmt.Errorf("error getting all if bridges: %w", err)
	}

	exitingNgBridges, err := GetAllNgBridges()
	if err != nil {
		slog.Error("error getting all ng bridges")

		return fmt.Errorf("error getting all ng bridges: %w", err)
	}

	for _, bridge := range allBridges {
		switch bridge.Type {
		case "IF":
			if util.ContainsStr(exitingIfBridges, bridge.Name) {
				slog.Debug("destroying if bridge", "name", bridge.Name)

				err = DestroyIfBridge(bridge.Name, true)
				if err != nil {
					slog.Error("error destroying if bridge", "err", err)

					return fmt.Errorf("error destroying if bridge: %w", err)
				}
			}
		case "NG":
			if util.ContainsStr(exitingNgBridges, bridge.Name) {
				slog.Debug("destroying ng bridge", "name", bridge.Name)

				err = DestroyNgBridge(bridge.Name)
				if err != nil {
					slog.Error("error destroying ng bridge", "err", err)

					return fmt.Errorf("error destroying ng bridge: %w", err)
				}
			}
		default:
			slog.Debug("unknown bridge type", "name", bridge.Name, "type", bridge.Type)

			return errSwitchInvalidType
		}
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

func DestroyNgBridge(netDev string) error {
	var err error

	if netDev == "" {
		return errSwitchInvalidNetDevEmpty
	}

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/usr/sbin/ngctl", "msg", netDev + ":", "shutdown"},
	)
	if err != nil {
		slog.Error("ngctl msg shutdown error",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("ngctl msg shutdown error: %w", err)
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
	if name == "" {
		return nil, errSwitchNotFound
	}

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

// GetNgDev returns the netDev (stored in DB) and netDevArg (passed to bhyve)
func GetNgDev(switchID string, name string) (string, string, error) {
	var err error

	thisSwitch, err := GetByID(switchID)
	if err != nil {
		slog.Error("switch lookup error", "switchid", switchID)

		return "", "", err
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

	switch netDevType {
	case "TAP":
		fallthrough
	case "VMNET":
		if switchInst.Type != "IF" {
			return res, errSwitchUplinkWrongType
		}
	case "NETGRAPH":
		if switchInst.Type != "NG" {
			return res, errSwitchUplinkWrongType
		}
	default:
		return res, errSwitchUnknownNicDevType
	}

	res = switchUUID.String()

	return res, nil
}

func (s *Switch) UnsetUplink() error {
	switch s.Type {
	case "IF":
		slog.Debug("unsetting IF bridge uplink", "id", s.ID)

		err := bridgeIfDeleteMember(s.Name, s.Uplink)
		if err != nil {
			return err
		}

		s.Uplink = ""

		err = s.Save()
		if err != nil {
			return err
		}

		return nil
	case "NG":
		slog.Debug("unsetting NG bridge uplink", "id", s.ID)

		err := bridgeNgRemoveUplink(s.Name, s.Uplink)
		if err != nil {
			return err
		}

		s.Uplink = ""

		err = s.Save()
		if err != nil {
			return err
		}

		return nil
	default:
		return errSwitchInvalidType
	}
}

func (s *Switch) SetUplink(uplink string) error {
	netDevs := util.GetHostInterfaces()

	if !util.ContainsStr(netDevs, uplink) {
		return errSwitchInvalidUplink
	}

	switch s.Type {
	case "IF":
		return setUplinkIf(uplink, s)
	case "NG":
		return setUplinkNG(uplink, s)
	default:
		return errSwitchInvalidType
	}
}

func (s *Switch) Save() error {
	db := getSwitchDB()

	res := db.Model(&s).
		Updates(map[string]interface{}{
			"name":        &s.Name,
			"description": &s.Description,
			"type":        &s.Type,
			"uplink":      &s.Uplink,
		},
		)

	if res.Error != nil {
		return errSwitchInternalDB
	}

	return nil
}
