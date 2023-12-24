package cmd

import (
	"cirrina/cirrinactl/rpc"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
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
var CheckReqStat = true

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
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	cobra.EnableCommandSorting = false

	rootCmd.PersistentFlags().StringVarP(&cfgFile,
		"config", "C", cfgFile, "config file (default $HOME/.cirrinactl.yaml)")

	rootCmd.PersistentFlags().StringP("server", "S", "localhost", "server")
	err := viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server"))
	if err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().Uint16P("port", "P", uint16(50051), "port")
	err = viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
	if err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().Uint64P("timeout", "T", uint64(2), "timeout in seconds")
	err = viper.BindPFlag("timeout", rootCmd.PersistentFlags().Lookup("timeout"))
	if err != nil {
		panic(err)
	}

	rootCmd.AddCommand(VmCmd)
	rootCmd.AddCommand(DiskCmd)
	rootCmd.AddCommand(NicCmd)
	rootCmd.AddCommand(SwitchCmd)
	rootCmd.AddCommand(IsoCmd)
	rootCmd.AddCommand(TuiCmd)
	rootCmd.AddCommand(ReqStatCmd)
	rootCmd.AddCommand(HostCmd)
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
