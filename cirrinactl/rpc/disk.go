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

	return diskID.GetValue(), nil
}

func GetDiskInfo(diskID string) (DiskInfo, error) {
	var err error

	var info DiskInfo

	var diskInfo *cirrina.DiskInfo

	diskInfo, err = serverClient.GetDiskInfo(defaultServerContext, &cirrina.DiskId{Value: diskID})
	if err != nil {
		return DiskInfo{}, fmt.Errorf("unable to get disk info: %w", err)
	}

	if diskInfo == nil {
		return DiskInfo{}, errInvalidServerResponse
	}

	info.Name = diskInfo.GetName()
	info.Descr = diskInfo.GetDescription()
	info.DiskType = mapDiskTypeTypeToString(diskInfo.GetDiskType())
	info.DiskDevType = mapDiskDevTypeTypeToString(diskInfo.GetDiskDevType())
	info.Cache = diskInfo.GetCache()
	info.Direct = diskInfo.GetDirect()

	return info, nil
}

func GetDiskSizeUsage(diskID string) (DiskSizeUsage, error) {
	diskSizeUsage, err := serverClient.GetDiskSizeUsage(defaultServerContext, &cirrina.DiskId{Value: diskID})
	if err != nil {
		return DiskSizeUsage{}, fmt.Errorf("unable to get disk info: %w", err)
	}

	return DiskSizeUsage{Size: diskSizeUsage.GetSizeNum(), Usage: diskSizeUsage.GetUsageNum()}, nil
}

func GetDisks() ([]string, error) {
	var err error

	var disks []string

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

		disks = append(disks, VMDisk.GetValue())
	}

	return disks, nil
}

func RmDisk(idPtr string) error {
	var err error

	var res *cirrina.ReqBool

	res, err = serverClient.RemoveDisk(defaultServerContext, &cirrina.DiskId{Value: idPtr})
	if err != nil {
		return fmt.Errorf("unable to remove disk: %w", err)
	}

	if !res.GetSuccess() {
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

	var diskIDs []string

	diskIDs, err = GetDisks()
	if err != nil {
		return "", err
	}

	found := false

	var res DiskInfo
	for _, aDiskID := range diskIDs {
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
		return "", ErrNotFound
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

func DiskGetVMID(diskID string) (string, error) {
	var err error

	if diskID == "" {
		return "", errDiskEmptyName
	}

	var vmID *cirrina.VMID

	vmID, err = serverClient.GetDiskVM(defaultServerContext, &cirrina.DiskId{Value: diskID})
	if err != nil {
		return "", fmt.Errorf("unable to get disk VM: %w", err)
	}

	return vmID.GetValue(), nil
}

func UpdateDisk(diskID string, newDesc *string, newType *string, direct *bool, cache *bool) error {
	var err error

	if diskID == "" {
		return errDiskEmptyID
	}

	diu := cirrina.DiskInfoUpdate{
		Id: diskID,
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

	if !res.GetSuccess() {
		return errReqFailed
	}

	return nil
}

func WipeDisk(diskID string) (string, error) {
	if diskID == "" {
		return "", errDiskEmptyID
	}

	var err error

	var reqID *cirrina.RequestID

	reqID, err = serverClient.WipeDisk(defaultServerContext, &cirrina.DiskId{Value: diskID})
	if err != nil {
		return "", fmt.Errorf("error wiping disk: %w", err)
	}

	return reqID.GetValue(), nil
}

func diskUploadFile(diskID string, diskSize uint64, diskChecksum string,
	diskFile *os.File, uploadStatChan chan<- UploadStat) {
	var err error

	var stream cirrina.VMInfo_UploadDiskClient

	defer func(diskFile *os.File) {
		_ = diskFile.Close()
	}(diskFile)

	// prevent timeouts
	defaultServerContext = context.Background()

	// setup stream
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

	err = diskUploadFileSetupRequest(diskID, diskSize, diskChecksum, stream, uploadStatChan)
	if err != nil {
		return
	}

	err = diskUploadFileBytes(diskFile, stream, uploadStatChan)
	if err != nil {
		return
	}

	diskUploadFileComplete(stream, uploadStatChan)
}

func diskUploadFileComplete(stream cirrina.VMInfo_UploadDiskClient, uploadStatChan chan<- UploadStat) {
	var err error

	var reply *cirrina.ReqBool

	reply, err = stream.CloseAndRecv()
	if err != nil {
		uploadStatChan <- UploadStat{
			UploadedChunk: false,
			Complete:      false,
			Err:           fmt.Errorf("unable to upload disk: %w", err),
		}

		return
	}

	if !reply.GetSuccess() {
		uploadStatChan <- UploadStat{
			UploadedChunk: false,
			Complete:      false,
			Err:           errReqFailed,
		}

		return
	}

	// finished!
	uploadStatChan <- UploadStat{
		UploadedChunk: false,
		Complete:      true,
		Err:           nil,
	}
}

func diskUploadFileBytes(diskFile *os.File, stream cirrina.VMInfo_UploadDiskClient,
	uploadStatChan chan<- UploadStat) error {
	var err error
	// send disk bytes
	reader := bufio.NewReader(diskFile)
	buffer := make([]byte, 1024*1024)

	var complete bool

	var nBytesRead int
	for !complete {
		nBytesRead, err = reader.Read(buffer)
		if errors.Is(err, io.EOF) {
			complete = true
		}

		if err != nil && !errors.Is(err, io.EOF) {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           err,
			}

			return fmt.Errorf("error reading file bytes: %w", err)
		}

		dataReq := &cirrina.DiskImageRequest{
			Data: &cirrina.DiskImageRequest_Image{
				Image: buffer[:nBytesRead],
			},
		}

		err = stream.Send(dataReq)
		if err != nil {
			uploadStatChan <- UploadStat{
				UploadedChunk: false,
				Complete:      false,
				Err:           err,
			}

			return fmt.Errorf("error sending file bytes: %w", err)
		}
		uploadStatChan <- UploadStat{
			UploadedChunk: true,
			Complete:      false,
			UploadedBytes: nBytesRead,
			Err:           nil,
		}
	}

	return nil
}

func diskUploadFileSetupRequest(diskID string, diskSize uint64, diskChecksum string,
	stream cirrina.VMInfo_UploadDiskClient, uploadStatChan chan<- UploadStat) error {
	var err error

	// create setup request
	setupReq := &cirrina.DiskImageRequest{
		Data: &cirrina.DiskImageRequest_Diskuploadinfo{
			Diskuploadinfo: &cirrina.DiskUploadInfo{
				Diskid:    &cirrina.DiskId{Value: diskID},
				Size:      diskSize,
				Sha512Sum: diskChecksum,
			},
		},
	}

	// send setup request
	err = stream.Send(setupReq)
	if err != nil {
		uploadStatChan <- UploadStat{
			UploadedChunk: false,
			Complete:      false,
			Err:           fmt.Errorf("unable to upload disk: %w", err),
		}

		return fmt.Errorf("unable to upload disk: %w", err)
	}

	return nil
}

func DiskUpload(diskID string, diskChecksum string,
	diskSize uint64, diskFile *os.File,
) (<-chan UploadStat, error) {
	uploadStatChan := make(chan UploadStat, 1)

	if diskID == "" {
		return uploadStatChan, errDiskEmptyID
	}

	// actually send file, sending status to status channel
	go diskUploadFile(diskID, diskSize, diskChecksum, diskFile, uploadStatChan)

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
