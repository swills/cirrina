package cmd

import (
	"cirrina/cirrina"
	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinactl/util"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"log"
)

var AutoStart bool
var AutoStartChanged bool
var Cpus uint8
var CpusChanged bool
var DescriptionChanged bool
var Mem uint32
var MemChanged bool

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
	VmConfigCmd.Flags().BoolVar(&AutoStart, "autostart", AutoStart, "autostart VM")

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

	VmCmd.AddCommand(VmDisksCmd)
	VmCmd.AddCommand(VmIsosCmd)
	VmCmd.AddCommand(VmNicsCmd)

}
