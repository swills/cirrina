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

func (s *server) AddVM(_ context.Context, v *cirrina.VMConfig) (*cirrina.VMID, error) {
	_, err := vm.GetByName(*v.Name)
	if err == nil {
		return &cirrina.VMID{}, errors.New(fmt.Sprintf("%v already exists", v.Name))

	}
	vmInst := vm.VM{
		Name:        *v.Name,
		Status:      vm.STOPPED,
		Description: *v.Description,
		Config: vm.Config{
			Cpu: *v.Cpu,
			Mem: *v.Mem,
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

func (s *server) GetVMConfig(_ context.Context, v *cirrina.VMID) (*cirrina.VMConfig, error) {
	var pvm cirrina.VMConfig
	vmInst, err := vm.GetByID(v.Value)
	if err != nil {
		log.Printf("error getting vm %v, %v", v.Value, err)
		return &pvm, err
	}
	pvm.Name = &vmInst.Name
	pvm.Description = &vmInst.Description
	pvm.Cpu = &vmInst.Config.Cpu
	pvm.Mem = &vmInst.Config.Mem
	pvm.MaxWait = &vmInst.Config.MaxWait
	pvm.Restart = &vmInst.Config.Restart
	pvm.RestartDelay = &vmInst.Config.RestartDelay
	pvm.Screen = &vmInst.Config.Screen
	pvm.ScreenWidth = &vmInst.Config.ScreenWidth
	pvm.ScreenHeight = &vmInst.Config.ScreenHeight
	pvm.Vncwait = &vmInst.Config.VNCWait
	pvm.Vncport = &vmInst.Config.VNCPort
	pvm.Wireguestmem = &vmInst.Config.WireGuestMem
	pvm.Tablet = &vmInst.Config.Tablet
	pvm.Storeuefi = &vmInst.Config.StoreUEFIVars
	pvm.Utc = &vmInst.Config.UTCTime
	pvm.Hostbridge = &vmInst.Config.HostBridge
	pvm.Acpi = &vmInst.Config.ACPI
	pvm.Hlt = &vmInst.Config.UseHLT
	pvm.Eop = &vmInst.Config.ExitOnPause
	pvm.Dpo = &vmInst.Config.DestroyPowerOff
	pvm.Ium = &vmInst.Config.IgnoreUnknownMSR
	pvm.Net = &vmInst.Config.Net
	pvm.Vncport = &vmInst.Config.VNCPort
	pvm.Mac = &vmInst.Config.Mac
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
		pvm.VncPort = vmInst.VNCPort
	case vm.RUNNING:
		pvm.Status = cirrina.VmStatus_STATUS_RUNNING
		pvm.VncPort = vmInst.VNCPort
	case vm.STOPPING:
		pvm.Status = cirrina.VmStatus_STATUS_STOPPING
		pvm.VncPort = vmInst.VNCPort
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

func (s *server) UpdateVM(_ context.Context, rc *cirrina.VMConfig) (*cirrina.ReqBool, error) {
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
	if isOptionPassed(reflect, "vncwait") {
		if *rc.Vncwait == true {
			vmInst.Config.VNCWait = true
		} else {
			vmInst.Config.VNCWait = false
		}
	}
	if isOptionPassed(reflect, "acpi") {
		if *rc.Acpi == true {
			vmInst.Config.ACPI = true
		} else {
			vmInst.Config.ACPI = false
		}
	}
	if isOptionPassed(reflect, "utc") {
		if *rc.Utc == true {
			vmInst.Config.UTCTime = true
		} else {
			vmInst.Config.UTCTime = false
		}
	}
	if isOptionPassed(reflect, "max_wait") {
		vmInst.Config.MaxWait = *rc.MaxWait
	}

	if isOptionPassed(reflect, "tablet") {
		if *rc.Tablet == true {
			vmInst.Config.Tablet = true
		} else {
			vmInst.Config.Tablet = false
		}
	}

	if isOptionPassed(reflect, "storeuefi") {
		if *rc.Storeuefi == true {
			vmInst.Config.StoreUEFIVars = true
		} else {
			vmInst.Config.StoreUEFIVars = false
		}
	}

	if isOptionPassed(reflect, "wireguestmem") {
		if *rc.Storeuefi == true {
			vmInst.Config.WireGuestMem = true
		} else {
			vmInst.Config.WireGuestMem = false
		}
	}

	if isOptionPassed(reflect, "restart") {
		if *rc.Restart == true {
			vmInst.Config.Restart = true
		} else {
			vmInst.Config.Restart = false
		}
	}

	if isOptionPassed(reflect, "screen") {
		if *rc.Screen == true {
			vmInst.Config.Screen = true
		} else {
			vmInst.Config.Screen = false
		}
	}

	if isOptionPassed(reflect, "hlt") {
		if *rc.Hlt == true {
			vmInst.Config.UseHLT = true
		} else {
			vmInst.Config.UseHLT = false
		}
	}

	if isOptionPassed(reflect, "eop") {
		if *rc.Eop == true {
			vmInst.Config.ExitOnPause = true
		} else {
			vmInst.Config.ExitOnPause = false
		}
	}

	if isOptionPassed(reflect, "dpo") {
		if *rc.Dpo == true {
			vmInst.Config.DestroyPowerOff = true
		} else {
			vmInst.Config.DestroyPowerOff = false
		}
	}

	if isOptionPassed(reflect, "ium") {
		if *rc.Ium == true {
			vmInst.Config.IgnoreUnknownMSR = true
		} else {
			vmInst.Config.IgnoreUnknownMSR = false
		}
	}
	if isOptionPassed(reflect, "hostbridge") {
		if *rc.Hostbridge == true {
			vmInst.Config.HostBridge = true
		} else {
			vmInst.Config.HostBridge = false
		}
	}

	//	optional bool net = 23;

	//	optional uint32 restart_delay = 8;
	//	optional uint32 screen_width = 10;
	//	optional uint32 screen_height = 11;
	//	optional string vncport = 24;
	//	optional string mac = 25;

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
