//go:build !test

package cmd

func init() {
	disableFlagSorting(VMNicsCmd)

	disableFlagSorting(VMNicsListCmd)
	addNameOrIDArgs(VMNicsListCmd, &VMName, &VMID, "VM")
	VMNicsListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	VMNicsListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)

	disableFlagSorting(VMNicsAddCmd)
	addNameOrIDArgs(VMNicsAddCmd, &VMName, &VMID, "VM")
	VMNicsAddCmd.Flags().StringVarP(&NicName, "nic-name", "N", NicName, "Name of Nic")
	VMNicsAddCmd.Flags().StringVarP(&NicID, "nic-id", "I", NicID, "ID of Nic")
	VMNicsAddCmd.MarkFlagsOneRequired("nic-name", "nic-id")
	VMNicsAddCmd.MarkFlagsMutuallyExclusive("nic-name", "nic-id")

	disableFlagSorting(VMNicsRmCmd)
	addNameOrIDArgs(VMNicsRmCmd, &VMName, &VMID, "VM")
	VMNicsRmCmd.Flags().StringVarP(&NicName, "nic-name", "N", NicName, "Name of Nic")
	VMNicsRmCmd.Flags().StringVarP(&NicID, "nic-id", "I", NicID, "ID of Nic")
	VMNicsRmCmd.MarkFlagsOneRequired("nic-name", "nic-id")
	VMNicsRmCmd.MarkFlagsMutuallyExclusive("nic-name", "nic-id")

	VMNicsCmd.AddCommand(VMNicsListCmd)
	VMNicsCmd.AddCommand(VMNicsAddCmd)
	VMNicsCmd.AddCommand(VMNicsRmCmd)

	VMCmd.AddCommand(VMNicsCmd)
}
