package handlers

import (
	"context"
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

type NICsHandler struct {
	GetNICs func(context.Context) ([]components.NIC, error)
}

func NewNICsHandler() NICsHandler {
	return NICsHandler{
		GetNICs: GetNICs,
	}
}

func GetNICs(ctx context.Context) ([]components.NIC, error) {
	var err error

	err = util.InitRPCConn()
	if err != nil {
		return []components.NIC{}, fmt.Errorf("error getting NICs: %w", err)
	}

	NICIDs, err := rpc.GetVMNicsAll(ctx)
	if err != nil {
		return []components.NIC{}, fmt.Errorf("error getting NICs: %w", err)
	}

	NICs := make([]components.NIC, 0, len(NICIDs))

	for _, NICID := range NICIDs {
		var nicName string

		nicName, err = rpc.GetVMNicName(ctx, NICID)
		if err != nil {
			return []components.NIC{}, fmt.Errorf("error getting NICs: %w", err)
		}

		NICs = append(NICs, components.NIC{Name: nicName, ID: NICID, NameOrID: nicName})
	}

	sort.Slice(NICs, func(i, j int) bool {
		return NICs[i].Name < NICs[j].Name
	})

	return NICs, nil
}

func (v NICsHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	NICs, err := v.GetNICs(request.Context())
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorNIC(writer, request, err)

		return
	}

	templ.Handler(components.NICs(NICs)).ServeHTTP(writer, request) //nolint:contextcheck
}

func serveErrorNIC(writer http.ResponseWriter, request *http.Request, err error) {
	// get list of NICs for the sidebar
	nicList, getNICsErr := GetNICs(request.Context())
	if getNICsErr != nil {
		util.LogError(err, request.RemoteAddr)

		http.Error(writer, "failed to retrieve VMs", http.StatusInternalServerError)

		return
	}

	if e, ok := status.FromError(err); ok {
		switch e.Code() {
		case codes.NotFound:
			templ.Handler(
				components.NICNotFoundComponent(nicList), //nolint:contextcheck
				templ.WithStatus(http.StatusNotFound),
			).ServeHTTP(writer, request)
		case codes.OK, codes.Canceled, codes.Unknown, codes.InvalidArgument, codes.DeadlineExceeded, codes.AlreadyExists, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unimplemented, codes.Internal, codes.Unavailable, codes.DataLoss, codes.Unauthenticated: //nolint:lll
			fallthrough
		default:
			templ.Handler(components.NICNotFoundComponent(nicList), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request) //nolint:lll,contextcheck
		}
	} else {
		templ.Handler(components.NICNotFoundComponent(nicList), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request) //nolint:lll,contextcheck
	}
}
