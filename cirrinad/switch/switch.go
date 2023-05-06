package _switch

import (
	"errors"
	"fmt"
	"golang.org/x/exp/slog"
	"strings"
)

func GetById(id string) (s *Switch, err error) {
	db := getSwitchDb()
	db.Limit(1).Find(&s, "id = ?", id)
	return s, nil
}

func GetByName(name string) (s *Switch, err error) {
	db := getSwitchDb()
	db.Limit(1).Find(&s, "name = ?", name)
	return s, nil
}

func GetAll() []*Switch {
	var result []*Switch
	db := getSwitchDb()
	db.Find(&result)
	return result
}

func Create(name string, description string, switchType string) (_switch *Switch, err error) {
	var switchInst *Switch
	if strings.Contains(name, "/") {
		return switchInst, errors.New("illegal character in switch name")
	}
	existingSwitch, err := GetByName(name)
	if err != nil {
		slog.Error("error checking db for switch", "name", name, "err", err)
		return switchInst, err
	}
	if existingSwitch.Name != "" {
		slog.Error("switch exists", "switch", name)
		return switchInst, errors.New("switch exists")
	}

	if switchType != "IF" && switchType != "NG" {
		slog.Error("bad switch type", "switchType", switchType)
		return switchInst, errors.New("bad switch type")
	}

	switchInst = &Switch{
		Name:        name,
		Description: description,
		Type:        switchType,
	}
	db := getSwitchDb()
	res := db.Create(&switchInst)
	return switchInst, res.Error
}

func Delete(id string) (err error) {
	if id == "" {
		return errors.New("unable to delete, switch id empty")
	}
	db := getSwitchDb()
	dSwitch, err := GetById(id)
	if err != nil {
		errorText := fmt.Sprintf("switch %v not found", id)
		return errors.New(errorText)
	}
	res := db.Limit(1).Delete(&dSwitch)
	if res.RowsAffected == 1 {
		return nil
	} else {
		errText := fmt.Sprintf("switch delete error, rows affected %v", res.RowsAffected)
		return errors.New(errText)
	}
}
