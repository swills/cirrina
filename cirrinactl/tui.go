package main

import (
	pb "cirrina/cirrina"
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"os"
	"sort"
	"time"
)

var docStyle = lipgloss.NewStyle()

type item struct {
	name, desc string
}

func (i item) Title() string       { return i.name }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.name }

type vmListModel struct {
	list list.Model
}

func (m vmListModel) Init() tea.Cmd {
	return nil
}

func (m vmListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m vmListModel) View() string {
	return docStyle.Render(m.list.View())
}

func getVms(addr string) []list.Item {
	var vmIds []string
	var vmItems []list.Item

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
		aItem := item{
			name: *res.Name,
			desc: *res.Description,
		}
		vmItems = append(vmItems, aItem)
	}

	sort.Slice(vmItems, func(i, j int) bool { return vmItems[i].FilterValue() < vmItems[j].FilterValue() })

	return vmItems
}

func startTea(serverAddr string) {

	vmItems := getVms(serverAddr)

	m := vmListModel{list: list.New(vmItems, list.NewDefaultDelegate(), 0, 0)}
	m.list.Title = "VMs"

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
