package vmswitch

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"cirrina/cirrina"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vmnic"
)

type Switch struct {
	ID          string `gorm:"uniqueIndex;not null;default:null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Name        string         `gorm:"uniqueIndex;not null;default:null"`
	Description string
	Type        string `gorm:"default:IF;check:type IN ('IF','NG')"`
	Uplink      string
}

var vmnicGetAllFunc = vmnic.GetAll

func (s *Switch) switchTypeValid() bool {
	switch s.Type {
	case "IF":
		return true
	case "NG":
		return true
	default:
		return false
	}
}

func (s *Switch) switchCheckUplink() error {
	netDevs := util.GetHostInterfaces()

	if !util.ContainsStr(netDevs, s.Uplink) {
		return ErrSwitchInvalidUplink
	}

	switch s.Type {
	case "IF":
		if s.Uplink != "" {
			alreadyUsed, err := memberUsedByIfSwitch(s.Uplink)
			if err != nil {
				return errSwitchInternalChecking
			}

			if alreadyUsed {
				return ErrSwitchUplinkInUse
			}
		}
	case "NG":
		if s.Uplink != "" {
			alreadyUsed, err := memberUsedByNgSwitch(s.Uplink)
			if err != nil {
				return errSwitchInternalChecking
			}

			if alreadyUsed {
				return ErrSwitchUplinkInUse
			}
		}
	default:
		slog.Error("bad switch type", "switchType", s.Type)

		return ErrSwitchInvalidType
	}

	return nil
}

func (s *Switch) validate() error {
	if !s.switchNameValid() {
		return ErrSwitchInvalidName
	}

	err := s.switchCheckUplink()
	if err != nil {
		return err
	}

	// default case unreachable because we checked the type above, unless the two are out of sync
	switch s.Type {
	case "IF":
		return s.validateIfSwitch()
	case "NG":
		return s.validateNgSwitch()
	default:
		return ErrSwitchInvalidType
	}
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

func (s *Switch) bringUpNewSwitch() error {
	if s == nil || s.ID == "" {
		return errSwitchInvalidID
	}

	switch s.Type {
	case "IF":
		slog.Debug("creating if switch", "name", s.Name)

		err := s.buildIfSwitch()
		if err != nil {
			slog.Error("error creating if switch", "err", err)
			// already created in db, so ignore system state and proceed on...
			return err
		}
	case "NG":
		slog.Debug("creating ng switch", "name", s.Name)

		err := s.buildNgSwitch()
		if err != nil {
			slog.Error("error creating ng switch", "err", err)
			// already created in db, so ignore system state and proceed on...
			return nil
		}
	default:
		slog.Error("unknown switch type bringing up new switch")

		return ErrSwitchInvalidType
	}

	return nil
}

func (s *Switch) switchNameValid() bool {
	if s.Name == "" {
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

	validChars := util.CheckInRange(s.Name, myRT)
	if !validChars {
		return false
	}

	switch s.Type {
	case "IF":
		if !strings.HasPrefix(s.Name, "bridge") {
			slog.Error("invalid name", "name", s.Name)

			return false
		}

		switchNumStr := strings.TrimPrefix(s.Name, "bridge")

		switchNum, err := strconv.ParseInt(switchNumStr, 10, 32)
		if err != nil {
			slog.Error("invalid switch name", "name", s.Name)

			return false
		}

		switchNumFormattedString := strconv.FormatInt(switchNum, 10)
		// Check for silly things like "0123"
		if switchNumStr != switchNumFormattedString {
			slog.Error("invalid name", "name", s.Name)

			return false
		}
	case "NG":
		if !strings.HasPrefix(s.Name, "bnet") {
			slog.Error("invalid switch name", "name", s.Name)

			return false
		}

		switchNumStr := strings.TrimPrefix(s.Name, "bnet")

		switchNum, err := strconv.ParseInt(switchNumStr, 10, 32)
		if err != nil {
			slog.Error("invalid switch name", "name", s.Name)

			return false
		}

		switchNumFormattedString := strconv.FormatInt(switchNum, 10)
		// Check for silly things like "0123"
		if switchNumStr != switchNumFormattedString {
			slog.Error("invalid name", "name", s.Name)

			return false
		}
	default:
		return false
	}

	return true
}

func Create(switchInst *Switch) error {
	switchAlreadyExists, err := switchExists(switchInst.Name)
	if err != nil {
		slog.Error("error checking db for switch", "name", switchInst.Name, "err", err)

		return err
	}

	if switchAlreadyExists {
		slog.Error("switch exists", "switch", switchInst.Name)

		return ErrSwitchExists
	}

	err = switchInst.validate()
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

func CreateSwitches() error {
	allSwitches := GetAll()

	for num, aSwitch := range allSwitches {
		slog.Debug("creating switches", "num", num, "switch", aSwitch.Name)

		switch aSwitch.Type {
		case "IF":
			slog.Debug("creating if switch", "name", aSwitch.Name)

			err := aSwitch.buildIfSwitch()
			if err != nil {
				slog.Error("error creating if switch", "err", err)

				return fmt.Errorf("error creating if switch: %w", err)
			}
		case "NG":
			slog.Debug("creating ng switch", "name", aSwitch.Name)

			err := aSwitch.buildNgSwitch()
			if err != nil {
				slog.Error("error creating ng switch",
					"name", aSwitch.Name,
					"err", err,
				)

				return fmt.Errorf("error creating ng switch: %w", err)
			}
		default:
			slog.Debug("unknown switch type", "name", aSwitch.Name, "type", aSwitch.Type)

			return ErrSwitchInvalidType
		}
	}

	return nil
}

func (s *Switch) Delete() error {
	switchDB := getSwitchDB()

	if s.inUse() {
		return ErrSwitchInUse
	}

	res := switchDB.Limit(1).Unscoped().Delete(&s)
	if res.RowsAffected != 1 {
		slog.Error("error saving switch", "res", res)

		return errSwitchInternalDB
	}

	err := s.destroySwitch()
	if err != nil {
		return fmt.Errorf("error deleting switch: %w", err)
	}

	return nil
}

func (s *Switch) inUse() bool {
	vmNics := vmnicGetAllFunc()
	for _, vmNic := range vmNics {
		if vmNic.SwitchID == s.ID {
			return true
		}
	}

	return false
}

func DestroySwitches() error {
	allSwitches := GetAll()

	exitingIfSwitches, err := getAllIfSwitches()
	if err != nil {
		slog.Error("error getting all if switches")

		return fmt.Errorf("error getting all if switches: %w", err)
	}

	exitingNgSwitches, err := getAllNgSwitches()
	if err != nil {
		slog.Error("error getting all ng switches")

		return fmt.Errorf("error getting all ng switches: %w", err)
	}

	for _, aSwitch := range allSwitches {
		switch aSwitch.Type {
		case "IF":
			if util.ContainsStr(exitingIfSwitches, aSwitch.Name) {
				slog.Debug("destroying if switch", "name", aSwitch.Name)

				err = destroyIfSwitch(aSwitch.Name, true)
				if err != nil {
					slog.Error("error destroying if switch", "err", err)

					return fmt.Errorf("error destroying if switch: %w", err)
				}
			}
		case "NG":
			if util.ContainsStr(exitingNgSwitches, aSwitch.Name) {
				slog.Debug("destroying ng switch", "name", aSwitch.Name)

				err = destroyNgSwitch(aSwitch.Name)
				if err != nil {
					slog.Error("error destroying ng switch", "err", err)

					return fmt.Errorf("error destroying ng switch: %w", err)
				}
			}
		default:
			slog.Debug("unknown switch type", "name", aSwitch.Name, "type", aSwitch.Type)

			return ErrSwitchInvalidType
		}
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
		return res, ErrSwitchInvalidName
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
		slog.Debug("unsetting IF switch uplink", "id", s.ID)

		err := switchIfDeleteMember(s.Name, s.Uplink)
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
		slog.Debug("unsetting NG switch uplink", "id", s.ID)

		err := switchNgRemoveUplink(s.Name, s.Uplink)
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
		return ErrSwitchInvalidType
	}
}

func (s *Switch) SetUplink(uplink string) error {
	switch s.Type {
	case "IF":
		return s.setUplinkIf(uplink)
	case "NG":
		return s.setUplinkNG(uplink)
	default:
		return ErrSwitchInvalidType
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

// CheckAll verifies that the uplink for the switch exists in the host
func CheckAll() {
	var ifUplinks []string

	var ngUplinks []string
	// validate every switch's uplink interface exist, check for duplicates
	allBridges := GetAll()
	for _, bridge := range allBridges {
		if bridge.Uplink == "" {
			continue
		}

		exists := util.CheckInterfaceExists(bridge.Uplink)
		if !exists {
			slog.Error("bridge uplink does not exist, will be ignored", "bridge", bridge.Name, "uplink", bridge.Uplink)

			continue
		}

		switch bridge.Type {
		case "IF":
			if util.ContainsStr(ifUplinks, bridge.Uplink) {
				slog.Error("uplink used twice", "bridge", bridge.Name, "uplink", bridge.Uplink)
			} else {
				ifUplinks = append(ifUplinks, bridge.Uplink)
			}
		case "NG":
			if util.ContainsStr(ngUplinks, bridge.Uplink) {
				slog.Error("uplink used twice", "bridge", bridge.Name, "uplink", bridge.Uplink)
			} else {
				ngUplinks = append(ngUplinks, bridge.Uplink)
			}
		default:
			slog.Error("unknown switch type checking uplinks", "bridge", bridge.Name, "type", bridge.Type)
		}
	}
}

// destroySwitch destroy a switch which is confirmed not in use by caller
func (s *Switch) destroySwitch() error {
	var err error

	switch s.Type {
	case "IF":
		err = destroyIfSwitch(s.Name, true)
		if err != nil {
			return fmt.Errorf("error destroying switch: %w", err)
		}

		return nil
	case "NG":
		err = destroyNgSwitch(s.Name)
		if err != nil {
			slog.Error("switch removal failure")

			return fmt.Errorf("error destroying switch: %w", err)
		}

		return nil
	default:
		return ErrSwitchInvalidType
	}
}

func (s *Switch) nicTypeMatch(vmNic *vmnic.VMNic) bool {
	switch s.Type {
	case "IF":
		if vmNic.NetDevType != "TAP" && vmNic.NetDevType != "VMNET" {
			slog.Error("switch/nic type mismatch",
				"switch.ID", s.ID,
				"switch.Type", s.Type,
				"nic.Name", vmNic.Name,
				"nic.ID", vmNic.ID,
				"nic.SwitchID", vmNic.SwitchID,
			)

			return false
		}
	case "NG":
		if vmNic.NetDevType != "NETGRAPH" {
			slog.Error("switch/nic type mismatch",
				"switch.ID", s.ID,
				"switch.Type", s.Type,
				"nic.Name", vmNic.Name,
				"nic.ID", vmNic.ID,
				"nic.SwitchID", vmNic.SwitchID,
			)

			return false
		}
	default:
		return false
	}

	return true
}

func (s *Switch) ConnectNic(vmNic *vmnic.VMNic) error {
	if !s.nicTypeMatch(vmNic) {
		return errSwitchUplinkWrongType
	}

	switch s.Type {
	case "IF":
		err := s.connectIfNic(vmNic)
		if err != nil {
			slog.Error("error connecting nic", "err", err)

			return fmt.Errorf("error connecting nic: %w", err)
		}
	case "NG":
		// nothing to do
		return nil
	default: // unreachable
		slog.Debug("unknown net type, unable to connect") // unreachable

		return ErrSwitchInvalidType // unreachable
	}

	return nil
}

func (s *Switch) DisconnectNic(vmNic *vmnic.VMNic) error {
	if !s.nicTypeMatch(vmNic) {
		return errSwitchUplinkWrongType
	}

	switch s.Type {
	case "IF":
		err := s.disconnectIfNic(vmNic)
		if err != nil {
			slog.Error("error connecting nic", "err", err)

			return fmt.Errorf("error connecting nic: %w", err)
		}
	case "NG":
		// nothing to do
	default: // unreachable
		// no error return so that we try to disconnect other bits just in case
		slog.Debug("unknown net type, unable to disconnect") // unreachable
	}

	return nil
}

func MapSwitchTypeTypeToDBString(switchType cirrina.SwitchType) (string, error) {
	switch switchType {
	case cirrina.SwitchType_IF:
		return "IF", nil
	case cirrina.SwitchType_NG:
		return "NG", nil
	default:
		return "", ErrSwitchInvalidType
	}
}

func MapSwitchTypeDBStringToType(switchType string) (*cirrina.SwitchType, error) {
	SwitchTypeIf := cirrina.SwitchType_IF
	SwitchTypeNg := cirrina.SwitchType_NG

	switch switchType {
	case "IF":
		return &SwitchTypeIf, nil
	case "NG":
		return &SwitchTypeNg, nil
	default:
		return nil, ErrSwitchInvalidType
	}
}
