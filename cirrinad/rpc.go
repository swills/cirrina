package main

import (
	"cirrina/cirrina"
	"cirrina/cirrinad/requests"
	vm2 "cirrina/cirrinad/vm"
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
	existsAlready := vm2.DbVMExists(v.Name)
	if existsAlready {
		return &cirrina.VMID{}, errors.New(fmt.Sprintf("%v already exists", v.Name))
	}
	vmInst := vm2.VM{
		Name:        v.Name,
		Status:      vm2.STOPPED,
		Description: v.Description,
		VMConfig: vm2.Config{
			Cpu: v.Cpu,
			Mem: v.Mem,
		},
	}
	err := vm2.DbCreateVM(&vmInst)
	log.Printf("Created VM %v", vmInst.ID)
	if err != nil {
		return &cirrina.VMID{}, errors.New("error Creating VM")
	}
	return &cirrina.VMID{Value: vmInst.ID}, nil
}

func (s *server) GetVM(_ context.Context, v *cirrina.VMID) (*cirrina.VM, error) {
	var pvm cirrina.VM
	vmInst, err := vm2.Get(v.Value)
	if err != nil {
		log.Printf("error getting vm %v, %v", v.Value, err)
		return &pvm, errors.New("not found")
	}
	pvm.Name = vmInst.Name
	pvm.Description = vmInst.Description
	pvm.Cpu = vmInst.VMConfig.Cpu
	pvm.Mem = vmInst.VMConfig.Mem
	pvm.MaxWait = vmInst.VMConfig.MaxWait
	pvm.Restart = vmInst.VMConfig.Restart
	pvm.RestartDelay = vmInst.VMConfig.RestartDelay
	pvm.Screen = vmInst.VMConfig.Screen
	pvm.ScreenWidth = vmInst.VMConfig.ScreenWidth
	pvm.ScreenHeight = vmInst.VMConfig.ScreenHeight
	return &pvm, nil
}

func (s *server) GetVMs(_ *cirrina.VMsQuery, stream cirrina.VMInfo_GetVMsServer) error {
	var vms []vm2.VM
	var pvmid cirrina.VMID

	db := vm2.GetVMDB()
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
	vmInst, err := vm2.Get(p.Value)
	if err != nil {
		log.Printf("error getting vm %v, %v", p.Value, err)
	}
	pvm := cirrina.VMState{}
	switch vmInst.Status {
	case vm2.STOPPED:
		pvm.Status = cirrina.VmStatus_STATUS_STOPPED
	case vm2.STARTING:
		pvm.Status = cirrina.VmStatus_STATUS_STARTING
	case vm2.RUNNING:
		pvm.Status = cirrina.VmStatus_STATUS_RUNNING
	case vm2.STOPPING:
		pvm.Status = cirrina.VmStatus_STATUS_STOPPING
	default:
		return &pvm, errors.New("internal error: unknown VM state")
	}
	return &pvm, nil
}

func (s *server) UpdateVM(_ context.Context, rc *cirrina.VMReConfig) (*cirrina.ReqBool, error) {
	re := cirrina.ReqBool{}
	re.Success = false
	var vm vm2.VM
	db := vm2.GetVMDB()
	db.Model(&vm2.VM{}).Preload("VMConfig").Limit(1).Find(&vm, &vm2.VM{ID: rc.Id})
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
	if !vm2.Exists(v.Value) {
		return &cirrina.RequestID{}, errors.New("VM not found")
	}
	if requests.PendingReqExists(v.Value) {
		return &cirrina.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	vmInst, err := vm2.Get(v.Value)
	if err != nil {
		log.Printf("error getting vm %v, %v", v.Value, err)
	}
	if vmInst.Status != vm2.STOPPED {
		return &cirrina.RequestID{}, errors.New("vm must be stopped before starting")
	}
	newReq, err := requests.Create(requests.START, v.Value)
	if err != nil {
		return nil, err
	}
	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) StopVM(_ context.Context, v *cirrina.VMID) (*cirrina.RequestID, error) {
	if !vm2.Exists(v.Value) {
		return &cirrina.RequestID{}, errors.New("VM not found")
	}
	if requests.PendingReqExists(v.Value) {
		return &cirrina.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	vmInst, err := vm2.Get(v.Value)
	if err != nil {
		log.Printf("error getting vm %v, %v", v.Value, err)
	}
	if vmInst.Status != vm2.RUNNING {
		return &cirrina.RequestID{}, errors.New("vm must be running before stopping")
	}
	newReq, err := requests.Create(requests.STOP, v.Value)
	if err != nil {
		return nil, err
	}
	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) DeleteVM(_ context.Context, v *cirrina.VMID) (*cirrina.RequestID, error) {
	vmInst, err := vm2.Get(v.Value)
	if err != nil {
		return &cirrina.RequestID{}, errors.New("VM not found")
	}
	if requests.PendingReqExists(v.Value) {
		return &cirrina.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	if vmInst.Status != vm2.STOPPED {
		return &cirrina.RequestID{}, errors.New("vm must be stopped before deleting")
	}
	newReq, err := requests.Create(requests.DELETE, v.Value)
	if err != nil {
		return nil, err
	}
	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) RequestStatus(_ context.Context, r *cirrina.RequestID) (*cirrina.ReqStatus, error) {
	rs := requests.Get(r.Value)
	if rs.ID == "" {
		return &cirrina.ReqStatus{}, errors.New("not found")
	}
	res := &cirrina.ReqStatus{
		Complete: rs.Complete,
		Success:  rs.Successful,
	}
	return res, nil
}

func isOptionPassed(reflect protoreflect.Message, name string) bool {
	field := reflect.Descriptor().Fields().ByName(protoreflect.Name(name))
	if reflect.Has(field) {
		return true
	}
	return false
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
