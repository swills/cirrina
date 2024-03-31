package rpc

import (
	"bufio"
	"cirrina/cirrina"
	"context"
	"errors"
	"fmt"

	"io"
	"os"

	"google.golang.org/grpc/status"
)

func AddDisk(diskName string, diskDescription string, diskSize string,
	diskType string, diskDevType string, diskCache bool, diskDirect bool,
) (string, error) {

	var thisDiskType cirrina.DiskType
	var thisDiskDevType cirrina.DiskDevType

	switch {
	case diskType == "NVME" || diskType == "nvme":
		thisDiskType = cirrina.DiskType_NVME
	case diskType == "AHCI" || diskType == "ahci" || diskType == "ahcihd":
		thisDiskType = cirrina.DiskType_AHCIHD
	case diskType == "VIRTIOBLK" || diskType == "virtioblk" || diskType == "virtio-blk":
		thisDiskType = cirrina.DiskType_VIRTIOBLK
	default:
		return "", fmt.Errorf("invalid disk type %s", diskType)
	}

	switch {
	case diskDevType == "FILE" || diskDevType == "file":
		thisDiskDevType = cirrina.DiskDevType_FILE
	case diskDevType == "ZVOL" || diskDevType == "zvol":
		thisDiskDevType = cirrina.DiskDevType_ZVOL
	default:
		return "", fmt.Errorf("invalid disk dev type %s", diskDevType)
	}

	newDiskInfo := &cirrina.DiskInfo{
		Name:        &diskName,
		Description: &diskDescription,
		Size:        &diskSize,
		DiskType:    &thisDiskType,
		DiskDevType: &thisDiskDevType,
		Cache:       &diskCache,
		Direct:      &diskDirect,
	}

	var diskId *cirrina.DiskId
	var err error
	diskId, err = serverClient.AddDisk(defaultServerContext, newDiskInfo)
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	return diskId.Value, nil
}

func GetDiskInfo(diskId string) (DiskInfo, error) {
	var err error

	var k *cirrina.DiskInfo
	k, err = serverClient.GetDiskInfo(defaultServerContext, &cirrina.DiskId{Value: diskId})
	if err != nil {
		return DiskInfo{}, errors.New(status.Convert(err).Message())
	}

	aDiskType := "unknown"
	switch *k.DiskType {
	case cirrina.DiskType_NVME:
		aDiskType = "nvme"
	case cirrina.DiskType_AHCIHD:
		aDiskType = "ahcihd"
	case cirrina.DiskType_VIRTIOBLK:
		aDiskType = "virtio-blk"
	}

	aDiskDevType := "unknown"
	if *k.DiskDevType == cirrina.DiskDevType_FILE {
		aDiskDevType = "file"
	} else if *k.DiskDevType == cirrina.DiskDevType_ZVOL {
		aDiskDevType = "zvol"
	}

	return DiskInfo{
		Name:        *k.Name,
		Descr:       *k.Description,
		Size:        *k.SizeNum,
		Usage:       *k.UsageNum,
		DiskType:    aDiskType,
		DiskDevType: aDiskDevType,
		Cache:       *k.Cache,
		Direct:      *k.Direct,
	}, nil
}

func GetDisks() ([]string, error) {
	var err error

	var rv []string

	var res cirrina.VMInfo_GetDisksClient
	res, err = serverClient.GetDisks(defaultServerContext, &cirrina.DisksQuery{})
	if err != nil {
		return []string{}, errors.New(status.Convert(err).Message())
	}

	for {
		VmDisk, err := res.Recv()
		if err == io.EOF {
			break
		}
		rv = append(rv, VmDisk.Value)
	}

	return rv, nil
}

func RmDisk(idPtr string) error {
	var err error

	var res *cirrina.ReqBool
	res, err = serverClient.RemoveDisk(defaultServerContext, &cirrina.DiskId{Value: idPtr})
	if err != nil {
		return errors.New(status.Convert(err).Message())
	}
	if !res.Success {
		return errors.New("disk delete failure")
	}
	return nil
}

func DiskNameToId(name string) (string, error) {
	var diskId string
	var err error
	if name == "" {
		return "", errors.New("disk name not specified")
	}

	var diskIds []string
	diskIds, err = GetDisks()
	if err != nil {
		return "", err
	}

	found := false
	var res DiskInfo
	for _, aDiskId := range diskIds {
		res, err = GetDiskInfo(aDiskId)
		if err != nil {
			return "", err
		}
		if res.Name == name {
			if found {
				return "", errors.New("duplicate disk found")
			}
			found = true
			diskId = aDiskId
		}
	}
	if !found {
		return "", &NotFoundError{}
	}
	return diskId, nil
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

func DiskGetVmId(id string) (string, error) {
	var err error
	if id == "" {
		return "", errors.New("disk id not specified")
	}

	var vmId *cirrina.VMID
	vmId, err = serverClient.GetDiskVm(defaultServerContext, &cirrina.DiskId{Value: id})
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	return vmId.Value, nil
}

func UpdateDisk(id string, newDesc *string, newType *string, direct *bool, cache *bool) error {
	var err error

	if id == "" {
		return errors.New("id not specified")
	}

	diu := cirrina.DiskInfoUpdate{
		Id: id,
	}

	if newDesc != nil {
		diu.Description = newDesc
	}

	DiskTypeNvme := cirrina.DiskType_NVME
	DiskTypeAHCIHD := cirrina.DiskType_AHCIHD
	DiskTypeVirtIoBlk := cirrina.DiskType_VIRTIOBLK
	if newType != nil {
		switch {
		case *newType == "NVME" || *newType == "nvme":
			diu.DiskType = &DiskTypeNvme
		case *newType == "AHCIHD" || *newType == "ahcihd" || *newType == "AHCI" || *newType == "ahci":
			diu.DiskType = &DiskTypeAHCIHD
		case *newType == "VIRTIO-BLK" || *newType == "virtio-blk" || *newType == "VIRTIOBLK" || *newType == "virtioblk":
			diu.DiskType = &DiskTypeVirtIoBlk
		default:
			return errors.New("invalid disk type specified " + *newType)
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
		return errors.New(status.Convert(err).Message())
	}
	if !res.Success {
		return errors.New("failed to update disk")
	}
	return nil
}

func DiskUpload(diskId string, diskChecksum string,
	diskSize uint64, diskFile *os.File) (<-chan UploadStat, error) {
	uploadStatChan := make(chan UploadStat, 1)

	if diskId == "" {
		return uploadStatChan, errors.New("empty disk id")
	}

	// actually send file, sending status to status channel
	go func(diskFile *os.File, uploadStatChan chan<- UploadStat) {
		defer func(diskFile *os.File) {
			_ = diskFile.Close()
		}(diskFile)
		var err error

		// prevent timeouts
		defaultServerContext = context.Background()

		thisDiskId := cirrina.DiskId{Value: diskId}

		setupReq := &cirrina.DiskImageRequest{
			Data: &cirrina.DiskImageRequest_Diskuploadinfo{
				Diskuploadinfo: &cirrina.DiskUploadInfo{
					Diskid:    &thisDiskId,
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
				Err:           errors.New(status.Convert(err).Message()),
			}
		}
		if stream == nil {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           errors.New("nil stream"),
			}
			return
		}

		err = stream.Send(setupReq)
		if err != nil {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           errors.New(status.Convert(err).Message()),
			}
		}

		reader := bufio.NewReader(diskFile)
		buffer := make([]byte, 1024*1024)

		var complete bool
		var n int
		for !complete {
			n, err = reader.Read(buffer)
			if err == io.EOF {
				complete = true
			}
			if err != nil && err != io.EOF {
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
					Err:           errors.New(status.Convert(err).Message()),
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
				Err:           errors.New(status.Convert(err).Message()),
			}
		}
		if !reply.Success {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           errors.New("failed"),
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
