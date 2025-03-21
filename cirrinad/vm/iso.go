package vm

import (
	"log/slog"

	"cirrina/cirrinad/iso"
)

func (v *VM) AttachIsos(isos []*iso.ISO) error {
	defer v.mu.Unlock()
	v.mu.Lock()

	if v.Status != STOPPED {
		return errVMNotStopped
	}

	v.ISOs = isos

	err := v.Save()
	if err != nil {
		slog.Error("error saving VM", "err", err)

		return err
	}

	return nil
}
