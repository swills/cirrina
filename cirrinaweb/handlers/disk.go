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
	returnDisk.Type = diskInfo.DiskType
	returnDisk.DevType = diskInfo.DiskDevType
	returnDisk.Cache = diskInfo.Cache
	returnDisk.Direct = diskInfo.Direct

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

//nolint:cyclop,funlen
func (d DiskHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var err error

	switch request.Method {
	case http.MethodDelete:
		var nameOrID string

		nameOrID = request.PathValue("nameOrID")

		err = DeleteDisk(nameOrID)
		if err != nil {
			writer.Header().Set("HX-Redirect", "/media/disk/"+nameOrID)
			writer.WriteHeader(http.StatusInternalServerError)

			return
		}

		writer.Header().Set("HX-Redirect", "/media/disk")
		writer.WriteHeader(http.StatusOK)

		return
	case http.MethodGet:
		nameOrID := request.PathValue("nameOrID")

		var disks []components.Disk

		disks, err = d.GetDisks()
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			http.Error(writer, "failed to retrieve Disks", http.StatusInternalServerError)

			return
		}

		if nameOrID != "" {
			var aDisk components.Disk

			aDisk, err = d.GetDisk(nameOrID)
			if err != nil {
				util.LogError(err, request.RemoteAddr)

				serveErrorDisk(writer, request, err)

				return
			}

			templ.Handler(components.DiskLayout(disks, aDisk)).ServeHTTP(writer, request)

			return
		}

		templ.Handler(components.NewDiskLayout(disks)).ServeHTTP(writer, request)
	case http.MethodPost:
		err = request.ParseForm()
		if err != nil {
			util.LogError(err, request.RemoteAddr)
			serveErrorVM(writer, request, err)

			return
		}

		diskName := request.PostForm["name"]
		diskType := request.PostForm["type"]
		diskDevType := request.PostForm["devtype"]
		diskSizeNum := request.PostForm["size-number"]
		diskSizeUnit := request.PostForm["size-unit"]
		diskDesc := request.PostForm["desc"]
		diskCache := request.PostForm["cache"]
		diskDirect := request.PostForm["direct"]

		if diskName == nil || diskType == nil || diskDevType == nil || diskSizeNum == nil || diskSizeUnit == nil ||
			diskDesc == nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorDisk(writer, request, err)

			return
		}

		var diskIsCached bool

		if diskCache == nil {
			diskIsCached = false
		} else {
			diskIsCached = true
		}

		var diskIsDirect bool

		if diskDirect == nil {
			diskIsDirect = false
		} else {
			diskIsDirect = true
		}

		rpc.ResetConnTimeout()

		_, err = rpc.AddDisk(diskName[0], diskDesc[0], diskSizeNum[0]+diskSizeUnit[0], diskType[0], diskDevType[0],
			diskIsCached, diskIsDirect)
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorDisk(writer, request, err)

			return
		}

		http.Redirect(writer, request, "/media/disk/"+diskName[0], http.StatusSeeOther)
	}
}
