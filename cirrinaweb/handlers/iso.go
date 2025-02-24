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

type ISOHandler struct {
	GetISO  func(string) (components.ISO, error)
	GetISOs func() ([]components.ISO, error)
}

func NewISOHandler() ISOHandler {
	return ISOHandler{
		GetISO:  GetISO,
		GetISOs: GetISOs,
	}
}

func GetISO(nameOrID string) (components.ISO, error) {
	var returnISO components.ISO

	var isoInfo rpc.IsoInfo

	var err error

	err = util.InitRPCConn()
	if err != nil {
		return components.ISO{}, fmt.Errorf("error getting ISO: %w", err)
	}

	parsedUUID, err := uuid.Parse(nameOrID)
	if err != nil {
		rpc.ResetConnTimeout()

		returnISO.ID, err = rpc.IsoNameToID(nameOrID)
		if err != nil {
			return components.ISO{}, fmt.Errorf("error getting ISO: %w", err)
		}

		returnISO.Name = nameOrID
	} else {
		returnISO.ID = parsedUUID.String()
	}

	rpc.ResetConnTimeout()

	isoInfo, err = rpc.GetIsoInfo(returnISO.ID)
	if err != nil {
		return components.ISO{}, fmt.Errorf("error getting ISO: %w", err)
	}

	returnISO.Name = isoInfo.Name
	returnISO.NameOrID = isoInfo.Name
	returnISO.Description = isoInfo.Descr
	returnISO.Size = humanize.IBytes(isoInfo.Size)

	var VMIDs []string

	rpc.ResetConnTimeout()

	VMIDs, err = rpc.ISOGetVMIDs(returnISO.ID)
	if err == nil {
		for _, VMID := range VMIDs {
			var aVM components.VM

			aVM, err = GetVM(VMID)
			if err != nil {
				continue
			}

			returnISO.VMs = append(returnISO.VMs, aVM)
		}
	}

	return returnISO, nil
}

func DeleteISO(nameOrID string) error {
	var err error

	var isoID string

	parsedUUID, err := uuid.Parse(nameOrID)
	if err != nil {
		rpc.ResetConnTimeout()

		isoID, err = rpc.IsoNameToID(nameOrID)
		if err != nil {
			return fmt.Errorf("error getting ISO: %w", err)
		}
	} else {
		isoID = parsedUUID.String()
	}

	rpc.ResetConnTimeout()

	err = rpc.RmIso(isoID)
	if err != nil {
		return fmt.Errorf("failed removing ISO: %w", err)
	}

	return nil
}

func (d ISOHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	nameOrID := request.PathValue("nameOrID")
	if request.Method == http.MethodDelete {
		err := DeleteISO(nameOrID)
		if err != nil {
			writer.Header().Set("HX-Redirect", "/media/iso/"+nameOrID)
			writer.WriteHeader(http.StatusInternalServerError)

			return
		}

		writer.Header().Set("HX-Redirect", "/media/isos")
		writer.WriteHeader(http.StatusOK)

		return
	}

	aISO, err := d.GetISO(nameOrID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorISO(writer, request, err)

		return
	}

	ISOs, err := d.GetISOs()
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		http.Error(writer, "failed to retrieve ISOs", http.StatusInternalServerError)

		return
	}

	templ.Handler(components.ISOLayout(ISOs, aISO)).ServeHTTP(writer, request)
}
