package disk

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	exec "golang.org/x/sys/execabs"

	"cirrina/cirrinad/config"
)

func GetAllZfsVolumes() ([]string, error) {
	var allVolumes []string
	var err error
	cmd := exec.Command("/sbin/zfs", "list", "-t", "volume", "-o", "name", "-H")
	defer func(cmd *exec.Cmd) {
		err = cmd.Wait()
		if err != nil {
			slog.Error("zfs error", "err", err)
		}
	}(cmd)
	var stdout io.ReadCloser
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		return []string{}, fmt.Errorf("failed running zfs command: %w", err)
	}
	if err = cmd.Start(); err != nil {
		return []string{}, fmt.Errorf("failed running zfs command: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		text := scanner.Text()
		textFields := strings.Fields(text)
		if len(textFields) != 1 {
			continue
		}
		allVolumes = append(allVolumes, textFields[0])
	}
	if err = scanner.Err(); err != nil {
		return []string{}, fmt.Errorf("failed parsing zfs output: %w", err)
	}

	return allVolumes, nil
}

func GetZfsVolumeSize(volumeName string) (uint64, error) {
	var volSize uint64
	var err error
	found := false
	cmd := exec.Command("/sbin/zfs", "list", "-H", "-p", "-t", "volume", "-o", "volsize", volumeName)
	defer func(cmd *exec.Cmd) {
		err = cmd.Wait()
		if err != nil {
			slog.Error("zfs error", "err", err)
		}
	}(cmd)
	var stdout io.ReadCloser
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("failed running zfs command: %w", err)
	}
	if err = cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed running zfs command: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	var volSizeStr string
	for scanner.Scan() {
		text := scanner.Text()
		textFields := strings.Fields(text)
		if len(textFields) != 1 {
			continue
		}
		volSizeStr = textFields[0]
		found = true
	}
	if err = scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed parsing zfs output: %w", err)
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

func getZfsVolBlockSize(volumeName string) (uint64, error) {
	found := false
	cmd := exec.Command("/sbin/zfs", "get", "-H", "-p", "volblocksize", volumeName)
	defer func(cmd *exec.Cmd) {
		err := cmd.Wait()
		if err != nil {
			slog.Error("zfs error", "err", err)
		}
	}(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("failed running zfs command: %w", err)
	}
	if err = cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed running zfs command: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	var volSizeStr string
	for scanner.Scan() {
		if found {
			return 0, errDiskDupe
		}
		text := scanner.Text()
		textFields := strings.Fields(text)
		if len(textFields) != 4 {
			continue
		}
		volSizeStr = textFields[2]
		found = true
	}
	if err = scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed parsing zfs output: %w", err)
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
		// maybe I don't care when uploading new disk image -- will care on disk expand, adjust this later so
		// we can force it if the user accepts data loss
		slog.Error("SetZfsVolumeSize", "error", "new disk smaller than current disk")

		return errDiskShrinkage
	}

	volSizeStr := fmt.Sprintf("volsize=%d", volSize)
	args := []string{"/sbin/zfs", "set", volSizeStr, volumeName}
	slog.Debug("setting disk size", "volName", volumeName, "size", volSize)
	cmd := exec.Command(config.Config.Sys.Sudo, args...)
	err = cmd.Run()
	if err != nil {
		slog.Error("failed to set disk size", "err", err)

		return fmt.Errorf("failed running zfs command: %w", err)
	}

	return nil
}

func GetZfsVolumeUsage(volumeName string) (uint64, error) {
	var volUsage uint64
	var err error
	found := false
	cmd := exec.Command("/sbin/zfs", "list", "-H", "-p", "-t", "volume", "-o", "refer", volumeName)
	defer func(cmd *exec.Cmd) {
		err = cmd.Wait()
		if err != nil {
			slog.Error("zfs error", "err", err)
		}
	}(cmd)
	var stdout io.ReadCloser
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("failed running zfs command: %w", err)
	}
	if err = cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed running zfs command: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	var volSizeStr string
	for scanner.Scan() {
		text := scanner.Text()
		textFields := strings.Fields(text)
		if len(textFields) != 1 {
			continue
		}
		volSizeStr = textFields[0]
		found = true
	}
	if err = scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed parsing zfs output: %w", err)
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
