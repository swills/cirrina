package main

import (
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/vm"
	"log"
)

func startVM(rs *requests.Request) {
	vmInst := vm.VM{ID: rs.VMID}
	vmInst.Start()
	requests.MarkReqSuccessful(rs)
}

func stopVM(rs *requests.Request) {
	log.Printf("stopping VM %v", rs.VMID)
	vmInst := vm.VM{ID: rs.VMID}
	vmInst.Stop()
	requests.MarkReqSuccessful(rs)
}

func deleteVM(rs *requests.Request) {
	vmInst := vm.VM{ID: rs.VMID}
	err := vmInst.Delete()
	if err != nil {
		log.Printf("failed to delete VM %v", rs.VMID)
		requests.MarkReqFailed(rs)
	}
	requests.MarkReqSuccessful(rs)
}
