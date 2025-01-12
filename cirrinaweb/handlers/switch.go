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

type SwitchHandler struct {
	GetSwitch   func(string) (components.Switch, error)
	GetSwitches func() ([]components.Switch, error)
}

func NewSwitchHandler() SwitchHandler {
	return SwitchHandler{
		GetSwitch:   GetSwitch,
		GetSwitches: GetSwitches,
	}
}

func GetSwitch(nameOrID string) (components.Switch, error) {
	var returnSwitch components.Switch

	var switchInfo rpc.SwitchInfo

	var err error

	err = util.InitRPCConn()
	if err != nil {
		return components.Switch{}, fmt.Errorf("error getting Switch: %w", err)
	}

	parsedUUID, err := uuid.Parse(nameOrID)
	if err != nil {
		rpc.ResetConnTimeout()

		returnSwitch.ID, err = rpc.SwitchNameToID(nameOrID)
		if err != nil {
			return components.Switch{}, fmt.Errorf("error getting Switch: %w", err)
		}

		returnSwitch.Name = nameOrID
	} else {
		returnSwitch.ID = parsedUUID.String()
	}

	rpc.ResetConnTimeout()

	switchInfo, err = rpc.GetSwitch(returnSwitch.ID)
	if err != nil {
		return components.Switch{}, fmt.Errorf("error getting Switch: %w", err)
	}

	returnSwitch.Name = switchInfo.Name
	returnSwitch.Description = switchInfo.Descr
	returnSwitch.Type = switchInfo.SwitchType
	returnSwitch.Uplink = switchInfo.Uplink

	return returnSwitch, nil
}

func (d SwitchHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	aSwitch, err := d.GetSwitch(request.PathValue("nameOrID"))
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorSwitch(writer, request, err)

		return
	}

	Switches, err := d.GetSwitches()
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		http.Error(writer, "failed to retrieve Switches", http.StatusInternalServerError)

		return
	}

	templ.Handler(components.SwitchLayout(Switches, aSwitch)).ServeHTTP(writer, request)
}
