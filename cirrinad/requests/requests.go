package requests

import (
	"database/sql"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type reqType string

const (
	START  reqType = "START"
	STOP   reqType = "STOP"
	DELETE reqType = "DELETE"
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

func (req *Request) BeforeCreate(_ *gorm.DB) (err error) {
	req.ID = uuid.NewString()
	return nil
}
