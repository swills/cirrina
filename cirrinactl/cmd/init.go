package cmd

import (
	"cirrina/cirrinactl/rpc"
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var myTableStyle = table.Style{
	Name: "myNewStyle",
	Box: table.BoxStyle{
		MiddleHorizontal: "-", // bug in go-pretty causes panic if this is empty
		PaddingRight:     "  ",
	},
	Format: table.FormatOptions{
		Footer: text.FormatUpper,
		Header: text.FormatUpper,
		Row:    text.FormatDefault,
	},
	Options: table.Options{
		DrawBorder:      false,
		SeparateColumns: false,
		SeparateFooter:  false,
		SeparateHeader:  false,
		SeparateRows:    false,
	},
}

var cfgFile string
var VmName string
var VmId string
var Humanize = true
var ShowUUID = false
var CheckReqStat = true

var defaultHost = "localhost"
var defaultPort = 50051
var defaultTimeout = 5

const (
	TXT = iota
	JSON
	YAML
)

var outputFormat = TXT

var outputFormatString = "txt"

var rootCmd = &cobra.Command{
	Use:     "cirrinactl",
	Version: mainVersion,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return rpc.GetConn()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		rpc.Finish()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func disableFlagSorting(cmd *cobra.Command) {
	cmd.Flags().SortFlags = false
	cmd.PersistentFlags().SortFlags = false
	cmd.InheritedFlags().SortFlags = false
}

func addNameOrIdArgs(cmd *cobra.Command, nameArg *string, idArg *string, objTypeName string) {
	cmd.Flags().StringVarP(nameArg, "name", "n", *nameArg, fmt.Sprintf("Name of %s", objTypeName))
	cmd.Flags().StringVarP(idArg, "id", "i", *idArg, fmt.Sprintf("Id of %s", objTypeName))
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

	rootCmd.AddCommand(VmCmd)
	rootCmd.AddCommand(DiskCmd)
	rootCmd.AddCommand(IsoCmd)
	rootCmd.AddCommand(NicCmd)
	rootCmd.AddCommand(SwitchCmd)
	rootCmd.AddCommand(TuiCmd)
	rootCmd.AddCommand(HostCmd)
	rootCmd.AddCommand(ReqStatCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".cirrinactl")
	}
	viper.SetEnvPrefix("CIRRINACTL")
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	rpc.ServerName = viper.GetString("server")
	rpc.ServerPort = viper.GetUint16("port")
	rpc.ServerTimeout = viper.GetUint64("timeout")
}

var mainVersion = "unknown"
