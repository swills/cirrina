package main

import (
	"context"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"

	pb "cirrina/cirrina"
)

const (
	port       = ":50051"
	configPath = "/usr/home/swills/.config/weasel/vms/"
)

type server struct {
	pb.UnimplementedVMInfoServer
}

func (s *server) GetVM(_ context.Context, in *pb.VmID) (*pb.VM, error) {
	log.Printf("Received: %v", in.GetValue())
	return &pb.VM{}, nil
}

func (s *server) GetVMs(_ *pb.VMsQuery, stream pb.VMInfo_GetVMsServer) error {
	log.Printf("Got GetVMs query")
	entries, err := os.ReadDir(configPath)
	if err != nil {
		log.Fatal(err)
	}

	for _, e := range entries {
		vmName := e.Name()[:len(e.Name())-5]
		err := stream.Send(&pb.VM{Name: vmName})
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterVMInfoServer(s, &server{})
	log.Printf("Starting gRPC listener on port " + port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
