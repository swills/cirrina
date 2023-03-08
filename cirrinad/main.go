package main

import (
	pb "cirrina/cirrina"
	"context"
	"errors"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
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
	VMID         string
	Cpu          uint32 `gorm:"default:1;check:cpu BETWEEN 1 and 16"`
	Mem          uint32 `gorm:"default:128;check:mem>=128"`
	MaxWait      uint32 `gorm:"default:120;check:max_wait>=0"`
	Restart      bool   `gorm:"default:True;check:restart IN (0,1)"`
	RestartDelay uint32 `gorm:"default:1;check:restart_delay>=0"`
	Screen       bool   `gorm:"default:True;check:screen IN (0,1)"`
	ScreenWidth  uint32 `gorm:"default:1920;check:screen_width BETWEEN 640 and 1920"`
	ScreenHeight uint32 `gorm:"default:1080;check:screen_height BETWEEN 480 and 1200"`
}

type VM struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null"`
	Name        string `gorm:"uniqueIndex;not null"`
	Description string
	Status      string
	BhyvePid    int32
	NetDev      string
	VNCPort     int32
	VMConfig    VMConfig
}

func (vm *VM) BeforeCreate(_ *gorm.DB) (err error) {
	vm.ID = uuid.NewString()
	return nil
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

func (s *server) AddVM(_ context.Context, v *pb.VM) (*pb.VMID, error) {
	db := getVMDB()
	var evm VM
	db.Limit(1).Find(&evm, &VM{Name: v.Name})
	if evm.ID != "" {
		return &pb.VMID{}, errors.New("already exists")
	}
	vm := VM{
		Name:        v.Name,
		Status:      "STOPPED",
		Description: v.Description,
		VMConfig: VMConfig{
			Cpu: v.Cpu,
			Mem: v.Mem,
		},
	}
	res := db.Create(&vm)
	if res.Error != nil {
		return &pb.VMID{}, errors.New("error Creating VM")
	}
	return &pb.VMID{Value: vm.ID}, nil
}

func (s *server) GetVM(_ context.Context, v *pb.VMID) (*pb.VM, error) {
	var vm VM
	var pvm pb.VM

	db := getVMDB()
	db.Model(&VM{}).Preload("VMConfig").Limit(1).Find(&vm, &VM{ID: v.Value})
	if vm.ID == "" {
		return &pvm, errors.New("not found")
	}
	pvm.Name = vm.Name
	pvm.Description = vm.Description
	pvm.Cpu = vm.VMConfig.Cpu
	pvm.Mem = vm.VMConfig.Mem
	pvm.MaxWait = vm.VMConfig.MaxWait
	pvm.Restart = vm.VMConfig.Restart
	pvm.RestartDelay = vm.VMConfig.RestartDelay
	pvm.Screen = vm.VMConfig.Screen
	pvm.ScreenWidth = vm.VMConfig.ScreenWidth
	pvm.ScreenHeight = vm.VMConfig.ScreenHeight
	return &pvm, nil
}

func (s *server) GetVMs(_ *pb.VMsQuery, stream pb.VMInfo_GetVMsServer) error {
	var vms []VM
	var pvm pb.VMID

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

func (s *server) GetVMState(_ context.Context, p *pb.VMID) (*pb.VMState, error) {
	v := VM{}
	r := pb.VMState{}
	vmDB := getVMDB()
	vmDB.Limit(1).Find(&v, &VM{ID: p.Value})
	if v.ID == "" {
		return &r, errors.New("not found")
	}
	r.Status = v.Status
	return &r, nil
}

func isOptionPassed(pref protoreflect.Message, name string) bool {
	field := pref.Descriptor().Fields().ByName(protoreflect.Name(name))
	if pref.Has(field) {
		return true
	}
	return false
}

func (s *server) UpdateVM(_ context.Context, rc *pb.VMReConfig) (*pb.ReqBool, error) {
	re := pb.ReqBool{}
	re.Success = false
	var vm VM
	db := getVMDB()
	db.Model(&VM{}).Preload("VMConfig").Limit(1).Find(&vm, &VM{ID: rc.Id})
	if vm.ID == "" {
		return &re, errors.New("not found")
	}
	pref := rc.ProtoReflect()
	if isOptionPassed(pref, "name") {
		vm.Name = *rc.Name
	}
	if isOptionPassed(pref, "description") {
		vm.Description = *rc.Description
	}
	if isOptionPassed(pref, "cpu") {
		vm.VMConfig.Cpu = *rc.Cpu
	}
	if isOptionPassed(pref, "mem") {
		vm.VMConfig.Mem = *rc.Mem
	}
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		return &re, errors.New("error updating VM")
	}
	re.Success = true
	return &re, nil
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
