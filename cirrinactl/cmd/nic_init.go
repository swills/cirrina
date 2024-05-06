//go:build !test

package cmd

import "fmt"

func init() {
	disableFlagSorting(NicCmd)

	setupNicListCmd()

	err := setupNicCreateCmd()
	if err != nil {
		panic(err)
	}

	setupNicRemoveCmd()
	setupNicSetSwitchCmd()
	setupNicCloneCmd()
	setupNicUpdateCmd()

	NicCmd.AddCommand(NicListCmd)
	NicCmd.AddCommand(NicCreateCmd)
	NicCmd.AddCommand(NicRemoveCmd)
	NicCmd.AddCommand(NicSetSwitchCmd)
	NicCmd.AddCommand(NicCloneCmd)
	NicCmd.AddCommand(NicUpdateCmd)
}

func setupNicUpdateCmd() {
	disableFlagSorting(NicUpdateCmd)
	addNameOrIDArgs(NicUpdateCmd, &NicName, &NicID, "NIC")
	NicUpdateCmd.Flags().StringVarP(&NicDescription,
		"description", "d", NicDescription, "description of NIC",
	)
	NicUpdateCmd.Flags().StringVarP(&NicType, "type", "t", NicType, "type of NIC")
	NicUpdateCmd.Flags().StringVarP(&NicDevType, "devtype", "v", NicDevType, "NIC dev type")
	NicUpdateCmd.Flags().StringVarP(&NicMac, "mac", "m", NicMac, "MAC address of NIC")
	NicUpdateCmd.Flags().StringVarP(&NicSwitchID,
		"switch-id", "I", NicSwitchID, "NIC uplink switch ID",
	)
	NicUpdateCmd.Flags().StringVarP(&NicSwitchName,
		"switch-name", "N", NicSwitchName, "NIC uplink switch name",
	)
	NicUpdateCmd.Flags().BoolVar(&NicRateLimited, "rate-limit", NicRateLimited, "Rate limit the NIC")
	NicUpdateCmd.Flags().Uint64Var(&NicRateIn, "rate-in", NicRateIn, "Inbound rate limit of NIC")
	NicUpdateCmd.Flags().Uint64Var(&NicRateOut, "rate-out", NicRateOut, "Outbound rate limit of NIC")
}

func setupNicCloneCmd() {
	disableFlagSorting(NicCloneCmd)
	addNameOrIDArgs(NicCloneCmd, &NicName, &NicID, "NIC")

	NicCloneCmd.Flags().StringVar(&NicCloneName,
		"new-name", NicCloneName, "Name of Cloned NIC",
	)
	NicCloneCmd.Flags().BoolVarP(&CheckReqStat, "status", "s", CheckReqStat, "Check status")
}

func setupNicSetSwitchCmd() {
	disableFlagSorting(NicSetSwitchCmd)
	addNameOrIDArgs(NicSetSwitchCmd, &NicName, &NicID, "NIC")
	NicSetSwitchCmd.Flags().StringVarP(&NicSwitchName,
		"switch-name", "N", SwitchName, "Switch Name",
	)
	NicSetSwitchCmd.Flags().StringVarP(&NicSwitchID, "switch-id", "I", SwitchID, "ID of Switch")
	NicSetSwitchCmd.MarkFlagsOneRequired("switch-name", "switch-id")
	NicSetSwitchCmd.MarkFlagsMutuallyExclusive("switch-name", "switch-id")
}

func setupNicRemoveCmd() {
	disableFlagSorting(NicRemoveCmd)
	addNameOrIDArgs(NicRemoveCmd, &NicName, &NicID, "NIC")
}

func setupNicCreateCmd() error {
	disableFlagSorting(NicCreateCmd)
	NicCreateCmd.Flags().StringVarP(&NicName, "name", "n", NicName, "name of NIC")

	err := NicCreateCmd.MarkFlagRequired("name")
	if err != nil {
		return fmt.Errorf("error marking flag required: %w", err)
	}

	NicCreateCmd.Flags().StringVarP(&NicDescription,
		"description", "d", NicDescription, "description of NIC",
	)
	NicCreateCmd.Flags().StringVarP(&NicType, "type", "t", NicType, "type of NIC")
	NicCreateCmd.Flags().StringVarP(&NicDevType, "devtype", "v", NicDevType, "NIC dev type")
	NicCreateCmd.Flags().StringVarP(&NicMac, "mac", "m", NicMac, "MAC address of NIC")
	NicCreateCmd.Flags().StringVar(&NicSwitchID,
		"switch-id", NicSwitchID, "NIC uplink switch ID",
	)
	NicCreateCmd.Flags().StringVar(&NicSwitchName,
		"switch-name", NicSwitchName, "NIC uplink switch name",
	)
	NicCreateCmd.Flags().BoolVar(&NicRateLimited, "rate-limit", NicRateLimited, "Rate limit the NIC")
	NicCreateCmd.Flags().Uint64Var(&NicRateIn, "rate-in", NicRateIn, "Inbound rate limit of NIC")
	NicCreateCmd.Flags().Uint64Var(&NicRateOut, "rate-out", NicRateOut, "Outbound rate limit of NIC")

	return nil
}

func setupNicListCmd() {
	disableFlagSorting(NicListCmd)
	NicListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print speeds in human readable form",
	)
	NicListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)
}
