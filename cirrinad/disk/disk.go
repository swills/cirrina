package disk

import (
	"errors"
	"strings"
)

func Create(name string, description string, path string) (disk *Disk, err error) {
	var diskInst *Disk
	if strings.Contains(name, "/") {
		return diskInst, errors.New("illegal character in disk name")
	}
	diskInst = &Disk{
		Name:        name,
		Description: description,
		Path:        path,
	}
	db := getDiskDb()
	res := db.Create(&diskInst)
	return diskInst, res.Error
}

func GetAll() []*Disk {
	var result []*Disk
	db := getDiskDb()
	db.Find(&result)
	return result
}

func GetById(id string) (d *Disk, err error) {
	db := getDiskDb()
	db.First(&d, "id = ?", id)
	return d, nil
}
