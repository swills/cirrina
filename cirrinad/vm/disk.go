package vm

import (
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"cirrina/cirrinad/disk"
)

func (v *VM) lockDisks() {
	for _, vmDisk := range v.Disks {
		if vmDisk == nil {
			continue
		}

		vmDisk.Lock()
	}
}

func (v *VM) unlockDisks() {
	for _, vmDisk := range v.Disks {
		if vmDisk == nil {
			continue
		}
		vmDisk.Unlock()
	}
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

		if diskAttached(aDisk, thisVM) {
			return errVMDiskAttached
		}
	}

	return nil
}

// diskAttached check if disk is attached to another VM besides this one
func diskAttached(aDisk string, thisVM *VM) bool {
	allVms := GetAll()
	for _, aVM := range allVms {
		for _, aVMDisk := range aVM.Disks {
			if aVMDisk == nil {
				continue
			}

			if aDisk == aVMDisk.ID && aVM.ID != thisVM.ID {
				return true
			}
		}
	}

	return false
}

func (v *VM) AttachDisks(diskids []string) error {
	defer v.mu.Unlock()
	v.mu.Lock()
	if v.Status != STOPPED {
		return errVMNotStopped
	}

	err := validateDisks(diskids, v)
	if err != nil {
		return err
	}

	v.Disks = []*disk.Disk{}

	for _, diskID := range diskids {
		var aDisk *disk.Disk

		aDisk, err = disk.GetByID(diskID)
		if err != nil {
			return fmt.Errorf("error attaching disk: %w", err)
		}

		v.Disks = append(v.Disks, aDisk)
	}

	err = v.Save()
	if err != nil {
		slog.Error("error saving VM", "err", err)

		return err
	}

	return nil
}
