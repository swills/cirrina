package vm_nics

import (
	"errors"
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
			return newNicId, errors.New("Bad MAC address")
		}
		VmNicInst.Mac = string(newMac)
	}

	// TODO -- validate switch

	if VmNicInst.NetType == "" {
		VmNicInst.NetType = "VIRTIONET"
	}

	if VmNicInst.NetType != "VIRTIONET" && VmNicInst.NetType != "E1000" {
		return newNicId, errors.New("Bad Net Type")
	}

	if VmNicInst.NetDevType == "" {
		VmNicInst.NetDevType = "TAP"
	}

	if VmNicInst.NetDevType != "TAP" && VmNicInst.NetDevType != "VMNET" && VmNicInst.NetDevType != "NETGRAPH" {
		return newNicId, errors.New("Bad Net Dev Type")
	}

	db := getVmNicDb()
	res := db.Create(&VmNicInst)
	if res.RowsAffected != 1 {
		return newNicId, res.Error
	}
	return VmNicInst.ID, res.Error
}
