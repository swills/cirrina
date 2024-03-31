package main

import (
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vm_nics"
	"encoding/json"
	"log/slog"
	"net"
	"reflect"
)

func nicClone(rs *requests.Request) {
	var err error
	var reqData requests.NicCloneReqData
	err = json.Unmarshal([]byte(rs.Data), &reqData)
	if err != nil {
		slog.Error("failed unmarshalling request data", "rsData", rs.Data, "reqType", reflect.TypeOf(reqData), "err", err)
		rs.Failed()
		return
	}
	var nicInst *vm_nics.VmNic
	nicInst, err = vm_nics.GetById(reqData.NicId)
	if err != nil {
		slog.Error("deleteVM error getting nic", "nic", reqData.NicId, "err", err)
		rs.Failed()
		return
	}
	pendingReqIds := requests.PendingReqExists(nicInst.ID)
	for _, pendingReqId := range pendingReqIds {
		if pendingReqId != rs.ID {
			slog.Error("failing request to clone NIC which has pending request", "nic", nicInst.ID)
			rs.Failed()
			return
		}
	}
	existingVmNic, err := vm_nics.GetByName(reqData.NewNicName)
	if err != nil {
		slog.Error("error getting name of new NIC", "nic", reqData.NicId, "err", err)
		rs.Failed()
		return
	}
	if existingVmNic.Name != "" {
		slog.Error("cloned nic already exists", "nic", reqData.NicId, "err", err, "newName", reqData.NewNicName)
		rs.Failed()
		return
	}
	var newMac net.HardwareAddr
	if reqData.NewNicMac != "" && reqData.NewNicMac != "AUTO" {
		isBroadcast, err := util.MacIsBroadcast(reqData.NewNicMac)
		if err != nil {
			slog.Error("error checking new nic mac", "err", err)
			rs.Failed()
			return
		}
		if isBroadcast {
			slog.Error("new nic mac is broadcast", "newNicMac", reqData.NewNicMac)
			rs.Failed()
			return
		}
		isMulticast, err := util.MacIsMulticast(reqData.NewNicMac)
		if err != nil {
			slog.Error("error checking new nic mac", "err", err)
			rs.Failed()
			return
		}
		if isMulticast {
			slog.Error("new nic mac is multicast", "newNicMac", reqData.NewNicMac)
			rs.Failed()
			return
		}
		newMac, err = net.ParseMAC(reqData.NewNicMac)
		if err != nil {
			slog.Error("error parsing new nic mac", "err", err)
			rs.Failed()
			return
		}
	}

	var newNic = *nicInst
	newNic.Name = reqData.NewNicName
	// ensure cloned nic is not attached to VM
	newNic.ConfigID = 0
	if reqData.NewNicMac != "" {
		newNic.Mac = newMac.String()
	}

	newVmNicId, err := vm_nics.Create(&newNic)
	if err != nil {
		slog.Error("error saving cloned nic", "err", err)
		rs.Failed()
		return
	}
	slog.Debug("cloned nic", "newVmNicId", newVmNicId)
	rs.Succeeded()
}
