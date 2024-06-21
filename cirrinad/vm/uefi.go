package vm

import (
	"fmt"
	"log/slog"
	"os"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

func (vm *VM) createUefiVarsFile() {
	uefiVarsFilePath := config.Config.Disk.VM.Path.State + "/" + vm.Name
	uefiVarsFile := uefiVarsFilePath + "/BHYVE_UEFI_VARS.fd"

	uvPathExists, err := util.PathExists(uefiVarsFilePath)
	if err != nil {
		return
	}

	if !uvPathExists {
		err = os.Mkdir(uefiVarsFilePath, 0o755)
		if err != nil {
			slog.Error("failed to create uefi vars path", "err", err)

			return
		}
	}

	uvFileExists, err := util.PathExists(uefiVarsFile)
	if err != nil {
		return
	}

	if !uvFileExists {
		_, err = util.CopyFile(config.Config.Rom.Vars.Template, uefiVarsFile)
		if err != nil {
			slog.Error("failed to copy uefiVars template", "err", err)
		}
	}
}

func (vm *VM) DeleteUEFIState() error {
	uefiVarsFilePath := config.Config.Disk.VM.Path.State + "/" + vm.Name
	uefiVarsFile := uefiVarsFilePath + "/BHYVE_UEFI_VARS.fd"

	uvFileExists, err := util.PathExists(uefiVarsFile)
	if err != nil {
		return fmt.Errorf("error checking if UEFI state file exists: %w", err)
	}

	if uvFileExists {
		err = os.Remove(uefiVarsFile)
		if err != nil {
			return fmt.Errorf("error removing UEFI state file: %w", err)
		}
	}

	return nil
}
