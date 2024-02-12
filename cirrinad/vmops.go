package main

import (
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/vm"
	"encoding/json"
	"log/slog"
	"reflect"
)

func startVM(rs *requests.Request) {
	var err error
	var reqData requests.VmReqData
	err = json.Unmarshal([]byte(rs.Data), &reqData)
	if err != nil {
		slog.Error("failed unmarshalling request data", "rsData", rs.Data, "reqType", reflect.TypeOf(reqData), "err", err)
		return
	}
	var vmInst *vm.VM
	vmInst, err = vm.GetById(reqData.VmId)
	if err != nil {
		slog.Error("startVM error getting vm", "vm", reqData.VmId, "err", err)
		return
	}
	pendingReqIds := requests.PendingReqExists(reqData.VmId)
	for _, pendingReqId := range pendingReqIds {
		if pendingReqId != rs.ID {
			slog.Error("failing request to start VM which has pending request", "vm", vmInst.ID)
			rs.Failed()
			return
		}
	}
	err = vmInst.Start()
	if err != nil {
		slog.Error("failed to start VM", "vm", vmInst.ID, "err", err)
		rs.Failed()
		return
	}
	rs.Succeeded()
}

func stopVM(rs *requests.Request) {
	var err error
	var reqData requests.VmReqData
	err = json.Unmarshal([]byte(rs.Data), &reqData)
	if err != nil {
		slog.Error("failed unmarshalling request data", "rsData", rs.Data, "reqType", reflect.TypeOf(reqData), "err", err)
		return
	}
	var vmInst *vm.VM
	vmInst, err = vm.GetById(reqData.VmId)
	if err != nil {
		slog.Error("stopVM error getting vm", "vm", reqData.VmId, "err", err)
		return
	}
	slog.Debug("stopping VM", "vm", reqData.VmId)
	pendingReqIds := requests.PendingReqExists(reqData.VmId)
	for _, pendingReqId := range pendingReqIds {
		if pendingReqId != rs.ID {
			slog.Error("failing request to stop VM which has pending request", "vm", vmInst.ID)
			rs.Failed()
			return
		}
	}
	err = vmInst.Stop()
	if err != nil {
		slog.Error("failed to stop VM", "vm", vmInst.ID, "err", err)
		rs.Failed()
		return
	}
	rs.Succeeded()
}

func deleteVM(rs *requests.Request) {
	var err error
	var reqData requests.VmReqData
	err = json.Unmarshal([]byte(rs.Data), &reqData)
	if err != nil {
		slog.Error("failed unmarshalling request data", "rsData", rs.Data, "reqType", reflect.TypeOf(reqData), "err", err)
		return
	}
	var vmInst *vm.VM
	vmInst, err = vm.GetById(reqData.VmId)
	if err != nil {
		slog.Error("deleteVM error getting vm", "vm", reqData.VmId, "err", err)
		return
	}
	pendingReqIds := requests.PendingReqExists(reqData.VmId)
	for _, pendingReqId := range pendingReqIds {
		if pendingReqId != rs.ID {
			slog.Error("failing request to delete VM which has pending request", "vm", vmInst.ID)
			rs.Failed()
			return
		}
	}
	slog.Debug("deleting VM", "id", reqData.VmId)
	defer vm.List.Mu.Unlock()
	vm.List.Mu.Lock()
	err = vmInst.Delete()
	if err != nil {
		slog.Error("failed to delete VM", "vm", vmInst.ID, "err", err)
		rs.Failed()
		return
	}
	rs.Succeeded()
	delete(vm.List.VmList, vmInst.ID)
}
