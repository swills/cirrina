//go:generate mockgen -destination=zfs_mocks.go -package=disk . ZfsVolInfoFetcher
package disk

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

type ZfsVolInfoFetcher interface {
	CheckExists(name string) (bool, error)
	Add(name string, size uint64) error
	FetchZfsVolumeSize(volumeName string) (uint64, error)
	FetchZfsVolBlockSize(volumeName string) (uint64, error)
	FetchZfsVolumeUsage(volumeName string) (uint64, error)
	ApplyZfsVolumeSize(volumeName string, volSize uint64) error
	FetchAll() ([]string, error)
}

type ZfsVolInfoCmds struct{}

type ZfsVolService struct {
	ZvolInfoImpl ZfsVolInfoFetcher
}

var fetchAllFunc = ZfsVolInfoCmds.FetchAll

func NewZfsVolInfoService(impl ZfsVolInfoFetcher) ZfsVolService {
	if impl == nil {
		impl = &ZfsVolInfoCmds{}
	}

	d := ZfsVolService{
		ZvolInfoImpl: impl,
	}

	return d
}

func (n ZfsVolService) GetSize(volumeName string) (uint64, error) {
	volSize, err := n.ZvolInfoImpl.FetchZfsVolumeSize(volumeName)
	if err != nil {
		return 0, fmt.Errorf("error getting volume size: %w", err)
	}

	return volSize, nil
}

func (n ZfsVolService) GetUsage(volumeName string) (uint64, error) {
	volSize, err := n.ZvolInfoImpl.FetchZfsVolumeUsage(volumeName)
	if err != nil {
		return 0, fmt.Errorf("error getting volume usage: %w", err)
	}

	return volSize, nil
}

func (n ZfsVolService) SetSize(volumeName string, volSize uint64) error {
	var err error

	var currentVolSize uint64

	currentVolSize, err = n.ZvolInfoImpl.FetchZfsVolumeSize(volumeName)
	if err != nil {
		slog.Error("SetSize", "msg", "failed getting current volume size", "err", err)

		return fmt.Errorf("failed getting zfs volume size: %w", err)
	}

	if volSize == currentVolSize {
		slog.Debug("SetSize requested vol size already set")

		return nil
	}

	// volsize must be a multiple of volume block size
	var vbs uint64

	vbs, err = n.ZvolInfoImpl.FetchZfsVolBlockSize(volumeName)
	if err != nil {
		slog.Error("error getting zfs vol block size", "err", err)

		return fmt.Errorf("failed getting zfs volume block size: %w", err)
	}

	// per zfsprops(7) -- "The volsize can only be set to
	//       a multiple of volblocksize, and cannot be zero."
	// so, if user asked for something not a multiple of volblocksize,
	// round up to a multiple of volblocksize

	// get modulus -- vbs can't be zero due to check for it in FetchZfsVolBlockSize, so no
	// need to worry about divide by zero
	mod := volSize % vbs

	// subtract modulus from volblocksize to get how much we need to increase the new vol size
	if mod > 0 {
		ads := vbs - mod
		volSize += ads
	}

	if volSize < currentVolSize {
		// maybe I don't care when uploading new disk image -- will care on disk expand, adjust this later, so
		// we can force it if the user accepts data loss
		slog.Error("SetSize", "error", "new disk smaller than current disk")

		return errDiskShrinkage
	}

	err = n.ZvolInfoImpl.ApplyZfsVolumeSize(volumeName, volSize)
	if err != nil {
		return fmt.Errorf("failed to set zfs volume size: %w", err)
	}

	return nil
}

func (n ZfsVolService) Exists(name string) (bool, error) {
	exists, err := n.ZvolInfoImpl.CheckExists(name)
	if err != nil {
		return true, fmt.Errorf("error checking file exists: %w", err)
	}

	return exists, nil
}

func (n ZfsVolService) Create(name string, size uint64) error {
	err := n.ZvolInfoImpl.Add(name, size)

	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}

	return nil
}

func (n ZfsVolService) GetAll() ([]string, error) {
	retVal, err := n.ZvolInfoImpl.FetchAll()

	if err != nil {
		return nil, fmt.Errorf("error creating file: %w", err)
	}

	return retVal, nil
}

func (n ZfsVolService) RemoveBacking(targetDisk *Disk) error {
	var err error

	volName := targetDisk.GetPath()

	hasSnapshot := hasEmptySnapshot(volName)

	if hasSnapshot {
		return rollBackToEmptySnapshot(volName)
	}

	var size uint64

	size, err = n.GetSize(volName)
	if err != nil {
		return err
	}

	err = destroyVol(volName)
	if err != nil {
		return err
	}

	err = n.Create(volName, size)
	if err != nil {
		return err
	}

	return nil
}

func (e ZfsVolInfoCmds) FetchZfsVolumeSize(volumeName string) (uint64, error) {
	var volSize uint64

	var err error

	found := false

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		"/sbin/zfs",
		[]string{"list", "-H", "-p", "-t", "volume", "-o", "volsize", volumeName},
	)

	if err != nil {
		slog.Error("failed to get zfs volume size",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return 0, fmt.Errorf("failed to get zfs volume size: %w", err)
	}

	var volSizeStr string

	for _, line := range strings.Split(string(stdOutBytes), "\n") {
		if len(line) == 0 {
			continue
		}

		textFields := strings.Fields(line)

		if len(textFields) != 1 {
			continue
		}

		volSizeStr = textFields[0]
		found = true
	}

	if !found {
		return 0, errDiskNotFound
	}

	volSize, err = strconv.ParseUint(volSizeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed parsing zfs output: %w", err)
	}

	return volSize, nil
}

func (e ZfsVolInfoCmds) FetchZfsVolumeUsage(volumeName string) (uint64, error) {
	var volSizeStr string

	var volUsage uint64

	found := false

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		"/sbin/zfs",
		[]string{"list", "-H", "-p", "-t", "volume", "-o", "refer", volumeName},
	)
	if err != nil {
		slog.Error("failed to get zfs volume usage",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)
	}

	for _, line := range strings.Split(string(stdOutBytes), "\n") {
		textFields := strings.Fields(line)
		if len(textFields) != 1 {
			continue
		}

		volSizeStr = textFields[0]
		found = true
	}

	if !found {
		return 0, errDiskNotFound
	}

	volUsage, err = strconv.ParseUint(volSizeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed parsing zfs output: %w", err)
	}

	return volUsage, nil
}

func (e ZfsVolInfoCmds) FetchZfsVolBlockSize(volumeName string) (uint64, error) {
	var volSizeStr string

	found := false

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		"/sbin/zfs",
		[]string{"get", "-H", "-p", "volblocksize", volumeName},
	)
	if err != nil {
		slog.Error("failed to get zfs volume block size",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return 0, fmt.Errorf("failed to get zfs volume block size: %w", err)
	}

	for _, line := range strings.Split(string(stdOutBytes), "\n") {
		if len(line) == 0 {
			continue
		}

		if found {
			return 0, errDiskDupe
		}

		textFields := strings.Fields(line)
		if len(textFields) != 4 {
			continue
		}

		volSizeStr = textFields[2]
		found = true
	}

	if !found {
		return 0, errDiskNotFound
	}

	var volBlockSize uint64

	volBlockSize, err = strconv.ParseUint(volSizeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed parsing zfs output: %w", err)
	}

	if volBlockSize == 0 {
		return 0, errDiskNotFound
	}

	return volBlockSize, nil
}

func (e ZfsVolInfoCmds) ApplyZfsVolumeSize(name string, newSize uint64) error {
	var err error

	volSizeStr := fmt.Sprintf("volsize=%d", newSize)
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/zfs", "set", volSizeStr, name},
	)

	if err != nil {
		slog.Error("failed to set zfs volume size",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("failed to apply zfs volume size: %w", err)
	}

	return nil
}

func (e ZfsVolInfoCmds) CheckExists(name string) (bool, error) {
	// for zvols, check both the volName and the volume name in zfs list
	allVolumes, err := fetchAllFunc(e)
	if err != nil {
		slog.Error("error checking if disk exists", "err", err)

		// assume disks exists if there's an error checking to be on safe side
		return true, fmt.Errorf("error checking if disk exists: %w", err)
	}

	if util.ContainsStr(allVolumes, name) {
		slog.Error("disk volume exists", "disk", name)

		return true, nil
	}

	diskExists, err := PathExistsFunc("/dev/zvol/" + name)
	if err != nil {
		slog.Error("error checking if disk exists", "name", name, "err", err)

		// assume disks exists if there's an error checking to be on safe side
		return true, fmt.Errorf("error checking if disk exists: %w", err)
	}

	if diskExists {
		slog.Error("disk vol path exists", "disk", name)

		return true, nil
	}

	return false, nil
}

func (e ZfsVolInfoCmds) Add(volName string, size uint64) error {
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/zfs", "create", "-o", "volmode=dev", "-V", strconv.FormatUint(size, 10), "-s", volName},
	)
	if err != nil {
		slog.Error("failed to create disk",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("error creating disk: %w", err)
	}

	// create snapshot with no data so that we can roll back to it later in RemoveBacking()
	stdOutBytes, stdErrBytes, returnCode, err = util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/zfs", "snapshot", volName + "@empty"},
	)
	if err != nil {
		slog.Error("failed to create disk",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("error creating disk: %w", err)
	}

	return nil
}

func (e ZfsVolInfoCmds) FetchAll() ([]string, error) {
	var err error

	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		"/sbin/zfs",
		[]string{"list", "-t", "volume", "-o", "name", "-H"},
	)

	if err != nil {
		slog.Error("failed to list zfs volumes",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return nil, fmt.Errorf("failed to list zfs volumes: %w", err)
	}

	volumesStrs := strings.Split(string(stdOutBytes), "\n")

	allVolumes := make([]string, 0, len(volumesStrs))

	for _, line := range volumesStrs {
		if len(line) == 0 {
			continue
		}

		textFields := strings.Fields(line)
		if len(textFields) != 1 {
			continue
		}

		allVolumes = append(allVolumes, textFields[0])
	}

	return allVolumes, nil
}

func destroyVol(volName string) error {
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/zfs", "destroy", "-r", volName},
	)

	if err != nil || returnCode != 0 {
		slog.Error("error destroying volume",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("error destroying volume: %w", err)
	}

	return nil
}

func hasEmptySnapshot(volName string) bool {
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/zfs", "list", "-H", "-t", "snapshot", volName + "@empty"},
	)

	stdOutFields := strings.Fields(string(stdOutBytes))

	if err == nil && returnCode == 0 && stdOutFields[0] == volName+"@empty" && len(stdErrBytes) == 0 {
		return true
	}

	return false
}

func rollBackToEmptySnapshot(volName string) error {
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/zfs", "rollback", "-r", volName + "@empty"},
	)

	if err != nil {
		slog.Error("failed to create disk",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("error creating disk: %w", err)
	}

	return nil
}

func checkLeftoversZvol() {
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		"/sbin/zfs",
		[]string{"list", "-t", "volume", "-o", "name", "-H"},
	)

	if err != nil {
		slog.Error("failed to list zfs volumes",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return
	}

	zvols := strings.Split(string(stdOutBytes), "\n")
	for _, aZVol := range zvols {
		// match GetPath()
		if !strings.HasPrefix(aZVol, config.Config.Disk.VM.Path.Zpool+"/") {
			continue
		}

		zvolName := strings.TrimPrefix(aZVol, config.Config.Disk.VM.Path.Zpool+"/")
		if util.ValidDiskName(zvolName) {
			_, err = GetByName(zvolName)
			if err != nil {
				slog.Warn("possible left over disk (zvol)", "disk.Name", zvolName, "vol.Name", zvolName)
			}
		}
	}
}
