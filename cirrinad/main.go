package main

import (
	pb "cirrina/cirrina"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"net"
	"time"
)

const (
	port = ":50051"
)

type server struct {
	pb.UnimplementedVMInfoServer
}

type reqType string

const (
	START  reqType = "START"
	STOP   reqType = "STOP"
	DELETE reqType = "DELETE"
)

type statusType string

const (
	STOPPED  statusType = "STOPPED"
	STARTING statusType = "STARTING"
	RUNNING  statusType = "RUNNING"
	STOPPING statusType = "STOPPING"
)

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
	Name        string `gorm:"not null"`
	Description string
	Status      statusType `gorm:"type:status_type"`
	BhyvePid    uint32     `gorm:"check:bhyve_pid>=0"`
	NetDev      string
	VNCPort     int32
	VMConfig    VMConfig
}

type Request struct {
	gorm.Model
	ID         string       `gorm:"uniqueIndex;not null"`
	StartedAt  sql.NullTime `gorm:"index"`
	Successful bool         `gorm:"default:False;check:successful IN (0,1)"`
	Complete   bool         `gorm:"default:False;check:complete IN (0,1)"`
	Type       reqType      `gorm:"type:req_type"`
	VMID       string
}

func (vm *VM) BeforeCreate(_ *gorm.DB) (err error) {
	vm.ID = uuid.NewString()
	return nil
}

func (req *Request) BeforeCreate(_ *gorm.DB) (err error) {
	req.ID = uuid.NewString()
	return nil
}

func getVMDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("cirrina.sqlite"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	err = db.AutoMigrate(&VM{})
	if err != nil {
		panic("failed to auto-migrate VMs")
	}
	err = db.AutoMigrate(&VMConfig{})
	if err != nil {
		panic("failed to auto-migrate Configs")
	}
	err = db.AutoMigrate(&Request{})
	if err != nil {
		panic("failed to auto-migrate Requests")
	}
	return db
}

func vmExists(v *pb.VMID) bool {
	vm := VM{}
	db := getVMDB()
	db.Model(&VM{}).Limit(1).Find(&vm, &VM{ID: v.Value})
	if vm.ID == "" {
		return false
	}
	return true
}

func pendingReqExists(v *pb.VMID) bool {
	db := getVMDB()
	eReq := Request{}
	db.Where(map[string]interface{}{"vm_id": v.Value, "complete": false}).Find(&eReq)
	if eReq.ID != "" {
		return true
	}
	return false
}

func isOptionPassed(reflect protoreflect.Message, name string) bool {
	field := reflect.Descriptor().Fields().ByName(protoreflect.Name(name))
	if reflect.Has(field) {
		return true
	}
	return false
}

func (s *server) AddVM(_ context.Context, v *pb.VM) (*pb.VMID, error) {
	db := getVMDB()
	var evm VM
	db.Limit(1).Find(&evm, &VM{Name: v.Name})
	if evm.ID != "" {
		return &pb.VMID{}, errors.New(fmt.Sprintf("%v already exists", v.Name))
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
	var pvmid pb.VMID

	db := getVMDB()
	db.Find(&vms)

	for e := range vms {
		pvmid.Value = vms[e].ID
		err := stream.Send(&pvmid)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *server) GetVMState(_ context.Context, p *pb.VMID) (*pb.VMState, error) {
	vm := VM{}
	pvm := pb.VMState{}
	vmDB := getVMDB()
	vmDB.Limit(1).Find(&vm, &VM{ID: p.Value})
	if vm.ID == "" {
		return &pvm, errors.New("not found")
	}
	switch vm.Status {
	case STOPPED:
		pvm.Status = pb.VmStatus_STATUS_STOPPED
	case STARTING:
		pvm.Status = pb.VmStatus_STATUS_STARTING
	case RUNNING:
		pvm.Status = pb.VmStatus_STATUS_RUNNING
	case STOPPING:
		pvm.Status = pb.VmStatus_STATUS_STOPPING
	default:
		return &pvm, errors.New("internal error: unknown VM state")
	}
	return &pvm, nil
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
	reflect := rc.ProtoReflect()
	if isOptionPassed(reflect, "name") {
		vm.Name = *rc.Name
	}
	if isOptionPassed(reflect, "description") {
		vm.Description = *rc.Description
	}
	if isOptionPassed(reflect, "cpu") {
		vm.VMConfig.Cpu = *rc.Cpu
	}
	if isOptionPassed(reflect, "mem") {
		vm.VMConfig.Mem = *rc.Mem
	}
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		return &re, errors.New("error updating VM")
	}
	re.Success = true
	return &re, nil
}

func (s *server) StartVM(_ context.Context, v *pb.VMID) (*pb.RequestID, error) {
	if !vmExists(v) {
		return &pb.RequestID{}, errors.New("VM not found")
	}
	if pendingReqExists(v) {
		return &pb.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	db := getVMDB()
	vm := VM{}
	db.Model(&VM{}).Preload("VMConfig").Limit(1).Find(&vm, &VM{ID: v.Value})
	if vm.Status != STOPPED {
		return &pb.RequestID{}, errors.New("vm must be stopped before starting")
	}
	newReq := Request{}
	newReq.Type = START
	newReq.VMID = v.Value
	db.Create(&newReq)
	return &pb.RequestID{Value: newReq.ID}, nil
}

func (s *server) StopVM(_ context.Context, v *pb.VMID) (*pb.RequestID, error) {
	if !vmExists(v) {
		return &pb.RequestID{}, errors.New("VM not found")
	}
	if pendingReqExists(v) {
		return &pb.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	db := getVMDB()
	vm := VM{}
	db.Model(&VM{}).Preload("VMConfig").Limit(1).Find(&vm, &VM{ID: v.Value})
	if vm.Status != RUNNING {
		return &pb.RequestID{}, errors.New("vm must be running before stopping")
	}
	newReq := Request{}
	newReq.Type = STOP
	newReq.VMID = v.Value
	db.Create(&newReq)
	return &pb.RequestID{Value: newReq.ID}, nil
}

func (s *server) DeleteVM(_ context.Context, v *pb.VMID) (*pb.RequestID, error) {
	if !vmExists(v) {
		return &pb.RequestID{}, errors.New("VM not found")
	}
	if pendingReqExists(v) {
		return &pb.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	db := getVMDB()
	vm := VM{}
	db.Model(&VM{}).Preload("VMConfig").Limit(1).Find(&vm, &VM{ID: v.Value})
	if vm.Status != STOPPED {
		return &pb.RequestID{}, errors.New("vm must be stopped before deleting")
	}
	newReq := Request{}
	newReq.Type = DELETE
	newReq.VMID = v.Value
	db.Create(&newReq)
	return &pb.RequestID{Value: newReq.ID}, nil
}

func startVM(rs *Request) {
	time.Sleep(5 * time.Second)
	vm := VM{ID: rs.VMID}
	vm.Status = RUNNING
	db := getVMDB()
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		db.Model(&rs).Limit(1).Updates(
			Request{
				Successful: false,
				Complete:   true,
			},
		)
		return
	}
	db.Model(&rs).Limit(1).Updates(
		Request{
			Successful: true,
			Complete:   true,
		},
	)
}

func stopVM(rs *Request) {
	time.Sleep(5 * time.Second)
	vm := VM{ID: rs.VMID}
	db := getVMDB()
	vm.Status = STOPPED
	res := db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&vm)
	if res.Error != nil {
		db.Model(&rs).Limit(1).Updates(
			Request{
				Successful: false,
				Complete:   true,
			},
		)
		return
	}
	db.Model(&rs).Limit(1).Updates(
		Request{
			Successful: true,
			Complete:   true,
		},
	)
}

func deleteVM(rs *Request) {
	time.Sleep(5 * time.Second)
	vm := VM{}
	db := getVMDB()
	db.Model(&VM{}).Preload("VMConfig").Limit(1).Find(&vm, &VM{ID: rs.VMID})
	res := db.Delete(&vm.VMConfig)
	if res.RowsAffected != 1 {
		db.Model(&rs).Limit(1).Updates(
			Request{
				Successful: false,
				Complete:   true,
			},
		)
		return
	}
	res = db.Delete(&vm)
	if res.RowsAffected != 1 {
		db.Model(&rs).Limit(1).Updates(
			Request{
				Successful: false,
				Complete:   true,
			},
		)
		return
	}
	db.Model(&rs).Limit(1).Updates(
		Request{
			Successful: true,
			Complete:   true,
		},
	)
}

func (s *server) RequestStatus(_ context.Context, r *pb.RequestID) (*pb.ReqStatus, error) {
	db := getVMDB()
	rs := Request{}
	db.Model(&Request{}).Limit(1).Find(&rs, &Request{ID: r.Value})
	if rs.ID == "" {
		return &pb.ReqStatus{}, errors.New("not found")
	}
	res := &pb.ReqStatus{
		Complete: rs.Complete,
		Success:  rs.Successful,
	}
	return res, nil
}

func rpcServer() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterVMInfoServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func processRequests() {
	db := getVMDB()
	for {
		rs := Request{}
		db.Limit(1).Where("started_at IS NULL").Find(&rs)
		if rs.ID != "" {
			rs.StartedAt.Time = time.Now()
			rs.StartedAt.Valid = true
			db.Model(&rs).Limit(1).Updates(rs)
			switch rs.Type {
			case START:
				go startVM(&rs)
			case STOP:
				go stopVM(&rs)
			case DELETE:
				go deleteVM(&rs)
			}

		}

		time.Sleep(500 * time.Millisecond)
	}
}

func main() {
	go rpcServer()
	go processRequests()
	for {
		time.Sleep(1 * time.Second)
	}
}
