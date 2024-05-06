package vmnic

import (
	"errors"
	"fmt"
	"log/slog"
	"net"

	"gorm.io/gorm"

	"cirrina/cirrina"
	"cirrina/cirrinad/util"
)

type VMNic struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null;default:null"`
	Name        string `gorm:"uniqueIndex;not null;default:null"`
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
	if res.RowsAffected != 1 {
		return fmt.Errorf("incorrect number of rows affected, err: %w", res.Error)
	}

	if res.Error != nil {
		return res.Error
	}

	return nil
}

func GetByName(name string) (*VMNic, error) {
	if name == "" {
		return nil, errNicNotFound
	}

	var aNic *VMNic

	db := GetVMNicDB()

	res := db.Limit(1).Find(&aNic, "name = ?", name)
	if res.Error != nil {
		return nil, res.Error
	}

	if res.RowsAffected != 1 {
		return nil, errNicNotFound
	}

	return aNic, nil
}

func GetByID(nicID string) (*VMNic, error) {
	if nicID == "" {
		return nil, errNicNotFound
	}

	var vmNic *VMNic

	db := GetVMNicDB()

	res := db.Limit(1).Find(&vmNic, "id = ?", nicID)
	if res.Error != nil {
		return nil, res.Error
	}

	if res.RowsAffected != 1 {
		return nil, errNicNotFound
	}

	return vmNic, nil
}

func GetNics(vmConfigID uint) []VMNic {
	var vmNics []VMNic

	db := GetVMNicDB()
	db.Where("config_id = ?", vmConfigID).Find(&vmNics)

	return vmNics
}

func GetAll() []*VMNic {
	var result []*VMNic

	db := GetVMNicDB()
	db.Find(&result)

	return result
}

func (d *VMNic) Delete() error {
	db := GetVMNicDB()

	res := db.Limit(1).Unscoped().Delete(&d)
	if res.RowsAffected != 1 {
		slog.Error("error saving vmnic", "res", res)

		return errNicInternalDB
	}

	return nil
}

func (d *VMNic) SetSwitch(switchid string) error {
	d.SwitchID = switchid

	err := d.Save()
	if err != nil {
		slog.Error("error saving VM nic", "err", err)

		return err
	}

	return nil
}

func (d *VMNic) Save() error {
	db := GetVMNicDB()

	res := db.Model(&d).
		Updates(map[string]interface{}{
			"name":         &d.Name,
			"description":  &d.Description,
			"mac":          &d.Mac,
			"net_dev":      &d.NetDev,
			"net_type":     &d.NetType,
			"net_dev_type": &d.NetDevType,
			"switch_id":    &d.SwitchID,
			"rate_limit":   &d.RateLimit,
			"rate_in":      &d.RateIn,
			"rate_out":     &d.RateOut,
			"inst_bridge":  &d.InstBridge,
			"inst_epair":   &d.InstEpair,
			"config_id":    &d.ConfigID,
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

	isBroadcast, err := util.MacIsBroadcast(macAddress)
	if err != nil {
		return "", errInvalidMac
	}

	if isBroadcast {
		return "", errInvalidMacBroadcast
	}

	isMulticast, err := util.MacIsMulticast(macAddress)
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
func validateNic(vmNicInst *VMNic) error {
	if !util.ValidNicName(vmNicInst.Name) {
		return errInvalidNicName
	}

	if !nicTypeValid(vmNicInst.NetType) {
		return errInvalidNetType
	}

	if !nicDevTypeValid(vmNicInst.NetDevType) {
		return errInvalidNetDevType
	}

	if vmNicInst.RateLimit {
		if vmNicInst.RateIn <= 0 || vmNicInst.RateOut <= 0 {
			return errInvalidNetworkRateLimit
		}
	}

	if vmNicInst.Mac != "AUTO" {
		newMac, err := net.ParseMAC(vmNicInst.Mac)
		if err != nil {
			return errInvalidMac
		}

		if len(newMac.String()) != 17 {
			return errInvalidMac
		}
		// normalize MAC
		vmNicInst.Mac = newMac.String()
	}

	return nil
}

func nicExists(nicName string) (bool, error) {
	var err error

	_, err = GetByName(nicName)
	if err != nil {
		if !errors.Is(err, errNicNotFound) {
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
