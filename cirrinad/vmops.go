package main

import (
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/vm"
	"golang.org/x/exp/slog"
)

func startVM(rs *requests.Request) {
	vmInst, err := vm.GetById(rs.VmId)
	if err != nil {
		slog.Error("startVM error getting vm", "vm", rs.VmId, "err", err)
		return
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
	vmInst, err := vm.GetById(rs.VmId)
	if err != nil {
		slog.Error("stopVM error getting vm", "vm", rs.VmId, "err", err)
		return
	}
	slog.Debug("stopping VM", "vm", rs.VmId)
	err = vmInst.Stop()
	if err != nil {
		slog.Error("failed to stop VM", "vm", vmInst.ID, "err", err)
		rs.Failed()
		return
	}
	rs.Succeeded()
}

func deleteVM(rs *requests.Request) {
	vmInst, err := vm.GetById(rs.VmId)
	if err != nil {
		slog.Error("deleteVM error getting vm", "vm", rs.VmId, "err", err)
		return
	}
	slog.Debug("deleting VM", "id", rs.VmId)
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
