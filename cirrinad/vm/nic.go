package vm

import (
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/epair"
	vmswitch "cirrina/cirrinad/switch"
	"cirrina/cirrinad/util"
	"cirrina/cirrinad/vmnic"
)

func netStartupIf(vmNic vmnic.VMNic) error {
	// Create interface
	stdOutBytes, stdErrBytes, returnCode, err := util.RunCmd(
		config.Config.Sys.Sudo, []string{"/sbin/ifconfig", vmNic.NetDev, "create", "group", "cirrinad"},
	)
	if err != nil {
		slog.Error("failed to create tap",
			"stdOutBytes", stdOutBytes,
			"stdErrBytes", stdErrBytes,
			"returnCode", returnCode,
			"err", err,
		)

		return fmt.Errorf("error running ifconfig command: %w", err)
	}

	if vmNic.SwitchID == "" {
		return nil
	}

	// Add interface to bridge
	thisSwitch, err := vmswitch.GetByID(vmNic.SwitchID)
	if err != nil {
		slog.Error("bad switch id",
			"nicname", vmNic.Name, "nicid", vmNic.ID, "switchid", vmNic.SwitchID)

		return fmt.Errorf("error getting switch id: %w", err)
	}

	if thisSwitch.Type != "IF" {
		slog.Error("bridge/interface type mismatch",
			"nicname", vmNic.Name,
			"nicid", vmNic.ID,
			"switchid", vmNic.SwitchID,
		)

		return errSwitchNICMismatch
	}

	var thisMemberName string

	if vmNic.RateLimit {
		var thisEpair string

		thisEpair, err = vmswitch.SetupVMNicRateLimit(vmNic)
		if err != nil {
			return fmt.Errorf("failed setting up nic: %w", err)
		}

		thisMemberName = thisEpair + "b"
	} else {
		thisMemberName = vmNic.NetDev
	}

	err = vmswitch.SwitchIfAddMember(thisSwitch.Name, thisMemberName)
	if err != nil {
		slog.Error("failed to add nic to switch",
			"nicname", vmNic.Name,
			"nicid", vmNic.ID,
			"switchid", vmNic.SwitchID,
			"netdev", vmNic.NetDev,
			"err", err,
		)

		return fmt.Errorf("error adding member to bridge: %w", err)
	}

	return nil
}

func netStartupNg(vmNic vmnic.VMNic) error {
	thisSwitch, err := vmswitch.GetByID(vmNic.SwitchID)
	if err != nil {
		slog.Error("bad switch id",
			"nicname", vmNic.Name, "nicid", vmNic.ID, "switchid", vmNic.SwitchID)

		return fmt.Errorf("error getting switch ID: %w", err)
	}

	if thisSwitch.Type != "NG" {
		slog.Error("bridge/interface type mismatch",
			"nicname", vmNic.Name,
			"nicid", vmNic.ID,
			"switchid", vmNic.SwitchID,
		)

		return errSwitchNICMismatch
	}

	return nil
}

// cleanupIfNic cleanup tap/vmnet type nic
func cleanupIfNic(vmNic vmnic.VMNic) error {
	var stdOutBytes []byte

	var stdErrBytes []byte

	var returnCode int

	var err1, err2, err3, err4, err5 error

	if vmNic.NetDev != "" {
		stdOutBytes, stdErrBytes, returnCode, err1 = util.RunCmd(
			config.Config.Sys.Sudo, []string{"/sbin/ifconfig", vmNic.NetDev, "destroy"},
		)
		if err1 != nil {
			// don't return error just in case the other bits are populated
			slog.Error("failed to destroy network interface",
				"stdOutBytes", stdOutBytes,
				"stdErrBytes", stdErrBytes,
				"returnCode", returnCode,
				"err", err1,
			)
		}
	}

	if vmNic.InstEpair != "" {
		err2 = epair.DestroyEpair(vmNic.InstEpair)
		if err2 != nil {
			slog.Error("failed to destroy epair", "err", err2)
		}
	}

	if vmNic.InstBridge != "" {
		err3 = vmswitch.DestroyIfSwitch(vmNic.InstBridge, false)
		if err3 != nil {
			slog.Error("failed to destroy switch", "err", err3)
		}
	}
	// tap/vmnet nics may be connected to an epair which is connected
	// to a netgraph pipe for purposes for rate limiting
	if vmNic.InstEpair != "" {
		err4 = epair.NgDestroyPipe(vmNic.InstEpair + "a")
		if err4 != nil {
			slog.Error("failed to destroy ng pipe", "err", err4)
		}

		err5 = epair.NgDestroyPipe(vmNic.InstEpair + "b")
		if err5 != nil {
			slog.Error("failed to destroy ng pipe", "err", err5)
		}
	}

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
		return errVMNICCleanupError
	}

	return nil
}

func (vm *VM) netStartup() error {
	vmNicsList, err := vmnic.GetNics(vm.Config.ID)
	if err != nil {
		slog.Error("netStartup failed to get nics", "err", err)

		return fmt.Errorf("error getting vm nics: %w", err)
	}

	for _, vmNic := range vmNicsList {
		// silence gosec
		vmNic := vmNic

		switch vmNic.NetDevType {
		case "TAP":
			err := netStartupIf(vmNic)
			if err != nil {
				slog.Error("error bringing up nic", "err", err)

				return fmt.Errorf("error starting vm nic: %w", err)
			}
		case "VMNET":
			err := netStartupIf(vmNic)
			if err != nil {
				slog.Error("error bringing up nic", "err", err)

				return fmt.Errorf("error starting vm nic: %w", err)
			}
		case "NETGRAPH":
			err := netStartupNg(vmNic)
			if err != nil {
				slog.Error("error bringing up nic", "err", err)

				return fmt.Errorf("error starting vm nic: %w", err)
			}
		default:
			slog.Debug("unknown net type, can't set up")

			return errVMUnknownNetDevType
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

// NetCleanup clean up all of a VMs nics
func (vm *VM) NetCleanup() {
	vmNicsList, err := vmnic.GetNics(vm.Config.ID)
	if err != nil {
		slog.Error("failed to get nics", "err", err)

		return
	}

	for _, vmNic := range vmNicsList {
		switch {
		case vmNic.NetDevType == "TAP" || vmNic.NetDevType == "VMNET":
			err = cleanupIfNic(vmNic)
			if err != nil {
				slog.Error("error cleaning up nic", "vmNic", vmNic, "err", err)
			}
		case vmNic.NetDevType == "NETGRAPH":
			// nothing to do for netgraph
		default:
			slog.Error("unknown net type, can't clean up")
		}

		vmNic.NetDev = ""
		vmNic.InstEpair = ""
		vmNic.InstBridge = ""
		err = vmNic.Save()

		if err != nil {
			slog.Error("failed to save net dev", "nic", vmNic.ID, "netdev", vmNic.NetDev)
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
