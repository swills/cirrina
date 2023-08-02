package main

import (
	"cirrina/cirrina"
	"context"
	"fmt"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"io"
	"log"
	"os"
)

func addVM(namePtr *string, c cirrina.VMInfoClient, ctx context.Context, descrPtr *string, cpuPtr *uint32, memPtr *uint32) {
	if *namePtr == "" {
		log.Fatalf("Name not specified")
		return
	}
	res, err := c.AddVM(ctx, &cirrina.VMConfig{
		Name:        namePtr,
		Description: descrPtr,
		Cpu:         cpuPtr,
		Mem:         memPtr,
	})
	if err != nil {
		log.Fatalf("could not create VM: %v", err)
		return
	}
	fmt.Printf("Created VM %v\n", res.Value)
}

func DeleteVM(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	reqId, err := c.DeleteVM(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not delete VM: %v", err)
	}
	fmt.Printf("Deleted request created, reqid: %v\n", reqId.Value)
}

func stopVM(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	reqId, err := c.StopVM(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not stop VM: %v", err)
	}
	fmt.Printf("Stopping request created, reqid: %v\n", reqId.Value)
}

func startVM(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	reqId, err := c.StartVM(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not start VM: %v", err)
	}
	fmt.Printf("Started request created, reqid: %v\n", reqId.Value)
}

func getVMConfig(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (res *cirrina.VMConfig) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	res, err := c.GetVMConfig(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
		return nil
	}
	return res
}

func getVM(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	res := getVMConfig(idPtr, c, ctx)
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

func getVmIds(c cirrina.VMInfoClient, ctx context.Context) (ids []string) {
	res, err := c.GetVMs(ctx, &cirrina.VMsQuery{})
	if err != nil {
		log.Fatalf("could not get VMs: %v", err)
		return ids
	}
	for {
		VM, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("GetVMs failed: %v", err)
		}
		ids = append(ids, VM.Value)
	}
	return ids
}

func getVMs(c cirrina.VMInfoClient, ctx context.Context) {
	ids := getVmIds(c, ctx)
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
		res, err := c.GetVMConfig(ctx, &cirrina.VMID{Value: id})
		if err != nil {
			log.Fatalf("could not get VM: %v", err)
			return
		}
		res2, err := c.GetVMState(ctx, &cirrina.VMID{Value: id})
		if err != nil {
			log.Fatalf("could not get VM: %v", err)
			return
		}

		status := "Unknown"
		if res2.Status == cirrina.VmStatus_STATUS_STOPPED {
			status = color.RedString("STOPPED")
		} else if res2.Status == cirrina.VmStatus_STATUS_STARTING {
			status = color.YellowString("STARTING")
		} else if res2.Status == cirrina.VmStatus_STATUS_RUNNING {
			status = color.GreenString("RUNNING")
		} else if res2.Status == cirrina.VmStatus_STATUS_STOPPING {
			status = color.YellowString("STOPPING")
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

func Reconfig(idPtr *string, err error, namePtr *string, descrPtr *string, cpuPtr *uint, memPtr *uint, autoStartPtr *bool, c cirrina.VMInfoClient, ctx context.Context) {
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
	_, err = c.UpdateVM(ctx, &newConfig)
	if err != nil {
		log.Fatalf("could not update VM: %v", err)
		return
	}
	fmt.Printf("Success\n")
}

func getVMState(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	res, err := c.GetVMState(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not get state: %v", err)
		return
	}
	var vmstate string
	switch res.Status {
	case cirrina.VmStatus_STATUS_STOPPED:
		vmstate = "stopped"
	case cirrina.VmStatus_STATUS_STARTING:
		vmstate = "starting"
	case cirrina.VmStatus_STATUS_RUNNING:
		vmstate = "running"
	case cirrina.VmStatus_STATUS_STOPPING:
		vmstate = "stopping"
	}
	fmt.Printf("vm id: %v state: %v vnc port: %v\n", *idPtr, vmstate, res.VncPort)
}

func vmNameToId(name string, c cirrina.VMInfoClient, ctx context.Context) (rid string) {
	found := false
	ids := getVmIds(c, ctx)
	for _, id := range ids {
		res := getVMConfig(&id, c, ctx)
		if *res.Name == name {
			if found == true {
				log.Fatalf("Duplicate VM name %v", name)
			} else {
				found = true
				rid = id
			}
		}
	}
	return rid
}
