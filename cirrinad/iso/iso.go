package iso

import (
	"errors"
	"fmt"
	"strings"
)

func Create(name string, description string) (iso *ISO, err error) {
	var isoInst *ISO
	if strings.Contains(name, "/") {
		return isoInst, errors.New("illegal character in ISO name")
	}
	isoInst = &ISO{
		Name:        name,
		Description: description,
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
	db.First(&result, "id = ?", id)
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
	res := db.Limit(1).Delete(&dDisk)
	if res.RowsAffected == 1 {
		return nil
	} else {
		errText := fmt.Sprintf("disk delete error, rows affected %v", res.RowsAffected)
		return errors.New(errText)
	}
}
