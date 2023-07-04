package main

import (
	"cirrina/cirrina"
	"context"
	"fmt"
	"log"
)

func addDisk(namePtr *string, c cirrina.VMInfoClient, ctx context.Context, descrPtr *string, sizePtr *string, diskTypePtr *string) {
	var thisDiskType cirrina.DiskType

	if *namePtr == "" {
		log.Fatalf("Name not specified")
		return
	}

	if *diskTypePtr == "" {
		log.Fatalf("Disk type not specified")
		return
	}
	if *diskTypePtr == "NVME" {
		thisDiskType = cirrina.DiskType_NVME
	} else if *diskTypePtr == "AHCI" {
		thisDiskType = cirrina.DiskType_AHCIHD
	} else if *diskTypePtr == "VIRTIOBLK" {
		thisDiskType = cirrina.DiskType_VIRTIOBLK
	} else {
		log.Fatalf("Invalid disk type specified")
		return
	}

	res, err := c.AddDisk(ctx, &cirrina.DiskInfo{
		Name:        namePtr,
		Description: descrPtr,
		Size:        sizePtr,
		DiskType:    &thisDiskType,
	})
	if err != nil {
		log.Fatalf("could not create Disk: %v", err)
		return
	}
	fmt.Printf("Created Disk %v\n", res.Value)

}
