package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cirrina/cirrinactl/rpc"
)

type VM struct {
	ID          string
	Name        string
	NameOrID    string
	CPUs        uint32
	Memory      uint32
	Description string
	Running     bool
	VNCPort     uint64
}

type VMHandler struct {
	GetVM  func(string) (VM, error)
	GetVMs func() ([]VM, error)
}

func NewVMHandler() VMHandler {
	return VMHandler{
		GetVM:  getVM,
		GetVMs: getVMs,
	}
}

func getVM(nameOrID string) (VM, error) {
	var returnVM VM

	var vmConfig rpc.VMConfig

	var err error

	rpc.ServerName = cirrinaServerName
	rpc.ServerPort = cirrinaServerPort
	rpc.ServerTimeout = cirrinaServerTimeout
	rpc.ResetConnTimeout()

	err = rpc.GetConn()
	if err != nil {
		return VM{}, fmt.Errorf("error getting VM: %w", err)
	}

	parsedUUID, err := uuid.Parse(nameOrID)
	if err != nil {
		rpc.ResetConnTimeout()

		returnVM.ID, err = rpc.GetVMId(nameOrID)
		if err != nil {
			return VM{}, fmt.Errorf("error getting VM: %w", err)
		}

		returnVM.Name = nameOrID
	} else {
		returnVM.ID = parsedUUID.String()

		rpc.ResetConnTimeout()

		returnVM.Name, err = rpc.GetVMName(parsedUUID.String())
		if err != nil {
			return VM{}, fmt.Errorf("error getting VM: %w", err)
		}
	}

	rpc.ResetConnTimeout()

	vmConfig, err = rpc.GetVMConfig(returnVM.ID)
	if err != nil {
		return VM{}, fmt.Errorf("error getting VM: %w", err)
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
		return VM{}, fmt.Errorf("error getting VM: %w", err)
	}

	switch vmState {
	case "STOPPED":
		returnVM.Running = false
	case "running", "starting", "stopping":
		returnVM.Running = true
		if vncPort != "" && vncPort != "0" {
			returnVM.VNCPort, err = strconv.ParseUint(vncPort, 10, 64)
			if err != nil {
				return VM{}, fmt.Errorf("error getting VM: %w", err)
			}
		}
	default:
		returnVM.Running = false
	}

	return returnVM, nil
}

func logError(err error, remoteAddr string) {
	t := time.Now()

	_, err = errorLog.WriteString(fmt.Sprintf("[%s] [server:error] [pid %d:tid %d] [client %s] %s\n",
		t.Format("Mon Jan 02 15:04:05.999999999 2006"),
		os.Getpid(),
		0,
		remoteAddr,
		err.Error(),
	))
	if err != nil {
		panic(err)
	}
}

func serveErrorVM(writer http.ResponseWriter, request *http.Request, err error) {
	// get list of VMs for the sidebar
	vmList, getVMsErr := getVMs()
	if getVMsErr != nil {
		logError(err, request.RemoteAddr)

		http.Error(writer, "failed to retrieve VMs", http.StatusInternalServerError)

		return
	}

	if e, ok := status.FromError(err); ok {
		switch e.Code() {
		case codes.NotFound:
			templ.Handler(vmNotFoundComponent(vmList), templ.WithStatus(http.StatusNotFound)).ServeHTTP(writer, request)
		case codes.OK, codes.Canceled, codes.Unknown, codes.InvalidArgument, codes.DeadlineExceeded, codes.AlreadyExists, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unimplemented, codes.Internal, codes.Unavailable, codes.DataLoss, codes.Unauthenticated: //nolint:lll
			fallthrough
		default:
			templ.Handler(vmNotFoundComponent(vmList), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request) //nolint:lll
		}
	} else {
		templ.Handler(vmNotFoundComponent(vmList), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request) //nolint:lll
	}
}

func (v VMHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	aVM, err := v.GetVM(request.PathValue("nameOrID"))
	if err != nil {
		logError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	VMs, err := v.GetVMs()
	if err != nil {
		logError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	templ.Handler(vm(VMs, aVM)).ServeHTTP(writer, request)
}

type VMStartPostHandler struct {
	GetVM func(string) (VM, error)
}

func NewVMStartHandler() VMStartPostHandler {
	return VMStartPostHandler{
		GetVM: getVM,
	}
}

func (v VM) start() error {
	rpc.ServerName = cirrinaServerName
	rpc.ServerPort = cirrinaServerPort
	rpc.ServerTimeout = cirrinaServerTimeout
	rpc.ResetConnTimeout()

	err := rpc.GetConn()
	if err != nil {
		return fmt.Errorf("error starting VM, failed to get connection: %w", err)
	}

	rpc.ResetConnTimeout()

	_, err = rpc.StartVM(v.ID)
	if err != nil {
		return fmt.Errorf("error starting VM: %w", err)
	}

	return nil
}

func (v VMStartPostHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	aVM, err := v.GetVM(request.PathValue("nameOrID"))
	if err != nil {
		logError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	err = aVM.start()
	if err != nil {
		logError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	templ.Handler(startButton(aVM)).ServeHTTP(writer, request)
}

type VMStopPostHandler struct {
	GetVM func(string) (VM, error)
}

func NewVMStopHandler() VMStopPostHandler {
	return VMStopPostHandler{
		GetVM: getVM,
	}
}

func (v VM) stop() error {
	rpc.ServerName = cirrinaServerName
	rpc.ServerPort = cirrinaServerPort
	rpc.ServerTimeout = cirrinaServerTimeout
	rpc.ResetConnTimeout()

	err := rpc.GetConn()
	if err != nil {
		return fmt.Errorf("error starting VM, failed to get connection: %w", err)
	}

	rpc.ResetConnTimeout()

	_, err = rpc.StopVM(v.ID)
	if err != nil {
		return fmt.Errorf("error stopping VM: %w", err)
	}

	return nil
}

func (v VMStopPostHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	aVM, err := v.GetVM(request.PathValue("nameOrID"))
	if err != nil {
		logError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	err = aVM.stop()
	if err != nil {
		logError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	templ.Handler(stopButton(aVM)).ServeHTTP(writer, request)
}
