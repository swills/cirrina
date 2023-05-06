package vm_nics

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VmNic struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null"`
	Name        string `gorm:"not null"`
	Description string
	Mac         string `gorm:"default:AUTO"`
	NetType     string `gorm:"default:VIRTIONET;check:net_type IN (\"VIRTIONET\",\"E1000\")"`
	NetDevType  string `gorm:"default:TAP;check:net_dev_type IN (\"TAP\",\"VMNET\",\"NETGRAPH\")"`
	VmId        string
	SwitchId    string
}

func (d *VmNic) BeforeCreate(_ *gorm.DB) (err error) {
	d.ID = uuid.NewString()
	return nil
}

func init() {
	db := getVmNicDb()
	err := db.AutoMigrate(&VmNic{})
	if err != nil {
		panic("failed to auto-migrate VmNics")
	}
}
