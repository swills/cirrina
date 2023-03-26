package main

import (
	"cirrina/cirrina"
	"cirrina/cirrinad/requests"
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gorm.io/gorm"
	"log"
	"net"
)

type server struct {
	cirrina.UnimplementedVMInfoServer
}

func (s *server) AddVM(_ context.Context, v *cirrina.VM) (*cirrina.VMID, error) {
	existsAlready := dbVMExists(v.Name)
	if existsAlready {
		return &cirrina.VMID{}, errors.New(fmt.Sprintf("%v already exists", v.Name))
	}
	vm := VM{
		Name:        v.Name,
		Status:      STOPPED,
		Description: v.Description,
		VMConfig: VMConfig{
			Cpu: v.Cpu,
			Mem: v.Mem,
		},
	}
	err := dbCreateVM(vm)
	if err != nil {
		return &cirrina.VMID{}, errors.New("error Creating VM")
	}
	return &cirrina.VMID{Value: vm.ID}, nil
}

func (s *server) GetVM(_ context.Context, v *cirrina.VMID) (*cirrina.VM, error) {
	var vm VM
	var pvm cirrina.VM

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

func (s *server) GetVMs(_ *cirrina.VMsQuery, stream cirrina.VMInfo_GetVMsServer) error {
	var vms []VM
	var pvmid cirrina.VMID

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

func (s *server) GetVMState(_ context.Context, p *cirrina.VMID) (*cirrina.VMState, error) {
	vm := VM{}
	pvm := cirrina.VMState{}
	db := getVMDB()
	db.Limit(1).Find(&vm, &VM{ID: p.Value})
	if vm.ID == "" {
		return &pvm, errors.New("not found")
	}
	switch vm.Status {
	case STOPPED:
		pvm.Status = cirrina.VmStatus_STATUS_STOPPED
	case STARTING:
		pvm.Status = cirrina.VmStatus_STATUS_STARTING
	case RUNNING:
		pvm.Status = cirrina.VmStatus_STATUS_RUNNING
	case STOPPING:
		pvm.Status = cirrina.VmStatus_STATUS_STOPPING
	default:
		return &pvm, errors.New("internal error: unknown VM state")
	}
	return &pvm, nil
}

func (s *server) UpdateVM(_ context.Context, rc *cirrina.VMReConfig) (*cirrina.ReqBool, error) {
	re := cirrina.ReqBool{}
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

func (s *server) StartVM(_ context.Context, v *cirrina.VMID) (*cirrina.RequestID, error) {
	if !vmExists(v) {
		return &cirrina.RequestID{}, errors.New("VM not found")
	}
	if pendingReqExists(v) {
		return &cirrina.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	db := getVMDB()
	vm := VM{}
	db.Model(&VM{}).Preload("VMConfig").Limit(1).Find(&vm, &VM{ID: v.Value})
	if vm.Status != STOPPED {
		return &cirrina.RequestID{}, errors.New("vm must be stopped before starting")
	}
	newReq := requests.Request{}
	newReq.Type = requests.START
	newReq.VMID = v.Value
	db.Create(&newReq)
	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) StopVM(_ context.Context, v *cirrina.VMID) (*cirrina.RequestID, error) {
	if !vmExists(v) {
		return &cirrina.RequestID{}, errors.New("VM not found")
	}
	if pendingReqExists(v) {
		return &cirrina.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	db := getVMDB()
	vm := VM{}
	db.Model(&VM{}).Preload("VMConfig").Limit(1).Find(&vm, &VM{ID: v.Value})
	if vm.Status != RUNNING {
		return &cirrina.RequestID{}, errors.New("vm must be running before stopping")
	}
	newReq := requests.Request{}
	newReq.Type = requests.STOP
	newReq.VMID = v.Value
	db.Create(&newReq)
	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) DeleteVM(_ context.Context, v *cirrina.VMID) (*cirrina.RequestID, error) {
	if !vmExists(v) {
		return &cirrina.RequestID{}, errors.New("VM not found")
	}
	if pendingReqExists(v) {
		return &cirrina.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	db := getVMDB()
	vm := VM{}
	db.Model(&VM{}).Preload("VMConfig").Limit(1).Find(&vm, &VM{ID: v.Value})
	if vm.Status != STOPPED {
		return &cirrina.RequestID{}, errors.New("vm must be stopped before deleting")
	}
	newReq := requests.Request{}
	newReq.Type = requests.DELETE
	newReq.VMID = v.Value
	db.Create(&newReq)
	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) RequestStatus(_ context.Context, r *cirrina.RequestID) (*cirrina.ReqStatus, error) {
	db := getVMDB()
	rs := requests.Request{}
	db.Model(&requests.Request{}).Limit(1).Find(&rs, &requests.Request{ID: r.Value})
	if rs.ID == "" {
		return &cirrina.ReqStatus{}, errors.New("not found")
	}
	res := &cirrina.ReqStatus{
		Complete: rs.Complete,
		Success:  rs.Successful,
	}
	return res, nil
}

func pendingReqExists(v *cirrina.VMID) bool {
	db := getVMDB()
	eReq := requests.Request{}
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

func vmExists(v *cirrina.VMID) bool {
	vm := VM{}
	db := getVMDB()
	db.Model(&VM{}).Limit(1).Find(&vm, &VM{ID: v.Value})
	if vm.ID == "" {
		return false
	}
	return true
}

func rpcServer() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	cirrina.RegisterVMInfoServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
