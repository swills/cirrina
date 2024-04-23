//go:build !test

package cmd

import "fmt"

func init() {
	disableFlagSorting(DiskCmd)

	setupDiskListCmd()
	err := setupDiskCreateCmd()
	if err != nil {
		panic(err)
	}
	setupDiskRemoveCmd()
	setupDiskUpdateCmd()
	err = setupDiskUploadCmd()
	if err != nil {
		panic(err)
	}

	DiskCmd.AddCommand(DiskListCmd)
	DiskCmd.AddCommand(DiskCreateCmd)
	DiskCmd.AddCommand(DiskRemoveCmd)
	DiskCmd.AddCommand(DiskUpdateCmd)
	DiskCmd.AddCommand(DiskUploadCmd)
}

func setupDiskUploadCmd() error {
	disableFlagSorting(DiskUploadCmd)
	addNameOrIDArgs(DiskUploadCmd, &DiskName, &DiskID, "disk")
	DiskUploadCmd.Flags().StringVarP(&DiskFilePath,
		"path", "p", DiskFilePath, "Path to Disk File to upload",
	)
	err := DiskUploadCmd.MarkFlagRequired("path")
	if err != nil {
		return fmt.Errorf("error marking flag required: %w", err)
	}
	DiskUploadCmd.Flags().BoolVarP(&CheckReqStat, "status", "s", CheckReqStat, "Check status")

	return nil
}

func setupDiskUpdateCmd() {
	disableFlagSorting(DiskUpdateCmd)
	addNameOrIDArgs(DiskUpdateCmd, &DiskName, &DiskID, "disk")
	DiskUpdateCmd.Flags().StringVarP(&DiskDescription,
		"description", "d", DiskDescription, "description of disk",
	)
	DiskUpdateCmd.Flags().StringVarP(&DiskType, "type", "t", DiskType, "type of disk - nvme, ahci, or virtioblk")
	DiskUpdateCmd.Flags().BoolVar(&DiskCache,
		"cache", DiskCache, "Enable or disable OS caching for this disk",
	)
	DiskUpdateCmd.Flags().BoolVar(&DiskDirect,
		"direct", DiskDirect, "Enable or disable synchronous writes for this disk",
	)
}

func setupDiskRemoveCmd() {
	disableFlagSorting(DiskRemoveCmd)
	addNameOrIDArgs(DiskRemoveCmd, &DiskName, &DiskID, "disk")
}

func setupDiskCreateCmd() error {
	var err error

	disableFlagSorting(DiskCreateCmd)
	DiskCreateCmd.Flags().StringVarP(&DiskName, "name", "n", DiskName, "name of disk")
	err = DiskCreateCmd.MarkFlagRequired("name")
	if err != nil {
		return fmt.Errorf("error marking flag required: %w", err)
	}
	DiskCreateCmd.Flags().StringVarP(&DiskSize, "size", "s", DiskName, "size of disk")
	err = DiskCreateCmd.MarkFlagRequired("size")
	if err != nil {
		return fmt.Errorf("error marking flag required: %w", err)
	}
	DiskCreateCmd.Flags().StringVarP(&DiskDescription,
		"description", "d", DiskDescription, "description of disk",
	)
	DiskCreateCmd.Flags().StringVarP(&DiskType, "type", "t", DiskType, "type of disk - nvme, ahci, or virtioblk")
	DiskCreateCmd.Flags().StringVar(&DiskDevType,
		"dev-type", DiskDevType, "Dev type of disk - file or zvol",
	)
	DiskCreateCmd.Flags().BoolVar(&DiskCache,
		"cache", DiskCache, "Enable or disable OS caching for this disk",
	)
	DiskCreateCmd.Flags().BoolVar(&DiskDirect,
		"direct", DiskDirect, "Enable or disable synchronous writes for this disk",
	)

	return nil
}

func setupDiskListCmd() {
	disableFlagSorting(DiskListCmd)
	DiskListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	DiskListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)
}
