//go:build !test

package cmd

func init() {
	disableFlagSorting(IsoCmd)

	disableFlagSorting(IsoListCmd)
	IsoListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	IsoListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)

	disableFlagSorting(IsoCreateCmd)
	IsoCreateCmd.Flags().StringVarP(&IsoName,
		"name", "n", IsoName, "name of ISO",
	)

	err := IsoCreateCmd.MarkFlagRequired("name")
	if err != nil {
		panic(err)
	}

	IsoCreateCmd.Flags().StringVarP(&IsoDescription,
		"description", "d", IsoDescription, "description of ISO",
	)

	disableFlagSorting(IsoRemoveCmd)
	addNameOrIDArgs(IsoRemoveCmd, &IsoName, &IsoID, "ISO")

	disableFlagSorting(IsoUploadCmd)
	addNameOrIDArgs(IsoUploadCmd, &IsoName, &IsoID, "ISO")
	IsoUploadCmd.Flags().StringVarP(&IsoFilePath,
		"path", "p", IsoFilePath, "Path to ISO File to upload",
	)

	err = IsoUploadCmd.MarkFlagRequired("path")
	if err != nil {
		panic(err)
	}

	IsoUploadCmd.Flags().BoolVarP(&CheckReqStat, "status", "s", CheckReqStat, "Check status")

	IsoCmd.AddCommand(IsoListCmd)
	IsoCmd.AddCommand(IsoCreateCmd)
	IsoCmd.AddCommand(IsoRemoveCmd)
	IsoCmd.AddCommand(IsoUploadCmd)
}
