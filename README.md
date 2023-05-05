# Cirrina

Daemon for [Bhyve](https://wiki.freebsd.org/bhyve) written in Go using gRPC

# How To

This probably won't work for you:

* You need to load the `vmm`, `ng_ether` and `ng_bridge` kernel modules
* You need to be able to `sudo` without a password prompt to run the following commands:
  * `/sbin/ifconfig`
  * `/usr/sbin/bhyvectl`
  * `/usr/sbin/ngctl`
  * `/usr/bin/truncate`
* `doas` may work instead of `sudo` if you change the `priv_command_prefix` setting, but it is untested.
* You have to have a bridge interface created and setup, something like this in `rc.conf`:
  ```
  cloned_interfaces="bridge0"
  ifconfig_bridge0="addm em0"
  ```
* Only one tap network interface per VM is supported
* At the moment some things can only be done with the cli (cirrinactl) and others can only be done with
  weasel (the py qt 5 gui)
* Only UEFI boot is supported, no bhyveload
* You must edit config.yml before starting cirrinad

# TODO

* Clean up
  * Separate log_path from config path
* Basics
  * Implement VM rename
  * Switch to using bhyve config file instead of command line args
* Booting
  * Add feature to manage UEFI settings such as boot order to GUI
* Devices
  * CPU
    * Implement CPU pinning support
    * Implement customization of CPU sockets, cores and threads
  * Serial
    * Implement serial console logging
  * Disk
    * Implement support for separate controllers and types and specifying which HD/CD are on which controller
    * Implement enlarging disks
    * Implement iSCSI disk support
    * Implement creating disks as zvols
    * Create disk from existing image (clone) or other disk image.
  * Networking
    * Add net device list
    * Support vxlan and vale switches
    * Implement picking tap mode bridge device
    * Implement picking netgraph bridge peer device
    * Add virtual network management
      * Add net device selection dialog
      * Add net device creation code
    * Implement tap mode bridge device
    * Implement netgraph bridge peer device
    * Add virtual network management
      * Add net device management code
      * Add net device creation code
  * Sound
    * Add sound device list
* Access/Sharing
  * VNC
    * Add VNC preview
  * SSH
    * Add SSH integration
  * RDP
  * Implement 9p sharing
* Other/Ideas
  * Add VNC recording
  * OVA import/export
  * AWS and/or other cloud import/export/interoperability
