package disk

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

func GetAllZfsVolumes() ([]string, error) {
	var err error

	var allVolumes []string

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

	for _, line := range strings.Split(string(stdOutBytes), "\n") {
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

func GetZfsVolumeSize(volumeName string) (uint64, error) {
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

func SetZfsVolumeSize(volumeName string, volSize uint64) error {
	var err error

	var currentVolSize uint64

	currentVolSize, err = GetZfsVolumeSize(volumeName)
	if err != nil {
		slog.Error("SetZfsVolumeSize", "msg", "failed getting current volume size", "err", err)

		return fmt.Errorf("failed getting zfs volume size: %w", err)
	}

	if volSize == currentVolSize {
		slog.Debug("SetZfsVolumeSize requested vol size already set")

		return nil
	}

	// volsize must be a multiple of volume block size
	var vbs uint64

	vbs, err = getZfsVolBlockSize(volumeName)
	if err != nil {
		slog.Error("error getting zfs vol block size", "err", err)

		return fmt.Errorf("failed getting zfs volume block size: %w", err)
	}

	// get modulus
	mod := volSize % vbs

	// subtract modulus from volblocksize to get how much we need to increase the new vol size
	if mod > 0 {
		ads := vbs - mod
		volSize += ads
	}

	if volSize == currentVolSize {
		slog.Debug("SetZfsVolumeSize adjusted vol size already set")

		return nil
	}

	if volSize < currentVolSize {
		// maybe I don't care when uploading new disk image -- will care on disk expand, adjust this later, so
		// we can force it if the user accepts data loss
		slog.Error("SetZfsVolumeSize", "error", "new disk smaller than current disk")

		return errDiskShrinkage
	}

	volSizeStr := fmt.Sprintf("volsize=%d", volSize)
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo,
		[]string{"/sbin/zfs", "set", volSizeStr, volumeName},
	)

	if err != nil {
		slog.Error("failed to set zfs volume size",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("failed to set zfs volume size: %w", err)
	}

	return nil
}

func GetZfsVolumeUsage(volumeName string) (uint64, error) {
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

func getZfsVolBlockSize(volumeName string) (uint64, error) {
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

	return volBlockSize, nil
}
