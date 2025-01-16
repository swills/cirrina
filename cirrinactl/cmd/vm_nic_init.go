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
	VMNicsAddCmd.Flags().StringVarP(&NicName, "nic-name", "N", NicName,
		"Name of NIC (may be specified multiple times to add multiple NICs)")
	VMNicsAddCmd.Flags().StringVarP(&NicID, "nic-id", "I", NicID,
		"ID of Nic (may be specified multiple times to add multiple NICs)")
	VMNicsAddCmd.MarkFlagsOneRequired("nic-name", "nic-id")
	VMNicsAddCmd.MarkFlagsMutuallyExclusive("nic-name", "nic-id")

	disableFlagSorting(VMNicsDisconnectCmd)
	addNameOrIDArgs(VMNicsDisconnectCmd, &VMName, &VMID, "VM")
	VMNicsDisconnectCmd.Flags().StringVarP(&NicName, "nic-name", "N", NicName, "Name of NIC")
	VMNicsDisconnectCmd.Flags().StringVarP(&NicID, "nic-id", "I", NicID, "ID of NIC")
	VMNicsDisconnectCmd.MarkFlagsOneRequired("nic-name", "nic-id")
	VMNicsDisconnectCmd.MarkFlagsMutuallyExclusive("nic-name", "nic-id")

	VMNicsCmd.AddCommand(VMNicsListCmd)
	VMNicsCmd.AddCommand(VMNicsAddCmd)
	VMNicsCmd.AddCommand(VMNicsDisconnectCmd)

	VMCmd.AddCommand(VMNicsCmd)
}
