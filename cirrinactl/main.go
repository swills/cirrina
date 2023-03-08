package main

import (
	pb "cirrina/cirrina"
	"context"
	"flag"
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
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
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
	r, err := c.AddVM(ctx, &pb.VM{
		Name:        *namePtr,
		Description: *descrPtr,
		Cpu:         uint32(*cpuPtr),
		Mem:         uint32(*memPtr),
	})
	if err != nil {
		log.Fatalf("Failed to create VM")
		return
	}
	log.Printf("Created VM %v", r.Value)
}

func getVM(idPtr *uint, c pb.VMInfoClient, ctx context.Context) {
	if *idPtr == 0 {
		log.Fatalf("ID not specified")
		return
	}
	r, err := c.GetVM(ctx, &pb.VmID{Value: uint32(*idPtr)})
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
	}
	if r.Name == "" {
		log.Fatalf("VM ID %v not found", *idPtr)
		return
	}
	log.Printf("name: %v desc: %v cpus: %v", r.Name, r.Description, r.Cpu)
}

func getVMs(c pb.VMInfoClient, ctx context.Context) {
	t, err := c.GetVMs(ctx, &pb.VMsQuery{})
	if err != nil {
		log.Fatalf("could not get VMs: %v", err)
		return
	}
	for {
		VM, err := t.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("GetVMs failed: %v", err)
		}
		log.Printf("VM: id: %v", VM.Value)
	}
}

func getVMState(idPtr *uint, c pb.VMInfoClient, ctx context.Context) {
	if *idPtr == 0 {
		log.Fatalf("ID not specified")
		return
	}
	log.Print("getting VM state")
	r, err := c.GetVMState(ctx, &pb.VmID{Value: uint32(*idPtr)})
	if err != nil {
		log.Fatalf("could not get state: %v", err)
		return
	}
	log.Printf("vm id: %v state: %v vnc port: %v", *idPtr, r.Status, r.VncPort)
}

func Reconfig(idPtr *uint, err error, namePtr *string, descrPtr *string, cpuPtr *uint, memPtr *uint, c pb.VMInfoClient, ctx context.Context) {
	if *idPtr == 0 {
		log.Fatalf("ID not specified")
		return
	}
	if err != nil {
		log.Fatalf("could not get state: %v", err)
	}
	newConfig := pb.VMReConfig{
		Id: uint32(*idPtr),
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
	if err == nil {
		log.Printf("Success")
		return
	}
	log.Printf("Fail")
}

func main() {
	actionPtr := flag.String("action", "", "action to take")
	idPtr := flag.Uint("id", 0, "ID of VM")
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

	if *actionPtr == "" {
		log.Fatalf("Action not specified")
		return
	}
	if *actionPtr == "getVM" {
		getVM(idPtr, c, ctx)
		return
	}
	if *actionPtr == "getVMs" {
		getVMs(c, ctx)
		return
	}
	if *actionPtr == "getVMState" {
		getVMState(idPtr, c, ctx)
		return
	}
	if *actionPtr == "addVM" {
		addVM(namePtr, c, ctx, descrPtr, cpuPtr, memPtr)
		return
	}
	if *actionPtr == "Reconfig" {
		Reconfig(idPtr, err, namePtr, descrPtr, cpuPtr, memPtr, c, ctx)
		return
	}
	log.Fatalf("Action %v unknown", *actionPtr)
}
