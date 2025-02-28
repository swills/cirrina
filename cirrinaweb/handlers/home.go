package handlers

import (
	"context"
	"net/http"

	"github.com/a-h/templ"

	"cirrina/cirrinaweb/components"
	"cirrina/cirrinaweb/util"
)

type HomeHandler struct {
	GetVMs func(context.Context) ([]components.VM, error)
}

func NewHomeHandler() HomeHandler {
	return HomeHandler{
		GetVMs: GetVMs,
	}
}

func (h HomeHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	VMs, err := h.GetVMs(request.Context())
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		http.Error(writer, "failed to retrieve VMs", http.StatusInternalServerError)

		return
	}

	templ.Handler(components.Home(VMs)).ServeHTTP(writer, request) //nolint:contextcheck
}
