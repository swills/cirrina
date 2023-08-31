package util

import (
	"cirrina/cirrina"
	"cirrina/cirrinactl/rpc"
	"context"
	"fmt"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"log"
	"os"
)

func GetVM(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	res, err := rpc.GetVMConfig(idPtr, c, ctx)
	if err != nil {
		log.Fatalf("Error getting VM: %v\n", err.Error())
	}

	res2, vncPort, debugPort, err := rpc.GetVMState(idPtr, c, ctx)
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
	fmt.Printf("status: %s\nvnc port: %s\ndebug port: %s\n", res2, vncPort, debugPort)
}

func GetVMs(c cirrina.VMInfoClient, ctx context.Context) {
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
		status, _, _, err := rpc.GetVMState(&id, c, ctx)
		if err != nil {
			log.Fatalf("could not get VM: %v", err)
			return
		}
		sstatus := "Unknown"

		if status == "stopped" {
			sstatus = color.RedString("STOPPED")
		} else if status == "starting" {
			sstatus = color.YellowString("STARTING")
		} else if status == "running" {
			sstatus = color.GreenString("RUNNING")
		} else if status == "stopping" {
			sstatus = color.YellowString("STOPPING")
		}

		t.AppendRow(table.Row{
			*res.Name,
			id,
			*res.Cpu,
			*res.Mem,
			sstatus,
			*res.Description,
		})
	}

	t.Render()
}

func StartVM(VmName string, c cirrina.VMInfoClient, ctx context.Context) string {
	vmId, err := rpc.VmNameToId(VmName, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s\n", VmName, err)
		return ""
	}
	stopped, err := rpc.VmStopped(&vmId, c, ctx)
	if err != nil {
		log.Fatalf("could not get VM state: %v", err)
		return ""
	}
	if !stopped {
		fmt.Printf("error: request to start VM »%s« failed: VM must be stopped in order to be started\n", VmName)
		return ""
	}
	reqId, err := rpc.StartVM(&vmId, c, ctx)
	return reqId
}

func StopVM(VmName string, c cirrina.VMInfoClient, ctx context.Context) bool {
	vmId, err := rpc.VmNameToId(VmName, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s", VmName, err)
		return true
	}
	running, err := rpc.VmRunning(&vmId, c, ctx)
	if err != nil {
		log.Fatalf("could not get VM state: %v", err)
		return true
	}
	if !running {
		fmt.Printf("error: request to stop VM »%s« failed: VM must be running in order to be stopped\n", VmName)
		return true
	}
	_, err = rpc.StopVM(&vmId, c, ctx)
	if err != nil {
		fmt.Printf("error: could not find VM »%s«: %s", VmName, err)
	}
	return false
}

func AddVM(VmName *string, c cirrina.VMInfoClient, ctx context.Context, Description *string, lCpus *uint32, Mem *uint32) {
	res, err := rpc.AddVM(VmName, c, ctx, Description, lCpus, Mem)
	if err != nil {
		log.Fatalf("could not create VM: %v", err)
		return
	}
	fmt.Printf("Created VM %s\n", res)
}

func DeleteVM(VmName string, c cirrina.VMInfoClient, ctx context.Context) {
	vmId, err := rpc.VmNameToId(VmName, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s\n", VmName, err)
		return
	}
	stopped, err := rpc.VmStopped(&vmId, c, ctx)
	if err != nil {
		log.Printf("could not get VM state: %v", err)
		return
	}
	if !stopped {
		fmt.Printf("error: request to start VM »%s« failed: VM must be stopped in order to be started\n", VmName)
		return
	}

	reqId, err := rpc.DeleteVM(&vmId, c, ctx)
	if err != nil {
		log.Fatalf("could not delete VM: %v", err)
	}
	fmt.Printf("Deleted request created, reqid: %v\n", reqId)
}

func GetVMDisks(VmName string, c cirrina.VMInfoClient, ctx context.Context) {
	vmId, err := rpc.VmNameToId(VmName, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s\n", VmName, err)
		return
	}
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{"NAME", "UUID", "SIZE", "TYPE", "DESCRIPTION"})

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
	diskIds, err := rpc.GetVmDisks(vmId, c, ctx)
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
		return
	}
	for _, id := range diskIds {
		res, err := rpc.GetDiskInfo(id, c, ctx)
		if err != nil {
			log.Fatalf("could not get VM: %v", err)
			return
		}

		t.AppendRow(table.Row{
			*res.Name,
			id,
			*res.Size,
			*res.DiskType,
			*res.Description,
		})
	}

	t.Render()
}

func GetVMIsos(VmName string, c cirrina.VMInfoClient, ctx context.Context) {
	vmId, err := rpc.VmNameToId(VmName, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s\n", VmName, err)
		return
	}
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{"NAME", "UUID", "DESCRIPTION"})

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
	isoIds, err := rpc.GetVmIsos(vmId, c, ctx)
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
		return
	}
	for _, id := range isoIds {
		res, err := rpc.GetIsoInfo(&id, c, ctx)
		if err != nil {
			log.Fatalf("could not get VM: %v", err)
			return
		}

		t.AppendRow(table.Row{
			*res.Name,
			id,
			*res.Description,
		})
	}

	t.Render()
}

func GetVmNics(VmName string, c cirrina.VMInfoClient, ctx context.Context) {
	vmId, err := rpc.VmNameToId(VmName, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s\n", VmName, err)
		return
	}
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{"NAME", "UUID", "MAC", "TYPE", "DEV-TYPE", "SWITCH", "RATE-LIMITED", "RATE-IN", "RATE-OUT", "DESCRIPTION"})

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
	nicIds, err := rpc.GetVmNics(vmId, c, ctx)
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
		return
	}
	for _, id := range nicIds {
		res, err := rpc.GetVmNicInfo(&id, c, ctx)
		if err != nil {
			log.Fatalf("could not get VM: %v", err)
			return
		}
		uplinkId := *res.Switchid

		var uplinkName string
		if uplinkId != "" {
			res2, err := rpc.GetSwitch(&uplinkId, c, ctx)
			if err != nil {
				log.Fatalf("could not get VM: %v", err)
				return
			}
			uplinkName = *res2.Name
		}

		t.AppendRow(table.Row{
			*res.Name,
			id,
			*res.Mac,
			*res.Nettype,
			*res.Netdevtype,
			uplinkName,
			*res.Ratelimit,
			*res.Ratein,
			*res.Rateout,
			*res.Description,
		})
	}

	t.Render()
}

func VmDiskAdd(VmName string, diskId string, c cirrina.VMInfoClient, ctx context.Context) {
	vmId, err := rpc.VmNameToId(VmName, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s\n", VmName, err)
		return
	}
	diskIds, err := rpc.GetVmDisks(vmId, c, ctx)
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
		return
	}

	diskIds = append(diskIds, diskId)
	res, err := rpc.VmSetDisks(vmId, diskIds, c, ctx)
	if err != nil {
		fmt.Printf("failed to add disk: %v", err)
	}
	if res {
		fmt.Printf("Added\n")
	} else {
		fmt.Printf("Failed\n")
	}
}

func VmDiskRm(VmName string, diskId string, c cirrina.VMInfoClient, ctx context.Context) {
	vmId, err := rpc.VmNameToId(VmName, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s\n", VmName, err)
		return
	}
	diskIds, err := rpc.GetVmDisks(vmId, c, ctx)
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
		return
	}

	var newDiskIds []string

	for _, id := range diskIds {
		if id != diskId {
			newDiskIds = append(newDiskIds, id)
		}
	}

	res, err := rpc.VmSetDisks(vmId, newDiskIds, c, ctx)
	if err != nil {
		fmt.Printf("failed to add disk: %v", err)
	}
	if res {
		fmt.Printf("Removed\n")
	} else {
		fmt.Printf("Failed\n")
	}
}

func VmIsoAdd(VmName string, isoId string, c cirrina.VMInfoClient, ctx context.Context) {
	vmId, err := rpc.VmNameToId(VmName, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s\n", VmName, err)
		return
	}
	isoIds, err := rpc.GetVmIsos(vmId, c, ctx)
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
		return
	}

	isoIds = append(isoIds, isoId)
	res, err := rpc.VmSetIsos(vmId, isoIds, c, ctx)
	if err != nil {
		fmt.Printf("failed to add iso: %v", err)
	}
	if res {
		fmt.Printf("Added\n")
	} else {
		fmt.Printf("Failed\n")
	}
}

func VmIsoRm(VmName string, isoId string, c cirrina.VMInfoClient, ctx context.Context) {
	vmId, err := rpc.VmNameToId(VmName, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s\n", VmName, err)
		return
	}
	isoIds, err := rpc.GetVmIsos(vmId, c, ctx)
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
		return
	}

	var newIsoIds []string

	for _, id := range isoIds {
		if id != isoId {
			newIsoIds = append(newIsoIds, id)
		}
	}

	res, err := rpc.VmSetIsos(vmId, newIsoIds, c, ctx)
	if err != nil {
		fmt.Printf("failed to add iso: %v", err)
	}
	if res {
		fmt.Printf("Removed\n")
	} else {
		fmt.Printf("Failed\n")
	}
}

func VmNicAdd(VmName string, nicId string, c cirrina.VMInfoClient, ctx context.Context) {
	vmId, err := rpc.VmNameToId(VmName, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s\n", VmName, err)
		return
	}
	nicIds, err := rpc.GetVmNics(vmId, c, ctx)
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
		return
	}

	nicIds = append(nicIds, nicId)
	res, err := rpc.VmSetNics(vmId, nicIds, c, ctx)
	if err != nil {
		fmt.Printf("failed to add nic: %v", err)
	}
	if res {
		fmt.Printf("Added\n")
	} else {
		fmt.Printf("Failed\n")
	}
}

func VmNicRm(VmName string, nicId string, c cirrina.VMInfoClient, ctx context.Context) {
	vmId, err := rpc.VmNameToId(VmName, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s\n", VmName, err)
		return
	}
	nicIds, err := rpc.GetVmNics(vmId, c, ctx)
	if err != nil {
		log.Fatalf("could not get VM: %v", err)
		return
	}

	var newNicIds []string

	for _, id := range nicIds {
		if id != nicId {
			newNicIds = append(newNicIds, id)
		}
	}

	res, err := rpc.VmSetNics(vmId, newNicIds, c, ctx)
	if err != nil {
		fmt.Printf("failed to add nic: %v", err)
	}
	if res {
		fmt.Printf("Removed\n")
	} else {
		fmt.Printf("Failed\n")
	}
}
