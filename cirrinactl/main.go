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
	idPtr := flag.String("id", "", "ID of VM")

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
		if *idPtr == "" {
			log.Fatalf("ID not specified")
			return
		}
		r, err := c.GetVM(ctx, &pb.VmID{Value: *idPtr})
		if err != nil {
			log.Fatalf("could not get VM: %v", err)
		}
		if r.Name == "" {
			log.Fatalf("VM ID %v not found", idPtr)
		}
		log.Printf("name: %v id: %v, desc: %v, cpu: %v, mem: %v", r.Name, r.Id, r.Description, r.Cpu, r.Mem)
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
			log.Printf("VM: id: %v name: %v desc: %v cpu: %v mem: %v", VM.Id, VM.Name, VM.Description, VM.Cpu, VM.Mem)
		}
		return
	}
	log.Fatalf("Action %v unknown", *actionPtr)
}
