# Cirrina

Daemon for [Bhyve](https://wiki.freebsd.org/bhyve) written in Go using gRPC

# Notes

This probably won't work for you:

* You need to load the `vmm`, `ng_ether` and `ng_bridge` kernel modules
* You need to be able to `sudo` without a password prompt to run the following commands:
  * `/sbin/ifconfig`
  * `/usr/sbin/bhyvectl`
  * `/usr/sbin/ngctl`
  * `/usr/bin/truncate`
* `doas` may work instead of `sudo` if you change the `priv_command_prefix` setting, but it is untested.
* At the moment some things can only be done with the cli (cirrinactl) and others can only be done with
  weasel (the py qt 5 gui)
* Only UEFI boot is supported, no bhyveload
* You must edit config.yml before starting cirrinad

# How to use

* Build cirinad:
  * `cd cirrinad`
  * `go build -o cirrinad ./`
* Create and edit config
  * `cp config.sample.yml config.yml`
  * `vi config.yml`
* Run cirrinad
  * `./cirrinad`
* Build cirrinactl:
  * `cd cirrinactl`
  * `go build ./...`
* Create a switch
  * `./cirrinactl -action addSwitch -name bridge0`
  * Or use the GUI (weasel)
* Set it's uplink
  * `./cirrinactl -action setSwitchUplink -switchId switchuuid -uplinkName "em0"`
  * Or use the GUI (weasel)
* Add an iso for your VM to use:
  * `./cirrinactl -action addISO -name something.iso`
  * `./cirrinactl -action uploadIso -id isoid -filePath /some/file/path.iso`
  * Or use the GUI (weasel)
* Add a disk for a VM:
  * `./cirrinactl -action addDisk -name something -descr 'a disk' -size 8g`
  * Or use the GUI (weasel)
* Add a NIC for a VM:
  * `./cirrinactl -action addVmNic -name something_int0 -switchId switchuuid`
  * Or use the GUI (weasel)
* Add a VM:
  * `./cirrinactl -action addVM -name something`
  * Or use the GUI (weasel)
* Start weasel
  * select VM, click edit, add disk, iso and nic to VM
  * start VM

# TODO

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
    * Support vxlan and vale switches
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
  * Build TUI with [Bubbletea](https://github.com/charmbracelet/bubbletea)
