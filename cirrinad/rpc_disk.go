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
	"strconv"
	"strings"
	"syscall"

	"github.com/google/uuid"

	"cirrina/cirrina"
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/disk"
	"cirrina/cirrinad/vm"
)

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

	diskInst, err := disk.Create(diskInfo.GetName(), diskInfo.GetDescription(), diskInfo.GetSize(), diskType,
		diskDevType, diskInfo.GetCache(), diskInfo.GetDirect())
	if err != nil {
		return &cirrina.DiskId{}, fmt.Errorf("error creating disk: %w", err)
	}
	if diskInst != nil && diskInst.ID != "" {
		return &cirrina.DiskId{Value: diskInst.ID}, nil
	}

	return &cirrina.DiskId{}, errDiskCreateGeneric
}

func (s *server) GetDiskInfo(_ context.Context, diskID *cirrina.DiskId) (*cirrina.DiskInfo, error) {
	var diskInfo cirrina.DiskInfo
	defaultDiskCache := true
	defaultDiskDirect := false

	diskUUID, err := uuid.Parse(diskID.GetValue())
	if err != nil {
		return &diskInfo, errInvalidID
	}
	diskInst, err := disk.GetByID(diskUUID.String())
	if err != nil {
		slog.Error("error getting disk", "disk", diskID.GetValue(), "err", err)

		return &diskInfo, errNotFound
	}

	if diskInst.Name == "" {
		slog.Debug("disk not found")

		return &diskInfo, errNotFound
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
		return &diskInfo, err
	}

	diskInfo.DiskDevType, err = mapDiskDevTypeDBStringToType(diskInst.DevType)
	if err != nil {
		return &diskInfo, err
	}

	switch diskInfo.GetDiskDevType() {
	case cirrina.DiskDevType_FILE:
		diskSize, diskBlocks, diskSizeNum, diskUsageNum, err := getDiskInfoFile(diskInst)
		if err != nil {
			return &diskInfo, err
		}
		diskInfo.Size = &diskSize
		diskInfo.SizeNum = &diskSizeNum
		diskInfo.Usage = &diskBlocks
		diskInfo.UsageNum = &diskUsageNum

		if strings.HasSuffix(diskInfo.GetName(), ".img") {
			*diskInfo.Name = strings.TrimSuffix(diskInfo.GetName(), ".img")
		}
	case cirrina.DiskDevType_ZVOL:
		if config.Config.Disk.VM.Path.Zpool == "" {
			return &cirrina.DiskInfo{}, errDiskZPoolNotConfigured
		}
		diskSize, diskBlocks, diskSizeNum, diskUsageNum, err := getDiskInfoZVOL(diskInst)
		if err != nil {
			return &diskInfo, err
		}

		diskInfo.Size = &diskSize
		diskInfo.SizeNum = &diskSizeNum
		diskInfo.Usage = &diskBlocks
		diskInfo.UsageNum = &diskUsageNum
	default:
		return &diskInfo, errDiskInvalidDevType
	}

	return &diskInfo, nil
}

func (s *server) RemoveDisk(_ context.Context, diskID *cirrina.DiskId) (*cirrina.ReqBool, error) {
	slog.Debug("deleting disk", "diskid", diskID.GetValue())
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

	// check that disk is not in use by a VM
	var diskVM *vm.VM
	diskVM, err = getDiskVM(diskUUID)
	if err != nil {
		return &res, err
	}

	if diskVM != nil {
		return &res, errDiskInUse
	}

	// prevent deleting disk while it's being uploaded etc
	defer diskInst.Unlock()
	diskInst.Lock()

	err = disk.Delete(diskInst.ID)
	if err != nil {
		slog.Error("error deleting disk", "err", err)

		return &res, errDiskDeleteGeneric
	}

	res.Success = true

	return &res, nil
}

func (s *server) GetDiskVM(_ context.Context, diskID *cirrina.DiskId) (*cirrina.VMID, error) {
	var err error
	var pvmID cirrina.VMID

	diskUUID, err := uuid.Parse(diskID.GetValue())
	if err != nil {
		return &pvmID, errInvalidID
	}

	var aVM *vm.VM
	aVM, err = getDiskVM(diskUUID)
	if err != nil {
		return &cirrina.VMID{}, err
	}

	if aVM != nil {
		pvmID.Value = aVM.ID
	}

	return &pvmID, nil
}

func getDiskVM(diskUUID uuid.UUID) (*vm.VM, error) {
	allVMs := vm.GetAll()
	found := false
	var aVM *vm.VM

	for _, thisVM := range allVMs {
		thisVMDisks, err := thisVM.GetDisks()
		if err != nil {
			return &vm.VM{}, fmt.Errorf("error getting disk: %w", err)
		}
		for _, vmDisk := range thisVMDisks {
			if vmDisk.ID == diskUUID.String() {
				if found {
					slog.Error("GetDiskVm disk in use by more than one VM",
						"diskUUID", diskUUID,
						"vmid", thisVM.ID,
					)

					return &vm.VM{}, errDiskInUse
				}
				found = true
				aVM = thisVM
			}
		}
	}

	return aVM, nil
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
		switch diu.GetDiskType() {
		case cirrina.DiskType_NVME:
			diskInst.Type = "NVME"
		case cirrina.DiskType_AHCIHD:
			diskInst.Type = "AHCI-HD"
		case cirrina.DiskType_VIRTIOBLK:
			diskInst.Type = "VIRTIO-BLK"
		default:
			return &res, errDiskInvalidType
		}
		slog.Debug("SetDiskInfo", "type", diskInst.Type)
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

func (s *server) UploadDisk(stream cirrina.VMInfo_UploadDiskServer) error {
	var res cirrina.ReqBool
	res.Success = false
	var imageSize uint64

	req, err := stream.Recv()
	if err != nil {
		slog.Error("cannot receive image info")
	}
	diskUploadReq := req.GetDiskuploadinfo()
	diskInst, err := validateDiskReq(diskUploadReq)
	if err != nil {
		return err
	}
	diskPath, err := diskInst.GetPath()
	if err != nil {
		slog.Error("error getting path", "err", err)

		return fmt.Errorf("failed getting disk path: %w", err)
	}

	defer diskInst.Unlock()
	diskInst.Lock()
	err = receiveDiskFile(stream, diskPath, imageSize, diskUploadReq)
	if err != nil {
		slog.Error("error during disk upload", "err", err)
		err2 := stream.SendAndClose(&res)
		if err2 != nil {
			slog.Error("UploadIso cannot send error response", "err", err, "err2", err2)

			return err
		}

		return err
	}

	// save to db
	err = diskInst.Save()
	if err != nil {
		slog.Error("UploadDisk", "msg", "Failed saving to db")
		err2 := stream.SendAndClose(&res)
		if err2 != nil {
			slog.Error("UploadDisk failed sending error response, ignoring", "err", err, "err2", err2)
		}

		return nil
	}
	// we're done, return success to client
	res.Success = true
	err = stream.SendAndClose(&res)
	if err != nil {
		slog.Error("UploadDisk cannot send response", "err", err)
	}

	return nil
}

func validateDiskReq(diskUploadReq *cirrina.DiskUploadInfo) (*disk.Disk, error) {
	if diskUploadReq == nil || diskUploadReq.GetDiskid() == nil {
		slog.Error("nil diskUploadReq or disk id")

		return &disk.Disk{}, errReqInvalid
	}
	diskUUID, err := uuid.Parse(diskUploadReq.GetDiskid().GetValue())
	if err != nil {
		slog.Error("disk id not specified or invalid on upload")

		return &disk.Disk{}, errInvalidID
	}
	diskInst, err := disk.GetByID(diskUUID.String())
	if err != nil {
		slog.Error("error getting disk", "id", diskUUID.String(), "err", err)

		return &disk.Disk{}, errNotFound
	}
	if diskInst.Name == "" {
		slog.Debug("disk not found")

		return &disk.Disk{}, errNotFound
	}
	var diskVM *vm.VM
	diskVM, err = getDiskVM(diskUUID)
	if err != nil {
		slog.Error("error getting disk VM", "err", err)

		return &disk.Disk{}, err
	}
	if diskVM != nil {
		if diskVM.Status != "STOPPED" {
			slog.Error("can not upload disk to VM that is not stopped")

			return &disk.Disk{}, errInvalidVMStateDiskUpload
		}
	}
	// not technically "validation" per se, but it needs to be done
	if diskInst.DevType == "ZVOL" {
		err = disk.SetZfsVolumeSize(config.Config.Disk.VM.Path.Zpool+"/"+diskInst.Name, diskUploadReq.GetSize())
		if err != nil {
			slog.Error("UploadDisk", "msg", "failed setting new volume size", "err", err)

			return &disk.Disk{}, fmt.Errorf("error setting vol size: %w", err)
		}
	}

	return diskInst, nil
}

func receiveDiskFile(stream cirrina.VMInfo_UploadDiskServer, diskPath string, imageSize uint64,
	diskUploadReq *cirrina.DiskUploadInfo,
) error {
	diskFile, err := os.Create(diskPath)
	if err != nil {
		slog.Error("Failed to open disk file", "err", err.Error())

		return fmt.Errorf("failed creating disk file: %w", err)
	}
	diskFileBuffer := bufio.NewWriter(diskFile)

	hasher := sha512.New()

	for {
		var req *cirrina.DiskImageRequest
		req, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			slog.Error("UploadDisk failed receiving", "err", err)

			return fmt.Errorf("failed receiving from stream: %w", err)
		}

		chunk := req.GetImage()
		size := len(chunk)

		imageSize += uint64(size)
		_, err = diskFileBuffer.Write(chunk)
		if err != nil {
			slog.Error("UploadDisk failed writing", "err", err)

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
		slog.Error("UploadDisk  upload size incorrect",
			"imageSize", imageSize,
			"diskUploadReq.Size", diskUploadReq.GetSize(),
		)

		return errDiskSizeFailure
	}

	// verify checksum
	if diskChecksum != diskUploadReq.GetSha512Sum() {
		slog.Debug("UploadDisk image upload checksum incorrect",
			"diskChecksum", diskChecksum,
			"diskUploadReq.Sha512Sum", diskUploadReq.GetSha512Sum(),
		)

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

func getDiskInfoZVOL(diskInst *disk.Disk) (string, string, uint64, uint64, error) {
	diskSizeNum, err := disk.GetZfsVolumeSize(config.Config.Disk.VM.Path.Zpool + "/" + diskInst.Name)
	if err != nil {
		slog.Error("getDiskInfoZVOL GetZfsVolumeSize error", "err", err)

		return "", "", 0, 0, fmt.Errorf("failed getting vol size: %w", err)
	}

	diskUsageNum, err := disk.GetZfsVolumeUsage(config.Config.Disk.VM.Path.Zpool + "/" + diskInst.Name)
	if err != nil {
		slog.Error("getDiskInfoZVOL GetZfsVolumeUsage error", "err", err)
		diskUsageNum = 0
	}

	diskBlocks := strconv.FormatInt(int64(diskUsageNum), 10)
	diskSize := strconv.FormatInt(int64(diskSizeNum), 10)

	return diskSize, diskBlocks, diskSizeNum, diskUsageNum, nil
}

func getDiskInfoFile(diskInst *disk.Disk) (string, string, uint64, uint64, error) {
	var stat syscall.Stat_t
	var blockSize int64 = 512
	diskPath, err := diskInst.GetPath()
	if err != nil {
		slog.Error("getDiskInfoFile error getting disk size", "err", err)

		return "", "", 0, 0, fmt.Errorf("failed getting disk path: %w", err)
	}

	diskFileStat, err := os.Stat(diskPath)
	if err != nil {
		slog.Error("getDiskInfoFile error getting disk size", "err", err)

		return "", "", 0, 0, fmt.Errorf("error stating disk path: %w", err)
	}

	err = syscall.Stat(diskPath, &stat)
	if err != nil {
		slog.Error("getDiskInfoFile unable to stat diskPath", "diskPath", diskPath)

		return "", "", 0, 0, fmt.Errorf("error stating disk file: %w", err)
	}

	diskSize := strconv.FormatInt(diskFileStat.Size(), 10)
	diskBlocks := strconv.FormatInt(stat.Blocks*blockSize, 10)
	diskSizeNum := uint64(diskFileStat.Size())
	diskUsageNum := uint64(stat.Blocks * blockSize)

	return diskSize, diskBlocks, diskSizeNum, diskUsageNum, nil
}
