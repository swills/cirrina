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

const (
	defaultName = "default"
)

var (
	addr = flag.String("addr", "localhost:50051", "the address to connect to")
	name = flag.String("name", defaultName, "Name to greet")
)

func main() {
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
	r, err := c.GetVM(ctx, &pb.VmID{Value: *name})
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
	}
	log.Printf("Greeting: %s", r.Name)

	t, err := c.GetVMs(ctx, &pb.VMsQuery{})
	if err != nil {
		log.Fatalf("could not get VMs: %v", err)
	}
	log.Printf("Getting VMs")
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
}
