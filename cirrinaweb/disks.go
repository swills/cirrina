package main

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/a-h/templ"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cirrina/cirrinactl/rpc"
)

type DisksHandler struct {
	GetDisks func() ([]Disk, error)
}

func NewDisksHandler() DisksHandler {
	return DisksHandler{
		GetDisks: getDisks,
	}
}

func getDisks() ([]Disk, error) {
	var err error

	rpc.ServerName = cirrinaServerName
	rpc.ServerPort = cirrinaServerPort
	rpc.ServerTimeout = cirrinaServerTimeout
	rpc.ResetConnTimeout()

	err = rpc.GetConn()
	if err != nil {
		return []Disk{}, fmt.Errorf("error getting Disks: %w", err)
	}

	DiskIDs, err := rpc.GetDisks()
	if err != nil {
		return []Disk{}, fmt.Errorf("error getting Disks: %w", err)
	}

	Disks := make([]Disk, 0, len(DiskIDs))

	for _, DiskID := range DiskIDs {
		var diskInfo rpc.DiskInfo

		rpc.ResetConnTimeout()

		diskInfo, err = rpc.GetDiskInfo(DiskID)
		if err != nil {
			return []Disk{}, fmt.Errorf("error getting Disks: %w", err)
		}

		Disks = append(Disks, Disk{Name: diskInfo.Name, ID: DiskID, Description: diskInfo.Descr})
	}

	sort.Slice(Disks, func(i, j int) bool {
		return Disks[i].Name < Disks[j].Name
	})

	return Disks, nil
}

func (v DisksHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	Disks, err := v.GetDisks()
	if err != nil {
		logError(err, request.RemoteAddr)

		serveErrorDisk(writer, request, err)

		return
	}

	templ.Handler(disks(Disks)).ServeHTTP(writer, request)
}

func serveErrorDisk(writer http.ResponseWriter, request *http.Request, err error) {
	// get list of Disks for the sidebar
	diskList, getDisksErr := getDisks()
	if getDisksErr != nil {
		logError(err, request.RemoteAddr)

		http.Error(writer, "failed to retrieve VMs", http.StatusInternalServerError)

		return
	}

	if e, ok := status.FromError(err); ok {
		switch e.Code() {
		case codes.NotFound:
			templ.Handler(diskNotFoundComponent(diskList), templ.WithStatus(http.StatusNotFound)).ServeHTTP(writer, request)
		case codes.OK, codes.Canceled, codes.Unknown, codes.InvalidArgument, codes.DeadlineExceeded, codes.AlreadyExists, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unimplemented, codes.Internal, codes.Unavailable, codes.DataLoss, codes.Unauthenticated: //nolint:lll
			fallthrough
		default:
			templ.Handler(diskNotFoundComponent(diskList), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request) //nolint:lll
		}
	} else {
		templ.Handler(diskNotFoundComponent(diskList), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request) //nolint:lll
	}
}
