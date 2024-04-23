//go:build !test

package main

import "github.com/spf13/cobra"

func init() {
	cobra.OnInitialize(initConfig)
	cobra.EnableCommandSorting = false
	disableFlagSorting(rootCmd)

	rootCmd.PersistentFlags().StringVarP(&cfgFile,
		"config", "C", cfgFile, "config file (default config.yml)",
	)
}
