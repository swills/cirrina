package cmd

import (
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"cirrina/cirrinactl/rpc"
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

var (
	cfgFile           string
	VMName            string
	VMID              string
	Humanize          = true
	ShowUUID          = false
	CheckReqStat      = true
	ShowDiskSizeUsage = false
)

var (
	defaultHost    = "localhost"
	defaultPort    = 50051
	defaultTimeout = 5
)

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
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		return rpc.GetConn()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
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
	rpc.ServerTimeout = viper.GetInt64("timeout")
}

var mainVersion = "unknown"
