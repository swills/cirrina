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

func addVM(namePtr *string, c pb.VMInfoClient, ctx context.Context, descrPtr *string, cpuPtr *uint, memPtr *uint) {
	if *namePtr == "" {
		log.Fatalf("Name not specified")
		return
	}
	res, err := c.AddVM(ctx, &pb.VM{
		Name:        *namePtr,
		Description: *descrPtr,
		Cpu:         uint32(*cpuPtr),
		Mem:         uint32(*memPtr),
	})
	if err != nil {
		log.Fatalf("could not create VM: %v", err)
		return
	}
	fmt.Printf("Created VM %v\n", res.Value)
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

func getVM(idPtr *string, c pb.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	res, err := c.GetVM(ctx, &pb.VMID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
	}
	fmt.Printf("name: %v desc: %v cpus: %v mem: %v\n",
		res.Name, res.Description, res.Cpu, res.Mem)
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

func Reconfig(idPtr *string, err error, namePtr *string, descrPtr *string, cpuPtr *uint, memPtr *uint, c pb.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	newConfig := pb.VMReConfig{
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
	_, err = c.UpdateVM(ctx, &newConfig)
	if err != nil {
		log.Fatalf("could not update VM: %v", err)
		return
	}
	fmt.Printf("Success\n")
}

func printActionHelp() {
	println("Actions: getVM, getVMs, getVMState, addVM, reConfig, deleteVM, reqStat, startVM, stopVM")
}

func main() {
	actionPtr := flag.String("action", "", "action to take")
	idPtr := flag.String("id", "", "ID of VM")
	namePtr := flag.String("name", "", "Name of VM")
	descrPtr := flag.String("descr", "", "Description of VM")
	cpuPtr := flag.Uint("cpus", 1, "Number of CPUs in VM")
	memPtr := flag.Uint("mem", 128, "Memory in VM (MB)")
	//maxWaitPtr := flag.Uint("maxWait", 120, "Max wait time for VM shutdown")
	//restartPtr := flag.Bool("restart", true, "Automatically restart VM")
	//restartDelayPtr := flag.Uint("restartDelay", 1, "How long to wait before restarting VM")
	//screenPtr := flag.Bool("screen", true, "Should the VM have a screen (frame buffer)")
	//screenWidthPtr := flag.Uint("screenWidth", 1920, "Width of VM screen")
	//screenHeightPtr := flag.Uint(screenHeight, 1080, "Height of VM screen")

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
	case "getVMs":
		getVMs(c, ctx)
	case "getVMState":
		getVMState(idPtr, c, ctx)
	case "addVM":
		addVM(namePtr, c, ctx, descrPtr, cpuPtr, memPtr)
	case "reConfig":
		Reconfig(idPtr, err, namePtr, descrPtr, cpuPtr, memPtr, c, ctx)
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
