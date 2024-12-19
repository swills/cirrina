package main

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/a-h/templ"

	"cirrina/cirrinactl/rpc"
)

type VMsHandler struct {
	GetVMs func() ([]VM, error)
}

func NewVMsHandler() VMsHandler {
	return VMsHandler{
		GetVMs: getVMs,
	}
}

func getVMs() ([]VM, error) {
	var err error

	rpc.ServerName = cirrinaServerName
	rpc.ServerPort = cirrinaServerPort
	rpc.ServerTimeout = cirrinaServerTimeout
	rpc.ResetConnTimeout()

	err = rpc.GetConn()
	if err != nil {
		return []VM{}, fmt.Errorf("error getting VMs: %w", err)
	}

	VMIDs, err := rpc.GetVMIds()
	if err != nil {
		return []VM{}, fmt.Errorf("error getting VMs: %w", err)
	}

	VMs := make([]VM, 0, len(VMIDs))

	for _, VMID := range VMIDs {
		var vmName string

		rpc.ResetConnTimeout()

		vmName, err = rpc.GetVMName(VMID)
		if err != nil {
			return []VM{}, fmt.Errorf("error getting VMs: %w", err)
		}

		VMs = append(VMs, VM{Name: vmName, ID: VMID})
	}

	sort.Slice(VMs, func(i, j int) bool {
		return VMs[i].Name < VMs[j].Name
	})

	return VMs, nil
}

func (v VMsHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	VMs, err := v.GetVMs()
	if err != nil {
		logError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	templ.Handler(vms(VMs)).ServeHTTP(writer, request)
}
