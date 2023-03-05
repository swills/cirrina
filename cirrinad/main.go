package main

import (
	"context"
	"google.golang.org/grpc"
	"log"
	"net"

	pb "cirrina/cirrina"
)

const (
	port = ":50051"
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
	err := stream.Send(&pb.VM{Name: "some VM"})
	if err != nil {
		return err
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
