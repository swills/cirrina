//go:build !test

package cmd

func init() {
	disableFlagSorting(VMDisksCmd)

	disableFlagSorting(VMDisksListCmd)
	addNameOrIDArgs(VMDisksListCmd, &VMName, &VMID, "VM")
	VMDisksListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	VMDisksListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)

	disableFlagSorting(VMDiskAddCmd)
	addNameOrIDArgs(VMDiskAddCmd, &VMName, &VMID, "VM")
	VMDiskAddCmd.Flags().StringVarP(&DiskName, "disk-name", "N", DiskName, "Name of Disk")
	VMDiskAddCmd.Flags().StringVarP(&DiskID, "disk-id", "I", DiskID, "ID of Disk")
	VMDiskAddCmd.MarkFlagsOneRequired("disk-name", "disk-id")
	VMDiskAddCmd.MarkFlagsMutuallyExclusive("disk-name", "disk-id")

	disableFlagSorting(VMDiskRmCmd)
	addNameOrIDArgs(VMDiskRmCmd, &VMName, &VMID, "VM")
	VMDiskRmCmd.Flags().StringVarP(&DiskName, "disk-name", "N", DiskName, "Name of Disk")
	VMDiskRmCmd.Flags().StringVarP(&DiskID, "disk-id", "I", DiskID, "ID of Disk")
	VMDiskRmCmd.MarkFlagsOneRequired("disk-name", "disk-id")
	VMDiskRmCmd.MarkFlagsMutuallyExclusive("disk-name", "disk-id")

	VMDisksCmd.AddCommand(VMDisksListCmd)
	VMDisksCmd.AddCommand(VMDiskAddCmd)
	VMDisksCmd.AddCommand(VMDiskRmCmd)

	VMCmd.AddCommand(VMDisksCmd)
}
