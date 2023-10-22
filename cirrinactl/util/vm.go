package util

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"cirrina/cirrina"
	"cirrina/cirrinactl/rpc"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
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
		"name: %v"+
			"\nid: %v"+
			"\ndesc: %v"+
			"\ncpus: %v"+
			"\nmem: %v"+
			"\npriority: %v"+
			"\nprotect: %v"+
			"\npcpu: %v"+
			"\nrbps: %v"+
			"\nwbps: %v"+
			"\nriops: %v"+
			"\nwiops: %v"+
			"\n",
		*res.Name,
		*idPtr,
		*res.Description,
		*res.Cpu,
		*res.Mem,
		*res.Priority,
		*res.Protect,
		*res.Pcpu,
		*res.Rbps,
		*res.Wbps,
		*res.Riops,
		*res.Wiops,
	)

	fmt.Printf(
		"\ncom1: %v"+
			"\ncom1-log: %v"+
			"\ncom1-dev: %v"+
			"\ncom1-speed: %v"+
			"\n",
		*res.Com1,
		*res.Com1Log,
		*res.Com1Dev,
		*res.Com1Speed,
	)

	fmt.Printf(
		"\ncom2: %v"+
			"\ncom2-log: %v"+
			"\ncom2-dev: %v"+
			"\ncom2-speed: %v"+
			"\n",
		*res.Com2,
		*res.Com2Log,
		*res.Com2Dev,
		*res.Com2Speed,
	)

	fmt.Printf(
		"\ncom3: %v"+
			"\ncom3-log: %v"+
			"\ncom3-dev: %v"+
			"\ncom3-speed: %v"+
			"\n",
		*res.Com3,
		*res.Com3Log,
		*res.Com3Dev,
		*res.Com3Speed,
	)

	fmt.Printf(
		"\ncom4: %v"+
			"\ncom4-log: %v"+
			"\ncom4-dev: %v"+
			"\ncom4-speed: %v"+
			"\n",
		*res.Com4,
		*res.Com4Log,
		*res.Com4Dev,
		*res.Com4Speed,
	)

	fmt.Printf(
		"\nscreen: %v"+
			"\nvnc port: %v"+
			"\nscreen width: %v"+
			"\nscreen height: %v"+
			"\nvncWait: %v"+
			"\ntablet mode: %v"+
			"\nkeyboard: %v"+
			"\n",
		*res.Screen,
		*res.Vncport,
		*res.ScreenWidth,
		*res.ScreenHeight,
		*res.Vncwait,
		*res.Tablet,
		*res.Keyboard,
	)

	fmt.Printf(
		"\nsound: %v"+
			"\nsound input: %v"+
			"\nsound output: %v"+
			"\n",
		*res.Sound,
		*res.SoundIn,
		*res.SoundOut,
	)

	fmt.Printf(
		"\nauto start: %v"+
			"\nauto start delay: %v"+
			"\nrestart: %v"+
			"\nrestart delay: %v"+
			"\nmax wait: %v"+
			"\n",
		*res.Autostart,
		*res.AutostartDelay,
		*res.Restart,
		*res.RestartDelay,
		*res.MaxWait,
	)
	var extraArgs string
	if res.ExtraArgs != nil {
		extraArgs = *res.ExtraArgs
	}

	fmt.Printf(
		"\nstore uefi vars: %v"+
			"\nuse utc time: %v "+
			"\ndestroy on power off: %v"+
			"\nwire guest mem: %v"+
			"\nuse host bridge: %v"+
			"\ngenerate acpi tables: %v"+
			"\nexit on PAUSE: %v"+
			"\nignore unknown msr: %v"+
			"\nyield on HLT: %v"+
			"\ndebug: %v"+
			"\ndebug wait: %v"+
			"\ndebug port: %v"+
			"\nextra args: %v"+
			"\n",
		*res.Storeuefi,
		*res.Utc,
		*res.Dpo,
		*res.Wireguestmem,
		*res.Hostbridge,
		*res.Acpi,
		*res.Eop,
		*res.Ium,
		*res.Hlt,
		*res.Debug,
		*res.DebugWait,
		*res.DebugPort,
		extraArgs,
	)

	fmt.Printf("\nstatus: %s\nvnc port: %s\ndebug port: %s\n", res2, vncPort, debugPort)
}

func GetVMs(c cirrina.VMInfoClient, ctx context.Context, useHumanize bool) {
	ids, err := rpc.GetVmIds(c, ctx)
	if err != nil {
		log.Fatalf(err.Error())
	}

	var names []string
	type ThisVmInfo struct {
		id      string
		cpu     uint32
		mem     string
		sstatus string
		descr   string
	}

	vmInfos := make(map[string]ThisVmInfo)

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

		var memi uint64
		var mems string
		memi = uint64(*res.Mem)
		if useHumanize {
			mems = humanize.IBytes(memi * 1024 * 1024)
		} else {
			mems = strconv.FormatUint(memi*1024*1024, 10)
		}

		if status == "stopped" {
			sstatus = color.RedString("STOPPED")
		} else if status == "starting" {
			sstatus = color.YellowString("STARTING")
		} else if status == "running" {
			sstatus = color.GreenString("RUNNING")
		} else if status == "stopping" {
			sstatus = color.YellowString("STOPPING")
		}

		aVmInfo := ThisVmInfo{
			id:      id,
			cpu:     *res.Cpu,
			mem:     mems,
			sstatus: sstatus,
			descr:   *res.Description,
		}
		vmInfos[*res.Name] = aVmInfo
		names = append(names, *res.Name)
	}

	sort.Strings(names)

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
	for _, name := range names {
		t.AppendRow(table.Row{
			name,
			vmInfos[name].id,
			vmInfos[name].cpu,
			vmInfos[name].mem,
			vmInfos[name].sstatus,
			vmInfos[name].descr,
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

func ClearUefiVars(VmName string, c cirrina.VMInfoClient, ctx context.Context) {
	vmId, err := rpc.VmNameToId(VmName, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s", VmName, err)
	}
	res, err := rpc.VmClearUefiVars(vmId, c, ctx)
	if err != nil {
		log.Fatalf(err.Error())
	}
	if !res {
		fmt.Printf("Failed\n")
	}
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

func GetVMDisks(VmName string, c cirrina.VMInfoClient, ctx context.Context, useHumanize bool) {
	vmId, err := rpc.VmNameToId(VmName, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s\n", VmName, err)
		return
	}
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{"NAME", "UUID", "TYPE", "SIZE", "USAGE", "DEV-TYPE", "CACHE", "DIRECT", "DESCRIPTION"})

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
		if useHumanize {
			t.AppendRow(table.Row{
				*res.Name,
				id,
				*res.DiskType,
				humanize.IBytes(*res.SizeNum),
				humanize.IBytes(*res.UsageNum),
				*res.DiskDevType,
				*res.Cache,
				*res.Direct,
				*res.Description,
			})
		} else {
			t.AppendRow(table.Row{
				*res.Name,
				id,
				*res.DiskType,
				*res.SizeNum,
				*res.UsageNum,
				*res.DiskDevType,
				*res.Cache,
				*res.Direct,
				*res.Description,
			})
		}
	}
	t.Render()
}

func GetVMIsos(VmName string, c cirrina.VMInfoClient, ctx context.Context, useHumanize bool) {
	vmId, err := rpc.VmNameToId(VmName, c, ctx)
	if err != nil || vmId == "" {
		fmt.Printf("error: could not find VM »%s«: %s\n", VmName, err)
		return
	}
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{"NAME", "UUID", "SIZE", "DESCRIPTION"})

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

		if useHumanize {
			t.AppendRow(table.Row{
				*res.Name,
				id,
				humanize.IBytes(*res.Size),
				*res.Description,
			})
		} else {
			t.AppendRow(table.Row{
				*res.Name,
				id,
				*res.Size,
				*res.Description,
			})
		}
	}

	t.Render()
}

func GetVmNics(VmName string, c cirrina.VMInfoClient, ctx context.Context, useHumanize bool) {
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

		var rateins string
		var rateouts string

		if *res.Ratelimit {
			if useHumanize {
				rateins = humanize.Bytes(*res.Ratein)
				rateins = strings.Replace(rateins, "B", "b", 1) + "ps"
				rateouts = humanize.Bytes(*res.Rateout)
				rateouts = strings.Replace(rateouts, "B", "b", 1) + "ps"
			} else {
				rateins = strconv.FormatUint(*res.Ratein, 10)
				rateouts = strconv.FormatUint(*res.Rateout, 10)
			}
		}

		t.AppendRow(table.Row{
			*res.Name,
			id,
			*res.Mac,
			*res.Nettype,
			*res.Netdevtype,
			uplinkName,
			*res.Ratelimit,
			rateins,
			rateouts,
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
		fmt.Printf("failed to add nic: %v\n", err)
		return
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
