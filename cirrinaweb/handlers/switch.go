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
	returnSwitch.NameOrID = switchInfo.Name
	returnSwitch.Description = switchInfo.Descr
	returnSwitch.Type = switchInfo.SwitchType
	returnSwitch.Uplink = switchInfo.Uplink

	return returnSwitch, nil
}

func DeleteSwitch(nameOrID string) error {
	var err error

	var switchID string

	parsedUUID, err := uuid.Parse(nameOrID)
	if err != nil {
		rpc.ResetConnTimeout()

		switchID, err = rpc.SwitchNameToID(nameOrID)
		if err != nil {
			return fmt.Errorf("error getting switch: %w", err)
		}
	} else {
		switchID = parsedUUID.String()
	}

	rpc.ResetConnTimeout()

	err = rpc.DeleteSwitch(switchID)
	if err != nil {
		return fmt.Errorf("failed removing switch: %w", err)
	}

	return nil
}

func (d SwitchHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	nameOrID := request.PathValue("nameOrID")
	if request.Method == http.MethodDelete {
		err := DeleteSwitch(nameOrID)
		if err != nil {
			writer.Header().Set("HX-Redirect", "/net/switch/"+nameOrID+"?err="+err.Error())
			writer.WriteHeader(http.StatusInternalServerError)

			return
		}

		writer.Header().Set("HX-Redirect", "/net/switches")
		writer.WriteHeader(http.StatusOK)

		return
	}

	aSwitch, err := d.GetSwitch(nameOrID)
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

	q := request.URL.Query()

	errString := q.Get("err")

	templ.Handler(components.SwitchLayout(Switches, aSwitch, errString)).ServeHTTP(writer, request)
}
