package util

import (
	"cirrina/cirrina"
	"cirrina/cirrinactl/rpc"
	"context"
	"fmt"
	"log"
)

func GetNics(c cirrina.VMInfoClient, ctx context.Context) {
	res, err := rpc.GetHostNics(c, ctx)
	if err != nil {
		log.Fatalf(err.Error())
	}
	for _, nic := range res {
		fmt.Printf("nic: name: %v\n", nic.InterfaceName)
	}
}
