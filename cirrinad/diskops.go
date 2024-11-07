package main

import (
	"encoding/json"
	"log/slog"

	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/requests"
)

func diskWipe(request *requests.Request) {
	var err error

	var diskWipeReqData requests.DiskCloneReqData

	var targetDisk *disk.Disk

	// check request type
	if request.Type != requests.DISKWIPE {
		slog.Error("disk wipe request called for wrong request type")
		request.Failed()

		return
	}

	// get request data
	err = json.Unmarshal([]byte(request.Data), &diskWipeReqData)
	if err != nil {
		slog.Error("failed unmarshalling request data: %w", "err", err)
		request.Failed()

		return
	}

	// check source exists
	targetDisk, err = disk.GetByID(diskWipeReqData.DiskID)
	if err != nil {
		slog.Error("disk wipe error getting disk: %w", "err", err)
		request.Failed()

		return
	}

	// check source is not busy
	if diskHasPendingReq(request.ID, targetDisk.ID) {
		slog.Error("failing request to wipe disk which has pending request")
		request.Failed()

		return
	}

	var diskService disk.InfoServicer

	switch targetDisk.DevType {
	case "FILE":
		diskService = disk.NewFileInfoService(disk.FileInfoFetcherImpl)

	case "ZVOL":
		diskService = disk.NewZfsVolInfoService(disk.ZfsInfoFetcherImpl)

	default:
		slog.Error("diskWipe request with invalid dev type",
			"request", request,
			"disk", targetDisk.ID,
		)

		return
	}

	diskPath := targetDisk.GetPath()

	diskSizeNum, err := diskService.GetSize(diskPath)
	if err != nil {
		slog.Error("error getting disk size", "err", err)

		return
	}

	// delete the existing backing
	err = diskService.RemoveBacking(targetDisk)
	if err != nil {
		slog.Error("error removing disk", "err", err)
	}

	// now recreate the disk
	err = diskService.Create(targetDisk.GetPath(), diskSizeNum)
	if err != nil {
		slog.Error("error creating disk", "err", err)
	}

	slog.Debug("wiped disk", "ID", targetDisk.ID)

	request.Succeeded()
}

// nicHasPendingReq check if the disk has pending requests other than this one
func diskHasPendingReq(thisReqID string, diskID string) bool {
	pendingReqIDs := requests.PendingReqExists(diskID)
	for _, pendingReqID := range pendingReqIDs {
		if pendingReqID != thisReqID {
			return true
		}
	}

	return false
}
