package main

import (
	pb "cirrina/cirrina"
	"context"
	"google.golang.org/grpc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"net"
)

const (
	port = ":50051"
)

type server struct {
	pb.UnimplementedVMInfoServer
}

func (s *server) GetVM(_ context.Context, v *pb.VmID) (*pb.VM, error) {
	log.Printf("Getting VM %v", v.Value)
	var vm pb.VM

	db, err := gorm.Open(sqlite.Open("cirrina.sqlite"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	err = db.AutoMigrate(&pb.VM{})
	if err != nil {
		panic("failed to auto-migrate")
	}
	db.Where("id = ?", v.Value).First(&vm)
	return &vm, nil
}

func (s *server) GetVMs(_ *pb.VMsQuery, stream pb.VMInfo_GetVMsServer) error {
	log.Printf("Getting VMs")
	var vms []pb.VM

	db, err := gorm.Open(sqlite.Open("cirrina.sqlite"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	err = db.AutoMigrate(&pb.VM{})
	if err != nil {
		panic("failed to auto-migrate")
	}
	db.Find(&vms)

	for e := range vms {
		err := stream.Send(&vms[e])
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
