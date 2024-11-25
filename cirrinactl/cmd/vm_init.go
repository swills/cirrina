//go:build !test

package cmd

import "fmt"

func init() {
	disableFlagSorting(VMCmd)

	setupVMListCmd()

	err := setupVMCreateCmd()
	if err != nil {
		panic(err)
	}

	setupVMDeleteCmd()
	setupVMStartCmd()
	setupVMStopCmd()
	setupVMConfigCmd()
	setupVMGetCmd()
	setupVMClearUefiVarsCmd()

	VMCmd.AddCommand(VMListCmd)
	VMCmd.AddCommand(VMCreateCmd)
	VMCmd.AddCommand(VMDeleteCmd)
	VMCmd.AddCommand(VMConfigCmd)
	VMCmd.AddCommand(VMGetCmd)
	VMCmd.AddCommand(VMStartCmd)
	VMCmd.AddCommand(VMStopCmd)
	VMCmd.AddCommand(VMCom1Cmd)
	VMCmd.AddCommand(VMCom2Cmd)
	VMCmd.AddCommand(VMCom3Cmd)
	VMCmd.AddCommand(VMCom4Cmd)
	VMCmd.AddCommand(VMClearUefiVarsCmd)
}

func setupVMClearUefiVarsCmd() {
	disableFlagSorting(VMClearUefiVarsCmd)
	addNameOrIDArgs(VMClearUefiVarsCmd, &VMName, &VMID, "VM")
}

func setupVMGetCmd() {
	disableFlagSorting(VMGetCmd)
	addNameOrIDArgs(VMGetCmd, &VMName, &VMID, "VM")
	VMGetCmd.Flags().StringVarP(&outputFormatString, "format", "f", outputFormatString,
		"Output format (txt, json, yaml",
	)
}

func setupVMConfigCmd() {
	addNameOrIDArgs(VMConfigCmd, &VMName, &VMID, "VM")
	disableFlagSorting(VMConfigCmd)
	setupVMConfigBasics()
	setupVMConfigVMPriorityLimits()
	setupVMConfigCom1()
	setupVMConfigCom2()
	setupVMConfigCom3()
	setupVMConfigCom4()
	setupVMConfigStart()
	setupVMConfigScreen()
	setupVMConfigSound()
	setupVMConfigAdvanced1()
	setupVMConfigAdvanced2()
	setupVMConfigDebug()
}

func setupVMConfigDebug() {
	VMConfigCmd.Flags().BoolVar(&Debug, "debug", Debug, "Enable Debug server")
	VMConfigCmd.Flags().BoolVar(&DebugWait,
		"debug-wait", DebugWait, "Wait for connection to debug server before starting VM",
	)
	VMConfigCmd.Flags().Uint32Var(&DebugPort, "debug-port", DebugPort, "TCP port to use for debug server")
}

func setupVMConfigAdvanced2() {
	VMConfigCmd.Flags().BoolVar(&Dpo, "dpo", Dpo, "Destroy the VM on guest initiated power off")
	VMConfigCmd.Flags().BoolVar(&Eop,
		"eop", Eop, "Force the virtual CPU(s) to exit when a PAUSE instruction is detected",
	)
	VMConfigCmd.Flags().BoolVar(&Ium, "ium", Ium, "Ignore unimplemented model specific register access")
	VMConfigCmd.Flags().BoolVar(&Hlt,
		"hlt", Hlt, "Yield the virtual CPU(s), when a HTL instruction is detected",
	)
	VMConfigCmd.Flags().StringVar(&ExtraArgs, "extra-args", ExtraArgs, "Extra args to pass to bhyve")
}

func setupVMConfigAdvanced1() {
	VMConfigCmd.Flags().BoolVar(&HostBridge, "host-bridge", HostBridge, "Enable host bridge")
	VMConfigCmd.Flags().BoolVar(&Acpi, "acpi", Acpi, "Enable ACPI tables")
	VMConfigCmd.Flags().BoolVar(&Uefi, "uefi", Uefi, "Store UEFI variables")
	VMConfigCmd.Flags().BoolVar(&Utc, "utc", Utc, "Store VM time in UTC")
	VMConfigCmd.Flags().BoolVar(&Wire, "wire", Wire, "Wire guest memory")
}

func setupVMConfigSound() {
	VMConfigCmd.Flags().BoolVar(&Sound, "sound", Sound, "Enabled Sound output on this VM")
	VMConfigCmd.Flags().StringVar(&SoundIn, "sound-in", SoundIn, "Device to use for sound input")
	VMConfigCmd.Flags().StringVar(&SoundOut, "sound-out", SoundOut, "Device to use for sound output")
}

func setupVMConfigScreen() {
	VMConfigCmd.Flags().BoolVar(&Screen, "screen", Screen, "Start VNC Server for this VM")
	VMConfigCmd.Flags().StringVar(&ScreenSize, "screen-size", ScreenSize,
		"Shortcut reference to standard screen dimensions: VGA, SVGA, XGA, SXGA, UXGA, WUXGA, QXGA, etc. up to QUXGA")
	VMConfigCmd.Flags().Uint32Var(&ScreenWidth, "screen-width", ScreenWidth, "Width of VNC server screen")
	VMConfigCmd.Flags().Uint32Var(&ScreenHeight,
		"screen-height", ScreenHeight, "Height of VNC server screen",
	)
	VMConfigCmd.Flags().StringVar(&VncPort,
		"vnc-port", VncPort, "Port to run VNC server on, AUTO for automatic, or TCP port number",
	)
	VMConfigCmd.Flags().BoolVar(&VncWait,
		"vnc-wait", VncWait, "Wait for VNC connection before starting VM",
	)
	VMConfigCmd.Flags().BoolVar(&VncTablet, "vnc-tablet", VncTablet, "VNC server in tablet mode")
	VMConfigCmd.Flags().StringVar(&VncKeyboard,
		"vnc-keyboard", VncKeyboard, "Keyboard layout used by VNC server",
	)
}

func setupVMConfigStart() {
	VMConfigCmd.Flags().BoolVar(&AutoStart, "autostart", AutoStart, "Autostart VM")
	VMConfigCmd.Flags().Uint32Var(&AutoStartDelay,
		"autostart-delay", AutoStartDelay, "How long to wait before starting this VM",
	)
	VMConfigCmd.Flags().BoolVar(&Restart,
		"restart", Restart, "Restart this VM if it stops, crashes, shuts down, reboots, etc.",
	)
	VMConfigCmd.Flags().Uint32Var(&RestartDelay,
		"restart-delay", RestartDelay, "How long to wait before restarting this VM",
	)
	VMConfigCmd.Flags().Uint32Var(&MaxWait,
		"max-wait", MaxWait, "How long to wait for this VM to shutdown before forcibly killing it",
	)
}

func setupVMConfigCom4() {
	VMConfigCmd.Flags().BoolVar(&Com4, "com4", Com4, "Enable COM4")
	VMConfigCmd.Flags().BoolVar(&Com4Log, "com4-log", Com4Log, "Log input and output of COM4")
	VMConfigCmd.Flags().StringVar(&Com4Dev, "com4-dev", Com4Dev, "Device to use for COM4")
	VMConfigCmd.Flags().Uint32Var(&Com4Speed, "com4-speed", Com4Speed, "Speed of COM4")
}

func setupVMConfigCom3() {
	VMConfigCmd.Flags().BoolVar(&Com3, "com3", Com3, "Enable COM3")
	VMConfigCmd.Flags().BoolVar(&Com3Log, "com3-log", Com3Log, "Log input and output of COM3")
	VMConfigCmd.Flags().StringVar(&Com3Dev, "com3-dev", Com3Dev, "Device to use for COM3")
	VMConfigCmd.Flags().Uint32Var(&Com3Speed, "com3-speed", Com3Speed, "Speed of COM3")
}

func setupVMConfigCom2() {
	VMConfigCmd.Flags().BoolVar(&Com2, "com2", Com2, "Enable COM2")
	VMConfigCmd.Flags().BoolVar(&Com2Log, "com2-log", Com2Log, "Log input and output of COM2")
	VMConfigCmd.Flags().StringVar(&Com2Dev, "com2-dev", Com2Dev, "Device to use for COM2")
	VMConfigCmd.Flags().Uint32Var(&Com2Speed, "com2-speed", Com2Speed, "Speed of COM2")
}

func setupVMConfigCom1() {
	VMConfigCmd.Flags().BoolVar(&Com1, "com1", Com1, "Enable COM1")
	VMConfigCmd.Flags().BoolVar(&Com1Log, "com1-log", Com1Log, "Log input and output of COM1")
	VMConfigCmd.Flags().StringVar(&Com1Dev, "com1-dev", Com1Dev, "Device to use for COM1")
	VMConfigCmd.Flags().Uint32Var(&Com1Speed, "com1-speed", Com1Speed, "Speed of COM1")
}

func setupVMConfigVMPriorityLimits() {
	VMConfigCmd.Flags().Int32Var(&Priority, "priority", Priority, "Priority of VM (nice)")
	VMConfigCmd.Flags().BoolVar(&Protect,
		"protect", Protect, "Protect VM from being killed when swap space is exhausted",
	)
	VMConfigCmd.Flags().Uint32Var(&Pcpu, "pcpu", Pcpu, "Max CPU usage in percent of a single CPU core")
	VMConfigCmd.Flags().Uint32Var(&Rbps, "rbps", Rbps, "Limit VM filesystem reads, in bytes per second")
	VMConfigCmd.Flags().Uint32Var(&Wbps, "wbps", Wbps, "Limit VM filesystem writes, in bytes per second")
	VMConfigCmd.Flags().Uint32Var(&Riops,
		"riops", Riops, "Limit VM filesystem reads, in operations per second",
	)
	VMConfigCmd.Flags().Uint32Var(&Wiops,
		"wiops", Wiops, "Limit VM filesystem writes, in operations per second",
	)
}

func setupVMConfigBasics() {
	VMConfigCmd.Flags().StringVarP(&VMDescription,
		"description", "d", VMDescription, "SwitchDescription of VM",
	)
	VMConfigCmd.Flags().Uint16VarP(&Cpus, "cpus", "c", Cpus, "Number of VM virtual CPUs")
	VMConfigCmd.Flags().Uint32VarP(&Mem,
		"mem", "m", Mem, "Amount of virtual memory in megabytes",
	)
}

func setupVMStopCmd() {
	addNameOrIDArgs(VMStopCmd, &VMName, &VMID, "VM")
	VMStopCmd.Flags().BoolVarP(&CheckReqStat, "status", "s", CheckReqStat, "Check status")
	disableFlagSorting(VMStopCmd)
}

func setupVMStartCmd() {
	disableFlagSorting(VMStartCmd)
	addNameOrIDArgs(VMStartCmd, &VMName, &VMID, "VM")
	VMStartCmd.Flags().BoolVarP(&CheckReqStat, "status", "s", CheckReqStat, "Check status")
}

func setupVMDeleteCmd() {
	disableFlagSorting(VMDeleteCmd)
	addNameOrIDArgs(VMDeleteCmd, &VMName, &VMID, "VM")
}

func setupVMCreateCmd() error {
	disableFlagSorting(VMCreateCmd)
	VMCreateCmd.Flags().StringVarP(&VMName, "name", "n", VMName, "Name of VM")

	err := VMCreateCmd.MarkFlagRequired("name")
	if err != nil {
		return fmt.Errorf("error marking flag required: %w", err)
	}

	VMCreateCmd.Flags().StringVarP(&VMDescription,
		"description", "d", SwitchDescription, "SwitchDescription of VM",
	)
	VMCreateCmd.Flags().Uint16VarP(&Cpus, "cpus", "c", Cpus, "Number of VM virtual CPUs")
	VMCreateCmd.Flags().Uint32VarP(&Mem,
		"mem", "m", Mem, "Amount of virtual memory in megabytes",
	)

	return nil
}

func setupVMListCmd() {
	disableFlagSorting(VMListCmd)
	VMListCmd.Flags().BoolVarP(&Humanize,
		"human", "H", Humanize, "Print sizes in human readable form",
	)
	VMListCmd.Flags().BoolVarP(&ShowUUID,
		"uuid", "u", ShowUUID, "Show UUIDs",
	)
}
