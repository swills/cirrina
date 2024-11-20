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
	RunE: func(_ *cobra.Command, _ []string) error {
		serverAddr := rpc.ServerName + ":" + strconv.FormatInt(int64(rpc.ServerPort), 10)
		err := StartTui(serverAddr)
		if err != nil {
			return fmt.Errorf("error starting: %w", err)
		}

		return nil
	},
}

type vmItem struct {
	name string
	desc string
}

var (
	app         *tview.Application
	mainFlex    *tview.Flex
	infoFlex    *tview.Flex
	vmList      *tview.List
	startButton *tview.Button
	stopButton  *tview.Button
	editButton  *tview.Button
	comButton   *tview.Button
	vncButton   *tview.Button
	vmItems     []vmItem
)

func getVMItems() ([]vmItem, error) {
	var vmIDs []string

	var err error

	vmIDs, err = rpc.GetVMIds()
	if err != nil {
		return []vmItem{}, fmt.Errorf("error getting vm list: %w", err)
	}

	theseVMItems := make([]vmItem, 0, len(vmIDs))

	for _, vmID := range vmIDs {
		var res rpc.VMConfig

		res, err = rpc.GetVMConfig(vmID)
		if err != nil {
			return []vmItem{}, fmt.Errorf("error getting vm config: %w", err)
		}

		var aItem vmItem
		aItem.name = res.Name
		aItem.desc = res.Description
		theseVMItems = append(theseVMItems, aItem)
	}

	sort.Slice(theseVMItems, func(i, j int) bool { return theseVMItems[i].name < theseVMItems[j].name })

	return theseVMItems, nil
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
	vmID, err := rpc.VMNameToID(name)
	if err != nil {
		return
	}

	if vmID == "" {
		return
	}

	_, _ = rpc.StartVM(vmID)
}

func vmStopFunc(name string) {
	vmID, err := rpc.VMNameToID(name)
	if err != nil {
		return
	}

	_, _ = rpc.StopVM(vmID)
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

	vmItems, err = getVMItems()
	if err != nil {
		return fmt.Errorf("error getting VMs: %w", err)
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
