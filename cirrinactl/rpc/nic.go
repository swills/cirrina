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

	return nicID.GetValue(), nil
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

	if !reqID.GetSuccess() {
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

	info.Name = res.GetName()
	info.Descr = res.GetDescription()
	info.Mac = res.GetMac()
	info.NetType = mapNicTypeTypeToString(res.GetNettype())
	info.NetDevType = mapNicDevTypeTypeToString(res.GetNetdevtype())

	if res.GetSwitchid() != "" {
		info.Uplink, err = SwitchIDToName(res.GetSwitchid())
		if err != nil {
			info.Uplink = ""
		}
	}

	if res.GetVmid() != "" {
		info.VMName, err = VMIdToName(res.GetVmid())
		if err != nil {
			info.VMName = ""
		}
	}

	info.RateLimited = res.GetRatelimit()
	info.RateIn = res.GetRatein()
	info.RateOut = res.GetRateout()

	return info, nil
}

func NicNameToID(name string) (string, error) {
	var nicID string

	var err error

	if name == "" {
		return "", errNicEmptyName
	}

	nicID, err = GetVMNicID(name)
	if err != nil {
		return "", err
	}

	return nicID, nil
}

func GetVMNicsAll() ([]string, error) {
	var err error

	var ids []string

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

		ids = append(ids, VMNicID.GetValue())
	}

	return ids, nil
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

	return reqID.GetValue(), nil
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

	if !reqStat.GetSuccess() {
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
