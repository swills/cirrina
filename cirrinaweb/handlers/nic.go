package handlers

import (
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/google/uuid"

	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinaweb/components"
	"cirrina/cirrinaweb/util"
)

type NICHandler struct {
	GetNIC  func(string) (components.NIC, error)
	GetNICs func() ([]components.NIC, error)
}

func NewNICHandler() NICHandler {
	return NICHandler{
		GetNIC:  GetNIC,
		GetNICs: GetNICs,
	}
}

func GetNIC(nameOrID string) (components.NIC, error) {
	var returnNIC components.NIC

	var nicInfo rpc.NicInfo

	var err error

	err = util.InitRPCConn()
	if err != nil {
		return components.NIC{}, fmt.Errorf("error getting NIC: %w", err)
	}

	parsedUUID, err := uuid.Parse(nameOrID)
	if err != nil {
		rpc.ResetConnTimeout()

		returnNIC.ID, err = rpc.NicNameToID(nameOrID)
		if err != nil {
			return components.NIC{}, fmt.Errorf("error getting NIC: %w", err)
		}

		returnNIC.Name = nameOrID
	} else {
		returnNIC.ID = parsedUUID.String()
	}

	rpc.ResetConnTimeout()

	nicInfo, err = rpc.GetVMNicInfo(returnNIC.ID)
	if err != nil {
		return components.NIC{}, fmt.Errorf("error getting NIC: %w", err)
	}

	returnNIC.Name = nicInfo.Name
	returnNIC.NameOrID = nicInfo.Name
	returnNIC.Description = nicInfo.Descr

	rpc.ResetConnTimeout()

	if nicInfo.Uplink != "" {
		returnNIC.Uplink, err = GetSwitch(nicInfo.Uplink)
		if err != nil {
			return components.NIC{}, fmt.Errorf("error getting NIC: %w", err)
		}
	}

	if nicInfo.VMName != "" {
		rpc.ResetConnTimeout()

		returnNIC.VM, err = GetVM(nicInfo.VMName)
		if err != nil {
			return components.NIC{}, fmt.Errorf("error getting Disk: %w", err)
		}
	}

	return returnNIC, nil
}

func DeleteNic(nameOrID string) error {
	var err error

	var nicID string

	parsedUUID, err := uuid.Parse(nameOrID)
	if err != nil {
		rpc.ResetConnTimeout()

		nicID, err = rpc.NicNameToID(nameOrID)
		if err != nil {
			return fmt.Errorf("error getting NIC: %w", err)
		}
	} else {
		nicID = parsedUUID.String()
	}

	rpc.ResetConnTimeout()

	err = rpc.RmNic(nicID)
	if err != nil {
		return fmt.Errorf("failed removing NIC: %w", err)
	}

	return nil
}

func (d NICHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	nameOrID := request.PathValue("nameOrID")

	if request.Method == http.MethodDelete {
		err := DeleteNic(nameOrID)
		if err != nil {
			writer.Header().Set("HX-Redirect", "/net/nic/"+nameOrID)
			writer.WriteHeader(http.StatusInternalServerError)

			return
		}

		writer.Header().Set("HX-Redirect", "/net/nics")
		writer.WriteHeader(http.StatusOK)

		return
	}

	aNIC, err := d.GetNIC(nameOrID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorNIC(writer, request, err)

		return
	}

	NICs, err := d.GetNICs()
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		http.Error(writer, "failed to retrieve NICs", http.StatusInternalServerError)

		return
	}

	templ.Handler(components.NICLayout(NICs, aNIC)).ServeHTTP(writer, request)
}
