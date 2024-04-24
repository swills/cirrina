//go:build !test

package cmd

func init() {
	disableFlagSorting(VMCom1Cmd)
	addNameOrIDArgs(VMCom1Cmd, &VMName, &VMID, "VM")

	disableFlagSorting(VMCom2Cmd)
	addNameOrIDArgs(VMCom2Cmd, &VMName, &VMID, "VM")

	disableFlagSorting(VMCom3Cmd)
	addNameOrIDArgs(VMCom3Cmd, &VMName, &VMID, "VM")

	disableFlagSorting(VMCom4Cmd)
	addNameOrIDArgs(VMCom4Cmd, &VMName, &VMID, "VM")
}
