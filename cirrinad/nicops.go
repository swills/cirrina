package main

import (
	"encoding/json"
	"log/slog"
	"net"
	"reflect"

	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vmnic"
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
	var nicInst *vmnic.VMNic
	nicInst, err = vmnic.GetByID(reqData.NicID)
	if err != nil {
		slog.Error("nicClone error getting nic", "nic", reqData.NicID, "err", err)
		rs.Failed()

		return
	}
	if nicHasPendingReq(rs.ID, nicInst.ID) {
		slog.Error("failing request to clone NIC which has pending request", "nic", nicInst.ID)
		rs.Failed()

		return
	}
	existingVMNic, err := vmnic.GetByName(reqData.NewNicName)
	if err != nil {
		slog.Error("error getting name of new NIC", "nic", reqData.NicID, "err", err)
		rs.Failed()

		return
	}
	if existingVMNic.Name != "" {
		slog.Error("cloned nic already exists", "nic", reqData.NicID, "err", err, "newName", reqData.NewNicName)
		rs.Failed()

		return
	}
	var newMac net.HardwareAddr

	// check that mac is not broadcast and is not multicast. do not need to check if it's parseable here
	// because both do that also
	if reqData.NewNicMac != "" && reqData.NewNicMac != "AUTO" {
		var isBroadcast bool
		isBroadcast, err = util.MacIsBroadcast(reqData.NewNicMac)
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
		var isMulticast bool
		isMulticast, err = util.MacIsMulticast(reqData.NewNicMac)
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
	}

	newNic := *nicInst
	newNic.Name = reqData.NewNicName
	// ensure cloned nic is not attached to VM
	newNic.ConfigID = 0
	if reqData.NewNicMac != "" {
		newNic.Mac = newMac.String()
	}

	newVMNicID, err := vmnic.Create(&newNic)
	if err != nil {
		slog.Error("error saving cloned nic", "err", err)
		rs.Failed()

		return
	}
	slog.Debug("cloned nic", "newVMNicID", newVMNicID)
	rs.Succeeded()
}

// nicHasPendingReq check if the nic has pending requests other than this one
func nicHasPendingReq(thisReqID string, nicID string) bool {
	pendingReqIDs := requests.PendingReqExists(nicID)
	for _, pendingReqID := range pendingReqIDs {
		if pendingReqID != thisReqID {
			return true
		}
	}

	return false
}
