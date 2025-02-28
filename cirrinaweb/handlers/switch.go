package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	epb "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"

	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinaweb/components"
	"cirrina/cirrinaweb/util"
)

type SwitchHandler struct {
	GetSwitch   func(context.Context, string) (components.Switch, error)
	GetSwitches func(context.Context) ([]components.Switch, error)
}

func NewSwitchHandler() SwitchHandler {
	return SwitchHandler{
		GetSwitch:   GetSwitch,
		GetSwitches: GetSwitches,
	}
}

func GetSwitch(ctx context.Context, nameOrID string) (components.Switch, error) {
	var returnSwitch components.Switch

	var switchInfo rpc.SwitchInfo

	var err error

	err = util.InitRPCConn()
	if err != nil {
		return components.Switch{}, fmt.Errorf("error getting Switch: %w", err)
	}

	parsedUUID, err := uuid.Parse(nameOrID)
	if err != nil {
		returnSwitch.ID, err = rpc.SwitchNameToID(ctx, nameOrID)
		if err != nil {
			return components.Switch{}, fmt.Errorf("error getting Switch: %w", err)
		}

		returnSwitch.Name = nameOrID
	} else {
		returnSwitch.ID = parsedUUID.String()
	}

	switchInfo, err = rpc.GetSwitch(ctx, returnSwitch.ID)
	if err != nil {
		return components.Switch{}, fmt.Errorf("error getting Switch: %w", err)
	}

	returnSwitch.Name = switchInfo.Name
	returnSwitch.NameOrID = switchInfo.Name
	returnSwitch.Description = switchInfo.Descr
	returnSwitch.Type = switchInfo.SwitchType
	returnSwitch.Uplink = switchInfo.Uplink

	return returnSwitch, nil
}

func DeleteSwitch(ctx context.Context, nameOrID string) error {
	var err error

	var switchID string

	parsedUUID, err := uuid.Parse(nameOrID)
	if err != nil {
		switchID, err = rpc.SwitchNameToID(ctx, nameOrID)
		if err != nil {
			return fmt.Errorf("error getting switch: %w", err)
		}
	} else {
		switchID = parsedUUID.String()
	}

	err = rpc.DeleteSwitch(ctx, switchID)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

//nolint:gocognit,funlen,cyclop
func (d SwitchHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var err error

	switch request.Method {
	case http.MethodDelete:
		nameOrID := request.PathValue("nameOrID")

		err = DeleteSwitch(request.Context(), nameOrID)
		if err != nil {
			var errMessage string

			s := status.Convert(err)
			for _, d := range s.Details() {
				switch info := d.(type) {
				case *epb.PreconditionFailure:
					var gotDesc bool
					for _, v := range info.GetViolations() {
						gotDesc = true
						errMessage = v.GetDescription()
					}

					if !gotDesc {
						errMessage = info.String()
					}
				default:
					errMessage = fmt.Sprintf("Unexpected type: %s", info)
				}
			}

			writer.Header().Set("HX-Redirect", "/net/switch/"+nameOrID+"?err="+errMessage)
			writer.WriteHeader(http.StatusInternalServerError)

			return
		}

		writer.Header().Set("HX-Redirect", "/net/switches")
		writer.WriteHeader(http.StatusOK)

		return
	case http.MethodGet:
		nameOrID := request.PathValue("nameOrID")

		var Switches []components.Switch

		Switches, err = d.GetSwitches(request.Context())
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			http.Error(writer, "failed to retrieve Switches", http.StatusInternalServerError)

			return
		}

		if nameOrID != "" {
			var aSwitch components.Switch

			aSwitch, err = d.GetSwitch(request.Context(), nameOrID)
			if err != nil {
				util.LogError(err, request.RemoteAddr)

				serveErrorSwitch(writer, request, err)

				return
			}

			q := request.URL.Query()

			errString := q.Get("err")

			templ.Handler(components.SwitchLayout(Switches, aSwitch, errString)).ServeHTTP(writer, request) //nolint:contextcheck

			return
		}

		var uplinks []string

		uplinks, err = rpc.GetHostNics(request.Context())
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorSwitch(writer, request, err)

			return
		}

		templ.Handler(components.NewSwitchLayout(Switches, uplinks)).ServeHTTP(writer, request) //nolint:contextcheck
	case http.MethodPost:
		err = request.ParseForm()
		if err != nil {
			util.LogError(err, request.RemoteAddr)
			serveErrorSwitch(writer, request, err)

			return
		}

		switchName := request.PostForm["name"]
		switchDesc := request.PostForm["desc"]
		switchType := request.PostForm["type"]
		switchUplink := request.PostForm["uplink"]

		if switchName == nil || switchDesc == nil || switchType == nil || switchUplink == nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorSwitch(writer, request, err)

			return
		}

		_, err = rpc.AddSwitch(request.Context(), switchName[0], &switchDesc[0], &switchType[0], &switchUplink[0])
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorSwitch(writer, request, err)

			return
		}

		http.Redirect(writer, request, "/net/switch/"+switchName[0], http.StatusSeeOther)
	}
}
