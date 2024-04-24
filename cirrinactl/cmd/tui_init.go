//go:build !test

package cmd

func init() {
	disableFlagSorting(TuiCmd)
}
