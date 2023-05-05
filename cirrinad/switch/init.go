package _switch

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Switch struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null"`
	Name        string `gorm:"uniqueIndex;not null"`
	Description string
	Type        string `gorm:"default:IF;check:type IN (\"IF\",\"NG\")"`
	Uplink      string
}

func (d *Switch) BeforeCreate(_ *gorm.DB) (err error) {
	d.ID = uuid.NewString()
	return nil
}

func init() {
	db := getSwitchDb()
	err := db.AutoMigrate(&Switch{})
	if err != nil {
		panic("failed to auto-migrate switches")
	}
}
