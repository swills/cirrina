package rpc

import (
	"bufio"
	"cirrina/cirrina"
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"io"
	"os"
	"time"
)

func AddDisk(diskName string, diskDescription string, diskSize string,
	diskType string, diskDevType string, diskCache bool, diskDirect bool,
) (string, error) {

	var thisDiskType cirrina.DiskType
	var thisDiskDevType cirrina.DiskDevType

	if diskType == "NVME" || diskType == "nvme" {
		thisDiskType = cirrina.DiskType_NVME
	} else if diskType == "AHCI" || diskType == "ahci" || diskType == "ahcihd" {
		thisDiskType = cirrina.DiskType_AHCIHD
	} else if diskType == "VIRTIOBLK" || diskType == "virtioblk" || diskType == "virtio-blk" {
		thisDiskType = cirrina.DiskType_VIRTIOBLK
	} else {
		return "", fmt.Errorf("invalid disk type %s", diskType)
	}

	if diskDevType == "FILE" || diskDevType == "file" {
		thisDiskDevType = cirrina.DiskDevType_FILE
	} else if diskDevType == "ZVOL" || diskDevType == "zvol" {
		thisDiskDevType = cirrina.DiskDevType_ZVOL
	} else {
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

	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()
	var diskId *cirrina.DiskId
	diskId, err = c.AddDisk(ctx, newDiskInfo)
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	return diskId.Value, nil
}

func GetDiskInfo(diskId string) (DiskInfo, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return DiskInfo{}, err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var k *cirrina.DiskInfo
	k, err = c.GetDiskInfo(ctx, &cirrina.DiskId{Value: diskId})
	if err != nil {
		return DiskInfo{}, errors.New(status.Convert(err).Message())
	}

	aDiskType := "unknown"
	if *k.DiskType == cirrina.DiskType_NVME {
		aDiskType = "nvme"
	} else if *k.DiskType == cirrina.DiskType_AHCIHD {
		aDiskType = "ahcihd"
	} else if *k.DiskType == cirrina.DiskType_VIRTIOBLK {
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
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return []string{}, err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var rv []string

	var res cirrina.VMInfo_GetDisksClient
	res, err = c.GetDisks(ctx, &cirrina.DisksQuery{})
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
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var res *cirrina.ReqBool
	res, err = c.RemoveDisk(ctx, &cirrina.DiskId{Value: idPtr})
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
		return "", errors.New("disk not found")
	}
	return diskId, nil
}

func DiskIdToName(id string) (string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var res *cirrina.DiskInfo
	res, err = c.GetDiskInfo(ctx, &cirrina.DiskId{Value: id})
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	return *res.Name, nil
}

func DiskGetVm(id string) (string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	if id == "" {
		return "", errors.New("disk id not specified")
	}

	var vmId *cirrina.VMID
	vmId, err = c.GetDiskVm(ctx, &cirrina.DiskId{Value: id})
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	if vmId.Value == "" {
		return "", nil
	}
	var vmName string
	vmName, err = VmIdToName(vmId.Value)
	if err != nil {
		return "", err
	}
	return vmName, nil
}

func UpdateDisk(id string, newDesc *string, newType *string) error {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

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
		if *newType == "NVME" || *newType == "nvme" {
			diu.DiskType = &DiskTypeNvme
		} else if *newType == "AHCIHD" || *newType == "ahcihd" || *newType == "AHCI" || *newType == "ahci" {
			diu.DiskType = &DiskTypeAHCIHD
		} else if *newType == "VIRTIO-BLK" || *newType == "virtio-blk" ||
			*newType == "VIRTIOBLK" || *newType == "virtioblk" {
			diu.DiskType = &DiskTypeVirtIoBlk
		} else {
			return errors.New("invalid disk type specified " + *newType)
		}
	}
	var res *cirrina.ReqBool
	res, err = c.SetDiskInfo(ctx, &diu)
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
		var conn *grpc.ClientConn
		var c cirrina.VMInfoClient
		var err error

		conn, c, err = SetupConnNoTimeoutNoContext()
		if err != nil {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           err,
			}
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)

		timeout := 1 * time.Hour
		var longCtx context.Context
		var longCancel context.CancelFunc

		longCtx, longCancel = context.WithTimeout(context.Background(), timeout)
		defer longCancel()

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
		stream, err = c.UploadDisk(longCtx)
		if err != nil {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           errors.New(status.Convert(err).Message()),
			}
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
