package rpc

import (
	"cirrina/cirrina"
	"context"
	"errors"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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
	Name           string
	Description    string
	Cpu            uint32
	Mem            uint32
	MaxWait        uint32
	Restart        bool
	RestartDelay   uint32
	Screen         bool
	ScreenWidth    uint32
	ScreenHeight   uint32
	Vncwait        bool
	Wireguestmem   bool
	Tablet         bool
	Storeuefi      bool
	Utc            bool
	Hostbridge     bool
	Acpi           bool
	Hlt            bool
	Eop            bool
	Dpo            bool
	Ium            bool
	Vncport        string
	Keyboard       string
	Autostart      bool
	Sound          bool
	SoundIn        string
	SoundOut       string
	Com1           bool
	Com1Dev        string
	Com2           bool
	Com2Dev        string
	Com3           bool
	Com3Dev        string
	Com4           bool
	Com4Dev        string
	ExtraArgs      string
	Com1Log        bool
	Com2Log        bool
	Com3Log        bool
	Com4Log        bool
	Com1Speed      uint32
	Com2Speed      uint32
	Com3Speed      uint32
	Com4Speed      uint32
	AutostartDelay uint32
	Debug          bool
	DebugWait      bool
	DebugPort      string
	Priority       int32
	Protect        bool
	Pcpu           uint32
	Rbps           uint32
	Wbps           uint32
	Riops          uint32
	Wiops          uint32
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

var serverConn *grpc.ClientConn
var serverClient cirrina.VMInfoClient
var defaultServerContext context.Context
var defaultCancelFunc context.CancelFunc

func GetConn() error {
	var err error
	serverAddr := ServerName + ":" + strconv.FormatInt(int64(ServerPort), 10)
	serverTimeoutDur := time.Second * time.Duration(ServerTimeout)

	if serverConn != nil {
		// already set, assume it's set to the right thing!
		return nil
	}

	// build server connection and client
	serverConn, err = grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return errors.New(status.Convert(err).Message())
	}

	serverClient = cirrina.NewVMInfoClient(serverConn)
	defaultServerContext, defaultCancelFunc = context.WithTimeout(context.Background(), serverTimeoutDur)
	return nil
}

func ResetConnTimeout() {
	defaultServerContext, defaultCancelFunc = context.WithTimeout(
		context.Background(), time.Second*time.Duration(ServerTimeout),
	)
}

func Finish() {
	if serverConn != nil {
		_ = serverConn.Close()
	}
	defaultCancelFunc()
}
