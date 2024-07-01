
# TODO

* Finish writing tests
* Make sure iso, nic, disk switch are uniform, ie, take same args for all funcs
  * For example, iso delete takes string, nic delete takes object
* Convert NICs to using custom join table, in order to preserve order
* Convert all UUIDs from strings to UUID type
* Convert all paths from strings to path/filepath
* Convert all bool in database to `sql.NullBool`
* Finish cloning -- NICs are done, need to do switches, disks and VMs
* Do templating
* Switch to using Netlink
* arm64 support: kern.osreldate 1500018 -- need to wait for 1500019 and test for that or higher
* Fix zvol ownership!
* Fix cirrinactl over IPv6
* Actually delete disks from the database
* Add a force kill function for OSs that won't shut down properly and when you don't want to wait
* Add feature to remove CD after first boot
* have all cirrinactl commands which use server make a call to hostPing() before doing anything with the server
* Use consistent terminology:
  * destroy, remove -> always use remove?
  * error, failed -> always use error?
  * switch, bridge -> always use switch?
  * nic, vmnic -> always use nic?
  * iso, ISO -> always use ISO in messages? something else?
  * Convert disk/iso on disk file name from actual disk/iso name to uuid, maybe
* Add option to cirrinactl com stream to:
  * Avoid quit info message and pause on startup
  * Avoid clearing screen
  * If logging, fetch last N bytes from log and play them back
  * Remain running on VM shutdown, either just displaying what was there, or polling the VM for startup
* In cirrinactl useCom, add features from cu(1) such as
  * Sending break
    * from src/usr.bin/tip/tip/cmds.c:
      * `ioctl(FD, TIOCSBRK, NULL); sleep(1); ioctl(FD, TIOCCBRK, NULL);`
  * Sending/receiving from/to local file
* Disk resize -- include force flag to allow reducing disk size which includes data loss
* Fix setting the switch on a NIC while a VM is running
  * Currently, it won't work until you stop then start the VM
  * It should work immediately, without even a reboot, but also if you reboot or stop then start
* Add error checking and result count checking if applicable to all db queries, as was done in cirrinad/switch/switch.go
* Return nil instead of empty value whenever possible (pointers)
* Ensure all error returns return no value(s), nil if possible
* Eliminate `map[string]...` things due to returning in random order
* Add disk update to server/client
* Add zvol support to GUI - including listing dev type in disk list
* Add vm priority (nice) stuff to GUI
* Add check for disk size reduction in file based image uploads
* Disk cloning
* VM cloning
* Auto decompress isos/disk images - delete compression type
* Update max vnc screen size, see src fb51ddb20d57a43d666508e600af1bc7ac85c4e8
  * use kern.osreldate: 1500017 and earlier, 1920x1200 is the max, for 15 or later, 3840x2160
* Cope with changes to ng_bridge in src 86a6393a7d6766875a9e03daa0273a2e55faacdd

* Basics
    * Ensuring no crashes
    * Fixing race conditions
    * Prevent modifying things that are in use and/or make it clear which changes will take effect when
    * Fetching logs from server to client
    * Implement VM rename
    * Switch from polling to streaming for VM status
    * Error checking
      * Ensure switch uplinks exist
      * Ensure config paths exist
      * Ensure disk paths/pool/volumes exist
* Booting
    * Add feature to manage UEFI settings such as boot order to GUI
* Devices
    * CPU
        * Implement CPU pinning support
        * Implement customization of CPU sockets, cores and threads
    * Serial
        * Input playback
    * Disk
        * Implement support for separate controllers and types and specifying which HD/CD are on which controller
        * Implement enlarging disks
        * Implement iSCSI disk support
          * [handbook docs](https://docs.freebsd.org/en/books/handbook/network-servers/#network-iscsi)
          * [man page](https://man.freebsd.org/cgi/man.cgi?query=ctladm)
    * Networking
        * Support [Open vSwitch](https://www.openvswitch.org/) ([port](https://www.freshports.org/net/openvswitch/))
        * Switch from `sudo ifconfig ...` to using [netlink](https://man.freebsd.org/cgi/man.cgi?netlink) via [go lib](https://pkg.go.dev/github.com/vishvananda/netlink) once netlink is in all supported FreeBSD versions
        * Support various network types from VBox
        * Support [vxlan](https://www.bsdcan.org/2016/schedule/attachments/341_VXLAN_BSDCan2016.pdf) [video](https://www.youtube.com/watch?v=_1Ne_TgF3MQ) and [stuff](https://www.bsdcan.org/2016/schedule/events/715.en.html) and vale switches
        * Maybe run a DHCP server on switches of the proper "type"
        * Support [NATing](https://github.com/zed-0xff/ng_sbinat) VMs [via](https://github.com/MonkWho/pfatt/blob/master/bin/pfatt.sh) [netgraph](https://reviews.freebsd.org/D23461)
        * Tunnel/VPN support
        * Make switches not send traffic not destined for the particular VM by default, like VMWare blocking "promiscuous" mode.
    * Sound
        * Add sound device list
* Access/Sharing
    * VNC
        * Proxying
        * Clipboard sharing
        * Resizing
        * Add VNC preview
        * Screenshots
        * Add VNC recording (see also [minimega feature](https://minimega.org/articles/vnc.article)
          and [vncproxy](https://pkg.go.dev/github.com/amitbet/vncproxy))
        * Maybe [screego](https://screego.net/) ([port](https://www.freshports.org/www/screego/)) or [neko](https://github.com/m1k1o/neko) could help here
        * Add VNC input playback
    * SSH
        * Add SSH integration
    * RDP
    * Implement [9p](https://reviews.freebsd.org/D41844) sharing or [this](https://github.com/swills/virtfs-9p-kmod/pull/4)
* Other/Ideas
    * Use real uuid type instead of string everywhere
    * Clients
        * Build TUI with [tview](https://github.com/rivo/tview)
          * Monitor VMs, similar to "cirrinactl vm list" in a loop
          * Add top/htop like interface, with VM list and resource usage
        * Build Web UI -- maybe [awesome-grpc](https://github.com/grpc-ecosystem/awesome-grpc) has suggestions
          * Use [this](https://github.com/kuba2k2/firefox-webserial) as a way to get serial stuff
    * Have GUI manage config for and automatically start daemon for a purely local mode
    * User/password auth - perhaps [spiffe](https://spiffe.io/) or [dapr](https://dapr.io/) could help here
    * VM grouping
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
    * Add cloud-init support
      style [meta-data](https://docs.openstack.org/nova/train/admin/metadata-service.html)
      [server](https://docs.tinkerbell.org/services/hegel/)
    * Test cloud-init with FreeBSD see [commit](https://cgit.freebsd.org/src/commit/?id=1f4ce7a39f0f4b0621ff55d228014ccddb366d37)
    * Clean up protobuf api, specify max string lengths, check for missing values, etc.
    * Consider [go-sqlite](https://github.com/glebarez/go-sqlite)
    * Compare with:
        * [libvirt bhyve driver](https://libvirt.org/drvbhyve.html)
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
    * See also [nc-vsock](https://github.com/stefanha/nc-vsock) and [this](https://gist.github.com/mcastelino/9a57d00ccf245b98de2129f0efe39857)
    * And [this](https://wiki.qemu.org/Features/VirtioVsock) and [this](https://lwn.net/Articles/556550/)
    * And [this](https://www.linux-kvm.org/images/8/83/01x08-Stefan_Hajnoczi-virtio-vsock_Zero-configuration_hostguest_communication.pdf)
    * And [this](https://pkg.go.dev/github.com/mdlayher/vsock@v1.2.1/cmd/vscp#section-readme) and [this](https://github.com/mdlayher/vsock)
    * And [this](https://stefano-garzarella.github.io/posts/2019-11-08-kvmforum-2019-vsock/) and [this](https://www.youtube.com/watch?v=_bYSQ68JPwE) and [this](https://www.youtube.com/watch?v=LFqz-VZPhFE)
    * More on [virtio-vsock](https://www.youtube.com/watch?v=LFqz-VZPhFE) [here](https://www.youtube.com/watch?v=_bYSQ68JPwE)
    * [Support](https://github.com/FreeBSD-UPB/freebsd-src/wiki/Virtual-Machine-Migration-using-bhyve) [live migration](https://lists.freebsd.org/archives/freebsd-virtualization/2023-June/001369.html)
    * [More](https://manpages.ubuntu.com/manpages/impish/man1/cloud-localds.1.html) [on](https://github.com/racingmars/vm-provision/blob/master/create.sh) [templates](https://www.youtube.com/watch?v=jxItb7iZyR0)
      * [snapshot fetch](https://download.freebsd.org/snapshots/VM-IMAGES/15.0-CURRENT/)
    * Run bhyve in jail
    * Run bhyve in [IP-less jail](https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=273557)
    * Support [BIOS](https://www.gulbra.net/freebsd-bhyve/)
    * Support [more](https://github.com/freenas/libhyve-remote) [VNC encoding](https://reviews.freebsd.org/D11768)
    * Support [QCOW2](https://github.com/xcllnt/libvdsk)
    * Test grpc interface with [grpcurl](https://github.com/fullstorydev/grpcurl)
    * 
    * Look into use QUIC via [quic-go](https://github.com/quic-go/quic-go), perhaps for host to host disk image transfer
    * [Convert](https://github.com/grpc/grpc-go/blob/master/examples/features/error_details/client/main.go) all grpc [errors](https://grpc.github.io/grpc/core/md_doc_statuscodes.html), especially invalid argument
    * More notes on gRPC error handling [here](https://cloud.google.com/apis/design/errors#error_model) and [here](https://grpc.io/docs/guides/error/)
    * Add more [detail](https://github.com/grpc/grpc-go/blob/master/Documentation/rpc-errors.md) to grpc errors
    * Also [here](https://jbrandhorst.com/post/grpc-errors/)
    * [Improve](https://protobuf.dev/programming-guides/techniques/) [grpc](https://protobuf.dev/programming-guides/dos-donts/) [stuff](https://protobuf.dev/programming-guides/api/)
    * Use well known [types](https://protobuf.dev/reference/protobuf/google.protobuf/) in proto, especially Empty
      * Plus some other things like:
        * google/protobuf/duration.proto
        * google/protobuf/timestamp.proto
        * google/protobuf/wrappers.proto
        * google/rpc/status.proto
        * google/longrunning/operations.proto
        * google/protobuf/any.proto
    * Improve python [protobuf](https://protobuf.dev/programming-guides/proto3/) [code](https://protobuf.dev/reference/python/python-generated/#fields)
    * [Rate limit grpc](https://stackoverflow.com/questions/62925871/grpc-rate-limiting-an-api-on-a-per-rpc-basis )
    * [OpenConfig](https://openconfig.net/) support? YANG/Netconf? [Sysconf](https://github.com/sysrepo/sysrepo)
    * http support [grpc-crud](https://github.com/Edmartt/grpc-crud) or [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway)
      * See also [grpc-transcoding](https://cloud.google.com/endpoints/docs/grpc/transcoding)
    * [OpenTelemetry](https://opentelemetry.io/docs/languages/go/getting-started/) support
    * [Prometheus](https://prometheus.io/docs/guides/go-application/) support
    * Follow [Go Style](https://google.github.io/styleguide/go/) and [Best Practices](https://google.github.io/styleguide/go/best-practices)
    * [arm64 support](https://cgit.freebsd.org/src/commit/?id=47e073941f4e7ca6e9bde3fa65abbfcfed6bfa2b)
    * Clustering using [etcd](https://pkg.go.dev/github.com/coreos/etcd/embed#Etcd)
    * Data sharing using [bbold](https://github.com/etcd-io/bbolt)
    * Add [virtio_random](https://man.freebsd.org/cgi/man.cgi?query=virtio_random&apropos=0&sektion=0&manpath=FreeBSD+14.0-RELEASE+and+Ports&arch=default&format=html) device?
    * Perhaps other [virtio](https://man.freebsd.org/cgi/man.cgi?query=virtio&sektion=4&apropos=0&manpath=FreeBSD+14.0-RELEASE+and+Ports) devices?
    * Eventually use tcp for console, see [review](https://reviews.freebsd.org/D43514)
    * TPM pass-through support
    * PCIe device pass-through
    * Support booting directly to [netboot.xyz](https://netboot.xyz/docs/booting/ipxe/) using [https boot](https://github.com/tianocore/tianocore.github.io/wiki/HTTP-Boot) in edk2
      * requires a better/different dns entry than the current public one, perhaps run our own and host the netboot.xyz.efi file too
