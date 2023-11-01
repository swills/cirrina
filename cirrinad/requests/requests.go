package requests

import (
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
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
	VmId       string
}

func (req *Request) BeforeCreate(_ *gorm.DB) (err error) {
	req.ID = uuid.NewString()
	return nil
}

func Create(r reqType, vmId string) (req Request, err error) {
	db := getReqDb()
	newReq := Request{
		Type: r,
		VmId: vmId,
	}
	res := db.Create(&newReq)
	if res.RowsAffected != 1 {
		return Request{}, errors.New("failed to create request")
	}
	return newReq, nil
}

func GetByID(id string) (rs Request, err error) {
	db := getReqDb()
	db.Model(&Request{}).Limit(1).Find(&rs, &Request{ID: id})
	return rs, nil
}

func GetUnStarted() Request {
	db := getReqDb()
	rs := Request{}
	db.Limit(1).Where("started_at IS NULL").Find(&rs)
	return rs
}

func (req *Request) Start() {
	db := getReqDb()
	req.StartedAt.Time = time.Now()
	req.StartedAt.Valid = true
	db.Model(&req).Limit(1).Updates(req)
}

func (req *Request) Succeeded() {
	db := getReqDb()
	db.Model(&req).Limit(1).Updates(
		Request{
			Successful: true,
			Complete:   true,
		},
	)
}

func (req *Request) Failed() {
	db := getReqDb()
	db.Model(&req).Limit(1).Updates(
		Request{
			Successful: false,
			Complete:   true,
		},
	)
}

func PendingReqExists(vmId string) bool {
	db := getReqDb()
	eReq := Request{}
	db.Where(map[string]interface{}{"vm_id": vmId, "complete": false}).Find(&eReq)
	if eReq.ID != "" {
		return true
	}
	return false
}

func FailAllPending() (cleared int64) {
	db := getReqDb()
	res := db.Where(map[string]interface{}{"complete": false}).Updates(
		Request{
			Complete: true,
		},
	)
	return res.RowsAffected
}
