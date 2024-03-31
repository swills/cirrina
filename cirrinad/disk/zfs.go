package disk

import (
	"bufio"
	"cirrina/cirrinad/config"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	exec "golang.org/x/sys/execabs"
)

func GetAllZfsVolumes() (allVolumes []string, err error) {
	cmd := exec.Command("zfs", "list", "-t", "volume", "-o", "name", "-H")
	defer func(cmd *exec.Cmd) {
		err := cmd.Wait()
		if err != nil {
			slog.Error("zfs error", "err", err)
		}
	}(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return []string{}, err
	}
	if err := cmd.Start(); err != nil {
		return []string{}, err
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
	if err := scanner.Err(); err != nil {
		return []string{}, err
	}
	return allVolumes, nil
}

func GetZfsVolumeSize(volumeName string) (volSize uint64, err error) {
	found := false
	cmd := exec.Command("zfs", "list", "-H", "-p", "-t", "volume", "-o", "volsize", volumeName)
	defer func(cmd *exec.Cmd) {
		err := cmd.Wait()
		if err != nil {
			slog.Error("zfs error", "err", err)
		}
	}(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	if err = cmd.Start(); err != nil {
		return 0, err
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
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	if !found {
		return 0, errors.New("not found")
	}
	volSize, err = strconv.ParseUint(volSizeStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return volSize, nil
}

func getZfsVolBlockSize(volumeName string) (uint64, error) {
	found := false
	cmd := exec.Command("zfs", "get", "-H", "-p", "volblocksize", volumeName)
	defer func(cmd *exec.Cmd) {
		err := cmd.Wait()
		if err != nil {
			slog.Error("zfs error", "err", err)
		}
	}(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	if err = cmd.Start(); err != nil {
		return 0, err
	}
	scanner := bufio.NewScanner(stdout)
	var volSizeStr string
	for scanner.Scan() {
		if found {
			return 0, errors.New("duplicate disk found")
		}
		text := scanner.Text()
		textFields := strings.Fields(text)
		if len(textFields) != 4 {
			continue
		}
		volSizeStr = textFields[2]
		found = true
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	if !found {
		return 0, errors.New("not found")
	}
	var volBlockSize uint64
	volBlockSize, err = strconv.ParseUint(volSizeStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return volBlockSize, nil
}

func SetZfsVolumeSize(volumeName string, volSize uint64) error {
	var err error

	var currentVolSize uint64
	currentVolSize, err = GetZfsVolumeSize(volumeName)
	if err != nil {
		slog.Error("SetZfsVolumeSize", "msg", "failed getting current volume size", "err", err)
		return err
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
		return err
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
		return errors.New("new disk smaller than current disk")
	}

	volSizeStr := fmt.Sprintf("volsize=%d", volSize)
	args := []string{"zfs", "set", volSizeStr, volumeName}
	slog.Debug("setting disk size", "volName", volumeName, "size", volSize)
	cmd := exec.Command(config.Config.Sys.Sudo, args...)
	err = cmd.Run()
	if err != nil {
		slog.Error("failed to set disk size", "err", err)
		return err
	}
	return nil
}

func GetZfsVolumeUsage(volumeName string) (volUsage uint64, err error) {
	found := false
	cmd := exec.Command("zfs", "list", "-H", "-p", "-t", "volume", "-o", "refer", volumeName)
	defer func(cmd *exec.Cmd) {
		err := cmd.Wait()
		if err != nil {
			slog.Error("zfs error", "err", err)
		}
	}(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	if err = cmd.Start(); err != nil {
		return 0, err
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
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	if !found {
		return 0, errors.New("not found")
	}
	volUsage, err = strconv.ParseUint(volSizeStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return volUsage, nil
}
