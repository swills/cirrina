package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"cirrina/cirrina"
	"cirrina/cirrinactl/rpc"
)

var (
	AutoStart             bool
	AutoStartChanged      bool
	AutoStartDelay        uint32
	AutoStartDelayChanged bool
	Restart               bool
	RestartChanged        bool
	RestartDelay          uint32
	RestartDelayChanged   bool
	MaxWait               uint32
	MaxWaitChanged        bool
	Cpus                  uint16
	CpusChanged           bool
	VMDescription         string
	VMDescriptionChanged  bool
	Mem                   uint32
	MemChanged            bool
	Priority              int32
	PriorityChanged       bool
	Protect               bool
	ProtectChanged        bool
	Pcpu                  uint32
	PcpuChanged           bool
	Rbps                  uint32
	RbpsChanged           bool
	Wbps                  uint32
	WbpsChanged           bool
	Riops                 uint32
	RiopsChanged          bool
	Wiops                 uint32
	WiopsChanged          bool
	Debug                 bool
	DebugChanged          bool
	DebugWait             bool
	DebugWaitChanged      bool
	DebugPort             uint32
	DebugPortChanged      bool
	Screen                bool
	ScreenSize            string
	ScreenSizeChanged     bool
	ScreenChanged         bool
	ScreenWidth           uint32
	ScreenWidthChanged    bool
	ScreenHeight          uint32
	ScreenHeightChanged   bool
	VncPort               = "AUTO"
	VncPortChanged        bool
	VncWait               bool
	VncWaitChanged        bool
	VncTablet             bool
	VncTabletChanged      bool
	VncKeyboard           = "default"
	VncKeyboardChanged    bool
	ExtraArgs             string
	ExtraArgsChanged      bool
	Sound                 bool
	SoundChanged          bool
	SoundIn               = "/dev/dsp0"
	SoundInChanged        bool
	SoundOut              = "/dev/dsp0"
	SoundOutChanged       bool
	Wire                  bool
	WireChanged           bool
	Uefi                  bool
	UefiChanged           bool
	Utc                   bool
	UtcChanged            bool
	HostBridge            bool
	HostBridgeChanged     bool
	Acpi                  bool
	AcpiChanged           bool
	Hlt                   bool
	HltChanged            bool
	Eop                   bool
	EopChanged            bool
	Dpo                   bool
	DpoChanged            bool
	Ium                   bool
	IumChanged            bool
)

var (
	Com1             bool
	Com1Changed      bool
	Com1Log          bool
	Com1LogChanged   bool
	Com1Dev          = "AUTO"
	Com1DevChanged   bool
	Com1Speed        uint32 = 115200
	Com1SpeedChanged bool
)

var (
	Com2             bool
	Com2Changed      bool
	Com2Log          bool
	Com2LogChanged   bool
	Com2Dev          = "AUTO"
	Com2DevChanged   bool
	Com2Speed        uint32 = 115200
	Com2SpeedChanged bool
)

var (
	Com3             bool
	Com3Changed      bool
	Com3Log          bool
	Com3LogChanged   bool
	Com3Dev          = "AUTO"
	Com3DevChanged   bool
	Com3Speed        uint32 = 115200
	Com3SpeedChanged bool
)

var (
	Com4             bool
	Com4Changed      bool
	Com4Log          bool
	Com4LogChanged   bool
	Com4Dev          = "AUTO"
	Com4DevChanged   bool
	Com4Speed        uint32 = 115200
	Com4SpeedChanged bool
)

var VMCreateCmd = &cobra.Command{
	Use:          "create",
	Short:        "Create a VM",
	SilenceUsage: true,
	Args: func(cmd *cobra.Command, _ []string) error {
		VMDescriptionChanged = cmd.Flags().Changed("description")
		CpusChanged = cmd.Flags().Changed("cpus")
		MemChanged = cmd.Flags().Changed("mem")

		return nil
	},
	RunE: func(_ *cobra.Command, _ []string) error {
		if VMName == "" {
			return errVMEmptyName
		}

		var lDesc *string
		var lCpus *uint32
		var lMem *uint32

		if !VMDescriptionChanged {
			lDesc = nil
		} else {
			lDesc = &VMDescription
		}

		if !CpusChanged {
			lCpus = nil
		} else {
			ltCpus := uint32(Cpus)
			lCpus = &ltCpus
		}

		if !MemChanged {
			lMem = nil
		} else {
			lMem = &Mem
		}

		// FIXME -- check request status
		_, err := rpc.AddVM(VMName, lDesc, lCpus, lMem)
		if err != nil {
			return fmt.Errorf("error adding VM: %w", err)
		}
		fmt.Print("VM Created\n")

		return nil
	},
}

var VMListCmd = &cobra.Command{
	Use:          "list",
	Short:        "List VMs",
	Long:         `List all VMs on specified server and their state`,
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		VMIDs, err := rpc.GetVMIds()
		if err != nil {
			return fmt.Errorf("error getting VM IDs: %w", err)
		}

		var names []string
		type vmListInfo struct {
			id     string
			status string
			cpu    string
			mem    string
			descr  string
		}

		vmInfos := make(map[string]vmListInfo)
		for _, VMID := range VMIDs {
			vmConfig, err := rpc.GetVMConfig(VMID)
			if err != nil {
				return fmt.Errorf("error getting VM config: %w", err)
			}

			var status string
			status, _, _, err = rpc.GetVMState(VMID)
			if err != nil {
				return fmt.Errorf("error getting VM state: %w", err)
			}
			sstatus := "Unknown"

			cpus := strconv.FormatUint(uint64(vmConfig.CPU), 10)
			var mems string
			if Humanize {
				mems = humanize.IBytes(uint64(vmConfig.Mem) * 1024 * 1024)
			} else {
				mems = strconv.FormatUint(uint64(vmConfig.Mem)*1024*1024, 10)
			}

			if status == "stopped" {
				sstatus = color.RedString("STOPPED")
			} else if status == "starting" {
				sstatus = color.YellowString("STARTING")
			} else if status == "running" {
				sstatus = color.GreenString("RUNNING")
			} else if status == "stopping" {
				sstatus = color.YellowString("STOPPING")
			}

			vmInfos[vmConfig.Name] = vmListInfo{
				id:     VMID,
				mem:    mems,
				cpu:    cpus,
				status: sstatus,
				descr:  vmConfig.Description,
			}
			names = append(names, vmConfig.Name)
		}

		sort.Strings(names)
		vmTableWriter := table.NewWriter()
		vmTableWriter.SetOutputMirror(os.Stdout)
		if ShowUUID {
			vmTableWriter.AppendHeader(table.Row{"NAME", "UUID", "CPUS", "MEMORY", "STATE", "DESCRIPTION"})
			vmTableWriter.SetColumnConfigs([]table.ColumnConfig{
				{Number: 3, Align: text.AlignRight, AlignHeader: text.AlignRight},
				{Number: 4, Align: text.AlignRight, AlignHeader: text.AlignRight},
			})
		} else {
			vmTableWriter.AppendHeader(table.Row{"NAME", "CPUS", "MEMORY", "STATE", "DESCRIPTION"})
			vmTableWriter.SetColumnConfigs([]table.ColumnConfig{
				{Number: 2, Align: text.AlignRight, AlignHeader: text.AlignRight},
				{Number: 3, Align: text.AlignRight, AlignHeader: text.AlignRight},
			})
		}
		vmTableWriter.SetStyle(myTableStyle)
		for _, name := range names {
			if ShowUUID {
				vmTableWriter.AppendRow(table.Row{
					name,
					vmInfos[name].id,
					vmInfos[name].cpu,
					vmInfos[name].mem,
					vmInfos[name].status,
					vmInfos[name].descr,
				})
			} else {
				vmTableWriter.AppendRow(table.Row{
					name,
					vmInfos[name].cpu,
					vmInfos[name].mem,
					vmInfos[name].status,
					vmInfos[name].descr,
				})
			}
		}
		vmTableWriter.Render()

		return nil
	},
}

var VMDeleteCmd = &cobra.Command{
	Use:          "delete",
	Short:        "Delete a VM",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error
		if VMID == "" {
			VMID, err = rpc.VMNameToID(VMName)
			if err != nil {
				return fmt.Errorf("error getting VM ID: %w", err)
			}
			if VMID == "" {
				return errVMNotFound
			}
		}

		var stopped bool
		stopped, err = rpc.VMStopped(VMID)
		if err != nil {
			return fmt.Errorf("failed checking VM state: %w", err)
		}
		if !stopped {
			return errVMInUseStop
		}

		// FIXME check request ID completion and status
		_, err = rpc.DeleteVM(VMID)
		if err != nil {
			return fmt.Errorf("failed deleting VM: %w", err)
		}
		fmt.Printf("VM Deleted\n")

		return nil
	},
}

var VMStopCmd = &cobra.Command{
	Use:          "stop",
	Short:        "Stop a VM",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error
		if VMID == "" {
			VMID, err = rpc.VMNameToID(VMName)
			if err != nil {
				return fmt.Errorf("failed getting VM ID: %w", err)
			}
			if VMID == "" {
				return errVMNotFound
			}
		}
		var running bool
		running, err = rpc.VMRunning(VMID)
		if err != nil {
			return fmt.Errorf("failed checking VM state: %w", err)
		}
		if !running {
			return errVMNotRunning
		}

		var vmConfig rpc.VMConfig
		vmConfig, err = rpc.GetVMConfig(VMID)
		if err != nil {
			return fmt.Errorf("failed getting VM config: %w", err)
		}

		// max wait + 10 seconds just in case
		timeout := time.Now().Add((time.Duration(int64(vmConfig.MaxWait)) * time.Second) + (time.Second * 10))

		var reqID string
		var reqStat rpc.ReqStatus
		reqID, err = rpc.StopVM(VMID)
		if err != nil {
			return fmt.Errorf("failed stopping VM: %w", err)
		}

		if !CheckReqStat {
			fmt.Printf("VM stopped\n")

			return nil
		}

		fmt.Printf("VM Stopping (timeout: %ds): ", vmConfig.MaxWait)
		for time.Now().Before(timeout) {
			reqStat, err = rpc.ReqStat(reqID)
			if err != nil {
				return fmt.Errorf("failed checking request status: %w", err)
			}
			if reqStat.Success {
				fmt.Printf(" done")
			}
			if reqStat.Complete {
				break
			}
			fmt.Printf(".")
			time.Sleep(time.Second)
			rpc.ResetConnTimeout()
		}
		fmt.Printf("\n")

		return nil
	},
}

var VMStartCmd = &cobra.Command{
	Use:          "start",
	Short:        "Start a VM",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error

		if VMID == "" {
			VMID, err = rpc.VMNameToID(VMName)
			if err != nil {
				return fmt.Errorf("failed getting VM ID: %w", err)
			}
			if VMID == "" {
				return errVMNotFound
			}
		}

		var stopped bool
		stopped, err = rpc.VMStopped(VMID)
		if err != nil {
			return fmt.Errorf("failed checking VM status: %w", err)
		}
		if !stopped {
			return errVMNotStopped
		}

		// borrow the max stop time as a timeout for waiting on startup
		var vmConfig rpc.VMConfig
		vmConfig, err = rpc.GetVMConfig(VMID)
		if err != nil {
			return fmt.Errorf("failed getting VM config: %w", err)
		}

		// max wait + 10 seconds just in case
		timeout := time.Now().Add((time.Duration(int64(vmConfig.MaxWait)) * time.Second) + (time.Second * 10))

		var reqID string
		var reqStat rpc.ReqStatus

		reqID, err = rpc.StartVM(VMID)
		if err != nil {
			return fmt.Errorf("failed starting VM: %w", err)
		}

		if !CheckReqStat {
			fmt.Print("VM started\n")

			return nil
		}

		fmt.Printf("VM Starting (timeout: %ds): ", vmConfig.MaxWait)
		for time.Now().Before(timeout) {
			reqStat, err = rpc.ReqStat(reqID)
			if err != nil {
				return fmt.Errorf("failed checking request status: %w", err)
			}
			if reqStat.Success {
				fmt.Printf(" done")
			}
			if reqStat.Complete {
				break
			}
			fmt.Printf(".")
			time.Sleep(time.Second)
			rpc.ResetConnTimeout()
		}
		fmt.Printf("\n")

		return nil
	},
}

var VMConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Reconfigure a VM",
	Args: func(cmd *cobra.Command, _ []string) error {
		VMDescriptionChanged = cmd.Flags().Changed("description")
		CpusChanged = cmd.Flags().Changed("cpus")
		MemChanged = cmd.Flags().Changed("mem")
		PriorityChanged = cmd.Flags().Changed("priority")
		ProtectChanged = cmd.Flags().Changed("protect")
		PcpuChanged = cmd.Flags().Changed("pcpu")
		RbpsChanged = cmd.Flags().Changed("rbps")
		WbpsChanged = cmd.Flags().Changed("wbps")
		RiopsChanged = cmd.Flags().Changed("riops")
		WiopsChanged = cmd.Flags().Changed("wiops")
		AutoStartChanged = cmd.Flags().Changed("autostart")
		AutoStartDelayChanged = cmd.Flags().Changed("autostart-delay")
		RestartChanged = cmd.Flags().Changed("restart")
		RestartDelayChanged = cmd.Flags().Changed("restart-delay")
		MaxWaitChanged = cmd.Flags().Changed("max-wait")
		DebugChanged = cmd.Flags().Changed("debug")
		DebugWaitChanged = cmd.Flags().Changed("debug-wait")
		DebugPortChanged = cmd.Flags().Changed("debug-port")
		ScreenChanged = cmd.Flags().Changed("screen")
		ScreenSizeChanged = cmd.Flags().Changed("screen-size")
		ScreenWidthChanged = cmd.Flags().Changed("screen-width")
		ScreenHeightChanged = cmd.Flags().Changed("screen-height")
		VncPortChanged = cmd.Flags().Changed("vnc-port")
		VncWaitChanged = cmd.Flags().Changed("vnc-wait")
		VncTabletChanged = cmd.Flags().Changed("vnc-tablet")
		VncKeyboardChanged = cmd.Flags().Changed("vnc-keyboard")
		ExtraArgsChanged = cmd.Flags().Changed("extra-args")
		SoundChanged = cmd.Flags().Changed("sound")
		SoundInChanged = cmd.Flags().Changed("sound-in")
		SoundOutChanged = cmd.Flags().Changed("sound-out")
		WireChanged = cmd.Flags().Changed("wire")
		UefiChanged = cmd.Flags().Changed("uefi")
		UtcChanged = cmd.Flags().Changed("utc")
		HostBridgeChanged = cmd.Flags().Changed("host-bridge")
		AcpiChanged = cmd.Flags().Changed("acpi")
		HltChanged = cmd.Flags().Changed("hlt")
		EopChanged = cmd.Flags().Changed("eop")
		DpoChanged = cmd.Flags().Changed("dpo")
		IumChanged = cmd.Flags().Changed("ium")
		Com1Changed = cmd.Flags().Changed("com1")
		Com1DevChanged = cmd.Flags().Changed("com1-dev")
		Com1LogChanged = cmd.Flags().Changed("com1-log")
		Com1SpeedChanged = cmd.Flags().Changed("com1-speed")
		Com2Changed = cmd.Flags().Changed("com2")
		Com2DevChanged = cmd.Flags().Changed("com2-dev")
		Com2LogChanged = cmd.Flags().Changed("com2-log")
		Com2SpeedChanged = cmd.Flags().Changed("com2-speed")
		Com3Changed = cmd.Flags().Changed("com3")
		Com3DevChanged = cmd.Flags().Changed("com3-dev")
		Com3LogChanged = cmd.Flags().Changed("com3-log")
		Com3SpeedChanged = cmd.Flags().Changed("com3-speed")
		Com4Changed = cmd.Flags().Changed("com4")
		Com4DevChanged = cmd.Flags().Changed("com4-dev")
		Com4LogChanged = cmd.Flags().Changed("com4-log")
		Com4SpeedChanged = cmd.Flags().Changed("com4-speed")

		return nil
	},
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error

		if VMID == "" {
			VMID, err = rpc.VMNameToID(VMName)
			if err != nil {
				return fmt.Errorf("failed getting VM ID: %w", err)
			}
			if VMID == "" {
				return errVMNotFound
			}
		}

		var newConfig cirrina.VMConfig
		newConfig.Id = VMID

		if VMDescriptionChanged {
			newConfig.Description = &VMDescription
		}

		if CpusChanged {
			newCPU := uint32(Cpus)
			newConfig.Cpu = &newCPU
		}

		if MemChanged {
			newMem := Mem
			if newMem < 128 {
				newMem = 128
			}
			newConfig.Mem = &newMem
		}

		if PriorityChanged {
			newPriority := Priority
			if newPriority < -20 {
				newPriority = -20
			}
			if newPriority > 20 {
				newPriority = 20
			}
			newConfig.Priority = &newPriority
		}

		if ProtectChanged {
			newConfig.Protect = &Protect
		}

		if PcpuChanged {
			newConfig.Pcpu = &Pcpu
		}

		if RbpsChanged {
			newConfig.Rbps = &Rbps
		}

		if WbpsChanged {
			newConfig.Wbps = &Wbps
		}

		if RiopsChanged {
			newConfig.Riops = &Riops
		}

		if WiopsChanged {
			newConfig.Wiops = &Wiops
		}

		if AutoStartChanged {
			newConfig.Autostart = &AutoStart
		}

		if AutoStartDelayChanged {
			newConfig.AutostartDelay = &AutoStartDelay
		}

		if RestartChanged {
			newConfig.Restart = &Restart
		}

		if RestartDelayChanged {
			newConfig.RestartDelay = &RestartDelay
		}

		if MaxWaitChanged {
			newConfig.MaxWait = &MaxWait
		}

		if ScreenChanged {
			newConfig.Screen = &Screen
		}

		if ScreenSizeChanged {
			nsw, nsh := parseScreenSize(ScreenSize)
			if nsw != 0 && nsh != 0 {
				newConfig.ScreenWidth = &nsw
				newConfig.ScreenHeight = &nsh
			}
		}

		if ScreenWidthChanged {
			newConfig.ScreenWidth = &ScreenWidth
		}

		if ScreenHeightChanged {
			newConfig.ScreenHeight = &ScreenHeight
		}

		if VncPortChanged {
			newConfig.Vncport = &VncPort
		}

		if VncWaitChanged {
			newConfig.Vncwait = &VncWait
		}

		if VncTabletChanged {
			newConfig.Tablet = &VncTablet
		}

		if VncKeyboardChanged {
			newConfig.Keyboard = &VncKeyboard
		}

		if SoundChanged {
			newConfig.Sound = &Sound
		}

		if SoundInChanged {
			newConfig.SoundIn = &SoundIn
		}

		if SoundOutChanged {
			newConfig.SoundOut = &SoundOut
		}

		if Com1Changed {
			newConfig.Com1 = &Com1
		}

		if Com1LogChanged {
			newConfig.Com1Log = &Com1Log
		}

		if Com1DevChanged {
			newConfig.Com1Dev = &Com1Dev
		}

		if Com1SpeedChanged {
			newConfig.Com1Speed = &Com1Speed
		}

		if Com2Changed {
			newConfig.Com2 = &Com2
		}

		if Com2LogChanged {
			newConfig.Com2Log = &Com2Log
		}

		if Com2DevChanged {
			newConfig.Com2Dev = &Com2Dev
		}

		if Com2SpeedChanged {
			newConfig.Com2Speed = &Com2Speed
		}

		if Com3Changed {
			newConfig.Com3 = &Com3
		}

		if Com3LogChanged {
			newConfig.Com3Log = &Com3Log
		}

		if Com3DevChanged {
			newConfig.Com3Dev = &Com3Dev
		}

		if Com3SpeedChanged {
			newConfig.Com3Speed = &Com3Speed
		}

		if Com4Changed {
			newConfig.Com4 = &Com4
		}

		if Com4LogChanged {
			newConfig.Com4Log = &Com4Log
		}

		if Com4DevChanged {
			newConfig.Com4Dev = &Com4Dev
		}

		if Com4SpeedChanged {
			newConfig.Com4Speed = &Com4Speed
		}

		if WireChanged {
			newConfig.Wireguestmem = &Wire
		}

		if UefiChanged {
			newConfig.Storeuefi = &Uefi
		}

		if UtcChanged {
			newConfig.Utc = &Utc
		}

		if HostBridgeChanged {
			newConfig.Hostbridge = &HostBridge
		}

		if AcpiChanged {
			newConfig.Acpi = &Acpi
		}

		if HltChanged {
			newConfig.Hlt = &Hlt
		}

		if EopChanged {
			newConfig.Eop = &Eop
		}

		if DpoChanged {
			newConfig.Dpo = &Dpo
		}

		if IumChanged {
			newConfig.Ium = &Ium
		}

		if DebugChanged {
			newConfig.Debug = &Debug
		}

		if DebugWaitChanged {
			newConfig.DebugWait = &DebugWait
		}

		if DebugPortChanged {
			d := strconv.FormatUint(uint64(DebugPort), 10)
			newConfig.DebugPort = &d
		}

		if ExtraArgsChanged {
			newConfig.ExtraArgs = &ExtraArgs
		}

		err = rpc.UpdateVMConfig(&newConfig)
		if err != nil {
			return fmt.Errorf("failed updating vm config: %w", err)
		}
		fmt.Printf("VM updated\n")

		return nil
	},
}

func parseScreenSize(size string) (uint32, uint32) {
	resMap := map[string][]uint32{
		"VGA":     {640, 480},
		"WVGA":    {768, 480},
		"WGA":     {800, 480},
		"FWVGA":   {854, 480},
		"SVGA":    {800, 600},
		"DVGA":    {960, 640},
		"WSVGA":   {1024, 600},
		"XGA":     {1024, 768},
		"WXGAmin": {1280, 720},
		"WXGA":    {1280, 768},
		"XGA+":    {1152, 864},
		"WXGAmax": {1280, 800},
		"WXGAHD":  {1366, 768},
		"SXGA−":   {1280, 960},
		"WSXGA":   {1440, 900},
		"WXGA+":   {1440, 900},
		"FHD":     {1920, 1080},
		"SXGA":    {1280, 1024},
		"SXGA+":   {1400, 1050},
		"qHD":     {960, 540},
		"HD+":     {1600, 900},
		"WSXGA+":  {1680, 1050},
		"UXGA":    {1600, 1200},
		"WUXGA":   {1920, 1200},
		"FHD+":    {1920, 1280},
		"QWXGA":   {2048, 1152},
		"CWSXGA":  {2880, 900},
		"TXGA":    {1920, 1400},
		"QXGA":    {2048, 1536},
		"WQXGA":   {2560, 1600},
		"QSXGA":   {2560, 2048},
		"WQHD":    {2560, 1440},
		"UWFHD":   {2560, 1080},
		"WQXGA+":  {3200, 1800},
		"QSXGA+":  {2800, 2100},
		"WQSXGA":  {3200, 2048},
		"QUXGA":   {3200, 2400},
	}

	res := resMap[size]
	if len(res) > 0 {
		return resMap[size][0], resMap[size][1]
	}

	return 0, 0
}

var VMGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get info on a VM",
	Args: func(_ *cobra.Command, _ []string) error {
		switch outputFormatString {
		case "TXT":
			outputFormat = TXT
		case "txt":
			outputFormat = TXT
		case "JSON":
			outputFormat = JSON
		case "json":
			outputFormat = JSON
		case "YAML":
			outputFormat = YAML
		case "yaml":
			outputFormat = YAML
		default:
			return errVMUnknownFormat
		}

		return nil
	},
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error

		if VMID == "" {
			VMID, err = rpc.VMNameToID(VMName)
			if err != nil {
				return fmt.Errorf("failed getting VM ID: %w", err)
			}
			if VMID == "" {
				return errVMNotFound
			}
		}
		var vmConfig rpc.VMConfig
		vmConfig, err = rpc.GetVMConfig(VMID)
		if err != nil {
			return fmt.Errorf("failed getting VM config: %w", err)
		}

		var vmState string
		var vncPort string
		var debugPort string
		vmState, vncPort, debugPort, err = rpc.GetVMState(VMID)
		if err != nil {
			return fmt.Errorf("failed getting VM state: %w", err)
		}

		type vmOutStat struct {
			Status    string `json:"Status"    yaml:"Status"`
			Vncport   string `json:"Vncport"   yaml:"Vncport"`
			Debugport string `json:"Debugport" yaml:"Debugport"`
		}
		type vmOutThing struct {
			Config rpc.VMConfig `json:"Config" yaml:"Config"`
			State  vmOutStat    `json:"State"  yaml:"State"`
		}
		vmOutSt := vmOutStat{
			Status:    vmState,
			Vncport:   vncPort,
			Debugport: debugPort,
		}
		vmOutStr := vmOutThing{
			Config: vmConfig,
			State:  vmOutSt,
		}

		switch outputFormat {
		case TXT:
			fmt.Printf("id: %v\n", VMID)
			fmt.Printf("name: %v\n", vmConfig.Name)
			fmt.Printf("desc: %v\n", vmConfig.Description)
			fmt.Printf("cpus: %v\n", vmConfig.CPU)
			fmt.Printf("mem: %v MB\n", vmConfig.Mem)
			fmt.Printf("priority: %v\n", vmConfig.Priority)
			fmt.Printf("protect: %v\n", vmConfig.Protect)
			fmt.Printf("pcpu: %v\n", vmConfig.Pcpu)
			fmt.Printf("rbps: %v\n", vmConfig.Rbps)
			fmt.Printf("Wbps: %v\n", vmConfig.Wbps)
			fmt.Printf("Riops: %v\n", vmConfig.Riops)
			fmt.Printf("Wiops: %v\n", vmConfig.Wiops)
			fmt.Printf("com1: %v\n", vmConfig.Com1)
			fmt.Printf("com1-log: %v\n", vmConfig.Com1Log)
			fmt.Printf("com1-dev: %v\n", vmConfig.Com1Dev)
			fmt.Printf("com1-speed: %v\n", vmConfig.Com1Speed)
			fmt.Printf("com2: %v\n", vmConfig.Com2)
			fmt.Printf("com2-log: %v\n", vmConfig.Com2Log)
			fmt.Printf("com2-dev: %v\n", vmConfig.Com2Dev)
			fmt.Printf("com2-speed: %v\n", vmConfig.Com2Speed)
			fmt.Printf("com3: %v\n", vmConfig.Com3)
			fmt.Printf("com3-log: %v\n", vmConfig.Com3Log)
			fmt.Printf("com3-dev: %v\n", vmConfig.Com3Dev)
			fmt.Printf("com3-speed: %v\n", vmConfig.Com3Speed)
			fmt.Printf("com4: %v\n", vmConfig.Com4)
			fmt.Printf("com4-log: %v\n", vmConfig.Com4Log)
			fmt.Printf("com4-dev: %v\n", vmConfig.Com4Dev)
			fmt.Printf("com4-speed: %v\n", vmConfig.Com4Speed)
			fmt.Printf("screen: %v\n", vmConfig.Screen)
			fmt.Printf("vnc-port: %v\n", vmConfig.Vncport)
			fmt.Printf("screen-width: %v\n", vmConfig.ScreenWidth)
			fmt.Printf("screen-height: %v\n", vmConfig.ScreenHeight)
			fmt.Printf("vnc-wait: %v\n", vmConfig.Vncwait)
			fmt.Printf("tablet-mode: %v\n", vmConfig.Tablet)
			fmt.Printf("Keyboard: %v\n", vmConfig.Keyboard)
			fmt.Printf("sound: %v\n", vmConfig.Sound)
			fmt.Printf("sound-input: %v\n", vmConfig.SoundIn)
			fmt.Printf("sound-output: %v\n", vmConfig.SoundOut)
			fmt.Printf("auto-start: %v\n", vmConfig.Autostart)
			fmt.Printf("auto-start-delay: %v\n", vmConfig.AutostartDelay)
			fmt.Printf("restart: %v\n", vmConfig.Restart)
			fmt.Printf("restart-delay: %v\n", vmConfig.RestartDelay)
			fmt.Printf("max-wait: %v\n", vmConfig.MaxWait)
			fmt.Printf("store-uefi-vars: %v\n", vmConfig.Storeuefi)
			fmt.Printf("use-utc-time: %v\n", vmConfig.Utc)
			fmt.Printf("destroy-on-power-off: %v\n", vmConfig.Dpo)
			fmt.Printf("wire-guest-mem: %v\n", vmConfig.Wireguestmem)
			fmt.Printf("use-host-bridge: %v\n", vmConfig.Hostbridge)
			fmt.Printf("generate-acpi-tables: %v\n", vmConfig.Acpi)
			fmt.Printf("exit-on-PAUSE: %v\n", vmConfig.Eop)
			fmt.Printf("ignore-unknown-MSR: %v\n", vmConfig.Ium)
			fmt.Printf("yield-on-HLT: %v\n", vmConfig.Hlt)
			fmt.Printf("debug: %v\n", vmConfig.Debug)
			fmt.Printf("debug-wait: %v\n", vmConfig.DebugWait)
			fmt.Printf("debug-port: %v\n", vmConfig.DebugPort)
			fmt.Printf("extra-args: %v\n", vmConfig.ExtraArgs)
			fmt.Printf("status: %v\n", vmState)
			fmt.Printf("vnc-port: %v\n", vncPort)
			fmt.Printf("debug-port: %v\n", debugPort)
		case JSON:
			bar, err := json.MarshalIndent(vmOutStr, "", "  ")
			if err != nil {
				return fmt.Errorf("failed generating json: %w", err)
			}
			fmt.Printf("%s\n", string(bar))
		case YAML:
			bar, err := yaml.Marshal(vmOutStr)
			if err != nil {
				return fmt.Errorf("failed generating yaml: %w", err)
			}
			fmt.Printf("%s\n", string(bar))
		default:
			fmt.Printf("unknown output format\n")
		}

		return nil
	},
}

var VMClearUefiVarsCmd = &cobra.Command{
	Use:          "clearuefivars",
	Short:        "Clear UEFI variable state",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		var err error

		if VMID == "" {
			VMID, err = rpc.VMNameToID(VMName)
			if err != nil {
				return fmt.Errorf("failed getting VM ID: %w", err)
			}
			if VMID == "" {
				return errVMNotFound
			}
		}
		var res bool
		res, err = rpc.VMClearUefiVars(VMID)
		if err != nil {
			return fmt.Errorf("failed clearning UEFI vars: %w", err)
		}
		if !res {
			return errReqFailed
		}
		fmt.Printf("UEFI Vars cleared\n")

		return nil
	},
}

var VMCmd = &cobra.Command{
	Use:   "vm",
	Short: "Create, list, modify, delete VMs",
}
