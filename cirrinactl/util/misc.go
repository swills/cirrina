package util

import (
	"cirrina/cirrina"
	"cirrina/cirrinactl/rpc"
	"context"
	"fmt"
	"log"
)

func ReqStat(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	res, err := rpc.ReqStat(idPtr, c, ctx)
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Printf("complete: %v success: %v\n", res.Complete, res.Success)
}
