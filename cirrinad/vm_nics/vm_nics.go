package vm_nics

import (
	"errors"
	"fmt"
	"log/slog"
	"net"

	"gorm.io/gorm"

	"cirrina/cirrina"
	"cirrina/cirrinad/util"
)

func GetByName(name string) (s *VmNic, err error) {
	db := GetVmNicDb()
	db.Limit(1).Find(&s, "name = ?", name)
	return s, nil
}

func GetById(id string) (v *VmNic, err error) {
	db := GetVmNicDb()
	db.Limit(1).Find(&v, "id = ?", id)
	return v, nil
}

func GetNics(vmConfigId uint) (vms []VmNic) {
	db := GetVmNicDb()
	db.Where("config_id = ?", vmConfigId).Find(&vms)
	return vms
}

func GetAll() []*VmNic {
	var result []*VmNic
	db := GetVmNicDb()
	db.Find(&result)
	return result
}

func Create(vmNicInst *VmNic) (newNicId string, err error) {
	if vmNicInst.Mac == "" {
		vmNicInst.Mac = "AUTO"
	}
	if vmNicInst.NetType == "" {
		vmNicInst.NetType = "VIRTIONET"
	}
	if vmNicInst.NetDevType == "" {
		vmNicInst.NetDevType = "TAP"
	}

	valid, err := validateNewNic(vmNicInst)
	if err != nil {
		slog.Error("error validating nic", "VmNic", vmNicInst, "err", err)
		return newNicId, err
	}
	if !valid {
		slog.Error("VmNic exists or not valid", "VmNic", vmNicInst.Name)
		return newNicId, errors.New("VmNic exists or not valid")
	}

	db := GetVmNicDb()
	res := db.Create(&vmNicInst)
	if res.RowsAffected != 1 {
		return newNicId, res.Error
	}
	return vmNicInst.ID, res.Error
}

func (d *VmNic) Delete() (err error) {
	db := GetVmNicDb()
	res := db.Limit(1).Unscoped().Delete(&d)
	if res.RowsAffected != 1 {
		errText := fmt.Sprintf("vmnic delete error, rows affected %v", res.RowsAffected)
		return errors.New(errText)
	}
	return nil
}

func (d *VmNic) SetSwitch(switchid string) error {
	d.SwitchId = switchid
	err := d.Save()
	if err != nil {
		slog.Error("error saving VM nic", "err", err)
		return err
	}

	return nil
}

func (d *VmNic) Save() error {
	db := GetVmNicDb()

	res := db.Model(&d).
		Updates(map[string]interface{}{
			"name":         &d.Name,
			"description":  &d.Description,
			"mac":          &d.Mac,
			"net_dev":      &d.NetDev,
			"net_type":     &d.NetType,
			"net_dev_type": &d.NetDevType,
			"switch_id":    &d.SwitchId,
			"rate_limit":   &d.RateLimit,
			"rate_in":      &d.RateIn,
			"rate_out":     &d.RateOut,
			"inst_bridge":  &d.InstBridge,
			"inst_epair":   &d.InstEpair,
			"config_id":    &d.ConfigID,
		},
		)

	if res.Error != nil {
		return errors.New("error updating vmnic")
	}

	return nil
}

type VmNic struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null;default:null"`
	Name        string `gorm:"uniqueIndex;not null;default:null"`
	Description string
	Mac         string `gorm:"default:AUTO"`
	NetDev      string
	NetType     string `gorm:"default:VIRTIONET;check:net_type IN ('VIRTIONET','E1000')"`
	NetDevType  string `gorm:"default:TAP;check:net_dev_type IN ('TAP','VMNET','NETGRAPH')"`
	SwitchId    string
	RateLimit   bool `gorm:"default:False;check:rate_limit IN(0,1)"`
	RateIn      uint64
	RateOut     uint64
	InstBridge  string
	InstEpair   string
	ConfigID    uint `gorm:"index;default:null"`
}

func ParseMac(macAddress string) (res string, err error) {
	if macAddress == "AUTO" {
		return macAddress, nil
	}
	if macAddress == "" {
		return "", errors.New("invalid MAC address")
	}
	isBroadcast, err := util.MacIsBroadcast(macAddress)
	if err != nil {
		return "", errors.New("invalid MAC address")
	}
	if isBroadcast {
		return "", errors.New("may not use broadcast MAC address")
	}
	isMulticast, err := util.MacIsMulticast(macAddress)
	if err != nil {
		return "", errors.New("invalid MAC address")
	}
	if isMulticast {
		return "", errors.New("may not use multicast MAC address")
	}
	var newMac net.HardwareAddr
	newMac, err = net.ParseMAC(macAddress)
	if err != nil {
		return "", err
	}
	return newMac.String(), nil
}

func ParseNetDevType(netDevType cirrina.NetDevType) (res string, err error) {
	switch netDevType {
	case cirrina.NetDevType_TAP:
		res = "TAP"
	case cirrina.NetDevType_VMNET:
		res = "VMNET"
	case cirrina.NetDevType_NETGRAPH:
		res = "NETGRAPH"
	default:
		err = errors.New("invalid net dev type name")
	}
	return res, err
}

func ParseNetType(netType cirrina.NetType) (res string, err error) {
	switch netType {
	case cirrina.NetType_VIRTIONET:
		res = "VIRTIONET"
	case cirrina.NetType_E1000:
		res = "E1000"
	default:
		err = errors.New("invalid net type name")
	}
	return res, err
}

// validateNewNic validate and normalize new nic
func validateNewNic(vmNicInst *VmNic) (bool, error) {
	if !util.ValidNicName(vmNicInst.Name) {
		return false, errors.New("invalid name")
	}
	existingVmNic, err := GetByName(vmNicInst.Name)
	if err != nil {
		return false, err
	}
	if existingVmNic.Name != "" {
		return true, nil
	}

	if vmNicInst.NetType != "VIRTIONET" && vmNicInst.NetType != "E1000" {
		return false, errors.New("bad net type")
	}

	if vmNicInst.NetDevType != "TAP" && vmNicInst.NetDevType != "VMNET" && vmNicInst.NetDevType != "NETGRAPH" {
		return false, errors.New("bad net dev type")
	}

	if vmNicInst.RateLimit {
		if vmNicInst.RateIn <= 0 || vmNicInst.RateOut <= 0 {
			return false, errors.New("bad network rate limit")
		}
	}
	if vmNicInst.Mac != "AUTO" {
		newMac, err := net.ParseMAC(vmNicInst.Mac)
		if err != nil {
			return false, errors.New("bad MAC address")
		}
		// normalize MAC
		vmNicInst.Mac = newMac.String()
	}

	return true, nil
}
