package main

import (
	"cirrina/cirrinad/requests"
	vm2 "cirrina/cirrinad/vm"
	"log"
)

func startVM(rs *requests.Request) {
	vm := vm2.VM{ID: rs.VMID}
	db := vm2.GetVMDB()
	db.Model(&vm2.VM{}).Preload("VMConfig").Limit(1).Find(&vm, &vm2.VM{ID: rs.VMID})
	vm2.DbSetVMStarting(rs.VMID)
	vm.Start()
	requests.MarkReqSuccessful(rs)
}

func stopVM(rs *requests.Request) {
	log.Printf("stopping VM %v", rs.VMID)
	vm := vm2.VM{ID: rs.VMID}
	vm2.DbSetVMStopping(vm.ID)
	vm.Stop()
	requests.MarkReqSuccessful(rs)
	vm2.DbSetVMStopped(rs.VMID)
}

func deleteVM(rs *requests.Request) {
	vm := vm2.VM{}
	db := vm2.GetVMDB()
	db.Model(&vm2.VM{}).Preload("VMConfig").Limit(1).Find(&vm, &vm2.VM{ID: rs.VMID})
	res := db.Delete(&vm.VMConfig)
	if res.RowsAffected != 1 {
		requests.MarkReqFailed(rs)
		return
	}
	res = db.Delete(&vm)
	if res.RowsAffected != 1 {
		requests.MarkReqFailed(rs)
		return
	}
	requests.MarkReqSuccessful(rs)
}
