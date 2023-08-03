package main

import (
	pb "cirrina/cirrina"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"time"
)

var serverAddr string

var myTableStyle = table.Style{
	Name: "myNewStyle",
	Box: table.BoxStyle{
		MiddleHorizontal: "-", // bug in go-pretty causes panic if this is empty
		PaddingRight:     "  ",
	},
	Format: table.FormatOptions{
		Footer: text.FormatUpper,
		Header: text.FormatUpper,
		Row:    text.FormatDefault,
	},
	Options: table.Options{
		DrawBorder:      false,
		SeparateColumns: false,
		SeparateFooter:  false,
		SeparateHeader:  false,
		SeparateRows:    false,
	},
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(fflag *flag.Flag) {
		if fflag.Name == name {
			found = true
		}
	})
	return found
}

func printActionHelp() {
	println("Actions: getVM, getVMs, getVMState, addVM, reConfig, deleteVM, reqStat, startVM, stopVM, " +
		"addISO, addDisk, addSwitch, addVmNic, getSwitches, getDisks, getVmNics, getSwitch, getVmNic, setVmNicVm, " +
		"setVmNicSwitch, rmSwitch, getHostNics, setSwitchUplink, uploadIso, useCom1, useCom2, useCom3, useCom4, tui")
}

func main() {
	serverAddrPtr := flag.String("server", "localhost", "name/ip of server")
	actionPtr := flag.String("action", "", "action to take")
	idPtr := flag.String("id", "", "ID of VM")
	namePtr := flag.String("name", "", "Name of VM/ISO/Disk")
	descrPtr := flag.String("descr", "", "Description of VM/ISO/Disk")
	sizePtr := flag.String("size", "", "Size of Disk")
	switchTypePtr := flag.String("switchType", "IF", "Type of switch (IF or NG)")
	nicIdPtr := flag.String("nicId", "", "ID of Nic")
	switchIdPtr := flag.String("switchId", "", "ID of Switch")
	uplinkNamePtr := flag.String("uplinkName", "value", "name of switch uplink")
	cpuPtr := flag.Uint("cpus", 1, "Number of CPUs in VM")
	cpuVal := *cpuPtr
	cpu32Val := uint32(cpuVal)
	cpu32Ptr := &cpu32Val
	memPtr := flag.Uint("mem", 128, "Memory in VM (MB)")
	memVal := *memPtr
	mem32Val := uint32(memVal)
	mem32Ptr := &mem32Val
	autoStartPtr := flag.Bool("autostart", false, "automatically start the VM")
	netTypePtr := flag.String("netType", "VIRTIONET", "Type of net (VIRTIONET or E1000")
	netDevTypePtr := flag.String("netDevType", "TAP", "type of net dev (TAP, VMNET or NETGRAPH")
	diskTypePtr := flag.String("diskType", "NVME", "Type of disk dev (NVME, AHCIHD, or VIRTIOBLK")
	macPtr := flag.String("mac", "AUTO", "Mac address of NIC (or AUTO)")
	filePathPtr := flag.String("filePath", "", "path to iso or disk file")
	//maxWaitPtr := flag.Uint("maxWait", 120, "Max wait time for VM shutdown")
	//restartPtr := flag.Bool("restart", true, "Automatically restart VM")
	//restartDelayPtr := flag.Uint("restartDelay", 1, "How long to wait before restarting VM")
	//screenPtr := flag.Bool("screen", true, "Should the VM have a screen (frame buffer)")
	//screenWidthPtr := flag.Uint("screenWidth", 1920, "Width of VM screen")
	//screenHeightPtr := flag.Uint(screenHeight, 1080, "Height of VM screen")

	flag.Parse()
	serverAddr = *serverAddrPtr + ":50051"
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Fatalf("Failed to close connection")
		}
	}(conn)
	c := pb.NewVMInfoClient(conn)

	// Contact the server and print out its response.
	timeout := time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	arg0 := flag.Arg(0)
	switch arg0 {
	case "list":
		getVMs(c, ctx)
		return
	case "switch":
		arg1 := flag.Arg(1)
		switch arg1 {
		case "list":
			getSwitches(c, ctx)
			return
		}
	case "nic":
		arg1 := flag.Arg(1)
		switch arg1 {
		case "list":
			getVmNicsAll(c, ctx)
			return
		}
	case "disk":
		arg1 := flag.Arg(1)
		switch arg1 {
		case "list":
			getDisks(c, ctx)
		}
	case "start":
		arg1 := flag.Arg(1)
		if arg1 == "" {
			usage()
			return
		}
		startVM(arg1, c, ctx, err)
		return
	case "stop":
		arg1 := flag.Arg(1)
		if &arg1 == nil {
			usage()
			return
		}
		stopVM(arg1, c, ctx, err)
		return
	}

	switch *actionPtr {
	case "":
		log.Fatalf("Action not specified, try \"help\"")
	case "help":
		printActionHelp()
	case "getVM":
		getVM(idPtr, c, ctx)
	case "getSwitch":
		getSwitch(idPtr, c, ctx)
	case "getVmNic":
		getVmNic(idPtr, c, ctx)
	case "getVMs":
		getVMs(c, ctx)
	case "getDisks":
		getDisks(c, ctx)
	case "getSwitches":
		getSwitches(c, ctx)
	case "getVmNics":
		getVmNics(c, ctx, idPtr)
	case "setVmNicVm":
		setVmNicVm(c, ctx)
	case "setVmNicSwitch":
		setVmNicSwitch(c, ctx, *nicIdPtr, *switchIdPtr)
	case "getVMState":
		getVMState(idPtr, c, ctx)
	case "addVM":
		addVM(namePtr, c, ctx, descrPtr, cpu32Ptr, mem32Ptr)
	case "addISO":
		addISO(namePtr, c, ctx, descrPtr)
	case "addDisk":
		addDisk(namePtr, c, ctx, descrPtr, sizePtr, diskTypePtr)
	case "addSwitch":
		addSwitch(namePtr, c, ctx, descrPtr, switchTypePtr)
	case "rmSwitch":
		rmSwitch(idPtr, c, ctx)
	case "addVmNic":
		addVmNic(namePtr, c, ctx, descrPtr, netTypePtr, netDevTypePtr, macPtr, switchIdPtr)
	case "rmVmNic":
		rmVmNic(idPtr, c, ctx)
	case "reConfig":
		Reconfig(idPtr, err, namePtr, descrPtr, cpuPtr, memPtr, autoStartPtr, c, ctx)
	case "deleteVM":
		DeleteVM(idPtr, c, ctx)
	case "reqStat":
		ReqStat(idPtr, c, ctx)
	case "startVM":
		reqId, err := rpcStartVM(idPtr, c, ctx)
		if err != nil {
			log.Fatalf("could not start VM: %v", err)
		}
		fmt.Printf("Started request created, reqid: %v\n", reqId)
	case "stopVM":
		reqId, err := rpcStopVM(idPtr, c, ctx)
		if err != nil {
			log.Fatalf("could not stop VM: %v", err)
		}
		fmt.Printf("Stopping request created, reqid: %v\n", reqId)
	case "getHostNics":
		getHostNics(c, ctx)
	case "setSwitchUplink":
		setSwitchUplink(c, ctx, switchIdPtr, uplinkNamePtr)
	case "uploadIso":
		timeout := time.Hour
		longCtx, longCancel := context.WithTimeout(context.Background(), timeout)
		defer longCancel()
		uploadIso(c, longCtx, idPtr, filePathPtr)
	case "useCom1":
		useCom(c, idPtr, 1)
	case "useCom2":
		useCom(c, idPtr, 2)
	case "useCom3":
		useCom(c, idPtr, 3)
	case "useCom4":
		useCom(c, idPtr, 4)
	case "tui":
		startTui()
	default:
		log.Fatalf("Action %v unknown", *actionPtr)
	}
}

func startVM(arg1 string, c pb.VMInfoClient, ctx context.Context, err error) string {
	vmId, err := getVmIdByName(&arg1, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s\n", arg1, err)
		return ""
	}
	res2, err := c.GetVMState(ctx, &pb.VMID{Value: vmId})
	if err != nil {
		log.Fatalf("could not get VM state: %v", err)
		return ""
	}

	if res2.Status != pb.VmStatus_STATUS_STOPPED {
		fmt.Printf("error: request to start VM »%s« failed: VM must be stopped in order to be started\n", arg1)
		return ""
	}
	reqId, err := rpcStartVM(&vmId, c, ctx)
	return reqId
}

func stopVM(arg1 string, c pb.VMInfoClient, ctx context.Context, err error) bool {
	vmId, err := getVmIdByName(&arg1, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s", arg1, err)
		return true
	}
	res2, err := c.GetVMState(ctx, &pb.VMID{Value: vmId})
	if err != nil {
		log.Fatalf("could not get VM state: %v", err)
		return true
	}

	if res2.Status != pb.VmStatus_STATUS_RUNNING {
		fmt.Printf("error: request to stop VM »%s« failed: VM must be running in order to be stopped\n", arg1)
		return true
	}
	_, err = rpcStopVM(&vmId, c, ctx)
	if err != nil {
		fmt.Printf("error: could not find VM »%s«: %s", arg1, err)
	}

	return false
}

func getVmIdByName(s *string, c pb.VMInfoClient, ctx context.Context) (string, error) {
	vmList, err := getVmIds(c, ctx)
	if err != nil {
		fmt.Printf(err.Error())
	}
	found := false
	rv := ""

	for _, id := range vmList {
		res, err := c.GetVMConfig(ctx, &pb.VMID{Value: id})
		if err != nil {
			em := fmt.Sprintf("could not get VM: %s", err)
			return rv, errors.New(em)
		}
		if *res.Name == *s {
			if found {
				em := fmt.Sprintf("duplicate names found")
				return rv, errors.New(em)
			} else {
				found = true
				rv = id
			}
		}

	}

	return rv, nil
}

func usage() {
	fmt.Printf("Usage: %s", "todo")
}
