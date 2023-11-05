package disk

import (
	"bufio"
	"errors"
	"golang.org/x/exp/slog"
	exec "golang.org/x/sys/execabs"
	"strconv"
	"strings"
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
