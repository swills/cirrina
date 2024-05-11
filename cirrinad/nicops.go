package main

import (
	"encoding/json"
	"errors"
	"log/slog"

	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vmnic"
)

func nicClone(request *requests.Request) {
	var err error

	var nicCloneReqData requests.NicCloneReqData

	var sourceNic *vmnic.VMNic

	// check request type
	if request.Type != "NICCLONE" {
		slog.Error("nic clone request called for wrong request type")
		request.Failed()

		return
	}

	// get request data
	err = json.Unmarshal([]byte(request.Data), &nicCloneReqData)
	if err != nil {
		slog.Error("failed unmarshalling request data: %w", err)
		request.Failed()

		return
	}

	// check source nic exists
	sourceNic, err = vmnic.GetByID(nicCloneReqData.NicID)
	if err != nil {
		slog.Error("nicClone error getting nic: %w", err)
		request.Failed()

		return
	}

	// check source nic is not busy
	if nicHasPendingReq(request.ID, sourceNic.ID) {
		slog.Error("failing request to clone NIC which has pending request: %w", errPendingReqExists)
		request.Failed()

		return
	}

	// check target nic name exists already
	existingVMNic, err := vmnic.GetByName(nicCloneReqData.NewNicName)
	if err != nil && !errors.Is(err, vmnic.ErrNicNotFound) {
		slog.Error("error getting name of new NIC: %w", err)
		request.Failed()

		return
	}

	if existingVMNic != nil && existingVMNic.Name == "" {
		slog.Error("clone nic already exists: %w", errNicExists)
		request.Failed()

		return
	}

	// check new nic name is valid
	if nicCloneReqData.NewNicName == "" || !util.ValidNicName(nicCloneReqData.NewNicName) {
		slog.Error("nic clone request failed validation", "err", errInvalidName)
		request.Failed()

		return
	}

	// actually clone nic
	newNic := *sourceNic
	newNic.ID = ""

	// check that new mac is not broadcast and is not multicast. do not need to check if it's parseable here
	// because both do that also. while here, normalize MAC
	newNic.Mac, err = vmnic.ParseMac(sourceNic.Mac)
	if err != nil {
		slog.Error("nic clone failed mac validation", "err", err)
		request.Failed()

		return
	}

	newNic.Name = nicCloneReqData.NewNicName

	// ensure cloned nic is not attached to VM
	newNic.ConfigID = 0

	err = vmnic.Create(&newNic)
	if err != nil {
		slog.Error("error saving cloned nic", "err", err)
		request.Failed()

		return
	}

	slog.Debug("cloned nic", "newVMNicID", newNic.ID)

	request.Succeeded()
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
