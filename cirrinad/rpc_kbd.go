package main

import (
	"bufio"
	"cirrina/cirrina"
	"cirrina/cirrinad/util"
	"log/slog"
	"os"
	"strings"
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
			return err
		}
	}
	return nil
}

func GetKbdLayoutNames() (kbdlayouts []string) {
	// ignore errors and just return empty list if err
	kbdlayouts, _ = util.OSReadDir(kbdlayoutpath)
	return kbdlayouts
}

func GetKbdDescription(path string) (description string, err error) {
	file, err := os.Open(path)
	if err != nil {
		slog.Error("error opening keyboard description dir", "err", err)
		return "", err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	lineNo := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineNo += 1
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
		return "", err
	}

	return description, nil
}
