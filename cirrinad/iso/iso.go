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

var pathExistsFunc = util.PathExists

func Create(isoInst *ISO) error {
	// check DB
	exists, err := isoExistsDB(isoInst.Name)
	if err != nil {
		slog.Error("error checking for iso", "isoInst", isoInst, "err", err)

		return err
	}

	if exists {
		slog.Error("iso exists", "iso", isoInst.Name)

		return errIsoExists
	}

	// check FS
	exists, err = isoExistsFS(isoInst.GetPath())
	if err != nil {
		slog.Error("error checking for iso", "name", isoInst.Name, "err", err)

		return fmt.Errorf("error checking iso exists: %w", err)
	}

	if exists {
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

func (i *ISO) Save() error {
	db := getIsoDB()

	res := db.Model(&i).
		Updates(map[string]interface{}{
			"name":        &i.Name,
			"description": &i.Description,
			"path":        &i.Path,
			"size":        &i.Size,
			"checksum":    &i.Checksum,
		},
		)

	if res.Error != nil {
		return errIsoInternalDB
	}

	return nil
}

func Delete(isoID string) error {
	if isoID == "" {
		return errIsoIDEmptyOrInvalid
	}

	isoDB := getIsoDB()

	dIso, err := GetByID(isoID)
	if err != nil {
		return errIsoNotFound
	}

	res := isoDB.Limit(1).Unscoped().Delete(&dIso)
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

func (i *ISO) GetPath() string {
	return filepath.Join(config.Config.Disk.VM.Path.Iso, i.Name)
}

func isoExistsDB(isoName string) (bool, error) {
	var err error

	_, err = GetByName(isoName)

	if err != nil {
		if !errors.Is(err, errIsoNotFound) {
			slog.Error("error checking db for iso", "name", isoName, "err", err)

			return true, err // fail safe
		}

		return false, nil
	}

	return true, nil
}

func isoExistsFS(name string) (bool, error) {
	isoPathExists, err := pathExistsFunc(name)
	if err != nil {
		slog.Error("error checking if iso exists", "name", name, "err", err)

		return true, fmt.Errorf("error checking if iso exists: %w", err)
	}

	if isoPathExists {
		return true, nil
	}

	return false, nil
}
