package handlers

import (
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"

	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinaweb/components"
	"cirrina/cirrinaweb/util"
)

type DiskHandler struct {
	GetDisk  func(string) (components.Disk, error)
	GetDisks func() ([]components.Disk, error)
}

func NewDiskHandler() DiskHandler {
	return DiskHandler{
		GetDisk:  GetDisk,
		GetDisks: GetDisks,
	}
}

func GetDisk(nameOrID string) (components.Disk, error) {
	var returnDisk components.Disk

	var diskInfo rpc.DiskInfo

	var err error

	err = util.InitRPCConn()
	if err != nil {
		return components.Disk{}, fmt.Errorf("error getting Disk: %w", err)
	}

	parsedUUID, err := uuid.Parse(nameOrID)
	if err != nil {
		rpc.ResetConnTimeout()

		returnDisk.ID, err = rpc.DiskNameToID(nameOrID)
		if err != nil {
			return components.Disk{}, fmt.Errorf("error getting Disk: %w", err)
		}

		returnDisk.Name = nameOrID
	} else {
		returnDisk.ID = parsedUUID.String()
	}

	rpc.ResetConnTimeout()

	diskInfo, err = rpc.GetDiskInfo(returnDisk.ID)
	if err != nil {
		return components.Disk{}, fmt.Errorf("error getting Disk: %w", err)
	}

	returnDisk.NameOrID = diskInfo.Name
	returnDisk.Name = diskInfo.Name
	returnDisk.Description = diskInfo.Descr

	var diskSizeUsage rpc.DiskSizeUsage

	rpc.ResetConnTimeout()

	diskSizeUsage, err = rpc.GetDiskSizeUsage(returnDisk.ID)
	if err != nil {
		return components.Disk{}, fmt.Errorf("error getting Disk: %w", err)
	}

	returnDisk.Size = humanize.IBytes(diskSizeUsage.Size)
	returnDisk.Usage = humanize.IBytes(diskSizeUsage.Usage)

	var vmID string

	rpc.ResetConnTimeout()

	vmID, err = rpc.DiskGetVMID(returnDisk.ID)
	if err != nil {
		return components.Disk{}, fmt.Errorf("error getting Disk: %w", err)
	}

	if vmID != "" {
		rpc.ResetConnTimeout()

		returnDisk.VM, err = GetVM(vmID)
		if err != nil {
			return components.Disk{}, fmt.Errorf("error getting Disk: %w", err)
		}
	}

	return returnDisk, nil
}

func DeleteDisk(nameOrID string) error {
	var err error

	var diskID string

	parsedUUID, err := uuid.Parse(nameOrID)
	if err != nil {
		rpc.ResetConnTimeout()

		diskID, err = rpc.DiskNameToID(nameOrID)
		if err != nil {
			return fmt.Errorf("error getting disk: %w", err)
		}
	} else {
		diskID = parsedUUID.String()
	}

	rpc.ResetConnTimeout()

	err = rpc.RmDisk(diskID)
	if err != nil {
		return fmt.Errorf("failed removing disk: %w", err)
	}

	return nil
}

func (d DiskHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	nameOrID := request.PathValue("nameOrID")
	if request.Method == http.MethodDelete {
		err := DeleteDisk(nameOrID)
		if err != nil {
			writer.Header().Set("HX-Redirect", "/media/disk/"+nameOrID)
			writer.WriteHeader(http.StatusInternalServerError)

			return
		}

		writer.Header().Set("HX-Redirect", "/media/disks")
		writer.WriteHeader(http.StatusOK)

		return
	}

	aDisk, err := d.GetDisk(nameOrID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorDisk(writer, request, err)

		return
	}

	Disks, err := d.GetDisks()
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		http.Error(writer, "failed to retrieve Disks", http.StatusInternalServerError)

		return
	}

	templ.Handler(components.DiskLayout(Disks, aDisk)).ServeHTTP(writer, request)
}
