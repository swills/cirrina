package main

import "cirrina/cirrina"

func mapDiskDevTypeTypeToDBString(diskDevType cirrina.DiskDevType) (string, error) {
	switch diskDevType {
	case cirrina.DiskDevType_FILE:
		return "FILE", nil
	case cirrina.DiskDevType_ZVOL:
		return "ZVOL", nil
	default:
		return "", errDiskInvalidDevType
	}
}

func mapDiskDevTypeDBStringToType(diskDevType string) (*cirrina.DiskDevType, error) {
	DiskDevTypeFile := cirrina.DiskDevType_FILE
	DiskDevTypeZvol := cirrina.DiskDevType_ZVOL

	switch diskDevType {
	case "FILE":
		return &DiskDevTypeFile, nil
	case "ZVOL":
		return &DiskDevTypeZvol, nil
	default:
		return nil, errDiskInvalidDevType
	}
}

func mapDiskTypeTypeToDBString(diskType cirrina.DiskType) (string, error) {
	switch diskType {
	case cirrina.DiskType_NVME:
		return "NVME", nil
	case cirrina.DiskType_AHCIHD:
		return "AHCI-HD", nil
	case cirrina.DiskType_VIRTIOBLK:
		return "VIRTIO-BLK", nil
	default:
		return "", errDiskInvalidType
	}
}

func mapDiskTypeDBStringToType(diskType string) (*cirrina.DiskType, error) {
	DiskTypeNVME := cirrina.DiskType_NVME
	DiskTypeAHCI := cirrina.DiskType_AHCIHD
	DiskTypeVIRT := cirrina.DiskType_VIRTIOBLK

	switch diskType {
	case "NVME":
		return &DiskTypeNVME, nil
	case "AHCI-HD":
		return &DiskTypeAHCI, nil
	case "VIRTIO-BLK":
		return &DiskTypeVIRT, nil
	default:
		return nil, errDiskInvalidType
	}
}
