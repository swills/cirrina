package main

import (
	pb "cirrina/cirrina"
	"context"
	"errors"
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
	VMID         uint32
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
	ID          uint32
	Name        string `gorm:"uniqueIndex;not null"`
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
	var vm VM
	var pvm pb.VM
	var config VMConfig

	db := getVMDB()
	db.Where("id = ?", v.Value).Find(&vm)
	if vm.Name == "" {
		return &pvm, errors.New("not found")
	}
	db.Where("vm_id = ?", v.Value).Find(&config)
	if vm.ID != 0 {
		pvm.Name = vm.Name
		pvm.Description = vm.Description
	}
	pvm.Cpu = config.Cpu
	pvm.Mem = config.Mem
	pvm.MaxWait = config.MaxWait
	pvm.Restart = config.Restart
	pvm.RestartDelay = config.RestartDelay
	pvm.Screen = config.Screen
	pvm.ScreenWidth = config.ScreenWidth
	pvm.ScreenHeight = config.ScreenHeight
	return &pvm, nil
}

func (s *server) GetVMs(_ *pb.VMsQuery, stream pb.VMInfo_GetVMsServer) error {
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
	vmDB := getVMDB()
	vmDB.Where(&VM{ID: p.Value}).Limit(1).Find(&v)
	if v.ID == 0 {
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
	var config VMConfig
	db := getVMDB()
	db.Where("id = ?", rc.Id).Find(&vm)
	if vm.Name == "" {
		return &re, errors.New("not found")
	}
	db.Where("vm_id = ?", rc.Id).Find(&config)
	pref := rc.ProtoReflect()
	if isOptionPassed(pref, "name") {
		vm.Name = *rc.Name
	}
	if isOptionPassed(pref, "description") {
		vm.Description = *rc.Description
	}
	if isOptionPassed(pref, "cpu") {
		config.Cpu = *rc.Cpu
	}
	if isOptionPassed(pref, "mem") {
		config.Mem = *rc.Mem
	}
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		return &re, errors.New("error updating VM")
	}
	if err := tx.Save(&vm).Error; err != nil {
		tx.Rollback()
		return &re, errors.New("error updating VM")
	}
	if err := tx.Save(&config).Error; err != nil {
		tx.Rollback()
		return &re, errors.New("error updating VM")
	}
	res := tx.Commit().Error
	if res != nil {
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
