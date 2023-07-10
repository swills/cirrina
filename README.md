# Cirrina

Daemon for [Bhyve](https://wiki.freebsd.org/bhyve) written in Go using gRPC

# Notes

This probably won't work for you:

* You need to load the `vmm`, `nmdm`, `if_bridge`, `if_epair`, `ng_pipe`, and `ng_bridge` kernel modules
* Only UEFI boot is supported, no bhyveload
* You need to be able to `sudo` *without a password prompt* to run the following commands:
  * `/sbin/ifconfig`
  * `/usr/bin/protect`
  * `/usr/sbin/bhyve`
  * `/usr/sbin/bhyvectl`
  * `/usr/sbin/ngctl`
  * `/usr/bin/truncate`

# Installation

Maybe you want to create a cirrinad user:

  ```
  # pw adduser cirrinad
  ```

You might want something like this in sudoers:

  ```
  Cmnd_Alias      CIRRINAD = /sbin/ifconfig, /usr/bin/protect, /usr/sbin/bhyve, /usr/sbin/bhyvectl, /usr/sbin/ngctl, /usr/bin/truncate
  cirrinad ALL=(ALL) NOPASSWD: CIRRINAD
  ```

You might want to do something like this:

  ```
  # mkdir -p /var/db/cirrinad /var/log/cirrinad /var/tmp/cirrinad /bhyve/disk /bhyve/isos /usr/local/etc/cirrinad
  # chown -R cirrina:cirrina /var/db/cirrinad /var/log/cirrinad /var/tmp/cirrinad /bhyve/disk /bhyve/isos
  # cp config.sample.yml /usr/local/etc/cirrinad/config.yml
  ```

You might want to edit /usr/local/etc/cirrinad/config.yml appropriately.

Maybe you want to add something like this to roots crontab:

  ```
  @reboot /usr/sbin/daemon -u cirrinad -f -r -S -P /var/run/cirrinad/cirrinad.daemon.pid -p /var/run/cirrinad/cirrinad.pid /usr/local/bin/cirrinad -config /usr/local/etc/cirrinad/config.yml
  ```

# How to use

## Build

* Build cirinad:
  * `cd cirrinad`
  * `go build -o cirrinad ./`
* Build cirrinactl:
  * `cd cirrinactl`
  * `go build ./...`

## Run Daemon

* Create and edit config
  * `cp config.sample.yml config.yml`
  * `vi config.yml`
  * Note:
    * Log, DB and ROM paths are files
    * Disk image, state and iso paths are directories
* Run cirrinad
  * `./cirrinad -config config.yml`

## Run Clients

* GUI
  * Start weasel
    * Create switch
    * Create VM
    * Add Disk
    * Add NIC
    * Upload ISO
    * Select VM, click edit, add disk, iso and nic to VM
    * Start VM
* Command line - Incomplete
  * Create a switch
    * `./cirrinactl -action addSwitch -name bridge0`
    * Note:
      * "IF" (`if_bridge`) type switches must have names which start with `bridge`
      * "NG" (`netgraph`) type switches must have names which start with `bnet`
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

# TODO

* Basics
  * Ensuring no crashes
  * Fixing race conditions
  * Fetching logs from server to client
  * Add auto-start delay
  * Implement VM rename - High Priority
  * Switch from polling to streaming for VM status
* Booting
  * Add feature to manage UEFI settings such as boot order to GUI
* Resources limiting
  * Disk I/O
* Devices
  * CPU
    * Implement CPU pinning support
    * Implement customization of CPU sockets, cores and threads - High Priority
    * CPU Usage limits
  * Serial
    * Input playback
  * Disk
    * Implement support for separate controllers and types and specifying which HD/CD are on which controller
    * Implement enlarging disks
    * Implement iSCSI disk support
    * Implement creating disks as zvols
    * Create disk from existing image (clone) or other disk image.
  * Networking
    * Support various network types from VBox
    * Support rate limiting - High Priority
    * Support vxlan and vale switches - High Priority
    * Maybe run a DHCP server on switches of the proper "type"
    * Tunnel/VPN support
  * Sound
    * Add sound device list
* Access/Sharing
  * VNC
    * Proxying - High Priority
    * Clipboard sharing
    * Resizing
    * Add VNC preview
    * Screenshots
    * Add VNC recording (see also [minimega feature](https://minimega.org/articles/vnc.article)
      and [vncproxy](https://pkg.go.dev/github.com/amitbet/vncproxy))
    * Add VNC input playback
  * SSH
    * Add SSH integration
  * RDP
  * Implement 9p sharing - High Priority
* Other/Ideas
  * Clients
    * Build TUI with [tview](https://github.com/rivo/tview)
    * Build Web UI -- maybe [awesome-grpc](https://github.com/grpc-ecosystem/awesome-grpc) has suggestions
  * Have GUI manage config for and automatically start daemon for a purely local mode
  * User/password auth - High Priority
  * Server manager with grouping - High Priority
  * VM grouping - High Priority
  * VM templates for various OSs
  * Automated OS install
  * VM Snapshots
  * Cloning
  * UI/API for specifying PCI bus/slot/feature of devices
  * OVA import/export
  * libvirt bhyve driver import/export
  * VM Stats (CPU/Mem/IO) in GUI
  * VM Logs in GUI
  * AWS and/or other cloud import/export/interoperability
  * Add cloud-init
    style [meta-data](https://docs.openstack.org/nova/train/admin/metadata-service.html)
    [server](https://docs.tinkerbell.org/services/hegel/)
  * Clean up protobuf api, specify max string lengths, check for missing values, etc.
  * Consider [go-sqlite](https://github.com/glebarez/go-sqlite)
  * Compare with:
    * [libvirt bhyve driver](https://libvirt.org/drvbhyve.html) - High Priority
    * [VirtualBox](https://www.virtualbox.org/wiki/Documentation)
    * [vm-bhyve](https://github.com/churchers/vm-bhyve)
    * [chyves](http://chyves.org/)
    * [ProxMox](https://pve.proxmox.com/)
    * [minimega](https://minimega.org/)
    * [Ganeti](https://ganeti.org/)
      Review [networking](https://freebsdfoundation.org/wp-content/uploads/2020/01/Arranging-Your-Virtual-Network-on-FreeBSD.pdf)
  * Consider a cirrinactl command for remote sound, similar to remote serial
  * Support suspend/resume
  * Switch to using bhyve config file instead of command line args
  * Support fw_cfg
  * Test grpc interface with [grpcurl](https://github.com/fullstorydev/grpcurl)
