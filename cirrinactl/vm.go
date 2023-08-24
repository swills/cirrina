package main

import (
	"cirrina/cirrina"
	"cirrina/cirrinactl/rpc"
	"context"
	"fmt"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"log"
	"os"
)

func getVM(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	res, err := rpc.GetVMConfig(idPtr, c, ctx)
	if err != nil {
		log.Fatalf("Error getting VM: %v\n", err.Error())
	}
	// TODO JSON output
	fmt.Printf(
		"name: %v "+
			"\ndesc: %v "+
			"\ncpus: %v "+
			"\nmem: %v "+
			"\nvncWait: %v "+
			"\nwire guest mem: %v "+
			"\ntablet mode: %v "+
			"\nstore uefi vars: %v "+
			"\nuse utc time: %v "+
			"\nuse host bridge: %v "+
			"\ngenerate acpi tables: %v "+
			"\nyield on HLT: %v "+
			"\nexit on PAUSE: %v "+
			"\ndestroy on power off: %v "+
			"\nignore unknown msr: %v "+
			"\nvnc port: %v "+
			"\nauto start: %v"+
			"\n",
		*res.Name,
		*res.Description,
		*res.Cpu,
		*res.Mem,
		*res.Vncwait,
		*res.Wireguestmem,
		*res.Tablet,
		*res.Storeuefi,
		*res.Utc,
		*res.Hostbridge,
		*res.Acpi,
		*res.Hlt,
		*res.Eop,
		*res.Dpo,
		*res.Ium,
		*res.Vncport,
		*res.Autostart,
	)
}

func getVMs(c cirrina.VMInfoClient, ctx context.Context) {
	ids, err := rpc.GetVmIds(c, ctx)
	if err != nil {
		log.Fatalf(err.Error())
	}
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{"NAME", "UUID", "CPUS", "MEMORY", "STATE", "DESCRIPTION"})
	t.SetStyle(table.Style{
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
	})
	for _, id := range ids {
		res, err := rpc.GetVMConfig(&id, c, ctx)
		if err != nil {
			log.Fatalf("could not get VM: %v", err)
			return
		}
		status, err := rpc.GetVMState(&id, c, ctx)
		if err != nil {
			log.Fatalf("could not get VM: %v", err)
			return
		}

		t.AppendRow(table.Row{
			*res.Name,
			id,
			*res.Cpu,
			*res.Mem,
			status,
			*res.Description,
		})
	}

	t.Render()
}

func startVM(arg1 string, c cirrina.VMInfoClient, ctx context.Context, err error) string {
	vmId, err := rpc.VmNameToId(arg1, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s\n", arg1, err)
		return ""
	}
	stopped, err := rpc.VmStopped(&vmId, c, ctx)
	if err != nil {
		log.Fatalf("could not get VM state: %v", err)
		return ""
	}
	if !stopped {
		fmt.Printf("error: request to start VM »%s« failed: VM must be stopped in order to be started\n", arg1)
		return ""
	}
	reqId, err := rpc.StartVM(&vmId, c, ctx)
	return reqId
}

func stopVM(arg1 string, c cirrina.VMInfoClient, ctx context.Context, err error) bool {
	vmId, err := rpc.VmNameToId(arg1, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s", arg1, err)
		return true
	}
	running, err := rpc.VmRunning(&vmId, c, ctx)
	if err != nil {
		log.Fatalf("could not get VM state: %v", err)
		return true
	}
	if !running {
		fmt.Printf("error: request to stop VM »%s« failed: VM must be running in order to be stopped\n", arg1)
		return true
	}
	_, err = rpc.StopVM(&vmId, c, ctx)
	if err != nil {
		fmt.Printf("error: could not find VM »%s«: %s", arg1, err)
	}
	return false
}

func reConfig(idPtr *string, err error, namePtr *string, descrPtr *string, cpuPtr *uint, memPtr *uint, autoStartPtr *bool, c cirrina.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	newConfig := cirrina.VMConfig{
		Id: *idPtr,
	}
	if isFlagPassed("name") {
		newConfig.Name = namePtr
	}
	if isFlagPassed("descr") {
		newConfig.Description = descrPtr
	}
	if isFlagPassed("cpus") {
		newCpu := uint32(*cpuPtr)
		if newCpu < 1 {
			newCpu = 1
		}
		if newCpu > 16 {
			newCpu = 16
		}
		newConfig.Cpu = &newCpu
	}
	if isFlagPassed("mem") {
		newMem := uint32(*memPtr)
		if newMem < 128 {
			newMem = 128
		}
		newConfig.Mem = &newMem
	}
	if isFlagPassed("autostart") {
		newConfig.Autostart = autoStartPtr
	}
	err = rpc.UpdateVMConfig(&newConfig, c, ctx)
	if err != nil {
		log.Fatalf("could not update VM: %v", err)
		return
	}
	fmt.Printf("Success\n")
}
