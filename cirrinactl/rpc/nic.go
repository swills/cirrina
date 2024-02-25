package rpc

import (
	"cirrina/cirrina"
	"errors"
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

	var err error
	var nicId *cirrina.VmNicId
	nicId, err = serverClient.AddVmNic(defaultServerContext, &newVmNic)
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	return nicId.Value, nil
}

func RmNic(idPtr string) error {
	var err error
	if idPtr == "" {
		return errors.New("id not specified")
	}
	var reqId *cirrina.ReqBool
	reqId, err = serverClient.RemoveVmNic(defaultServerContext, &cirrina.VmNicId{Value: idPtr})
	if err != nil {
		return errors.New(status.Convert(err).Message())
	}
	if reqId.Success {
		return errors.New("nic delete failure")
	}
	return nil
}

func GetVmNicInfo(id string) (NicInfo, error) {
	var err error
	var res *cirrina.VmNicInfo
	res, err = serverClient.GetVmNicInfo(defaultServerContext, &cirrina.VmNicId{Value: id})
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
//	res, err := c.GetVmNicInfo(defaultServerContext, &cirrina.VmNicId{Value: s})
//	print("")
//	if err != nil {
//		return "", err
//	}
//	return *res.Name, nil
//}

//func GetVmNicOne(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
//	var rv string
//	res, err := c.GetVmNics(defaultServerContext, &cirrina.VMID{Value: *idPtr})
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
	var err error
	var rv []string
	var res cirrina.VMInfo_GetVmNicsAllClient
	res, err = serverClient.GetVmNicsAll(defaultServerContext, &cirrina.VmNicsQuery{})
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
	var err error
	if id == "" {
		return "", errors.New("nic id not specified")
	}
	var res *cirrina.VMID
	res, err = serverClient.GetVmNicVm(defaultServerContext, &cirrina.VmNicId{Value: id})
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

func CloneNic(id string, newName string) (string, error) {
	if id == "" || newName == "" {
		return "", errors.New("id name not specified")
	}

	var err error
	var cloneReq cirrina.VmNicCloneReq
	var existingNicId cirrina.VmNicId
	existingNicId.Value = id
	cloneReq.Vmnicid = &existingNicId
	cloneReq.NewVmNicName = wrapperspb.String(newName)
	var reqId *cirrina.RequestID
	reqId, err = serverClient.CloneVmNic(defaultServerContext, &cloneReq)
	if err != nil {
		return "", errors.New(status.Convert(err).Message())
	}
	return reqId.Value, nil
}

func UpdateNic(id string, description *string, mac *string, nicType *string, nicDevType *string,
	rateLimit *bool, rateIn *uint64, rateOut *uint64, switchId *string) error {
	var err error

	j := cirrina.VmNicInfoUpdate{
		Vmnicid: &cirrina.VmNicId{Value: id},
	}

	if description != nil {
		j.Description = description
	}

	if mac != nil {
		j.Mac = mac
	}

	if nicType != nil {
		var useNetType cirrina.NetType
		switch *nicType {
		case "VIRTIONET":
			fallthrough
		case "virtio-net":
			useNetType = cirrina.NetType_VIRTIONET
		case "E1000":
			fallthrough
		case "e1000":
			useNetType = cirrina.NetType_E1000
		default:
			useNetType = cirrina.NetType_VIRTIONET
		}
		j.Nettype = &useNetType
	}

	if nicDevType != nil {
		var useNetDevType cirrina.NetDevType
		switch *nicDevType {
		case "TAP":
			fallthrough
		case "tap":
			useNetDevType = cirrina.NetDevType_TAP
		case "VMNET":
			fallthrough
		case "vmnet":
			useNetDevType = cirrina.NetDevType_VMNET
		case "NETGRAPH":
			fallthrough
		case "netgraph":
			useNetDevType = cirrina.NetDevType_NETGRAPH
		default:
			useNetDevType = cirrina.NetDevType_TAP
		}
		j.Netdevtype = &useNetDevType
	}

	if rateLimit != nil {
		j.Ratelimit = rateLimit
	}

	if rateIn != nil {
		j.Ratein = rateIn
	}

	if rateOut != nil {
		j.Rateout = rateOut
	}
	if switchId != nil {
		j.Switchid = switchId
	}

	var reqStat *cirrina.ReqBool
	reqStat, err = serverClient.UpdateVmNic(defaultServerContext, &j)
	if err != nil {
		return errors.New(status.Convert(err).Message())
	}
	if !reqStat.Success {
		return errors.New("failed to update switch")
	}
	return nil
}
