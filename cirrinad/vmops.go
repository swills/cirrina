package main

import (
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/vm"
	"log"
)

func startVM(rs *requests.Request) {
	vmInst, err := vm.GetByID(rs.VMID)
	if err != nil {
		log.Printf("error getting vm %v, %v", rs.VMID, err)
	}
	log.Printf("startVM: %v", vmInst.Name)
	vmInst.Start()
	requests.MarkSuccessful(rs)
}

func stopVM(rs *requests.Request) {
	log.Printf("stopping VM %v", rs.VMID)
	vmInst, err := vm.GetByID(rs.VMID)
	if err != nil {
		log.Printf("error getting vm %v, %v", rs.VMID, err)
	}
	vmInst.Stop()
	requests.MarkSuccessful(rs)
}

func deleteVM(rs *requests.Request) {
	vmInst, err := vm.GetByID(rs.VMID)
	if err != nil {
		log.Printf("error getting vm %v, %v", rs.VMID, err)
	}
	err = vmInst.Delete()
	if err != nil {
		log.Printf("failed to delete VM %v", rs.VMID)
		requests.MarkFailed(rs)
	}
	requests.MarkSuccessful(rs)
}
