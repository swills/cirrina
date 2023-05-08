package vm_nics

import (
	"errors"
	"fmt"
	"golang.org/x/exp/slog"
	"net"
	"strings"
)

func GetByName(name string) (s *VmNic, err error) {
	db := getVmNicDb()
	db.Limit(1).Find(&s, "name = ?", name)
	return s, nil
}

func GetById(id string) (v *VmNic, err error) {
	db := getVmNicDb()
	db.Limit(1).Find(&v, "id = ?", id)
	return v, nil
}

func GetAll() []*VmNic {
	var result []*VmNic
	db := getVmNicDb()
	db.Find(&result)
	return result
}

func Create(VmNicInst *VmNic) (newNicId string, err error) {
	if strings.Contains(VmNicInst.Name, "/") {
		return newNicId, errors.New("illegal character in VmNic name")
	}
	existingVmNic, err := GetByName(VmNicInst.Name)
	if err != nil {
		slog.Error("error checking db for VmNic", "name", VmNicInst.Name, "err", err)
		return newNicId, err
	}
	if existingVmNic.Name != "" {
		slog.Error("VmNic exists", "VmNic", VmNicInst.Name)
		return newNicId, errors.New("VmNic exists")
	}

	if VmNicInst.Mac == "" {
		VmNicInst.Mac = "AUTO"
	}

	if VmNicInst.Mac != "AUTO" {
		newMac, err := net.ParseMAC(VmNicInst.Mac)
		if err != nil {
			return newNicId, errors.New("bad MAC address")
		}
		VmNicInst.Mac = string(newMac)
	}

	// TODO -- validate switch

	if VmNicInst.NetType == "" {
		VmNicInst.NetType = "VIRTIONET"
	}

	if VmNicInst.NetType != "VIRTIONET" && VmNicInst.NetType != "E1000" {
		return newNicId, errors.New("bad net type")
	}

	if VmNicInst.NetDevType == "" {
		VmNicInst.NetDevType = "TAP"
	}

	if VmNicInst.NetDevType != "TAP" && VmNicInst.NetDevType != "VMNET" && VmNicInst.NetDevType != "NETGRAPH" {
		return newNicId, errors.New("bad net dev type")
	}

	db := getVmNicDb()
	res := db.Create(&VmNicInst)
	if res.RowsAffected != 1 {
		return newNicId, res.Error
	}
	return VmNicInst.ID, res.Error
}

func Delete(id string) (err error) {

	// TODO check that vmnic is not in use

	if id == "" {
		return errors.New("unable to delete, vmnic id empty")
	}
	db := getVmNicDb()
	dNic, err := GetById(id)
	if err != nil {
		errorText := fmt.Sprintf("vmnic %v not found", id)
		return errors.New(errorText)
	}
	res := db.Limit(1).Delete(&dNic)
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
	db := getVmNicDb()

	res := db.Model(&d).
		Updates(map[string]interface{}{
			"name":         &d.Name,
			"description":  &d.Description,
			"mac":          &d.Mac,
			"net_dev":      &d.NetDev,
			"net_type":     &d.NetType,
			"net_dev_type": &d.NetDevType,
			"switch_id":    &d.SwitchId,
		},
		)

	if res.Error != nil {
		return errors.New("error updating vmnic")
	}

	return nil
}
