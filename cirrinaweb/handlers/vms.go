package handlers

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/a-h/templ"

	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinaweb/components"
	"cirrina/cirrinaweb/util"
)

type VMsHandler struct {
	GetVMs func(ctx context.Context) ([]components.VM, error)
}

func (v VMsHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	VMs, err := v.GetVMs(request.Context())
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	templ.Handler(components.Vms(VMs)).ServeHTTP(writer, request) //nolint:contextcheck
}

func GetVMs(ctx context.Context) ([]components.VM, error) {
	var err error

	err = util.InitRPCConn()
	if err != nil {
		return []components.VM{}, fmt.Errorf("error getting VMs: %w", err)
	}

	VMIDs, err := rpc.GetVMIds(ctx)
	if err != nil {
		return []components.VM{}, fmt.Errorf("error getting VMs: %w", err)
	}

	VMs := make([]components.VM, 0, len(VMIDs))

	for _, VMID := range VMIDs {
		var vmName string

		vmName, err = rpc.GetVMName(ctx, VMID)
		if err != nil {
			return []components.VM{}, fmt.Errorf("error getting VMs: %w", err)
		}

		VMs = append(VMs, components.VM{Name: vmName, ID: VMID})
	}

	sort.Slice(VMs, func(i, j int) bool {
		return VMs[i].Name < VMs[j].Name
	})

	return VMs, nil
}
