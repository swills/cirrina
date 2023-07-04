package main

import (
	"cirrina/cirrina"
	"context"
	"fmt"
	"io"
	"log"
)

func addVM(namePtr *string, c cirrina.VMInfoClient, ctx context.Context, descrPtr *string, cpuPtr *uint32, memPtr *uint32) {
	if *namePtr == "" {
		log.Fatalf("Name not specified")
		return
	}
	res, err := c.AddVM(ctx, &cirrina.VMConfig{
		Name:        namePtr,
		Description: descrPtr,
		Cpu:         cpuPtr,
		Mem:         memPtr,
	})
	if err != nil {
		log.Fatalf("could not create VM: %v", err)
		return
	}
	fmt.Printf("Created VM %v\n", res.Value)
}

func DeleteVM(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	reqId, err := c.DeleteVM(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not delete VM: %v", err)
	}
	fmt.Printf("Deleted request created, reqid: %v\n", reqId.Value)
}

func stopVM(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	reqId, err := c.StopVM(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not stop VM: %v", err)
	}
	fmt.Printf("Stopping request created, reqid: %v\n", reqId.Value)
}

func startVM(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	reqId, err := c.StartVM(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not start VM: %v", err)
	}
	fmt.Printf("Started request created, reqid: %v\n", reqId.Value)
}

func getVM(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	res, err := c.GetVMConfig(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
	}
	fmt.Printf(
		"name: %v "+
			"desc: %v "+
			"cpus: %v "+
			"mem: %v "+
			"vncWait: %v "+
			"wire guest mem: %v "+
			"tablet mode: %v "+
			"store uefi vars: %v "+
			"use utc time: %v "+
			"use host bridge: %v "+
			"generate acpi tables: %v "+
			"yield on HLT: %v "+
			"exit on PAUSE: %v "+
			"destroy on power off: %v "+
			"ignore unknown msr: %v "+
			"Use network %v "+
			"vnc port: %v "+
			"mac address: %v "+
			"auto start: %v"+
			"\n",
		*res.Name,
		*res.Description,
		*res.Cpu,
		*res.Mem,
		*res.Vncwait,
		*res.Wireguestmem,
		*res.Tablet,
		*res.Storeuefi,
		*res.Utc,
		*res.Hostbridge,
		*res.Acpi,
		*res.Hlt,
		*res.Eop,
		*res.Dpo,
		*res.Ium,
		*res.Net,
		*res.Vncport,
		*res.Mac,
		*res.Autostart,
	)
}

func getVMs(c cirrina.VMInfoClient, ctx context.Context) {
	res, err := c.GetVMs(ctx, &cirrina.VMsQuery{})
	if err != nil {
		log.Fatalf("could not get VMs: %v", err)
		return
	}
	for {
		VM, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("GetVMs failed: %v", err)
		}
		fmt.Printf("VM: id: %v\n", VM.Value)
	}
}

func Reconfig(idPtr *string, err error, namePtr *string, descrPtr *string, cpuPtr *uint, memPtr *uint, autoStartPtr *bool, c cirrina.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	newConfig := cirrina.VMConfig{
		Id: *idPtr,
	}
	if isFlagPassed("name") {
		newConfig.Name = namePtr
	}
	if isFlagPassed("descr") {
		newConfig.Description = descrPtr
	}
	if isFlagPassed("cpus") {
		newCpu := uint32(*cpuPtr)
		if newCpu < 1 {
			newCpu = 1
		}
		if newCpu > 16 {
			newCpu = 16
		}
		newConfig.Cpu = &newCpu
	}
	if isFlagPassed("mem") {
		newMem := uint32(*memPtr)
		if newMem < 128 {
			newMem = 128
		}
		newConfig.Mem = &newMem
	}
	if isFlagPassed("autostart") {
		newConfig.Autostart = autoStartPtr
	}
	_, err = c.UpdateVM(ctx, &newConfig)
	if err != nil {
		log.Fatalf("could not update VM: %v", err)
		return
	}
	fmt.Printf("Success\n")
}

func getVMState(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	res, err := c.GetVMState(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not get state: %v", err)
		return
	}
	var vmstate string
	switch res.Status {
	case cirrina.VmStatus_STATUS_STOPPED:
		vmstate = "stopped"
	case cirrina.VmStatus_STATUS_STARTING:
		vmstate = "starting"
	case cirrina.VmStatus_STATUS_RUNNING:
		vmstate = "running"
	case cirrina.VmStatus_STATUS_STOPPING:
		vmstate = "stopping"
	}
	fmt.Printf("vm id: %v state: %v vnc port: %v\n", *idPtr, vmstate, res.VncPort)
}
