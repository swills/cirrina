package rpc

import (
	"cirrina/cirrina"
	"context"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"strconv"
	"time"
)

type NotFoundError struct{}

func (m NotFoundError) Error() string {
	return "not found"
}

type DiskInfo struct {
	Name        string
	Descr       string
	Size        uint64
	Usage       uint64
	DiskType    string
	DiskDevType string
	Cache       bool
	Direct      bool
}

type IsoInfo struct {
	Name  string
	Descr string
	Size  uint64
}

type NicInfo struct {
	Name        string
	Descr       string
	Mac         string
	NetType     string
	NetDevType  string
	Uplink      string
	VmName      string
	RateLimited bool
	RateIn      uint64
	RateOut     uint64
}

type SwitchInfo struct {
	Name       string
	Descr      string
	SwitchType string
	Uplink     string
}

type VmConfig struct {
	Id             string
	Name           *string
	Description    *string
	Cpu            *uint32
	Mem            *uint32
	MaxWait        *uint32
	Restart        *bool
	RestartDelay   *uint32
	Screen         *bool
	ScreenWidth    *uint32
	ScreenHeight   *uint32
	Vncwait        *bool
	Wireguestmem   *bool
	Tablet         *bool
	Storeuefi      *bool
	Utc            *bool
	Hostbridge     *bool
	Acpi           *bool
	Hlt            *bool
	Eop            *bool
	Dpo            *bool
	Ium            *bool
	Vncport        *string
	Keyboard       *string
	Autostart      *bool
	Sound          *bool
	SoundIn        *string
	SoundOut       *string
	Com1           *bool
	Com1Dev        *string
	Com2           *bool
	Com2Dev        *string
	Com3           *bool
	Com3Dev        *string
	Com4           *bool
	Com4Dev        *string
	ExtraArgs      *string
	Com1Log        *bool
	Com2Log        *bool
	Com3Log        *bool
	Com4Log        *bool
	Com1Speed      *uint32
	Com2Speed      *uint32
	Com3Speed      *uint32
	Com4Speed      *uint32
	AutostartDelay *uint32
	Debug          *bool
	DebugWait      *bool
	DebugPort      *string
	Priority       *int32
	Protect        *bool
	Pcpu           *uint32
	Rbps           *uint32
	Wbps           *uint32
	Riops          *uint32
	Wiops          *uint32
}

type UploadStat struct {
	UploadedChunk bool
	UploadedBytes int
	Complete      bool
	Err           error
}

type ReqStatus struct {
	Complete bool
	Success  bool
}

var ServerName string
var ServerPort uint16
var ServerTimeout uint64

func SetupConn() (*grpc.ClientConn, cirrina.VMInfoClient,
	context.Context, context.CancelFunc, error,
) {
	serverAddr := ServerName + ":" + strconv.FormatInt(int64(ServerPort), 10)
	serverTimeoutDur := time.Second * time.Duration(ServerTimeout)

	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, nil, nil, errors.New(status.Convert(err).Message())
	}

	c := cirrina.NewVMInfoClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), serverTimeoutDur)
	return conn, c, ctx, cancel, nil
}

func SetupConnNoTimeoutNoContext() (*grpc.ClientConn, cirrina.VMInfoClient, error) {
	serverAddr := ServerName + ":" + strconv.FormatInt(int64(ServerPort), 10)

	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, errors.New(status.Convert(err).Message())
	}

	c := cirrina.NewVMInfoClient(conn)

	return conn, c, nil
}
