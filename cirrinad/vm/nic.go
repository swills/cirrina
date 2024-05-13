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

		return errVMSwitchNICMismatch
	}

	var thisMemberName string

	if vmNic.RateLimit {
		var thisEpair string

		thisEpair, err = setupVMNicRateLimit(vmNic)
		if err != nil {
			return err
		}

		thisMemberName = thisEpair + "b"
	} else {
		thisMemberName = vmNic.NetDev
	}

	err = vmswitch.BridgeIfAddMember(thisSwitch.Name, thisMemberName)
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

func setupVMNicRateLimit(vmNic vmnic.VMNic) (string, error) {
	var err error

	thisEpair := epair.GetDummyEpairName()
	slog.Debug("netStartup rate limiting", "thisEpair", thisEpair)

	err = epair.CreateEpair(thisEpair)
	if err != nil {
		slog.Error("error creating epair", err)

		return "", fmt.Errorf("error creating epair: %w", err)
	}

	vmNic.InstEpair = thisEpair
	err = vmNic.Save()

	if err != nil {
		slog.Error("failed to save net dev", "nic", vmNic.ID, "netdev", vmNic.NetDev)

		return "", fmt.Errorf("error saving NIC: %w", err)
	}

	err = epair.SetRateLimit(thisEpair, vmNic.RateIn, vmNic.RateOut)
	if err != nil {
		slog.Error("failed to set epair rate limit", "epair", thisEpair)

		return "", fmt.Errorf("error setting rate limit: %w", err)
	}

	thisInstSwitch := vmswitch.GetDummyBridgeName()

	var bridgeMembers []string
	bridgeMembers = append(bridgeMembers, thisEpair+"a")
	bridgeMembers = append(bridgeMembers, vmNic.NetDev)

	err = vmswitch.CreateIfBridgeWithMembers(thisInstSwitch, bridgeMembers)
	if err != nil {
		slog.Error("failed to create switch",
			"nic", vmNic.ID,
			"thisInstSwitch", thisInstSwitch,
			"err", err,
		)

		return "", fmt.Errorf("error creating bridge: %w", err)
	}

	vmNic.InstBridge = thisInstSwitch
	err = vmNic.Save()

	if err != nil {
		slog.Error("failed to save net dev", "nic", vmNic.ID, "netdev", vmNic.NetDev)

		return "", fmt.Errorf("error saving NIC: %w", err)
	}

	return thisEpair, nil
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

		return errVMSwitchNICMismatch
	}

	return nil
}

func (vm *VM) netStartup() {
	vmNicsList, err := vm.GetNics()
	if err != nil {
		slog.Error("netStartup failed to get nics", "err", err)

		return
	}

	for _, vmNic := range vmNicsList {
		switch {
		case vmNic.NetDevType == "TAP" || vmNic.NetDevType == "VMNET":
			err := netStartupIf(vmNic)
			if err != nil {
				slog.Error("error bringing up nic", "err", err)

				continue
			}
		case vmNic.NetDevType == "NETGRAPH":
			err := netStartupNg(vmNic)
			if err != nil {
				slog.Error("error bringing up nic", "err", err)

				continue
			}
		default:
			slog.Debug("unknown net type, can't set up")

			continue
		}
	}
}

// NetCleanup clean up all of a VMs nics
func (vm *VM) NetCleanup() {
	vmNicsList, err := vm.GetNics()
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
	err := removeAllNicsFromVM(vm)
	if err != nil {
		return err
	}

	// check that these nics can be attached to this VM
	err = validateNics(nicIDs, vm)
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

// validateNics check if nics can be attached to a VM
func validateNics(nicIDs []string, thisVM *VM) error {
	occurred := map[string]bool{}

	for _, aNic := range nicIDs {
		slog.Debug("checking nic exists", "vmnic", aNic)

		nicUUID, err := uuid.Parse(aNic)
		if err != nil {
			return errVMNICInvalid
		}

		thisNic, err := vmnic.GetByID(nicUUID.String())
		if err != nil {
			slog.Error("error getting nic", "nic", aNic, "err", err)

			return fmt.Errorf("nic not found: %w", err)
		}

		if thisNic.Name == "" {
			return errVMNICNotFound
		}

		if !occurred[aNic] {
			occurred[aNic] = true
		} else {
			slog.Error("duplicate nic id", "nic", aNic)

			return errVMNicDupe
		}

		slog.Debug("checking if nic is already attached", "nic", aNic)

		err = nicAttached(aNic, thisVM)
		if err != nil {
			return err
		}
	}

	return nil
}

// nicAttached check if nic is attached to another VM besides this one
func nicAttached(aNic string, thisVM *VM) error {
	allVms := GetAll()
	for _, aVM := range allVms {
		vmNics, err := aVM.GetNics()
		if err != nil {
			return err
		}

		for _, aVMNic := range vmNics {
			if aNic == aVMNic.ID && aVM.ID != thisVM.ID {
				slog.Error("nic is already attached to VM", "disk", aNic, "vm", aVM.ID)

				return errVMNicAttached
			}
		}
	}

	return nil
}

// removeAllNicsFromVM does what it says on the tin, mate
func removeAllNicsFromVM(thisVM *VM) error {
	thisVMNics, err := thisVM.GetNics()
	if err != nil {
		slog.Error("error looking up nics", "err", err)

		return err
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

// cleanup tap/vmnet type nic
func cleanupIfNic(vmNic vmnic.VMNic) error {
	var stdOutBytes []byte

	var stdErrBytes []byte

	var returnCode int

	var err error

	if vmNic.NetDev != "" {
		stdOutBytes, stdErrBytes, returnCode, err = util.RunCmd(
			config.Config.Sys.Sudo, []string{"/sbin/ifconfig", vmNic.NetDev, "destroy"},
		)
		if err != nil {
			slog.Error("failed to destroy network interface",
				"stdOutBytes", stdOutBytes,
				"stdErrBytes", stdErrBytes,
				"returnCode", returnCode,
				"err", err,
			)
		}
	}

	if vmNic.InstEpair != "" {
		err = epair.DestroyEpair(vmNic.InstEpair)
		if err != nil {
			slog.Error("failed to destroy epair", err)
		}
	}

	if vmNic.InstBridge != "" {
		err = vmswitch.DestroyIfBridge(vmNic.InstBridge, false)
		if err != nil {
			slog.Error("failed to destroy switch", err)
		}
	}
	// tap/vmnet nics may be connected to an epair which is connected
	// to a netgraph pipe for purposes for rate limiting
	if vmNic.InstEpair != "" {
		err = epair.NgDestroyPipe(vmNic.InstEpair + "a")
		if err != nil {
			slog.Error("failed to ng pipe", err)
		}

		err = epair.NgDestroyPipe(vmNic.InstEpair + "b")
		if err != nil {
			slog.Error("failed to ng pipe", err)
		}
	}

	if err != nil {
		return fmt.Errorf("error cleaning up NIC: %w", err)
	}

	return nil
}

func (vm *VM) GetNics() ([]vmnic.VMNic, error) {
	nics := vmnic.GetNics(vm.Config.ID)

	return nics, nil
}
