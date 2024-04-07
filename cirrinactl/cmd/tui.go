package cmd

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"

	"cirrina/cirrinactl/rpc"
)

var TuiCmd = &cobra.Command{
	Use:          "tui",
	Short:        "Start terminal UI",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		serverAddr := rpc.ServerName + ":" + strconv.FormatInt(int64(rpc.ServerPort), 10)
		err := StartTui(serverAddr)
		if err != nil {
			return err
		}

		return nil
	},
}

type vmItem struct {
	name string
	desc string
}

var app *tview.Application
var mainFlex *tview.Flex
var infoFlex *tview.Flex
var vmList *tview.List
var startButton *tview.Button
var stopButton *tview.Button
var editButton *tview.Button
var comButton *tview.Button
var vncButton *tview.Button
var vmItems []vmItem

func getVmItems() ([]vmItem, error) {
	var vmIds []string
	var vmItems []vmItem
	var err error

	vmIds, err = rpc.GetVmIds()
	if err != nil {
		return []vmItem{}, err
	}

	for _, vmId := range vmIds {
		var res rpc.VmConfig
		res, err = rpc.GetVMConfig(vmId)
		if err != nil {
			return []vmItem{}, err
		}
		var aItem vmItem
		aItem.name = res.Name
		aItem.desc = res.Description
		vmItems = append(vmItems, aItem)
	}

	sort.Slice(vmItems, func(i, j int) bool { return vmItems[i].name < vmItems[j].name })

	return vmItems, nil
}

func startButtonExit(key tcell.Key) {
	switch key {
	case tcell.KeyEscape:
		app.SetFocus(vmList)
	case tcell.KeyTab:
		app.SetFocus(stopButton)
	case tcell.KeyBacktab:
		app.SetFocus(vncButton)
	default:
	}
}

func stopButtonExit(key tcell.Key) {
	switch key {
	case tcell.KeyEscape:
		app.SetFocus(vmList)
	case tcell.KeyTab:
		app.SetFocus(editButton)
	case tcell.KeyBacktab:
		app.SetFocus(startButton)
	default:
	}
}

func editButtonExit(key tcell.Key) {
	switch key {
	case tcell.KeyEscape:
		app.SetFocus(vmList)
	case tcell.KeyTab:
		app.SetFocus(comButton)
	case tcell.KeyBacktab:
		app.SetFocus(stopButton)
	default:
	}
}

func comButtonExit(key tcell.Key) {
	switch key {
	case tcell.KeyEscape:
		app.SetFocus(vmList)
	case tcell.KeyTab:
		app.SetFocus(vncButton)
	case tcell.KeyBacktab:
		app.SetFocus(editButton)
	default:
	}
}

func vncButtonExit(key tcell.Key) {
	switch key {
	case tcell.KeyEscape:
		app.SetFocus(vmList)
	case tcell.KeyTab:
		app.SetFocus(startButton)
	case tcell.KeyBacktab:
		app.SetFocus(comButton)
	default:
	}
}

func vmStartFunc(name string) {
	vmId, err := rpc.VmNameToId(name)
	if err != nil {
		return
	}
	if vmId == "" {
		return
	}
	_, _ = rpc.StartVM(vmId)
}

func vmStopFunc(name string) {
	vmId, err := rpc.VmNameToId(name)
	if err != nil {
		return
	}
	_, _ = rpc.StopVM(vmId)
}

func vmChangedFunc(index int, name string, _ string, _ rune) {
	infoFlex.Clear()
	if index >= len(vmItems) {
		quit := tview.NewTextView()
		quit.SetText("Quit?")
		infoFlex.AddItem(quit, 0, 1, false)

		return
	}

	buttonRowFlex := tview.NewFlex()
	startButton = tview.NewButton("Start")
	startButton.SetExitFunc(startButtonExit)
	startButton.SetSelectedFunc(func() { vmStartFunc(name) })
	buttonRowFlex.AddItem(startButton, 0, 1, true)

	stopButton = tview.NewButton("Stop")
	stopButton.SetExitFunc(stopButtonExit)
	stopButton.SetSelectedFunc(func() { vmStopFunc(name) })
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

func StartTui(serverAddr string) error {
	title := fmt.Sprintf(" cirrinactl - %v ", serverAddr)
	var err error
	vmList = tview.NewList()
	vmItems, err = getVmItems()
	if err != nil {
		return err
	}
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

	return nil
}

func init() {
	disableFlagSorting(TuiCmd)
}
