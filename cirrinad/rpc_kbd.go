package main

import (
	"bufio"
	"cirrina/cirrina"
	"cirrina/cirrinad/util"
	"fmt"
	"os"
	"strings"
)

func (s *server) GetKeyboardLayouts(_ *cirrina.KbdQuery, stream cirrina.VMInfo_GetKeyboardLayoutsServer) error {
	var kbdlayoutpath = "/usr/share/bhyve/kbdlayout"
	var layout cirrina.KbdLayout

	files, err := util.OSReadDir(kbdlayoutpath)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		layout.Name = file
		if file == "default" {
			layout.Description = "default"
		} else {
			layout.Description, err = getKbdDescription(kbdlayoutpath + "/" + file)
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

func getKbdDescription(path string) (description string, err error) {
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
