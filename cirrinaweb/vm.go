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
	ID       string
	Name     string
	NameOrID string
	CPUs     uint64
	Running  bool
	VNCPort  uint64
}

type VMHandler struct {
	GetVM func(string) (VM, error)
}

func NewVMHandler() VMHandler {
	return VMHandler{
		GetVM: getVM,
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
	returnVM.CPUs = uint64(vmConfig.CPU)

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

func (v VMHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	aVM, err := v.GetVM(request.PathValue("nameOrID"))
	if err != nil {
		t := time.Now()

		_, err = errorLog.WriteString(fmt.Sprintf("[%s] [server:error] [pid %d:tid %d] [client %s] %s\n",
			t.Format("Mon Jan 02 15:04:05.999999999 2006"),
			os.Getpid(),
			0,
			request.RemoteAddr,
			err.Error(),
		))
		if err != nil {
			panic(err)
		}

		if e, ok := status.FromError(err); ok {
			switch e.Code() {
			case codes.NotFound:
				templ.Handler(vmNotFoundComponent(), templ.WithStatus(http.StatusNotFound)).ServeHTTP(writer, request)
			case codes.OK, codes.Canceled, codes.Unknown, codes.InvalidArgument, codes.DeadlineExceeded, codes.AlreadyExists, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unimplemented, codes.Internal, codes.Unavailable, codes.DataLoss, codes.Unauthenticated: //nolint:lll
				fallthrough
			default:
				templ.Handler(vmNotFoundComponent(), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request)
			}
		} else {
			templ.Handler(vmNotFoundComponent(), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request)
		}

		return
	}

	templ.Handler(vm(aVM)).ServeHTTP(writer, request)
}
