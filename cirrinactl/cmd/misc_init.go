//go:build !test

package cmd

func init() {
	disableFlagSorting(ReqStatCmd)
	ReqStatCmd.Flags().StringVarP(&ReqID, "id", "i", ReqID, "ID of request")

	err := ReqStatCmd.MarkFlagRequired("id")
	if err != nil {
		panic(err)
	}
}
