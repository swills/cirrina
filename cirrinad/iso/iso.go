package iso

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"gorm.io/gorm"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

type ISO struct {
	gorm.Model
	ID          string `gorm:"uniqueIndex;not null;default:null"`
	Name        string `gorm:"uniqueIndex;not null;default:null"`
	Description string
	Path        string `gorm:"not null;default:null"`
	Size        uint64
	Checksum    string
}

func Create(isoInst *ISO) error {
	thisIsoExists, err := isoExists(isoInst.Name)
	if err != nil {
		slog.Error("error checking for iso", "isoInst", isoInst, "err", err)

		return err
	}
	if thisIsoExists {
		slog.Error("iso exists", "iso", isoInst.Name)

		return errIsoExists
	}

	err = validateIso(isoInst)
	if err != nil {
		return fmt.Errorf("error creating iso: %w", err)
	}

	db := getIsoDB()
	res := db.Create(&isoInst)
	if res.RowsAffected != 1 {
		return fmt.Errorf("incorrect number of rows affected, err: %w", res.Error)
	}
	if res.Error != nil {
		return res.Error
	}

	return nil
}

func GetAll() []*ISO {
	var result []*ISO
	db := getIsoDB()
	db.Find(&result)

	return result
}

func GetByID(id string) (*ISO, error) {
	var result *ISO
	db := getIsoDB()
	res := db.Limit(1).Find(&result, "id = ?", id)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected != 1 {
		return nil, errIsoNotFound
	}

	return result, nil
}

func GetByName(name string) (*ISO, error) {
	var result *ISO
	db := getIsoDB()
	res := db.Limit(1).Find(&result, "name = ?", name)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected != 1 {
		return nil, errIsoNotFound
	}

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

func Delete(id string) error {
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

func validateIso(isoInst *ISO) error {
	if !util.ValidIsoName(isoInst.Name) {
		return errIsoInvalidName
	}

	return nil
}

func isoExists(isoName string) (bool, error) {
	var err error

	// check DB
	isoInst, err := GetByName(isoName)

	if err != nil {
		if !errors.Is(err, errIsoNotFound) {
			slog.Error("error checking db for iso", "name", isoName, "err", err)

			return false, err
		}

		return false, nil
	}

	if isoInst != nil && isoInst.Name != "" {
		return true, nil
	}

	path := filepath.Join(config.Config.Disk.VM.Path.Iso, isoName)

	// check disk
	isoPathExists, err := util.PathExists(path)
	if err != nil {
		slog.Error("error checking if iso exists", "path", path, "err", err)

		return false, fmt.Errorf("error checking if iso exists: %w", err)
	}
	if isoPathExists {
		slog.Error("iso exists", "iso", isoInst)

		return true, nil
	}

	return false, nil
}
