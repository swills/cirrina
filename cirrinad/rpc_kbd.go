package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"cirrina/cirrina"
	"cirrina/cirrinad/util"
)

const kbdlayoutpath = "/usr/share/bhyve/kbdlayout"

func (s *server) GetKeyboardLayouts(_ *cirrina.KbdQuery, stream cirrina.VMInfo_GetKeyboardLayoutsServer) error {
	var layout cirrina.KbdLayout

	var err error

	files := GetKbdLayoutNames()
	for _, file := range files {
		layout.Name = file
		if file == "default" {
			layout.Description = "default"
		} else {
			layout.Description, err = GetKbdDescription(kbdlayoutpath + "/" + file)
			if err != nil {
				return err
			}
		}

		err = stream.Send(&layout)
		if err != nil {
			return fmt.Errorf("error sending to stream: %w", err)
		}
	}

	return nil
}

func GetKbdLayoutNames() []string {
	var kbdLayouts []string
	// ignore errors and just return empty list if err
	kbdLayouts, _ = util.OSReadDir(kbdlayoutpath)

	return kbdLayouts
}

func GetKbdDescription(path string) (string, error) {
	var description string

	var err error

	file, err := os.Open(path)
	if err != nil {
		slog.Error("error opening keyboard description dir", "err", err)

		return "", fmt.Errorf("error opening keyboard file: %w", err)
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	lineNo := 0
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lineNo++
		if lineNo > 2 {
			continue
		}

		if lineNo == 2 {
			de := strings.Split(scanner.Text(), ":")
			if len(de) > 1 {
				desc := strings.TrimSpace(de[1])
				description = strings.TrimSuffix(desc, ")")
			} else {
				description = "unknown"
			}
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("error scanning keyboard description dir", "err", err)

		return "", fmt.Errorf("error parsing keyboard description: %w", err)
	}

	return description, nil
}
