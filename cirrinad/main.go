package main

import (
	pb "cirrina/cirrina"
	"context"
	"errors"
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

type VMConfig struct {
	gorm.Model
	VMID         uint32
	Cpu          uint32 `gorm:"default:1;check:cpu BETWEEN 1 and 16"`
	Mem          uint32 `gorm:"default:128;check:mem>=128"`
	MaxWait      uint32 `gorm:"default:120"`
	Restart      bool   `gorm:"default:True"`
	RestartDelay uint32 `gorm:"default:1"`
	Screen       bool   `gorm:"default:True"`
	ScreenWidth  uint32 `gorm:"default:1920;check:screen_width BETWEEN 640 and 1920"`
	ScreenHeight uint32 `gorm:"default:1080;check:screen_height BETWEEN 480 and 1200"`
}

type VM struct {
	gorm.Model
	ID          uint32
	Name        string `gorm:"uniqueIndex"`
	Description string
	Status      string
	BhyvePid    int32
	NetDev      string
	VNCPort     int32
	Config      VMConfig
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
	err = db.AutoMigrate(&VMConfig{})
	if err != nil {
		panic("failed to auto-migrate")
	}
	return db
}

func (s *server) AddVM(_ context.Context, v *pb.VM) (*pb.VmID, error) {
	log.Printf("Adding VM %v", v.Name)

	db := getVMDB()
	vm := VM{
		Name:        v.Name,
		Status:      "STOPPED",
		Description: v.Description,
		Config: VMConfig{
			Cpu: v.Cpu,
			Mem: v.Mem,
		},
	}
	res := db.Create(&vm)
	if res.RowsAffected != 1 {
		return &pb.VmID{}, errors.New("error Creating VM")
	}
	return &pb.VmID{Value: vm.ID}, nil
}

func (s *server) GetVM(_ context.Context, v *pb.VmID) (*pb.VM, error) {
	log.Printf("Getting VM %v", v.Value)
	var vm VM
	var pvm pb.VM

	db := getVMDB()
	db.Where("id = ?", v.Value).First(&vm)
	if vm.ID != 0 {
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
	if v.ID != 0 {
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
