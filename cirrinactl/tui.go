package main

import (
	pb "cirrina/cirrina"
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"os"
	"sort"
	"time"
)

type model struct {
	choices  []string
	cursor   int
	selected map[int]struct{}
}

func initialModel(addr string) model {
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

	var vmIds []string
	var vmNames []string

	res, err := c.GetVMs(ctx, &pb.VMsQuery{})
	if err != nil {
		return model{}
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
		vmNames = append(vmNames, *res.Name)
	}

	sort.Strings(vmNames)

	return model{
		choices:  vmNames,
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter", " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	s := "Select VM\n\n"

	for i, choice := range m.choices {

		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
		}

		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	s += "\nPress q to quit.\n"

	return s
}

func startTea(serverAddr string) {

	p := tea.NewProgram(initialModel(serverAddr))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

}
