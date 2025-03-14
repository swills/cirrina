package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"

	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinaweb/components"
	"cirrina/cirrinaweb/util"
)

type NICHandler struct {
	GetNIC  func(context.Context, string) (components.NIC, error)
	GetNICs func(context.Context) ([]components.NIC, error)
}

func NewNICHandler() NICHandler {
	return NICHandler{
		GetNIC:  GetNIC,
		GetNICs: GetNICs,
	}
}

func GetNIC(ctx context.Context, nameOrID string) (components.NIC, error) {
	var returnNIC components.NIC

	var nicInfo rpc.NicInfo

	var err error

	err = util.InitRPCConn()
	if err != nil {
		return components.NIC{}, fmt.Errorf("error getting NIC: %w", err)
	}

	parsedUUID, err := uuid.Parse(nameOrID)
	if err != nil {
		returnNIC.ID, err = rpc.NicNameToID(ctx, nameOrID)
		if err != nil {
			return components.NIC{}, fmt.Errorf("error getting NIC: %w", err)
		}

		returnNIC.Name = nameOrID
	} else {
		returnNIC.ID = parsedUUID.String()
	}

	nicInfo, err = rpc.GetVMNicInfo(ctx, returnNIC.ID)
	if err != nil {
		return components.NIC{}, fmt.Errorf("error getting NIC: %w", err)
	}

	returnNIC.Name = nicInfo.Name
	returnNIC.NameOrID = nicInfo.Name
	returnNIC.Description = nicInfo.Descr
	returnNIC.Type = nicInfo.NetType
	returnNIC.DevType = nicInfo.NetDevType
	returnNIC.RateLimited = nicInfo.RateLimited
	returnNIC.RateIn = humanize.Bytes(nicInfo.RateIn)
	returnNIC.RateIn = strings.Replace(returnNIC.RateIn, "B", "b", 1) + "ps"
	returnNIC.RateOut = humanize.Bytes(nicInfo.RateOut)
	returnNIC.RateOut = strings.Replace(returnNIC.RateOut, "B", "b", 1) + "ps"

	if nicInfo.Uplink != "" {
		returnNIC.Uplink, err = GetSwitch(ctx, nicInfo.Uplink)
		if err != nil {
			return components.NIC{}, fmt.Errorf("error getting NIC: %w", err)
		}
	}

	if nicInfo.VMName != "" {
		returnNIC.VM, err = GetVM(ctx, nicInfo.VMName)
		if err != nil {
			return components.NIC{}, fmt.Errorf("error getting Disk: %w", err)
		}
	}

	return returnNIC, nil
}

func DeleteNic(ctx context.Context, nameOrID string) error {
	var err error

	var nicID string

	parsedUUID, err := uuid.Parse(nameOrID)
	if err != nil {
		nicID, err = rpc.NicNameToID(ctx, nameOrID)
		if err != nil {
			return fmt.Errorf("error getting NIC: %w", err)
		}
	} else {
		nicID = parsedUUID.String()
	}

	err = rpc.RmNic(ctx, nicID)
	if err != nil {
		return fmt.Errorf("failed removing NIC: %w", err)
	}

	return nil
}

func (d NICHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var err error

	switch request.Method {
	case http.MethodDelete:
		nameOrID := request.PathValue("nameOrID")

		err = DeleteNic(request.Context(), nameOrID)
		if err != nil {
			writer.Header().Set("HX-Redirect", "/net/nic/"+nameOrID)
			writer.WriteHeader(http.StatusInternalServerError)

			return
		}

		writer.Header().Set("HX-Redirect", "/net/nics")
		writer.WriteHeader(http.StatusOK)

		return
	case http.MethodGet:
		nameOrID := request.PathValue("nameOrID")

		var NICs []components.NIC

		NICs, err = d.GetNICs(request.Context())
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			http.Error(writer, "failed to retrieve NICs", http.StatusInternalServerError)

			return
		}

		if nameOrID != "" {
			var aNIC components.NIC

			aNIC, err = d.GetNIC(request.Context(), nameOrID)
			if err != nil {
				util.LogError(err, request.RemoteAddr)

				serveErrorNIC(writer, request, err)

				return
			}

			templ.Handler(components.NICLayout(NICs, aNIC)).ServeHTTP(writer, request) //nolint:contextcheck

			return
		}

		templ.Handler(components.NewNICLayout(NICs)).ServeHTTP(writer, request) //nolint:contextcheck
	case http.MethodPost:
		err = request.ParseForm()
		if err != nil {
			util.LogError(err, request.RemoteAddr)
			serveErrorVM(writer, request, err)

			return
		}

		nicName := request.PostForm["name"]
		nicMac := request.PostForm["mac"]
		nicType := request.PostForm["type"]
		nicDevType := request.PostForm["devtype"]

		if nicName == nil || nicMac == nil || nicType == nil || nicDevType == nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorNIC(writer, request, err)

			return
		}

		_, err = rpc.AddNic(request.Context(),
			nicName[0], "", nicMac[0], nicType[0], nicDevType[0],
			false, 0, 0, "",
		)
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorNIC(writer, request, err)

			return
		}

		http.Redirect(writer, request, "/net/nic/"+nicName[0], http.StatusSeeOther)
	}
}

func DisconnectNICUplink(ctx context.Context, aNIC components.NIC) error {
	err := rpc.SetVMNicSwitch(ctx, aNIC.ID, "")
	if err != nil {
		return fmt.Errorf("error setting nic uplink: %w", err)
	}

	return nil
}

type NICUplinkHandler struct{}

func NewNICUplinkHandler() NICUplinkHandler {
	return NICUplinkHandler{}
}

func NICAddUplink(ctx context.Context, nic components.NIC, switchName string) error {
	var err error

	switchID, err := rpc.SwitchNameToID(ctx, switchName)
	if err != nil {
		return fmt.Errorf("error getting switch id: %w", err)
	}

	if switchID == "" {
		return ErrEmptySwitch
	}

	err = rpc.SetVMNicSwitch(ctx, nic.ID, switchID)
	if err != nil {
		return fmt.Errorf("error setting nic uplink: %w", err)
	}

	return nil
}

func (n NICUplinkHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	nameOrID := request.PathValue("nameOrID")

	aNIC, err := GetNIC(request.Context(), nameOrID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorNIC(writer, request, err)

		return
	}

	switch request.Method {
	case http.MethodDelete:
		err = DisconnectNICUplink(request.Context(), aNIC)
		if err != nil {
			util.LogError(err, request.RemoteAddr)
		}

		writer.Header().Set("HX-Redirect", "/net/nic/"+nameOrID)
		writer.WriteHeader(http.StatusOK)

		return
	case http.MethodGet:
		var NICs []components.NIC

		NICs, err = GetNICs(request.Context())
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			http.Error(writer, "failed to retrieve NICs", http.StatusInternalServerError)

			return
		}

		var switches []components.Switch

		switches, err = GetSwitches(request.Context())
		if err != nil {
			util.LogError(err, request.RemoteAddr)
			serveErrorVM(writer, request, err)

			return
		}

		templ.Handler(components.NICSwitchAdd(nameOrID, NICs, switches)).ServeHTTP(writer, request) //nolint:contextcheck
	case http.MethodPost:
		err = request.ParseForm()
		if err != nil {
			util.LogError(err, request.RemoteAddr)
			serveErrorVM(writer, request, err)

			return
		}

		switchAdded := request.PostForm["switches"]

		var switchName string

		if len(switchAdded) == 0 {
			util.LogError(err, request.RemoteAddr)

			serveErrorVM(writer, request, err)

			return
		}

		switchName = switchAdded[0]

		err = NICAddUplink(request.Context(), aNIC, switchName)
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorVM(writer, request, err)

			return
		}

		http.Redirect(writer, request, "/net/nic/"+nameOrID, http.StatusSeeOther)

	default:
		util.LogError(err, request.RemoteAddr)

		serveErrorNIC(writer, request, err)

		return
	}
}
