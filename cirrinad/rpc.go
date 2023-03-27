package main

import (
	"cirrina/cirrina"
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/vm"
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"log"
	"net"
)

type server struct {
	cirrina.UnimplementedVMInfoServer
}

func (s *server) AddVM(_ context.Context, v *cirrina.VM) (*cirrina.VMID, error) {
	_, err := vm.GetByName(v.Name)
	if err == nil {
		return &cirrina.VMID{}, errors.New(fmt.Sprintf("%v already exists", v.Name))

	}
	vmInst := vm.VM{
		Name:        v.Name,
		Status:      vm.STOPPED,
		Description: v.Description,
		Config: vm.Config{
			Cpu: v.Cpu,
			Mem: v.Mem,
		},
	}
	err = vm.Create(&vmInst)
	log.Printf("Created VM %v", vmInst.ID)
	if err != nil {
		return &cirrina.VMID{}, err
	}
	return &cirrina.VMID{Value: vmInst.ID}, nil
}

func (s *server) DeleteVM(_ context.Context, v *cirrina.VMID) (*cirrina.RequestID, error) {
	vmInst, err := vm.GetByID(v.Value)
	if err != nil {
		return &cirrina.RequestID{}, err
	}
	if requests.PendingReqExists(v.Value) {
		return &cirrina.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	if vmInst.Status != vm.STOPPED {
		return &cirrina.RequestID{}, errors.New("vm must be stopped before deleting")
	}
	newReq, err := requests.Create(requests.DELETE, v.Value)
	if err != nil {
		return &cirrina.RequestID{}, err
	}
	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) GetVM(_ context.Context, v *cirrina.VMID) (*cirrina.VM, error) {
	var pvm cirrina.VM
	vmInst, err := vm.GetByID(v.Value)
	if err != nil {
		log.Printf("error getting vm %v, %v", v.Value, err)
		return &pvm, err
	}
	pvm.Name = vmInst.Name
	pvm.Description = vmInst.Description
	pvm.Cpu = vmInst.Config.Cpu
	pvm.Mem = vmInst.Config.Mem
	pvm.MaxWait = vmInst.Config.MaxWait
	pvm.Restart = vmInst.Config.Restart
	pvm.RestartDelay = vmInst.Config.RestartDelay
	pvm.Screen = vmInst.Config.Screen
	pvm.ScreenWidth = vmInst.Config.ScreenWidth
	pvm.ScreenHeight = vmInst.Config.ScreenHeight
	return &pvm, nil
}

func (s *server) GetVMs(_ *cirrina.VMsQuery, stream cirrina.VMInfo_GetVMsServer) error {
	var vms []vm.VM
	var pvmId cirrina.VMID

	vms = vm.GetAll()
	for e := range vms {
		pvmId.Value = vms[e].ID
		err := stream.Send(&pvmId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *server) GetVMState(_ context.Context, p *cirrina.VMID) (*cirrina.VMState, error) {
	vmInst, err := vm.GetByID(p.Value)
	pvm := cirrina.VMState{}
	if err != nil {
		log.Printf("error getting vm %v, %v", p.Value, err)
		return &pvm, err
	}
	switch vmInst.Status {
	case vm.STOPPED:
		pvm.Status = cirrina.VmStatus_STATUS_STOPPED
	case vm.STARTING:
		pvm.Status = cirrina.VmStatus_STATUS_STARTING
	case vm.RUNNING:
		pvm.Status = cirrina.VmStatus_STATUS_RUNNING
	case vm.STOPPING:
		pvm.Status = cirrina.VmStatus_STATUS_STOPPING
	default:
		return &pvm, errors.New("unknown VM state")
	}
	return &pvm, nil
}

func (s *server) RequestStatus(_ context.Context, r *cirrina.RequestID) (*cirrina.ReqStatus, error) {
	rs, err := requests.GetByID(r.Value)
	if err != nil {
		return &cirrina.ReqStatus{}, err
	}
	res := &cirrina.ReqStatus{
		Complete: rs.Complete,
		Success:  rs.Successful,
	}
	return res, nil
}

func (s *server) StartVM(_ context.Context, v *cirrina.VMID) (*cirrina.RequestID, error) {
	_, err := vm.GetByID(v.Value)
	if err != nil {
		return &cirrina.RequestID{}, err
	}
	if requests.PendingReqExists(v.Value) {
		return &cirrina.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	vmInst, err := vm.GetByID(v.Value)
	if err != nil {
		log.Printf("error getting vm %v, %v", v.Value, err)
		return &cirrina.RequestID{}, err
	}
	if vmInst.Status != vm.STOPPED {
		return &cirrina.RequestID{}, errors.New("vm must be stopped before starting")
	}
	newReq, err := requests.Create(requests.START, v.Value)
	if err != nil {
		return &cirrina.RequestID{}, err
	}
	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) StopVM(_ context.Context, v *cirrina.VMID) (*cirrina.RequestID, error) {
	_, err := vm.GetByID(v.Value)
	if err != nil {
		return &cirrina.RequestID{}, err
	}
	if requests.PendingReqExists(v.Value) {
		return &cirrina.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	vmInst, err := vm.GetByID(v.Value)
	if err != nil {
		log.Printf("error getting vm %v, %v", v.Value, err)
		return &cirrina.RequestID{}, err
	}
	if vmInst.Status != vm.RUNNING {
		return &cirrina.RequestID{}, errors.New("vm must be running before stopping")
	}
	newReq, err := requests.Create(requests.STOP, v.Value)
	if err != nil {
		return &cirrina.RequestID{}, err
	}
	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) UpdateVM(_ context.Context, rc *cirrina.VMReConfig) (*cirrina.ReqBool, error) {
	re := cirrina.ReqBool{}
	re.Success = false
	vmInst, err := vm.GetByID(rc.Id)
	if err != nil {
		log.Printf("error getting vm %v, %v", rc.Id, err)
		return &cirrina.ReqBool{}, err
	}
	reflect := rc.ProtoReflect()
	if isOptionPassed(reflect, "name") {
		vmInst.Name = *rc.Name
	}
	if isOptionPassed(reflect, "description") {
		vmInst.Description = *rc.Description
	}
	if isOptionPassed(reflect, "cpu") {
		vmInst.Config.Cpu = *rc.Cpu
	}
	if isOptionPassed(reflect, "mem") {
		vmInst.Config.Mem = *rc.Mem
	}
	err = vmInst.Save()
	if err != nil {
		return &re, err
	}
	re.Success = true
	return &re, nil
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
