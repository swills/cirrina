package main

import (
	pb "cirrina/cirrina"
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"time"
)

var (
	addr = flag.String("addr", "localhost:50051", "the address to connect to")
)

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(fflag *flag.Flag) {
		if fflag.Name == name {
			found = true
		}
	})
	return found
}

func addVM(namePtr *string, c pb.VMInfoClient, ctx context.Context, descrPtr *string, cpuPtr *uint32, memPtr *uint32) {
	if *namePtr == "" {
		log.Fatalf("Name not specified")
		return
	}
	res, err := c.AddVM(ctx, &pb.VMConfig{
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

func addISO(namePtr *string, c pb.VMInfoClient, ctx context.Context, descrPtr *string, pathPtr *string) {
	if *namePtr == "" {
		log.Fatalf("Name not specified")
		return
	}
	res, err := c.AddISO(ctx, &pb.ISOInfo{
		Name:        namePtr,
		Description: descrPtr,
		Path:        pathPtr,
	})
	if err != nil {
		log.Fatalf("could not create ISO: %v", err)
		return
	}
	fmt.Printf("Created ISO %v\n", res.Value)
}

func addVmNic(name *string, c pb.VMInfoClient, ctx context.Context, descrptr *string, nettypeptr *string, netdevtypeptr *string, macPtr *string) {
	var thisVmNic pb.VmNicInfo
	var thisNetType pb.NetType
	var thisNetDevType pb.NetDevType

	thisVmNic.Name = name
	thisVmNic.Description = descrptr
	thisVmNic.Mac = macPtr

	if *nettypeptr == "VIRTIONET" {
		thisNetType = pb.NetType_VIRTIONET
	} else if *nettypeptr == "E1000" {
		thisNetType = pb.NetType_E1000
	} else {
		log.Fatalf("Net type must be either \"VIRTIONET\" or \"E1000\"")
		return
	}
	if *netdevtypeptr == "TAP" {
		thisNetDevType = pb.NetDevType_TAP
	} else if *nettypeptr == "VMNET" {
		thisNetDevType = pb.NetDevType_VMNET
	} else if *nettypeptr == "NETGRAPH" {
		thisNetDevType = pb.NetDevType_NETGRAPH
	} else {
		log.Fatalf("Net dev type must be either \"TAP\" or \"VMNET\" or \"NETGRAPH\"")
		return
	}

	thisVmNic.Nettype = &thisNetType
	thisVmNic.Netdevtype = &thisNetDevType

	res, err := c.AddVmNic(ctx, &thisVmNic)
	if err != nil {
		log.Fatalf("could not create nic: %v", err)
		return
	}
	fmt.Printf("Created vmnic %v\n", res.Value)

}

func addSwitch(namePtr *string, c pb.VMInfoClient, ctx context.Context, descrPtr *string, switchTypePtr *string) {
	var thisSwitchType pb.SwitchType
	if *namePtr == "" {
		log.Fatalf("Name not specified")
		return
	}
	if *switchTypePtr == "" {
		log.Fatalf("Switch type not specified")
		return
	}
	if *switchTypePtr == "IF" {
		thisSwitchType = pb.SwitchType_IF
	} else if *switchTypePtr == "NG" {
		thisSwitchType = pb.SwitchType_NG
	} else {
		log.Fatalf("Switch type must be either \"IF\" or \"NG\"")
		return
	}

	log.Printf("Creating switch %v type %v", *namePtr, *switchTypePtr)

	var thisSwitchInfo pb.SwitchInfo
	thisSwitchInfo.Name = namePtr
	thisSwitchInfo.Description = descrPtr
	thisSwitchInfo.SwitchType = &thisSwitchType

	res, err := c.AddSwitch(ctx, &thisSwitchInfo)
	if err != nil {
		log.Fatalf("could not create switch: %v", err)
		return
	}
	fmt.Printf("Created switch %v\n", res.Value)
}

func addDisk(namePtr *string, c pb.VMInfoClient, ctx context.Context, descrPtr *string, sizePtr *string) {
	if *namePtr == "" {
		log.Fatalf("Name not specified")
		return
	}
	res, err := c.AddDisk(ctx, &pb.DiskInfo{
		Name:        namePtr,
		Description: descrPtr,
		Size:        sizePtr,
	})
	if err != nil {
		log.Fatalf("could not create Disk: %v", err)
		return
	}
	fmt.Printf("Created Disk %v\n", res.Value)

}

func DeleteVM(idPtr *string, c pb.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	reqId, err := c.DeleteVM(ctx, &pb.VMID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not delete VM: %v", err)
	}
	fmt.Printf("Deleted request created, reqid: %v\n", reqId.Value)
}

func stopVM(idPtr *string, c pb.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	reqId, err := c.StopVM(ctx, &pb.VMID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not stop VM: %v", err)
	}
	fmt.Printf("Stopping request created, reqid: %v\n", reqId.Value)
}

func startVM(idPtr *string, c pb.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	reqId, err := c.StartVM(ctx, &pb.VMID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not start VM: %v", err)
	}
	fmt.Printf("Started request created, reqid: %v\n", reqId.Value)
}

func ReqStat(idPtr *string, c pb.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	res, err := c.RequestStatus(ctx, &pb.RequestID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not get req: %v", err)
	}
	fmt.Printf("complete: %v status: %v\n", res.Complete, res.Success)

}

func getSwitch(idPtr *string, c pb.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	res, err := c.GetSwitchInfo(ctx, &pb.SwitchId{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
	}
	fmt.Printf(
		"name: %v "+
			"description: %v "+
			"type: %v "+
			"\n",
		*res.Name,
		*res.Description,
		*res.SwitchType,
	)

}

func getVmNic(idPtr *string, c pb.VMInfoClient, ctx context.Context) {
	var netTypeString string
	var netDevTypeString string
	var descriptionStr string

	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	res, err := c.GetVmNicInfo(ctx, &pb.VmNicId{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
	}

	if res.Description != nil {
		descriptionStr = *res.Description
	}

	if *res.Nettype == pb.NetType_VIRTIONET {
		netTypeString = "VirtioNet"
	} else if *res.Nettype == pb.NetType_E1000 {
		netTypeString = "E1000"
	}

	if *res.Netdevtype == pb.NetDevType_TAP {
		netDevTypeString = "TAP"
	} else if *res.Netdevtype == pb.NetDevType_VMNET {
		netDevTypeString = "VMNet"
	} else if *res.Netdevtype == pb.NetDevType_NETGRAPH {
		netDevTypeString = "Netgraph"
	}

	fmt.Printf(
		"name: %v "+
			"desc: %v "+
			"Mac: %v "+
			"Net_type: %v "+
			"Net_dev_type: %v "+
			"vm_id: %v "+
			"switch_id: %v "+
			"\n",
		*res.Name,
		descriptionStr,
		*res.Mac,
		netTypeString,
		netDevTypeString,
		*res.Vmid,
		*res.Switchid,
	)
}

func getVM(idPtr *string, c pb.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	res, err := c.GetVMConfig(ctx, &pb.VMID{Value: *idPtr})
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

func getVMs(c pb.VMInfoClient, ctx context.Context) {
	res, err := c.GetVMs(ctx, &pb.VMsQuery{})
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

func getVmNics(c pb.VMInfoClient, ctx context.Context) {
	res, err := c.GetVmNicsAll(ctx, &pb.VmNicsQuery{})
	if err != nil {
		log.Fatalf("could not get VmNics: %v", err)
		return
	}
	for {
		VMNicId, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("GetVmNiss failed: %v", err)
		}
		fmt.Printf("VmNic: id: %v\n", VMNicId.Value)
	}
}

func getSwitches(c pb.VMInfoClient, ctx context.Context) {
	res, err := c.GetSwitches(ctx, &pb.SwitchesQuery{})
	if err != nil {
		log.Fatalf("could not get Switches: %v", err)
		return
	}
	for {
		VmSwitch, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("GetSwitches failed: %v", err)
		}
		fmt.Printf("Switch: id: %v\n", VmSwitch.Value)
	}
}

func getVMState(idPtr *string, c pb.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	res, err := c.GetVMState(ctx, &pb.VMID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not get state: %v", err)
		return
	}
	var vmstate string
	switch res.Status {
	case pb.VmStatus_STATUS_STOPPED:
		vmstate = "stopped"
	case pb.VmStatus_STATUS_STARTING:
		vmstate = "starting"
	case pb.VmStatus_STATUS_RUNNING:
		vmstate = "running"
	case pb.VmStatus_STATUS_STOPPING:
		vmstate = "stopping"
	}
	fmt.Printf("vm id: %v state: %v vnc port: %v\n", *idPtr, vmstate, res.VncPort)
}

func Reconfig(idPtr *string, err error, namePtr *string, descrPtr *string, cpuPtr *uint, memPtr *uint, autoStartPtr *bool, c pb.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	newConfig := pb.VMConfig{
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

func printActionHelp() {
	println("Actions: getVM, getVMs, getVMState, addVM, reConfig, deleteVM, reqStat, startVM, stopVM, "+
		"addISO, addDisk, addSwitch, addVmNic", "getSwitches", "getVmNics", "getSwitch", "getVmNic")
}

func main() {
	actionPtr := flag.String("action", "", "action to take")
	idPtr := flag.String("id", "", "ID of VM")
	namePtr := flag.String("name", "", "Name of VM/ISO/Disk")
	descrPtr := flag.String("descr", "", "Description of VM/ISO/Disk")
	sizePtr := flag.String("size", "", "Size of Disk")
	switchTypePtr := flag.String("switchType", "IF", "Type of switch (IF or NG)")
	cpuPtr := flag.Uint("cpus", 1, "Number of CPUs in VM")
	cpuVal := *cpuPtr
	cpu32Val := uint32(cpuVal)
	cpu32Ptr := &cpu32Val
	memPtr := flag.Uint("mem", 128, "Memory in VM (MB)")
	memVal := *memPtr
	mem32Val := uint32(memVal)
	mem32Ptr := &mem32Val
	autoStartPtr := flag.Bool("autostart", false, "automatically start the VM")
	netTypePtr := flag.String("netType", "VIRTIONET", "Type of net (VIRTIONET or E1000")
	netDevTypePtr := flag.String("netDevType", "TAP", "type of net dev (TAP, VMNET or NETGRAPH")
	macPtr := flag.String("mac", "AUTO", "Mac address of NIC (or AUTO)")
	//maxWaitPtr := flag.Uint("maxWait", 120, "Max wait time for VM shutdown")
	//restartPtr := flag.Bool("restart", true, "Automatically restart VM")
	//restartDelayPtr := flag.Uint("restartDelay", 1, "How long to wait before restarting VM")
	//screenPtr := flag.Bool("screen", true, "Should the VM have a screen (frame buffer)")
	//screenWidthPtr := flag.Uint("screenWidth", 1920, "Width of VM screen")
	//screenHeightPtr := flag.Uint(screenHeight, 1080, "Height of VM screen")
	pathPtr := flag.String("path", "", "path of ISO")

	flag.Parse()
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Fatalf("Failed to close connection")
		}
	}(conn)
	c := pb.NewVMInfoClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	switch *actionPtr {
	case "":
		log.Fatalf("Action not specified")
	case "help":
		printActionHelp()
	case "getVM":
		getVM(idPtr, c, ctx)
	case "getSwitch":
		getSwitch(idPtr, c, ctx)
	case "getVmNic":
		getVmNic(idPtr, c, ctx)
	case "getVMs":
		getVMs(c, ctx)
	case "getSwitches":
		getSwitches(c, ctx)
	case "getVmNics":
		getVmNics(c, ctx)
	case "getVMState":
		getVMState(idPtr, c, ctx)
	case "addVM":
		addVM(namePtr, c, ctx, descrPtr, cpu32Ptr, mem32Ptr)
	case "addISO":
		addISO(namePtr, c, ctx, descrPtr, pathPtr)
	case "addDisk":
		addDisk(namePtr, c, ctx, descrPtr, sizePtr)
	case "addSwitch":
		addSwitch(namePtr, c, ctx, descrPtr, switchTypePtr)
	case "addVmNic":
		addVmNic(namePtr, c, ctx, descrPtr, netTypePtr, netDevTypePtr, macPtr)
	case "reConfig":
		Reconfig(idPtr, err, namePtr, descrPtr, cpuPtr, memPtr, autoStartPtr, c, ctx)
	case "deleteVM":
		DeleteVM(idPtr, c, ctx)
	case "reqStat":
		ReqStat(idPtr, c, ctx)
	case "startVM":
		startVM(idPtr, c, ctx)
	case "stopVM":
		stopVM(idPtr, c, ctx)
	default:
		log.Fatalf("Action %v unknown", *actionPtr)
	}
}
