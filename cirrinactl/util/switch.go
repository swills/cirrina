package util

import (
	"cirrina/cirrina"
	"cirrina/cirrinactl/rpc"
	"context"
	"fmt"
	"github.com/jedib0t/go-pretty/table"
	"google.golang.org/grpc/status"
	"log"
	"os"
	"sort"
)

func GetSwitches(c cirrina.VMInfoClient, ctx context.Context) {
	res, err := rpc.GetSwitches(c, ctx)
	if err != nil {
		log.Fatalf("could not get Switches: %v", err)
		return
	}

	var names []string
	type ThisSwitchInfo struct {
		id         string
		switchtype string
		uplink     string
		descr      string
	}

	switchInfos := make(map[string]ThisSwitchInfo)

	for _, id := range res {
		res2, err := rpc.GetSwitch(&id, c, ctx)
		if err != nil {
			log.Fatalf("could not get switch: %v", err)
		}
		switchType := "Unknown"
		if *res2.SwitchType == cirrina.SwitchType_IF {
			switchType = "bridge"
		} else if *res2.SwitchType == cirrina.SwitchType_NG {
			switchType = "netgraph"
		}

		aIsoInfo := ThisSwitchInfo{
			id:         id,
			switchtype: switchType,
			uplink:     *res2.Uplink,
			descr:      *res2.Description,
		}
		switchInfos[*res2.Name] = aIsoInfo
		names = append(names, *res2.Name)

	}

	sort.Strings(names)

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"NAME", "UUID", "TYPE", "UPLINK", "DESCRIPTION"})
	t.SetStyle(myTableStyle)
	for _, name := range names {
		t.AppendRow(table.Row{
			name,
			switchInfos[name].id,
			switchInfos[name].switchtype,
			switchInfos[name].uplink,
			switchInfos[name].descr,
		})

	}
	t.Render()
}

func SetUplink(switchName string, c cirrina.VMInfoClient, ctx context.Context, uplinkName string) bool {
	switchId, err := rpc.SwitchNameToId(&switchName, c, ctx)
	if err != nil || switchId == "" {
		fmt.Printf("error: could not find switch: no switch with the given name found\n")
		return true
	}
	err = rpc.SetSwitchUplink(c, ctx, &switchId, &uplinkName)
	if err != nil {
		fmt.Printf("error: could not set switch uplink: %s\n", err.Error())
		return true
	}
	return false
}

func AddSwitch(name string, c cirrina.VMInfoClient, ctx context.Context, description string, switchType string) bool {
	_, err := rpc.AddSwitch(&name, c, ctx, &description, &switchType)
	if err != nil {
		s := status.Convert(err)
		fmt.Printf("error: could not create a new switch: %s\n", s.Message())
		return true
	}
	return false
}

func RmSwitch(name string, c cirrina.VMInfoClient, ctx context.Context) {
	switchId, err := rpc.SwitchNameToId(&name, c, ctx)
	if err != nil {
		s := status.Convert(err)
		fmt.Printf("error: could not delete switch: %s\n", s.Message())
		return
	}
	if switchId == "" {
		fmt.Printf("error: could not find switch: no switch with the given name found\n")
		return
	}
	err = rpc.RemoveSwitch(&switchId, c, ctx)
	if err != nil {
		s := status.Convert(err)
		fmt.Printf("error: could not delete switch: %s\n", s.Message())
	}
	return
}
