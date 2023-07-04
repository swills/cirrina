package main

import (
	"cirrina/cirrina"
	"context"
	"fmt"
	"io"
	"log"
)

func getHostNics(c cirrina.VMInfoClient, ctx context.Context) {
	res, err := c.GetNetInterfaces(ctx, &cirrina.NetInterfacesReq{})
	if err != nil {
		log.Fatalf("could not get host nics: %v", err)
		return
	}
	for {
		hostNic, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("GetNetInterfaces failed: %v", err)
		}
		fmt.Printf("nic: name: %v\n", hostNic.InterfaceName)
	}

}
