
# TODO

* Convert all bool in database to `sql.NullBool`
* Remove paths from db, construct it on the fly
* Ensure all listing in client default to human-readable, add option for exact number (-p)
* Add disk update to server/client
* Add zvol support to GUI

* Basics
    * Ensuring no crashes
    * Fixing race conditions
    * Fetching logs from server to client
    * Implement VM rename
    * Switch from polling to streaming for VM status
    * Error checking
      * Prevent running two copies of cirrinad
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
        * Create disk from existing image (clone) or other disk image.
    * Networking
        * Support various network types from VBox
        * Support [vxlan](https://www.bsdcan.org/2016/schedule/attachments/341_VXLAN_BSDCan2016.pdf) [video](https://www.youtube.com/watch?v=_1Ne_TgF3MQ) and [stuff](https://www.bsdcan.org/2016/schedule/events/715.en.html) and vale switches
        * Maybe run a DHCP server on switches of the proper "type"
        * Support [NATing](https://github.com/zed-0xff/ng_sbinat) VMs [via](https://github.com/MonkWho/pfatt/blob/master/bin/pfatt.sh) [netgraph](https://reviews.freebsd.org/D23461)
        * Tunnel/VPN support
        * Make switches not send traffic not destined for the particular VM by default, like VMware blocking "promiscuous" mode.
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
    * Implement 9p sharing
* Other/Ideas
    * Use cobra/viper in cirrinad
    * Use [deque](https://pkg.go.dev/github.com/gammazero/deque) to replace the requests table
    * Use real uuid type instead of string everywhere
    * Clients
        * Build TUI with [tview](https://github.com/rivo/tview)
        * Build Web UI -- maybe [awesome-grpc](https://github.com/grpc-ecosystem/awesome-grpc) has suggestions
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
    * Add cloud-init
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
    * Run bhyve in jail
    * Run bhyve in [IP-less jail](https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=273557)
    * Support [BIOS](https://www.gulbra.net/freebsd-bhyve/)
    * Support [more](https://github.com/freenas/libhyve-remote) [VNC encoding](https://reviews.freebsd.org/D11768)
    * Support [QCOW2](https://github.com/xcllnt/libvdsk)
    * Test grpc interface with [grpcurl](https://github.com/fullstorydev/grpcurl)
    * Look into use QUIC via [quic-go](https://github.com/quic-go/quic-go), perhaps for host to host disk image transfer
    * [Convert](https://github.com/grpc/grpc-go/blob/master/examples/features/error_details/client/main.go) all grpc [errors](https://grpc.github.io/grpc/core/md_doc_statuscodes.html), especially invalid argument
    * More notes on gRPC error handling [here](https://cloud.google.com/apis/design/errors#error_model) and [here](https://grpc.io/docs/guides/error/)
    * Add more [detail](https://github.com/grpc/grpc-go/blob/master/Documentation/rpc-errors.md) to grpc errors
    * [Improve](https://protobuf.dev/programming-guides/techniques/) [grpc](https://protobuf.dev/programming-guides/dos-donts/) [stuff](https://protobuf.dev/programming-guides/api/)
    * Use well known [types](https://protobuf.dev/reference/protobuf/google.protobuf/) in proto, especially Empty
    * Improve python [protobuf](https://protobuf.dev/programming-guides/proto3/) [code](https://protobuf.dev/reference/python/python-generated/#fields)
    * [Rate limit grpc](https://stackoverflow.com/questions/62925871/grpc-rate-limiting-an-api-on-a-per-rpc-basis )
