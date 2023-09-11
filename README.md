# Cirrina

Daemon for [Bhyve](https://wiki.freebsd.org/bhyve) written in Go using gRPC

# Warning

This is still fairly new software. Only UEFI boot is supported, no bhyveload.

# Installation

## Requirements

### User

Run (as `root`):

```
pw adduser cirrinad
```

### sudo

Ensure you have `sudo` installed. Then, run `visudo` and add:

```
Cmnd_Alias      CIRRINAD = /sbin/ifconfig, /usr/bin/protect, /usr/sbin/bhyve, /usr/sbin/bhyvectl, /usr/sbin/ngctl, /usr/bin/truncate
cirrinad ALL=(ALL) NOPASSWD: CIRRINAD
```

### kernel modules

Run (as `root`):

```
sysrc kld_list="vmm nmdm if_bridge if_epair ng_bridge ng_ether ng_pipe"
service kld restart
```

## Build and install binaries:

```
cd cirrinad
go build -o cirrinad ./
cp cirrinad /usr/local/bin
mkdir /usr/local/etc/cirrinad
cp config.sample.yml /usr/local/etc/cirrinad/config.yml
cd ../cirrinactl
go build ./...
cp cirrinactl /usr/local/bin
```

## Setup

### Directories

Run (as `root`):

```
mkdir -p /var/db/cirrinad /var/log/cirrinad /var/tmp/cirrinad /bhyve/disk /bhyve/isos
chown -R cirrinad:cirrinad /var/db/cirrinad /var/log/cirrinad /var/tmp/cirrinad /bhyve/disk /bhyve/isos
```

### Config

Edit `/usr/local/etc/cirrinad/config.yml` if necessary. Note: Log, DB and ROM paths must be files. Disk image, state
and iso paths must be directories.

### Startup

Run (as `root`):

```
crontab -e
```

and add:

```
@reboot /usr/sbin/daemon -u cirrinad -f -r -S -P /var/run/cirrinad/cirrinad.daemon.pid -p /var/run/cirrinad/cirrinad.pid /usr/local/bin/cirrinad -config /usr/local/etc/cirrinad/config.yml
```

# How to use

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
* Command line
  * Create a switch
    * `./cirrinactl switch create -n bridge0`
    * Note:
      * "IF" (`if_bridge`) type switches must have names which start with `bridge`
      * "NG" (`netgraph`) type switches must have names which start with `bnet`
  * Set it's uplink
    * `./cirrinactl switch set-uplink -n bridge0 -u em0`
  * Add an iso for your VM to use:
    * `./cirrinactl iso add -n something.iso`
    * This returns an iso id (UUID). Use this ID to upload the iso file:
    * `./cirrinactl iso upload -i <isoid> -P /some/file/path/something.iso`
  * Add a disk for a VM:
    * `./cirrinactl disk create -n somediskname -s 32G`
  * Add a NIC for a VM:
    * `./cirrinactl nic create -n something_int0`
    * `./cirrinactl nic setswitch -n something_int0 -N bridge0`
  * Add a VM:
    * `./cirrinactl vm create -n something`
  * Add disk to VM
    * `./cirrainctl vm disk add -n something -N somediskname`
  * Add NIC to VM
    * `./cirrinactl vm nic add -n something -N something_int0`
  * Set config for a VM:
    * `./cirrainctl vm config -n something -c 2 -m 4096`
    * `./cirrinactl vm config -n something --description "some description"`
  * Start the VM:
    * `./cirrinactl vm start -n something`

# TODO

* Basics
  * Ensuring no crashes
  * Fixing race conditions
  * Fetching logs from server to client
  * Implement VM rename - High Priority
  * Switch from polling to streaming for VM status
* Booting
  * Add feature to manage UEFI settings such as boot order to GUI
* Resources limiting
  * [Disk](https://arstechnica.com/gadgets/2020/02/how-fast-are-your-disks-find-out-the-open-source-way-with-fio/) I/O
    * Have tested with [rctl](https://man.freebsd.org/cgi/man.cgi?rctl) but it's per process not per VM disk
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
    * Support [vxlan](https://www.bsdcan.org/2016/schedule/attachments/341_VXLAN_BSDCan2016.pdf) [video](https://www.youtube.com/watch?v=_1Ne_TgF3MQ) and [stuff](https://www.bsdcan.org/2016/schedule/events/715.en.html) and vale switches - High Priority
    * Maybe run a DHCP server on switches of the proper "type"
    * Support [NATing](https://github.com/zed-0xff/ng_sbinat) VMs [via](https://github.com/MonkWho/pfatt/blob/master/bin/pfatt.sh) [netgraph](https://reviews.freebsd.org/D23461)
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
  * Use cobra/viper in cirrinad
  * Use real uuid type instead of string everywhere
  * Clients
    * Build TUI with [tview](https://github.com/rivo/tview)
    * Build Web UI -- maybe [awesome-grpc](https://github.com/grpc-ecosystem/awesome-grpc) has suggestions
  * Have GUI manage config for and automatically start daemon for a purely local mode
  * User/password auth - High Priority
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
  * Set CDSR_OFLOW ("stty dsrflow") on nmdm devs to enforce port speed
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
  * Review [networking](https://freebsdfoundation.org/wp-content/uploads/2020/01/Arranging-Your-Virtual-Network-on-FreeBSD.pdf)
  * Consider a cirrinactl command for remote sound, similar to remote serial
  * Support suspend/resume
  * Switch to using bhyve config file instead of command line args
  * Support fw_cfg
  * Remove calls to external programs -- particularly hard for ngctl, but doable, tho requires setting up the socket
  * [Distribute](https://en.wikipedia.org/wiki/Distributed_SQL) the database via mvsqlite, dqlite or something similar, but not rqlite because it lacks a sql or gorm driver and can't really have one
  * Use [virtio-vsock](https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=271793) to [communicate](https://github.com/linuxkit/virtsock) with VMs
  * More on [virtio-vsock](https://www.youtube.com/watch?v=LFqz-VZPhFE) [here](https://www.youtube.com/watch?v=_bYSQ68JPwE)
  * [Support](https://github.com/FreeBSD-UPB/freebsd-src/wiki/Virtual-Machine-Migration-using-bhyve) [live migration](https://lists.freebsd.org/archives/freebsd-virtualization/2023-June/001369.html)
  * More on [templates](https://www.youtube.com/watch?v=jxItb7iZyR0)
  * Run bhyve in jail
  * Run bhyve in [IP-less jail](https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=273557)
  * Support [BIOS](https://www.gulbra.net/freebsd-bhyve/) 
  * Support [more](https://github.com/freenas/libhyve-remote) [VNC encoding](https://reviews.freebsd.org/D11768)
  * Support [QCOW2](https://github.com/xcllnt/libvdsk)
  * Test grpc interface with [grpcurl](https://github.com/fullstorydev/grpcurl)
  * [Convert](https://github.com/grpc/grpc-go/blob/master/examples/features/error_details/client/main.go) all grpc [errors](https://grpc.github.io/grpc/core/md_doc_statuscodes.html), especially invalid argument
  * Add more [detail](https://github.com/grpc/grpc-go/blob/master/Documentation/rpc-errors.md) to grpc errors
  * [Improve](https://protobuf.dev/programming-guides/techniques/) [grpc](https://protobuf.dev/programming-guides/dos-donts/) [stuff](https://protobuf.dev/programming-guides/api/)
  * Use well known [types](https://protobuf.dev/reference/protobuf/google.protobuf/) in proto, especially Empty
  * Improve python [protobuf](https://protobuf.dev/programming-guides/proto3/) [code](https://protobuf.dev/reference/python/python-generated/#fields)
  * [Rate limit grpc](https://stackoverflow.com/questions/62925871/grpc-rate-limiting-an-api-on-a-per-rpc-basis )
