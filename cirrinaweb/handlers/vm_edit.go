package handlers

import (
	"net/http"
	"strconv"
	"strings"

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
		if len(newCpus) > 0 {
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
		if len(newMem) > 0 {
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
		if len(newDesc) > 0 {
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

type VMEditDiskHandler struct{}

func NewVMEditDiskHandler() VMEditDiskHandler {
	return VMEditDiskHandler{}
}

func (v VMEditDiskHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
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

		aVM.Disks, err = GetVMDisks(aVM.ID)
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

		templ.Handler(components.VMEditDisk(VMs, aVM)).ServeHTTP(writer, request)

		return
	default:
		http.Redirect(writer, request, "/vm/", http.StatusSeeOther)
	}
}

type VMEditISOHandler struct{}

func NewVMEditISOHandler() VMEditISOHandler {
	return VMEditISOHandler{}
}

func (v VMEditISOHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
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

		aVM.ISOs, err = GetVMISOs(aVM.ID)
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

		templ.Handler(components.VMEditISO(VMs, aVM)).ServeHTTP(writer, request)

		return
	default:
		http.Redirect(writer, request, "/vm/", http.StatusSeeOther)
	}
}

type VMEditNICHandler struct{}

func NewVMEditNICHandler() VMEditNICHandler {
	return VMEditNICHandler{}
}

func (v VMEditNICHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
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

		aVM.NICs, err = GetVMNICs(aVM.ID)
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

		templ.Handler(components.VMEditNIC(VMs, aVM)).ServeHTTP(writer, request)

		return
	default:
		http.Redirect(writer, request, "/vm/", http.StatusSeeOther)
	}
}

type VMEditSerialHandler struct{}

func NewVMEditSerialHandler() VMEditSerialHandler {
	return VMEditSerialHandler{}
}

//nolint:gocognit,cyclop,gocyclo,funlen,maintidx
func (v VMEditSerialHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
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

		templ.Handler(components.VMEditSerial(VMs, aVM)).ServeHTTP(writer, request)

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

		// com1

		newCom1Enabled := request.PostForm["com1enabled"]

		var newCom1EnabledB bool

		if newCom1Enabled != nil {
			newCom1EnabledB = true
		} else {
			newCom1EnabledB = false
		}

		if oldVMConfig.Com1 != newCom1EnabledB {
			newConfig.Com1 = &newCom1EnabledB
			haveChanges = true
		}

		newCom1Dev := request.PostForm["com1dev"]
		if len(newCom1Dev) > 0 {
			if oldVMConfig.Com1Dev != newCom1Dev[0] {
				newConfig.Com1Dev = &newCom1Dev[0]
				haveChanges = true
			}
		}

		newCom1Log := request.PostForm["com1log"]

		var newCom1LogB bool
		if newCom1Log != nil {
			newCom1LogB = true
		} else {
			newCom1LogB = false
		}

		if oldVMConfig.Com1Log != newCom1LogB {
			newConfig.Com1Log = &newCom1LogB
			haveChanges = true
		}

		newCom1Speed := request.PostForm["com1speed"]

		if len(newCom1Speed) > 0 {
			var newCom1SpeedNum uint64

			newCom1SpeedNum, err = strconv.ParseUint(newCom1Speed[0], 10, 32)
			if err != nil {
				util.LogError(err, request.RemoteAddr)

				serveErrorVM(writer, request, err)

				return
			}

			if oldVMConfig.Com1Speed != uint32(newCom1SpeedNum) {
				newCom1SpeedI := uint32(newCom1SpeedNum)
				newConfig.Com1Speed = &newCom1SpeedI
				haveChanges = true
			}
		}

		// com2

		newCom2Enabled := request.PostForm["com2enabled"]

		var newCom2EnabledB bool

		if newCom2Enabled != nil {
			newCom2EnabledB = true
		} else {
			newCom2EnabledB = false
		}

		if oldVMConfig.Com2 != newCom2EnabledB {
			newConfig.Com2 = &newCom2EnabledB
			haveChanges = true
		}

		newCom2Dev := request.PostForm["com2dev"]
		if len(newCom2Dev) > 0 {
			if oldVMConfig.Com2Dev != newCom2Dev[0] {
				newConfig.Com2Dev = &newCom2Dev[0]
				haveChanges = true
			}
		}

		newCom2Log := request.PostForm["com2log"]

		var newCom2LogB bool
		if newCom2Log != nil {
			newCom2LogB = true
		} else {
			newCom2LogB = false
		}

		if oldVMConfig.Com2Log != newCom2LogB {
			newConfig.Com2Log = &newCom2LogB
			haveChanges = true
		}

		newCom2Speed := request.PostForm["com2speed"]

		if len(newCom2Speed) > 0 {
			var newCom2SpeedNum uint64

			newCom2SpeedNum, err = strconv.ParseUint(newCom2Speed[0], 10, 32)
			if err != nil {
				util.LogError(err, request.RemoteAddr)

				serveErrorVM(writer, request, err)

				return
			}

			if oldVMConfig.Com2Speed != uint32(newCom2SpeedNum) {
				newCom2SpeedI := uint32(newCom2SpeedNum)
				newConfig.Com2Speed = &newCom2SpeedI
				haveChanges = true
			}
		}

		// com3

		newCom3Enabled := request.PostForm["com3enabled"]

		var newCom3EnabledB bool

		if newCom3Enabled != nil {
			newCom3EnabledB = true
		} else {
			newCom3EnabledB = false
		}

		if oldVMConfig.Com3 != newCom3EnabledB {
			newConfig.Com3 = &newCom3EnabledB
			haveChanges = true
		}

		newCom3Dev := request.PostForm["com3dev"]
		if len(newCom3Dev) > 0 {
			if oldVMConfig.Com3Dev != newCom3Dev[0] {
				newConfig.Com3Dev = &newCom3Dev[0]
				haveChanges = true
			}
		}

		newCom3Log := request.PostForm["com3log"]

		var newCom3LogB bool
		if newCom3Log != nil {
			newCom3LogB = true
		} else {
			newCom3LogB = false
		}

		if oldVMConfig.Com3Log != newCom3LogB {
			newConfig.Com3Log = &newCom3LogB
			haveChanges = true
		}

		newCom3Speed := request.PostForm["com3speed"]

		if len(newCom3Speed) > 0 {
			var newCom3SpeedNum uint64

			newCom3SpeedNum, err = strconv.ParseUint(newCom3Speed[0], 10, 32)
			if err != nil {
				util.LogError(err, request.RemoteAddr)

				serveErrorVM(writer, request, err)

				return
			}

			if oldVMConfig.Com3Speed != uint32(newCom3SpeedNum) {
				newCom3SpeedI := uint32(newCom3SpeedNum)
				newConfig.Com3Speed = &newCom3SpeedI
				haveChanges = true
			}
		}

		// com4

		newCom4Enabled := request.PostForm["com4enabled"]

		var newCom4EnabledB bool

		if newCom4Enabled != nil {
			newCom4EnabledB = true
		} else {
			newCom4EnabledB = false
		}

		if oldVMConfig.Com4 != newCom4EnabledB {
			newConfig.Com4 = &newCom4EnabledB
			haveChanges = true
		}

		newCom4Dev := request.PostForm["com4dev"]
		if len(newCom4Dev) > 0 {
			if oldVMConfig.Com4Dev != newCom4Dev[0] {
				newConfig.Com4Dev = &newCom4Dev[0]
				haveChanges = true
			}
		}

		newCom4Log := request.PostForm["com4log"]

		var newCom4LogB bool
		if newCom4Log != nil {
			newCom4LogB = true
		} else {
			newCom4LogB = false
		}

		if oldVMConfig.Com4Log != newCom4LogB {
			newConfig.Com4Log = &newCom4LogB
			haveChanges = true
		}

		newCom4Speed := request.PostForm["com4speed"]

		if len(newCom4Speed) > 0 {
			var newCom4SpeedNum uint64

			newCom4SpeedNum, err = strconv.ParseUint(newCom4Speed[0], 10, 32)
			if err != nil {
				util.LogError(err, request.RemoteAddr)

				serveErrorVM(writer, request, err)

				return
			}

			if oldVMConfig.Com4Speed != uint32(newCom4SpeedNum) {
				newCom4SpeedI := uint32(newCom4SpeedNum)
				newConfig.Com4Speed = &newCom4SpeedI
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
	default:
		http.Redirect(writer, request, "/vm/", http.StatusSeeOther)
	}
}

type VMEditDisplayHandler struct{}

func NewVMEditDisplayHandler() VMEditDisplayHandler {
	return VMEditDisplayHandler{}
}

//nolint:gocognit,cyclop,funlen
func (v VMEditDisplayHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
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

		templ.Handler(components.VMEditDisplay(VMs, aVM)).ServeHTTP(writer, request)

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

		displayEnabled := request.PostForm["displayenabled"]

		var displayEnabledB bool

		if len(displayEnabled) > 0 {
			displayEnabledB = true
		} else {
			displayEnabledB = false
		}

		if oldVMConfig.Screen != displayEnabledB {
			newConfig.Screen = &displayEnabledB
			haveChanges = true
		}

		vncPortNew := request.PostForm["vncport"]

		if len(vncPortNew) > 0 {
			vncPortNewStr := strings.ToUpper(vncPortNew[0])
			if vncPortNewStr == "AUTO" {
				newConfig.Vncport = &vncPortNewStr
				haveChanges = true
			}

			var vncPortNewPortNum uint64

			vncPortNewPortNum, err = strconv.ParseUint(vncPortNewStr, 10, 32)
			if err == nil && vncPortNewPortNum < 65536 {
				newConfig.Vncport = &vncPortNewStr
				haveChanges = true
			}
		}

		screenWidthNew := request.PostForm["screenwidth"]

		if len(screenWidthNew) > 0 {
			var newWidthNum uint64

			newWidthNum, err = strconv.ParseUint(screenWidthNew[0], 10, 32)
			if err == nil && newWidthNum <= 3840 {
				n := uint32(newWidthNum)
				newConfig.ScreenWidth = &n
				haveChanges = true
			}
		}

		screenHeightNew := request.PostForm["screenheight"]

		if len(screenHeightNew) > 0 {
			var newHeightNum uint64

			newHeightNum, err = strconv.ParseUint(screenHeightNew[0], 10, 32)
			if err == nil && newHeightNum <= 3840 {
				n := uint32(newHeightNum)
				newConfig.ScreenHeight = &n
				haveChanges = true
			}
		}

		vncwaitEnabled := request.PostForm["vncwait"]

		var vncwaitEnabledB bool

		if vncwaitEnabled != nil {
			vncwaitEnabledB = true
		} else {
			vncwaitEnabledB = false
		}

		if oldVMConfig.Vncwait != vncwaitEnabledB {
			newConfig.Vncwait = &vncwaitEnabledB
			haveChanges = true
		}

		vncTabletEnabled := request.PostForm["vnctablet"]

		var vncTabletEnabledB bool

		if vncTabletEnabled != nil {
			vncTabletEnabledB = true
		} else {
			vncTabletEnabledB = false
		}

		if oldVMConfig.Tablet != vncTabletEnabledB {
			newConfig.Tablet = &vncTabletEnabledB
			haveChanges = true
		}

		newKeyboardLayout := request.PostForm["keyboardlayout"]

		if len(newKeyboardLayout) > 0 {
			if oldVMConfig.Keyboard != newKeyboardLayout[0] {
				newConfig.Keyboard = &newKeyboardLayout[0]
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
	default:
		http.Redirect(writer, request, "/vm/", http.StatusSeeOther)
	}
}

type VMEditAudioHandler struct{}

func NewVMEditAudioHandler() VMEditAudioHandler {
	return VMEditAudioHandler{}
}

//nolint:funlen
func (v VMEditAudioHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
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

		templ.Handler(components.VMEditAudio(VMs, aVM)).ServeHTTP(writer, request)

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

		audioEnabled := request.PostForm["audioenabled"]

		var audioEnabledB bool

		if len(audioEnabled) > 0 {
			audioEnabledB = true
		} else {
			audioEnabledB = false
		}

		if oldVMConfig.Sound != audioEnabledB {
			newConfig.Sound = &audioEnabledB
			haveChanges = true
		}

		audioInputNew := request.PostForm["audioinput"]

		if len(audioInputNew) > 0 {
			newConfig.SoundIn = &audioInputNew[0]
			haveChanges = true
		}

		audioOutputNew := request.PostForm["audiooutput"]

		if len(audioOutputNew) > 0 {
			newConfig.SoundOut = &audioOutputNew[0]
			haveChanges = true
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
	default:
		http.Redirect(writer, request, "/vm/", http.StatusSeeOther)
	}
}

type VMEditStartHandler struct{}

func NewVMEditStartHandler() VMEditStartHandler {
	return VMEditStartHandler{}
}

//nolint:gocognit,cyclop,funlen
func (v VMEditStartHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
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

		templ.Handler(components.VMEditStart(VMs, aVM)).ServeHTTP(writer, request)

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

		// auto start
		autostartEnabled := request.PostForm["autostartenabled"]

		var autostartEnabledB bool

		if len(autostartEnabled) > 0 {
			autostartEnabledB = true
		} else {
			autostartEnabledB = false
		}

		if oldVMConfig.Autostart != autostartEnabledB {
			newConfig.Autostart = &autostartEnabledB
			haveChanges = true
		}

		autoStartDelayNew := request.PostForm["autostartdelay"]

		if len(autoStartDelayNew) > 0 {
			var autoStartDelayNewNum uint64

			autoStartDelayNewNum, err = strconv.ParseUint(autoStartDelayNew[0], 10, 32)
			if err == nil {
				n := uint32(autoStartDelayNewNum)
				newConfig.AutostartDelay = &n
				haveChanges = true
			}
		}

		// auto restart
		autorestartEnabled := request.PostForm["autorestartenabled"]

		var autorestartEnabledB bool

		if len(autorestartEnabled) > 0 {
			autorestartEnabledB = true
		} else {
			autorestartEnabledB = false
		}

		if oldVMConfig.Restart != autorestartEnabledB {
			newConfig.Restart = &autorestartEnabledB
			haveChanges = true
		}

		autoRestartDelayNew := request.PostForm["autorestartdelay"]

		if len(autoRestartDelayNew) > 0 {
			var autoRestartDelayNewNum uint64

			autoRestartDelayNewNum, err = strconv.ParseUint(autoRestartDelayNew[0], 10, 32)
			if err == nil {
				n := uint32(autoRestartDelayNewNum)
				newConfig.RestartDelay = &n
				haveChanges = true
			}
		}

		// shutdown timeout

		shutdownTimeoutNew := request.PostForm["shutdowntimeout"]

		if len(shutdownTimeoutNew) > 0 {
			var shutdownTimeoutNewNum uint64

			shutdownTimeoutNewNum, err = strconv.ParseUint(shutdownTimeoutNew[0], 10, 32)
			if err == nil {
				n := uint32(shutdownTimeoutNewNum)
				newConfig.MaxWait = &n
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
	default:
		http.Redirect(writer, request, "/vm/", http.StatusSeeOther)
	}
}

type VMEditAdvancedHandler struct{}

func NewVMEditAdvancedHandler() VMEditAdvancedHandler {
	return VMEditAdvancedHandler{}
}

//nolint:gocognit,gocyclo,cyclop,funlen,maintidx
func (v VMEditAdvancedHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
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

		templ.Handler(components.VMEditAdvanced(VMs, aVM)).ServeHTTP(writer, request)

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

		// storeuefivars
		storeuefivarsEnabled := request.PostForm["storeuefivarsenabled"]

		var storeuefivarsEnabledB bool

		if len(storeuefivarsEnabled) > 0 {
			storeuefivarsEnabledB = true
		} else {
			storeuefivarsEnabledB = false
		}

		if oldVMConfig.Storeuefi != storeuefivarsEnabledB {
			newConfig.Storeuefi = &storeuefivarsEnabledB
			haveChanges = true
		}

		// clockutc
		clockutcEnabled := request.PostForm["clockutcenabled"]

		var clockutcEnabledB bool

		if len(clockutcEnabled) > 0 {
			clockutcEnabledB = true
		} else {
			clockutcEnabledB = false
		}

		if oldVMConfig.Utc != clockutcEnabledB {
			newConfig.Utc = &clockutcEnabledB
			haveChanges = true
		}

		// dpo
		dpoEnabled := request.PostForm["dpoenabled"]

		var dpoEnabledB bool

		if len(dpoEnabled) > 0 {
			dpoEnabledB = true
		} else {
			dpoEnabledB = false
		}

		if oldVMConfig.Dpo != dpoEnabledB {
			newConfig.Dpo = &dpoEnabledB
			haveChanges = true
		}

		// wire
		wireEnabled := request.PostForm["wireenabled"]

		var wireEnabledB bool

		if len(wireEnabled) > 0 {
			wireEnabledB = true
		} else {
			wireEnabledB = false
		}

		if oldVMConfig.Wireguestmem != wireEnabledB {
			newConfig.Wireguestmem = &wireEnabledB
			haveChanges = true
		}

		// hostbridge
		hostbridgeEnabled := request.PostForm["hostbridgeenabled"]

		var hostbridgeEnabledB bool

		if len(hostbridgeEnabled) > 0 {
			hostbridgeEnabledB = true
		} else {
			hostbridgeEnabledB = false
		}

		if oldVMConfig.Hostbridge != hostbridgeEnabledB {
			newConfig.Hostbridge = &hostbridgeEnabledB
			haveChanges = true
		}

		// acpi
		acpiEnabled := request.PostForm["acpienabled"]

		var acpiEnabledB bool

		if len(acpiEnabled) > 0 {
			acpiEnabledB = true
		} else {
			acpiEnabledB = false
		}

		if oldVMConfig.Acpi != acpiEnabledB {
			newConfig.Acpi = &acpiEnabledB
			haveChanges = true
		}

		// eop
		eopEnabled := request.PostForm["eopenabled"]

		var eopEnabledB bool

		if len(eopEnabled) > 0 {
			eopEnabledB = true
		} else {
			eopEnabledB = false
		}

		if oldVMConfig.Eop != eopEnabledB {
			newConfig.Eop = &eopEnabledB
			haveChanges = true
		}

		// ium
		iumEnabled := request.PostForm["iumenabled"]

		var iumEnabledB bool

		if len(iumEnabled) > 0 {
			iumEnabledB = true
		} else {
			iumEnabledB = false
		}

		if oldVMConfig.Ium != iumEnabledB {
			newConfig.Ium = &iumEnabledB
			haveChanges = true
		}

		// hlt
		hltEnabled := request.PostForm["hltenabled"]

		var hltEnabledB bool

		if len(hltEnabled) > 0 {
			hltEnabledB = true
		} else {
			hltEnabledB = false
		}

		if oldVMConfig.Hlt != hltEnabledB {
			newConfig.Hlt = &hltEnabledB
			haveChanges = true
		}

		// debug
		debugEnabled := request.PostForm["debugenabled"]

		var debugEnabledB bool

		if len(debugEnabled) > 0 {
			debugEnabledB = true
		} else {
			debugEnabledB = false
		}

		if oldVMConfig.Debug != debugEnabledB {
			newConfig.Debug = &debugEnabledB
			haveChanges = true
		}

		// debugwait
		debugwaitEnabled := request.PostForm["debugwaitenabled"]

		var debugwaitEnabledB bool

		if len(debugwaitEnabled) > 0 {
			debugwaitEnabledB = true
		} else {
			debugwaitEnabledB = false
		}

		if oldVMConfig.DebugWait != debugwaitEnabledB {
			newConfig.DebugWait = &debugwaitEnabledB
			haveChanges = true
		}

		// debugport
		debugPortNew := request.PostForm["debugport"]

		if len(debugPortNew) > 0 {
			debugPortNewStr := strings.ToUpper(debugPortNew[0])
			if debugPortNewStr == "AUTO" {
				newConfig.DebugPort = &debugPortNewStr
				haveChanges = true
			}

			var debugPortNewNum uint64

			debugPortNewNum, err = strconv.ParseUint(debugPortNewStr, 10, 32)
			if err == nil && debugPortNewNum < 65536 {
				newConfig.DebugPort = &debugPortNewStr
				haveChanges = true
			}
		}

		// extra args
		extraArgs := request.PostForm["extraargs"]
		if len(extraArgs) > 0 {
			if oldVMConfig.ExtraArgs != extraArgs[0] {
				newConfig.ExtraArgs = &extraArgs[0]
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
	default:
		http.Redirect(writer, request, "/vm/", http.StatusSeeOther)
	}
}
