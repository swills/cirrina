package rpc

import (
	"fmt"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"cirrina/cirrina"
)

type DiskInfo struct {
	Name        string
	Descr       string
	DiskType    string
	DiskDevType string
	Cache       bool
	Direct      bool
}

type DiskSizeUsage struct {
	Size  uint64
	Usage uint64
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
	VMName      string
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

type VMConfig struct {
	ID             string `json:"ID"             yaml:"ID"`
	Name           string `json:"Name"           yaml:"Name"`
	Description    string `json:"Description"    yaml:"Description"`
	CPU            uint32 `json:"CPU"            yaml:"CPU"`
	Mem            uint32 `json:"Mem"            yaml:"Mem"`
	MaxWait        uint32 `json:"MaxWait"        yaml:"MaxWait"`
	Restart        bool   `json:"Restart"        yaml:"Restart"`
	RestartDelay   uint32 `json:"RestartDelay"   yaml:"RestartDelay"`
	Screen         bool   `json:"Screen"         yaml:"Screen"`
	ScreenWidth    uint32 `json:"ScreenWidth"    yaml:"ScreenWidth"`
	ScreenHeight   uint32 `json:"ScreenHeight"   yaml:"ScreenHeight"`
	Vncwait        bool   `json:"Vncwait"        yaml:"Vncwait"`
	Wireguestmem   bool   `json:"Wireguestmem"   yaml:"Wireguestmem"`
	Tablet         bool   `json:"Tablet"         yaml:"Tablet"`
	Storeuefi      bool   `json:"Storeuefi"      yaml:"Storeuefi"`
	Utc            bool   `json:"Utc"            yaml:"Utc"`
	Hostbridge     bool   `json:"Hostbridge"     yaml:"Hostbridge"`
	Acpi           bool   `json:"Acpi"           yaml:"Acpi"`
	Hlt            bool   `json:"Hlt"            yaml:"Hlt"`
	Eop            bool   `json:"Eop"            yaml:"Eop"`
	Dpo            bool   `json:"Dpo"            yaml:"Dpo"`
	Ium            bool   `json:"Ium"            yaml:"Ium"`
	Vncport        string `json:"Vncport"        yaml:"Vncport"`
	Keyboard       string `json:"Keyboard"       yaml:"Keyboard"`
	Autostart      bool   `json:"Autostart"      yaml:"Autostart"`
	Sound          bool   `json:"Sound"          yaml:"Sound"`
	SoundIn        string `json:"SoundIn"        yaml:"SoundIn"`
	SoundOut       string `json:"SoundOut"       yaml:"SoundOut"`
	Com1           bool   `json:"Com1"           yaml:"Com1"`
	Com1Dev        string `json:"Com1Dev"        yaml:"Com1Dev"`
	Com2           bool   `json:"Com2"           yaml:"Com2"`
	Com2Dev        string `json:"Com2Dev"        yaml:"Com2Dev"`
	Com3           bool   `json:"Com3"           yaml:"Com3"`
	Com3Dev        string `json:"Com3Dev"        yaml:"Com3Dev"`
	Com4           bool   `json:"Com4"           yaml:"Com4"`
	Com4Dev        string `json:"Com4Dev"        yaml:"Com4Dev"`
	ExtraArgs      string `json:"ExtraArgs"      yaml:"ExtraArgs"`
	Com1Log        bool   `json:"Com1Log"        yaml:"Com1Log"`
	Com2Log        bool   `json:"Com2Log"        yaml:"Com2Log"`
	Com3Log        bool   `json:"Com3Log"        yaml:"Com3Log"`
	Com4Log        bool   `json:"Com4Log"        yaml:"Com4Log"`
	Com1Speed      uint32 `json:"Com1Speed"      yaml:"Com1Speed"`
	Com2Speed      uint32 `json:"Com2Speed"      yaml:"Com2Speed"`
	Com3Speed      uint32 `json:"Com3Speed"      yaml:"Com3Speed"`
	Com4Speed      uint32 `json:"Com4Speed"      yaml:"Com4Speed"`
	AutostartDelay uint32 `json:"AutostartDelay" yaml:"AutostartDelay"`
	Debug          bool   `json:"Debug"          yaml:"Debug"`
	DebugWait      bool   `json:"DebugWait"      yaml:"DebugWait"`
	DebugPort      string `json:"DebugPort"      yaml:"DebugPort"`
	Priority       int32  `json:"Priority"       yaml:"Priority"`
	Protect        bool   `json:"Protect"        yaml:"Protect"`
	Pcpu           uint32 `json:"Pcpu"           yaml:"Pcpu"`
	Rbps           uint32 `json:"Rbps"           yaml:"Rbps"`
	Wbps           uint32 `json:"Wbps"           yaml:"Wbps"`
	Riops          uint32 `json:"Riops"          yaml:"Riops"`
	Wiops          uint32 `json:"Wiops"          yaml:"Wiops"`
}

type UploadStat struct {
	UploadedChunk bool
	UploadedBytes uint64
	Complete      bool
	Err           error
}

type ReqStatus struct {
	Complete bool
	Success  bool
}

var (
	ServerName    string
	ServerPort    uint16
	ServerTimeout int64
)

var (
	serverConn   *grpc.ClientConn
	serverClient cirrina.VMInfoClient
)

func GetConn() error {
	var err error

	serverAddr := ServerName + ":" + strconv.FormatInt(int64(ServerPort), 10)

	if serverConn != nil {
		// already set, assume it's set to the right thing!
		return nil
	}

	// build server connection and client
	serverConn, err = grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("unable to connect: %w", err)
	}

	serverClient = cirrina.NewVMInfoClient(serverConn)

	return nil
}
