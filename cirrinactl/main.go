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

func main() {
	actionPtr := flag.String("action", "", "action to take")
	idPtr := flag.Uint("id", 0, "ID of VM")
	namePtr := flag.String("name", "", "Name of VM")
	descrPtr := flag.String("descr", "", "Description of VM")
	cpuPtr := flag.Uint("cpus", 1, "Number of CPUs in VM")
	memPtr := flag.Uint("mem", 128, "Memory in VM (MB)")

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
		if *idPtr == 0 {
			log.Fatalf("ID not specified")
			return
		}
		r, err := c.GetVM(ctx, &pb.VmID{Value: uint32(*idPtr)})
		if err != nil {
			log.Fatalf("could not get VM: %v", err)
		}
		if r.Name == "" {
			log.Fatalf("VM ID %v not found", idPtr)
		}
		log.Printf("name: %v desc: %v", r.Name, r.Description)
		return
	}
	if *actionPtr == "getVMs" {
		log.Printf("Getting VMs")
		t, err := c.GetVMs(ctx, &pb.VMsQuery{})
		if err != nil {
			log.Fatalf("could not get VMs: %v", err)
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
		return
	}
	if *actionPtr == "getVMState" {
		if *idPtr == 0 {
			log.Fatalf("ID not specified")
			return
		}
		log.Print("getting VM state")
		r, err := c.GetVMState(ctx, &pb.VmID{Value: uint32(*idPtr)})
		if err != nil {
			log.Fatalf("could not get state: %v", err)
		}
		log.Printf("vm id: %v state: %v vnc port: %v", *idPtr, r.Status, r.VncPort)
		return
	}
	if *actionPtr == "addVM" {
		if *namePtr == "" {
			log.Fatalf("Name not specified")
		}
		r, err := c.AddVM(ctx, &pb.VM{
			Name:        *namePtr,
			Description: *descrPtr,
			Cpu:         uint32(*cpuPtr),
			Mem:         uint32(*memPtr),
		})
		if err != nil {
			log.Fatalf("Failed to create VM")
		}
		log.Printf("Created VM %v", r.Value)
		return
	}
	log.Fatalf("Action %v unknown", *actionPtr)
}
