package cmd

import (
	"encoding/json"
	"errors"
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

var AutoStart bool
var AutoStartChanged bool
var AutoStartDelay uint32
var AutoStartDelayChanged bool
var Restart bool
var RestartChanged bool
var RestartDelay uint32
var RestartDelayChanged bool
var MaxWait uint32
var MaxWaitChanged bool
var Cpus uint16
var CpusChanged bool
var VmDescription string
var VmDescriptionChanged bool
var Mem uint32
var MemChanged bool
var Priority int32
var PriorityChanged bool
var Protect bool
var ProtectChanged bool
var Pcpu uint32
var PcpuChanged bool
var Rbps uint32
var RbpsChanged bool
var Wbps uint32
var WbpsChanged bool
var Riops uint32
var RiopsChanged bool
var Wiops uint32
var WiopsChanged bool
var Debug bool
var DebugChanged bool
var DebugWait bool
var DebugWaitChanged bool
var DebugPort uint32
var DebugPortChanged bool
var Screen bool
var ScreenChanged bool
var ScreenWidth uint32
var ScreenWidthChanged bool
var ScreenHeight uint32
var ScreenHeightChanged bool
var VncPort = "AUTO"
var VncPortChanged bool
var VncWait bool
var VncWaitChanged bool
var VncTablet bool
var VncTabletChanged bool
var VncKeyboard = "default"
var VncKeyboardChanged bool
var ExtraArgs string
var ExtraArgsChanged bool
var Sound bool
var SoundChanged bool
var SoundIn = "/dev/dsp0"
var SoundInChanged bool
var SoundOut = "/dev/dsp0"
var SoundOutChanged bool
var Wire bool
var WireChanged bool
var Uefi bool
var UefiChanged bool
var Utc bool
var UtcChanged bool
var HostBridge bool
var HostBridgeChanged bool
var Acpi bool
var AcpiChanged bool
var Hlt bool
var HltChanged bool
var Eop bool
var EopChanged bool
var Dpo bool
var DpoChanged bool
var Ium bool
var IumChanged bool

var Com1 bool
var Com1Changed bool
var Com1Log bool
var Com1LogChanged bool
var Com1Dev = "AUTO"
var Com1DevChanged bool
var Com1Speed uint32 = 115200
var Com1SpeedChanged bool

var Com2 bool
var Com2Changed bool
var Com2Log bool
var Com2LogChanged bool
var Com2Dev = "AUTO"
var Com2DevChanged bool
var Com2Speed uint32 = 115200
var Com2SpeedChanged bool

var Com3 bool
var Com3Changed bool
var Com3Log bool
var Com3LogChanged bool
var Com3Dev = "AUTO"
var Com3DevChanged bool
var Com3Speed uint32 = 115200
var Com3SpeedChanged bool

var Com4 bool
var Com4Changed bool
var Com4Log bool
var Com4LogChanged bool
var Com4Dev = "AUTO"
var Com4DevChanged bool
var Com4Speed uint32 = 115200
var Com4SpeedChanged bool

var VmCreateCmd = &cobra.Command{
	Use:          "create",
	Short:        "Create a VM",
	SilenceUsage: true,
	Args: func(cmd *cobra.Command, args []string) error {
		VmDescriptionChanged = cmd.Flags().Changed("description")
		CpusChanged = cmd.Flags().Changed("cpus")
		MemChanged = cmd.Flags().Changed("mem")

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if VmName == "" {
			return errors.New("empty VM name")
		}

		var lDesc *string
		var lCpus *uint32
		var lMem *uint32

		if !VmDescriptionChanged {
			lDesc = nil
		} else {
			lDesc = &VmDescription
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
		_, err := rpc.AddVM(VmName, lDesc, lCpus, lMem)
		if err != nil {
			return err
		}
		fmt.Print("VM Created\n")

		return nil
	},
}

var VmListCmd = &cobra.Command{
	Use:          "list",
	Short:        "List VMs",
	Long:         `List all VMs on specified server and their state`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ids, err := rpc.GetVmIds()
		if err != nil {
			return err
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
		for _, id := range ids {
			vmConfig, err := rpc.GetVMConfig(id)
			if err != nil {
				return err
			}

			var status string
			status, _, _, err = rpc.GetVMState(id)
			if err != nil {
				return err
			}
			sstatus := "Unknown"

			cpus := strconv.FormatUint(uint64(vmConfig.Cpu), 10)
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
				id:     id,
				mem:    mems,
				cpu:    cpus,
				status: sstatus,
				descr:  vmConfig.Description,
			}
			names = append(names, vmConfig.Name)
		}

		sort.Strings(names)
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		if ShowUUID {
			t.AppendHeader(table.Row{"NAME", "UUID", "CPUS", "MEMORY", "STATE", "DESCRIPTION"})
			t.SetColumnConfigs([]table.ColumnConfig{
				{Number: 3, Align: text.AlignRight, AlignHeader: text.AlignRight},
				{Number: 4, Align: text.AlignRight, AlignHeader: text.AlignRight},
			})
		} else {
			t.AppendHeader(table.Row{"NAME", "CPUS", "MEMORY", "STATE", "DESCRIPTION"})
			t.SetColumnConfigs([]table.ColumnConfig{
				{Number: 2, Align: text.AlignRight, AlignHeader: text.AlignRight},
				{Number: 3, Align: text.AlignRight, AlignHeader: text.AlignRight},
			})
		}
		t.SetStyle(myTableStyle)
		for _, name := range names {
			if ShowUUID {
				t.AppendRow(table.Row{
					name,
					vmInfos[name].id,
					vmInfos[name].cpu,
					vmInfos[name].mem,
					vmInfos[name].status,
					vmInfos[name].descr,
				})
			} else {
				t.AppendRow(table.Row{
					name,
					vmInfos[name].cpu,
					vmInfos[name].mem,
					vmInfos[name].status,
					vmInfos[name].descr,
				})
			}
		}
		t.Render()

		return nil
	},
}

var VmDestroyCmd = &cobra.Command{
	Use:          "destroy",
	Short:        "Remove a VM",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName)
			if err != nil {
				return err
			}
			if VmId == "" {
				return errors.New("VM not found")
			}
		}

		var stopped bool
		stopped, err = rpc.VmStopped(VmId)
		if err != nil {
			return err
		}
		if !stopped {
			return errors.New("VM must be stopped in order to be destroyed")
		}

		// FIXME check request ID completion and status
		_, err = rpc.DeleteVM(VmId)
		if err != nil {
			return err
		}
		fmt.Printf("VM Removed\n")

		return nil
	},
}

var VmStopCmd = &cobra.Command{
	Use:          "stop",
	Short:        "Stop a VM",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName)
			if err != nil {
				return err
			}
			if VmId == "" {
				return errors.New("VM not found")
			}
		}
		var running bool
		running, err = rpc.VmRunning(VmId)
		if err != nil {
			return err
		}
		if !running {
			return errors.New("VM not running")
		}

		var vmConfig rpc.VmConfig
		vmConfig, err = rpc.GetVMConfig(VmId)
		if err != nil {
			return err
		}

		// max wait + 10 seconds just in case
		timeout := time.Now().Add((time.Duration(int64(vmConfig.MaxWait)) * time.Second) + (time.Second * 10))

		var reqId string
		var reqStat rpc.ReqStatus
		reqId, err = rpc.StopVM(VmId)
		if err != nil {
			return err
		}

		if !CheckReqStat {
			fmt.Printf("VM stopped\n")

			return nil
		}

		fmt.Printf("VM Stopping (timeout: %ds): ", vmConfig.MaxWait)
		for time.Now().Before(timeout) {
			reqStat, err = rpc.ReqStat(reqId)
			if err != nil {
				return err
			}
			if reqStat.Success {
				fmt.Printf(" done")
			}
			if reqStat.Complete {
				break
			}
			fmt.Printf(".")
			time.Sleep(time.Second)
		}
		fmt.Printf("\n")

		return nil
	},
}

var VmStartCmd = &cobra.Command{
	Use:          "start",
	Short:        "Start a VM",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName)
			if err != nil {
				return err
			}
			if VmId == "" {
				return errors.New("VM not found")
			}
		}

		var stopped bool
		stopped, err = rpc.VmStopped(VmId)
		if err != nil {
			return err
		}
		if !stopped {
			return errors.New("VM must be stopped in order to be started")
		}

		// borrow the max stop time as a timeout for waiting on startup
		var vmConfig rpc.VmConfig
		vmConfig, err = rpc.GetVMConfig(VmId)
		if err != nil {
			return err
		}

		// max wait + 10 seconds just in case
		timeout := time.Now().Add((time.Duration(int64(vmConfig.MaxWait)) * time.Second) + (time.Second * 10))

		var reqId string
		var reqStat rpc.ReqStatus

		reqId, err = rpc.StartVM(VmId)
		if err != nil {
			return err
		}

		if !CheckReqStat {
			fmt.Print("VM started\n")

			return nil
		}

		fmt.Printf("VM Starting (timeout: %ds): ", vmConfig.MaxWait)
		for time.Now().Before(timeout) {
			reqStat, err = rpc.ReqStat(reqId)
			if err != nil {
				return err
			}
			if reqStat.Success {
				fmt.Printf(" done")
			}
			if reqStat.Complete {
				break
			}
			fmt.Printf(".")
			time.Sleep(time.Second)
		}
		fmt.Printf("\n")

		return nil
	},
}

var VmConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Reconfigure a VM",
	Args: func(cmd *cobra.Command, args []string) error {
		VmDescriptionChanged = cmd.Flags().Changed("description")
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
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName)
			if err != nil {
				return err
			}
			if VmId == "" {
				return errors.New("VM not found")
			}
		}

		var newConfig cirrina.VMConfig
		newConfig.Id = VmId

		if VmDescriptionChanged {
			newConfig.Description = &VmDescription
		}

		if CpusChanged {
			newCpu := uint32(Cpus)
			newConfig.Cpu = &newCpu
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
			return err
		}
		fmt.Printf("VM updated\n")

		return nil
	},
}

var VmGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get info on a VM",
	Args: func(cmd *cobra.Command, args []string) error {
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
			return errors.New("unknown output format")
		}

		return nil
	},
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName)
			if err != nil {
				return err
			}
			if VmId == "" {
				return errors.New("VM not found")
			}
		}
		var vmConfig rpc.VmConfig
		vmConfig, err = rpc.GetVMConfig(VmId)
		if err != nil {
			return err
		}

		var vmState string
		var vncPort string
		var debugPort string
		vmState, vncPort, debugPort, err = rpc.GetVMState(VmId)
		if err != nil {
			return err
		}

		type vmOutStat struct {
			Status    string `json:"Status"    yaml:"Status"`
			Vncport   string `json:"Vncport"   yaml:"Vncport"`
			Debugport string `json:"Debugport" yaml:"Debugport"`
		}
		type vmOutThing struct {
			Config rpc.VmConfig `json:"Config" yaml:"Config"`
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
			fmt.Printf("id: %v\n", VmId)
			fmt.Printf("name: %v\n", vmConfig.Name)
			fmt.Printf("desc: %v\n", vmConfig.Description)
			fmt.Printf("cpus: %v\n", vmConfig.Cpu)
			fmt.Printf("mem: %v\n", vmConfig.Mem)
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
				return err
			}
			fmt.Printf("%s\n", string(bar))
		case YAML:
			bar, err := yaml.Marshal(vmOutStr)
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", string(bar))
		default:
			fmt.Printf("unknown output format\n")
		}

		return nil
	},
}

var VmClearUefiVarsCmd = &cobra.Command{
	Use:          "clearuefivars",
	Short:        "Clear UEFI variable state",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName)
			if err != nil {
				return err
			}
			if VmId == "" {
				return errors.New("VM not found")
			}
		}
		var res bool
		res, err = rpc.VmClearUefiVars(VmId)
		if err != nil {
			return err
		}
		if !res {
			return errors.New("failed")
		}
		fmt.Printf("UEFI Vars cleared\n")

		return nil
	},
}

var VmCmd = &cobra.Command{
	Use:   "vm",
	Short: "Create, list, modify, destroy VMs",
}

func init() {
	disableFlagSorting(VmCmd)

	disableFlagSorting(VmListCmd)
	VmListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	VmListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)

	disableFlagSorting(VmCreateCmd)
	VmCreateCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	err := VmCreateCmd.MarkFlagRequired("name")
	if err != nil {
		panic(err)
	}
	VmCreateCmd.Flags().StringVarP(&VmDescription,
		"description", "d", SwitchDescription, "SwitchDescription of VM",
	)
	VmCreateCmd.Flags().Uint16VarP(&Cpus, "cpus", "c", Cpus, "Number of VM virtual CPUs")
	VmCreateCmd.Flags().Uint32VarP(&Mem,
		"mem", "m", Mem, "Amount of virtual memory in megabytes",
	)

	disableFlagSorting(VmDestroyCmd)
	addNameOrIdArgs(VmDestroyCmd, &VmName, &VmId, "VM")

	disableFlagSorting(VmStartCmd)
	addNameOrIdArgs(VmStartCmd, &VmName, &VmId, "VM")
	VmStartCmd.Flags().BoolVarP(&CheckReqStat, "status", "s", CheckReqStat, "Check status")

	addNameOrIdArgs(VmStopCmd, &VmName, &VmId, "VM")
	VmStopCmd.Flags().BoolVarP(&CheckReqStat, "status", "s", CheckReqStat, "Check status")
	disableFlagSorting(VmStopCmd)

	addNameOrIdArgs(VmConfigCmd, &VmName, &VmId, "VM")
	disableFlagSorting(VmConfigCmd)
	VmConfigCmd.Flags().StringVarP(&VmDescription,
		"description", "d", VmDescription, "SwitchDescription of VM",
	)
	VmConfigCmd.Flags().Uint16VarP(&Cpus, "cpus", "c", Cpus, "Number of VM virtual CPUs")
	VmConfigCmd.Flags().Uint32VarP(&Mem,
		"mem", "m", Mem, "Amount of virtual memory in megabytes",
	)
	VmConfigCmd.Flags().Int32Var(&Priority, "priority", Priority, "Priority of VM (nice)")
	VmConfigCmd.Flags().BoolVar(&Protect,
		"protect", Protect, "Protect VM from being killed when swap space is exhausted",
	)
	VmConfigCmd.Flags().Uint32Var(&Pcpu, "pcpu", Pcpu, "Max CPU usage in percent of a single CPU core")
	VmConfigCmd.Flags().Uint32Var(&Rbps, "rbps", Rbps, "Limit VM filesystem reads, in bytes per second")
	VmConfigCmd.Flags().Uint32Var(&Wbps, "wbps", Wbps, "Limit VM filesystem writes, in bytes per second")
	VmConfigCmd.Flags().Uint32Var(&Riops,
		"riops", Riops, "Limit VM filesystem reads, in operations per second",
	)
	VmConfigCmd.Flags().Uint32Var(&Wiops,
		"wiops", Wiops, "Limit VM filesystem writes, in operations per second",
	)
	VmConfigCmd.Flags().BoolVar(&Com1, "com1", Com1, "Enable COM1")
	VmConfigCmd.Flags().BoolVar(&Com1Log, "com1-log", Com1Log, "Log input and output of COM1")
	VmConfigCmd.Flags().StringVar(&Com1Dev, "com1-dev", Com1Dev, "Device to use for COM1")
	VmConfigCmd.Flags().Uint32Var(&Com1Speed, "com1-speed", Com1Speed, "Speed of COM1")
	VmConfigCmd.Flags().BoolVar(&Com2, "com2", Com2, "Enable COM2")
	VmConfigCmd.Flags().BoolVar(&Com2Log, "com2-log", Com2Log, "Log input and output of COM2")
	VmConfigCmd.Flags().StringVar(&Com2Dev, "com2-dev", Com2Dev, "Device to use for COM2")
	VmConfigCmd.Flags().Uint32Var(&Com2Speed, "com2-speed", Com2Speed, "Speed of COM2")
	VmConfigCmd.Flags().BoolVar(&Com3, "com3", Com3, "Enable COM3")
	VmConfigCmd.Flags().BoolVar(&Com3Log, "com3-log", Com3Log, "Log input and output of COM3")
	VmConfigCmd.Flags().StringVar(&Com3Dev, "com3-dev", Com3Dev, "Device to use for COM3")
	VmConfigCmd.Flags().Uint32Var(&Com3Speed, "com3-speed", Com3Speed, "Speed of COM3")
	VmConfigCmd.Flags().BoolVar(&Com4, "com4", Com4, "Enable COM4")
	VmConfigCmd.Flags().BoolVar(&Com4Log, "com4-log", Com4Log, "Log input and output of COM4")
	VmConfigCmd.Flags().StringVar(&Com4Dev, "com4-dev", Com4Dev, "Device to use for COM4")
	VmConfigCmd.Flags().Uint32Var(&Com4Speed, "com4-speed", Com4Speed, "Speed of COM4")
	VmConfigCmd.Flags().BoolVar(&AutoStart, "autostart", AutoStart, "Autostart VM")
	VmConfigCmd.Flags().Uint32Var(&AutoStartDelay,
		"autostart-delay", AutoStartDelay, "How long to wait before starting this VM",
	)
	VmConfigCmd.Flags().BoolVar(&Restart,
		"restart", Restart, "Restart this VM if it stops, crashes, shuts down, reboots, etc.",
	)
	VmConfigCmd.Flags().Uint32Var(&RestartDelay,
		"restart-delay", RestartDelay, "How long to wait before restarting this VM",
	)
	VmConfigCmd.Flags().Uint32Var(&MaxWait,
		"max-wait", MaxWait, "How long to wait for this VM to shutdown before forcibly killing it",
	)
	VmConfigCmd.Flags().BoolVar(&Screen, "screen", Screen, "Start VNC Server for this VM")
	VmConfigCmd.Flags().Uint32Var(&ScreenWidth, "screen-width", ScreenWidth, "Width of VNC server screen")
	VmConfigCmd.Flags().Uint32Var(&ScreenHeight,
		"screen-height", ScreenHeight, "Height of VNC server screen",
	)
	VmConfigCmd.Flags().StringVar(&VncPort,
		"vnc-port", VncPort, "Port to run VNC server on, AUTO for automatic, or TCP port number",
	)
	VmConfigCmd.Flags().BoolVar(&VncWait,
		"vnc-wait", VncWait, "Wait for VNC connection before starting VM",
	)
	VmConfigCmd.Flags().BoolVar(&VncTablet, "vnc-tablet", VncTablet, "VNC server in tablet mode")
	VmConfigCmd.Flags().StringVar(&VncKeyboard,
		"vnc-keyboard", VncKeyboard, "Keyboard layout used by VNC server",
	)
	VmConfigCmd.Flags().BoolVar(&Sound, "sound", Sound, "Enabled Sound output on this VM")
	VmConfigCmd.Flags().StringVar(&SoundIn, "sound-in", SoundIn, "Device to use for sound input")
	VmConfigCmd.Flags().StringVar(&SoundOut, "sound-out", SoundOut, "Device to use for sound output")
	VmConfigCmd.Flags().BoolVar(&Wire, "wire", Wire, "Wire guest memory")
	VmConfigCmd.Flags().BoolVar(&Uefi, "uefi", Uefi, "Store UEFI variables")
	VmConfigCmd.Flags().BoolVar(&Utc, "utc", Utc, "Store VM time in UTC")
	VmConfigCmd.Flags().BoolVar(&HostBridge, "host-bridge", HostBridge, "Enable host bridge")
	VmConfigCmd.Flags().BoolVar(&Acpi, "acpi", Acpi, "Enable ACPI tables")
	VmConfigCmd.Flags().BoolVar(&Hlt,
		"hlt", Hlt, "Yield the virtual CPU(s), when a HTL instruction is detected",
	)
	VmConfigCmd.Flags().BoolVar(&Eop,
		"eop", Eop, "Force the virtual CPU(s) to exit when a PAUSE instruction is detected",
	)
	VmConfigCmd.Flags().BoolVar(&Dpo, "dpo", Dpo, "Destroy the VM on guest initiated power off")
	VmConfigCmd.Flags().BoolVar(&Ium, "ium", Ium, "Ignore unimplemented model specific register access")
	VmConfigCmd.Flags().BoolVar(&Debug, "debug", Debug, "Enable Debug server")
	VmConfigCmd.Flags().BoolVar(&DebugWait,
		"debug-wait", DebugWait, "Wait for connection to debug server before starting VM",
	)
	VmConfigCmd.Flags().Uint32Var(&DebugPort, "debug-port", DebugPort, "TCP port to use for debug server")
	VmConfigCmd.Flags().StringVar(&ExtraArgs, "extra-args", ExtraArgs, "Extra args to pass to bhyve")

	disableFlagSorting(VmGetCmd)
	addNameOrIdArgs(VmGetCmd, &VmName, &VmId, "VM")
	VmGetCmd.Flags().StringVarP(&outputFormatString, "format", "f", outputFormatString,
		"Output format (txt, json, yaml",
	)

	disableFlagSorting(VmClearUefiVarsCmd)
	addNameOrIdArgs(VmClearUefiVarsCmd, &VmName, &VmId, "VM")

	VmCmd.AddCommand(VmListCmd)
	VmCmd.AddCommand(VmCreateCmd)
	VmCmd.AddCommand(VmDestroyCmd)
	VmCmd.AddCommand(VmConfigCmd)
	VmCmd.AddCommand(VmGetCmd)
	VmCmd.AddCommand(VmStartCmd)
	VmCmd.AddCommand(VmStopCmd)
	VmCmd.AddCommand(VmCom1Cmd)
	VmCmd.AddCommand(VmCom2Cmd)
	VmCmd.AddCommand(VmCom3Cmd)
	VmCmd.AddCommand(VmCom4Cmd)
	VmCmd.AddCommand(VmClearUefiVarsCmd)
}
