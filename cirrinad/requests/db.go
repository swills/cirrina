package requests

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func getReqDb() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("cirrina.sqlite"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	return db

}
