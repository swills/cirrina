# Cirrina

Daemon for [Bhyve](https://wiki.freebsd.org/bhyve) written in Go using gRPC

# Warning

This is still fairly new software. Only UEFI boot is supported, no bhyveload.

# Installation

## Requirements

### User

Run (as `root`):

```
pw groupadd -n cirrinad -g 10001
pw useradd -n cirrinad -u 10001 -g cirrinad -s /usr/sbin/nologin -d /nonexistent
```

### edk2

Run (as `root`):

```
pkg install sudo edk2-bhyve
```

### sudo

Ensure you have `sudo` installed. Then, run `visudo` and add:

```
Cmnd_Alias      CIRRINAD = /sbin/ifconfig, /sbin/zfs, /usr/bin/nice, /usr/bin/protect, /usr/bin/rctl, /usr/bin/truncate, /usr/sbin/bhyve, /usr/sbin/bhyvectl, /usr/sbin/ngctl, /bin/pgrep, /usr/sbin/chown
cirrinad ALL=(ALL) NOPASSWD: CIRRINAD
```

### kernel modules

Run (as `root`):

```
echo kern.racct.enable=1 >> /boot/loader.conf
reboot
sysrc kld_list+="vmm nmdm if_bridge if_epair ng_bridge ng_ether ng_pipe"
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
go build .
cp cirrinactl /usr/local/bin
```

## Setup

### Directories

Run (as `root`):

```
mkdir -p /var/run/cirrinad /var/db/cirrinad /var/log/cirrinad /var/tmp/cirrinad /bhyve/disk /bhyve/isos
chown -R cirrinad:cirrinad /var/run/cirrinad /var/db/cirrinad /var/log/cirrinad /var/tmp/cirrinad /bhyve/disk /bhyve/isos
```

### Config

Edit `/usr/local/etc/cirrinad/config.yml` if necessary. Note: Log, DB and ROM paths must be files. Disk image, state
and iso paths must be directories.

### Startup

Run (as `root`):

```
cp rc/cirrinad /usr/local/etc/rc.d/
sysrc cirrinad_enable=yes
service cirrinad start
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
    * `./cirrinactl iso upload -n something.iso -P /some/file/path/something.iso`
  * Add a disk for a VM:
    * `./cirrinactl disk create -n somediskname -s 32G`
  * Add a NIC for a VM:
    * `./cirrinactl nic create -n something_int0`
    * `./cirrinactl nic setswitch -n something_int0 -N bridge0`
  * Add a VM:
    * `./cirrinactl vm create -n something`
  * Add disk to VM
    * `./cirrinactl vm disk add -n something -N somediskname`
  * Add NIC to VM
    * `./cirrinactl vm nic add -n something -N something_int0`
  * Set config for a VM:
    * `./cirrinactl vm config -n something -c 2 -m 4096`
    * `./cirrinactl vm config -n something --description "some description"`
  * Start the VM:
    * `./cirrinactl vm start -n something`
