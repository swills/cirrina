package vm

import (
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"cirrina/cirrinad/iso"
)

func (vm *VM) GetISOs() ([]iso.ISO, error) {
	var isos []iso.ISO
	// TODO remove all these de-normalizations in favor of gorm native "Has Many" relationships
	for _, configValue := range strings.Split(vm.Config.ISOs, ",") {
		if configValue == "" {
			continue
		}

		aISO, err := iso.GetByID(configValue)
		if err == nil {
			isos = append(isos, *aISO)
		} else {
			slog.Error("bad iso", "iso", configValue, "vm", vm.ID)
		}
	}

	return isos, nil
}

func (vm *VM) AttachIsos(isoIDs []string) error {
	defer vm.mu.Unlock()
	vm.mu.Lock()

	if vm.Status != STOPPED {
		return errVMNotStopped
	}

	for _, aIso := range isoIDs {
		slog.Debug("checking iso exists", "iso", aIso)

		isoUUID, err := uuid.Parse(aIso)
		if err != nil {
			return errVMIsoInvalid
		}

		thisIso, err := iso.GetByID(isoUUID.String())
		if err != nil {
			slog.Error("error getting disk", "disk", aIso, "err", err)

			return errVMIsoNotFound
		}

		if thisIso == nil || thisIso.Name == "" {
			return errVMIsoNotFound
		}
	}

	var isoConfigVal string

	count := 0

	for _, isoID := range isoIDs {
		if count > 0 {
			isoConfigVal += ","
		}

		isoConfigVal += isoID
		count++
	}

	vm.Config.ISOs = isoConfigVal

	err := vm.Save()
	if err != nil {
		slog.Error("error saving VM", "err", err)

		return err
	}

	return nil
}
