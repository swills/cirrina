package main

import (
	pb "cirrina/cirrina"
	"context"
	"flag"
	"fmt"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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

	timeout := time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// new style arg parsing
	arg0 := flag.Arg(0)
	switch arg0 {
	case "list":
		getVMs(c, ctx)
		return
	case "switch":
		arg1 := flag.Arg(1)
		if arg1 == "" {
			fmt.Printf("error: missing subcommand\n")
			switchUsage()
			return
		}
		switch arg1 {
		case "list":
			getSwitches(c, ctx)
			return
		case "set-uplink":
			switchName := flag.Arg(2)
			uplinkName := flag.Arg(3)
			if switchName == "" {
				fmt.Printf("error: bad arguments\n")
				fmt.Printf("usage: set-uplink <switch> <uplink>\n")
				switchUsage()
				return
			}

			switchId, err := getSwitchByName(&switchName, c, ctx)
			if err != nil || switchId == "" {
				fmt.Printf("error: could not find switch: no switch with the given name found\n")
				return
			}
			err = setSwitchUplink(c, ctx, &switchId, &uplinkName)
			if err != nil {
				fmt.Printf("error: could not set switch uplink: %s\n", err.Error())
				return
			}

			return
		case "create":
			name := ""
			description := ""
			switchType := "IF"
			//uplink := ""
			argCount := flag.NArg()
			for argNum, argval := range flag.Args() {
				if argNum < 2 {
					continue
				}
				switch argval {
				case "--name":
					fallthrough
				case "-n":
					argNum = argNum + 1
					if argCount < argNum+1 {
						fmt.Printf("option requires an argument -- %s\n", argval)
						return
					}
					name = flag.Arg(argNum)
					if name == "" {
						fmt.Printf("name cannot be empty\n")
						return
					}
				case "--description":
					fallthrough
				case "-d":
					argNum = argNum + 1
					if argCount < argNum+1 {
						fmt.Printf("option requires an argument -- %s\n", argval)
						return
					}
					description = flag.Arg(argNum)
				case "--type":
					fallthrough
				case "-t":
					argNum = argNum + 1
					if argCount < argNum+1 {
						fmt.Printf("option requires an argument -- %s\n", argval)
						return
					}
					switchType = flag.Arg(argNum)
					if switchType != "bridge" && switchType != "netgraph" {
						fmt.Printf("error: invalid nettype. expected one of the following:\n" +
							"\t- e1000\n\t- virtio-net\n")
						return
					}
					//case "--uplink":
					//	fallthrough
					//case "-u":
					//	argNum = argNum + 1
					//	if argCount < argNum+1 {
					//		fmt.Printf("option requires an argument -- %s\n", argval)
					//		return
					//	}
					//	uplink = flag.Arg(argNum)
				}
			}
			_, err := addSwitch(&name, c, ctx, &description, &switchType)
			if err != nil {
				s := status.Convert(err)
				fmt.Printf("error: could not create a new switch: %s\n", s.Message())
				return
			}
			return
		case "destroy":
			name := ""
			argCount := flag.NArg()
			for argNum, argval := range flag.Args() {
				if argNum < 2 {
					continue
				}
				switch argval {
				case "--name":
					fallthrough
				case "-n":
					argNum = argNum + 1
					if argCount < argNum+1 {
						fmt.Printf("option requires an argument -- %s\n", argval)
						return
					}
					name = flag.Arg(argNum)
					if name == "" {
						fmt.Printf("name cannot be empty\n")
						return
					}
				}
			}
			switchId, err := getSwitchByName(&name, c, ctx)
			if err != nil {
				s := status.Convert(err)
				fmt.Printf("error: could not delete switch: %s\n", s.Message())
				return
			}
			if switchId == "" {
				fmt.Printf("error: could not find switch: no switch with the given name found\n")
				return
			}
			err = rmSwitch(&switchId, c, ctx)
			if err != nil {
				s := status.Convert(err)
				fmt.Printf("error: could not delete switch: %s\n", s.Message())
				return
			}
			return
		}
	case "nic":
		arg1 := flag.Arg(1)
		switch arg1 {
		case "list":
			getVmNicsAll(c, ctx)
			return
		case "create":
			name := ""
			description := ""
			nettype := "virtio-net"
			netdevtype := "tap"
			mac := "AUTO"
			switchId := ""
			argCount := flag.NArg()
			for argNum, argval := range flag.Args() {
				if argNum < 2 {
					continue
				}
				switch argval {
				case "--name":
					fallthrough
				case "-n":
					argNum = argNum + 1
					if argCount < argNum+1 {
						fmt.Printf("option requires an argument -- %s\n", argval)
						return
					}
					name = flag.Arg(argNum)
					if name == "" {
						fmt.Printf("name cannot be empty\n")
						return
					}
				case "--description":
					fallthrough
				case "-d":
					argNum = argNum + 1
					if argCount < argNum+1 {
						fmt.Printf("option requires an argument -- %s\n", argval)
						return
					}
					description = flag.Arg(argNum)
				case "--nettype":
					fallthrough
				case "-t":
					argNum = argNum + 1
					if argCount < argNum+1 {
						fmt.Printf("option requires an argument -- %s\n", argval)
						return
					}
					nettype = flag.Arg(argNum)
					if nettype != "e1000" && nettype != "virtio-net" {
						fmt.Printf("error: invalid nettype. expected one of the following:\n" +
							"\t- e1000\n\t- virtio-net\n")
						return
					}
				case "--netdevtype":
					fallthrough
				case "-v":
					argNum = argNum + 1
					if argCount < argNum+1 {
						fmt.Printf("option requires an argument -- %s\n", argval)
						return
					}
					netdevtype = flag.Arg(argNum)
					if netdevtype != "netgraph" && netdevtype != "vmnet" && netdevtype != "tap" {
						fmt.Printf("error: invalid netdevtype. expected one of the following:\n" +
							"\t- netgraph\n\t- vmnet\n\t- tap\n")
						return
					}
				case "--mac":
					fallthrough
				case "-m":
					argNum = argNum + 1
					if argCount < argNum+1 {
						fmt.Printf("option requires an argument -- %s\n", argval)
						return
					}
					mac = flag.Arg(argNum)
				}

			}
			_, err := addVmNic(&name, c, ctx, &description, &nettype, &netdevtype, &mac, &switchId)
			if err != nil {
				s := status.Convert(err)
				fmt.Printf("error: could not create a new NIC: %s\n", s.Message())
				return
			}
			return
		case "destroy":
			name := ""
			argCount := flag.NArg()
			for argNum, argval := range flag.Args() {
				if argNum < 2 {
					continue
				}
				switch argval {
				case "--name":
					fallthrough
				case "-n":
					argNum = argNum + 1
					if argCount < argNum+1 {
						fmt.Printf("option requires an argument -- %s\n", argval)
						return
					}
					name = flag.Arg(argNum)
					if name == "" {
						fmt.Printf("name cannot be empty\n")
						return
					}
				}
			}
			nicId, err := getNicByName(&name, c, ctx)
			if err != nil {
				s := status.Convert(err)
				fmt.Printf("error: could not delete nic: %s\n", s.Message())
				return
			}
			err = deleteNic(&nicId, c, ctx)
			if err != nil {
				s := status.Convert(err)
				fmt.Printf("error: could not delete nic: %s\n", s.Message())
				return
			}
			return
		}
	case "disk":
		arg1 := flag.Arg(1)
		switch arg1 {
		case "list":
			getDisks(c, ctx)
			return
		case "create":
			name := ""
			description := ""
			diskType := "nvme"
			diskSize := "1G"
			argCount := flag.NArg()
			for argNum, argval := range flag.Args() {
				if argNum < 2 {
					continue
				}
				switch argval {
				case "--name":
					fallthrough
				case "-n":
					argNum = argNum + 1
					if argCount < argNum+1 {
						fmt.Printf("option requires an argument -- %s\n", argval)
						return
					}
					name = flag.Arg(argNum)
					if name == "" {
						fmt.Printf("name cannot be empty\n")
						return
					}
				case "--description":
					fallthrough
				case "-d":
					argNum = argNum + 1
					if argCount < argNum+1 {
						fmt.Printf("option requires an argument -- %s\n", argval)
						return
					}
					description = flag.Arg(argNum)
				case "--type":
					fallthrough
				case "-t":
					argNum = argNum + 1
					if argCount < argNum+1 {
						fmt.Printf("option requires an argument -- %s\n", argval)
						return
					}
					diskType = flag.Arg(argNum)
				case "--size":
					fallthrough
				case "-s":
					argNum = argNum + 1
					if argCount < argNum+1 {
						fmt.Printf("option requires an argument -- %s\n", argval)
						return
					}
					diskSize = flag.Arg(argNum)
				}
			}
			_, err := addDisk(&name, c, ctx, &description, &diskSize, &diskType)
			if err != nil {
				s := status.Convert(err)
				fmt.Printf("error: could not create a new disk: %s\n", s.Message())
				return
			}
			return
		case "destroy":
			name := ""
			argCount := flag.NArg()
			for argNum, argval := range flag.Args() {
				if argNum < 2 {
					continue
				}
				switch argval {
				case "--name":
					fallthrough
				case "-n":
					argNum = argNum + 1
					if argCount < argNum+1 {
						fmt.Printf("option requires an argument -- %s\n", argval)
						return
					}
					name = flag.Arg(argNum)
					if name == "" {
						fmt.Printf("name cannot be empty\n")
						return
					}
				}
			}
			diskId, err := getDiskByName(&name, c, ctx)
			if err != nil {
				s := status.Convert(err)
				fmt.Printf("error: could not delete disk: %s\n", s.Message())
				return
			}
			err = deleteDisk(&diskId, c, ctx)
			if err != nil {
				s := status.Convert(err)
				fmt.Printf("error: could not delete disk: %s\n", s.Message())
				return
			}
			return
		}
	case "start":
		arg1 := flag.Arg(1)
		if arg1 == "" {
			fmt.Printf("error: stray arguments or missing VM name\n")
			usage()
			return
		}
		startVM(arg1, c, ctx, err)
		return
	case "stop":
		arg1 := flag.Arg(1)
		if &arg1 == nil {
			fmt.Printf("error: stray arguments or missing VM name\n")
			usage()
			return
		}
		stopVM(arg1, c, ctx, err)
		return
	case "help":
		usage()
	}

	// old style arg parsing
	switch *actionPtr {
	case "":
		log.Fatalf("Action not specified, try \"help\"")
	case "help":
		printActionHelp()

	// VMs
	case "addVM":
		addVM(namePtr, c, ctx, descrPtr, cpu32Ptr, mem32Ptr)
	case "deleteVM":
		DeleteVM(idPtr, c, ctx)
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
	case "getVM":
		getVM(idPtr, c, ctx)
	case "getVMs":
		getVMs(c, ctx)
	case "getVMState":
		getVMState(idPtr, c, ctx)
	case "reConfig":
		Reconfig(idPtr, err, namePtr, descrPtr, cpuPtr, memPtr, autoStartPtr, c, ctx)

	// Disks
	case "getDisks":
		getDisks(c, ctx)
	case "addDisk":
		diskId, err := addDisk(namePtr, c, ctx, descrPtr, sizePtr, diskTypePtr)
		if err != nil {
			fmt.Printf("failed to create disk: %s", err)
		}
		fmt.Printf("Created Disk %v\n", diskId)

	// CDs
	case "addISO":
		addISO(namePtr, c, ctx, descrPtr)
	case "uploadIso":
		timeout := time.Hour
		longCtx, longCancel := context.WithTimeout(context.Background(), timeout)
		defer longCancel()
		uploadIso(c, longCtx, idPtr, filePathPtr)

	// NICs
	case "addVmNic":
		nicId, err := addVmNic(namePtr, c, ctx, descrPtr, netTypePtr, netDevTypePtr, macPtr, switchIdPtr)
		if err != nil {
			fmt.Printf("could not create nic: %v\n", err)
		}
		fmt.Printf("Created vmnic %v\n", nicId)
	case "rmVmNic":
		rmVmNic(idPtr, c, ctx)
	case "getHostNics":
		getHostNics(c, ctx)
	case "getVmNic":
		getVmNic(idPtr, c, ctx)
	case "getVmNics":
		getVmNics(c, ctx, idPtr)
	case "setVmNicVm":
		setVmNicVm(c, ctx)
	case "setVmNicSwitch":
		setVmNicSwitch(c, ctx, *nicIdPtr, *switchIdPtr)

	// Switches
	case "addSwitch":
		res, err := addSwitch(namePtr, c, ctx, descrPtr, switchTypePtr)
		if err != nil {
			log.Fatalf("could not create switch: %v", err)
		}
		fmt.Printf("Created switch %v\n", res)
	case "getSwitch":
		res, err := getSwitch(idPtr, c, ctx)
		if err != nil {
			log.Fatalf("could not get switch: %v", err)
		}
		fmt.Printf(
			"name: %v "+
				"description: %v "+
				"type: %v "+
				"uplink: %v"+
				"\n",
			*res.Name,
			*res.Description,
			*res.SwitchType,
			*res.Uplink,
		)
	case "getSwitches":
		getSwitches(c, ctx)
	case "rmSwitch":
		err := rmSwitch(idPtr, c, ctx)
		if err == nil {
			fmt.Printf("Delete successful")
		} else {
			fmt.Printf("Delete failed")
		}
	case "setSwitchUplink":
		err = setSwitchUplink(c, ctx, switchIdPtr, uplinkNamePtr)
		if err == nil {
			fmt.Printf("Switch uplink set successful")
		} else {
			fmt.Printf("Switch uplink set failed")
		}

	// Serial ports
	case "useCom1":
		useCom(c, idPtr, 1)
	case "useCom2":
		useCom(c, idPtr, 2)
	case "useCom3":
		useCom(c, idPtr, 3)
	case "useCom4":
		useCom(c, idPtr, 4)

	// Misc
	case "reqStat":
		ReqStat(idPtr, c, ctx)
	case "tui":
		startTui()

	default:
		log.Fatalf("Action %v unknown", *actionPtr)
	}
}

func usage() {
	usageString := `usage: cirrinactl [global options] [subcommand]
OPTIONS:
  -h <host>       Connect to the given host [localhost]
  -p <port>       Connect to the given port [50051]
  -c <config>     Read a config from the given file

SUBCOMMANDS:
   list           List VMs
   switch         Inspect, create, update and delete switches
   nic            Inspect, create, update and delete NICs
   disk           Inspect, create, update and delete virtual disks
   start          Start a VM
   stop           Stop a VM
`
	fmt.Printf(usageString)
}

func switchUsage() {
	usageString := `usage: cirrinactl switch [subcommand]
OPTIONS:
   None.

SUBCOMMANDS:
   list           List switches
   set-uplink     Set a switch uplink interface
`
	fmt.Printf(usageString)
}
