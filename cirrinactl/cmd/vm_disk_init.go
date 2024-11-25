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

	disableFlagSorting(VMDiskDisconnectCmd)
	addNameOrIDArgs(VMDiskDisconnectCmd, &VMName, &VMID, "VM")
	VMDiskDisconnectCmd.Flags().StringVarP(&DiskName, "disk-name", "N", DiskName, "Name of Disk")
	VMDiskDisconnectCmd.Flags().StringVarP(&DiskID, "disk-id", "I", DiskID, "ID of Disk")
	VMDiskDisconnectCmd.MarkFlagsOneRequired("disk-name", "disk-id")
	VMDiskDisconnectCmd.MarkFlagsMutuallyExclusive("disk-name", "disk-id")

	VMDisksCmd.AddCommand(VMDisksListCmd)
	VMDisksCmd.AddCommand(VMDiskAddCmd)
	VMDisksCmd.AddCommand(VMDiskDisconnectCmd)

	VMCmd.AddCommand(VMDisksCmd)
}
