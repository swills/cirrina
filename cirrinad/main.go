package main

import (
	pb "cirrina/cirrina"
	"context"
	"errors"
	"github.com/google/uuid"
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

type VM struct {
	gorm.Model
	ID          string
	Name        string `gorm:"uniqueIndex"`
	Description string
	ConfigID    string
	Status      string
	BhyvePid    int32
	NetDev      string
	VNCPort     int32
}

func getVMDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("cirrina.sqlite"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	err = db.AutoMigrate(&VM{})
	if err != nil {
		panic("failed to auto-migrate")
	}
	return db
}

func (s *server) AddVM(_ context.Context, v *pb.VM) (*pb.VmID, error) {
	log.Printf("Adding VM %v", v.Name)
	var vm VM

	db := getVMDB()
	vm.ID = uuid.NewString()
	vm.Name = v.Name
	vm.Status = "STOPPED"
	vm.Description = v.Description
	res := db.Create(&vm)

	log.Printf("Added %v", res.RowsAffected)
	if res.RowsAffected == 1 {
		return &pb.VmID{Value: vm.ID}, nil
	}
	return &pb.VmID{}, errors.New("error Creating VM")
}

func (s *server) GetVM(_ context.Context, v *pb.VmID) (*pb.VM, error) {
	log.Printf("Getting VM %v", v.Value)
	var vm VM
	var pvm pb.VM

	db := getVMDB()
	db.Where("id = ?", v.Value).First(&vm)
	if vm.ID != "" {
		pvm.Name = vm.Name
		pvm.Description = vm.Description
	}
	return &pvm, nil
}

func (s *server) GetVMs(_ *pb.VMsQuery, stream pb.VMInfo_GetVMsServer) error {
	log.Printf("Getting VMs")
	var vms []VM
	var pvm pb.VmID

	db := getVMDB()
	db.Find(&vms)

	for e := range vms {
		pvm.Value = vms[e].ID
		err := stream.Send(&pvm)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *server) GetVMState(_ context.Context, p *pb.VmID) (*pb.VMState, error) {
	v := VM{}
	r := pb.VMState{}
	log.Printf("Finding %v", p.Value)
	vmDB := getVMDB()
	vmDB.Where(&VM{ID: p.Value}).Limit(1).Find(&v)
	log.Printf("v: %v", v.ID)
	if v.ID != "" {
		r.Status = v.Status
	}
	return &r, nil
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
