# Cirrina

Daemon for [Bhyve](https://wiki.freebsd.org/bhyve) written in Go using gRPC

# Notes

This probably won't work for you:

* You need to load the `vmm`, `nmdm`, `if_bridge`, and `ng_bridge` kernel modules
* You need to be able to `sudo` *without a password prompt* to run the following commands:
  * `/sbin/ifconfig`
  * `/usr/bin/protect`
  * `/usr/sbin/bhyve`
  * `/usr/sbin/bhyvectl`
  * `/usr/sbin/ngctl`
  * `/usr/bin/truncate`
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
    * Support epair
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
  * Build Web UI -- maybe [awesome-grpc](https://github.com/grpc-ecosystem/awesome-grpc) has suggestions
  * Maybe run a DHCP server on switches of the proper "type"
  * Add cloud-init
    style [meta-data](https://docs.openstack.org/nova/train/admin/metadata-service.html) [server](https://docs.tinkerbell.org/services/hegel/)
  * Clean up protobuf api, specify max string lengths, check for missing values, etc.
  * Consider [go-sqlite](https://github.com/glebarez/go-sqlite)
  * Compare
    with [VirtualBox](https://www.virtualbox.org/wiki/Documentation), [vm-bhyve](https://github.com/churchers/vm-bhyve)
    and [chyves](http://chyves.org/) for missing features
  *
  Review [networking](https://freebsdfoundation.org/wp-content/uploads/2020/01/Arranging-Your-Virtual-Network-on-FreeBSD.pdf)
  * Consider a cirrinactl command for remote sound, similar to remote serial
  * Support suspend/resume
  * Support fw_cfg