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
	db := getVMDB()
	for {
		rs := Request{}
		db.Limit(1).Where("started_at IS NULL").Find(&rs)
		if rs.ID != "" {
			rs.StartedAt.Time = time.Now()
			rs.StartedAt.Valid = true
			db.Model(&rs).Limit(1).Updates(rs)
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
