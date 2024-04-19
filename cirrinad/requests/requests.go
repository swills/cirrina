package requests

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

type VMReqData struct {
	VMID string `json:"vm_id"`
}

type NicCloneReqData struct {
	NicID      string `json:"nic_id"`
	NewNicName string `json:"new_nic_name"`
	NewNicMac  string `json:"new_nic_mac,omitempty"`
	NewNicDesc string `json:"new_nic_desc,omitempty"`
}

type DiskCloneReqData struct {
	DiskID      string `json:"disk_id"`
	NewDiskName string `json:"new_disk_name"`
}

type VMCloneReqData struct {
	VMID      string `json:"vm_id"`
	NewVMName string `json:"new_vm_name"`
}

func (req *Request) BeforeCreate(_ *gorm.DB) error {
	req.ID = uuid.NewString()

	return nil
}

func CreateNicCloneReq(nicID string, newName string) (Request, error) {
	var err error
	var reqData []byte
	reqData, err = json.Marshal(NicCloneReqData{NicID: nicID, NewNicName: newName})
	if err != nil {
		slog.Error("failed parsing NicCloneReqData: %w", err)

		return Request{}, fmt.Errorf("internal error parsing NicClone request: %w", err)
	}
	db := GetReqDB()
	newReq := Request{
		Data: string(reqData),
		Type: NICCLONE,
	}
	res := db.Create(&newReq)
	if res.RowsAffected != 1 {
		return Request{}, errRequestCreateFailure
	}

	return newReq, nil
}

func CreateVMReq(requestType reqType, vmID string) (Request, error) {
	var err error
	var reqData []byte
	reqData, err = json.Marshal(VMReqData{VMID: vmID})
	if err != nil {
		slog.Error("failed parsing CreateVMReq: %w", err)

		return Request{}, fmt.Errorf("internal error parsing CreateVM request: %w", err)
	}
	db := GetReqDB()
	newReq := Request{
		Data: string(reqData),
		Type: requestType,
	}
	res := db.Create(&newReq)
	if res.RowsAffected != 1 {
		return Request{}, errRequestCreateFailure
	}

	return newReq, nil
}

func GetByID(id string) (Request, error) {
	var rs Request
	db := GetReqDB()
	db.Model(&Request{}).Limit(1).Find(&rs, &Request{ID: id})

	return rs, nil
}

func GetUnStarted() Request {
	db := GetReqDB()
	rs := Request{}
	db.Limit(1).Where("started_at IS NULL").Find(&rs)

	return rs
}

func (req *Request) Start() {
	db := GetReqDB()
	req.StartedAt.Time = time.Now()
	req.StartedAt.Valid = true
	db.Model(&req).Limit(1).Updates(req)
}

func (req *Request) Succeeded() {
	db := GetReqDB()
	db.Model(&req).Limit(1).Updates(
		Request{
			Successful: true,
			Complete:   true,
		},
	)
}

func (req *Request) Failed() {
	db := GetReqDB()
	db.Model(&req).Limit(1).Updates(
		Request{
			Successful: false,
			Complete:   true,
		},
	)
}

// PendingReqExists return pending request IDs for given object ID
func PendingReqExists(objID string) []string {
	var reqIDs []string
	db := GetReqDB()
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
			var reqData VMReqData
			err = json.Unmarshal([]byte(incompleteRequest.Data), &reqData)
			if err != nil {
				continue
			}
			if reqData.VMID == objID {
				reqIDs = append(reqIDs, incompleteRequest.ID)
			}
		case NICCLONE:
			var reqData VMCloneReqData
			err = json.Unmarshal([]byte(incompleteRequest.Data), &reqData)
			if err != nil {
				continue
			}
			if reqData.VMID == objID {
				reqIDs = append(reqIDs, incompleteRequest.ID)
			}
		}
	}

	return reqIDs
}

func FailAllPending() int64 {
	db := GetReqDB()
	res := db.Where(map[string]interface{}{"complete": false}).Updates(
		Request{
			Complete: true,
		},
	)

	return res.RowsAffected
}

func DBInitialized() bool {
	db := GetReqDB()

	return db.Migrator().HasColumn(Request{}, "id")
}
