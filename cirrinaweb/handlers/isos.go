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

type ISOsHandler struct {
	GetISOs func() ([]components.ISO, error)
}

func NewISOsHandler() ISOsHandler {
	return ISOsHandler{
		GetISOs: GetISOs,
	}
}

func GetISOs() ([]components.ISO, error) {
	var err error

	err = util.InitRPCConn()
	if err != nil {
		return []components.ISO{}, fmt.Errorf("error getting ISOs: %w", err)
	}

	ISOIDs, err := rpc.GetIsoIDs()
	if err != nil {
		return []components.ISO{}, fmt.Errorf("error getting ISOs: %w", err)
	}

	ISOs := make([]components.ISO, 0, len(ISOIDs))

	for _, ISOID := range ISOIDs {
		var isoInfo rpc.IsoInfo

		rpc.ResetConnTimeout()

		isoInfo, err = rpc.GetIsoInfo(ISOID)
		if err != nil {
			return []components.ISO{}, fmt.Errorf("error getting ISOs: %w", err)
		}

		ISOs = append(ISOs, components.ISO{Name: isoInfo.Name, ID: ISOID, Description: isoInfo.Descr})
	}

	sort.Slice(ISOs, func(i, j int) bool {
		return ISOs[i].Name < ISOs[j].Name
	})

	return ISOs, nil
}

func (v ISOsHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ISOs, err := v.GetISOs()
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorISO(writer, request, err)

		return
	}

	templ.Handler(components.ISOs(ISOs)).ServeHTTP(writer, request)
}

func serveErrorISO(writer http.ResponseWriter, request *http.Request, err error) {
	// get list of ISOs for the sidebar
	isoList, getISOsErr := GetISOs()
	if getISOsErr != nil {
		util.LogError(err, request.RemoteAddr)

		http.Error(writer, "failed to retrieve VMs", http.StatusInternalServerError)

		return
	}

	if e, ok := status.FromError(err); ok {
		switch e.Code() {
		case codes.NotFound:
			templ.Handler(
				components.ISONotFoundComponent(isoList),
				templ.WithStatus(http.StatusNotFound),
			).ServeHTTP(writer, request)
		case codes.OK, codes.Canceled, codes.Unknown, codes.InvalidArgument, codes.DeadlineExceeded, codes.AlreadyExists, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unimplemented, codes.Internal, codes.Unavailable, codes.DataLoss, codes.Unauthenticated: //nolint:lll
			fallthrough
		default:
			templ.Handler(components.ISONotFoundComponent(isoList), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request) //nolint:lll
		}
	} else {
		templ.Handler(components.ISONotFoundComponent(isoList), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request) //nolint:lll
	}
}
