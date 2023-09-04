package cmd

import (
	conn2 "cirrina/cirrinactl/rpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
)

var cfgFile string
var VmName string
var VmId string

var rootCmd = &cobra.Command{
	Use: "cirrinactl",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "C", cfgFile, "config file (default is $HOME/.cirrinactl.yaml)")

	rootCmd.PersistentFlags().StringP("server", "S", "localhost", "server")
	err := viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server"))
	if err != nil {
		log.Fatal(err)
	}

	rootCmd.PersistentFlags().Uint16P("port", "P", uint16(50051), "port")
	err = viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
	if err != nil {
		log.Fatal(err)
	}

	rootCmd.PersistentFlags().Uint64P("timeout", "T", uint64(1), "timeout in seconds")
	err = viper.BindPFlag("timeout", rootCmd.PersistentFlags().Lookup("timeout"))
	if err != nil {
		log.Fatal(err)
	}

	// some VM commands are duplicated at the root
	rootCmd.AddCommand(VmCreateCmd)
	rootCmd.AddCommand(VmListCmd)
	rootCmd.AddCommand(VmStartCmd)
	rootCmd.AddCommand(VmStopCmd)
	rootCmd.AddCommand(VmDestroyCmd)
	rootCmd.AddCommand(VmConfigCmd)
	rootCmd.AddCommand(VmGetCmd)
	rootCmd.AddCommand(VmCom1Cmd)
	rootCmd.AddCommand(VmCom2Cmd)
	rootCmd.AddCommand(VmCom3Cmd)
	rootCmd.AddCommand(VmCom4Cmd)

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

	conn2.ServerName = viper.GetString("server")
	conn2.ServerPort = viper.GetUint16("port")
	conn2.ServerTimeout = viper.GetUint64("timeout")
}
