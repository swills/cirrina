package vm_nics

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VmNic struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null;default:null"`
	Name        string `gorm:"uniqueIndex;not null;default:null"`
	Description string
	Mac         string `gorm:"default:AUTO"`
	NetDev      string
	NetType     string `gorm:"default:VIRTIONET;check:net_type IN (\"VIRTIONET\",\"E1000\")"`
	NetDevType  string `gorm:"default:TAP;check:net_dev_type IN (\"TAP\",\"VMNET\",\"NETGRAPH\")"`
	SwitchId    string
	RateLimit   bool `gorm:"default:False;check:rate_limit IN(0,1)"`
	RateIn      uint64
	RateOut     uint64
	InstBridge  string
	InstEpair   string
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
