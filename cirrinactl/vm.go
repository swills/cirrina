package main

import (
	"cirrina/cirrina"
	"context"
	"errors"
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

func rpcStopVM(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
	if *idPtr == "" {
		return "", errors.New("id not specified")
	}
	reqId, err := c.StopVM(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		return "", err
	}
	return reqId.Value, nil
}

func rpcStartVM(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
	if *idPtr == "" {
		return "", errors.New("id not specified")
	}
	reqId, err := c.StartVM(ctx, &cirrina.VMID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not start VM: %v", err)
		return "", err
	}
	return reqId.Value, nil
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

func getVmIds(c cirrina.VMInfoClient, ctx context.Context) (ids []string, err error) {
	res, err := c.GetVMs(ctx, &cirrina.VMsQuery{})
	if err != nil {
		em := fmt.Sprintf("error: could not fetch list of VMs: %s", err)
		return ids, errors.New(em)
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
	return ids, nil
}

func getVMs(c cirrina.VMInfoClient, ctx context.Context) {
	ids, err := getVmIds(c, ctx)
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
	ids, err := getVmIds(c, ctx)
	if err != nil {
		log.Fatalf("failed to get vm")
	}
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

func getVmIdByName(s *string, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
	vmList, err := getVmIds(c, ctx)
	if err != nil {
		fmt.Printf(err.Error())
	}
	found := false
	rv := ""

	for _, id := range vmList {
		res, err := c.GetVMConfig(ctx, &cirrina.VMID{Value: id})
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

func startVM(arg1 string, c cirrina.VMInfoClient, ctx context.Context, err error) string {
	vmId, err := getVmIdByName(&arg1, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s\n", arg1, err)
		return ""
	}
	res2, err := c.GetVMState(ctx, &cirrina.VMID{Value: vmId})
	if err != nil {
		log.Fatalf("could not get VM state: %v", err)
		return ""
	}

	if res2.Status != cirrina.VmStatus_STATUS_STOPPED {
		fmt.Printf("error: request to start VM »%s« failed: VM must be stopped in order to be started\n", arg1)
		return ""
	}
	reqId, err := rpcStartVM(&vmId, c, ctx)
	return reqId
}

func stopVM(arg1 string, c cirrina.VMInfoClient, ctx context.Context, err error) bool {
	vmId, err := getVmIdByName(&arg1, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s", arg1, err)
		return true
	}
	res2, err := c.GetVMState(ctx, &cirrina.VMID{Value: vmId})
	if err != nil {
		log.Fatalf("could not get VM state: %v", err)
		return true
	}

	if res2.Status != cirrina.VmStatus_STATUS_RUNNING {
		fmt.Printf("error: request to stop VM »%s« failed: VM must be running in order to be stopped\n", arg1)
		return true
	}
	_, err = rpcStopVM(&vmId, c, ctx)
	if err != nil {
		fmt.Printf("error: could not find VM »%s«: %s", arg1, err)
	}

	return false
}
