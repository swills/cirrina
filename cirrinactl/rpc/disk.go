package rpc

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"cirrina/cirrina"
)

func AddDisk(diskName string, diskDescription string, diskSize string,
	diskType string, diskDevType string, diskCache bool, diskDirect bool,
) (string, error) {
	var err error
	var thisDiskType *cirrina.DiskType
	var thisDiskDevType *cirrina.DiskDevType

	thisDiskType, err = mapDiskTypeStringToType(diskType)
	if err != nil {
		return "", err
	}

	thisDiskDevType, err = mapDiskDevTypeStringToType(diskDevType)
	if err != nil {
		return "", err
	}

	newDiskInfo := &cirrina.DiskInfo{
		Name:        &diskName,
		Description: &diskDescription,
		Size:        &diskSize,
		DiskType:    thisDiskType,
		DiskDevType: thisDiskDevType,
		Cache:       &diskCache,
		Direct:      &diskDirect,
	}

	var diskID *cirrina.DiskId
	diskID, err = serverClient.AddDisk(defaultServerContext, newDiskInfo)
	if err != nil {
		return "", fmt.Errorf("unable to add disk: %w", err)
	}

	return diskID.Value, nil
}

func GetDiskInfo(diskID string) (DiskInfo, error) {
	var err error
	var info DiskInfo
	var k *cirrina.DiskInfo

	k, err = serverClient.GetDiskInfo(defaultServerContext, &cirrina.DiskId{Value: diskID})
	if err != nil {
		return DiskInfo{}, fmt.Errorf("unable to get disk info: %w", err)
	}
	if k == nil {
		return DiskInfo{}, errInvalidServerResponse
	}

	if k.Name != nil {
		info.Name = *k.Name
	}

	if k.Description != nil {
		info.Descr = *k.Description
	}

	if k.SizeNum != nil {
		info.Size = *k.SizeNum
	}

	if k.UsageNum != nil {
		info.Usage = *k.UsageNum
	}

	if k.DiskType != nil {
		info.DiskType = mapDiskTypeTypeToString(*k.DiskType)
	}

	if k.DiskDevType != nil {
		info.DiskDevType = mapDiskDevTypeTypeToString(*k.DiskDevType)
	}

	if k.Cache != nil {
		info.Cache = *k.Cache
	}

	if k.Direct != nil {
		info.Direct = *k.Direct
	}

	return info, nil
}

func GetDisks() ([]string, error) {
	var err error

	var rv []string

	var res cirrina.VMInfo_GetDisksClient
	res, err = serverClient.GetDisks(defaultServerContext, &cirrina.DisksQuery{})
	if err != nil {
		return []string{}, fmt.Errorf("unable to get disks: %w", err)
	}

	for {
		VMDisk, err := res.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		rv = append(rv, VMDisk.Value)
	}

	return rv, nil
}

func RmDisk(idPtr string) error {
	var err error

	var res *cirrina.ReqBool
	res, err = serverClient.RemoveDisk(defaultServerContext, &cirrina.DiskId{Value: idPtr})
	if err != nil {
		return fmt.Errorf("unable to remove disk: %w", err)
	}
	if !res.Success {
		return errReqFailed
	}

	return nil
}

func DiskNameToID(name string) (string, error) {
	var diskID string
	var err error
	if name == "" {
		return "", errDiskEmptyName
	}

	var diskIds []string
	diskIds, err = GetDisks()
	if err != nil {
		return "", err
	}

	found := false
	var res DiskInfo
	for _, aDiskID := range diskIds {
		res, err = GetDiskInfo(aDiskID)
		if err != nil {
			return "", err
		}
		if res.Name == name {
			if found {
				return "", errDiskDuplicate
			}
			found = true
			diskID = aDiskID
		}
	}
	if !found {
		return "", errNotFound
	}

	return diskID, nil
}

// func DiskIdToName(id string) (string, error) {
// 	var err error
//
// 	var res *cirrina.DiskInfo
// 	res, err = serverClient.GetDiskInfo(defaultServerContext, &cirrina.DiskId{Value: id})
// 	if err != nil {
// 		return "", errors.New(status.Convert(err).Message())
// 	}
// 	return *res.Name, nil
// }

func DiskGetVMID(id string) (string, error) {
	var err error
	if id == "" {
		return "", errDiskEmptyName
	}

	var vmID *cirrina.VMID
	vmID, err = serverClient.GetDiskVM(defaultServerContext, &cirrina.DiskId{Value: id})
	if err != nil {
		return "", fmt.Errorf("unable to get disk VM: %w", err)
	}

	return vmID.Value, nil
}

func UpdateDisk(id string, newDesc *string, newType *string, direct *bool, cache *bool) error {
	var err error

	if id == "" {
		return errDiskEmptyID
	}

	diu := cirrina.DiskInfoUpdate{
		Id: id,
	}

	if newDesc != nil {
		diu.Description = newDesc
	}

	if newType != nil {
		diu.DiskType, err = mapDiskTypeStringToType(*newType)
		if err != nil {
			return err
		}
	}

	if direct != nil {
		diu.Direct = direct
	}

	if cache != nil {
		diu.Cache = cache
	}

	var res *cirrina.ReqBool
	res, err = serverClient.SetDiskInfo(defaultServerContext, &diu)
	if err != nil {
		return fmt.Errorf("unable to set disk info: %w", err)
	}
	if !res.Success {
		return errReqFailed
	}

	return nil
}

func DiskUpload(diskID string, diskChecksum string,
	diskSize uint64, diskFile *os.File) (<-chan UploadStat, error) {
	uploadStatChan := make(chan UploadStat, 1)

	if diskID == "" {
		return uploadStatChan, errDiskEmptyID
	}

	// actually send file, sending status to status channel
	go func(diskFile *os.File, uploadStatChan chan<- UploadStat) {
		defer func(diskFile *os.File) {
			_ = diskFile.Close()
		}(diskFile)
		var err error

		// prevent timeouts
		defaultServerContext = context.Background()

		thisDiskID := cirrina.DiskId{Value: diskID}

		setupReq := &cirrina.DiskImageRequest{
			Data: &cirrina.DiskImageRequest_Diskuploadinfo{
				Diskuploadinfo: &cirrina.DiskUploadInfo{
					Diskid:    &thisDiskID,
					Size:      diskSize,
					Sha512Sum: diskChecksum,
				},
			},
		}

		var stream cirrina.VMInfo_UploadDiskClient
		stream, err = serverClient.UploadDisk(defaultServerContext)
		if err != nil {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           fmt.Errorf("unable to upload disk: %w", err),
			}
		}
		if stream == nil {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           errInternalError,
			}

			return
		}

		err = stream.Send(setupReq)
		if err != nil {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           fmt.Errorf("unable to upload disk: %w", err),
			}
		}

		reader := bufio.NewReader(diskFile)
		buffer := make([]byte, 1024*1024)

		var complete bool
		var n int
		for !complete {
			n, err = reader.Read(buffer)
			if errors.Is(err, io.EOF) {
				complete = true
			}
			if err != nil && !errors.Is(err, io.EOF) {
				uploadStatChan <- UploadStat{
					UploadedChunk: false,
					Complete:      false,
					Err:           err,
				}
			}
			dataReq := &cirrina.DiskImageRequest{
				Data: &cirrina.DiskImageRequest_Image{
					Image: buffer[:n],
				},
			}
			err = stream.Send(dataReq)
			if err != nil {
				uploadStatChan <- UploadStat{
					UploadedChunk: false,
					Complete:      false,
					Err:           fmt.Errorf("unable to upload disk: %w", err),
				}
			}
			uploadStatChan <- UploadStat{
				UploadedChunk: true,
				Complete:      false,
				UploadedBytes: n,
				Err:           nil,
			}
		}

		var reply *cirrina.ReqBool
		reply, err = stream.CloseAndRecv()
		if err != nil {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           fmt.Errorf("unable to upload disk: %w", err),
			}
		}
		if !reply.Success {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           errReqFailed,
			}
		}

		// finished!
		uploadStatChan <- UploadStat{
			UploadedChunk: false,
			Complete:      true,
			Err:           nil,
		}
	}(diskFile, uploadStatChan)

	return uploadStatChan, nil
}

func mapDiskTypeStringToType(diskType string) (*cirrina.DiskType, error) {
	DiskTypeNvme := cirrina.DiskType_NVME
	DiskTypeAHCIHD := cirrina.DiskType_AHCIHD
	DiskTypeVirtIoBlk := cirrina.DiskType_VIRTIOBLK

	switch {
	case diskType == "NVME" || diskType == "nvme":
		return &DiskTypeNvme, nil
	case diskType == "AHCIHD" || diskType == "ahcihd" || diskType == "AHCI" || diskType == "ahci":
		return &DiskTypeAHCIHD, nil
	case diskType == "VIRTIO-BLK" || diskType == "virtio-blk" || diskType == "VIRTIOBLK" || diskType == "virtioblk":
		return &DiskTypeVirtIoBlk, nil
	default:
		return nil, errDiskTypeUnknown
	}
}

func mapDiskDevTypeStringToType(diskDevType string) (*cirrina.DiskDevType, error) {
	DiskDevTypeFile := cirrina.DiskDevType_FILE
	DiskDevTypeZVOL := cirrina.DiskDevType_ZVOL

	switch {
	case diskDevType == "FILE" || diskDevType == "file":
		return &DiskDevTypeFile, nil
	case diskDevType == "ZVOL" || diskDevType == "zvol":
		return &DiskDevTypeZVOL, nil
	default:
		return nil, errDiskDevTypeUnknown
	}
}

func mapDiskTypeTypeToString(diskType cirrina.DiskType) string {
	switch diskType {
	case cirrina.DiskType_NVME:
		return "nvme"
	case cirrina.DiskType_AHCIHD:
		return "ahcihd"
	case cirrina.DiskType_VIRTIOBLK:
		return "virtio-blk"
	default:
		return ""
	}
}

func mapDiskDevTypeTypeToString(diskDevType cirrina.DiskDevType) string {
	switch diskDevType {
	case cirrina.DiskDevType_FILE:
		return "file"
	case cirrina.DiskDevType_ZVOL:
		return "zvol"
	default:
		return ""
	}
}
