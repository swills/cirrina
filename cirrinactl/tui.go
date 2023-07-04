package main

import (
	pb "cirrina/cirrina"
	"context"
	"github.com/rivo/tview"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"sort"
	"time"
)

type vmItem struct {
	name string
	desc string
}

func getVms(addr string) []vmItem {
	var vmIds []string
	var vmItems []vmItem

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	res, err := c.GetVMs(ctx, &pb.VMsQuery{})
	if err != nil {
		return vmItems
	}

	for {
		VM, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("GetVMs failed: %v", err)
		}
		vmIds = append(vmIds, VM.Value)
	}

	for _, vmId := range vmIds {
		res, err := c.GetVMConfig(ctx, &pb.VMID{Value: vmId})
		if err != nil {
			log.Fatalf("could not get VM: %v", err)
		}
		aItem := vmItem{
			name: *res.Name,
			desc: *res.Description,
		}
		vmItems = append(vmItems, aItem)
	}

	sort.Slice(vmItems, func(i, j int) bool { return vmItems[i].name < vmItems[j].name })

	return vmItems
}

func startTui(serverAddr string) {

	vmList := tview.NewList()
	vmItems := getVms(serverAddr)

	app := tview.NewApplication()
	for _, vmItem := range vmItems {
		vmList.AddItem(vmItem.name, vmItem.desc, 0, nil)
	}

	vmList.AddItem("Quit", "Press to exit", 'q', func() {
		app.Stop()
	})

	if err := app.SetRoot(vmList, true).SetFocus(vmList).Run(); err != nil {
		panic(err)
	}
}
