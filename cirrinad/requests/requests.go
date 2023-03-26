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

func PendingReqExists(vmId string) bool {
	db := getReqDb()
	eReq := Request{}
	db.Where(map[string]interface{}{"vm_id": vmId, "complete": false}).Find(&eReq)
	if eReq.ID != "" {
		return true
	}
	return false
}

func Get(requestID string) Request {
	db := getReqDb()
	rs := Request{}
	db.Model(&Request{}).Limit(1).Find(&rs, &Request{ID: requestID})
	return rs
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

func GetUnStarted() Request {
	db := getReqDb()
	rs := Request{}
	db.Limit(1).Where("started_at IS NULL").Find(&rs)
	return rs
}

func Start(rs Request) {
	db := getReqDb()
	rs.StartedAt.Time = time.Now()
	rs.StartedAt.Valid = true
	db.Model(&rs).Limit(1).Updates(rs)
}

func MarkSuccessful(rs *Request) *gorm.DB {
	db := getReqDb()
	return db.Model(&rs).Limit(1).Updates(
		Request{
			Successful: true,
			Complete:   true,
		},
	)
}

func MarkFailed(rs *Request) *gorm.DB {
	db := getReqDb()
	return db.Model(&rs).Limit(1).Updates(
		Request{
			Successful: false,
			Complete:   true,
		},
	)
}
