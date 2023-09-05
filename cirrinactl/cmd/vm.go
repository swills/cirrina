package cmd

import (
	"cirrina/cirrina"
	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinactl/util"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"log"
	"strconv"
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
var Cpus uint8
var CpusChanged bool
var DescriptionChanged bool
var Mem uint32
var MemChanged bool
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
	Use:   "create",
	Short: "Create a VM",
	Args: func(cmd *cobra.Command, args []string) error {
		DescriptionChanged = cmd.Flags().Changed("description")
		CpusChanged = cmd.Flags().Changed("cpus")
		MemChanged = cmd.Flags().Changed("mem")
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()

		var lDesc *string
		var lCpus *uint32
		var lMem *uint32

		if !DescriptionChanged {
			lDesc = nil
		} else {
			lDesc = &Description
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

		util.AddVM(&VmName, c, ctx, lDesc, lCpus, lMem)
	},
}

var VmListCmd = &cobra.Command{
	Use:   "list",
	Short: "List VMs",
	Long:  `List all VMs on specified server and their state`,
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		util.GetVMs(c, ctx)
	},
}

var VmDestroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Remove a VM",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if VmId == "" {
				log.Fatalf("VM not found")
			}
		}
		if VmName == "" {
			VmName, err = rpc.VmIdToName(&VmId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}

		util.DeleteVM(VmName, c, ctx)
	},
}

var VmStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a VM",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if VmId == "" {
				log.Fatalf("VM not found")
			}
		}
		if VmName == "" {
			VmName, err = rpc.VmIdToName(&VmId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		util.StopVM(VmName, c, ctx)
	},
}

var VmStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a VM",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()

		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if VmId == "" {
				log.Fatalf("VM not found")
			}
		}
		if VmName == "" {
			VmName, err = rpc.VmIdToName(&VmId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}

		util.StartVM(VmName, c, ctx)
	},
}

var VmConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Reconfigure a VM",
	Args: func(cmd *cobra.Command, args []string) error {
		DescriptionChanged = cmd.Flags().Changed("description")
		CpusChanged = cmd.Flags().Changed("cpus")
		MemChanged = cmd.Flags().Changed("mem")
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
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()

		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if VmId == "" {
				log.Fatalf("VM not found")
			}
		}
		if VmName == "" {
			VmName, err = rpc.VmIdToName(&VmId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}

		newConfig := cirrina.VMConfig{
			Id: VmId,
		}

		if DescriptionChanged {
			newConfig.Description = &Description
		}

		if CpusChanged {
			newCpu := uint32(Cpus)

			if newCpu < 1 {
				newCpu = 1
			}
			if newCpu > 16 {
				newCpu = 16
			}
			newConfig.Cpu = &newCpu
		}

		if MemChanged {
			newMem := Mem
			if newMem < 128 {
				newMem = 128
			}
			newConfig.Mem = &newMem
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

		err = rpc.UpdateVMConfig(&newConfig, c, ctx)
		if err != nil {
			log.Fatalf(err.Error())
		}
	},
}

var VmGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get info on a VM",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()

		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if VmId == "" {
				log.Fatalf("VM not found")
			}
		}
		if VmName == "" {
			VmName, err = rpc.VmIdToName(&VmId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		util.GetVM(&VmId, c, ctx)
	},
}

var VmDisksGetCmd = &cobra.Command{
	Use:   "list",
	Short: "Get list of disks connected to VM",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if VmId == "" {
				log.Fatalf("VM not found")
			}
		}
		if VmName == "" {
			VmName, err = rpc.VmIdToName(&VmId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		util.GetVMDisks(VmName, c, ctx)
	},
}

var VmDiskAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add disk to VM",
	Args: func(cmd *cobra.Command, args []string) error {
		DiskIdChanged = cmd.Flags().Changed("disk-id")
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()

		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if VmId == "" {
				log.Fatalf("VM not found")
			}
		}
		if VmName == "" {
			VmName, err = rpc.VmIdToName(&VmId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		if DiskId == "" && !DiskIdChanged && DiskName != "" {
			DiskId, err = rpc.DiskNameToId(&DiskName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if DiskId == "" {
				log.Fatalf("Disk not found")
			}
		}
		util.VmDiskAdd(VmName, DiskId, c, ctx)
	},
}

var VmDiskRmCmd = &cobra.Command{
	Use:   "remove",
	Short: "Un-attach a disk from a VM",
	Args: func(cmd *cobra.Command, args []string) error {
		DiskIdChanged = cmd.Flags().Changed("disk-id")
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()

		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if VmId == "" {
				log.Fatalf("VM not found")
			}
		}
		if VmName == "" {
			VmName, err = rpc.VmIdToName(&VmId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		if DiskId == "" && !DiskIdChanged && DiskName != "" {
			DiskId, err = rpc.DiskNameToId(&DiskName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if DiskId == "" {
				log.Fatalf("Disk not found")
			}
		}
		util.VmDiskRm(VmName, DiskId, c, ctx)
	},
}

var VmDisksCmd = &cobra.Command{
	Use:   "disk",
	Short: "Disk operations on VMs",
	Long:  "List disks attached to VMs, attach disks to VMs and un-attach disks from VMs",
}

var VmIsosGetCmd = &cobra.Command{
	Use:   "list",
	Short: "Get list of ISOs connected to VM",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if VmId == "" {
				log.Fatalf("VM not found")
			}
		}
		if VmName == "" {
			VmName, err = rpc.VmIdToName(&VmId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		util.GetVMIsos(VmName, c, ctx)
	},
}

var VmIsosAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add ISO to VM",
	Args: func(cmd *cobra.Command, args []string) error {
		IsoIdChanged = cmd.Flags().Changed("Iso-id")
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()

		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if VmId == "" {
				log.Fatalf("VM not found")
			}
		}
		if VmName == "" {
			VmName, err = rpc.VmIdToName(&VmId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		if IsoId == "" && !IsoIdChanged && IsoName != "" {
			IsoId, err = rpc.IsoNameToId(&IsoName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if IsoId == "" {
				log.Fatalf("Isos not found")
			}
		}
		util.VmIsoAdd(VmName, IsoId, c, ctx)
	},
}

var VmIsosRmCmd = &cobra.Command{
	Use:   "remove",
	Short: "Un-attach a ISO from a VM",
	Args: func(cmd *cobra.Command, args []string) error {
		IsoIdChanged = cmd.Flags().Changed("iso-id")
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()

		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if VmId == "" {
				log.Fatalf("VM not found")
			}
		}
		if VmName == "" {
			VmName, err = rpc.VmIdToName(&VmId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		if IsoId == "" && !IsoIdChanged && IsoName != "" {
			IsoId, err = rpc.IsoNameToId(&IsoName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if IsoId == "" {
				log.Fatalf("Isos not found")
			}
		}
		util.VmIsoRm(VmName, IsoId, c, ctx)
	},
}

var VmIsosCmd = &cobra.Command{
	Use:   "iso",
	Short: "ISO related operations on VMs",
	Long:  "List ISOs attached to VMs, attach ISOs to VMs and un-attach ISOs from VMs",
}

var VmNicsGetCmd = &cobra.Command{
	Use:   "list",
	Short: "Get list of NICs connected to VM",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if VmId == "" {
				log.Fatalf("VM not found")
			}
		}
		if VmName == "" {
			VmName, err = rpc.VmIdToName(&VmId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		util.GetVmNics(VmName, c, ctx)
	},
}

var VmNicsAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add NIC to VM",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()

		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if VmId == "" {
				log.Fatalf("VM not found")
			}
		}
		if VmName == "" {
			VmName, err = rpc.VmIdToName(&VmId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		if NicId == "" && !NicIdChanged && NicName != "" {
			NicId, err = rpc.NicNameToId(&NicName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if NicId == "" {
				log.Fatalf("NIC not found")
			}
		}
		util.VmNicAdd(VmName, NicId, c, ctx)
	},
}

var VmNicsRmCmd = &cobra.Command{
	Use:   "remove",
	Short: "Un-attach a NIC from a VM",
	Args: func(cmd *cobra.Command, args []string) error {
		NicIdChanged = cmd.Flags().Changed("nic-id")
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()

		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if VmId == "" {
				log.Fatalf("VM not found")
			}
		}
		if VmName == "" {
			VmName, err = rpc.VmIdToName(&VmId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		if NicId == "" && !NicIdChanged && NicName != "" {
			NicId, err = rpc.NicNameToId(&NicName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if NicId == "" {
				log.Fatalf("Nic not found")
			}
		}
		util.VmNicRm(VmName, NicId, c, ctx)
	},
}

var VmNicsCmd = &cobra.Command{
	Use:   "nic",
	Short: "NIC related operations on VMs",
	Long:  "List NICs attached to VMs, attach NICs to VMs and un-attach NICs from VMs",
}

var VmClearUefiVarsCmd = &cobra.Command{
	Use:   "clearuefivars",
	Short: "Clear UEFI variable state",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()

		if VmId == "" {
			VmId, err = rpc.VmNameToId(VmName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if VmId == "" {
				log.Fatalf("VM not found")
			}
		}
		if VmName == "" {
			VmName, err = rpc.VmIdToName(&VmId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		util.ClearUefiVars(VmName, c, ctx)
	},
}

var VmCmd = &cobra.Command{
	Use:   "vm",
	Short: "Create, list, modify, destroy VMs",
}

func init() {
	VmCreateCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	err := VmCreateCmd.MarkFlagRequired("name")
	if err != nil {
		panic(err)
	}
	VmCreateCmd.Flags().StringVarP(&Description, "description", "d", Description, "Description of VM")
	VmCreateCmd.Flags().Uint8VarP(&Cpus, "cpus", "c", Cpus, "Number of VM virtual CPUs")
	VmCreateCmd.Flags().Uint32VarP(&Mem, "mem", "m", Mem, "Amount of virtual memory in megabytes")

	VmStartCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmStartCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmStartCmd.MarkFlagsOneRequired("name", "id")

	VmStopCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmStopCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmStopCmd.MarkFlagsOneRequired("name", "id")

	VmDestroyCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmDestroyCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmDestroyCmd.MarkFlagsOneRequired("name", "id")

	VmConfigCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmConfigCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmConfigCmd.MarkFlagsOneRequired("name", "id")
	VmConfigCmd.Flags().StringVarP(&Description, "description", "d", Description, "Description of VM")
	VmConfigCmd.Flags().Uint8VarP(&Cpus, "cpus", "c", Cpus, "Number of VM virtual CPUs")
	VmConfigCmd.Flags().Uint32VarP(&Mem, "mem", "m", Mem, "Amount of virtual memory in megabytes")
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
	VmConfigCmd.Flags().Uint32Var(&AutoStartDelay, "autostart-delay", AutoStartDelay, "How long to wait before starting this VM")
	VmConfigCmd.Flags().BoolVar(&Restart, "restart", Restart, "Restart this VM if it stops, crashes, shuts down, reboots, etc.")
	VmConfigCmd.Flags().Uint32Var(&RestartDelay, "restart-delay", RestartDelay, "How long to wait before restarting this VM")
	VmConfigCmd.Flags().Uint32Var(&MaxWait, "max-wait", MaxWait, "How long to wait for this VM to shutdown before forcibly killing it")
	VmConfigCmd.Flags().BoolVar(&Screen, "screen", Screen, "Start VNC Server for this VM")
	VmConfigCmd.Flags().Uint32Var(&ScreenWidth, "screen-width", ScreenWidth, "Width of VNC server screen")
	VmConfigCmd.Flags().Uint32Var(&ScreenHeight, "screen-height", ScreenHeight, "Height of VNC server screen")
	VmConfigCmd.Flags().StringVar(&VncPort, "vnc-port", VncPort, "Port to run VNC server on, AUTO for automatic, or TCP port number")
	VmConfigCmd.Flags().BoolVar(&VncWait, "vnc-wait", VncWait, "Wait for VNC connection before starting VM")
	VmConfigCmd.Flags().BoolVar(&VncTablet, "vnc-tablet", VncTablet, "VNC server in tablet mode")
	VmConfigCmd.Flags().StringVar(&VncKeyboard, "vnc-keyboard", VncKeyboard, "Keyboard layout used by VNC server")
	VmConfigCmd.Flags().BoolVar(&Sound, "sound", Sound, "Enabled Sound output on this VM")
	VmConfigCmd.Flags().StringVar(&SoundIn, "sound-in", SoundIn, "Device to use for sound input")
	VmConfigCmd.Flags().StringVar(&SoundOut, "sound-out", SoundOut, "Device to use for sound output")
	VmConfigCmd.Flags().BoolVar(&Wire, "wire", Wire, "Wire guest memory")
	VmConfigCmd.Flags().BoolVar(&Uefi, "uefi", Uefi, "Store UEFI variables")
	VmConfigCmd.Flags().BoolVar(&Utc, "utc", Utc, "Store VM time in UTC")
	VmConfigCmd.Flags().BoolVar(&HostBridge, "host-bridge", HostBridge, "Enable host bridge")
	VmConfigCmd.Flags().BoolVar(&Acpi, "acpi", Acpi, "Enable ACPI tables")
	VmConfigCmd.Flags().BoolVar(&Hlt, "hlt", Hlt, "Yield the virtual CPU(s), when a HTL instruction is detected")
	VmConfigCmd.Flags().BoolVar(&Eop, "eop", Eop, "Force the virtual CPU(s) to exit when a PAUSE instruction is detected")
	VmConfigCmd.Flags().BoolVar(&Dpo, "dpo", Dpo, "Destroy the VM on guest initiated power off")
	VmConfigCmd.Flags().BoolVar(&Ium, "ium", Ium, "Ignore unimplemented model specific register access")
	VmConfigCmd.Flags().BoolVar(&Debug, "debug", Debug, "Enable Debug server")
	VmConfigCmd.Flags().BoolVar(&DebugWait, "debug-wait", DebugWait, "Wait for connection to debug server before starting VM")
	VmConfigCmd.Flags().Uint32Var(&DebugPort, "debug-port", DebugPort, "TCP port to use for debug server")
	VmConfigCmd.Flags().StringVar(&ExtraArgs, "extra-args", ExtraArgs, "Extra args to pass to bhyve")

	VmGetCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmGetCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmGetCmd.MarkFlagsOneRequired("name", "id")

	VmDisksGetCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmDisksGetCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmDisksGetCmd.MarkFlagsOneRequired("name", "id")

	VmDiskAddCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmDiskAddCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmDiskAddCmd.MarkFlagsOneRequired("name", "id")

	VmDiskAddCmd.Flags().StringVarP(&DiskName, "disk-name", "N", DiskName, "Name of Disk")
	VmDiskAddCmd.Flags().StringVarP(&DiskId, "disk-id", "I", DiskId, "Id of Disk")
	VmDiskAddCmd.MarkFlagsOneRequired("disk-name", "disk-id")

	VmDiskRmCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmDiskRmCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmDiskRmCmd.MarkFlagsOneRequired("name", "id")

	VmDiskRmCmd.Flags().StringVarP(&DiskName, "disk-name", "N", DiskName, "Name of Disk")
	VmDiskRmCmd.Flags().StringVarP(&DiskId, "disk-id", "I", DiskId, "Id of Disk")
	VmDiskRmCmd.MarkFlagsOneRequired("disk-name", "disk-id")

	VmIsosGetCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmIsosGetCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmIsosGetCmd.MarkFlagsOneRequired("name", "id")

	VmIsosAddCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmIsosAddCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmIsosAddCmd.MarkFlagsOneRequired("name", "id")

	VmIsosAddCmd.Flags().StringVarP(&IsoName, "iso-name", "N", IsoName, "Name of Iso")
	VmIsosAddCmd.Flags().StringVarP(&IsoId, "iso-id", "I", IsoId, "Id of Iso")
	VmIsosAddCmd.MarkFlagsOneRequired("iso-name", "iso-id")

	VmIsosRmCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmIsosRmCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmIsosRmCmd.MarkFlagsOneRequired("name", "id")

	VmIsosRmCmd.Flags().StringVarP(&IsoName, "iso-name", "N", IsoName, "Name of Iso")
	VmIsosRmCmd.Flags().StringVarP(&IsoId, "iso-id", "I", IsoId, "Id of Iso")
	VmIsosRmCmd.MarkFlagsOneRequired("iso-name", "iso-id")

	VmNicsGetCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmNicsGetCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmNicsGetCmd.MarkFlagsOneRequired("name", "id")

	VmNicsAddCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmNicsAddCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmNicsAddCmd.MarkFlagsOneRequired("name", "id")

	VmNicsAddCmd.Flags().StringVarP(&NicName, "nic-name", "N", NicName, "Name of Nic")
	VmNicsAddCmd.Flags().StringVarP(&NicId, "nic-id", "I", NicId, "Id of Nic")
	VmNicsAddCmd.MarkFlagsOneRequired("nic-name", "nic-id")

	VmNicsRmCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmNicsRmCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmNicsRmCmd.MarkFlagsOneRequired("name", "id")

	VmNicsRmCmd.Flags().StringVarP(&NicName, "nic-name", "N", NicName, "Name of Nic")
	VmNicsRmCmd.Flags().StringVarP(&NicId, "nic-id", "I", NicId, "Id of Nic")
	VmNicsRmCmd.MarkFlagsOneRequired("nic-name", "nic-id")

	VmClearUefiVarsCmd.Flags().StringVarP(&VmName, "name", "n", VmName, "Name of VM")
	VmClearUefiVarsCmd.Flags().StringVarP(&VmId, "id", "i", VmId, "Id of VM")
	VmClearUefiVarsCmd.MarkFlagsOneRequired("name", "id")

	VmDisksCmd.AddCommand(VmDisksGetCmd)
	VmDisksCmd.AddCommand(VmDiskAddCmd)
	VmDisksCmd.AddCommand(VmDiskRmCmd)

	VmIsosCmd.AddCommand(VmIsosGetCmd)
	VmIsosCmd.AddCommand(VmIsosAddCmd)
	VmIsosCmd.AddCommand(VmIsosRmCmd)

	VmNicsCmd.AddCommand(VmNicsGetCmd)
	VmNicsCmd.AddCommand(VmNicsAddCmd)
	VmNicsCmd.AddCommand(VmNicsRmCmd)

	VmCmd.AddCommand(VmCreateCmd)
	VmCmd.AddCommand(VmListCmd)
	VmCmd.AddCommand(VmStartCmd)
	VmCmd.AddCommand(VmStopCmd)
	VmCmd.AddCommand(VmDestroyCmd)
	VmCmd.AddCommand(VmConfigCmd)
	VmCmd.AddCommand(VmGetCmd)
	VmCmd.AddCommand(VmCom1Cmd)
	VmCmd.AddCommand(VmCom2Cmd)
	VmCmd.AddCommand(VmCom3Cmd)
	VmCmd.AddCommand(VmCom4Cmd)
	VmCmd.AddCommand(VmClearUefiVarsCmd)

	VmCmd.AddCommand(VmDisksCmd)
	VmCmd.AddCommand(VmIsosCmd)
	VmCmd.AddCommand(VmNicsCmd)

}
