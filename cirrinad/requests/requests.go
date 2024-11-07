package requests

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"cirrina/cirrinad/util"
)

type reqType string

const (
	VMSTART  reqType = "VMSTART"
	VMSTOP   reqType = "VMSTOP"
	VMDELETE reqType = "VMDELETE"
	NICCLONE reqType = "NICCLONE"
	DISKWIPE reqType = "DISKWIPE"
)

type Request struct {
	ID         string `gorm:"primaryKey;uniqueIndex;not null;default:null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
	StartedAt  sql.NullTime   `gorm:"index"`
	Successful bool           `gorm:"default:False;check:successful IN (0,1)"`
	Complete   bool           `gorm:"default:False;check:complete IN (0,1)"`
	Type       reqType        `gorm:"type:req_type"`
	Data       string
}

type VMReqData struct {
	VMID string `json:"vm_id"`
}

type DiskReqData struct {
	DiskID string `json:"disk_id"`
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

// validVMReqType Check if Request type is valid for VMs
func validVMReqType(aReqType reqType) bool {
	switch aReqType {
	case VMSTART:
		return true
	case VMSTOP:
		return true
	case VMDELETE:
		return true
	case NICCLONE:
		return false
	case DISKWIPE:
		return false
	default:
		return false
	}
}

func validDiskReqType(aReqType reqType) bool {
	switch aReqType {
	case VMSTART:
		return false
	case VMSTOP:
		return false
	case VMDELETE:
		return false
	case NICCLONE:
		return false
	case DISKWIPE:
		return true
	default:
		return false
	}
}

// CreateNicCloneReq creates a request to clone a NIC
func CreateNicCloneReq(nicID string, newName string) (Request, error) {
	var err error

	if nicID == "" {
		return Request{}, ErrInvalidRequest
	}

	_, err = uuid.Parse(nicID)
	if err != nil {
		return Request{}, ErrInvalidRequest
	}

	if newName == "" || !util.ValidNicName(newName) {
		return Request{}, ErrInvalidRequest
	}

	var reqData []byte

	reqData, err = json.Marshal(NicCloneReqData{NicID: nicID, NewNicName: newName})
	if err != nil {
		slog.Error("failed parsing NicCloneReqData", "err", err)

		return Request{}, fmt.Errorf("internal error parsing NicClone request: %w", err)
	}

	reqDB := GetReqDB()
	newReq := Request{
		Data: string(reqData),
		Type: NICCLONE,
	}

	res := reqDB.Create(&newReq)
	if res.RowsAffected != 1 {
		return Request{}, errRequestCreateFailure
	}

	return newReq, nil
}

// CreateVMReq create Request for a VM type operation only
func CreateVMReq(requestType reqType, vmID string) (Request, error) {
	var err error

	if vmID == "" {
		return Request{}, ErrInvalidRequest
	}

	_, err = uuid.Parse(vmID)
	if err != nil {
		return Request{}, ErrInvalidRequest
	}

	if !validVMReqType(requestType) {
		return Request{}, ErrInvalidRequest
	}

	var reqData []byte

	reqData, _ = json.Marshal(VMReqData{VMID: vmID}) //nolint:errchkjson

	reqDB := GetReqDB()
	newReq := Request{
		Data: string(reqData),
		Type: requestType,
	}

	res := reqDB.Create(&newReq)
	if res.Error != nil {
		return Request{}, res.Error
	}

	if res.RowsAffected != 1 {
		return Request{}, errRequestCreateFailure
	}

	return newReq, nil
}

func CreateDiskReq(requestType reqType, diskID string) (Request, error) {
	var err error

	if diskID == "" {
		return Request{}, ErrInvalidRequest
	}

	_, err = uuid.Parse(diskID)
	if err != nil {
		return Request{}, ErrInvalidRequest
	}

	if !validDiskReqType(requestType) {
		return Request{}, ErrInvalidRequest
	}

	var reqData []byte

	reqData, _ = json.Marshal(DiskReqData{DiskID: diskID}) //nolint:errchkjson

	reqDB := GetReqDB()

	newReq := Request{
		Data: string(reqData),
		Type: requestType,
	}

	res := reqDB.Create(&newReq)
	if res.Error != nil {
		return Request{}, res.Error
	}

	if res.RowsAffected != 1 {
		return Request{}, errRequestCreateFailure
	}

	return newReq, nil
}

// GetByID Request lookup by ID
func GetByID(requestID string) (Request, error) {
	var request Request

	if requestID == "" {
		return Request{}, errRequestNotFound
	}

	db := GetReqDB()

	res := db.Model(&Request{}).Limit(1).Find(&request, &Request{ID: requestID})
	if res.Error != nil {
		return Request{}, res.Error
	}

	if res.RowsAffected != 1 {
		return Request{}, errRequestNotFound
	}

	return request, nil
}

// GetUnStarted returns all requests which have not been started
func GetUnStarted() Request {
	db := GetReqDB()
	rs := Request{}
	db.Limit(1).Where("started_at IS NULL").Find(&rs)

	return rs
}

// PendingReqExists return pending request IDs for given object ID
func PendingReqExists(objID string) []string {
	var reqIDs []string

	reqDB := GetReqDB()

	var err error

	var incompleteRequests []Request

	reqDB.Where(map[string]interface{}{"complete": false}).Find(&incompleteRequests)

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
			var reqData NicCloneReqData

			err = json.Unmarshal([]byte(incompleteRequest.Data), &reqData)
			if err != nil {
				continue
			}

			if reqData.NicID == objID {
				reqIDs = append(reqIDs, incompleteRequest.ID)
			}
		case DISKWIPE:
			var reqData DiskReqData

			err = json.Unmarshal([]byte(incompleteRequest.Data), &reqData)
			if err != nil {
				continue
			}

			if reqData.DiskID == objID {
				reqIDs = append(reqIDs, incompleteRequest.ID)
			}
		}
	}

	return reqIDs
}

// FailAllPending marks all requests which are not complete as failed
func FailAllPending() int64 {
	reqDB := GetReqDB()
	res := reqDB.Where(map[string]interface{}{"complete": false}).Updates(
		Request{
			Complete: true,
		},
	)

	return res.RowsAffected
}

// DBInitialized checks if the requests database has been initialized
func DBInitialized() bool {
	db := GetReqDB()

	return db.Migrator().HasColumn(Request{}, "id")
}

// Start marks a request as started
func (r *Request) Start() {
	db := GetReqDB()
	r.StartedAt.Time = time.Now()
	r.StartedAt.Valid = true
	db.Model(&r).Limit(1).Updates(r)
}

// Succeeded marks a request as completed successfully
func (r *Request) Succeeded() {
	db := GetReqDB()
	db.Model(&r).Limit(1).Updates(
		Request{
			Successful: true,
			Complete:   true,
		},
	)
}

// Failed marks a request as having completed with failure
func (r *Request) Failed() {
	db := GetReqDB()
	db.Model(&r).Limit(1).Updates(
		Request{
			Successful: false,
			Complete:   true,
		},
	)
}
