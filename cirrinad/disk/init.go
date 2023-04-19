package disk

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Disk struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null"`
	Name        string `gorm:"not null"`
	Description string
	Path        string `gorm:"not null"`
}

func (d *Disk) BeforeCreate(_ *gorm.DB) (err error) {
	d.ID = uuid.NewString()
	return nil
}

func init() {
	db := getDiskDb()
	err := db.AutoMigrate(&Disk{})
	if err != nil {
		panic("failed to auto-migrate ISO")
	}
}
