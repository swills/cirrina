//go:build !test

package cmd

func init() {
	disableFlagSorting(SwitchCmd)

	disableFlagSorting(SwitchListCmd)
	SwitchListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)

	disableFlagSorting(SwitchCreateCmd)
	SwitchCreateCmd.Flags().StringVarP(&SwitchName, "name", "n", SwitchName, "name of switch")
	err := SwitchCreateCmd.MarkFlagRequired("name")
	if err != nil {
		panic(err)
	}
	SwitchCreateCmd.Flags().StringVarP(&SwitchDescription,
		"description", "d", SwitchDescription, "description of switch",
	)
	SwitchCreateCmd.Flags().StringVarP(&SwitchType, "type", "t", SwitchType, "type of switch")
	SwitchCreateCmd.Flags().StringVarP(&SwitchUplinkName,
		"uplink", "u", SwitchName, "uplink name",
	)

	disableFlagSorting(SwitchDestroyCmd)
	addNameOrIDArgs(SwitchDestroyCmd, &SwitchName, &SwitchID, "switch")

	disableFlagSorting(SwitchUplinkCmd)
	addNameOrIDArgs(SwitchUplinkCmd, &SwitchName, &SwitchID, "switch")
	SwitchUplinkCmd.Flags().StringVarP(&SwitchUplinkName,
		"uplink", "u", SwitchName, "uplink name",
	)
	err = SwitchUplinkCmd.MarkFlagRequired("uplink")
	if err != nil {
		panic(err)
	}

	disableFlagSorting(SwitchUpdateCmd)
	addNameOrIDArgs(SwitchUpdateCmd, &SwitchName, &SwitchID, "switch")
	SwitchUpdateCmd.Flags().StringVarP(&SwitchDescription,
		"description", "d", SwitchDescription, "description of switch",
	)

	SwitchCmd.AddCommand(SwitchListCmd)
	SwitchCmd.AddCommand(SwitchCreateCmd)
	SwitchCmd.AddCommand(SwitchDestroyCmd)
	SwitchCmd.AddCommand(SwitchUpdateCmd)
	SwitchCmd.AddCommand(SwitchUplinkCmd)
}
