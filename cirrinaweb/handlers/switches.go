package handlers

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/a-h/templ"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinaweb/components"
	"cirrina/cirrinaweb/util"
)

type SwitchesHandler struct {
	GetSwitches func() ([]components.Switch, error)
}

func NewSwitchesHandler() SwitchesHandler {
	return SwitchesHandler{
		GetSwitches: GetSwitches,
	}
}

func GetSwitches() ([]components.Switch, error) {
	var err error

	err = util.InitRPCConn()
	if err != nil {
		return []components.Switch{}, fmt.Errorf("error getting Switches: %w", err)
	}

	SwitchIDs, err := rpc.GetSwitches()
	if err != nil {
		return []components.Switch{}, fmt.Errorf("error getting Switches: %w", err)
	}

	Switches := make([]components.Switch, 0, len(SwitchIDs))

	for _, SwitchID := range SwitchIDs {
		var switchInfo rpc.SwitchInfo

		rpc.ResetConnTimeout()

		switchInfo, err = rpc.GetSwitch(SwitchID)
		if err != nil {
			return []components.Switch{}, fmt.Errorf("error getting Switches: %w", err)
		}

		Switches = append(Switches, components.Switch{Name: switchInfo.Name, ID: SwitchID, Description: switchInfo.Descr})
	}

	sort.Slice(Switches, func(i, j int) bool {
		return Switches[i].Name < Switches[j].Name
	})

	return Switches, nil
}

func (v SwitchesHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	Switches, err := v.GetSwitches()
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorSwitch(writer, request, err)

		return
	}

	templ.Handler(components.Switches(Switches)).ServeHTTP(writer, request)
}

func serveErrorSwitch(writer http.ResponseWriter, request *http.Request, err error) {
	// get list of Switches for the sidebar
	switchList, getSwitchesErr := GetSwitches()
	if getSwitchesErr != nil {
		util.LogError(err, request.RemoteAddr)

		http.Error(writer, "failed to retrieve VMs", http.StatusInternalServerError)

		return
	}

	if e, ok := status.FromError(err); ok {
		switch e.Code() {
		case codes.NotFound:
			templ.Handler(
				components.SwitchNotFoundComponent(switchList),
				templ.WithStatus(http.StatusNotFound),
			).ServeHTTP(writer, request)
		case codes.OK, codes.Canceled, codes.Unknown, codes.InvalidArgument, codes.DeadlineExceeded, codes.AlreadyExists, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unimplemented, codes.Internal, codes.Unavailable, codes.DataLoss, codes.Unauthenticated: //nolint:lll
			fallthrough
		default:
			templ.Handler(components.SwitchNotFoundComponent(switchList), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request) //nolint:lll
		}
	} else {
		templ.Handler(components.SwitchNotFoundComponent(switchList), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request) //nolint:lll
	}
}
