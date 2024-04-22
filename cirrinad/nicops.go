package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"

	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vmnic"
)

func nicClone(request *requests.Request) {
	var err error
	var reqData requests.NicCloneReqData
	var newMac net.HardwareAddr
	var nicInst *vmnic.VMNic

	nicInst, err = nicCloneRequestValidate(request.Data, reqData)
	if err != nil {
		slog.Error("nic clone request failed validation", "err", err)
		request.Failed()

		return
	}

	// check new nic name
	if nicInst.Name == "" || !util.ValidNicName(nicInst.Name) {
		slog.Error("nic clone request failed validation", "err", errInvalidName)
		request.Failed()

		return
	}

	// check that new mac is not broadcast and is not multicast. do not need to check if it's parseable here
	// because both do that also
	err = nicCloneValidateMac(reqData.NewNicMac)
	if err != nil {
		slog.Error("nic clone failed mac validation", "err", err)
		request.Failed()

		return
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
		request.Failed()

		return
	}
	slog.Debug("cloned nic", "newVMNicID", newVMNicID)

	request.Succeeded()
}

func nicCloneValidateMac(newMac string) error {
	var err error
	if newMac != "" && newMac != "AUTO" {
		var isBroadcast bool
		isBroadcast, err = util.MacIsBroadcast(newMac)
		if err != nil {
			return fmt.Errorf("error checking new nic mac: %w", err)
		}
		if isBroadcast {
			return fmt.Errorf("new nic mac is broadcast: %w", errNicMacIsBroadcast)
		}
		var isMulticast bool
		isMulticast, err = util.MacIsMulticast(newMac)
		if err != nil {
			return fmt.Errorf("error checking new nic mac: %w", err)
		}
		if isMulticast {
			return fmt.Errorf("new nic mac is broadcast: %w", errNicMacIsMulticast)
		}
	}

	return nil
}

func nicCloneRequestValidate(requestData string, reqData requests.NicCloneReqData) (*vmnic.VMNic, error) {
	var err error
	var nicInst *vmnic.VMNic

	err = json.Unmarshal([]byte(requestData), &reqData)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshalling request data: %w", err)
	}
	nicInst, err = vmnic.GetByID(reqData.NicID)
	if err != nil {
		return nil, fmt.Errorf("nicClone error getting nic: %w", err)
	}
	if nicHasPendingReq(reqData.NicID, nicInst.ID) {
		return nil, fmt.Errorf("failing request to clone NIC which has pending request: %w", errPendingReqExists)
	}
	existingVMNic, err := vmnic.GetByName(reqData.NewNicName)
	if err != nil {
		return nil, fmt.Errorf("error getting name of new NIC: %w", err)
	}
	if existingVMNic.Name != "" {
		return nil, fmt.Errorf("cloned nic already exists: %w", errNicExists)
	}

	return nicInst, nil
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
