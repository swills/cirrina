//go:build !test

package cmd

func init() {
	disableFlagSorting(VMIsosCmd)

	disableFlagSorting(VMIsoListCmd)
	addNameOrIDArgs(VMIsoListCmd, &VMName, &VMID, "VM")
	VMIsoListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	VMIsoListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)

	disableFlagSorting(VMIsosAddCmd)
	addNameOrIDArgs(VMIsosAddCmd, &VMName, &VMID, "VM")
	VMIsosAddCmd.Flags().StringVarP(&IsoName, "iso-name", "N", IsoName, "Name of Iso")
	VMIsosAddCmd.Flags().StringVarP(&IsoID, "iso-id", "I", IsoID, "ID of Iso")
	VMIsosAddCmd.MarkFlagsOneRequired("iso-name", "iso-id")
	VMIsosAddCmd.MarkFlagsMutuallyExclusive("iso-name", "iso-id")

	disableFlagSorting(VMIsosRmCmd)
	addNameOrIDArgs(VMIsosRmCmd, &VMName, &VMID, "VM")
	VMIsosRmCmd.Flags().StringVarP(&IsoName, "iso-name", "N", IsoName, "Name of Iso")
	VMIsosRmCmd.Flags().StringVarP(&IsoID, "iso-id", "I", IsoID, "ID of Iso")
	VMIsosRmCmd.MarkFlagsOneRequired("iso-name", "iso-id")
	VMIsosRmCmd.MarkFlagsMutuallyExclusive("iso-name", "iso-id")

	VMIsosCmd.AddCommand(VMIsoListCmd)
	VMIsosCmd.AddCommand(VMIsosAddCmd)
	VMIsosCmd.AddCommand(VMIsosRmCmd)

	VMCmd.AddCommand(VMIsosCmd)
}
