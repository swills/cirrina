package disk

import (
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"log/slog"
)

type Disk struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null;default:null"`
	Name        string `gorm:"uniqueIndex;not null;default:null"`
	Description string
	Type        string       `gorm:"default:NVME;check:type IN (\"NVME\",\"AHCI-HD\",\"VIRTIO-BLK\")"`
	DevType     string       `gorm:"default:FILE;check:dev_type IN (\"FILE\",\"ZVOL\")"`
	DiskCache   sql.NullBool `gorm:"default:True;check:disk_cache IN(0,1)"`
	DiskDirect  sql.NullBool `gorm:"default:False;check:disk_direct IN(0,1)"`
}

func (d *Disk) BeforeCreate(_ *gorm.DB) (err error) {
	d.ID = uuid.NewString()
	if d.Name == "" {
		return errors.New("invalid disk name")
	}
	return nil
}

func init() {
	db := getDiskDb()
	err := db.AutoMigrate(&Disk{})
	if err != nil {
		panic("failed to auto-migrate disk")
	}
	err = db.Migrator().DropColumn(&Disk{}, "path")
	if err != nil {
		slog.Error("DiskDb init failed to drop path column, continuing anyway")
	}
}
