package main

import (
	"cirrina/cirrina"
	"context"
	"fmt"
	"log"
)

func ReqStat(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	res, err := c.RequestStatus(ctx, &cirrina.RequestID{Value: *idPtr})
	if err != nil {
		log.Fatalf("could not get req: %v", err)
	}
	fmt.Printf("complete: %v status: %v\n", res.Complete, res.Success)

}
