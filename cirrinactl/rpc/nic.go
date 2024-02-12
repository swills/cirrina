package rpc

import (
	"cirrina/cirrina"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"io"
)

func AddNic(name string, description string, mac string, nicType string, nicDevType string,
	rateLimit bool, rateIn uint64, rateOut uint64, switchId string) (string, error) {
	if name == "" {
		return "", errors.New("nic name not specified")
	}

	var newVmNic cirrina.VmNicInfo

	var thisNetType cirrina.NetType
	var thisNetDevType cirrina.NetDevType

	newVmNic.Name = &name
	newVmNic.Description = &description
	newVmNic.Mac = &mac
	newVmNic.Switchid = &switchId
	newVmNic.Ratelimit = &rateLimit
	newVmNic.Ratein = &rateIn
	newVmNic.Rateout = &rateOut

	if nicType == "VIRTIONET" || nicType == "virtio-net" {
		thisNetType = cirrina.NetType_VIRTIONET
	} else if nicType == "E1000" || nicType == "e1000" {
		thisNetType = cirrina.NetType_E1000
	} else {
		return "", errors.New("net type must be either VIRTIONET or E1000")
	}

	if nicDevType == "TAP" || nicDevType == "tap" {
		thisNetDevType = cirrina.NetDevType_TAP
	} else if nicDevType == "VMNET" || nicDevType == "vmnet" {
		thisNetDevType = cirrina.NetDevType_VMNET
	} else if nicDevType == "NETGRAPH" || nicDevType == "netgraph" {
		thisNetDevType = cirrina.NetDevType_NETGRAPH
	} else {
		return "", errors.New("net dev type must be one of TAP or VMNET or NETGRAPH")
	}

	newVmNic.Nettype = &thisNetType
	newVmNic.Netdevtype = &thisNetDevType

	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var nicId *cirrina.VmNicId
	nicId, err = c.AddVmNic(ctx, &newVmNic)
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	return nicId.Value, nil
}

func RmNic(idPtr string) error {

	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	if idPtr == "" {
		return errors.New("id not specified")
	}
	var reqId *cirrina.ReqBool
	reqId, err = c.RemoveVmNic(ctx, &cirrina.VmNicId{Value: idPtr})
	if err != nil {
		return errors.New(status.Convert(err).Message())
	}
	if reqId.Success {
		return errors.New("nic delete failure")
	}
	return nil
}

func GetVmNicInfo(id string) (NicInfo, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return NicInfo{}, err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var res *cirrina.VmNicInfo
	res, err = c.GetVmNicInfo(ctx, &cirrina.VmNicId{Value: id})
	if err != nil {
		return NicInfo{}, errors.New(status.Convert(err).Message())
	}
	netDevType := "unknown"
	if *res.Netdevtype == cirrina.NetDevType_TAP {
		netDevType = "tap"
	} else if *res.Netdevtype == cirrina.NetDevType_VMNET {
		netDevType = "vmnet"
	} else if *res.Netdevtype == cirrina.NetDevType_NETGRAPH {
		netDevType = "netgraph"
	}
	netType := "unknown"
	if *res.Nettype == cirrina.NetType_VIRTIONET {
		netType = "virtio-net"
	} else if *res.Nettype == cirrina.NetType_E1000 {
		netType = "e1000"
	}
	uplinkName := ""
	if res.Switchid != nil && *res.Switchid != "" {
		uplinkName, err = SwitchIdToName(*res.Switchid)
		if err != nil {
			return NicInfo{}, err
		}
	}
	var vmName string
	vmName, err = NicGetVm(id)
	if err != nil {
		return NicInfo{}, err
	}
	return NicInfo{
		Name:        *res.Name,
		Descr:       *res.Description,
		Mac:         *res.Mac,
		NetType:     netType,
		NetDevType:  netDevType,
		Uplink:      uplinkName,
		VmName:      vmName,
		RateLimited: *res.Ratelimit,
		RateIn:      *res.Ratein,
		RateOut:     *res.Rateout,
	}, nil
}

func NicNameToId(name string) (nicId string, err error) {
	if name == "" {
		return "", errors.New("nic name not specified")
	}
	var nicIds []string
	nicIds, err = GetVmNicsAll()
	if err != nil {
		return "", err
	}

	found := false
	for _, aNicId := range nicIds {
		res, err := GetVmNicInfo(aNicId)
		if err != nil {
			return "", err
		}
		if res.Name == name {
			if found {
				return "", errors.New("duplicate nic found")
			}
			found = true
			nicId = aNicId
		}
	}
	if !found {
		return "", errors.New("nic not found")
	}
	return nicId, nil
}

//func NicIdToName(s string, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
//	res, err := c.GetVmNicInfo(ctx, &cirrina.VmNicId{Value: s})
//	print("")
//	if err != nil {
//		return "", err
//	}
//	return *res.Name, nil
//}

//func GetVmNicOne(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
//	var rv string
//	res, err := c.GetVmNics(ctx, &cirrina.VMID{Value: *idPtr})
//	if err != nil {
//		return "", err
//	}
//	found := false
//	for {
//		VMNicId, err := res.Recv()
//		if err == io.EOF {
//			break
//		}
//		if err != nil {
//			return "", err
//		}
//		if found {
//			return "", errors.New("duplicate nic")
//		} else {
//			found = true
//			rv = VMNicId.Value
//		}
//	}
//	return rv, nil
//}

func GetVmNicsAll() ([]string, error) {

	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return []string{}, err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var rv []string
	var res cirrina.VMInfo_GetVmNicsAllClient
	res, err = c.GetVmNicsAll(ctx, &cirrina.VmNicsQuery{})
	if err != nil {
		return []string{}, errors.New(status.Convert(err).Message())
	}

	for {
		var VMNicId *cirrina.VmNicId
		VMNicId, err = res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []string{}, errors.New(status.Convert(err).Message())
		}
		rv = append(rv, VMNicId.Value)
	}
	return rv, nil
}

func NicGetVm(id string) (string, error) {
	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	if id == "" {
		return "", errors.New("nic id not specified")
	}
	var res *cirrina.VMID
	res, err = c.GetVmNicVm(ctx, &cirrina.VmNicId{Value: id})
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	if res.Value == "" {
		return "", nil
	}
	var res2 string
	res2, err = VmIdToName(res.Value)
	if err != nil {
		return "", err
	}
	return res2, nil
}

func CloneNic(id string, newName string, newMac string) (string, error) {
	if id == "" || newName == "" {
		return "", errors.New("id name not specified")
	}

	conn, c, ctx, cancel, err := SetupConn()
	if err != nil {
		return "", err
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)
	defer cancel()

	var cloneReq cirrina.VmNicCloneReq
	var existingNicId cirrina.VmNicId
	existingNicId.Value = id
	cloneReq.Vmnicid = &existingNicId
	cloneReq.NewVmNicName = wrapperspb.String(newName)
	cloneReq.NewVmNicMac = wrapperspb.String(newMac)
	var reqId *cirrina.RequestID
	reqId, err = c.CloneVmNic(ctx, &cloneReq)
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	return reqId.Value, nil
}
