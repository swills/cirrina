package main

import (
	"bufio"
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
	"os"
	"strings"
)

type server struct {
	cirrina.UnimplementedVMInfoServer
}

func (s *server) AddVM(_ context.Context, v *cirrina.VMConfig) (*cirrina.VMID, error) {
	if _, err := vm.GetByName(*v.Name); err != nil {
		return &cirrina.VMID{}, errors.New(fmt.Sprintf("%v already exists", v.Name))

	}
	defer vm.List.Mu.Unlock()
	vm.List.Mu.Lock()
	vmInst, err := vm.Create(*v.Name, *v.Description, *v.Cpu, *v.Mem)
	if err != nil {
		return &cirrina.VMID{}, err
	}
	vm.List.VmList[vmInst.ID] = vmInst
	return &cirrina.VMID{Value: vmInst.ID}, nil
}

func (s *server) DeleteVM(_ context.Context, v *cirrina.VMID) (*cirrina.RequestID, error) {
	vmInst, err := vm.GetById(v.Value)
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
	vmInst, err := vm.GetById(v.Value)
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
	pvm.Keyboard = &vmInst.Config.KbdLayout
	pvm.Autostart = &vmInst.Config.AutoStart
	NetTypeVIRTIONET := cirrina.NetType_VIRTIONET
	NetTypeE1000 := cirrina.NetType_E1000
	if vmInst.Config.NetType == "VIRTIONET" {
		pvm.Nettype = &NetTypeVIRTIONET
	} else if vmInst.Config.NetType == "E1000" {
		pvm.Nettype = &NetTypeE1000
	}
	NetDevTypeTAP := cirrina.NetDevType_TAP
	NetDevTypeVMNET := cirrina.NetDevType_VMNET
	NetDevTypeNETGRAPH := cirrina.NetDevType_NETGRAPH
	if vmInst.Config.NetDevType == "TAP" {
		pvm.Netdevtype = &NetDevTypeTAP
	} else if vmInst.Config.NetDevType == "VMNET" {
		pvm.Netdevtype = &NetDevTypeVMNET
	} else if vmInst.Config.NetDevType == "NETGRAPH" {
		pvm.Netdevtype = &NetDevTypeNETGRAPH
	}
	return &pvm, nil
}

func (s *server) GetVMs(_ *cirrina.VMsQuery, stream cirrina.VMInfo_GetVMsServer) error {
	var vms []*vm.VM
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
	vmInst, err := vm.GetById(p.Value)
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
	_, err := vm.GetById(v.Value)
	if err != nil {
		return &cirrina.RequestID{}, err
	}
	if requests.PendingReqExists(v.Value) {
		return &cirrina.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	vmInst, err := vm.GetById(v.Value)
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
	_, err := vm.GetById(v.Value)
	if err != nil {
		return &cirrina.RequestID{}, err
	}
	if requests.PendingReqExists(v.Value) {
		return &cirrina.RequestID{}, errors.New(fmt.Sprintf("pending request for %v already exists", v.Value))
	}
	vmInst, err := vm.GetById(v.Value)
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

func (s *server) GetKeyboardLayouts(_ *cirrina.KbdQuery, stream cirrina.VMInfo_GetKeyboardLayoutsServer) error {
	var kbdlayoutpath = "/usr/share/bhyve/kbdlayout"
	var layout cirrina.KbdLayout

	files, err := OSReadDir(kbdlayoutpath)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		layout.Name = file
		if file == "default" {
			layout.Description = "default"
		} else {
			layout.Description, err = getKbdDescription(kbdlayoutpath + "/" + file)
			if err != nil {
				return err
			}
		}
		err = stream.Send(&layout)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *server) UpdateVM(_ context.Context, rc *cirrina.VMConfig) (*cirrina.ReqBool, error) {
	re := cirrina.ReqBool{}
	re.Success = false
	vmInst, err := vm.GetById(rc.Id)
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
	if isOptionPassed(reflect, "restart_delay") {
		vmInst.Config.RestartDelay = *rc.RestartDelay
	}

	if isOptionPassed(reflect, "net") {
		if *rc.Net {
			vmInst.Config.Net = true
		} else {
			vmInst.Config.Net = false
		}
	}
	if isOptionPassed(reflect, "screen_width") {
		vmInst.Config.ScreenWidth = *rc.ScreenWidth
	}
	if isOptionPassed(reflect, "screen_height") {
		vmInst.Config.ScreenHeight = *rc.ScreenHeight
	}
	if isOptionPassed(reflect, "vncport") {
		vmInst.Config.VNCPort = *rc.Vncport
	}
	if isOptionPassed(reflect, "mac") {
		// TODO -- validate mac
		vmInst.Config.Mac = *rc.Mac
	}
	if isOptionPassed(reflect, "keyboard") {
		vmInst.Config.KbdLayout = *rc.Keyboard
	}
	if isOptionPassed(reflect, "autostart") {
		if *rc.Autostart == true {
			vmInst.Config.AutoStart = true
		} else {
			vmInst.Config.AutoStart = false
		}
	}
	if isOptionPassed(reflect, "netdevtype") {
		if *rc.Netdevtype == 0 {
			vmInst.Config.NetDevType = "TAP"
		} else if *rc.Netdevtype == 1 {
			vmInst.Config.NetDevType = "VMNET"
		} else if *rc.Netdevtype == 2 {
			vmInst.Config.NetDevType = "NETGRAPH"
		}
	}
	if isOptionPassed(reflect, "nettype") {
		if *rc.Nettype == 0 {
			vmInst.Config.NetType = "VIRTIONET"
		} else if *rc.Nettype == 1 {
			vmInst.Config.NetType = "E1000"
		}
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

func OSReadDir(root string) ([]string, error) {
	var files []string
	f, err := os.Open(root)
	if err != nil {
		return files, err
	}
	fileInfo, err := f.Readdir(-1)
	_ = f.Close()
	if err != nil {
		return files, err
	}

	for _, file := range fileInfo {
		files = append(files, file.Name())
	}
	return files, nil
}

func getKbdDescription(path string) (description string, err error) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	lineNo := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineNo += 1
		if lineNo > 2 {
			continue
		}
		if lineNo == 2 {
			de := strings.Split(scanner.Text(), ":")
			if len(de) > 1 {
				desc := strings.TrimSpace(de[1])
				description = strings.TrimSuffix(desc, ")")
			} else {
				description = "unknown"
			}

		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}

	return description, nil
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
