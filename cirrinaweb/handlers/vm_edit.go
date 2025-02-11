package handlers

import (
	"net/http"
	"strconv"

	"github.com/a-h/templ"

	"cirrina/cirrina"
	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinaweb/components"
	"cirrina/cirrinaweb/util"
)

type VMEditBasicHandler struct{}

func NewVMEditBasicHandler() VMEditBasicHandler {
	return VMEditBasicHandler{}
}

//nolint:cyclop,funlen,gocognit
func (v VMEditBasicHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var err error

	switch request.Method {
	case http.MethodGet:
		nameOrID := request.PathValue("nameOrID")

		var aVM components.VM

		aVM, err = GetVM(nameOrID)
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorVM(writer, request, err)

			return
		}

		var VMs []components.VM

		VMs, err = GetVMs()
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorVM(writer, request, err)

			return
		}

		templ.Handler(components.VMEditBasic(VMs, aVM)).ServeHTTP(writer, request)

		return
	case http.MethodPost:
		err = request.ParseForm()
		if err != nil {
			util.LogError(err, request.RemoteAddr)
			serveErrorVM(writer, request, err)

			return
		}

		nameOrID := request.PathValue("nameOrID")

		haveChanges := false

		var aVM components.VM

		aVM, err = GetVM(nameOrID)
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorVM(writer, request, err)

			return
		}

		var newConfig cirrina.VMConfig
		newConfig.Id = aVM.ID

		rpc.ResetConnTimeout()

		var oldVMConfig rpc.VMConfig

		oldVMConfig, err = rpc.GetVMConfig(aVM.ID)
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorVM(writer, request, err)

			return
		}

		newCpus := request.PostForm["cpus"]
		if newCpus != nil {
			var newCpusNum uint64

			newCpusNum, err = strconv.ParseUint(newCpus[0], 10, 32)
			if err != nil {
				util.LogError(err, request.RemoteAddr)

				serveErrorVM(writer, request, err)

				return
			}

			if oldVMConfig.CPU != uint32(newCpusNum) {
				newCPU := uint32(newCpusNum)
				newConfig.Cpu = &newCPU
				haveChanges = true
			}
		}

		newMem := request.PostForm["mem-number"]
		if newMem != nil {
			var newMemNum uint64

			newMemNum, err = strconv.ParseUint(newMem[0], 10, 32)
			if err != nil {
				util.LogError(err, request.RemoteAddr)

				serveErrorVM(writer, request, err)

				return
			}

			if oldVMConfig.Mem != uint32(newMemNum) {
				newMemI := uint32(newMemNum)
				newConfig.Mem = &newMemI
				haveChanges = true
			}
		}

		newDesc := request.PostForm["desc"]
		if newDesc != nil {
			if oldVMConfig.Description != newDesc[0] {
				newConfig.Description = &newDesc[0]
				haveChanges = true
			}
		}

		if haveChanges {
			rpc.ResetConnTimeout()

			err = rpc.UpdateVMConfig(&newConfig)
			if err != nil {
				util.LogError(err, request.RemoteAddr)

				serveErrorVM(writer, request, err)

				return
			}
		}

		http.Redirect(writer, request, "/vm/"+aVM.Name, http.StatusSeeOther)
	}
}

type VMEditSerialHandler struct{}

func NewVMEditSerialHandler() VMEditSerialHandler {
	return VMEditSerialHandler{}
}

func (v VMEditSerialHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {
}

type VMEditDisplayHandler struct{}

func NewVMEditDisplayHandler() VMEditDisplayHandler {
	return VMEditDisplayHandler{}
}

func (v VMEditDisplayHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {
}

type VMEditAudioHandler struct{}

func NewVMEditAudioHandler() VMEditAudioHandler {
	return VMEditAudioHandler{}
}

func (v VMEditAudioHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {
}

type VMEditStartHandler struct{}

func NewVMEditStartHandler() VMEditStartHandler {
	return VMEditStartHandler{}
}

func (v VMEditStartHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {
}

type VMEditAdvancedHandler struct{}

func NewVMEditAdvancedHandler() VMEditAdvancedHandler {
	return VMEditAdvancedHandler{}
}

func (v VMEditAdvancedHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {
}
