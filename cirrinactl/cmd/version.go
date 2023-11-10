package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var mainVersion = "unknown"

type Empty struct{}

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "print client version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("version: %s\n", mainVersion)
	},
}
