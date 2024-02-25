package requests

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type reqType string

const (
	VMSTART  reqType = "VMSTART"
	VMSTOP   reqType = "VMSTOP"
	VMDELETE reqType = "VMDELETE"
	NICCLONE reqType = "NICCLONE"
)

type Request struct {
	gorm.Model
	ID         string       `gorm:"uniqueIndex;not null;default:null"`
	StartedAt  sql.NullTime `gorm:"index"`
	Successful bool         `gorm:"default:False;check:successful IN (0,1)"`
	Complete   bool         `gorm:"default:False;check:complete IN (0,1)"`
	Type       reqType      `gorm:"type:req_type"`
	Data       string
}

type VmReqData struct {
	VmId string `json:"vm_id"`
}

type NicCloneReqData struct {
	NicId      string `json:"nic_id"`
	NewNicName string `json:"new_nic_name"`
	NewNicMac  string `json:"new_nic_mac,omitempty"`
	NewNicDesc string `json:"new_nic_desc,omitempty"`
}

type DiskCloneReqData struct {
	DiskId      string `json:"disk_id"`
	NewDiskName string `json:"new_disk_name"`
}

type VmCloneReqData struct {
	VmId      string `json:"vm_id"`
	NewVmName string `json:"new_vm_name"`
}

func (req *Request) BeforeCreate(_ *gorm.DB) (err error) {
	req.ID = uuid.NewString()
	return nil
}

func CreateNicCloneReq(nicId string, newName string) (req Request, err error) {
	reqType := NICCLONE
	var reqData []byte
	reqData, err = json.Marshal(NicCloneReqData{NicId: nicId, NewNicName: newName})
	if err != nil {
		return Request{}, err
	}
	db := GetReqDb()
	newReq := Request{
		Data: string(reqData),
		Type: reqType,
	}
	res := db.Create(&newReq)
	if res.RowsAffected != 1 {
		return Request{}, errors.New("failed to create request")
	}
	return newReq, nil
}

func CreateVmReq(r reqType, vmId string) (req Request, err error) {
	var reqData []byte
	reqData, err = json.Marshal(VmReqData{VmId: vmId})
	if err != nil {
		return Request{}, err
	}
	db := GetReqDb()
	newReq := Request{
		Data: string(reqData),
		Type: r,
	}
	res := db.Create(&newReq)
	if res.RowsAffected != 1 {
		return Request{}, errors.New("failed to create request")
	}
	return newReq, nil
}

func GetByID(id string) (rs Request, err error) {
	db := GetReqDb()
	db.Model(&Request{}).Limit(1).Find(&rs, &Request{ID: id})
	return rs, nil
}

func GetUnStarted() Request {
	db := GetReqDb()
	rs := Request{}
	db.Limit(1).Where("started_at IS NULL").Find(&rs)
	return rs
}

func (req *Request) Start() {
	db := GetReqDb()
	req.StartedAt.Time = time.Now()
	req.StartedAt.Valid = true
	db.Model(&req).Limit(1).Updates(req)
}

func (req *Request) Succeeded() {
	db := GetReqDb()
	db.Model(&req).Limit(1).Updates(
		Request{
			Successful: true,
			Complete:   true,
		},
	)
}

func (req *Request) Failed() {
	db := GetReqDb()
	db.Model(&req).Limit(1).Updates(
		Request{
			Successful: false,
			Complete:   true,
		},
	)
}

func PendingReqExists(objId string) (reqIds []string) {
	db := GetReqDb()
	var err error
	var incompleteRequests []Request
	db.Where(map[string]interface{}{"complete": false}).Find(&incompleteRequests)

	for _, incompleteRequest := range incompleteRequests {
		switch incompleteRequest.Type {
		case VMSTOP:
			fallthrough
		case VMSTART:
			fallthrough
		case VMDELETE:
			var reqData VmReqData
			err = json.Unmarshal([]byte(incompleteRequest.Data), &reqData)
			if err != nil {
				continue
			}
			if reqData.VmId == objId {
				reqIds = append(reqIds, incompleteRequest.ID)
			}
		case NICCLONE:
			var reqData VmCloneReqData
			err = json.Unmarshal([]byte(incompleteRequest.Data), &reqData)
			if err != nil {
				continue
			}
			if reqData.VmId == objId {
				reqIds = append(reqIds, incompleteRequest.ID)
			}
		}
	}
	return reqIds
}

func FailAllPending() (cleared int64) {
	db := GetReqDb()
	res := db.Where(map[string]interface{}{"complete": false}).Updates(
		Request{
			Complete: true,
		},
	)
	return res.RowsAffected
}

func DbInitialized() bool {
	db := GetReqDb()
	if db.Migrator().HasColumn(Request{}, "id") {
		return true
	}
	return false
}
