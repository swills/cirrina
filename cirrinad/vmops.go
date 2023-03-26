package main

import (
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/vm"
	"log"
)

func startVM(rs *requests.Request) {
	vmInst, err := vm.GetByID(rs.VmId)
	if err != nil {
		log.Printf("error getting vm %v, %v", rs.VmId, err)
	}
	log.Printf("startVM: %v", vmInst.Name)
	err = vmInst.Start()
	if err != nil {
		requests.MarkFailed(rs)
	}
	requests.MarkSuccessful(rs)
}

func stopVM(rs *requests.Request) {
	log.Printf("stopping VM %v", rs.VmId)
	vmInst, err := vm.GetByID(rs.VmId)
	if err != nil {
		log.Printf("error getting vm %v, %v", rs.VmId, err)
	}
	err = vmInst.Stop()
	if err != nil {
		requests.MarkFailed(rs)
	}
	requests.MarkSuccessful(rs)
}

func deleteVM(rs *requests.Request) {
	vmInst, err := vm.GetByID(rs.VmId)
	if err != nil {
		log.Printf("error getting vm %v, %v", rs.VmId, err)
	}
	err = vmInst.Delete()
	if err != nil {
		log.Printf("failed to delete VM %v", rs.VmId)
		requests.MarkFailed(rs)
	}
	requests.MarkSuccessful(rs)
}
