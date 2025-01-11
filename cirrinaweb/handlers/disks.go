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

type DisksHandler struct {
	GetDisks func() ([]components.Disk, error)
}

func NewDisksHandler() DisksHandler {
	return DisksHandler{
		GetDisks: GetDisks,
	}
}

func GetDisks() ([]components.Disk, error) {
	var err error

	err = util.InitRPCConn()
	if err != nil {
		return []components.Disk{}, fmt.Errorf("error getting Disks: %w", err)
	}

	DiskIDs, err := rpc.GetDisks()
	if err != nil {
		return []components.Disk{}, fmt.Errorf("error getting Disks: %w", err)
	}

	Disks := make([]components.Disk, 0, len(DiskIDs))

	for _, DiskID := range DiskIDs {
		var diskInfo rpc.DiskInfo

		rpc.ResetConnTimeout()

		diskInfo, err = rpc.GetDiskInfo(DiskID)
		if err != nil {
			return []components.Disk{}, fmt.Errorf("error getting Disks: %w", err)
		}

		Disks = append(Disks, components.Disk{Name: diskInfo.Name, ID: DiskID, Description: diskInfo.Descr})
	}

	sort.Slice(Disks, func(i, j int) bool {
		return Disks[i].Name < Disks[j].Name
	})

	return Disks, nil
}

func (v DisksHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	Disks, err := v.GetDisks()
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorDisk(writer, request, err)

		return
	}

	templ.Handler(components.Disks(Disks)).ServeHTTP(writer, request)
}

func serveErrorDisk(writer http.ResponseWriter, request *http.Request, err error) {
	// get list of Disks for the sidebar
	diskList, getDisksErr := GetDisks()
	if getDisksErr != nil {
		util.LogError(err, request.RemoteAddr)

		http.Error(writer, "failed to retrieve VMs", http.StatusInternalServerError)

		return
	}

	if e, ok := status.FromError(err); ok {
		switch e.Code() {
		case codes.NotFound:
			templ.Handler(
				components.DiskNotFoundComponent(diskList),
				templ.WithStatus(http.StatusNotFound),
			).ServeHTTP(writer, request)
		case codes.OK, codes.Canceled, codes.Unknown, codes.InvalidArgument, codes.DeadlineExceeded, codes.AlreadyExists, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unimplemented, codes.Internal, codes.Unavailable, codes.DataLoss, codes.Unauthenticated: //nolint:lll
			fallthrough
		default:
			templ.Handler(components.DiskNotFoundComponent(diskList), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request) //nolint:lll
		}
	} else {
		templ.Handler(components.DiskNotFoundComponent(diskList), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request) //nolint:lll
	}
}
