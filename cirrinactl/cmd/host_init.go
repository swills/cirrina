//go:build !test

package cmd

func init() {
	disableFlagSorting(HostCmd)

	disableFlagSorting(HostVersionCmd)

	disableFlagSorting(HostNicsCmd)

	HostCmd.AddCommand(HostNicsCmd)
	HostCmd.AddCommand(HostVersionCmd)
}
