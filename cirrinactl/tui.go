package main

import (
	pb "cirrina/cirrina"
	"context"
	"fmt"
	"github.com/gdamore/tcell/v2"
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

var app *tview.Application
var mainFlex *tview.Flex
var infoFlex *tview.Flex
var vmItems []vmItem
var vmList *tview.List
var startButton *tview.Button
var stopButton *tview.Button
var editButton *tview.Button
var comButton *tview.Button
var vncButton *tview.Button

func getVmItems(addr string) []vmItem {
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

func startButtonExit(key tcell.Key) {
	if key == tcell.KeyEscape {
		app.SetFocus(vmList)
	} else if key == tcell.KeyTab {
		app.SetFocus(stopButton)
	} else if key == tcell.KeyBacktab {
		app.SetFocus(vncButton)
	}
}

func stopButtonExit(key tcell.Key) {
	if key == tcell.KeyEscape {
		app.SetFocus(vmList)
	} else if key == tcell.KeyTab {
		app.SetFocus(editButton)
	} else if key == tcell.KeyBacktab {
		app.SetFocus(startButton)
	}
}

func editButtonExit(key tcell.Key) {
	if key == tcell.KeyEscape {
		app.SetFocus(vmList)
	} else if key == tcell.KeyTab {
		app.SetFocus(comButton)
	} else if key == tcell.KeyBacktab {
		app.SetFocus(stopButton)
	}
}

func comButtonExit(key tcell.Key) {
	if key == tcell.KeyEscape {
		app.SetFocus(vmList)
	} else if key == tcell.KeyTab {
		app.SetFocus(vncButton)
	} else if key == tcell.KeyBacktab {
		app.SetFocus(editButton)
	}
}

func vncButtonExit(key tcell.Key) {
	if key == tcell.KeyEscape {
		app.SetFocus(vmList)
	} else if key == tcell.KeyTab {
		app.SetFocus(startButton)
	} else if key == tcell.KeyBacktab {
		app.SetFocus(comButton)
	}
}

func vmChangedFunc(index int, name string, _ string, _ rune) {
	infoFlex.Clear()
	if index >= len(vmItems) {
		quit := tview.NewTextView()
		quit.SetText("Quit?\n")
		infoFlex.AddItem(quit, 0, 1, false)
		return
	}

	buttonRowFlex := tview.NewFlex()
	startButton = tview.NewButton("Start")
	startButton.SetExitFunc(startButtonExit)
	buttonRowFlex.AddItem(startButton, 0, 1, true)

	stopButton = tview.NewButton("Stop")
	stopButton.SetExitFunc(stopButtonExit)
	buttonRowFlex.AddItem(stopButton, 0, 1, true)

	editButton = tview.NewButton("Edit")
	editButton.SetExitFunc(editButtonExit)
	buttonRowFlex.AddItem(editButton, 0, 1, true)

	comButton = tview.NewButton("Com")
	comButton.SetExitFunc(comButtonExit)
	buttonRowFlex.AddItem(comButton, 0, 1, true)

	vncButton = tview.NewButton("VNC")
	vncButton.SetExitFunc(vncButtonExit)
	buttonRowFlex.AddItem(vncButton, 0, 1, true)

	infoFlex.AddItem(buttonRowFlex, 0, 1, true)

	nameView := tview.NewTextView()
	nameView.SetText(
		"Name: " + name + "\n" +
			"Description: " + vmItems[index].desc,
	)
	infoFlex.AddItem(nameView, 0, 6, false)
}

func vmSelectedFunc(_ int, _ string, _ string, _ rune) {
	app.SetFocus(infoFlex)
}

func startTui(serverAddr string) {
	title := fmt.Sprintf(" cirrinactl - %v ", serverAddr)

	vmList = tview.NewList()
	vmItems = getVmItems(serverAddr)
	mainFlex = tview.NewFlex()
	mainFlex.SetBorder(true)
	mainFlex.SetTitle(title)
	infoFlex = tview.NewFlex().SetDirection(tview.FlexRow)

	app = tview.NewApplication()
	for _, vmItem := range vmItems {
		vmList.AddItem(vmItem.name, "", 0, nil)
	}

	// force first item selected
	if len(vmItems) > 0 {
		vmChangedFunc(0, vmItems[0].name, "", 0)
	}

	vmList.AddItem("Quit", "Press to exit", 'q', func() {
		app.Stop()
	})

	vmList.ShowSecondaryText(false)
	vmList.SetHighlightFullLine(true)
	vmList.SetChangedFunc(vmChangedFunc)
	vmList.SetSelectedFunc(vmSelectedFunc)
	mainFlex.AddItem(vmList, 0, 1, true)
	mainFlex.AddItem(infoFlex, 0, 2, true)
	if err := app.SetRoot(mainFlex, true).SetFocus(vmList).Run(); err != nil {
		panic(err)
	}
}
