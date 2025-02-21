package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinaweb/components"
	"cirrina/cirrinaweb/util"
)

type VMHandler struct {
	GetVM      func(string) (components.VM, error)
	GetVMs     func() ([]components.VM, error)
	GetVMDisks func(string) ([]components.Disk, error)
	GetVMISOs  func(string) ([]components.ISO, error)
	GetVMNICs  func(string) ([]components.NIC, error)
}

func NewVMHandler() VMHandler {
	return VMHandler{
		GetVM:      GetVM,
		GetVMs:     GetVMs,
		GetVMDisks: GetVMDisks,
		GetVMISOs:  GetVMISOs,
		GetVMNICs:  GetVMNICs,
	}
}

func DeleteVM(nameOrID string) error {
	var err error

	var VMID string

	parsedUUID, err := uuid.Parse(nameOrID)
	if err != nil {
		rpc.ResetConnTimeout()

		VMID, err = rpc.VMNameToID(nameOrID)
		if err != nil {
			return fmt.Errorf("error getting VM: %w", err)
		}
	} else {
		VMID = parsedUUID.String()
	}

	rpc.ResetConnTimeout()

	_, err = rpc.DeleteVM(VMID)
	if err != nil {
		return fmt.Errorf("failed removing VM: %w", err)
	}

	return nil
}

func (v VMHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	nameOrID := request.PathValue("nameOrID")
	if request.Method == http.MethodDelete {
		err := DeleteVM(nameOrID)
		if err != nil {
			writer.Header().Set("HX-Redirect", "/vm/"+nameOrID)
			writer.WriteHeader(http.StatusInternalServerError)

			return
		}

		writer.Header().Set("HX-Redirect", "/vms")
		writer.WriteHeader(http.StatusOK)

		return
	}

	aVM, err := v.GetVM(nameOrID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	VMs, err := v.GetVMs()
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	aVM.Disks, err = v.GetVMDisks(aVM.ID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	aVM.ISOs, err = v.GetVMISOs(aVM.ID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	aVM.NICs, err = v.GetVMNICs(aVM.ID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	listenHost := util.GetListenHost()
	websockifyPort := util.GetWebsockifyPort()

	templ.Handler(components.Vm(VMs, aVM, listenHost, websockifyPort)).ServeHTTP(writer, request)
}

type VMStartPostHandler struct {
	GetVM func(string) (components.VM, error)
}

func NewVMStartHandler() VMStartPostHandler {
	return VMStartPostHandler{
		GetVM: GetVM,
	}
}

func (v VMStartPostHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	aVM, err := v.GetVM(request.PathValue("nameOrID"))
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	err = aVM.Start()
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	templ.Handler(components.StartButton(aVM)).ServeHTTP(writer, request)
}

type VMStopPostHandler struct {
	GetVM func(string) (components.VM, error)
}

func NewVMStopHandler() VMStopPostHandler {
	return VMStopPostHandler{
		GetVM: GetVM,
	}
}

func (v VMStopPostHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	aVM, err := v.GetVM(request.PathValue("nameOrID"))
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	err = aVM.Stop()
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	templ.Handler(components.StopButton(aVM)).ServeHTTP(writer, request)
}

func serveErrorVM(writer http.ResponseWriter, request *http.Request, err error) {
	// get list of VMs for the sidebar
	vmList, getVMsErr := GetVMs()
	if getVMsErr != nil {
		util.LogError(err, request.RemoteAddr)

		http.Error(writer, "failed to retrieve VMs", http.StatusInternalServerError)

		return
	}

	if e, ok := status.FromError(err); ok {
		switch e.Code() {
		case codes.NotFound:
			templ.Handler(
				components.VmNotFoundComponent(vmList),
				templ.WithStatus(http.StatusNotFound),
			).ServeHTTP(writer, request)
		case codes.OK, codes.Canceled, codes.Unknown, codes.InvalidArgument, codes.DeadlineExceeded, codes.AlreadyExists, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unimplemented, codes.Internal, codes.Unavailable, codes.DataLoss, codes.Unauthenticated: //nolint:lll
			fallthrough
		default:
			templ.Handler(components.VmNotFoundComponent(vmList), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request) //nolint:lll
		}
	} else {
		templ.Handler(components.VmNotFoundComponent(vmList), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(writer, request) //nolint:lll
	}
}

func GetVMDisks(vmID string) ([]components.Disk, error) {
	var vmDisks []string

	var err error

	err = util.InitRPCConn()
	if err != nil {
		return []components.Disk{}, fmt.Errorf("error getting VM Disks: %w", err)
	}

	rpc.ResetConnTimeout()

	vmDisks, err = rpc.GetVMDisks(vmID)
	if err != nil {
		return []components.Disk{}, fmt.Errorf("error getting VM Disks: %w", err)
	}

	returnDisks := make([]components.Disk, 0, len(vmDisks))

	for _, diskID := range vmDisks {
		var aDisk rpc.DiskInfo

		rpc.ResetConnTimeout()

		aDisk, err = rpc.GetDiskInfo(diskID)
		if err != nil {
			return []components.Disk{}, fmt.Errorf("error getting VM Disks: %w", err)
		}

		returnDisks = append(returnDisks, components.Disk{Name: aDisk.Name, ID: diskID, NameOrID: aDisk.Name})
	}

	return returnDisks, nil
}

func GetVMISOs(vmID string) ([]components.ISO, error) {
	var vmISOs []string

	var err error

	err = util.InitRPCConn()
	if err != nil {
		return []components.ISO{}, fmt.Errorf("error getting VM ISOs: %w", err)
	}

	rpc.ResetConnTimeout()

	vmISOs, err = rpc.GetVMIsos(vmID)
	if err != nil {
		return []components.ISO{}, fmt.Errorf("error getting VM ISOs: %w", err)
	}

	returnISOs := make([]components.ISO, 0, len(vmISOs))

	for _, isoID := range vmISOs {
		var aISO rpc.IsoInfo

		rpc.ResetConnTimeout()

		aISO, err = rpc.GetIsoInfo(isoID)
		if err != nil {
			return []components.ISO{}, fmt.Errorf("error getting VM ISOs: %w", err)
		}

		returnISOs = append(returnISOs, components.ISO{Name: aISO.Name, ID: isoID, NameOrID: aISO.Name})
	}

	return returnISOs, nil
}

func GetVMNICs(vmID string) ([]components.NIC, error) {
	var vmNICs []string

	var err error

	err = util.InitRPCConn()
	if err != nil {
		return []components.NIC{}, fmt.Errorf("error getting VM NICs: %w", err)
	}

	rpc.ResetConnTimeout()

	vmNICs, err = rpc.GetVMNics(vmID)
	if err != nil {
		return []components.NIC{}, fmt.Errorf("error getting VM NICs: %w", err)
	}

	returnNICs := make([]components.NIC, 0, len(vmNICs))

	for _, NICID := range vmNICs {
		var aNIC rpc.NicInfo

		rpc.ResetConnTimeout()

		aNIC, err = rpc.GetVMNicInfo(NICID)
		if err != nil {
			return []components.NIC{}, fmt.Errorf("error getting VM NICs: %w", err)
		}

		returnNICs = append(returnNICs, components.NIC{Name: aNIC.Name, ID: NICID, NameOrID: aNIC.Name})
	}

	return returnNICs, nil
}

//nolint:funlen
func GetVM(nameOrID string) (components.VM, error) {
	var returnVM components.VM

	var vmConfig rpc.VMConfig

	var err error

	err = util.InitRPCConn()
	if err != nil {
		return components.VM{}, fmt.Errorf("error getting VM: %w", err)
	}

	parsedUUID, err := uuid.Parse(nameOrID)
	if err != nil {
		rpc.ResetConnTimeout()

		returnVM.ID, err = rpc.GetVMId(nameOrID)
		if err != nil {
			return components.VM{}, fmt.Errorf("error getting VM: %w", err)
		}

		returnVM.Name = nameOrID
	} else {
		returnVM.ID = parsedUUID.String()

		rpc.ResetConnTimeout()

		returnVM.Name, err = rpc.GetVMName(parsedUUID.String())
		if err != nil {
			return components.VM{}, fmt.Errorf("error getting VM: %w", err)
		}
	}

	rpc.ResetConnTimeout()

	vmConfig, err = rpc.GetVMConfig(returnVM.ID)
	if err != nil {
		return components.VM{}, fmt.Errorf("error getting VM: %w", err)
	}

	returnVM.NameOrID = vmConfig.Name
	returnVM.CPUs = vmConfig.CPU
	returnVM.Memory = vmConfig.Mem
	returnVM.Description = vmConfig.Description

	returnVM.COM1 = components.COM{
		Enabled: vmConfig.Com1,
		Dev:     vmConfig.Com1Dev,
		Log:     vmConfig.Com1Log,
		Speed:   vmConfig.Com1Speed,
	}

	returnVM.COM2 = components.COM{
		Enabled: vmConfig.Com2,
		Dev:     vmConfig.Com2Dev,
		Log:     vmConfig.Com2Log,
		Speed:   vmConfig.Com2Speed,
	}

	returnVM.COM3 = components.COM{
		Enabled: vmConfig.Com3,
		Dev:     vmConfig.Com3Dev,
		Log:     vmConfig.Com3Log,
		Speed:   vmConfig.Com3Speed,
	}

	returnVM.COM4 = components.COM{
		Enabled: vmConfig.Com4,
		Dev:     vmConfig.Com4Dev,
		Log:     vmConfig.Com4Log,
		Speed:   vmConfig.Com4Speed,
	}

	returnVM.Display.Enabled = vmConfig.Screen
	returnVM.Display.Width = vmConfig.ScreenWidth
	returnVM.Display.Height = vmConfig.ScreenHeight
	returnVM.Display.TabletMode = vmConfig.Tablet
	returnVM.Display.VNCPort = vmConfig.Vncport
	returnVM.Display.VNCWait = vmConfig.Vncwait
	returnVM.Display.KeyboardLayout = vmConfig.Keyboard

	returnVM.Audio.Enabled = vmConfig.Sound
	returnVM.Audio.Input = vmConfig.SoundIn
	returnVM.Audio.Output = vmConfig.SoundOut

	returnVM.RuntimeSettings.AutoStart = vmConfig.Autostart
	returnVM.RuntimeSettings.AutoRestart = vmConfig.Restart
	returnVM.RuntimeSettings.AutoStartDelay = vmConfig.AutostartDelay
	returnVM.RuntimeSettings.AutoRestartDelay = vmConfig.RestartDelay
	returnVM.RuntimeSettings.ShutdownTimeout = vmConfig.MaxWait

	returnVM.AdvancedSettings.StoreUEFI = vmConfig.Storeuefi
	returnVM.AdvancedSettings.Wire = vmConfig.Wireguestmem
	returnVM.AdvancedSettings.ExitOnPause = vmConfig.Eop
	returnVM.AdvancedSettings.ClockUTC = vmConfig.Utc
	returnVM.AdvancedSettings.HostBridge = vmConfig.Hostbridge
	returnVM.AdvancedSettings.IgnoreUnimplementedMSR = vmConfig.Ium
	returnVM.AdvancedSettings.DestroyOnPowerOff = vmConfig.Dpo
	returnVM.AdvancedSettings.GenerateACPITables = vmConfig.Acpi
	returnVM.AdvancedSettings.UseHLT = vmConfig.Hlt
	returnVM.AdvancedSettings.StartDebugServer = vmConfig.Debug
	returnVM.AdvancedSettings.WaitDebugConn = vmConfig.DebugWait
	returnVM.AdvancedSettings.DebugPort = vmConfig.DebugPort
	returnVM.AdvancedSettings.ExtraArgs = vmConfig.ExtraArgs

	var vmState string

	var vncPort string

	rpc.ResetConnTimeout()

	vmState, vncPort, _, err = rpc.GetVMState(returnVM.ID)
	if err != nil {
		return components.VM{}, fmt.Errorf("error getting VM: %w", err)
	}

	switch vmState {
	case "running", "starting", "stopping":
		returnVM.Running = true
		if vncPort != "" && vncPort != "0" {
			returnVM.VNCPort, err = strconv.ParseUint(vncPort, 10, 64)
			if err != nil {
				return components.VM{}, fmt.Errorf("error getting VM: %w", err)
			}
		}
	default:
		returnVM.Running = false
	}

	return returnVM, nil
}

type VMDataHandler struct {
	GetVM      func(string) (components.VM, error)
	GetVMs     func() ([]components.VM, error)
	GetVMDisks func(string) ([]components.Disk, error)
	GetVMISOs  func(string) ([]components.ISO, error)
	GetVMNICs  func(string) ([]components.NIC, error)
}

func NewVMDataHandler() VMDataHandler {
	return VMDataHandler{
		GetVM:      GetVM,
		GetVMs:     GetVMs,
		GetVMDisks: GetVMDisks,
		GetVMISOs:  GetVMISOs,
		GetVMNICs:  GetVMNICs,
	}
}

func (v VMDataHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	aVM, err := v.GetVM(request.PathValue("nameOrID"))
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	VMs, err := v.GetVMs()
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	aVM.Disks, err = v.GetVMDisks(aVM.ID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	aVM.ISOs, err = v.GetVMISOs(aVM.ID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	aVM.NICs, err = v.GetVMNICs(aVM.ID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	listenHost := util.GetListenHost()
	websockifyPort := util.GetWebsockifyPort()

	templ.Handler(components.VmDataOnly(VMs, aVM, listenHost, websockifyPort)).ServeHTTP(writer, request)
}

type VMDiskHandler struct {
}

func NewVMDiskHandler() VMDiskHandler {
	return VMDiskHandler{}
}

func RemoveVMDisk(aVM components.VM, aDisk components.Disk) error {
	var diskIDs []string

	var err error

	rpc.ResetConnTimeout()

	diskIDs, err = rpc.GetVMDisks(aVM.ID)
	if err != nil {
		return fmt.Errorf("error getting disks: %w", err)
	}

	var newDiskIDs []string

	for _, id := range diskIDs {
		if id != aDisk.ID {
			newDiskIDs = append(newDiskIDs, id)
		}
	}

	var res bool

	rpc.ResetConnTimeout()

	res, err = rpc.VMSetDisks(aVM.ID, newDiskIDs)
	if err != nil {
		return ErrRemoveDisk
	}

	if !res {
		return ErrRemoveDisk
	}

	return nil
}

func (v VMDiskHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	vmNameOrID := request.PathValue("vmNameOrID")
	diskNameOrID := request.PathValue("diskNameOrID")

	var err error

	var aVM components.VM

	var aDisk components.Disk

	aVM, err = GetVM(vmNameOrID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	aDisk, err = GetDisk(diskNameOrID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	switch request.Method {
	case http.MethodDelete:
		err = RemoveVMDisk(aVM, aDisk)
		if err != nil {
			util.LogError(err, request.RemoteAddr)
		}

		writer.Header().Set("HX-Redirect", "/vm/"+vmNameOrID)
		writer.WriteHeader(http.StatusOK)

		return
	default:
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}
}

type VMISOHandler struct{}

func NewVMISOHandler() VMISOHandler {
	return VMISOHandler{}
}

func RemoveVMISO(aVM components.VM, aISO components.ISO) error {
	var isoIDs []string

	var err error

	rpc.ResetConnTimeout()

	isoIDs, err = rpc.GetVMIsos(aVM.ID)
	if err != nil {
		return fmt.Errorf("error getting isos: %w", err)
	}

	var newIsoIDs []string

	var deleted bool

	for _, id := range isoIDs {
		if !deleted && id == aISO.ID {
			deleted = true
		} else {
			newIsoIDs = append(newIsoIDs, id)
		}
	}

	if !deleted {
		return ErrRemoveISO
	}

	var res bool

	rpc.ResetConnTimeout()

	res, err = rpc.VMSetIsos(aVM.ID, newIsoIDs)
	if err != nil {
		return fmt.Errorf("failed setting VM ISOs: %w", err)
	}

	if !res {
		return ErrRemoveISO
	}

	return nil
}

func (v VMISOHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	vmNameOrID := request.PathValue("vmNameOrID")
	isoNameOrID := request.PathValue("isoNameOrID")

	var err error

	var aVM components.VM

	var aISO components.ISO

	aVM, err = GetVM(vmNameOrID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	aISO, err = GetISO(isoNameOrID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	switch request.Method {
	case http.MethodDelete:
		err = RemoveVMISO(aVM, aISO)
		if err != nil {
			util.LogError(err, request.RemoteAddr)
		}

		writer.Header().Set("HX-Redirect", "/vm/"+vmNameOrID)
		writer.WriteHeader(http.StatusOK)

		return
	default:
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}
}

type VMNICHandler struct{}

func NewVMNICHandler() VMNICHandler {
	return VMNICHandler{}
}

func RemoveVMNIC(aVM components.VM, aNIC components.NIC) error {
	var nicIDs []string

	var err error

	rpc.ResetConnTimeout()

	nicIDs, err = rpc.GetVMNics(aVM.ID)
	if err != nil {
		return fmt.Errorf("error getting NICs: %w", err)
	}

	var newNICIDs []string

	for _, id := range nicIDs {
		if id != aNIC.ID {
			newNICIDs = append(newNICIDs, id)
		}
	}

	var res bool

	rpc.ResetConnTimeout()

	res, err = rpc.VMSetNics(aVM.ID, newNICIDs)
	if err != nil {
		return fmt.Errorf("failed setting VM NICs: %w", err)
	}

	if !res {
		return ErrRemoveNIC
	}

	return nil
}

func (v VMNICHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	vmNameOrID := request.PathValue("vmNameOrID")
	nicNameOrID := request.PathValue("nicNameOrID")

	var err error

	var aVM components.VM

	var aNIC components.NIC

	aVM, err = GetVM(vmNameOrID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	aNIC, err = GetNIC(nicNameOrID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	switch request.Method {
	case http.MethodDelete:
		err = RemoveVMNIC(aVM, aNIC)
		if err != nil {
			util.LogError(err, request.RemoteAddr)
		}

		writer.Header().Set("HX-Redirect", "/vm/"+vmNameOrID)
		writer.WriteHeader(http.StatusOK)

		return
	default:
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}
}

type VMDiskAddHandler struct{}

func NewVMDiskAddHandler() VMDiskAddHandler {
	return VMDiskAddHandler{}
}

func VMAddDisk(aVM components.VM, diskName string) error {
	var err error

	var newDisk components.Disk

	var newDisks []components.Disk

	newDisk, err = GetDisk(diskName)
	if err != nil {
		return err
	}

	newDisks, err = GetVMDisks(aVM.ID)
	if err != nil {
		return err
	}

	newDisks = append(newDisks, newDisk)

	newDiskIDs := make([]string, 0, len(newDisks))

	for _, n := range newDisks {
		newDiskIDs = append(newDiskIDs, n.ID)
	}

	rpc.ResetConnTimeout()

	_, err = rpc.VMSetDisks(aVM.ID, newDiskIDs)
	if err != nil {
		return fmt.Errorf("error adding disk to VM: %w", err)
	}

	return nil
}

func GetDisksUnattached() ([]components.Disk, error) {
	var err error

	var disks []components.Disk

	var allDisks []components.Disk

	allDisks, err = GetDisks()
	if err != nil {
		return []components.Disk{}, fmt.Errorf("error getting disks: %w", err)
	}

	// only list disks not already attached to a VM
	for _, aDisk := range allDisks {
		var vmid string

		rpc.ResetConnTimeout()

		vmid, err = rpc.DiskGetVMID(aDisk.ID)
		if err != nil {
			return []components.Disk{}, fmt.Errorf("error getting disks: %w", err)
		}

		if vmid == "" {
			disks = append(disks, aDisk)
		}
	}

	return disks, nil
}

func (v VMDiskAddHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var err error

	nameOrID := request.PathValue("nameOrID")

	var aVM components.VM

	aVM, err = GetVM(nameOrID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	switch request.Method {
	case http.MethodGet:
		var disks []components.Disk

		var VMs []components.VM

		VMs, err = GetVMs()
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorVM(writer, request, err)

			return
		}

		disks, err = GetDisksUnattached()
		if err != nil {
			util.LogError(err, request.RemoteAddr)
			serveErrorVM(writer, request, err)

			return
		}

		templ.Handler(components.VmDiskAdd(nameOrID, VMs, aVM, disks)).ServeHTTP(writer, request)
	case http.MethodPost:
		err = request.ParseForm()
		if err != nil {
			util.LogError(err, request.RemoteAddr)
			serveErrorVM(writer, request, err)

			return
		}

		disksAdded := request.PostForm["disks"]

		var diskName string

		if len(disksAdded) == 0 {
			util.LogError(err, request.RemoteAddr)

			serveErrorVM(writer, request, err)

			return
		}

		diskName = disksAdded[0]

		err = VMAddDisk(aVM, diskName)
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorVM(writer, request, err)

			return
		}

		http.Redirect(writer, request, "/vm/"+nameOrID, http.StatusSeeOther)
	default:
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}
}

type VMISOAddHandler struct{}

func NewVMISOAddHandler() VMISOAddHandler {
	return VMISOAddHandler{}
}

func VMAddISO(aVM components.VM, isoName string) error {
	var err error

	var newISO components.ISO

	var newISOs []components.ISO

	newISO, err = GetISO(isoName)
	if err != nil {
		return err
	}

	newISOs, err = GetVMISOs(aVM.ID)
	if err != nil {
		return err
	}

	newISOs = append(newISOs, newISO)

	newISOIDs := make([]string, 0, len(newISOs))

	for _, n := range newISOs {
		newISOIDs = append(newISOIDs, n.ID)
	}

	rpc.ResetConnTimeout()

	_, err = rpc.VMSetIsos(aVM.ID, newISOIDs)
	if err != nil {
		return fmt.Errorf("error adding disk to VM: %w", err)
	}

	return nil
}

func (v VMISOAddHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var err error

	nameOrID := request.PathValue("nameOrID")

	var aVM components.VM

	aVM, err = GetVM(nameOrID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	switch request.Method {
	case http.MethodGet:
		var ISOs []components.ISO

		var VMs []components.VM

		VMs, err = GetVMs()
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorVM(writer, request, err)

			return
		}

		ISOs, err = GetISOs()
		if err != nil {
			util.LogError(err, request.RemoteAddr)
			serveErrorVM(writer, request, err)

			return
		}

		templ.Handler(components.VmISOAdd(nameOrID, VMs, aVM, ISOs)).ServeHTTP(writer, request)
	case http.MethodPost:
		err = request.ParseForm()
		if err != nil {
			util.LogError(err, request.RemoteAddr)
			serveErrorVM(writer, request, err)

			return
		}

		isosAdded := request.PostForm["isos"]

		var isoName string

		if len(isosAdded) == 0 {
			util.LogError(err, request.RemoteAddr)

			serveErrorVM(writer, request, err)

			return
		}

		isoName = isosAdded[0]

		err = VMAddISO(aVM, isoName)
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorVM(writer, request, err)

			return
		}

		http.Redirect(writer, request, "/vm/"+nameOrID, http.StatusSeeOther)
	default:
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}
}

type VMNICAddHandler struct{}

func NewVMNICAddHandler() VMNICAddHandler {
	return VMNICAddHandler{}
}

func VMAddNIC(aVM components.VM, nicName string) error {
	var err error

	var newNIC components.NIC

	var newNICs []components.NIC

	newNIC, err = GetNIC(nicName)
	if err != nil {
		return err
	}

	newNICs, err = GetVMNICs(aVM.ID)
	if err != nil {
		return err
	}

	newNICs = append(newNICs, newNIC)

	newNICIDs := make([]string, 0, len(newNICs))

	for _, n := range newNICs {
		newNICIDs = append(newNICIDs, n.ID)
	}

	rpc.ResetConnTimeout()

	_, err = rpc.VMSetNics(aVM.ID, newNICIDs)
	if err != nil {
		return fmt.Errorf("error adding nic to VM: %w", err)
	}

	return nil
}

func GetNICsUnattached() ([]components.NIC, error) {
	var err error

	var nics []components.NIC

	var allNICs []components.NIC

	allNICs, err = GetNICs()
	if err != nil {
		return []components.NIC{}, fmt.Errorf("error getting nics: %w", err)
	}

	// only list nics not already attached to a VM
	for _, aNIC := range allNICs {
		var vmID string

		rpc.ResetConnTimeout()

		vmID, err := rpc.GetVMNicVM(aNIC.ID)
		if err != nil {
			return []components.NIC{}, fmt.Errorf("error getting nics: %w", err)
		}

		if vmID == "" {
			nics = append(nics, aNIC)
		}
	}

	return nics, nil
}

func (v VMNICAddHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var err error

	nameOrID := request.PathValue("nameOrID")

	var aVM components.VM

	aVM, err = GetVM(nameOrID)
	if err != nil {
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}

	switch request.Method {
	case http.MethodGet:
		var nics []components.NIC

		var VMs []components.VM

		VMs, err = GetVMs()
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorVM(writer, request, err)

			return
		}

		nics, err = GetNICsUnattached()
		if err != nil {
			util.LogError(err, request.RemoteAddr)
			serveErrorVM(writer, request, err)

			return
		}

		templ.Handler(components.VmNICAdd(nameOrID, VMs, aVM, nics)).ServeHTTP(writer, request)
	case http.MethodPost:
		err = request.ParseForm()
		if err != nil {
			util.LogError(err, request.RemoteAddr)
			serveErrorVM(writer, request, err)

			return
		}

		nicsAdded := request.PostForm["nics"]

		var nicName string

		if len(nicsAdded) == 0 {
			util.LogError(err, request.RemoteAddr)

			serveErrorVM(writer, request, err)

			return
		}

		nicName = nicsAdded[0]

		err = VMAddNIC(aVM, nicName)
		if err != nil {
			util.LogError(err, request.RemoteAddr)

			serveErrorVM(writer, request, err)

			return
		}

		http.Redirect(writer, request, "/vm/"+nameOrID, http.StatusSeeOther)
	default:
		util.LogError(err, request.RemoteAddr)

		serveErrorVM(writer, request, err)

		return
	}
}
