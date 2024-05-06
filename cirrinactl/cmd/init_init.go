//go:build !test

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func disableFlagSorting(cmd *cobra.Command) {
	cmd.Flags().SortFlags = false
	cmd.PersistentFlags().SortFlags = false
	cmd.InheritedFlags().SortFlags = false
}

func addNameOrIDArgs(cmd *cobra.Command, nameArg *string, idArg *string, objTypeName string) {
	cmd.Flags().StringVarP(nameArg, "name", "n", *nameArg, "Name of "+objTypeName)
	cmd.Flags().StringVarP(idArg, "id", "i", *idArg, "ID of "+objTypeName)
	cmd.MarkFlagsOneRequired("name", "id")
	cmd.MarkFlagsMutuallyExclusive("name", "id")
}

func init() {
	cobra.OnInitialize(initConfig)
	cobra.EnableCommandSorting = false

	disableFlagSorting(rootCmd)

	rootCmd.PersistentFlags().StringVarP(&cfgFile,
		"config", "C", cfgFile, "config file (default $HOME/.cirrinactl.yaml)")

	rootCmd.PersistentFlags().StringP("server", "S", defaultHost, "server")

	err := viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server"))
	if err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().Uint16P("port", "P", uint16(defaultPort), "port")

	err = viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
	if err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().Uint64P("timeout", "T", uint64(defaultTimeout), "timeout in seconds")

	err = viper.BindPFlag("timeout", rootCmd.PersistentFlags().Lookup("timeout"))
	if err != nil {
		panic(err)
	}

	rootCmd.AddCommand(VMCmd)
	rootCmd.AddCommand(DiskCmd)
	rootCmd.AddCommand(IsoCmd)
	rootCmd.AddCommand(NicCmd)
	rootCmd.AddCommand(SwitchCmd)
	rootCmd.AddCommand(TuiCmd)
	rootCmd.AddCommand(HostCmd)
	rootCmd.AddCommand(ReqStatCmd)
}
