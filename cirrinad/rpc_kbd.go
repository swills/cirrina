package main

import (
	"bufio"
	"cirrina/cirrina"
	"cirrina/cirrinad/util"
	"fmt"
	"os"
	"strings"
)

const kbdlayoutpath = "/usr/share/bhyve/kbdlayout"

func (s *server) GetKeyboardLayouts(_ *cirrina.KbdQuery, stream cirrina.VMInfo_GetKeyboardLayoutsServer) error {
	util.Trace()
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
	util.Trace()
	// ignore errors and just return empty list if err
	kbdlayouts, _ = util.OSReadDir(kbdlayoutpath)
	return kbdlayouts
}

func GetKbdDescription(path string) (description string, err error) {
	util.Trace()
	file, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
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
		fmt.Println(err)
	}

	return description, nil
}
