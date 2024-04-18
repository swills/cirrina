package iso

import (
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

func Create(name string, description string) (iso *ISO, err error) {
	var isoInst *ISO

	if !util.ValidIsoName(name) {
		return isoInst, errIsoInvalidName
	}

	// check if it exists on disk
	path := config.Config.Disk.VM.Path.Iso + "/" + name
	isoExists, err := util.PathExists(path)
	if err != nil {
		slog.Error("error checking if iso exists", "path", path, "err", err)

		return isoInst, fmt.Errorf("error checking if iso exists: %w", err)
	}
	if isoExists {
		slog.Error("iso exists", "iso", name)

		return isoInst, errIsoExists
	}

	// check if it exists in DB
	existingISO, err := GetByName(name)
	if err != nil {
		slog.Error("error checking db for iso", "name", name, "err", err)

		return isoInst, fmt.Errorf("error checking if iso exists: %w", err)
	}
	if existingISO.Name != "" {
		slog.Error("iso exists", "iso", name)

		return isoInst, errIsoExists
	}

	isoInst = &ISO{
		Name:        name,
		Description: description,
		Path:        path,
	}
	db := getIsoDB()
	res := db.Create(&isoInst)

	return isoInst, res.Error
}

func GetAll() []*ISO {
	var result []*ISO
	db := getIsoDB()
	db.Find(&result)

	return result
}

func GetByID(id string) (result *ISO, err error) {
	db := getIsoDB()
	db.Limit(1).Find(&result, "id = ?", id)

	return result, nil
}

func GetByName(name string) (result *ISO, err error) {
	db := getIsoDB()
	db.Limit(1).Find(&result, "name = ?", name)

	return result, nil
}

func (iso *ISO) Save() error {
	db := getIsoDB()

	res := db.Model(&iso).
		Updates(map[string]interface{}{
			"name":        &iso.Name,
			"description": &iso.Description,
			"path":        &iso.Path,
			"size":        &iso.Size,
			"checksum":    &iso.Checksum,
		},
		)

	if res.Error != nil {
		return errIsoInternalDB
	}

	return nil
}

func Delete(id string) (err error) {
	if id == "" {
		return errIsoIDEmptyOrInvalid
	}
	db := getIsoDB()
	dDisk, err := GetByID(id)
	if err != nil {
		return errIsoNotFound
	}
	res := db.Limit(1).Unscoped().Delete(&dDisk)
	if res.RowsAffected != 1 {
		slog.Error("iso delete error", "RowsAffected", res.RowsAffected)

		return errIsoInternalDB
	}

	return nil
}

type ISO struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null;default:null"`
	Name        string `gorm:"uniqueIndex;not null;default:null"`
	Description string
	Path        string `gorm:"not null;default:null"`
	Size        uint64
	Checksum    string
}
