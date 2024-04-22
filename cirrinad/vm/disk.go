package vm

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"cirrina/cirrinad/disk"
)

func (vm *VM) lockDisks() error {
	vmDisks, err := vm.GetDisks()
	if err != nil {
		return err
	}
	for _, vmDisk := range vmDisks {
		vmDisk.Lock()
	}

	return nil
}

func (vm *VM) unlockDisks() error {
	vmDisks, err := vm.GetDisks()
	if err != nil {
		return err
	}
	for _, vmDisk := range vmDisks {
		vmDisk.Unlock()
	}

	return nil
}

func (vm *VM) GetDisks() ([]*disk.Disk, error) {
	var disks []*disk.Disk
	// TODO remove all these de-normalizations in favor of gorm native "Has Many" relationships
	for _, configValue := range strings.Split(vm.Config.Disks, ",") {
		if configValue == "" {
			continue
		}
		aDisk, err := disk.GetByID(configValue)
		if err == nil {
			disks = append(disks, aDisk)
		} else {
			slog.Error("bad disk", "disk", configValue, "vm", vm.ID)
		}
	}

	return disks, nil
}

func (vm *VM) AttachDisks(diskids []string) error {
	defer vm.mu.Unlock()
	vm.mu.Lock()
	if vm.Status != STOPPED {
		return errVMNotStopped
	}

	err := validateDisks(diskids, vm)
	if err != nil {
		return err
	}

	// build disk list string to put into DB
	var disksConfigVal string
	count := 0
	for _, diskID := range diskids {
		if count > 0 {
			disksConfigVal += ","
		}
		disksConfigVal += diskID
		count++
	}
	vm.Config.Disks = disksConfigVal
	err = vm.Save()
	if err != nil {
		slog.Error("error saving VM", "err", err)

		return err
	}

	return nil
}

// validateDisks check if disks can be attached to a VM
func validateDisks(diskids []string, thisVM *VM) error {
	occurred := map[string]bool{}

	for _, aDisk := range diskids {
		slog.Debug("checking disk exists", "disk", aDisk)

		diskUUID, err := uuid.Parse(aDisk)
		if err != nil {
			return errVMDiskInvalid
		}

		thisDisk, err := disk.GetByID(diskUUID.String())
		if err != nil {
			slog.Error("error getting disk", "disk", aDisk, "err", err)

			return fmt.Errorf("error getting disk: %w", err)
		}
		if thisDisk.Name == "" {
			return errVMDiskNotFound
		}

		if !occurred[aDisk] {
			occurred[aDisk] = true
		} else {
			slog.Error("duplicate disk id", "disk", aDisk)

			return errVMDiskDupe
		}

		slog.Debug("checking if disk is attached to another VM", "disk", aDisk)
		diskIsAttached, err := diskAttached(aDisk, thisVM)
		if err != nil {
			return err
		}
		if diskIsAttached {
			return errVMDiskAttached
		}
	}

	return nil
}

// diskAttached check if disk is attached to another VM besides this one
func diskAttached(aDisk string, thisVM *VM) (bool, error) {
	allVms := GetAll()
	for _, aVM := range allVms {
		vmDisks, err := aVM.GetDisks()
		if err != nil {
			return true, err
		}
		for _, aVMDisk := range vmDisks {
			if aDisk == aVMDisk.ID && aVM.ID != thisVM.ID {
				return true, nil
			}
		}
	}

	return false, nil
}
