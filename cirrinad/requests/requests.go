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
	var err error

	var newUUID uuid.UUID

	newUUID, err = uuid.NewV7()
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.ID = newUUID.String()

	return nil
}

// CreateNicCloneReq creates a request to clone a NIC
func CreateNicCloneReq(nicID string, newName string) (Request, error) {
	var err error

	if nicID == "" {
		return Request{}, errInvalidRequest
	}

	_, err = uuid.Parse(nicID)
	if err != nil {
		return Request{}, errInvalidRequest
	}

	if newName == "" || !util.ValidNicName(newName) {
		return Request{}, errInvalidRequest
	}

	var reqData []byte

	reqData, err = json.Marshal(NicCloneReqData{NicID: nicID, NewNicName: newName})
	if err != nil {
		slog.Error("failed parsing NicCloneReqData: %w", err)

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
	default:
		return false
	}
}

// CreateVMReq create Request for a VM type operation only
func CreateVMReq(requestType reqType, vmID string) (Request, error) {
	var err error

	if vmID == "" {
		return Request{}, errInvalidRequest
	}

	_, err = uuid.Parse(vmID)
	if err != nil {
		return Request{}, errInvalidRequest
	}

	if !validVMReqType(requestType) {
		return Request{}, errInvalidRequest
	}

	var reqData []byte

	reqData, err = json.Marshal(VMReqData{VMID: vmID})
	if err != nil {
		slog.Error("failed parsing CreateVMReq: %w", err)

		return Request{}, fmt.Errorf("internal error parsing CreateVM request: %w", err)
	}

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

// Start marks a request as started
func (req *Request) Start() {
	db := GetReqDB()
	req.StartedAt.Time = time.Now()
	req.StartedAt.Valid = true
	db.Model(&req).Limit(1).Updates(req)
}

// Succeeded marks a request as completed successfully
func (req *Request) Succeeded() {
	db := GetReqDB()
	db.Model(&req).Limit(1).Updates(
		Request{
			Successful: true,
			Complete:   true,
		},
	)
}

// Failed marks a request as having completed with failure
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
