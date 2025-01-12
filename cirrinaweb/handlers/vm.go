package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinaweb/components"
	"cirrina/cirrinaweb/util"
)

type VMHandler struct {
	GetVM      func(string) (components.VM, error)
	GetVMs     func() ([]components.VM, error)
	GetVMDisks func(string) ([]components.Disk, error)
	GetVMISOs  func(string) ([]components.ISO, error)
	GetVMNICs  func(string) ([]components.NIC, error)
}

func NewVMHandler() VMHandler {
	return VMHandler{
		GetVM:      GetVM,
		GetVMs:     GetVMs,
		GetVMDisks: GetVMDisks,
		GetVMISOs:  GetVMISOs,
		GetVMNICs:  GetVMNICs,
	}
}

func (v VMHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	aVM, err := v.GetVM(request.PathValue("nameOrID"))
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	VMs, err := v.GetVMs()
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	aVM.Disks, err = v.GetVMDisks(aVM.ID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	aVM.ISOs, err = v.GetVMISOs(aVM.ID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	aVM.NICs, err = v.GetVMNICs(aVM.ID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	listenHost := util.GetListenHost()
	websockifyPort := util.GetWebsockifyPort()

	templ.Handler(components.Vm(VMs, aVM, listenHost, websockifyPort)).ServeHTTP(writer, request)
}

type VMStartPostHandler struct {
	GetVM func(string) (components.VM, error)
}

func NewVMStartHandler() VMStartPostHandler {
	return VMStartPostHandler{
		GetVM: GetVM,
	}
}

func (v VMStartPostHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	aVM, err := v.GetVM(request.PathValue("nameOrID"))
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	err = aVM.Start()
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	templ.Handler(components.StartButton(aVM)).ServeHTTP(writer, request)
}

type VMStopPostHandler struct {
	GetVM func(string) (components.VM, error)
}

func NewVMStopHandler() VMStopPostHandler {
	return VMStopPostHandler{
		GetVM: GetVM,
	}
}

func (v VMStopPostHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	aVM, err := v.GetVM(request.PathValue("nameOrID"))
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	err = aVM.Stop()
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	templ.Handler(components.StopButton(aVM)).ServeHTTP(writer, request)
}

func serveErrorVM(writer http.ResponseWriter, request *http.Request, err error) {
	// get list of VMs for the sidebar
	vmList, getVMsErr := GetVMs()
	if getVMsErr != nil {
		util.LogError(err, request.RemoteAddr)

		http.Error(writer, "failed to retrieve VMs", http.StatusInternalServerError)

		return
	}

	if e, ok := status.FromError(err); ok {
		switch e.Code() {
		case codes.NotFound:
			templ.Handler(
				components.VmNotFoundComponent(vmList),
				templ.WithStatus(http.StatusNotFound),
			).ServeHTTP(writer, request)
		case codes.OK, codes.Canceled, codes.Unknown, codes.InvalidArgument, codes.DeadlineExceeded, codes.AlreadyExists, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unimplemented, codes.Internal, codes.Unavailable, codes.DataLoss, codes.Unauthenticated: //nolint:lll
			fallthrough
		default:
			templ.Handler(components.VmNotFoundComponent(vmList), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request) //nolint:lll
		}
	} else {
		templ.Handler(components.VmNotFoundComponent(vmList), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request) //nolint:lll
	}
}

func GetVMDisks(vmID string) ([]components.Disk, error) {
	var vmDisks []string

	var err error

	err = util.InitRPCConn()
	if err != nil {
		return []components.Disk{}, fmt.Errorf("error getting VM Disks: %w", err)
	}

	rpc.ResetConnTimeout()

	vmDisks, err = rpc.GetVMDisks(vmID)
	if err != nil {
		return []components.Disk{}, fmt.Errorf("error getting VM Disks: %w", err)
	}

	returnDisks := make([]components.Disk, 0, len(vmDisks))

	for _, diskID := range vmDisks {
		var aDisk rpc.DiskInfo

		rpc.ResetConnTimeout()

		aDisk, err = rpc.GetDiskInfo(diskID)
		if err != nil {
			return []components.Disk{}, fmt.Errorf("error getting VM Disks: %w", err)
		}

		returnDisks = append(returnDisks, components.Disk{Name: aDisk.Name, ID: diskID})
	}

	return returnDisks, nil
}

func GetVMISOs(vmID string) ([]components.ISO, error) {
	var vmISOs []string

	var err error

	err = util.InitRPCConn()
	if err != nil {
		return []components.ISO{}, fmt.Errorf("error getting VM ISOs: %w", err)
	}

	rpc.ResetConnTimeout()

	vmISOs, err = rpc.GetVMIsos(vmID)
	if err != nil {
		return []components.ISO{}, fmt.Errorf("error getting VM ISOs: %w", err)
	}

	returnISOs := make([]components.ISO, 0, len(vmISOs))

	for _, isoID := range vmISOs {
		var aISO rpc.IsoInfo

		rpc.ResetConnTimeout()

		aISO, err = rpc.GetIsoInfo(isoID)
		if err != nil {
			return []components.ISO{}, fmt.Errorf("error getting VM ISOs: %w", err)
		}

		returnISOs = append(returnISOs, components.ISO{Name: aISO.Name, ID: isoID})
	}

	return returnISOs, nil
}

func GetVMNICs(vmID string) ([]components.NIC, error) {
	var vmNICs []string

	var err error

	err = util.InitRPCConn()
	if err != nil {
		return []components.NIC{}, fmt.Errorf("error getting VM NICs: %w", err)
	}

	rpc.ResetConnTimeout()

	vmNICs, err = rpc.GetVMNics(vmID)
	if err != nil {
		return []components.NIC{}, fmt.Errorf("error getting VM NICs: %w", err)
	}

	returnNICs := make([]components.NIC, 0, len(vmNICs))

	for _, NICID := range vmNICs {
		var aNIC rpc.NicInfo

		rpc.ResetConnTimeout()

		aNIC, err = rpc.GetVMNicInfo(NICID)
		if err != nil {
			return []components.NIC{}, fmt.Errorf("error getting VM NICs: %w", err)
		}

		returnNICs = append(returnNICs, components.NIC{Name: aNIC.Name, ID: NICID})
	}

	return returnNICs, nil
}

func GetVM(nameOrID string) (components.VM, error) {
	var returnVM components.VM

	var vmConfig rpc.VMConfig

	var err error

	err = util.InitRPCConn()
	if err != nil {
		return components.VM{}, fmt.Errorf("error getting VM: %w", err)
	}

	parsedUUID, err := uuid.Parse(nameOrID)
	if err != nil {
		rpc.ResetConnTimeout()

		returnVM.ID, err = rpc.GetVMId(nameOrID)
		if err != nil {
			return components.VM{}, fmt.Errorf("error getting VM: %w", err)
		}

		returnVM.Name = nameOrID
	} else {
		returnVM.ID = parsedUUID.String()

		rpc.ResetConnTimeout()

		returnVM.Name, err = rpc.GetVMName(parsedUUID.String())
		if err != nil {
			return components.VM{}, fmt.Errorf("error getting VM: %w", err)
		}
	}

	rpc.ResetConnTimeout()

	vmConfig, err = rpc.GetVMConfig(returnVM.ID)
	if err != nil {
		return components.VM{}, fmt.Errorf("error getting VM: %w", err)
	}

	returnVM.NameOrID = nameOrID
	returnVM.CPUs = vmConfig.CPU
	returnVM.Memory = vmConfig.Mem
	returnVM.Description = vmConfig.Description

	var vmState string

	var vncPort string

	rpc.ResetConnTimeout()

	vmState, vncPort, _, err = rpc.GetVMState(returnVM.ID)
	if err != nil {
		return components.VM{}, fmt.Errorf("error getting VM: %w", err)
	}

	switch vmState {
	case "running", "starting", "stopping":
		returnVM.Running = true
		if vncPort != "" && vncPort != "0" {
			returnVM.VNCPort, err = strconv.ParseUint(vncPort, 10, 64)
			if err != nil {
				return components.VM{}, fmt.Errorf("error getting VM: %w", err)
			}
		}
	default:
		returnVM.Running = false
	}

	return returnVM, nil
}

type VMDataHandler struct {
	GetVM      func(string) (components.VM, error)
	GetVMs     func() ([]components.VM, error)
	GetVMDisks func(string) ([]components.Disk, error)
	GetVMISOs  func(string) ([]components.ISO, error)
	GetVMNICs  func(string) ([]components.NIC, error)
}

func NewVMDataHandler() VMDataHandler {
	return VMDataHandler{
		GetVM:      GetVM,
		GetVMs:     GetVMs,
		GetVMDisks: GetVMDisks,
		GetVMISOs:  GetVMISOs,
		GetVMNICs:  GetVMNICs,
	}
}

func (v VMDataHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	aVM, err := v.GetVM(request.PathValue("nameOrID"))
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	VMs, err := v.GetVMs()
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	aVM.Disks, err = v.GetVMDisks(aVM.ID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	aVM.ISOs, err = v.GetVMISOs(aVM.ID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	aVM.NICs, err = v.GetVMNICs(aVM.ID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	listenHost := util.GetListenHost()
	websockifyPort := util.GetWebsockifyPort()

	templ.Handler(components.VmDataOnly(VMs, aVM, listenHost, websockifyPort)).ServeHTTP(writer, request)
}
