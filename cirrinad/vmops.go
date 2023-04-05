package main

import (
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/vm"
	"log"
)

func startVM(rs *requests.Request) {
	vmInst, err := vm.GetById(rs.VmId)
	if err != nil {
		log.Printf("error getting vm %v, %v", rs.VmId, err)
		return
	}
	log.Printf("startVM: %v", vmInst.Name)
	err = vmInst.Start()
	if err != nil {
		log.Printf("failed to start VM %v: %v", vmInst.ID, err)
		rs.Failed()
		return
	}
	rs.Succeeded()
}

func stopVM(rs *requests.Request) {
	vmInst, err := vm.GetById(rs.VmId)
	if err != nil {
		log.Printf("error getting vm %v, %v", rs.VmId, err)
		return
	}
	log.Printf("stopping VM %v", rs.VmId)
	err = vmInst.Stop()
	if err != nil {
		log.Printf("failed to stop VM %v: %v", vmInst.ID, err)
		rs.Failed()
		return
	}
	rs.Succeeded()
}

func deleteVM(rs *requests.Request) {
	vmInst, err := vm.GetById(rs.VmId)
	if err != nil {
		log.Printf("error getting vm %v, %v", rs.VmId, err)
		return
	}
	log.Printf("deleting VM %v", rs.VmId)
	defer vm.List.Mu.Unlock()
	vm.List.Mu.Lock()
	err = vmInst.Delete()
	if err != nil {
		log.Printf("failed to delete VM %v: %v", vmInst.ID, err)
		rs.Failed()
		return
	}
	rs.Succeeded()
	delete(vm.List.VmList, vmInst.ID)
}
