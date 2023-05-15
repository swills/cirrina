package iso

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ISO struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null"`
	Name        string `gorm:"not null"`
	Description string
	Path        string `gorm:"not null"`
	Size        uint64
	Checksum    string
}

func (iso *ISO) BeforeCreate(_ *gorm.DB) (err error) {
	iso.ID = uuid.NewString()
	return nil
}

func init() {
	db := getIsoDb()
	err := db.AutoMigrate(&ISO{})
	if err != nil {
		panic("failed to auto-migrate ISO")
	}
}
