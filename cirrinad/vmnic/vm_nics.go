package vmnic

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"cirrina/cirrina"
	"cirrina/cirrinad/util"
)

type VMNic struct {
	ID          string `gorm:"uniqueIndex;not null;default:null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Name        string         `gorm:"uniqueIndex;not null;default:null"`
	Description string
	Mac         string `gorm:"default:AUTO"`
	NetDev      string
	NetType     string `gorm:"default:VIRTIONET;check:net_type IN ('VIRTIONET','E1000')"`
	NetDevType  string `gorm:"default:TAP;check:net_dev_type IN ('TAP','VMNET','NETGRAPH')"`
	SwitchID    string
	RateLimit   bool `gorm:"default:False;check:rate_limit IN(0,1)"`
	RateIn      uint64
	RateOut     uint64
	InstBridge  string
	InstEpair   string
	ConfigID    uint `gorm:"index;default:null"`
}

var MacIsBroadcastFunc = util.MacIsBroadcast
var MacIsMulticastFunc = util.MacIsBroadcast

func Create(vmNicInst *VMNic) error {
	if vmNicInst.Mac == "" {
		vmNicInst.Mac = "AUTO"
	}

	if vmNicInst.NetType == "" {
		vmNicInst.NetType = "VIRTIONET"
	}

	if vmNicInst.NetDevType == "" {
		vmNicInst.NetDevType = "TAP"
	}

	err := validateNic(vmNicInst)
	if err != nil {
		slog.Error("error validating nic", "VMNic", vmNicInst, "err", err)

		return err
	}

	nicAlreadyExists, err := nicExists(vmNicInst.Name)
	if err != nil {
		slog.Error("error checking db for nic", "name", vmNicInst.Name, "err", err)

		return err
	}

	if nicAlreadyExists {
		slog.Error("nic exists in DB", "nic", vmNicInst.Name)

		return errNicExists
	}

	db := GetVMNicDB()

	res := db.Create(&vmNicInst)
	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected != 1 {
		return fmt.Errorf("incorrect number of rows affected, err: %w", res.Error)
	}

	return nil
}

func GetByName(name string) (*VMNic, error) {
	if name == "" {
		return nil, ErrNicNotFound
	}

	var aNic *VMNic

	db := GetVMNicDB()

	res := db.Limit(1).Find(&aNic, "name = ?", name)
	if res.Error != nil {
		return nil, res.Error
	}

	if res.RowsAffected != 1 {
		return nil, ErrNicNotFound
	}

	return aNic, nil
}

func GetByID(nicID string) (*VMNic, error) {
	if nicID == "" {
		return nil, ErrNicNotFound
	}

	var vmNic *VMNic

	db := GetVMNicDB()

	res := db.Limit(1).Find(&vmNic, "id = ?", nicID)
	if res.Error != nil {
		return nil, res.Error
	}

	if res.RowsAffected != 1 {
		return nil, ErrNicNotFound
	}

	return vmNic, nil
}

func GetNics(vmConfigID uint) ([]VMNic, error) {
	var vmNics []VMNic

	db := GetVMNicDB()

	res := db.Where("config_id = ?", vmConfigID).Find(&vmNics)
	if res.Error != nil {
		return nil, res.Error
	}

	return vmNics, nil
}

func GetAll() []*VMNic {
	var result []*VMNic

	db := GetVMNicDB()
	db.Find(&result)

	return result
}

func (v *VMNic) Delete() error {
	nicDB := GetVMNicDB()

	_, err := uuid.Parse(v.ID)
	if err != nil {
		return ErrInvalidNic
	}

	res := nicDB.Limit(1).Unscoped().Delete(&v)
	if res.RowsAffected != 1 {
		slog.Error("error saving vmnic", "res", res)

		return errNicInternalDB
	}

	return nil
}

func (v *VMNic) SetSwitch(switchID string) error {
	v.SwitchID = switchID

	err := v.Save()
	if err != nil {
		slog.Error("error saving VM nic", "err", err)

		return err
	}

	return nil
}

func (v *VMNic) Save() error {
	db := GetVMNicDB()

	res := db.Model(&v).
		Updates(map[string]interface{}{
			"name":         &v.Name,
			"description":  &v.Description,
			"mac":          &v.Mac,
			"net_dev":      &v.NetDev,
			"net_type":     &v.NetType,
			"net_dev_type": &v.NetDevType,
			"switch_id":    &v.SwitchID,
			"rate_limit":   &v.RateLimit,
			"rate_in":      &v.RateIn,
			"rate_out":     &v.RateOut,
			"inst_bridge":  &v.InstBridge,
			"inst_epair":   &v.InstEpair,
			"config_id":    &v.ConfigID,
		},
		)

	if res.Error != nil {
		slog.Error("error updating nic", "res", res)

		return errNicInternalDB
	}

	return nil
}

func ParseMac(macAddress string) (string, error) {
	if macAddress == "AUTO" {
		return macAddress, nil
	}

	if macAddress == "" {
		return "", errInvalidMac
	}

	isBroadcast, err := MacIsBroadcastFunc(macAddress)
	if err != nil {
		return "", errInvalidMac
	}

	if isBroadcast {
		return "", errInvalidMacBroadcast
	}

	isMulticast, err := MacIsMulticastFunc(macAddress)
	if err != nil {
		return "", errInvalidMac
	}

	if isMulticast {
		return "", errInvalidMacMulticast
	}

	var newMac net.HardwareAddr

	newMac, err = net.ParseMAC(macAddress)
	if err != nil {
		return "", errInvalidMac
	}

	// ensure we have an ethernet MAC address, not some other type, see net.ParseMac docs
	if len(newMac.String()) != 17 {
		return "", errInvalidMac
	}

	return newMac.String(), nil
}

func ParseNetDevType(netDevType cirrina.NetDevType) (string, error) {
	var res string

	var err error

	switch netDevType {
	case cirrina.NetDevType_TAP:
		res = "TAP"
	case cirrina.NetDevType_VMNET:
		res = "VMNET"
	case cirrina.NetDevType_NETGRAPH:
		res = "NETGRAPH"
	default:
		err = errInvalidNetDevType
	}

	return res, err
}

func ParseNetType(netType cirrina.NetType) (string, error) {
	var err error

	var res string

	switch netType {
	case cirrina.NetType_VIRTIONET:
		res = "VIRTIONET"
	case cirrina.NetType_E1000:
		res = "E1000"
	default:
		err = errInvalidNetType
	}

	return res, err
}

// validateNic validate and normalize new nic
func validateNic(vmNic *VMNic) error {
	if !util.ValidNicName(vmNic.Name) {
		return ErrInvalidNicName
	}

	if !nicTypeValid(vmNic.NetType) {
		return errInvalidNetType
	}

	if !nicDevTypeValid(vmNic.NetDevType) {
		return errInvalidNetDevType
	}

	if vmNic.Mac != "AUTO" {
		newMac, err := net.ParseMAC(vmNic.Mac)
		if err != nil {
			return errInvalidMac
		}

		// ensure we have an ethernet MAC address, not some other type, see net.ParseMac docs
		if len(newMac.String()) != 17 {
			return errInvalidMac
		}
		// normalize MAC
		vmNic.Mac = newMac.String()
	}

	return nil
}

func nicExists(nicName string) (bool, error) {
	var err error

	_, err = GetByName(nicName)
	if err != nil {
		if !errors.Is(err, ErrNicNotFound) {
			return false, err
		}

		return false, nil
	}

	return true, nil
}

func nicDevTypeValid(nicDevType string) bool {
	switch nicDevType {
	case "TAP":
		return true
	case "VMNET":
		return true
	case "NETGRAPH":
		return true
	default:
		return false
	}
}

func nicTypeValid(nicType string) bool {
	switch nicType {
	case "VIRTIONET":
		return true
	case "E1000":
		return true
	default:
		return false
	}
}
