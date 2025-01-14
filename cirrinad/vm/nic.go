package vm

import (
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	vmswitch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/vmnic"
)

func (vm *VM) netStart() error {
	vmNicsList, err := vmnic.GetNics(vm.Config.ID)
	if err != nil {
		slog.Error("netStart failed to get nics", "err", err)

		return fmt.Errorf("error getting vm nics: %w", err)
	}

	for _, vmNic := range vmNicsList {
		err = vmNic.Build()
		if err != nil {
			slog.Error("error creating nic", "err", err)

			continue
		}

		// nothing to do
		if vmNic.SwitchID == "" {
			continue
		}

		// get switch
		thisSwitch, err := vmswitch.GetByID(vmNic.SwitchID)
		if err != nil {
			slog.Error("bad switch id",
				"err", err,
				"nic.Name", vmNic.Name,
				"nic.ID", vmNic.ID,
				"switch.ID", vmNic.SwitchID,
			)

			continue
		}

		// connect nic to switch
		err = thisSwitch.ConnectNic(&vmNic)
		if err != nil {
			slog.Error("error connecting switch",
				"err", err,
				"nic.Name", vmNic.Name,
				"nic.ID", vmNic.ID,
				"switch.ID", vmNic.SwitchID,
			)

			continue
		}
	}

	return nil
}

// validateNics check if nics can be attached to a VM
func (vm *VM) validateNics(nicIDs []string) error {
	occurred := map[string]bool{}

	for _, aNic := range nicIDs {
		nicUUID, err := uuid.Parse(aNic)
		if err != nil {
			return fmt.Errorf("nic invalid: %w", err)
		}

		thisNic, err := vmnic.GetByID(nicUUID.String())
		if err != nil {
			slog.Error("error getting nic", "nic", aNic, "err", err)

			return fmt.Errorf("nic not found: %w", err)
		}

		err = thisNic.Validate()
		if err != nil {
			return fmt.Errorf("nic invalid: %w", err)
		}

		if !occurred[aNic] {
			occurred[aNic] = true
		} else {
			slog.Error("duplicate nic id", "nic", aNic)

			return errVMNicDupe
		}

		err = vm.nicAttached(aNic)
		if err != nil {
			return err
		}
	}

	return nil
}

// nicAttached check if nic is attached to another VM besides this one
func (vm *VM) nicAttached(aNic string) error {
	allVms := GetAll()
	for _, aVM := range allVms {
		vmNics, err := vmnic.GetNics(aVM.Config.ID)
		if err != nil {
			slog.Error("error looking up nics", "err", err)

			return fmt.Errorf("error getting vm nics: %w", err)
		}

		for _, aVMNic := range vmNics {
			if aNic == aVMNic.ID && aVM.ID != vm.ID {
				slog.Error("nic is already attached to VM", "disk", aNic, "vm", aVM.ID)

				return errVMNicAttached
			}
		}
	}

	return nil
}

// removeAllNicsFromVM removes all nics from a VM
func (vm *VM) removeAllNicsFromVM() error {
	thisVMNics, err := vmnic.GetNics(vm.Config.ID)
	if err != nil {
		slog.Error("error looking up nics", "err", err)

		return fmt.Errorf("error getting vm nics: %w", err)
	}

	for _, aNic := range thisVMNics {
		aNic.ConfigID = 0

		err := aNic.Save()
		if err != nil {
			slog.Error("error saving NIC", "err", err)

			return fmt.Errorf("error saving NIC: %w", err)
		}
	}

	return nil
}

// NetStop clean up all of a VMs nics
func (vm *VM) NetStop() {
	vmNicsList, err := vmnic.GetNics(vm.Config.ID)
	if err != nil {
		slog.Error("failed to get nics", "err", err)

		return
	}

	for _, vmNic := range vmNicsList {
		// disconnect nic from switch
		if vmNic.SwitchID != "" {
			thisSwitch, err := vmswitch.GetByID(vmNic.SwitchID)
			if err != nil || thisSwitch == nil {
				slog.Error("bad switch id",
					"nic.Name", vmNic.Name, "nic.ID", vmNic.ID, "switch.ID", vmNic.SwitchID)
			} else {
				err = thisSwitch.DisconnectNic(&vmNic)
				if err != nil {
					slog.Error("error disconnecting switch",
						"err", err,
						"nic.Name", vmNic.Name,
						"nic.ID", vmNic.ID,
						"switch.ID", vmNic.SwitchID,
					)
				}
			}
		}

		// destroy interface
		err := vmNic.Demolish()
		if err != nil {
			slog.Error("error destroying nic", "err", err)
		}
	}
}

// SetNics sets the list of nics attached to a VM to the list passed in
func (vm *VM) SetNics(nicIDs []string) error {
	defer vm.mu.Unlock()
	vm.mu.Lock()
	if vm.Status != STOPPED {
		return errVMNotStopped
	}

	// remove all nics from VM
	err := vm.removeAllNicsFromVM()
	if err != nil {
		return err
	}

	// check that these nics can be attached to this VM
	err = vm.validateNics(nicIDs)
	if err != nil {
		return err
	}

	// add the nics
	for _, nicID := range nicIDs {
		vmNic, err := vmnic.GetByID(nicID)
		if err != nil {
			slog.Error("error looking up nic", "err", err)

			return fmt.Errorf("error getting NIC: %w", err)
		}

		vmNic.ConfigID = vm.Config.ID

		err = vmNic.Save()
		if err != nil {
			slog.Error("error saving NIC", "err", err)

			return fmt.Errorf("error saving NIC: %w", err)
		}
	}

	return nil
}
