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
	switch s.Type {
	case "IF":
		if s.Uplink != "" {
			alreadyUsed, err := memberUsedByIfSwitch(s.Uplink)
			if err != nil {
				return errSwitchInternalChecking
			}

			if alreadyUsed {
				return errSwitchUplinkInUse
			}
		}
	case "NG":
		if s.Uplink != "" {
			alreadyUsed, err := memberUsedByNgSwitch(s.Uplink)
			if err != nil {
				return errSwitchInternalChecking
			}

			if alreadyUsed {
				return errSwitchUplinkInUse
			}
		}
	default:
		slog.Error("bad switch type", "switchType", s.Type)

		return errSwitchInvalidType
	}

	return nil
}

func (s *Switch) validate() error {
	// switchNameValid also checks type, no need to check here
	if !s.switchNameValid() {
		return ErrSwitchInvalidName
	}

	if !s.switchTypeValid() {
		return errSwitchInvalidType
	}

	err := s.switchCheckUplink()
	if err != nil {
		return err
	}

	// default case unreachable
	switch s.Type {
	case "IF":
		return s.validateIfSwitch()
	case "NG":
		return s.validateNgSwitch()
	default:
		return errSwitchInvalidType
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

		return errSwitchInvalidType
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

		return errSwitchExists
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

func DestroySwitches() error {
	allSwitches := GetAll()

	exitingIfSwitches, err := GetAllIfSwitches()
	if err != nil {
		slog.Error("error getting all if switches")

		return fmt.Errorf("error getting all if switches: %w", err)
	}

	exitingNgSwitches, err := GetAllNgSwitches()
	if err != nil {
		slog.Error("error getting all ng switches")

		return fmt.Errorf("error getting all ng switches: %w", err)
	}

	for _, aSwitch := range allSwitches {
		switch aSwitch.Type {
		case "IF":
			if util.ContainsStr(exitingIfSwitches, aSwitch.Name) {
				slog.Debug("destroying if switch", "name", aSwitch.Name)

				err = DestroyIfSwitch(aSwitch.Name, true)
				if err != nil {
					slog.Error("error destroying if switch", "err", err)

					return fmt.Errorf("error destroying if switch: %w", err)
				}
			}
		case "NG":
			if util.ContainsStr(exitingNgSwitches, aSwitch.Name) {
				slog.Debug("destroying ng switch", "name", aSwitch.Name)

				err = DestroyNgSwitch(aSwitch.Name)
				if err != nil {
					slog.Error("error destroying ng switch", "err", err)

					return fmt.Errorf("error destroying ng switch: %w", err)
				}
			}
		default:
			slog.Debug("unknown switch type", "name", aSwitch.Name, "type", aSwitch.Type)

			return errSwitchInvalidType
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
		return s.setUplinkIf(uplink)
	case "NG":
		return s.setUplinkNG(uplink)
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

func CheckAll() {
	var ifUplinks []string

	var ngUplinks []string
	// validate every switch's uplink interface exist, check for duplicates
	allBridges := GetAll()
	for _, bridge := range allBridges {
		if bridge.Uplink == "" {
			continue
		}

		exists := CheckInterfaceExists(bridge.Uplink)
		if !exists {
			slog.Warn("bridge uplink does not exist, will be ignored", "bridge", bridge.Name, "uplink", bridge.Uplink)

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
