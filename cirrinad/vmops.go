package main

import (
	"encoding/json"
	"log/slog"
	"reflect"

	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/vm"
)

func startVM(request *requests.Request) {
	var err error

	var reqData requests.VMReqData

	err = json.Unmarshal([]byte(request.Data), &reqData)
	if err != nil {
		slog.Error("failed unmarshalling request data",
			"rsData", request.Data, "reqType", reflect.TypeOf(reqData), "err", err)

		return
	}

	var vmInst *vm.VM

	vmInst, err = vm.GetByID(reqData.VMID)
	if err != nil {
		slog.Error("startVM error getting vm", "vm", reqData.VMID, "err", err)

		return
	}

	pendingReqIDs := requests.PendingReqExists(reqData.VMID)
	for _, pendingReqID := range pendingReqIDs {
		if pendingReqID != request.ID {
			slog.Error("failing request to start VM which has pending request", "vm", vmInst.ID)
			request.Failed()

			return
		}
	}

	err = vmInst.Start()
	if err != nil {
		slog.Error("failed to start VM", "vm", vmInst.ID, "err", err)
		request.Failed()

		return
	}

	request.Succeeded()
}

func stopVM(request *requests.Request) {
	var err error

	var reqData requests.VMReqData

	err = json.Unmarshal([]byte(request.Data), &reqData)
	if err != nil {
		slog.Error("failed unmarshalling request data",
			"rsData", request.Data, "reqType", reflect.TypeOf(reqData), "err", err)

		return
	}

	var vmInst *vm.VM

	vmInst, err = vm.GetByID(reqData.VMID)
	if err != nil {
		slog.Error("stopVM error getting vm", "vm", reqData.VMID, "err", err)

		return
	}

	slog.Debug("stopping VM", "vm", reqData.VMID)

	pendingReqIDs := requests.PendingReqExists(reqData.VMID)
	for _, pendingReqID := range pendingReqIDs {
		if pendingReqID != request.ID {
			slog.Error("failing request to stop VM which has pending request", "vm", vmInst.ID)
			request.Failed()

			return
		}
	}

	err = vmInst.Stop()
	if err != nil {
		slog.Error("failed to stop VM", "vm", vmInst.ID, "err", err)
		request.Failed()

		return
	}

	request.Succeeded()
}

func deleteVM(request *requests.Request) {
	var err error

	var reqData requests.VMReqData

	err = json.Unmarshal([]byte(request.Data), &reqData)
	if err != nil {
		slog.Error("failed unmarshalling request data",
			"rsData", request.Data, "reqType", reflect.TypeOf(reqData), "err", err)

		return
	}

	var vmInst *vm.VM

	vmInst, err = vm.GetByID(reqData.VMID)
	if err != nil {
		slog.Error("deleteVM error getting vm", "vm", reqData.VMID, "err", err)

		return
	}

	pendingReqIDs := requests.PendingReqExists(reqData.VMID)
	for _, pendingReqID := range pendingReqIDs {
		if pendingReqID != request.ID {
			slog.Error("failing request to delete VM which has pending request", "vm", vmInst.ID)
			request.Failed()

			return
		}
	}

	slog.Debug("deleting VM", "id", reqData.VMID)
	defer vm.List.Mu.Unlock()
	vm.List.Mu.Lock()

	err = vmInst.Delete()
	if err != nil {
		slog.Error("failed to delete VM", "vm", vmInst.ID, "err", err)
		request.Failed()

		return
	}

	request.Succeeded()
	delete(vm.List.VMList, vmInst.ID)
}
