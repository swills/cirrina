package main

import (
	"database/sql"
	"gorm.io/gorm"
	"time"
)

type Request struct {
	gorm.Model
	ID         string       `gorm:"uniqueIndex;not null"`
	StartedAt  sql.NullTime `gorm:"index"`
	Successful bool         `gorm:"default:False;check:successful IN (0,1)"`
	Complete   bool         `gorm:"default:False;check:complete IN (0,1)"`
	Type       reqType      `gorm:"type:req_type"`
	VMID       string
}

func processRequests() {
	for {
		rs := getUnStartedReq()
		if rs.ID != "" {
			startReq(rs)
			switch rs.Type {
			case START:
				go startVM(&rs)
			case STOP:
				go stopVM(&rs)
			case DELETE:
				go deleteVM(&rs)
			}

		}
		time.Sleep(500 * time.Millisecond)
	}
}
