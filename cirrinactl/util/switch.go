package util

import (
	"cirrina/cirrina"
	"cirrina/cirrinactl/rpc"
	"context"
	"fmt"
	"github.com/jedib0t/go-pretty/table"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"os"
)

func GetSwitches(c cirrina.VMInfoClient, ctx context.Context) {
	res, err := rpc.GetSwitches(c, ctx)
	if err != nil {
		log.Fatalf("could not get Switches: %v", err)
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{"NAME", "UUID", "TYPE", "UPLINK", "DESCRIPTION"})
	t.SetStyle(myTableStyle)

	for {
		VmSwitch, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("GetSwitches failed: %v", err)
		}
		res2, err := rpc.GetSwitch(&VmSwitch.Value, c, ctx)
		if err != nil {
			log.Fatalf("could not get switch: %v", err)
		}
		switchType := "Unknown"
		if *res2.SwitchType == cirrina.SwitchType_IF {
			switchType = "bridge"
		} else if *res2.SwitchType == cirrina.SwitchType_NG {
			switchType = "netgraph"
		}
		t.AppendRow(table.Row{*res2.Name, VmSwitch.Value, switchType, *res2.Uplink, *res2.Description})
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
