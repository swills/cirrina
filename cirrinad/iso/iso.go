package iso

import (
	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

func Create(name string, description string) (iso *ISO, err error) {
	var isoInst *ISO

	if !util.ValidIsoName(name) {
		return isoInst, errors.New("invalid name")
	}

	path := config.Config.Disk.VM.Path.Iso + "/" + name
	isoExists, err := util.PathExists(path)
	if err != nil {
		slog.Error("error checking if iso exists", "path", path, "err", err)
		message := fmt.Sprintf("error checking if iso exists: %s", err)
		return isoInst, errors.New(message)
	}
	if isoExists {
		slog.Error("iso exists", "iso", name)
		return isoInst, errors.New("iso exists")
	}
	existingISO, err := GetByName(name)
	if err != nil {
		slog.Error("error checking db for iso", "name", name, "err", err)
		message := fmt.Sprintf("error checking if iso exists: %s", err)
		return isoInst, errors.New(message)
	}
	if existingISO.Name != "" {
		slog.Error("iso exists", "iso", name)
		return isoInst, errors.New("iso exists")
	}

	isoInst = &ISO{
		Name:        name,
		Description: description,
		Path:        path,
	}
	db := getIsoDb()
	res := db.Create(&isoInst)
	return isoInst, res.Error
}

func GetAll() []*ISO {
	var result []*ISO
	db := getIsoDb()
	db.Find(&result)
	return result
}

func GetById(id string) (result *ISO, err error) {
	db := getIsoDb()
	db.Limit(1).Find(&result, "id = ?", id)
	return result, nil
}

func GetByName(name string) (result *ISO, err error) {
	db := getIsoDb()
	db.Limit(1).Find(&result, "name = ?", name)
	return result, nil
}

func (iso *ISO) Save() error {
	db := getIsoDb()

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
		return errors.New("error updating iso")
	}

	return nil
}

func Delete(id string) (err error) {
	if id == "" {
		return errors.New("unable to delete, iso id empty")
	}
	db := getIsoDb()
	dDisk, err := GetById(id)
	if err != nil {
		errorText := fmt.Sprintf("iso %v not found", id)
		return errors.New(errorText)
	}
	res := db.Limit(1).Unscoped().Delete(&dDisk)
	if res.RowsAffected != 1 {
		errText := fmt.Sprintf("iso delete error, rows affected %v", res.RowsAffected)
		return errors.New(errText)
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
