package cmd

import (
	"context"
	"log"
	"time"

	"cirrina/cirrinactl/rpc"
	"cirrina/cirrinactl/util"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var (
	IsoName        string
	IsoDescription string
	IsoId          string
	IsoIdChanged   bool
	IsoFilePath    string
	IsoUseHumanize bool
)

var IsoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List ISOs",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()

		util.ListIsos(c, ctx, IsoUseHumanize)
	},
}

var IsoCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an ISO",
	Long:  "Create a name entry for an ISO with no content -- see upload to add content",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		util.AddISO(&IsoName, c, ctx, &IsoDescription)
	},
}

var IsoUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload an ISO",
	Long:  "Upload an ISO image from local storage",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, _, _, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)

		timeout := time.Hour
		longCtx, longCancel := context.WithTimeout(context.Background(), timeout)
		defer longCancel()

		util.UploadIso(c, longCtx, &IsoId, &IsoFilePath)
	},
}

var IsoRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove an ISO",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c, ctx, cancel, err := rpc.SetupConn()
		if err != nil {
			log.Fatal(err)
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		defer cancel()
		if IsoId == "" {
			IsoId, err = rpc.IsoNameToId(&IsoName, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if IsoId == "" {
				log.Fatalf("Iso not found")
			}
		}
		if IsoName == "" {
			IsoName, err = rpc.IsoIdToName(IsoId, c, ctx)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		util.RmIso(IsoName, c, ctx)
	},
}

var IsoCmd = &cobra.Command{
	Use:   "iso",
	Short: "Create, list, modify, destroy ISOs",
}

func init() {
	IsoCreateCmd.Flags().StringVarP(&IsoName, "name", "n", IsoName, "name of ISO")
	IsoCreateCmd.Flags().StringVarP(&IsoDescription, "description", "d", IsoDescription, "description of ISO")
	err := IsoCreateCmd.MarkFlagRequired("name")
	if err != nil {
		log.Fatalf(err.Error())
	}
	IsoListCmd.Flags().BoolVarP(&IsoUseHumanize, "human", "H", IsoUseHumanize, "Print sizes in human readable form")

	IsoUploadCmd.Flags().StringVarP(&IsoId, "id", "i", IsoId, "Id of ISO to upload")
	IsoUploadCmd.Flags().StringVarP(&IsoFilePath, "path", "p", IsoFilePath, "Path to ISO File to upload")
	err = IsoUploadCmd.MarkFlagRequired("id")
	if err != nil {
		log.Fatalf(err.Error())
	}
	err = IsoUploadCmd.MarkFlagRequired("path")
	if err != nil {
		log.Fatalf(err.Error())
	}

	IsoRemoveCmd.Flags().StringVarP(&IsoName, "name", "n", IsoName, "name of iso")
	IsoRemoveCmd.Flags().StringVarP(&IsoId, "id", "i", DiskId, "id of iso")
	IsoRemoveCmd.MarkFlagsOneRequired("name", "id")

	IsoCmd.AddCommand(IsoListCmd)
	IsoCmd.AddCommand(IsoCreateCmd)
	IsoCmd.AddCommand(IsoUploadCmd)
	IsoCmd.AddCommand(IsoRemoveCmd)
}
