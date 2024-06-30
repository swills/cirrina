package vm

import (
	"log/slog"

	"cirrina/cirrinad/iso"
)

func (vm *VM) AttachIsos(isos []iso.ISO) error {
	defer vm.mu.Unlock()
	vm.mu.Lock()

	if vm.Status != STOPPED {
		return errVMNotStopped
	}

	vm.ISOs = isos

	err := vm.Save()
	if err != nil {
		slog.Error("error saving VM", "err", err)

		return err
	}

	return nil
}
