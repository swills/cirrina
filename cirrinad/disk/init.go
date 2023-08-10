package disk

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Disk struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null"`
	Name        string `gorm:"uniqueIndex;not null"`
	Description string
	Path        string `gorm:"not null"`
	Type        string `gorm:"default:NVME;check:type IN (\"NVME\",\"AHCI-HD\",\"VIRTIO-BLK\")"`
}

func (d *Disk) BeforeCreate(_ *gorm.DB) (err error) {
	d.ID = uuid.NewString()
	return nil
}

func init() {
	db := getDiskDb()
	err := db.AutoMigrate(&Disk{})
	if err != nil {
		panic("failed to auto-migrate disk")
	}
}
