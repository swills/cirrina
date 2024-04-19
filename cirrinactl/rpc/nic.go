package rpc

import (
	"errors"
	"fmt"
	"io"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"cirrina/cirrina"
)

func AddNic(name string, description string, mac string, nicType string, nicDevType string,
	rateLimit bool, rateIn uint64, rateOut uint64, switchID string,
) (string, error) {
	if name == "" {
		return "", errNicEmptyName
	}

	var newVMNic cirrina.VmNicInfo
	var err error

	newVMNic.Name = &name
	newVMNic.Description = &description
	newVMNic.Mac = &mac
	newVMNic.Switchid = &switchID
	newVMNic.Ratelimit = &rateLimit
	newVMNic.Ratein = &rateIn
	newVMNic.Rateout = &rateOut

	newVMNic.Nettype, err = mapNicTypeStringToType(nicType)
	if err != nil {
		return "", err
	}

	newVMNic.Netdevtype, err = mapNicDevTypeStringToType(nicDevType)
	if err != nil {
		return "", err
	}

	var nicID *cirrina.VmNicId
	nicID, err = serverClient.AddVMNic(defaultServerContext, &newVMNic)
	if err != nil {
		return "", fmt.Errorf("unable to add nic: %w", err)
	}

	return nicID.Value, nil
}

func RmNic(idPtr string) error {
	var err error
	if idPtr == "" {
		return errNicEmptyID
	}
	var reqID *cirrina.ReqBool
	reqID, err = serverClient.RemoveVMNic(defaultServerContext, &cirrina.VmNicId{Value: idPtr})
	if err != nil {
		return fmt.Errorf("unable to remove nic: %w", err)
	}
	if !reqID.Success {
		return errReqFailed
	}

	return nil
}

func GetVMNicInfo(nicID string) (NicInfo, error) {
	var err error
	var info NicInfo
	var res *cirrina.VmNicInfo

	res, err = serverClient.GetVMNicInfo(defaultServerContext, &cirrina.VmNicId{Value: nicID})
	if err != nil {
		return NicInfo{}, fmt.Errorf("unable to get nic info: %w", err)
	}
	if res == nil {
		return NicInfo{}, errInvalidServerResponse
	}

	if res.Name != nil {
		info.Name = *res.Name
	}

	if res.Description != nil {
		info.Descr = *res.Description
	}

	if res.Mac != nil {
		info.Mac = *res.Mac
	}

	if res.Nettype != nil {
		info.NetType = mapNicTypeTypeToString(*res.Nettype)
	}

	if res.Netdevtype != nil {
		info.NetDevType = mapNicDevTypeTypeToString(*res.Netdevtype)
	}

	if res.Switchid != nil && *res.Switchid != "" {
		info.Uplink, err = SwitchIDToName(*res.Switchid)
		if err != nil {
			info.Uplink = ""
		}
	}

	info.VMName, err = NicGetVM(nicID)
	if err != nil {
		info.VMName = ""
	}

	if res.Ratelimit != nil {
		info.RateLimited = *res.Ratelimit
	}

	if res.Ratein != nil {
		info.RateIn = *res.Ratein
	}

	if res.Rateout != nil {
		info.RateOut = *res.Rateout
	}

	return info, nil
}

func NicNameToID(name string) (string, error) {
	var nicID string
	var err error
	if name == "" {
		return "", errNicEmptyName
	}
	var nicIDs []string
	nicIDs, err = GetVMNicsAll()
	if err != nil {
		return "", err
	}

	found := false
	for _, aNicID := range nicIDs {
		res, err := GetVMNicInfo(aNicID)
		if err != nil {
			return "", err
		}
		if res.Name == name {
			if found {
				return "", errNicDuplicate
			}
			found = true
			nicID = aNicID
		}
	}
	if !found {
		return "", errNicNotFound
	}

	return nicID, nil
}

// func NicIdToName(s string, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
// 	res, err := c.GetVMNicInfo(defaultServerContext, &cirrina.VmNicId{Value: s})
// 	print("")
// 	if err != nil {
// 		return "", err
// 	}
// 	return *res.Name, nil
// }

// func GetVmNicOne(idPtr *string, c cirrina.VMInfoClient, ctx context.Context) (string, error) {
// 	var rv string
// 	res, err := c.GetVMNics(defaultServerContext, &cirrina.VMID{Value: *idPtr})
// 	if err != nil {
// 		return "", err
// 	}
// 	found := false
// 	for {
// 		VMNicId, err := res.Recv()
// 		if err == io.EOF {
// 			break
// 		}
// 		if err != nil {
// 			return "", err
// 		}
// 		if found {
// 			return "", errors.New("duplicate nic")
// 		} else {
// 			found = true
// 			rv = VMNicId.Value
// 		}
// 	}
// 	return rv, nil
// }

func GetVMNicsAll() ([]string, error) {
	var err error
	var allNics []string
	var res cirrina.VMInfo_GetVMNicsAllClient
	res, err = serverClient.GetVMNicsAll(defaultServerContext, &cirrina.VmNicsQuery{})
	if err != nil {
		return []string{}, fmt.Errorf("unable to get nics: %w", err)
	}

	for {
		var VMNicID *cirrina.VmNicId
		VMNicID, err = res.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return []string{}, fmt.Errorf("unable to get nics: %w", err)
		}
		allNics = append(allNics, VMNicID.Value)
	}

	return allNics, nil
}

func NicGetVM(nicID string) (string, error) {
	var err error
	if nicID == "" {
		return "", errNicEmptyID
	}
	var res *cirrina.VMID
	res, err = serverClient.GetVMNicVM(defaultServerContext, &cirrina.VmNicId{Value: nicID})
	if err != nil {
		return "", fmt.Errorf("unable to get nic VM: %w", err)
	}
	if res.Value == "" {
		return "", nil
	}
	var res2 string
	res2, err = VMIdToName(res.Value)
	if err != nil {
		return "", err
	}

	return res2, nil
}

func CloneNic(nicID string, newName string) (string, error) {
	if nicID == "" || newName == "" {
		return "", errNicEmptyID
	}

	var err error
	var cloneReq cirrina.VmNicCloneReq
	var existingNicID cirrina.VmNicId
	existingNicID.Value = nicID
	cloneReq.Vmnicid = &existingNicID
	cloneReq.NewVmNicName = wrapperspb.String(newName)
	var reqID *cirrina.RequestID
	reqID, err = serverClient.CloneVMNic(defaultServerContext, &cloneReq)
	if err != nil {
		return "", fmt.Errorf("unable to clone nic: %w", err)
	}

	return reqID.Value, nil
}

func UpdateNic(nicID string, description *string, mac *string, nicType *string, nicDevType *string,
	rateLimit *bool, rateIn *uint64, rateOut *uint64, switchID *string,
) error {
	var err error

	newNicInfo := cirrina.VmNicInfoUpdate{
		Vmnicid: &cirrina.VmNicId{Value: nicID},
	}

	if description != nil {
		newNicInfo.Description = description
	}

	if mac != nil {
		newNicInfo.Mac = mac
	}

	if nicType != nil {
		newNicInfo.Nettype, err = mapNicTypeStringToType(*nicType)
		if err != nil {
			return err
		}
	}

	if nicDevType != nil {
		newNicInfo.Netdevtype, err = mapNicDevTypeStringToType(*nicDevType)
		if err != nil {
			return err
		}
	}

	if rateLimit != nil {
		newNicInfo.Ratelimit = rateLimit
	}

	if rateIn != nil {
		newNicInfo.Ratein = rateIn
	}

	if rateOut != nil {
		newNicInfo.Rateout = rateOut
	}

	if switchID != nil {
		newNicInfo.Switchid = switchID
	}

	var reqStat *cirrina.ReqBool
	reqStat, err = serverClient.UpdateVMNic(defaultServerContext, &newNicInfo)
	if err != nil {
		return fmt.Errorf("unable to update nic: %w", err)
	}
	if !reqStat.Success {
		return errReqFailed
	}

	return nil
}

func mapNicTypeStringToType(nicType string) (*cirrina.NetType, error) {
	NetTypeVirtioNet := cirrina.NetType_VIRTIONET
	NetTypeE1000 := cirrina.NetType_E1000

	switch {
	case nicType == "VIRTIONET" || nicType == "virtionet" || nicType == "VIRTIO-NET" || nicType == "virtio-net":
		return &NetTypeVirtioNet, nil
	case nicType == "E1000" || nicType == "e1000":
		return &NetTypeE1000, nil
	default:
		return nil, errNicInvalidType
	}
}

func mapNicDevTypeStringToType(nicDevType string) (*cirrina.NetDevType, error) {
	NetDevTypeTAP := cirrina.NetDevType_TAP
	NetDevTypeVMNet := cirrina.NetDevType_VMNET
	NetDevTypeNetGraph := cirrina.NetDevType_NETGRAPH

	switch {
	case nicDevType == "TAP" || nicDevType == "tap":
		return &NetDevTypeTAP, nil
	case nicDevType == "VMNET" || nicDevType == "vmnet":
		return &NetDevTypeVMNet, nil
	case nicDevType == "NETGRAPH" || nicDevType == "netgraph":
		return &NetDevTypeNetGraph, nil
	default:
		return nil, errNicInvalidDevType
	}
}

func mapNicTypeTypeToString(nicType cirrina.NetType) string {
	switch nicType {
	case cirrina.NetType_VIRTIONET:
		return "virtio-net"
	case cirrina.NetType_E1000:
		return "e1000"
	default:
		return ""
	}
}

func mapNicDevTypeTypeToString(nicDevType cirrina.NetDevType) string {
	switch nicDevType {
	case cirrina.NetDevType_TAP:
		return "tap"
	case cirrina.NetDevType_VMNET:
		return "vmnet"
	case cirrina.NetDevType_NETGRAPH:
		return "netgraph"
	default:
		return ""
	}
}
