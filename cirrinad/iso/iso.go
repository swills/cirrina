package iso

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/util"
)

type ISO struct {
	ID          string `gorm:"uniqueIndex;not null;default:null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Name        string         `gorm:"uniqueIndex;not null;default:null"`
	Description string
	Path        string `gorm:"not null;default:null"`
	Size        uint64
	Checksum    string
}

var pathExistsFunc = util.PathExists

func (i *ISO) validate() error {
	if !util.ValidIsoName(i.Name) {
		return ErrIsoInvalidName
	}

	return nil
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

	err = isoInst.validate()
	if err != nil {
		return fmt.Errorf("error creating iso: %w", err)
	}

	db := GetIsoDB()

	res := db.Create(&isoInst)
	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected != 1 {
		return fmt.Errorf("incorrect number of rows affected, err: %w", res.Error)
	}

	return nil
}

func GetAll() []*ISO {
	var result []*ISO

	db := GetIsoDB()
	db.Find(&result)

	return result
}

func GetByID(isoID string) (*ISO, error) {
	if isoID == "" {
		return nil, errIsoIDEmptyOrInvalid
	}

	var result *ISO

	db := GetIsoDB()

	res := db.Limit(1).Find(&result, "id = ?", isoID)
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

	db := GetIsoDB()

	res := db.Limit(1).Find(&result, "name = ?", name)
	if res.Error != nil {
		return nil, res.Error
	}

	if res.RowsAffected != 1 {
		return nil, errIsoNotFound
	}

	return result, nil
}

func (i *ISO) Delete() error {
	isoDB := GetIsoDB()

	if i.InUse() {
		return errIsoInUse
	}

	res := isoDB.Limit(1).Unscoped().Delete(&i)
	if res.RowsAffected != 1 {
		slog.Error("iso delete error", "RowsAffected", res.RowsAffected)

		return errIsoInternalDB
	}

	// TODO actually delete data from disk, maybe?
	return nil
}

func (i *ISO) InUse() bool {
	db := GetIsoDB()

	res := db.Table("vm_isos").Select([]string{"vm_id", "iso_id", "position"}).
		Where("iso_id LIKE ?", i.ID).Limit(1)

	rows, rowErr := res.Rows()
	if rowErr != nil {
		slog.Error("error getting vm_disks rows", "rowErr", rowErr)

		// fail-safe
		return true
	}

	err := rows.Err()
	if err != nil {
		slog.Error("error getting vm_disks rows", "err", err)

		// fail-safe
		return true
	}

	defer func() {
		_ = rows.Close()
	}()

	count := 0
	for rows.Next() {
		count++
	}

	return count > 0
}

func (i *ISO) Save() error {
	db := GetIsoDB()

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

func (i *ISO) GetPath() string {
	return filepath.Join(config.Config.Disk.VM.Path.Iso, i.Name)
}

func (i *ISO) GetVMIDs() []string {
	var retVal []string

	db := GetIsoDB()

	res := db.Table("vm_isos").Select([]string{"vm_id"}).
		Where("iso_id LIKE ?", i.ID)

	rows, rowErr := res.Rows()

	defer func() {
		_ = rows.Close()
	}()

	if rowErr != nil {
		slog.Error("error getting vm_isos rows", "rowErr", rowErr)

		return retVal
	}

	err := rows.Err()
	if err != nil {
		slog.Error("error getting vm_isos rows", "err", err)

		return retVal
	}

	for rows.Next() {
		var vmID string

		err = rows.Scan(&vmID)
		if err != nil {
			slog.Error("error scanning vm_isos row", "err", err)

			continue
		}

		retVal = append(retVal, vmID)
	}

	return retVal
}

// CheckAll checks that the file for the ISO actually exists and also checks for any iso files that don't exist in
// the database
func CheckAll() {
	for _, anISO := range GetAll() {
		exists, err := isoExistsFS(filepath.Join(config.Config.Disk.VM.Path.Iso, anISO.Name))
		if err != nil {
			slog.Error("error checking iso exist", "err", err)

			return
		}

		if !exists {
			slog.Error("iso backing does not exists", "iso.Name", anISO.Name, "iso.ID", anISO.ID)
		}
	}

	isoFiles, err := os.ReadDir(config.Config.Disk.VM.Path.Iso)
	if err != nil {
		slog.Error("failed checking isos", "err", err)
	} else {
		for _, v := range isoFiles {
			if util.ValidIsoName(v.Name()) {
				_, err = GetByName(v.Name())
				if err != nil {
					slog.Warn("possible left over iso", "iso.Name", v.Name(), "file.Name", v.Name())
				}
			}
		}
	}
}
