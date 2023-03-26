package main

import "gorm.io/gorm"

type reqType string

const (
	START  reqType = "START"
	STOP   reqType = "STOP"
	DELETE reqType = "DELETE"
)

type StatusType string

const (
	STOPPED  StatusType = "STOPPED"
	STARTING StatusType = "STARTING"
	RUNNING  StatusType = "RUNNING"
	STOPPING StatusType = "STOPPING"
)

type VMConfig struct {
	gorm.Model
	VMID         string
	Cpu          uint32 `gorm:"default:1;check:cpu BETWEEN 1 and 16"`
	Mem          uint32 `gorm:"default:128;check:mem>=128"`
	MaxWait      uint32 `gorm:"default:120;check:max_wait>=0"`
	Restart      bool   `gorm:"default:True;check:restart IN (0,1)"`
	RestartDelay uint32 `gorm:"default:1;check:restart_delay>=0"`
	Screen       bool   `gorm:"default:True;check:screen IN (0,1)"`
	ScreenWidth  uint32 `gorm:"default:1920;check:screen_width BETWEEN 640 and 1920"`
	ScreenHeight uint32 `gorm:"default:1080;check:screen_height BETWEEN 480 and 1200"`
}

type VM struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null"`
	Name        string `gorm:"not null"`
	Description string
	Status      StatusType `gorm:"type:status_type"`
	BhyvePid    uint32     `gorm:"check:bhyve_pid>=0"`
	NetDev      string
	VNCPort     int32
	VMConfig    VMConfig
}
