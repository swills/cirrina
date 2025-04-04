package main

import (
	"bufio"
	"context"
	"crypto/sha512"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"

	"cirrina/cirrina"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/requests"
	"cirrina/cirrinad/vm"
)

func (s *server) AddDisk(_ context.Context, diskInfo *cirrina.DiskInfo) (*cirrina.DiskId, error) {
	var diskType string

	var diskDevType string

	var err error

	defaultDiskDescription := ""
	defaultDiskType := "NVME"
	defaultDiskSize := config.Config.Disk.Default.Size
	defaultDiskDevType := "FILE"
	defaultDiskCache := true
	defaultDiskDirect := false

	if diskInfo.Name == nil || diskInfo.GetName() == "" {
		return &cirrina.DiskId{}, errInvalidName
	}

	if diskInfo.Size == nil || diskInfo.GetSize() == "" {
		diskInfo.Size = &defaultDiskSize
	}

	if diskInfo.Description == nil {
		diskInfo.Description = &defaultDiskDescription
	}

	if diskInfo.DiskType == nil {
		diskType = defaultDiskType
	} else {
		diskType, err = mapDiskTypeTypeToDBString(diskInfo.GetDiskType())
		if err != nil {
			return &cirrina.DiskId{}, err
		}
	}

	if diskInfo.DiskDevType == nil {
		diskDevType = defaultDiskDevType
	} else {
		diskDevType, err = mapDiskDevTypeTypeToDBString(diskInfo.GetDiskDevType())
		if err != nil {
			return &cirrina.DiskId{}, err
		}
	}

	if diskInfo.Cache == nil {
		diskInfo.Cache = &defaultDiskCache
	}

	if diskInfo.Direct == nil {
		diskInfo.Direct = &defaultDiskDirect
	}

	diskInst := &disk.Disk{
		Name:        diskInfo.GetName(),
		Description: diskInfo.GetDescription(),
		Type:        diskType,
		DevType:     diskDevType,
		DiskCache:   sql.NullBool{Bool: diskInfo.GetCache(), Valid: true},
		DiskDirect:  sql.NullBool{Bool: diskInfo.GetDirect(), Valid: true},
	}

	err = disk.Create(diskInst, diskInfo.GetSize())
	if err != nil {
		return nil, fmt.Errorf("error creating disk: %w", err)
	}

	return &cirrina.DiskId{Value: diskInst.ID}, nil
}

func (s *server) GetDiskInfo(_ context.Context, diskID *cirrina.DiskId) (*cirrina.DiskInfo, error) {
	var diskInfo cirrina.DiskInfo

	var err error

	defaultDiskCache := true
	defaultDiskDirect := false

	var diskInst *disk.Disk

	diskInst, err = validateGetDiskInfoRequest(diskID)
	if err != nil {
		return &cirrina.DiskInfo{}, err
	}

	diskInfo.Name = &diskInst.Name
	diskInfo.Description = &diskInst.Description

	if diskInst.DiskCache.Valid {
		diskInfo.Cache = &diskInst.DiskCache.Bool
	} else {
		diskInfo.Cache = &defaultDiskCache
	}

	if diskInst.DiskDirect.Valid {
		diskInfo.Direct = &diskInst.DiskDirect.Bool
	} else {
		diskInfo.Direct = &defaultDiskDirect
	}

	diskInfo.DiskType, err = mapDiskTypeDBStringToType(diskInst.Type)
	if err != nil {
		return &cirrina.DiskInfo{}, err
	}

	diskInfo.DiskDevType, err = mapDiskDevTypeDBStringToType(diskInst.DevType)
	if err != nil {
		return &cirrina.DiskInfo{}, err
	}

	// leave size/usage blank here since it's expensive, time-wise.
	// clients should call GetDiskSizeUsage instead

	return &diskInfo, nil
}

func (s *server) GetDiskSizeUsage(_ context.Context, diskID *cirrina.DiskId) (*cirrina.DiskSizeUsage, error) {
	var err error

	var diskInst *disk.Disk

	var diskInfo cirrina.DiskInfo

	var diskSizeUsage cirrina.DiskSizeUsage

	diskInst, err = validateGetDiskInfoRequest(diskID)
	if err != nil {
		return &cirrina.DiskSizeUsage{}, err
	}

	diskInfo.DiskType, err = mapDiskTypeDBStringToType(diskInst.Type)
	if err != nil {
		return &cirrina.DiskSizeUsage{}, err
	}

	var diskService disk.InfoServicer

	diskInfo.DiskDevType, err = mapDiskDevTypeDBStringToType(diskInst.DevType)
	if err != nil {
		return &cirrina.DiskSizeUsage{}, err
	}

	switch diskInfo.GetDiskDevType() {
	case cirrina.DiskDevType_FILE:
		diskService = disk.NewFileInfoService(disk.FileInfoFetcherImpl)
	case cirrina.DiskDevType_ZVOL:
		if config.Config.Disk.VM.Path.Zpool == "" {
			return &cirrina.DiskSizeUsage{}, errDiskZPoolNotConfigured
		}

		diskService = disk.NewZfsVolInfoService(disk.ZfsInfoFetcherImpl)
	default:
		return &cirrina.DiskSizeUsage{}, errDiskInvalidDevType
	}

	err = getDiskInfo(diskService, diskInst, &diskInfo)
	if err != nil {
		return &cirrina.DiskSizeUsage{}, err
	}

	size := diskInfo.GetSize()
	usage := diskInfo.GetUsage()
	sizeNum := diskInfo.GetSizeNum()
	usageNum := diskInfo.GetUsageNum()

	diskSizeUsage.Size = &size
	diskSizeUsage.Usage = &usage
	diskSizeUsage.SizeNum = &sizeNum
	diskSizeUsage.UsageNum = &usageNum

	return &diskSizeUsage, nil
}

func (s *server) RemoveDisk(_ context.Context, diskID *cirrina.DiskId) (*cirrina.ReqBool, error) {
	slog.Debug("deleting disk", "diskID", diskID.GetValue())

	res := cirrina.ReqBool{}
	res.Success = false

	diskUUID, err := uuid.Parse(diskID.GetValue())
	if err != nil {
		return &res, errInvalidID
	}

	diskInst, err := disk.GetByID(diskUUID.String())
	if err != nil {
		slog.Error("error getting disk", "disk", diskID.GetValue(), "err", err)

		return &res, errNotFound
	}

	if diskInst.Name == "" {
		slog.Debug("disk not found")

		return &res, errNotFound
	}

	// prevent deleting disk while it's being uploaded etc
	defer diskInst.Unlock()
	diskInst.Lock()

	err = diskInst.Delete()
	if err != nil {
		slog.Error("error deleting disk", "err", err)

		return &res, errDiskDeleteGeneric
	}

	res.Success = true

	return &res, nil
}

func (s *server) WipeDisk(_ context.Context, diskID *cirrina.DiskId) (*cirrina.RequestID, error) {
	diskUUID, err := uuid.Parse(diskID.GetValue())
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error parsing VM ID: %w", err)
	}

	diskInst, err := disk.GetByID(diskUUID.String())
	if err != nil {
		slog.Error("WipeDisk error getting disk", "disk", diskID.GetValue(), "err", err)

		return &cirrina.RequestID{}, fmt.Errorf("error getting VM: %w", err)
	}

	if diskInst.Name == "" {
		return &cirrina.RequestID{}, errNotFound
	}

	pendingReqIDs := requests.PendingReqExists(diskUUID.String())
	if len(pendingReqIDs) > 0 {
		return &cirrina.RequestID{}, errReqExists
	}

	// check that disk is not in use by a VM
	var diskVM *vm.VM

	diskVM, err = getDiskVM(diskUUID)
	if err != nil {
		return &cirrina.RequestID{}, err
	}

	if diskVM != nil {
		return &cirrina.RequestID{}, errDiskInUse
	}

	newReq, err := requests.CreateDiskReq(requests.DISKWIPE, diskUUID.String())
	if err != nil {
		return &cirrina.RequestID{}, fmt.Errorf("error creating request: %w", err)
	}

	return &cirrina.RequestID{Value: newReq.ID}, nil
}

func (s *server) GetDiskVM(_ context.Context, diskID *cirrina.DiskId) (*cirrina.VMID, error) {
	var err error

	var pvmID cirrina.VMID

	diskUUID, err := uuid.Parse(diskID.GetValue())
	if err != nil {
		return &pvmID, errInvalidID
	}

	diskInst, err := disk.GetByID(diskUUID.String())
	if err != nil {
		slog.Error("error getting disk", "disk", diskID.GetValue(), "err", err)

		return &pvmID, errNotFound
	}

	if diskInst.Name == "" {
		slog.Debug("disk not found")

		return &pvmID, errNotFound
	}

	var diskVM *vm.VM

	diskVM, err = getDiskVM(diskUUID)
	if err != nil {
		return &cirrina.VMID{}, err
	}

	if diskVM != nil {
		pvmID.Value = diskVM.ID
	}

	return &pvmID, nil
}

func (s *server) SetDiskInfo(_ context.Context, diu *cirrina.DiskInfoUpdate) (*cirrina.ReqBool, error) {
	var res cirrina.ReqBool
	res.Success = false

	if diu.GetId() == "" {
		return &res, errInvalidID
	}

	diskUUID, err := uuid.Parse(diu.GetId())
	if err != nil {
		return &res, errInvalidID
	}

	diskInst, err := disk.GetByID(diskUUID.String())
	if err != nil {
		return &res, fmt.Errorf("error getting disk: %w", err)
	}

	if diu.Description != nil {
		diskInst.Description = diu.GetDescription()
		slog.Debug("SetDiskInfo", "description", diskInst.Description)
	}

	if diu.DiskType != nil {
		diskInst.Type, err = mapDiskTypeTypeToDBString(diu.GetDiskType())
		if err != nil {
			return &res, fmt.Errorf("error: %w", err)
		}
	}

	if diu.DiskDevType != nil {
		diskInst.DevType, err = mapDiskDevTypeTypeToDBString(diu.GetDiskDevType())
		if err != nil {
			return &res, fmt.Errorf("error: %w", err)
		}
	}

	if diu.Cache != nil {
		diskInst.DiskCache = sql.NullBool{Bool: diu.GetCache(), Valid: true}
	}

	if diu.Direct != nil {
		diskInst.DiskDirect = sql.NullBool{Bool: diu.GetDirect(), Valid: true}
	}

	slog.Debug("SetDiskInfo saving disk")

	err = diskInst.Save()
	if err != nil {
		return &res, errDiskUpdateGeneric
	}

	res.Success = true

	return &res, nil
}

func (s *server) GetDisks(_ *cirrina.DisksQuery, stream cirrina.VMInfo_GetDisksServer) error {
	var diskID cirrina.DiskId
	for _, diskInst := range disk.List.DiskList {
		diskID.Value = diskInst.ID

		err := stream.Send(&diskID)
		if err != nil {
			return fmt.Errorf("error writing to stream: %w", err)
		}
	}

	return nil
}

var osOpenFileFunc = os.OpenFile
var osCreateFunc = os.Create

func (s *server) UploadDisk(stream cirrina.VMInfo_UploadDiskServer) error {
	var diskFile *os.File

	req, err := stream.Recv()
	if err != nil || req == nil {
		return fmt.Errorf("unable to receive from stream: %w", err)
	}

	diskUploadReq := req.GetDiskuploadinfo()

	diskInst, err := validateDiskReq(diskUploadReq)
	if err != nil {
		err = stream.SendAndClose(&cirrina.ReqBool{Success: false})
		if err != nil {
			return fmt.Errorf("failed validating disk upload request: %w", err)
		}

		return nil
	}

	diskPath := diskInst.GetPath()

	switch diskInst.DevType {
	case "ZVOL":
		diskPath = filepath.Join("/dev/zvol/", diskPath)

		diskFile, err = osOpenFileFunc(diskPath, os.O_RDWR, 0644)
		if err != nil {
			slog.Error("Failed to open disk file", "err", err.Error())

			err = stream.SendAndClose(&cirrina.ReqBool{Success: false})
			if err != nil {
				return fmt.Errorf("failed to open disk: %w", err)
			}

			return nil
		}
	case "FILE":
		diskFile, err = osCreateFunc(diskPath)
		if err != nil {
			slog.Error("Failed to creating disk file", "err", err.Error())

			err = stream.SendAndClose(&cirrina.ReqBool{Success: false})
			if err != nil {
				return fmt.Errorf("failed to open disk: %w", err)
			}

			return nil
		}
	default:
		slog.Error("request invalid, bad disk type")

		err = stream.SendAndClose(&cirrina.ReqBool{Success: false})
		if err != nil {
			return fmt.Errorf("error returning status: %w", err)
		}

		return nil
	}

	defer diskInst.Unlock()
	diskInst.Lock()

	err = receiveDiskFile(stream, diskUploadReq, diskFile)
	if err != nil {
		slog.Error("error during disk upload", "err", err)

		err = stream.SendAndClose(&cirrina.ReqBool{Success: false})
		if err != nil {
			return fmt.Errorf("error during disk upload: %w", err)
		}

		return nil
	}

	err = stream.SendAndClose(&cirrina.ReqBool{Success: true})
	if err != nil {
		return fmt.Errorf("error returning status: %w", err)
	}

	return nil
}

func validateGetDiskInfoRequest(diskID *cirrina.DiskId) (*disk.Disk, error) {
	diskUUID, err := uuid.Parse(diskID.GetValue())
	if err != nil {
		return nil, errInvalidID
	}

	diskInst, err := disk.GetByID(diskUUID.String())
	if err != nil {
		slog.Error("error getting disk", "disk", diskID.GetValue(), "err", err)

		return nil, errNotFound
	}

	if diskInst.Name == "" {
		slog.Debug("disk not found")

		return nil, errNotFound
	}

	return diskInst, nil
}

func getDiskVM(diskUUID uuid.UUID) (*vm.VM, error) {
	allVMs := vm.GetAll()
	found := false

	var aVM *vm.VM

	for _, thisVM := range allVMs {
		for _, vmDisk := range thisVM.Disks {
			if vmDisk == nil {
				continue
			}

			if vmDisk.ID == diskUUID.String() {
				if found {
					slog.Error("getDiskVm disk in use by more than one VM",
						"diskUUID", diskUUID,
						"vmid", thisVM.ID,
					)

					return nil, errDiskUsedByTwo
				}

				found = true
				aVM = thisVM
			}
		}
	}

	return aVM, nil
}

func validateDiskReq(diskUploadReq *cirrina.DiskUploadInfo) (*disk.Disk, error) {
	if diskUploadReq == nil || diskUploadReq.GetDiskid() == nil {
		slog.Error("nil diskUploadReq or disk id")

		return nil, errInvalidRequest
	}

	diskUUID, err := uuid.Parse(diskUploadReq.GetDiskid().GetValue())
	if err != nil {
		slog.Error("disk id not specified or invalid on upload")

		return nil, errInvalidID
	}

	diskInst, err := disk.GetByID(diskUUID.String())
	if err != nil {
		slog.Error("error getting disk", "id", diskUUID.String(), "err", err)

		return nil, errNotFound
	}

	if diskInst.Name == "" {
		slog.Debug("disk not found")

		return nil, errNotFound
	}

	var diskVM *vm.VM

	diskVM, err = getDiskVM(diskUUID)
	if err != nil {
		slog.Error("error getting disk VM", "err", err)

		return nil, err
	}

	if diskVM != nil {
		if diskVM.Status != "STOPPED" {
			slog.Error("can not upload disk to VM that is not stopped")

			return nil, errInvalidVMStateDiskUpload
		}
	}

	// not technically "validation" per se, but it needs to be done
	if diskInst.DevType == "ZVOL" {
		diskService := disk.NewZfsVolInfoService(nil)

		err = diskService.SetSize(config.Config.Disk.VM.Path.Zpool+"/"+diskInst.Name, diskUploadReq.GetSize())
		if err != nil {
			slog.Error("UploadDisk", "msg", "failed setting new volume size", "err", err)

			return nil, fmt.Errorf("error setting vol size: %w", err)
		}
	}

	return diskInst, nil
}

func receiveDiskFile(stream cirrina.VMInfo_UploadDiskServer, diskUploadReq *cirrina.DiskUploadInfo, diskFile *os.File,
) error {
	var err error

	var imageSize uint64

	diskFileBuffer := bufio.NewWriter(diskFile)

	hasher := sha512.New()

	for {
		var req *cirrina.DiskImageRequest

		req, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return fmt.Errorf("failed receiving from stream: %w", err)
		}

		chunk := req.GetImage()
		imageSize += uint64(len(chunk))

		if imageSize > diskUploadReq.GetSize() {
			return errDiskSizeFailure
		}

		_, err = diskFileBuffer.Write(chunk)
		if err != nil {
			return fmt.Errorf("failed writing to disk: %w", err)
		}

		hasher.Write(chunk)
	}

	diskChecksum := hex.EncodeToString(hasher.Sum(nil))

	// flush buffer
	err = diskFileBuffer.Flush()
	if err != nil {
		slog.Error("UploadDisk cannot send response", "err", err)

		return fmt.Errorf("failed flushing disk: %w", err)
	}

	// verify size
	if imageSize != diskUploadReq.GetSize() {
		return errDiskSizeFailure
	}

	// verify checksum
	if diskChecksum != diskUploadReq.GetSha512Sum() {
		return errDiskChecksumFailure
	}

	// finish saving file
	err = diskFile.Close()
	if err != nil {
		slog.Error("error closing disk file", "err", err)

		return fmt.Errorf("failed closing disk: %w", err)
	}

	return nil
}

func getDiskInfo(diskService disk.InfoServicer, diskInst *disk.Disk, diskInfo *cirrina.DiskInfo) error {
	diskPath := diskInst.GetPath()

	diskSizeNum, err := diskService.GetSize(diskPath)
	if err != nil {
		slog.Error("GetDiskInfoFile error getting disk size", "err", err)

		return fmt.Errorf("error stating disk path: %w", err)
	}

	diskUsageNum, err := diskService.GetUsage(diskPath)
	if err != nil {
		slog.Error("GetDiskInfoFile error getting disk usage", "err", err)

		return fmt.Errorf("error stating disk file: %w", err)
	}

	diskSize := strconv.FormatUint(diskSizeNum, 10)
	diskBlocks := strconv.FormatUint(diskUsageNum, 10)

	diskInfo.Size = &diskSize
	diskInfo.SizeNum = &diskSizeNum
	diskInfo.Usage = &diskBlocks
	diskInfo.UsageNum = &diskUsageNum

	return nil
}
